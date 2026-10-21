package integration

import (
	"net/http"
	"slices"
	"sort"
	"strconv"
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
