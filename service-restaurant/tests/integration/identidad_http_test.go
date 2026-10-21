package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"service-restaurant/internal/bootstrap"
	"service-restaurant/internal/modules/identidad/infrastructure/security"
	"service-restaurant/internal/platform/config"
	"service-restaurant/internal/platform/database"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	adminUser = "admin"
	adminPass = "AdminSegura2026"
)

var (
	setupOnce sync.Once
	handler   http.Handler
	setupErr  error
)

func cfg() config.Config {
	return config.Config{
		Addr: ":0", DatabaseURL: testURL, LogLevel: "error", MaxBodyBytes: 1 << 20,
		JWTSecret: []byte(strings.Repeat("s", 40)), JWTIssuer: "service-restaurant", JWTAudience: "service-restaurant-api",
	}
}

// app instala el administrador inicial una sola vez y devuelve el router real.
func app(t *testing.T) http.Handler {
	t.Helper()
	setupOnce.Do(func() {
		ctx := context.Background()
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		_, setupErr = bootstrap.InstalarAdministrador(ctx, cfg(), logger, bootstrap.AdminParams{
			Username: adminUser, Password: adminPass,
			RUC: "20123456789", RazonSocial: "Restaurante de Prueba SAC", NombreComercial: "Prueba",
			DireccionFiscal: "Av. Principal 123", SucursalCodigo: "PRINCIPAL", SucursalNombre: "Local Principal",
			SucursalDireccion: "Av. Principal 123",
		})
		if setupErr != nil {
			return
		}
		a, err := bootstrap.New(ctx, cfg(), logger)
		if err != nil {
			setupErr = err
			return
		}
		handler = a.Handler()
	})
	if setupErr != nil {
		t.Fatalf("instalación inicial: %v", setupErr)
	}
	return handler
}

type resp struct {
	*httptest.ResponseRecorder
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body.Bytes(), v); err != nil {
		t.Fatalf("cuerpo no es JSON: %q (%v)", r.Body.String(), err)
	}
}

func call(t *testing.T, method, path, body, token string) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	app(t).ServeHTTP(rec, req)
	return resp{rec}
}

func login(t *testing.T, user, pass string) resp {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	return call(t, http.MethodPost, "/api/v1/auth/login", string(b), "")
}

func token(t *testing.T, user, pass string) string {
	t.Helper()
	r := login(t, user, pass)
	if r.Code != http.StatusOK {
		t.Fatalf("login de %s: %d %s", user, r.Code, r.Body.String())
	}
	var tk struct {
		AccessToken string `json:"access_token"`
	}
	r.json(t, &tk)
	return tk.AccessToken
}

// asSystem ejecuta SQL de preparación con el actor técnico (exigido por la V4).
func asSystem(t *testing.T, stmts ...string) {
	t.Helper()
	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.usuarios WHERE es_sistema`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := database.SetActor(ctx, tx, id); err != nil {
		t.Fatal(err)
	}
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// crearUsuario inserta un usuario humano con el rol y la sucursal del administrador.
func crearUsuario(t *testing.T, username, password, rol string) {
	t.Helper()
	app(t) // la instalación inicial debe ocurrir antes que cualquier otro usuario humano
	h, err := security.NewArgon2Hasher(security.Argon2ParamsTest)
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := h.Hash(password)
	asSystem(t,
		`INSERT INTO sch_restaurante.usuarios(username, password_hash) VALUES ('`+username+`', '`+hash+`')`,
		`INSERT INTO sch_restaurante.usuarios_roles(usuario_id, rol_id)
		 SELECT u.id, r.id FROM sch_restaurante.usuarios u, sch_restaurante.roles r WHERE u.username='`+username+`' AND r.codigo='`+rol+`'`,
		`INSERT INTO sch_restaurante.usuarios_sucursales(usuario_id, sucursal_id)
		 SELECT u.id, s.id FROM sch_restaurante.usuarios u, sch_restaurante.sucursales s WHERE u.username='`+username+`' AND s.codigo='PRINCIPAL'`)
}

func TestInstalacionInicialNoSeRepite(t *testing.T) {
	app(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := bootstrap.InstalarAdministrador(context.Background(), cfg(), logger, bootstrap.AdminParams{
		Username: "otro", Password: "OtraClave2026Segura", RUC: "20123456789", RazonSocial: "x", NombreComercial: "x",
		DireccionFiscal: "x", SucursalCodigo: "X", SucursalNombre: "x", SucursalDireccion: "x",
	})
	var ae *apperror.AppError
	if !errors.As(err, &ae) || ae.Code != "ya_instalado" {
		t.Fatalf("debía rechazar con ya_instalado, obtuve %v", err)
	}
}

func TestLoginYSesion(t *testing.T) {
	r := login(t, adminUser, adminPass)
	if r.Code != http.StatusOK || r.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("login: %d %v", r.Code, r.Header())
	}
	var tk struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	r.json(t, &tk)
	if tk.TokenType != "Bearer" || tk.ExpiresIn != 7200 || strings.Count(tk.AccessToken, ".") != 2 {
		t.Fatalf("token incorrecto: %+v", tk)
	}

	s := call(t, http.MethodGet, "/api/v1/sesion", "", tk.AccessToken)
	if s.Code != http.StatusOK {
		t.Fatalf("sesion: %d %s", s.Code, s.Body.String())
	}
	var ses struct {
		UsuarioID  string   `json:"usuario_id"`
		Username   string   `json:"username"`
		Sucursales []string `json:"sucursales"`
		Permisos   []string `json:"permisos"`
	}
	s.json(t, &ses)
	if ses.Username != adminUser || len(ses.Sucursales) != 1 || len(ses.Permisos) != 16 || ses.UsuarioID == "" {
		t.Fatalf("sesión incorrecta: %+v", ses)
	}
}

func TestSesionExigeTokenValido(t *testing.T) {
	good := token(t, adminUser, adminPass)
	partes := strings.Split(good, ".")
	manipulado := partes[0] + "." + partes[1] + "." + strings.Repeat("A", len(partes[2]))
	for name, tok := range map[string]string{"sin token": "", "basura": "abc", "firma alterada": manipulado} {
		r := call(t, http.MethodGet, "/api/v1/sesion", "", tok)
		if r.Code != http.StatusUnauthorized || r.Header().Get("Content-Type") != "application/problem+json" ||
			r.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Errorf("%s: %d %v %s", name, r.Code, r.Header(), r.Body.String())
		}
	}
	// Esquema distinto de Bearer.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sesion", nil)
	req.Header.Set("Authorization", "Basic "+good)
	rec := httptest.NewRecorder()
	app(t).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Basic: %d", rec.Code)
	}
}

// RES-RF-001: todo rechazo es idéntico; no revela si el usuario existe.
func TestRechazosDeLoginSonIdenticos(t *testing.T) {
	crearUsuario(t, "inactivo", "Inactivo2026Clave", "CAJERO")
	asSystem(t, `UPDATE sch_restaurante.usuarios SET es_activo=false, motivo_eliminacion='prueba' WHERE username='inactivo'`)

	normalizar := func(r resp) string {
		var p map[string]any
		r.json(t, &p)
		delete(p, "requestId")
		b, _ := json.Marshal(p)
		return string(b)
	}
	casos := [][2]string{
		{adminUser, "claveIncorrecta99"}, {"noexiste", "ClaveCualquiera2026"}, {"inactivo", "Inactivo2026Clave"},
		{"__sistema__", "loquesea12345"}, {"", ""},
	}
	var ref string
	for i, c := range casos {
		r := login(t, c[0], c[1])
		if r.Code != http.StatusUnauthorized {
			t.Fatalf("%v: %d", c, r.Code)
		}
		got := normalizar(r)
		if i == 0 {
			ref = got
		} else if got != ref {
			t.Errorf("%v: cuerpo distinto\n got %s\nwant %s", c, got, ref)
		}
	}
	// El intento fallido del administrador real no debe bloquearlo tras uno solo.
	if r := login(t, adminUser, adminPass); r.Code != http.StatusOK {
		t.Fatalf("admin debía poder entrar: %d", r.Code)
	}
}

func TestBloqueoPersisteTrasCincoFallos(t *testing.T) {
	crearUsuario(t, "bloq", "BloqueoClave2026", "CAJERO")
	for i := 0; i < 5; i++ {
		if r := login(t, "bloq", "equivocada"); r.Code != http.StatusUnauthorized {
			t.Fatalf("intento %d: %d", i, r.Code)
		}
	}
	// Los rechazos se CONFIRMARON: el contador sobrevivió a la transacción.
	if n := scalar(t, `SELECT intentos_fallidos FROM sch_restaurante.usuarios WHERE username='bloq'`); n != 5 {
		t.Fatalf("intentos_fallidos = %d, quería 5", n)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE username='bloq' AND bloqueado_hasta > now()`); n != 1 {
		t.Fatal("debe estar bloqueado")
	}
	if r := login(t, "bloq", "BloqueoClave2026"); r.Code != http.StatusUnauthorized {
		t.Fatalf("bloqueado: %d", r.Code)
	}
	// Vencido el bloqueo, entra y se limpia el contador.
	asSystem(t, `UPDATE sch_restaurante.usuarios SET bloqueado_hasta = now() - interval '1 minute' WHERE username='bloq'`)
	if r := login(t, "bloq", "BloqueoClave2026"); r.Code != http.StatusOK {
		t.Fatalf("tras el bloqueo: %d", r.Code)
	}
	if n := scalar(t, `SELECT intentos_fallidos FROM sch_restaurante.usuarios WHERE username='bloq'`); n != 0 {
		t.Fatalf("intentos_fallidos = %d, quería 0", n)
	}
}

func TestLogoutRevocaAlInstante(t *testing.T) {
	tk := token(t, adminUser, adminPass)
	if r := call(t, http.MethodGet, "/api/v1/sesion", "", tk); r.Code != http.StatusOK {
		t.Fatalf("antes: %d", r.Code)
	}
	if r := call(t, http.MethodPost, "/api/v1/auth/logout", "", tk); r.Code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", r.Code, r.Body.String())
	}
	if r := call(t, http.MethodGet, "/api/v1/sesion", "", tk); r.Code != http.StatusUnauthorized {
		t.Fatalf("token revocado debe dar 401, dio %d", r.Code)
	}
	// El logout es idempotente en servidor, pero el token revocado ya no autentica.
	if r := call(t, http.MethodPost, "/api/v1/auth/logout", "", tk); r.Code != http.StatusUnauthorized {
		t.Fatalf("segundo logout: %d", r.Code)
	}
}

func TestBajaDeUsuarioYPermisosSeAplicanDeInmediato(t *testing.T) {
	crearUsuario(t, "cajero1", "CajeroClave2026", "ADMIN")
	tk := token(t, "cajero1", "CajeroClave2026")

	// Quitar la sede deja al usuario autenticado pero sin permisos efectivos.
	asSystem(t, `UPDATE sch_restaurante.usuarios_sucursales SET es_activo=false
		WHERE usuario_id=(SELECT id FROM sch_restaurante.usuarios WHERE username='cajero1')`)
	s := call(t, http.MethodGet, "/api/v1/sesion", "", tk)
	var ses struct {
		Permisos   []string `json:"permisos"`
		Sucursales []string `json:"sucursales"`
	}
	s.json(t, &ses)
	if s.Code != http.StatusOK || len(ses.Permisos) != 0 || len(ses.Sucursales) != 0 {
		t.Fatalf("sin sede no debe tener permisos: %d %+v", s.Code, ses)
	}

	// La baja del usuario invalida su sesión de inmediato.
	asSystem(t, `UPDATE sch_restaurante.usuarios SET es_activo=false, motivo_eliminacion='baja de prueba' WHERE username='cajero1'`)
	if r := call(t, http.MethodGet, "/api/v1/sesion", "", tk); r.Code != http.StatusUnauthorized {
		t.Fatalf("usuario dado de baja debe dar 401, dio %d", r.Code)
	}
}

func TestSesionVencidaPorInactividad(t *testing.T) {
	crearUsuario(t, "lento", "LentoClave2026Ok", "CAJERO")
	tk := token(t, "lento", "LentoClave2026Ok")
	if _, err := testDB.Exec(`UPDATE sch_restaurante.sesiones SET expira_inactividad_en = now() - interval '1 second'
		WHERE usuario_id=(SELECT id FROM sch_restaurante.usuarios WHERE username='lento')`); err != nil {
		t.Fatal(err)
	}
	if r := call(t, http.MethodGet, "/api/v1/sesion", "", tk); r.Code != http.StatusUnauthorized {
		t.Fatalf("sesión inactiva debe dar 401, dio %d", r.Code)
	}
}

func TestLoginValidaElCuerpo(t *testing.T) {
	casos := map[string]struct {
		body, ct string
	}{
		"campo desconocido": {`{"username":"a","password":"b","extra":1}`, "application/json"},
		"json roto":         {`{"username":`, "application/json"},
		"dos objetos":       {`{"username":"a","password":"b"}{}`, "application/json"},
		"vacío":             {``, "application/json"},
		"content-type":      {`{"username":"a","password":"b"}`, "text/plain"},
	}
	for name, c := range casos {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(c.body))
		req.Header.Set("Content-Type", c.ct)
		rec := httptest.NewRecorder()
		app(t).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
	}
}

// ADR-03: la auditoría registra el actor y nunca los hashes de contraseña.
func TestAuditoriaDeLoginSinSecretos(t *testing.T) {
	crearUsuario(t, "audit", "AuditClave2026Ok", "CAJERO")
	login(t, "audit", "equivocada")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos a
		JOIN sch_restaurante.usuarios s ON s.id = a.creado_por AND s.es_sistema
		WHERE a.tabla='usuarios' AND a.accion='UPDATE'
		  AND a.registro_id=(SELECT id FROM sch_restaurante.usuarios WHERE username='audit')`); n < 1 {
		t.Fatal("el fallo de login debe auditarse con el actor técnico")
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos
		WHERE tabla='usuarios' AND (datos_antes::text LIKE '%argon2%' OR datos_despues::text LIKE '%argon2%'
		   OR datos_antes::text LIKE '%password_hash%' OR datos_despues::text LIKE '%password_hash%')`); n != 0 {
		t.Fatalf("%d eventos de auditoría contienen el hash de contraseña", n)
	}
}
