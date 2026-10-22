package integration

import (
	"net/http"
	"slices"
	"strconv"
	"testing"
)

func TestMesasLecturaRespetaLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Zona mesas L")
	crearZonaSQL(t, "MESA-AJENA", "Zona mesas ajena")
	crearMesaSQL(t, "PRINCIPAL", "Zona mesas L", "ML-1", 4)
	crearMesaSQL(t, "MESA-AJENA", "Zona mesas ajena", "MAJ-1", 2)
	tk := lectorMesas(t)

	numeros := listarMesas(t, "", tk).numeros()
	if !slices.Contains(numeros, "ML-1") || slices.Contains(numeros, "MAJ-1") {
		t.Fatalf("mesas visibles = %v: debe incluir ML-1 y no MAJ-1", numeros)
	}
	esperar(t, call(t, "GET", "/api/v1/mesas?sucursal_id="+idDe(t, "sucursales", "codigo", "MESA-AJENA"), "", tk),
		http.StatusNotFound, "no_encontrado")

	id := idDe(t, "mesas", "numero_mesa", "ML-1")
	r := call(t, "GET", "/api/v1/mesas/"+id, "", tk)
	esperar(t, r, http.StatusOK, "")
	var m mesaResp
	r.json(t, &m)
	if m.NumeroMesa != "ML-1" || m.Capacidad != 4 || !m.Habilitada || !m.EsActivo || m.PosicionX != nil || m.Motivo != nil ||
		m.SucursalID != idDe(t, "sucursales", "codigo", "PRINCIPAL") || m.ZonaID != idDe(t, "zonas", "nombre", "Zona mesas L") {
		t.Fatalf("mesa: %+v", m)
	}
	esperar(t, call(t, "GET", "/api/v1/mesas/"+idDe(t, "mesas", "numero_mesa", "MAJ-1"), "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/mesas/999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/mesas/abc", "", tk), http.StatusBadRequest, "id_invalido")
}

func TestMesasExigeAlgunPermisoDeMesa(t *testing.T) {
	rolConPermisos(t, "T_SIN_MESA", "NEGOCIO_VER")
	asegurarUsuario(t, "mesa.sin", "MesaSin2026Clave", "T_SIN_MESA", "PRINCIPAL")
	tk := token(t, "mesa.sin", "MesaSin2026Clave")
	esperar(t, call(t, "GET", "/api/v1/mesas", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/mesas/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/mesas", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func TestMesasFiltroYPaginacion(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Zona pag mesas")
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-1", 2)
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-2", 2)
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-3", 2)
	asSystem(t, `UPDATE sch_restaurante.mesas SET es_activo=false, motivo_eliminacion='Retirada' WHERE numero_mesa='PM-3'`)
	tk := lectorMesas(t)

	if got := listarMesas(t, "?es_activo=true", tk).numeros(); slices.Contains(got, "PM-3") || !slices.Contains(got, "PM-1") {
		t.Fatalf("es_activo=true: %v", got)
	}
	inactivas := listarMesas(t, "?es_activo=false", tk)
	if got := inactivas.numeros(); !slices.Contains(got, "PM-3") || slices.Contains(got, "PM-1") {
		t.Fatalf("es_activo=false: %v", got)
	}
	for _, m := range inactivas.Items {
		if m.NumeroMesa == "PM-3" && (m.Motivo == nil || *m.Motivo != "Retirada" || m.EliminadoPor == nil) {
			t.Fatalf("una mesa inactiva debe exponer motivo y actor: %+v", m)
		}
	}
	esperar(t, call(t, "GET", "/api/v1/mesas?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/mesas?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/mesas?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")

	p1 := listarMesas(t, "?limit=1", tk)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarMesas(t, "?limit=1&cursor="+*p1.NextCursor, tk)
	id1, _ := strconv.Atoi(p1.Items[0].ID)
	id2, _ := strconv.Atoi(p2.Items[0].ID)
	if len(p2.Items) != 1 || id1 >= id2 {
		t.Fatalf("la segunda página debe continuar en orden ascendente: %+v / %+v", p1, p2)
	}
}
