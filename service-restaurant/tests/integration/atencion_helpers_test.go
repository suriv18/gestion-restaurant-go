package integration

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Los datos de prueba se comparten durante todo el proceso y las pruebas que modifican filas
// deben poder repetirse (-count=N) sobre la misma base: cada ejecución usa nombres propios.
var (
	sufijoBase = strconv.FormatInt(time.Now().UnixNano()%1679616, 36) // hasta 4 caracteres base 36
	sufijoSeq  atomic.Int64
)

// sufijo devuelve un texto corto (hasta 8 caracteres) distinto en cada llamada del proceso.
func sufijo() string { return sufijoBase + strconv.FormatInt(sufijoSeq.Add(1), 36) }

type zonaResp struct {
	ID         string `json:"id"`
	SucursalID string `json:"sucursal_id"`
	Nombre     string `json:"nombre"`
	Orden      int    `json:"orden_visualizacion"`
	EsActivo   bool   `json:"es_activo"`
	CreadoPor  string `json:"creado_por"`
}

type zonasPagina struct {
	Items      []zonaResp `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

func (p zonasPagina) nombres() []string {
	out := make([]string, 0, len(p.Items))
	for _, z := range p.Items {
		out = append(out, z.Nombre)
	}
	sort.Strings(out)
	return out
}

type mesaResp struct {
	ID           string  `json:"id"`
	SucursalID   string  `json:"sucursal_id"`
	ZonaID       string  `json:"zona_id"`
	NumeroMesa   string  `json:"numero_mesa"`
	Capacidad    int     `json:"capacidad"`
	Habilitada   bool    `json:"habilitada"`
	PosicionX    *string `json:"posicion_x"`
	PosicionY    *string `json:"posicion_y"`
	EsActivo     bool    `json:"es_activo"`
	Motivo       *string `json:"motivo_eliminacion"`
	EliminadoPor *string `json:"eliminado_por"`
	CreadoPor    string  `json:"creado_por"`
}

type mesasPagina struct {
	Items      []mesaResp `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

func (p mesasPagina) numeros() []string {
	out := make([]string, 0, len(p.Items))
	for _, m := range p.Items {
		out = append(out, m.NumeroMesa)
	}
	sort.Strings(out)
	return out
}

// crearZonaSQL inserta una zona activa en la sede (por código) si no existe y devuelve su id.
func crearZonaSQL(t *testing.T, sede, nombre string) string {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.zonas(sucursal_id, nombre)
		SELECT s.id, '`+nombre+`' FROM sch_restaurante.sucursales s WHERE s.codigo='`+sede+`'
		AND NOT EXISTS (SELECT 1 FROM sch_restaurante.zonas z WHERE z.sucursal_id=s.id AND z.nombre='`+nombre+`')`)
	return scalarStr(t, `SELECT z.id FROM sch_restaurante.zonas z JOIN sch_restaurante.sucursales s ON s.id=z.sucursal_id
		WHERE s.codigo=$1 AND z.nombre=$2`, sede, nombre)
}

// crearMesaSQL inserta una mesa activa en la zona (por nombre y sede) si no existe y devuelve su id.
func crearMesaSQL(t *testing.T, sede, zona, numero string, capacidad int) string {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.mesas(sucursal_id, zona_id, numero_mesa, capacidad)
		SELECT z.sucursal_id, z.id, '`+numero+`', `+strconv.Itoa(capacidad)+` FROM sch_restaurante.zonas z
		JOIN sch_restaurante.sucursales s ON s.id=z.sucursal_id
		WHERE s.codigo='`+sede+`' AND z.nombre='`+zona+`'
		AND NOT EXISTS (SELECT 1 FROM sch_restaurante.mesas m WHERE m.sucursal_id=z.sucursal_id AND m.numero_mesa='`+numero+`')`)
	return scalarStr(t, `SELECT m.id FROM sch_restaurante.mesas m JOIN sch_restaurante.sucursales s ON s.id=m.sucursal_id
		WHERE s.codigo=$1 AND m.numero_mesa=$2`, sede, numero)
}

// gestorMesas devuelve el token de un usuario con MESA_VER y MESA_ADMINISTRAR en las sedes dadas
// (por código). Es idempotente: varias pruebas lo reutilizan.
func gestorMesas(t *testing.T, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, "T_MESA_GESTOR", "MESA_VER", "MESA_ADMINISTRAR")
	asegurarUsuario(t, "mesa.gestor", "MesaGestor2026Clave", "T_MESA_GESTOR", append([]string{"PRINCIPAL"}, sedes...)...)
	return token(t, "mesa.gestor", "MesaGestor2026Clave")
}

// lectorMesas devuelve el token de un usuario que solo tiene MESA_VER en las sedes dadas.
func lectorMesas(t *testing.T, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, "T_MESA_LECTOR", "MESA_VER")
	asegurarUsuario(t, "mesa.lector", "MesaLector2026Clave", "T_MESA_LECTOR", append([]string{"PRINCIPAL"}, sedes...)...)
	return token(t, "mesa.lector", "MesaLector2026Clave")
}

// conLimite añade limit=100 si la consulta no trae uno: la base se comparte entre pruebas y el
// tamaño de página por defecto (25) dejaría fuera los datos de prueba según el orden de ejecución.
func conLimite(query string) string {
	if strings.Contains(query, "limit=") {
		return query
	}
	switch {
	case query == "":
		return "?limit=100"
	case strings.HasPrefix(query, "?"):
		return query + "&limit=100"
	default:
		return "?" + query + "&limit=100"
	}
}

func (p zonasPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, z := range p.Items {
		out = append(out, z.ID)
	}
	return out
}

func listarZonas(t *testing.T, query, tk string) zonasPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/zonas"+conLimite(query), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p zonasPagina
	r.json(t, &p)
	return p
}

func listarMesas(t *testing.T, query, tk string) mesasPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/mesas"+conLimite(query), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p mesasPagina
	r.json(t, &p)
	return p
}

// scalarStr devuelve el primer valor de una consulta como texto (para ids).
func scalarStr(t *testing.T, query string, args ...any) string {
	t.Helper()
	var v any
	if err := testDB.QueryRowContext(context.Background(), query, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return fmt.Sprint(v)
}
