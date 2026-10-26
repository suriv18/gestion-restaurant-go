package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type estacionResp struct {
	ID             string `json:"id"`
	SucursalID     string `json:"sucursal_id"`
	Codigo         string `json:"codigo"`
	Nombre         string `json:"nombre"`
	EsActivo       bool   `json:"es_activo"`
	CreadoEn       string `json:"creado_en"`
	CreadoPor      string `json:"creado_por"`
	ActualizadoEn  string `json:"actualizado_en"`
	ActualizadoPor string `json:"actualizado_por"`
}

type estacionesPagina struct {
	Items      []estacionResp `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

func (p estacionesPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, e := range p.Items {
		out = append(out, e.ID)
	}
	return out
}

// sedeEstaciones crea una sede propia de la prueba (código único) y devuelve su código y su id.
func sedeEstaciones(t *testing.T) (codigo, id string) {
	t.Helper()
	codigo = "E" + strings.ToUpper(sufijo())
	crearSedeSQL(t, codigo)
	return codigo, idDe(t, "sucursales", "codigo", codigo)
}

// gestorEstaciones devuelve el token de un usuario propio de la prueba con ESTACION_VER y
// ESTACION_ADMINISTRAR en las sedes dadas (por código).
func gestorEstaciones(t *testing.T, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, "T_EST_GESTOR", "ESTACION_VER", "ESTACION_ADMINISTRAR")
	user := "est.gestor." + strings.ToLower(sufijo())
	asegurarUsuario(t, user, "EstGestor2026Clave", "T_EST_GESTOR", sedes...)
	return token(t, user, "EstGestor2026Clave")
}

// lectorEstaciones es como gestorEstaciones pero solo con ESTACION_VER.
func lectorEstaciones(t *testing.T, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, "T_EST_LECTOR", "ESTACION_VER")
	user := "est.lector." + strings.ToLower(sufijo())
	asegurarUsuario(t, user, "EstLector2026Clave", "T_EST_LECTOR", sedes...)
	return token(t, user, "EstLector2026Clave")
}

func cuerpoEstacion(sede, codigo, nombre string) string {
	return fmt.Sprintf(`{"sucursal_id":"%s","codigo":"%s","nombre":"%s"}`, sede, codigo, nombre)
}

func crearEstacion(t *testing.T, tk, sede, codigo, nombre string) estacionResp {
	t.Helper()
	r := post(t, "/api/v1/estaciones", cuerpoEstacion(sede, codigo, nombre), tk)
	esperar(t, r, http.StatusCreated, "")
	var e estacionResp
	r.json(t, &e)
	return e
}

func listarEstaciones(t *testing.T, query, tk string) estacionesPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/estaciones"+conLimite(query), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p estacionesPagina
	r.json(t, &p)
	return p
}

func TestEstacionesPermisos(t *testing.T) {
	esperar(t, call(t, "GET", "/api/v1/estaciones", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/estaciones/1", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/estaciones", `{}`, ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, patch(t, "/api/v1/estaciones/1", `{"nombre":"X"}`, ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/estaciones/1/baja", ``, ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, post(t, "/api/v1/estaciones/1/restauracion", ``, ""), http.StatusUnauthorized, "no_autenticado")

	sede, sedeID := sedeEstaciones(t)
	gestor := gestorEstaciones(t, sede)
	e := crearEstacion(t, gestor, sedeID, "perm", "Permisos")

	// Un rol sin permisos de estación no lee ni escribe.
	rolConPermisos(t, "T_EST_NADA", "CLIENTE_VER")
	asegurarUsuario(t, "est.nada", "EstNada2026Clave", "T_EST_NADA", sede)
	nada := token(t, "est.nada", "EstNada2026Clave")
	esperar(t, call(t, "GET", "/api/v1/estaciones", "", nada), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/estaciones/"+e.ID, "", nada), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "no", "No"), nada), http.StatusForbidden, "sin_permiso")

	// Con solo ESTACION_VER se lee, pero cualquier escritura es 403 sin_permiso (ningún permiso de
	// administración en ninguna sede).
	lector := lectorEstaciones(t, sede)
	esperar(t, call(t, "GET", "/api/v1/estaciones"+conLimite(""), "", lector), http.StatusOK, "")
	esperar(t, call(t, "GET", "/api/v1/estaciones/"+e.ID, "", lector), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "no", "No"), lector), http.StatusForbidden, "sin_permiso")
	esperar(t, patch(t, "/api/v1/estaciones/"+e.ID, `{"nombre":"No"}`, lector), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/estaciones/"+e.ID+"/baja", ``, lector), http.StatusForbidden, "sin_permiso")
	esperar(t, post(t, "/api/v1/estaciones/"+e.ID+"/restauracion", ``, lector), http.StatusForbidden, "sin_permiso")

	// MESERO trae ESTACION_VER de la migración 00010: lee en su sede.
	asegurarUsuario(t, "est.mesero", "EstMesero2026Clave", "MESERO", sede)
	mesero := token(t, "est.mesero", "EstMesero2026Clave")
	esperar(t, call(t, "GET", "/api/v1/estaciones/"+e.ID, "", mesero), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "no", "No"), mesero), http.StatusForbidden, "sin_permiso")
}

func TestEstacionAltaIdempotenciaYValidaciones(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	otraSede, otraID := sedeEstaciones(t)
	tk := gestorEstaciones(t, sede)

	cuerpo := cuerpoEstacion(sedeID, "  coc-1  ", "  Cocina fría  ")
	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/estaciones", cuerpo, tk, clave)
	esperar(t, r, http.StatusCreated, "")
	var e estacionResp
	r.json(t, &e)
	if e.Codigo != "COC-1" || e.Nombre != "Cocina fría" || e.SucursalID != sedeID || !e.EsActivo || e.ID == "" ||
		e.CreadoPor == "" || e.ActualizadoPor != e.CreadoPor || e.CreadoEn == "" {
		t.Fatalf("estación creada: %+v", e)
	}
	// Misma clave y mismo cuerpo: misma respuesta, sin duplicar.
	rep := callH(t, "POST", "/api/v1/estaciones", cuerpo, tk, clave)
	esperar(t, rep, http.StatusCreated, "")
	var e2 estacionResp
	rep.json(t, &e2)
	if e2.ID != e.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.estaciones_preparacion WHERE sucursal_id=`+sedeID) != 1 {
		t.Fatalf("la repetición creó otra estación: %s vs %s", e2.ID, e.ID)
	}
	esperar(t, callH(t, "POST", "/api/v1/estaciones", cuerpoEstacion(sedeID, "OTRA", "Otra"), tk, clave), http.StatusConflict, "idempotency_key_reutilizada")
	esperar(t, call(t, "POST", "/api/v1/estaciones", cuerpoEstacion(sedeID, "SIN", "Sin clave"), tk), http.StatusBadRequest, "idempotency_key_requerida")

	// Validaciones de código y nombre.
	for _, c := range []string{"", "   ", "-A", "A B", "A.B", "ñ", strings.Repeat("A", 21)} {
		esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, c, "Nombre"), tk), http.StatusUnprocessableEntity, "codigo_invalido")
	}
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, strings.Repeat("a", 20), "Veinte"), tk), http.StatusCreated, "")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "N1", ""), tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "N2", "dos\\nlíneas"), tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "N3", strings.Repeat("a", 81)), tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "N4", strings.Repeat("ñ", 80)), tk), http.StatusCreated, "")

	// sucursal_id ausente o mal formado, campos desconocidos.
	esperar(t, post(t, "/api/v1/estaciones", `{"codigo":"X","nombre":"X"}`, tk), http.StatusBadRequest, "sucursal_id_invalido")
	esperar(t, post(t, "/api/v1/estaciones", `{"sucursal_id":"abc","codigo":"X","nombre":"X"}`, tk), http.StatusBadRequest, "sucursal_id_invalido")
	esperar(t, post(t, "/api/v1/estaciones", `{"sucursal_id":"`+sedeID+`","codigo":"X","nombre":"X","extra":1}`, tk), http.StatusBadRequest, "")

	// (sede, código) es único, también tras la baja y sin distinguir mayúsculas; en otra sede se puede repetir.
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "coc-1", "Repetida"), tk), http.StatusConflict, "estacion_duplicada")
	esperar(t, post(t, "/api/v1/estaciones/"+e.ID+"/baja", ``, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "COC-1", "Repetida"), tk), http.StatusConflict, "estacion_duplicada")
	gestor2 := gestorEstaciones(t, otraSede)
	crearEstacion(t, gestor2, otraID, "COC-1", "Misma clave, otra sede")

	// Sede: el actor no ve la otra sede (404), una sede inexistente tampoco (404).
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(otraID, "ZZ", "No"), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion("999999999", "ZZ", "No"), tk), http.StatusNotFound, "no_encontrado")
}

// Una sede dada de baja deja de figurar en los permisos del actor (v_permisos_efectivos solo
// cuenta sedes activas): sus estaciones "desaparecen" (404) y no se puede crear ni restaurar en
// ella. El rechazo del trigger de sede inactiva (422 referencia_invalida) queda cubierto en las
// pruebas unitarias del caso de uso, porque por HTTP la autorización se adelanta.
func TestEstacionSedeInactiva(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	viva, _ := sedeEstaciones(t)
	tk := gestorEstaciones(t, sede, viva)
	e := crearEstacion(t, tk, sedeID, "SI", "Sede inactiva")
	esperar(t, post(t, "/api/v1/estaciones/"+e.ID+"/baja", ``, tk), http.StatusOK, "")

	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.sucursales SET es_activo=false, motivo_eliminacion='Cierre de prueba' WHERE id=%s`, sedeID))

	esperar(t, post(t, "/api/v1/estaciones/"+e.ID+"/restauracion", ``, tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/estaciones", cuerpoEstacion(sedeID, "NUEVA", "Nueva"), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/estaciones/"+e.ID, "", tk), http.StatusNotFound, "no_encontrado")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.estaciones_preparacion WHERE id=`+e.ID+` AND NOT es_activo`) != 1 {
		t.Fatal("la estación debe seguir dada de baja")
	}
}

func TestEstacionLecturaYEdicion(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	tk := gestorEstaciones(t, sede)
	e := crearEstacion(t, tk, sedeID, "ed1", "Original")
	otra := crearEstacion(t, tk, sedeID, "ED2", "Otra")

	var leida estacionResp
	g := call(t, "GET", "/api/v1/estaciones/"+e.ID, "", tk)
	esperar(t, g, http.StatusOK, "")
	g.json(t, &leida)
	if leida.ID != e.ID || leida.Codigo != "ED1" || leida.Nombre != "Original" || leida.SucursalID != sedeID {
		t.Fatalf("lectura: %+v", leida)
	}
	esperar(t, call(t, "GET", "/api/v1/estaciones/999999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/estaciones/abc", "", tk), http.StatusBadRequest, "id_invalido")

	// PATCH parcial y normalización del código.
	var ed estacionResp
	r := patch(t, "/api/v1/estaciones/"+e.ID, `{"nombre":"  Renombrada  "}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.Nombre != "Renombrada" || ed.Codigo != "ED1" {
		t.Fatalf("PATCH nombre: %+v", ed)
	}
	r = patch(t, "/api/v1/estaciones/"+e.ID, `{"codigo":"barra_2"}`, tk)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &ed)
	if ed.Codigo != "BARRA_2" || ed.Nombre != "Renombrada" || ed.SucursalID != sedeID {
		t.Fatalf("PATCH código: %+v", ed)
	}

	esperar(t, patch(t, "/api/v1/estaciones/"+e.ID, `{}`, tk), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, patch(t, "/api/v1/estaciones/"+e.ID, `{"codigo":"a b"}`, tk), http.StatusUnprocessableEntity, "codigo_invalido")
	esperar(t, patch(t, "/api/v1/estaciones/"+e.ID, `{"nombre":" "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, patch(t, "/api/v1/estaciones/"+e.ID, `{"codigo":"ed2"}`, tk), http.StatusConflict, "estacion_duplicada")
	// La sede no es editable: campo desconocido → 400 (desvío del schema de contrato).
	esperar(t, patch(t, "/api/v1/estaciones/"+e.ID, `{"sucursal_id":"`+sedeID+`"}`, tk), http.StatusBadRequest, "")
	esperar(t, patch(t, "/api/v1/estaciones/"+e.ID, `{"campo_raro":1}`, tk), http.StatusBadRequest, "")
	esperar(t, patch(t, "/api/v1/estaciones/999999999", `{"nombre":"Z"}`, tk), http.StatusNotFound, "no_encontrado")

	// Una estación dada de baja no se edita.
	esperar(t, post(t, "/api/v1/estaciones/"+otra.ID+"/baja", ``, tk), http.StatusOK, "")
	esperar(t, patch(t, "/api/v1/estaciones/"+otra.ID, `{"nombre":"No"}`, tk), http.StatusConflict, "estacion_inactiva")
}

func TestEstacionesListadoPorSedesVisibles(t *testing.T) {
	sedeA, idA := sedeEstaciones(t)
	sedeB, idB := sedeEstaciones(t)
	tkA := gestorEstaciones(t, sedeA)
	tkB := gestorEstaciones(t, sedeB)
	a1 := crearEstacion(t, tkA, idA, "COCINA", "Cocina caliente")
	a2 := crearEstacion(t, tkA, idA, "BARRA", "Barra principal")
	b1 := crearEstacion(t, tkB, idB, "COCINA", "Cocina de B")
	esperar(t, post(t, "/api/v1/estaciones/"+a2.ID+"/baja", ``, tkA), http.StatusOK, "")

	// Quien solo ve la sede A no ve nada de B.
	ver := lectorEstaciones(t, sedeA)
	p := listarEstaciones(t, "", ver)
	if !slices.Contains(p.ids(), a1.ID) || !slices.Contains(p.ids(), a2.ID) || slices.Contains(p.ids(), b1.ID) {
		t.Fatalf("listado de A: %v", p.ids())
	}
	for _, it := range p.Items {
		if it.SucursalID == idB {
			t.Fatalf("apareció una estación de B: %+v", it)
		}
	}
	esperar(t, call(t, "GET", "/api/v1/estaciones"+conLimite("sucursal_id="+idB), "", ver), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/estaciones"+conLimite("sucursal_id=999999999"), "", ver), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/estaciones/"+b1.ID, "", ver), http.StatusNotFound, "no_encontrado")
	esperar(t, patch(t, "/api/v1/estaciones/"+b1.ID, `{"nombre":"X"}`, tkA), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/estaciones/"+b1.ID+"/baja", ``, tkA), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/estaciones/"+b1.ID+"/restauracion", ``, tkA), http.StatusNotFound, "no_encontrado")

	// Filtros: sucursal_id propia, es_activo, q (código o nombre).
	if got := listarEstaciones(t, "sucursal_id="+idA, ver).ids(); len(got) != 2 {
		t.Fatalf("sucursal_id=A: %v", got)
	}
	if got := listarEstaciones(t, "sucursal_id="+idA+"&es_activo=true", ver).ids(); !slices.Equal(got, []string{a1.ID}) {
		t.Fatalf("es_activo=true: %v", got)
	}
	if got := listarEstaciones(t, "sucursal_id="+idA+"&es_activo=false", ver).ids(); !slices.Equal(got, []string{a2.ID}) {
		t.Fatalf("es_activo=false: %v", got)
	}
	if got := listarEstaciones(t, "sucursal_id="+idA+"&q=barr", ver).ids(); !slices.Equal(got, []string{a2.ID}) {
		t.Fatalf("q por nombre: %v", got)
	}
	if got := listarEstaciones(t, "sucursal_id="+idA+"&q=cocin", ver).ids(); !slices.Equal(got, []string{a1.ID}) {
		t.Fatalf("q por código: %v", got)
	}
	if got := listarEstaciones(t, "sucursal_id="+idA+"&q=%25", ver).ids(); len(got) != 0 {
		t.Fatalf("el comodín %% debe tratarse como texto: %v", got)
	}

	// Quien ve ambas sedes ve las tres.
	ambas := gestorEstaciones(t, sedeA, sedeB)
	todas := listarEstaciones(t, "", ambas).ids()
	for _, id := range []string{a1.ID, a2.ID, b1.ID} {
		if !slices.Contains(todas, id) {
			t.Fatalf("falta %s en %v", id, todas)
		}
	}

	// Paginación por cursor dentro de una sede.
	r := call(t, "GET", "/api/v1/estaciones?sucursal_id="+idA+"&limit=1", "", ver)
	esperar(t, r, http.StatusOK, "")
	var pag estacionesPagina
	r.json(t, &pag)
	if len(pag.Items) != 1 || pag.Items[0].ID != a1.ID || pag.NextCursor == nil {
		t.Fatalf("primera página: %+v", pag)
	}
	r = call(t, "GET", "/api/v1/estaciones?sucursal_id="+idA+"&limit=1&cursor="+*pag.NextCursor, "", ver)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &pag)
	if len(pag.Items) != 1 || pag.Items[0].ID != a2.ID || pag.NextCursor != nil {
		t.Fatalf("segunda página: %+v", pag)
	}
	esperar(t, call(t, "GET", "/api/v1/estaciones?limit=0", "", ver), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/estaciones?es_activo=x", "", ver), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/estaciones?sucursal_id=x", "", ver), http.StatusBadRequest, "sucursal_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/estaciones?cursor=%%%", "", ver), http.StatusBadRequest, "")
}

func TestEstacionBajaYRestauracion(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	tk := gestorEstaciones(t, sede)
	e := crearEstacion(t, tk, sedeID, "BAJA", "Para baja")
	baja := "/api/v1/estaciones/" + e.ID + "/baja"
	rest := "/api/v1/estaciones/" + e.ID + "/restauracion"

	esperar(t, post(t, rest, ``, tk), http.StatusConflict, "ya_activa")

	clave := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", baja, "", tk, clave)
	esperar(t, r, http.StatusOK, "")
	var b estacionResp
	r.json(t, &b)
	if b.ID != e.ID || b.EsActivo {
		t.Fatalf("baja: %+v", b)
	}
	esperar(t, callH(t, "POST", baja, "", tk, clave), http.StatusOK, "")
	esperar(t, call(t, "POST", baja, "", tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, baja, ``, tk), http.StatusConflict, "ya_dada_de_baja")
	esperar(t, post(t, baja, `{"motivo":"x"}`, tk), http.StatusBadRequest, "")
	esperar(t, post(t, "/api/v1/estaciones/999999999/baja", ``, tk), http.StatusNotFound, "no_encontrado")

	r = post(t, rest, `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rs estacionResp
	r.json(t, &rs)
	if rs.ID != e.ID || !rs.EsActivo || rs.Codigo != "BAJA" {
		t.Fatalf("restauración: %+v", rs)
	}
	esperar(t, post(t, rest, ``, tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/estaciones/999999999/restauracion", ``, tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, baja, ``, tk), http.StatusOK, "")
}

func TestEstacionBajaBloqueadaPorProductosSucursal(t *testing.T) {
	sede, sedeID := sedeEstaciones(t)
	tk := gestorEstaciones(t, sede)
	e := crearEstacion(t, tk, sedeID, "USO", "En uso")

	cat := "CatEst " + sufijo()
	codigo := "PE" + numeroUnico(12)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.categorias(nombre) VALUES ('%s')`, cat))
	catID := idDe(t, "categorias", "nombre", cat)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.productos(categoria_id, codigo, nombre, tipo, precio_venta, afectacion_igv)
		VALUES (%s, '%s', 'Plato de estación', 'PREPARADO', 10, '20')`, catID, codigo))
	prodID := idDe(t, "productos", "codigo", codigo)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.productos_sucursal(producto_id, sucursal_id, estacion_id)
		VALUES (%s, %s, %s)`, prodID, sedeID, e.ID))

	baja := "/api/v1/estaciones/" + e.ID + "/baja"
	esperar(t, post(t, baja, ``, tk), http.StatusConflict, "estacion_en_uso")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.estaciones_preparacion WHERE id=`+e.ID+` AND es_activo`) != 1 {
		t.Fatal("la estación debe seguir activa")
	}

	// Con la fila de producto-sede dada de baja (o sin estación), ya se puede.
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos_sucursal SET es_activo=false WHERE producto_id=%s AND sucursal_id=%s`, prodID, sedeID))
	esperar(t, post(t, baja, ``, tk), http.StatusOK, "")
}
