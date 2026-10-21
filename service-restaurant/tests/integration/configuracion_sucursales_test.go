package integration

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func sortedCopy(in []string) []string {
	out := slices.Clone(in)
	sort.Strings(out)
	return out
}

type sedesPagina struct {
	Items      []sedeResp `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

func (p sedesPagina) codigos() []string {
	out := make([]string, 0, len(p.Items))
	for _, s := range p.Items {
		out = append(out, s.Codigo)
	}
	sort.Strings(out)
	return out
}

func listarSedes(t *testing.T, query, tk string) sedesPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/sucursales"+query, "", tk)
	esperar(t, r, http.StatusOK, "")
	var p sedesPagina
	r.json(t, &p)
	return p
}

func crearSedeInactivaSQL(t *testing.T, codigo string) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.sucursales(negocio_id, codigo, nombre, direccion, es_activo, motivo_eliminacion)
		SELECT id, '`+codigo+`', 'Sede `+codigo+`', 'Calle cerrada 1', false, 'Local cerrado' FROM sch_restaurante.negocio
		WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.sucursales WHERE codigo='`+codigo+`')`)
}

func TestSucursalesSoloMuestraLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "CFG-A")
	crearSedeSQL(t, "CFG-B")
	rolConPermisos(t, "T_SUC_VER", "SUCURSAL_VER")
	asegurarUsuario(t, "suc.ver", "SucVer2026Clave", "T_SUC_VER", "PRINCIPAL", "CFG-A")
	tk := token(t, "suc.ver", "SucVer2026Clave")

	if got := listarSedes(t, "", tk).codigos(); !slices.Equal(got, []string{"CFG-A", "PRINCIPAL"}) {
		t.Fatalf("sedes visibles = %v, quería [CFG-A PRINCIPAL] (CFG-B no es suya)", got)
	}
	// El administrador instalado solo tiene PRINCIPAL asignada. Como administra sedes también ve
	// las inactivas de cualquier sede, por eso se filtra por activas para no depender del orden.
	if got := listarSedes(t, "?es_activo=true", adminToken(t)).codigos(); !slices.Equal(got, []string{"PRINCIPAL"}) {
		t.Fatalf("el admin ve %v, quería [PRINCIPAL]", got)
	}

	propia := idDe(t, "sucursales", "codigo", "CFG-A")
	r := call(t, "GET", "/api/v1/sucursales/"+propia, "", tk)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.Codigo != "CFG-A" || !s.EsActivo || s.NegocioID == "" || s.Motivo != nil {
		t.Fatalf("sucursal: %+v", s)
	}
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+idDe(t, "sucursales", "codigo", "CFG-B"), "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/sucursales/999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/sucursales/abc", "", tk), http.StatusBadRequest, "id_invalido")
}

func TestSucursalesExigeAlgunPermisoDeSucursal(t *testing.T) {
	asegurarUsuario(t, "suc.sin", "SucSin2026Clave", "CAJERO", "PRINCIPAL")
	tk := token(t, "suc.sin", "SucSin2026Clave")
	esperar(t, call(t, "GET", "/api/v1/sucursales", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/sucursales/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/sucursales", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func TestSucursalesFiltroPaginacionEInactivas(t *testing.T) {
	crearSedeSQL(t, "CFG-A")
	crearSedeInactivaSQL(t, "CFG-INACTIVA")
	rolConPermisos(t, "T_SUC_VER", "SUCURSAL_VER")
	rolConPermisos(t, "T_SUC_GESTOR", "SUCURSAL_VER", "SUCURSAL_ADMINISTRAR")
	asegurarUsuario(t, "suc.ver", "SucVer2026Clave", "T_SUC_VER", "PRINCIPAL", "CFG-A")
	asegurarUsuario(t, "suc.gestor", "SucGestor2026Clave", "T_SUC_GESTOR", "PRINCIPAL")
	ver, gestor := token(t, "suc.ver", "SucVer2026Clave"), token(t, "suc.gestor", "SucGestor2026Clave")

	// Quien solo ve no descubre sedes inactivas; quien administra sedes sí (para restaurarlas).
	if got := listarSedes(t, "?es_activo=false", ver).codigos(); len(got) != 0 {
		t.Fatalf("VER no debe ver inactivas: %v", got)
	}
	if got := listarSedes(t, "?es_activo=false", gestor).codigos(); !slices.Contains(got, "CFG-INACTIVA") {
		t.Fatalf("el gestor debe ver las inactivas: %v", got)
	}
	r := call(t, "GET", "/api/v1/sucursales/"+idDe(t, "sucursales", "codigo", "CFG-INACTIVA"), "", gestor)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.EsActivo || s.Motivo == nil || *s.Motivo != "Local cerrado" || s.EliminadoPor == nil {
		t.Fatalf("sede inactiva: %+v", s)
	}
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+s.ID, "", ver), http.StatusNotFound, "no_encontrado")
	for _, x := range listarSedes(t, "?es_activo=true", gestor).Items {
		if !x.EsActivo {
			t.Fatalf("es_activo=true devolvió una inactiva: %+v", x)
		}
	}
	esperar(t, call(t, "GET", "/api/v1/sucursales?es_activo=quizas", "", gestor), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/sucursales?limit=0", "", gestor), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/sucursales?cursor=%25%25", "", gestor), http.StatusBadRequest, "cursor_invalido")
	esperar(t, call(t, "GET", "/api/v1/sucursales?cursor=!!!", "", gestor), http.StatusBadRequest, "cursor_invalido")

	// Paginación por cursor sobre las dos sedes de suc.ver.
	p1 := listarSedes(t, "?limit=1", ver)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarSedes(t, "?limit=1&cursor="+*p1.NextCursor, ver)
	if len(p2.Items) != 1 || p2.Items[0].ID == p1.Items[0].ID || p2.NextCursor != nil {
		t.Fatalf("segunda página: %+v", p2)
	}
	id1, err1 := strconv.Atoi(p1.Items[0].ID)
	id2, err2 := strconv.Atoi(p2.Items[0].ID)
	if err1 != nil || err2 != nil || id1 >= id2 {
		t.Fatalf("orden ascendente por id roto: %q (%v) vs %q (%v)", p1.Items[0].ID, err1, p2.Items[0].ID, err2)
	}
	if got := []string{p1.Items[0].Codigo, p2.Items[0].Codigo}; !slices.Equal(sortedCopy(got), []string{"CFG-A", "PRINCIPAL"}) {
		t.Fatalf("las dos páginas suman %v, quería [CFG-A PRINCIPAL]", got)
	}
}

const gestorUser, gestorPass = "suc.gestor", "SucGestor2026Clave"

func gestorToken(t *testing.T) string {
	t.Helper()
	rolConPermisos(t, "T_SUC_GESTOR", "SUCURSAL_VER", "SUCURSAL_ADMINISTRAR")
	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "PRINCIPAL")
	return token(t, gestorUser, gestorPass)
}

func altaSede(codigo string) string {
	return fmt.Sprintf(`{"codigo":"%s","nombre":"Sede %s","direccion":"Av. Nueva 123","ubigeo":"150101","telefono":"01-555-0000"}`, codigo, codigo)
}

func TestSucursalCrearConIdempotenciaYValidaciones(t *testing.T) {
	tk := gestorToken(t)
	clave := nuevaClave()
	h := map[string]string{"Idempotency-Key": clave}

	r1 := callH(t, "POST", "/api/v1/sucursales", altaSede("nueva-1"), tk, h)
	esperar(t, r1, http.StatusCreated, "")
	var s sedeResp
	r1.json(t, &s)
	if s.Codigo != "NUEVA-1" || !s.EsActivo || s.NegocioID == "" || s.CreadoPor != idDe(t, "usuarios", "username", gestorUser) ||
		s.Ubigeo == nil || *s.Ubigeo != "150101" {
		t.Fatalf("sede creada: %+v", s)
	}
	// Repetir la misma intención no duplica; otro contenido con la misma clave es conflicto.
	r2 := callH(t, "POST", "/api/v1/sucursales", altaSede("nueva-1"), tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var rep sedeResp
	r2.json(t, &rep)
	if rep.ID != s.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE codigo='NUEVA-1'`) != 1 {
		t.Fatalf("la repetición debe devolver la misma sede: %+v vs %+v", rep, s)
	}
	esperar(t, callH(t, "POST", "/api/v1/sucursales", altaSede("otra-9"), tk, h), http.StatusConflict, "idempotency_key_reutilizada")

	// Otra clave con el mismo código: conflicto de unicidad específico del campo.
	esperar(t, post(t, "/api/v1/sucursales", altaSede("NUEVA-1"), tk), http.StatusConflict, "codigo_duplicado")
	// Sin clave, con campos que el cliente no puede fijar, y validaciones por campo.
	esperar(t, call(t, "POST", "/api/v1/sucursales", altaSede("sin-clave"), tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"CON-NEG","nombre":"x","direccion":"y","negocio_id":"1"}`, tk), http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"creado_por","nombre":"x","direccion":"y","creado_por":"1"}`, tk), http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"x","nombre":"x","direccion":"y"}`, tk), http.StatusUnprocessableEntity, "codigo_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"BIEN-1","nombre":" ","direccion":"y"}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"BIEN-1","nombre":"x","direccion":""}`, tk), http.StatusUnprocessableEntity, "direccion_invalida")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"BIEN-1","nombre":"x","direccion":"y","ubigeo":"12"}`, tk), http.StatusUnprocessableEntity, "ubigeo_invalido")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE codigo IN ('BIEN-1','SIN-CLAVE','CON-NEG')`) != 0 {
		t.Fatal("los rechazos no deben dejar sedes")
	}

	// Sin SUCURSAL_ADMINISTRAR no se crea.
	rolConPermisos(t, "T_SUC_VER", "SUCURSAL_VER")
	asegurarUsuario(t, "suc.ver.w", "SucVerW2026Clave", "T_SUC_VER", "PRINCIPAL")
	esperar(t, post(t, "/api/v1/sucursales", altaSede("SIN-PODER"), token(t, "suc.ver.w", "SucVerW2026Clave")), http.StatusForbidden, "sin_permiso")
}

func TestSucursalCreadaNoDaAccesoAlCreadorHastaAsignarla(t *testing.T) {
	tk := gestorToken(t)
	r := post(t, "/api/v1/sucursales", altaSede("SIN-ACCESO"), tk)
	esperar(t, r, http.StatusCreated, "")
	var s sedeResp
	r.json(t, &s)
	// Los permisos son por sede asignada: el creador aún no la ve ni la administra.
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+s.ID, "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+s.ID, `{"nombre":"X"}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/sucursales/"+s.ID+"/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")

	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "SIN-ACCESO")
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+s.ID, "", tk), http.StatusOK, "")
}

func TestSucursalEditar(t *testing.T) {
	tk := gestorToken(t)
	crearSedeSQL(t, "EDITABLE")
	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "EDITABLE")
	id := idDe(t, "sucursales", "codigo", "EDITABLE")

	r := callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"nombre":"Sede Editada","telefono":"999","codigo_establecimiento_sunat":"0001"}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.Nombre != "Sede Editada" || s.Telefono == nil || *s.Telefono != "999" || s.CodigoSunat == nil || *s.CodigoSunat != "0001" || s.Codigo != "EDITABLE" {
		t.Fatalf("edición parcial: %+v", s)
	}
	r = callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"telefono":null}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &s)
	if s.Telefono != nil || s.CodigoSunat == nil || s.Nombre != "Sede Editada" {
		t.Fatalf("null debe vaciar solo el teléfono: %+v", s)
	}
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"codigo":"PRINCIPAL"}`, tk, nil), http.StatusConflict, "codigo_duplicado")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"negocio_id":"2"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/999999", `{"nombre":"x"}`, tk, nil), http.StatusNotFound, "no_encontrado")

	// Quien solo tiene SUCURSAL_VER no edita, aunque vea la sede.
	rolConPermisos(t, "T_SUC_VER", "SUCURSAL_VER")
	asegurarUsuario(t, "suc.ver.w", "SucVerW2026Clave", "T_SUC_VER", "EDITABLE")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"nombre":"Hack"}`, token(t, "suc.ver.w", "SucVerW2026Clave"), nil),
		http.StatusForbidden, "sin_permiso")
}

func TestSucursalBajaYRestauracion(t *testing.T) {
	tk := gestorToken(t)
	crearSedeSQL(t, "CICLO")
	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "CICLO")
	id := idDe(t, "sucursales", "codigo", "CICLO")

	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")

	clave := nuevaClave()
	h := map[string]string{"Idempotency-Key": clave}
	r := callH(t, "POST", "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Cierre del local"}`, tk, h)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.EsActivo || s.Motivo == nil || *s.Motivo != "Cierre del local" || s.EliminadoPor == nil ||
		*s.EliminadoPor != idDe(t, "usuarios", "username", gestorUser) {
		t.Fatalf("baja: %+v", s)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='sucursales' AND accion='UPDATE'
		AND registro_id=`+id+` AND creado_por=`+idDe(t, "usuarios", "username", gestorUser)) == 0 {
		t.Fatal("la baja debe quedar auditada con el actor")
	}
	// La misma intención se repite sin nuevo efecto; con otra clave la sede ya está de baja.
	antes := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='sucursales' AND registro_id=`+id)
	esperar(t, callH(t, "POST", "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Cierre del local"}`, tk, h), http.StatusOK, "")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='sucursales' AND registro_id=`+id) != antes {
		t.Fatal("repetir la misma clave no debe volver a escribir")
	}
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Otra vez"}`, tk), http.StatusConflict, "ya_dada_de_baja")

	// Sin permisos sobre la sede inactiva, la restaura quien administra sedes en otra (PRINCIPAL).
	r = post(t, "/api/v1/sucursales/"+id+"/restauracion", `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rest sedeResp
	r.json(t, &rest)
	if rest.ID != s.ID || !rest.EsActivo || rest.Motivo != nil || rest.EliminadoPor != nil {
		t.Fatalf("restauración: %+v (debe conservar el ID y limpiar los marcadores)", rest)
	}
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/restauracion", "", tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/sucursales/999999/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")
}

func TestSucursalNoSePuedeDarDeBajaLaUltimaSedeActiva(t *testing.T) {
	tk := gestorToken(t)
	crearSedeSQL(t, "OTRA-SEDE")
	// Deja PRINCIPAL como única sede activa y lo revierte al terminar.
	const marca = "prueba-ultima-sede"
	asSystem(t, `UPDATE sch_restaurante.sucursales SET es_activo=false, motivo_eliminacion='`+marca+`'
		WHERE codigo<>'PRINCIPAL' AND es_activo`)
	t.Cleanup(func() {
		asSystem(t, `UPDATE sch_restaurante.sucursales SET es_activo=true WHERE motivo_eliminacion='`+marca+`'`)
	})

	id := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Intento"}`, tk), http.StatusUnprocessableEntity, "ultima_sede_activa")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE codigo='PRINCIPAL' AND es_activo`) != 1 {
		t.Fatal("PRINCIPAL debe seguir activa")
	}
}

// Dar de baja una sede quita los permisos efectivos de quienes solo los tenían allí: si era la
// última con USUARIO_ADMINISTRAR efectivo, se rechaza y la baja se revierte.
func TestSucursalNoSePuedeDarDeBajaLaSedeDelUltimoAdministrador(t *testing.T) {
	tk := gestorToken(t)
	crearSedeSQL(t, "CFG-A")
	principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")

	// Neutraliza (con restauración) a cualquier otro administrador efectivo fuera de PRINCIPAL:
	// las pruebas comparten la base y otras pueden haber dejado administradores en otras sedes.
	rows, err := testDB.Query(`SELECT DISTINCT us.id FROM sch_restaurante.usuarios_sucursales us
		JOIN sch_restaurante.v_permisos_efectivos v ON v.usuario_id=us.usuario_id AND v.sucursal_id=us.sucursal_id
		WHERE v.permiso='USUARIO_ADMINISTRAR' AND us.es_activo AND us.sucursal_id<>` + principal)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) > 0 {
		lista := strings.Join(ids, ",")
		asSystem(t, `UPDATE sch_restaurante.usuarios_sucursales SET es_activo=false WHERE id IN (`+lista+`)`)
		t.Cleanup(func() {
			asSystem(t, `UPDATE sch_restaurante.usuarios_sucursales SET es_activo=true WHERE id IN (`+lista+`)`)
		})
	}
	// Precondición: queda un único administrador efectivo y solo en PRINCIPAL, con otra sede activa.
	if n := scalar(t, `SELECT count(DISTINCT usuario_id) FROM sch_restaurante.v_permisos_efectivos
		WHERE permiso='USUARIO_ADMINISTRAR' AND sucursal_id<>`+principal); n != 0 {
		t.Fatalf("precondición: %d administradores efectivos fuera de PRINCIPAL", n)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE es_activo AND codigo<>'PRINCIPAL'`) == 0 {
		t.Fatal("precondición: debe haber otra sede activa")
	}

	esperar(t, post(t, "/api/v1/sucursales/"+principal+"/baja", `{"motivo":"Intento"}`, tk), http.StatusUnprocessableEntity, "ultimo_administrador")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE codigo='PRINCIPAL' AND es_activo AND motivo_eliminacion IS NULL`) != 1 {
		t.Fatal("el rechazo debe revertir la baja: PRINCIPAL sigue activa")
	}
}
