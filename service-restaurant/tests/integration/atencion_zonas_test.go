package integration

import (
	"net/http"
	"slices"
	"testing"
)

func TestZonasLecturaRespetaLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Salon L1")
	crearZonaSQL(t, "MESA-AJENA", "Salon ajeno")
	tk := lectorMesas(t) // solo PRINCIPAL

	nombres := listarZonas(t, "", tk).nombres()
	if !slices.Contains(nombres, "Salon L1") || slices.Contains(nombres, "Salon ajeno") {
		t.Fatalf("zonas visibles = %v: debe incluir la de PRINCIPAL y no la de MESA-AJENA", nombres)
	}
	ajena := idDe(t, "sucursales", "codigo", "MESA-AJENA")
	esperar(t, call(t, "GET", "/api/v1/zonas?sucursal_id="+ajena, "", tk), http.StatusNotFound, "no_encontrado")

	propia := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	if got := listarZonas(t, "?sucursal_id="+propia, tk).nombres(); !slices.Contains(got, "Salon L1") {
		t.Fatalf("sucursal_id propio: %v", got)
	}

	r := call(t, "GET", "/api/v1/zonas", "", tk)
	var p zonasPagina
	r.json(t, &p)
	var z zonaResp
	for _, x := range p.Items {
		if x.Nombre == "Salon L1" {
			z = x
		}
	}
	r = call(t, "GET", "/api/v1/zonas/"+z.ID, "", tk)
	esperar(t, r, http.StatusOK, "")
	var leida zonaResp
	r.json(t, &leida)
	if leida.ID != z.ID || leida.SucursalID != propia || !leida.EsActivo || leida.CreadoPor == "" {
		t.Fatalf("zona: %+v", leida)
	}
	ajenaZona := scalarStr(t, `SELECT z.id FROM sch_restaurante.zonas z WHERE z.nombre='Salon ajeno'`)
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

	activas := listarZonas(t, "?es_activo=true", tk).nombres()
	if slices.Contains(activas, "Pag Inactiva") || !slices.Contains(activas, "Pag A") {
		t.Fatalf("es_activo=true: %v", activas)
	}
	inactivas := listarZonas(t, "?es_activo=false", tk).nombres()
	if !slices.Contains(inactivas, "Pag Inactiva") || slices.Contains(inactivas, "Pag A") {
		t.Fatalf("es_activo=false: %v", inactivas)
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
