package integration

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func TestZonasLecturaRespetaLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Salon L1")
	crearZonaSQL(t, "MESA-AJENA", "Salon ajeno")
	tk := lectorMesas(t) // solo PRINCIPAL

	propiaZona := scalarStr(t, `SELECT z.id FROM sch_restaurante.zonas z JOIN sch_restaurante.sucursales s ON s.id=z.sucursal_id WHERE s.codigo='PRINCIPAL' AND z.nombre='Salon L1'`)
	ajenaZona := scalarStr(t, `SELECT z.id FROM sch_restaurante.zonas z JOIN sch_restaurante.sucursales s ON s.id=z.sucursal_id WHERE s.codigo='MESA-AJENA' AND z.nombre='Salon ajeno'`)
	ids := listarZonas(t, "", tk).ids()
	if !slices.Contains(ids, propiaZona) || slices.Contains(ids, ajenaZona) {
		t.Fatalf("zonas visibles = %v: debe incluir %s (PRINCIPAL) y no %s (MESA-AJENA)", ids, propiaZona, ajenaZona)
	}
	ajena := idDe(t, "sucursales", "codigo", "MESA-AJENA")
	esperar(t, call(t, "GET", "/api/v1/zonas?sucursal_id="+ajena, "", tk), http.StatusNotFound, "no_encontrado")

	propia := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	if got := listarZonas(t, "?sucursal_id="+propia, tk).ids(); !slices.Contains(got, propiaZona) || slices.Contains(got, ajenaZona) {
		t.Fatalf("sucursal_id propio: %v", got)
	}

	z := zonaResp{ID: propiaZona}
	r := call(t, "GET", "/api/v1/zonas/"+z.ID, "", tk)
	esperar(t, r, http.StatusOK, "")
	var leida zonaResp
	r.json(t, &leida)
	if leida.ID != z.ID || leida.SucursalID != propia || !leida.EsActivo || leida.CreadoPor == "" {
		t.Fatalf("zona: %+v", leida)
	}
	esperar(t, call(t, "GET", "/api/v1/zonas/"+ajenaZona, "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/zonas/999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/zonas/abc", "", tk), http.StatusBadRequest, "id_invalido")
}

func TestZonasExigeAlgunPermisoDeMesa(t *testing.T) {
	rolConPermisos(t, "T_SIN_MESA", "NEGOCIO_VER")
	asegurarUsuario(t, "mesa.sin", "MesaSin2026Clave", "T_SIN_MESA", "PRINCIPAL")
	tk := token(t, "mesa.sin", "MesaSin2026Clave")
	esperar(t, call(t, "GET", "/api/v1/zonas", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/zonas/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/zonas", "", ""), http.StatusUnauthorized, "no_autenticado")
	// MESERO y CAJERO reciben MESA_VER con la migración 00008.
	asegurarUsuario(t, "mesa.cajero", "MesaCajero2026Clave", "CAJERO", "PRINCIPAL")
	esperar(t, call(t, "GET", "/api/v1/zonas", "", token(t, "mesa.cajero", "MesaCajero2026Clave")), http.StatusOK, "")
}

func TestZonasFiltroYPaginacion(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Pag A")
	crearZonaSQL(t, "PRINCIPAL", "Pag B")
	crearZonaSQL(t, "PRINCIPAL", "Pag Inactiva")
	asSystem(t, `UPDATE sch_restaurante.zonas SET es_activo=false WHERE nombre='Pag Inactiva'`)
	tk := lectorMesas(t)

	idA := scalarStr(t, `SELECT id FROM sch_restaurante.zonas WHERE nombre='Pag A'`)
	idI := scalarStr(t, `SELECT id FROM sch_restaurante.zonas WHERE nombre='Pag Inactiva'`)
	activas := listarZonas(t, "?es_activo=true", tk)
	for _, x := range activas.Items {
		if !x.EsActivo {
			t.Fatalf("es_activo=true devolvió una inactiva: %+v", x)
		}
	}
	if ids := activas.ids(); slices.Contains(ids, idI) || !slices.Contains(ids, idA) {
		t.Fatalf("es_activo=true: %v", ids)
	}
	inactivas := listarZonas(t, "?es_activo=false", tk)
	for _, x := range inactivas.Items {
		if x.EsActivo {
			t.Fatalf("es_activo=false devolvió una activa: %+v", x)
		}
	}
	if ids := inactivas.ids(); !slices.Contains(ids, idI) || slices.Contains(ids, idA) {
		t.Fatalf("es_activo=false: %v", ids)
	}
	esperar(t, call(t, "GET", "/api/v1/zonas?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/zonas?limit=0", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/zonas?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")
	esperar(t, call(t, "GET", "/api/v1/zonas?sucursal_id=x", "", tk), http.StatusBadRequest, "sucursal_id_invalido")

	p1 := listarZonas(t, "?limit=1", tk)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarZonas(t, "?limit=1&cursor="+*p1.NextCursor, tk)
	if len(p2.Items) != 1 || p2.Items[0].ID == p1.Items[0].ID {
		t.Fatalf("segunda página: %+v", p2)
	}
}

func altaZona(sucursalID, nombre string) string {
	return fmt.Sprintf(`{"sucursal_id":"%s","nombre":"%s","orden_visualizacion":3}`, sucursalID, nombre)
}

func TestZonaCrearConIdempotenciaYValidaciones(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	tk := gestorMesas(t)
	principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	h := map[string]string{"Idempotency-Key": nuevaClave()}
	s := sufijo()
	nueva := "Zona nueva " + s

	r1 := callH(t, "POST", "/api/v1/zonas", altaZona(principal, nueva), tk, h)
	esperar(t, r1, http.StatusCreated, "")
	var z zonaResp
	r1.json(t, &z)
	if z.Nombre != nueva || !z.EsActivo || z.SucursalID != principal || z.Orden != 3 ||
		z.CreadoPor != idDe(t, "usuarios", "username", "mesa.gestor") {
		t.Fatalf("zona creada: %+v", z)
	}
	// Repetir la misma intención no duplica; otro contenido con la misma clave es conflicto.
	r2 := callH(t, "POST", "/api/v1/zonas", altaZona(principal, nueva), tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var rep zonaResp
	r2.json(t, &rep)
	if rep.ID != z.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.zonas WHERE sucursal_id=`+principal+` AND nombre=$1`, nueva) != 1 {
		t.Fatalf("la repetición debe devolver la misma zona: %+v vs %+v", rep, z)
	}
	esperar(t, callH(t, "POST", "/api/v1/zonas", altaZona(principal, "Otra "+s), tk, h), http.StatusConflict, "idempotency_key_reutilizada")

	zonasAntes := scalar(t, `SELECT count(*) FROM sch_restaurante.zonas`)
	esperar(t, post(t, "/api/v1/zonas", altaZona(principal, nueva), tk), http.StatusConflict, "nombre_duplicado")
	esperar(t, call(t, "POST", "/api/v1/zonas", altaZona(principal, "Sin clave "+s), tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"`+principal+`","nombre":"x","creado_por":"1"}`, tk), http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"abc","nombre":"x"}`, tk), http.StatusBadRequest, "sucursal_id_invalido")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"`+principal+`","nombre":"  "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"`+principal+`","nombre":"x","orden_visualizacion":-1}`, tk), http.StatusUnprocessableEntity, "orden_invalido")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.zonas`) != zonasAntes {
		t.Fatal("los rechazos no deben dejar zonas")
	}

	// Sede ajena: 404. Solo lectura: 403 sin_permiso.
	ajena := idDe(t, "sucursales", "codigo", "MESA-AJENA")
	esperar(t, post(t, "/api/v1/zonas", altaZona(ajena, "Intrusa "+s), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/zonas", altaZona(principal, "Sin poder "+s), lectorMesas(t)), http.StatusForbidden, "sin_permiso")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.zonas`) != zonasAntes {
		t.Fatal("los rechazos de sede y permiso no deben dejar zonas")
	}
}

func TestZonaEditar(t *testing.T) {
	s := sufijo()
	crearSedeSQL(t, "MESA-AJENA")
	id := crearZonaSQL(t, "PRINCIPAL", "Editable "+s)
	ajenaID := crearZonaSQL(t, "MESA-AJENA", "Ajena editable "+s)
	otra := "Otra existente " + s
	crearZonaSQL(t, "PRINCIPAL", otra)
	tk := gestorMesas(t)
	editada := "Editada " + s

	r := callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"`+editada+`","orden_visualizacion":7}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var z zonaResp
	r.json(t, &z)
	if z.ID != id || z.Nombre != editada || z.Orden != 7 {
		t.Fatalf("edición: %+v", z)
	}
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"sucursal_id":"1"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":" "}`, tk, nil), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/999999", `{"nombre":"x"}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+ajenaID, `{"nombre":"Hack"}`, tk, nil), http.StatusNotFound, "no_encontrado")

	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"`+otra+`"}`, tk, nil), http.StatusConflict, "nombre_duplicado")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"Hack"}`, lectorMesas(t), nil), http.StatusForbidden, "sin_permiso")
	if got := scalarStr(t, `SELECT nombre FROM sch_restaurante.zonas WHERE id=$1`, id); got != editada {
		t.Fatalf("los rechazos no deben cambiar la zona: %q", got)
	}
}

func TestZonaBajaRestauracionYMesasActivas(t *testing.T) {
	s := sufijo()
	id := crearZonaSQL(t, "PRINCIPAL", "Ciclo zona "+s)
	mesa := crearMesaSQL(t, "PRINCIPAL", "Ciclo zona "+s, "CZ-"+s, 4)
	tk := gestorMesas(t)

	// Con una mesa activa la zona no se desactiva.
	esperar(t, post(t, "/api/v1/zonas/"+id+"/baja", "", tk), http.StatusConflict, "zona_con_mesas")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.zonas WHERE id=$1 AND es_activo`, id) != 1 {
		t.Fatal("la zona debe seguir activa")
	}
	asSystem(t, `UPDATE sch_restaurante.mesas SET es_activo=false, motivo_eliminacion='Fuera de servicio' WHERE id=`+mesa)

	h := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/zonas/"+id+"/baja", "", tk, h)
	esperar(t, r, http.StatusOK, "")
	var z zonaResp
	r.json(t, &z)
	if z.EsActivo {
		t.Fatalf("baja: %+v", z)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='zonas' AND accion='UPDATE'
		AND registro_id=`+id+` AND creado_por=`+idDe(t, "usuarios", "username", "mesa.gestor")) == 0 {
		t.Fatal("la baja debe quedar auditada con el actor")
	}
	antes := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='zonas' AND registro_id=`+id)
	esperar(t, callH(t, "POST", "/api/v1/zonas/"+id+"/baja", "", tk, h), http.StatusOK, "")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='zonas' AND registro_id=`+id) != antes {
		t.Fatal("repetir la misma clave no debe volver a escribir")
	}
	esperar(t, post(t, "/api/v1/zonas/"+id+"/baja", "", tk), http.StatusConflict, "ya_dada_de_baja")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"x"}`, tk, nil), http.StatusConflict, "zona_inactiva")

	r = post(t, "/api/v1/zonas/"+id+"/restauracion", `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rest zonaResp
	r.json(t, &rest)
	if rest.ID != z.ID || !rest.EsActivo {
		t.Fatalf("restauración debe conservar el ID: %+v", rest)
	}
	esperar(t, post(t, "/api/v1/zonas/"+id+"/restauracion", "", tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/zonas/999999/baja", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/zonas/"+id+"/baja", "", lectorMesas(t)), http.StatusForbidden, "sin_permiso")
}
