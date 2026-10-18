package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

var idemSeq atomic.Int64

// nuevaClave genera una Idempotency-Key UUID distinta en cada llamada.
func nuevaClave() string {
	return fmt.Sprintf("aaaaaaaa-0000-4000-8000-%012d", idemSeq.Add(1))
}

func callH(t *testing.T, method, path, body, tk string, headers map[string]string) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if tk != "" {
		req.Header.Set("Authorization", "Bearer "+tk)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	app(t).ServeHTTP(rec, req)
	return resp{rec}
}

// post envía una escritura con su propia clave de idempotencia.
func post(t *testing.T, path, body, tk string) resp {
	return callH(t, http.MethodPost, path, body, tk, map[string]string{"Idempotency-Key": nuevaClave()})
}

type problema struct {
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func esperar(t *testing.T, r resp, status int, code string) {
	t.Helper()
	if r.Code != status {
		t.Fatalf("estado %d, quería %d: %s", r.Code, status, r.Body.String())
	}
	if code != "" {
		var p problema
		r.json(t, &p)
		if p.Code != code {
			t.Fatalf("code %q, quería %q: %s", p.Code, code, r.Body.String())
		}
	}
}

type entidad struct {
	ID       string  `json:"id"`
	Codigo   string  `json:"codigo"`
	Nombre   string  `json:"nombre"`
	Desc     *string `json:"descripcion"`
	EsActivo bool    `json:"es_activo"`
	Username string  `json:"username"`
}

func adminToken(t *testing.T) string { return token(t, adminUser, adminPass) }

func adminID(t *testing.T) int64 {
	return int64(scalar(t, `SELECT id FROM sch_restaurante.usuarios WHERE username='admin'`))
}

func idDe(t *testing.T, tabla, campo, valor string) string {
	t.Helper()
	return fmt.Sprint(scalar(t, `SELECT id FROM sch_restaurante.`+tabla+` WHERE `+campo+`='`+valor+`'`))
}

// RES-RF-033: solo quien tiene USUARIO_ADMINISTRAR administra accesos.
func TestAdministracionRequierePermiso(t *testing.T) {
	crearUsuario(t, "sinpoder", "SinPoder2026Clave", "CAJERO")
	tk := token(t, "sinpoder", "SinPoder2026Clave")
	for _, c := range []struct{ metodo, ruta, cuerpo string }{
		{"GET", "/api/v1/roles", ""}, {"GET", "/api/v1/permisos", ""},
		{"POST", "/api/v1/roles", `{"codigo":"XX","nombre":"x"}`},
		{"POST", "/api/v1/usuarios", `{"username":"zz","password":"Clave2026Segura"}`},
		{"PUT", "/api/v1/usuarios/1/asignaciones", `{"rol_ids":["1"],"sucursal_ids":["1"]}`},
	} {
		r := callH(t, c.metodo, c.ruta, c.cuerpo, tk, map[string]string{"Idempotency-Key": nuevaClave()})
		esperar(t, r, http.StatusForbidden, "sin_permiso")
	}
	esperar(t, call(t, "GET", "/api/v1/roles", "", ""), http.StatusUnauthorized, "no_autenticado")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.roles WHERE codigo='XX'`); n != 0 {
		t.Fatal("un rechazo por permiso no debe escribir")
	}
}

func TestRolesCicloCompletoEIdempotencia(t *testing.T) {
	tk := adminToken(t)
	clave := nuevaClave()
	cuerpo := `{"codigo":"jefe_sala","nombre":"Jefe de sala"}`
	h := map[string]string{"Idempotency-Key": clave}

	r1 := callH(t, "POST", "/api/v1/roles", cuerpo, tk, h)
	esperar(t, r1, http.StatusCreated, "")
	var rol entidad
	r1.json(t, &rol)
	if rol.Codigo != "JEFE_SALA" || !rol.EsActivo || rol.ID == "" {
		t.Fatalf("rol creado: %+v", rol)
	}

	// Repetir la misma intención devuelve el mismo efecto sin duplicar.
	r2 := callH(t, "POST", "/api/v1/roles", cuerpo, tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var rep entidad
	r2.json(t, &rep)
	if rep.ID != rol.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.roles WHERE codigo='JEFE_SALA'`) != 1 {
		t.Fatalf("la repetición debe devolver el mismo rol: %+v vs %+v", rep, rol)
	}
	// Misma clave, otro contenido: conflicto.
	esperar(t, callH(t, "POST", "/api/v1/roles", `{"codigo":"otro","nombre":"Otro"}`, tk, h), http.StatusConflict, "idempotency_key_reutilizada")
	// Otra clave, código repetido: conflicto de unicidad.
	esperar(t, post(t, "/api/v1/roles", `{"codigo":"JEFE_SALA","nombre":"Dup"}`, tk), http.StatusConflict, "registro_duplicado")
	// Sin clave / clave inválida.
	esperar(t, call(t, "POST", "/api/v1/roles", cuerpo, tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, callH(t, "POST", "/api/v1/roles", cuerpo, tk, map[string]string{"Idempotency-Key": "abc"}), http.StatusBadRequest, "idempotency_key_invalida")
	// Validaciones.
	esperar(t, post(t, "/api/v1/roles", `{"codigo":"1mal","nombre":"x"}`, tk), http.StatusUnprocessableEntity, "codigo_invalido")
	esperar(t, post(t, "/api/v1/roles", `{"codigo":"BIEN","nombre":"   "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/roles", `{"codigo":"BIEN","nombre":"x","extra":1}`, tk), http.StatusBadRequest, "json_invalido")

	// Lectura, edición parcial y edición que vacía la descripción (null).
	esperar(t, call(t, "GET", "/api/v1/roles/99999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/roles/abc", "", tk), http.StatusBadRequest, "id_invalido")
	g := call(t, "GET", "/api/v1/roles/"+rol.ID, "", tk)
	esperar(t, g, http.StatusOK, "")

	p := callH(t, "PATCH", "/api/v1/roles/"+rol.ID, `{"nombre":"Jefa de sala","descripcion":"Coordina el salón"}`, tk, nil)
	esperar(t, p, http.StatusOK, "")
	var ed entidad
	p.json(t, &ed)
	if ed.Nombre != "Jefa de sala" || ed.Desc == nil || *ed.Desc != "Coordina el salón" || ed.Codigo != "JEFE_SALA" {
		t.Fatalf("edición parcial: %+v", ed)
	}
	p = callH(t, "PATCH", "/api/v1/roles/"+rol.ID, `{"descripcion":null}`, tk, nil)
	esperar(t, p, http.StatusOK, "")
	p.json(t, &ed)
	if ed.Desc != nil || ed.Nombre != "Jefa de sala" {
		t.Fatalf("null debe vaciar solo la descripción: %+v", ed)
	}
	esperar(t, callH(t, "PATCH", "/api/v1/roles/"+rol.ID, `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/roles/99999", `{"nombre":"x"}`, tk, nil), http.StatusNotFound, "no_encontrado")

	// Baja y restauración (sin cuerpo y con "{}").
	b := post(t, "/api/v1/roles/"+rol.ID+"/baja", "", tk)
	esperar(t, b, http.StatusOK, "")
	b.json(t, &ed)
	if ed.EsActivo {
		t.Fatal("la baja debe dejar el rol inactivo")
	}
	rs := post(t, "/api/v1/roles/"+rol.ID+"/restauracion", `{}`, tk)
	esperar(t, rs, http.StatusOK, "")
	rs.json(t, &ed)
	if !ed.EsActivo {
		t.Fatal("la restauración debe reactivar")
	}
}

func TestPaginacionPorCursorDeRoles(t *testing.T) {
	tk := adminToken(t)
	for i := 0; i < 4; i++ {
		esperar(t, post(t, "/api/v1/roles", fmt.Sprintf(`{"codigo":"PAG_%d","nombre":"Paginado %d"}`, i, i), tk), http.StatusCreated, "")
	}
	total := scalar(t, `SELECT count(*) FROM sch_restaurante.roles`)

	var vistos []string
	cursor := ""
	for paginas := 0; paginas < 20; paginas++ {
		ruta := "/api/v1/roles?limit=3"
		if cursor != "" {
			ruta += "&cursor=" + cursor
		}
		r := call(t, "GET", ruta, "", tk)
		esperar(t, r, http.StatusOK, "")
		var pag struct {
			Items      []entidad `json:"items"`
			NextCursor *string   `json:"nextCursor"`
		}
		r.json(t, &pag)
		if len(pag.Items) == 0 || len(pag.Items) > 3 {
			t.Fatalf("página con %d elementos", len(pag.Items))
		}
		for _, it := range pag.Items {
			vistos = append(vistos, it.ID)
		}
		if pag.NextCursor == nil {
			break
		}
		cursor = *pag.NextCursor
	}
	if len(vistos) != total {
		t.Fatalf("se recorrieron %d roles, hay %d", len(vistos), total)
	}
	seen := map[string]bool{}
	var anterior int
	for _, id := range vistos {
		if seen[id] {
			t.Fatalf("id repetido entre páginas: %s", id)
		}
		seen[id] = true
		var n int
		fmt.Sscan(id, &n)
		if n <= anterior {
			t.Fatalf("orden no estable: %d después de %d", n, anterior)
		}
		anterior = n
	}
	esperar(t, call(t, "GET", "/api/v1/roles?limit=0", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/roles?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/roles?cursor=%%%", "", tk), http.StatusBadRequest, "query_invalida")
	esperar(t, call(t, "GET", "/api/v1/roles?cursor=abc", "", tk), http.StatusBadRequest, "cursor_invalido")
}

func TestPermisosCiclo(t *testing.T) {
	tk := adminToken(t)
	r := post(t, "/api/v1/permisos", `{"codigo":"reporte_ver","descripcion":"Ver reportes"}`, tk)
	esperar(t, r, http.StatusCreated, "")
	var p entidad
	r.json(t, &p)
	if p.Codigo != "REPORTE_VER" {
		t.Fatalf("permiso: %+v", p)
	}
	esperar(t, post(t, "/api/v1/permisos", `{"codigo":"REPORTE_VER","descripcion":"otra"}`, tk), http.StatusConflict, "registro_duplicado")
	esperar(t, callH(t, "PATCH", "/api/v1/permisos/"+p.ID, `{"descripcion":"Ver reportes de ventas"}`, tk, nil), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/permisos/"+p.ID+"/baja", "", tk), http.StatusOK, "")
	rs := post(t, "/api/v1/permisos/"+p.ID+"/restauracion", "", tk)
	esperar(t, rs, http.StatusOK, "")
	rs.json(t, &p)
	if !p.EsActivo {
		t.Fatal("debe quedar activo")
	}
	l := call(t, "GET", "/api/v1/permisos?limit=100", "", tk)
	esperar(t, l, http.StatusOK, "")
	if !strings.Contains(l.Body.String(), "USUARIO_ADMINISTRAR") {
		t.Fatal("el listado debe incluir los permisos de la semilla")
	}
}

type usuarioResp struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	EsSistema bool   `json:"es_sistema"`
	EsActivo  bool   `json:"es_activo"`
}

func TestUsuariosAltaYAsignaciones(t *testing.T) {
	tk := adminToken(t)
	clave := nuevaClave()
	cuerpo := `{"username":"Mesero.Uno","nombre":"Ana","email":"ana@example.com","password":"MeseroClave2026"}`
	r := callH(t, "POST", "/api/v1/usuarios", cuerpo, tk, map[string]string{"Idempotency-Key": clave})
	esperar(t, r, http.StatusCreated, "")
	var u usuarioResp
	r.json(t, &u)
	if u.EsSistema || !u.EsActivo || u.Username != "Mesero.Uno" {
		t.Fatalf("usuario: %+v", u)
	}
	for _, prohibido := range []string{"password", "argon2", "hash"} {
		if strings.Contains(strings.ToLower(r.Body.String()), prohibido) {
			t.Fatalf("la respuesta no debe exponer %q: %s", prohibido, r.Body.String())
		}
	}
	// Repetir la intención no duplica al usuario.
	r2 := callH(t, "POST", "/api/v1/usuarios", cuerpo, tk, map[string]string{"Idempotency-Key": clave})
	esperar(t, r2, http.StatusCreated, "")
	var rep usuarioResp
	r2.json(t, &rep)
	if rep.ID != u.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE lower(username)='mesero.uno'`) != 1 {
		t.Fatal("la repetición debe devolver el mismo usuario")
	}
	// La misma clave con otra contraseña es otro contenido.
	cambiada := strings.Replace(cuerpo, "MeseroClave2026", "OtraClaveDistinta2026", 1)
	esperar(t, callH(t, "POST", "/api/v1/usuarios", cambiada, tk, map[string]string{"Idempotency-Key": clave}), http.StatusConflict, "idempotency_key_reutilizada")
	// Validaciones y unicidad (sin distinguir mayúsculas).
	esperar(t, post(t, "/api/v1/usuarios", `{"username":"x1","password":"corta"}`, tk), http.StatusUnprocessableEntity, "clave_debil")
	esperar(t, post(t, "/api/v1/usuarios", `{"username":"x2","password":"SoloLetrasSinNumeros"}`, tk), http.StatusUnprocessableEntity, "clave_debil")
	esperar(t, post(t, "/api/v1/usuarios", `{"username":"x3","password":"Valida2026Clave","email":"no-es-correo"}`, tk), http.StatusUnprocessableEntity, "email_invalido")
	esperar(t, post(t, "/api/v1/usuarios", `{"username":"MESERO.UNO","password":"Valida2026Clave"}`, tk), http.StatusConflict, "registro_duplicado")

	// Sin roles ni sedes puede autenticarse pero no tiene permisos.
	nuevo := token(t, "Mesero.Uno", "MeseroClave2026")
	s := call(t, "GET", "/api/v1/sesion", "", nuevo)
	esperar(t, s, http.StatusOK, "")
	if strings.Contains(s.Body.String(), "PEDIDO") {
		t.Fatal("sin asignaciones no hay permisos")
	}

	// Asignar rol y sede.
	mesero, cocina, sede := idDe(t, "roles", "codigo", "MESERO"), idDe(t, "roles", "codigo", "COCINA"), idDe(t, "sucursales", "codigo", "PRINCIPAL")
	asignar := func(rol string) resp {
		return callH(t, "PUT", "/api/v1/usuarios/"+u.ID+"/asignaciones",
			fmt.Sprintf(`{"rol_ids":["%s"],"sucursal_ids":["%s"]}`, rol, sede), tk, nil)
	}
	esperar(t, asignar(mesero), http.StatusOK, "")
	esperar(t, asignar(mesero), http.StatusOK, "") // repetir no duplica filas
	filasRoles := func() int {
		return scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios_roles WHERE usuario_id=`+u.ID)
	}
	activos := func() int {
		return scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios_roles WHERE es_activo AND usuario_id=`+u.ID)
	}
	if filasRoles() != 1 || activos() != 1 {
		t.Fatalf("filas=%d activas=%d, quería 1/1", filasRoles(), activos())
	}
	// Cambiar de rol: el anterior queda inactivo (no se borra) y volver lo reactiva sin duplicar.
	esperar(t, asignar(cocina), http.StatusOK, "")
	if filasRoles() != 2 || activos() != 1 {
		t.Fatalf("tras cambiar: filas=%d activas=%d, quería 2/1", filasRoles(), activos())
	}
	esperar(t, asignar(mesero), http.StatusOK, "")
	if filasRoles() != 2 || activos() != 1 {
		t.Fatalf("tras volver: filas=%d activas=%d, quería 2/1 (reactiva, no duplica)", filasRoles(), activos())
	}
	// Validaciones de asignación.
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/"+u.ID+"/asignaciones", `{"rol_ids":[],"sucursal_ids":["`+sede+`"]}`, tk, nil), http.StatusUnprocessableEntity, "asignaciones_invalidas")
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/"+u.ID+"/asignaciones", `{"rol_ids":["999999"],"sucursal_ids":["`+sede+`"]}`, tk, nil), http.StatusUnprocessableEntity, "regla_de_datos")
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/"+u.ID+"/asignaciones", `{"rol_ids":["x"],"sucursal_ids":["1"]}`, tk, nil), http.StatusBadRequest, "id_invalido")
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/999999/asignaciones", fmt.Sprintf(`{"rol_ids":["%s"],"sucursal_ids":["%s"]}`, mesero, sede), tk, nil), http.StatusNotFound, "no_encontrado")
}

// Un administrador solo agrega o quita sedes donde él mismo administra accesos.
func TestAsignacionDeSedeAjenaSeRechaza(t *testing.T) {
	tk := adminToken(t)
	asSystem(t, `INSERT INTO sch_restaurante.sucursales(negocio_id, codigo, nombre, direccion)
		SELECT id, 'SEGUNDA', 'Segunda sede', 'Calle 2' FROM sch_restaurante.negocio`)
	crearUsuario(t, "sedeajena", "SedeAjena2026Clave", "CAJERO")
	uid := idDe(t, "usuarios", "username", "sedeajena")
	cajero, p, s2 := idDe(t, "roles", "codigo", "CAJERO"), idDe(t, "sucursales", "codigo", "PRINCIPAL"), idDe(t, "sucursales", "codigo", "SEGUNDA")
	r := callH(t, "PUT", "/api/v1/usuarios/"+uid+"/asignaciones",
		fmt.Sprintf(`{"rol_ids":["%s"],"sucursal_ids":["%s","%s"]}`, cajero, p, s2), tk, nil)
	esperar(t, r, http.StatusForbidden, "sede_no_autorizada")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios_sucursales WHERE usuario_id=`+uid+` AND sucursal_id=`+s2); n != 0 {
		t.Fatal("el rechazo no debe asignar la sede")
	}
}

func TestBajaDeUsuarioRevocaSesionesYNoSeRevive(t *testing.T) {
	tk := adminToken(t)
	crearUsuario(t, "temporal", "Temporal2026Clave", "CAJERO")
	uid := idDe(t, "usuarios", "username", "temporal")
	viejo := token(t, "temporal", "Temporal2026Clave")
	esperar(t, call(t, "GET", "/api/v1/sesion", "", viejo), http.StatusOK, "")

	esperar(t, post(t, "/api/v1/usuarios/"+uid+"/baja", `{"motivo":""}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	b := post(t, "/api/v1/usuarios/"+uid+"/baja", `{"motivo":"fin de contrato"}`, tk)
	esperar(t, b, http.StatusOK, "")
	var u usuarioResp
	b.json(t, &u)
	if u.EsActivo {
		t.Fatal("debe quedar inactivo")
	}
	esperar(t, call(t, "GET", "/api/v1/sesion", "", viejo), http.StatusUnauthorized, "no_autenticado")
	esperar(t, login(t, "temporal", "Temporal2026Clave"), http.StatusUnauthorized, "credenciales_invalidas")

	rs := post(t, "/api/v1/usuarios/"+uid+"/restauracion", "", tk)
	esperar(t, rs, http.StatusOK, "")
	rs.json(t, &u)
	if !u.EsActivo {
		t.Fatal("debe quedar activo")
	}
	// Restaurar al usuario NO revive su sesión anterior (fue revocada); debe iniciar una nueva.
	esperar(t, call(t, "GET", "/api/v1/sesion", "", viejo), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/sesion", "", token(t, "temporal", "Temporal2026Clave")), http.StatusOK, "")
	// El motivo y quién dio la baja quedan registrados en el historial de auditoría.
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos
		WHERE tabla='usuarios' AND registro_id=`+uid+` AND creado_por=`+fmt.Sprint(adminID(t))+`
		  AND datos_despues::text LIKE '%fin de contrato%'`); n < 1 {
		t.Fatal("la baja debe auditarse con el administrador como actor y el motivo")
	}
}

func TestReglasSobreElPropioAdminYElActorTecnico(t *testing.T) {
	tk := adminToken(t)
	esperar(t, post(t, "/api/v1/usuarios/"+fmt.Sprint(adminID(t))+"/baja", `{"motivo":"x"}`, tk), http.StatusUnprocessableEntity, "no_autobaja")
	sistema := fmt.Sprint(scalar(t, `SELECT id FROM sch_restaurante.usuarios WHERE es_sistema`))
	esperar(t, post(t, "/api/v1/usuarios/"+sistema+"/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/"+sistema+"/asignaciones",
		`{"rol_ids":["1"],"sucursal_ids":["1"]}`, tk, nil), http.StatusNotFound, "no_encontrado")
}

// Ningún cambio puede dejar el sistema sin un administrador efectivo: se revierte.
func TestNoSePuedeDejarElSistemaSinAdministrador(t *testing.T) {
	tk := adminToken(t)
	crearUsuario(t, "admin2", "Admin2Clave2026Ok", "ADMIN")
	admin2 := idDe(t, "usuarios", "username", "admin2")
	if n := scalar(t, `SELECT count(DISTINCT usuario_id) FROM sch_restaurante.v_permisos_efectivos WHERE permiso='USUARIO_ADMINISTRAR'`); n != 2 {
		t.Fatalf("precondición: 2 administradores efectivos, hay %d", n)
	}
	// Con dos administradores se puede dar de baja a uno.
	esperar(t, post(t, "/api/v1/usuarios/"+admin2+"/baja", `{"motivo":"rotación"}`, tk), http.StatusOK, "")

	adminRol := idDe(t, "roles", "codigo", "ADMIN")
	esperar(t, post(t, "/api/v1/roles/"+adminRol+"/baja", "", tk), http.StatusUnprocessableEntity, "ultimo_administrador")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.roles WHERE codigo='ADMIN' AND es_activo`); n != 1 {
		t.Fatal("el rechazo debe revertir la baja del rol")
	}
	permiso := idDe(t, "permisos", "codigo", "USUARIO_ADMINISTRAR")
	esperar(t, post(t, "/api/v1/permisos/"+permiso+"/baja", "", tk), http.StatusUnprocessableEntity, "ultimo_administrador")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo='USUARIO_ADMINISTRAR' AND es_activo`); n != 1 {
		t.Fatal("el rechazo debe revertir la baja del permiso")
	}
	// Quitarle el rol ADMIN al único administrador también se revierte.
	cajero, sede := idDe(t, "roles", "codigo", "CAJERO"), idDe(t, "sucursales", "codigo", "PRINCIPAL")
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/"+fmt.Sprint(adminID(t))+"/asignaciones",
		fmt.Sprintf(`{"rol_ids":["%s"],"sucursal_ids":["%s"]}`, cajero, sede), tk, nil), http.StatusUnprocessableEntity, "ultimo_administrador")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios_roles WHERE es_activo AND usuario_id=`+fmt.Sprint(adminID(t))+` AND rol_id=`+adminRol); n != 1 {
		t.Fatal("el rechazo debe conservar el rol ADMIN del administrador")
	}
	// El administrador sigue pudiendo operar.
	esperar(t, call(t, "GET", "/api/v1/sesion", "", tk), http.StatusOK, "")
}

// La V4 exige un actor: el audit trail de las escrituras de la API apunta al administrador autenticado.
func TestEscriturasDeLaAPIAuditanAlActor(t *testing.T) {
	tk := adminToken(t)
	r := post(t, "/api/v1/roles", `{"codigo":"AUDITADO","nombre":"Auditado"}`, tk)
	esperar(t, r, http.StatusCreated, "")
	var rol entidad
	r.json(t, &rol)
	if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM sch_restaurante.auditoria_eventos
		WHERE tabla='roles' AND accion='INSERT' AND registro_id=%s AND creado_por=%d`, rol.ID, adminID(t))); n != 1 {
		t.Fatalf("eventos con el administrador como actor = %d, quería 1", n)
	}
	if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM sch_restaurante.roles WHERE id=%s AND creado_por=%d AND actualizado_por=%d`,
		rol.ID, adminID(t), adminID(t))); n != 1 {
		t.Fatal("creado_por y actualizado_por deben ser el administrador autenticado, nunca un dato del cliente")
	}
	// El cuerpo no puede suplantar la autoría.
	esperar(t, post(t, "/api/v1/roles", `{"codigo":"SUPLANTA","nombre":"x","creado_por":"1"}`, tk), http.StatusBadRequest, "json_invalido")
}

// RES-CP-033: un rol inactivo deja de conceder permisos aunque la asignación exista.
func TestRolInactivoDejaDeConcederPermisos(t *testing.T) {
	tk := adminToken(t)
	r := post(t, "/api/v1/roles", `{"codigo":"ROL_PRUEBA_PERMISOS","nombre":"Rol de prueba"}`, tk)
	esperar(t, r, http.StatusCreated, "")
	var rol entidad
	r.json(t, &rol)
	// La API no expone la asignación rol-permiso (hueco del contrato): se prepara por SQL.
	asSystem(t, `INSERT INTO sch_restaurante.roles_permisos(rol_id, permiso_id)
		SELECT r.id, p.id FROM sch_restaurante.roles r, sch_restaurante.permisos p
		WHERE r.codigo='ROL_PRUEBA_PERMISOS' AND p.codigo='PEDIDO_CREAR'`)
	crearUsuario(t, "conrolprueba", "ConRol2026ClaveOk", "ROL_PRUEBA_PERMISOS")
	usuario := token(t, "conrolprueba", "ConRol2026ClaveOk")

	permisos := func() string {
		s := call(t, "GET", "/api/v1/sesion", "", usuario)
		esperar(t, s, http.StatusOK, "")
		return s.Body.String()
	}
	if !strings.Contains(permisos(), "PEDIDO_CREAR") {
		t.Fatal("el rol activo debe conceder PEDIDO_CREAR")
	}
	esperar(t, post(t, "/api/v1/roles/"+rol.ID+"/baja", "", tk), http.StatusOK, "")
	if strings.Contains(permisos(), "PEDIDO_CREAR") {
		t.Fatal("con el rol inactivo no debe conceder permisos, aunque la asignación exista")
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios_roles ur JOIN sch_restaurante.roles r ON r.id=ur.rol_id
		JOIN sch_restaurante.usuarios u ON u.id=ur.usuario_id WHERE u.username='conrolprueba' AND r.codigo='ROL_PRUEBA_PERMISOS' AND ur.es_activo`); n != 1 {
		t.Fatal("la asignación usuario-rol debe seguir existiendo y activa")
	}
	esperar(t, post(t, "/api/v1/roles/"+rol.ID+"/restauracion", "", tk), http.StatusOK, "")
	if !strings.Contains(permisos(), "PEDIDO_CREAR") {
		t.Fatal("al restaurar el rol vuelve a conceder el permiso")
	}
}

type equipoItem struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	EsSistema  bool    `json:"es_sistema"`
	EsActivo   bool    `json:"es_activo"`
	MotivoBaja *string `json:"motivo_baja"`
	Roles      []struct {
		ID       string `json:"id"`
		Codigo   string `json:"codigo"`
		EsActivo bool   `json:"es_activo"`
	} `json:"roles"`
	Sucursales []struct {
		ID     string `json:"id"`
		Codigo string `json:"codigo"`
	} `json:"sucursales"`
}

func listarEquipo(t *testing.T, tk string) []equipoItem {
	t.Helper()
	var todos []equipoItem
	cursor := ""
	for pagina := 0; pagina < 50; pagina++ {
		ruta := "/api/v1/usuarios?limit=5"
		if cursor != "" {
			ruta += "&cursor=" + cursor
		}
		r := call(t, "GET", ruta, "", tk)
		esperar(t, r, http.StatusOK, "")
		var p struct {
			Items      []equipoItem `json:"items"`
			NextCursor *string      `json:"nextCursor"`
		}
		r.json(t, &p)
		todos = append(todos, p.Items...)
		if p.NextCursor == nil {
			return todos
		}
		cursor = *p.NextCursor
	}
	t.Fatal("la paginación no termina")
	return nil
}

func TestEquipoListaPersonasConSusAccesosVigentes(t *testing.T) {
	tk := adminToken(t)
	crearUsuario(t, "equipo.cajero", "EquipoCajero2026Ok", "CAJERO")
	crearUsuario(t, "equipo.baja", "EquipoBaja2026Ok", "MESERO")
	bajaID := idDe(t, "usuarios", "username", "equipo.baja")
	esperar(t, post(t, "/api/v1/usuarios/"+bajaID+"/baja", `{"motivo":"cambio de turno"}`, tk), http.StatusOK, "")

	equipo := listarEquipo(t, tk)
	porNombre := map[string]equipoItem{}
	ids := map[string]bool{}
	for _, u := range equipo {
		if ids[u.ID] {
			t.Fatalf("persona repetida entre páginas: %s", u.ID)
		}
		ids[u.ID] = true
		porNombre[u.Username] = u
		if u.EsSistema || u.Username == "__sistema__" {
			t.Fatal("el actor técnico nunca debe aparecer en el equipo")
		}
	}
	if len(equipo) != scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE NOT es_sistema`) {
		t.Fatalf("el listado debe incluir a todas las personas (activas y de baja): %d", len(equipo))
	}

	c := porNombre["equipo.cajero"]
	if !c.EsActivo || len(c.Roles) != 1 || c.Roles[0].Codigo != "CAJERO" || !c.Roles[0].EsActivo ||
		len(c.Sucursales) != 1 || c.Sucursales[0].Codigo != "PRINCIPAL" {
		t.Fatalf("accesos del cajero: %+v", c)
	}
	b := porNombre["equipo.baja"]
	if b.EsActivo || b.MotivoBaja == nil || *b.MotivoBaja != "cambio de turno" {
		t.Fatalf("la persona de baja debe mostrar su motivo: %+v", b)
	}

	// Solo asignaciones vigentes: al cambiar de rol, el anterior deja de listarse.
	sede := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	cocina := idDe(t, "roles", "codigo", "COCINA")
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/"+c.ID+"/asignaciones",
		fmt.Sprintf(`{"rol_ids":["%s"],"sucursal_ids":["%s"]}`, cocina, sede), tk, nil), http.StatusOK, "")
	for _, u := range listarEquipo(t, tk) {
		if u.ID == c.ID && (len(u.Roles) != 1 || u.Roles[0].Codigo != "COCINA") {
			t.Fatalf("debe listar solo el rol vigente: %+v", u.Roles)
		}
	}

	// Nunca se expone el hash de contraseña.
	raw := call(t, "GET", "/api/v1/usuarios?limit=100", "", tk).Body.String()
	for _, prohibido := range []string{"password", "argon2", "hash"} {
		if strings.Contains(strings.ToLower(raw), prohibido) {
			t.Fatalf("el listado no debe exponer %q", prohibido)
		}
	}
	esperar(t, call(t, "GET", "/api/v1/usuarios?limit=0", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/usuarios?cursor=%%%", "", tk), http.StatusBadRequest, "query_invalida")
}

func TestEquipoYSedesAsignablesExigenAdministrador(t *testing.T) {
	crearUsuario(t, "equipo.sinpoder", "EquipoSin2026Ok", "CAJERO")
	tk := token(t, "equipo.sinpoder", "EquipoSin2026Ok")
	esperar(t, call(t, "GET", "/api/v1/usuarios", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/sucursales-asignables", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/usuarios", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/sucursales-asignables", "", ""), http.StatusUnauthorized, "no_autenticado")
}

// Las sedes que se ofrecen son exactamente las que la regla de asignación permite.
func TestSucursalesAsignablesSonLasQueElAdministradorAdministra(t *testing.T) {
	tk := adminToken(t)
	asSystem(t, `INSERT INTO sch_restaurante.sucursales(negocio_id, codigo, nombre, direccion)
		SELECT id, 'TERCERA', 'Tercera sede', 'Calle 3' FROM sch_restaurante.negocio
		WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.sucursales WHERE codigo='TERCERA')`)
	r := call(t, "GET", "/api/v1/sucursales-asignables", "", tk)
	esperar(t, r, http.StatusOK, "")
	var l struct {
		Items []struct {
			ID     string `json:"id"`
			Codigo string `json:"codigo"`
			Nombre string `json:"nombre"`
		} `json:"items"`
	}
	r.json(t, &l)
	codigos := []string{}
	for _, s := range l.Items {
		codigos = append(codigos, s.Codigo)
	}
	// El administrador solo administra PRINCIPAL: las otras sedes existen pero no se ofrecen.
	if len(codigos) != 1 || codigos[0] != "PRINCIPAL" || l.Items[0].Nombre == "" {
		t.Fatalf("sedes ofrecidas: %v (existen más sedes, pero no son suyas)", codigos)
	}
	// Coherencia con la regla: asignar una sede no ofrecida se rechaza.
	crearUsuario(t, "equipo.sede", "EquipoSede2026Ok", "CAJERO")
	uid := idDe(t, "usuarios", "username", "equipo.sede")
	cajero, tercera := idDe(t, "roles", "codigo", "CAJERO"), idDe(t, "sucursales", "codigo", "TERCERA")
	esperar(t, callH(t, "PUT", "/api/v1/usuarios/"+uid+"/asignaciones",
		fmt.Sprintf(`{"rol_ids":["%s"],"sucursal_ids":["%s"]}`, cajero, tercera), tk, nil), http.StatusForbidden, "sede_no_autorizada")
}
