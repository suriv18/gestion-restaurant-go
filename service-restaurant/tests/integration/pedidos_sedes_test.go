package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// RES-CP-002: un usuario de la sede A no puede leer ni modificar un pedido de la sede B,
// aunque conozca su id.
func TestPedidosAislamientoPorSede(t *testing.T) {
	crearSedeSQL(t, "PED-AJENA")
	ajenoID, ajenaCuenta := pedidoSQL(t, "PED-AJENA")
	propio := abrirLlevar(t, cajeroPedidos(t), "PRINCIPAL")
	prod := productoVentaSQL(t, "PED-AJENA", "PREPARADO", "10.00", true)
	tk := cajeroPedidos(t) // solo PRINCIPAL

	// Línea real en el pedido ajeno: el PATCH debe fallar por la sede, no por un id inexistente.
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.detalles_pedido(pedido_id,cuenta_id,producto_id,descripcion,cantidad,precio_unitario,afectacion_igv,tasa_impuesto)
		VALUES (%s,%s,%s,'Línea ajena',1,10,'10',18)`, ajenoID, ajenaCuenta, prod))
	lineaAjena := scalarStr(t, `SELECT id FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, ajenoID)

	esperar(t, call(t, "GET", "/api/v1/pedidos/"+ajenoID, "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, agregarConsumo(t, tk, ajenoID, ajenaCuenta, prod, "1", ""), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+ajenoID+"/detalles/"+lineaAjena, `{"cantidad":"5"}`, tk), http.StatusNotFound, "no_encontrado")
	if c := scalarStr(t, `SELECT cantidad::numeric(12,3) FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, lineaAjena); c != "1.000" {
		t.Fatalf("la línea ajena no debe cambiar, cantidad = %s", c)
	}
	esperar(t, post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, "PED-AJENA")), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/pedidos?sucursal_id="+sedeID(t, "PED-AJENA"), "", tk), http.StatusNotFound, "no_encontrado")

	ids := listarTodosPedidos(t, "", tk)
	if !slices.Contains(ids, propio.ID) || slices.Contains(ids, ajenoID) {
		t.Fatalf("el listado debe incluir el propio y no el ajeno: %v", ids)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, ajenoID); n != 1 {
		t.Fatalf("el pedido ajeno debe conservar solo su línea sembrada (%d)", n)
	}

	// Con la sede asignada el mismo usuario sí lo ve.
	tkAmbas := tokenPedidos(t, "ped.multi", "T_PED_CAJERO",
		[]string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR", "DESCUENTO_AUTORIZAR"}, "PED-AJENA")
	if g := obtenerPedido(t, tkAmbas, ajenoID); g.ID != ajenoID {
		t.Fatalf("con la sede asignada debe verlo: %+v", g)
	}
	// Control positivo del PATCH: con la sede asignada la misma línea sí se edita.
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+ajenoID+"/detalles/"+lineaAjena, `{"cantidad":"2"}`, tkAmbas), http.StatusOK, "")
}

func TestPedidosVerSinPermisoDeEscritura(t *testing.T) {
	lector := lectorPedidos(t)
	p := abrirLlevar(t, cajeroPedidos(t), "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	// El lector ve el pedido, pero no puede abrir ni modificar: tiene PEDIDO_VER sin PEDIDO_CREAR ni PEDIDO_EDITAR.
	if g := obtenerPedido(t, lector, p.ID); g.ID != p.ID {
		t.Fatalf("el lector debe ver el pedido: %+v", g)
	}
	esperar(t, post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, "PRINCIPAL")), lector), http.StatusForbidden, "sin_permiso")
	esperar(t, agregarConsumo(t, lector, p.ID, p.Cuentas[0].ID, prod, "1", ""), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+p.ID+"/detalles/1", `{"cantidad":"1"}`, lector), http.StatusForbidden, "sin_permiso")
}

func TestPedidosExigeAlgunPermisoDePedidos(t *testing.T) {
	rolConPermisos(t, "T_SIN_PEDIDOS", "NEGOCIO_VER")
	asegurarUsuario(t, "ped.sin", "Pedidos2026Clave", "T_SIN_PEDIDOS", "PRINCIPAL")
	tk := token(t, "ped.sin", "Pedidos2026Clave")
	esperar(t, call(t, "GET", "/api/v1/pedidos", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/pedidos/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/pedidos", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func listarPedidos(t *testing.T, query, tk string) pedidosPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/pedidos"+conLimite(query), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p pedidosPagina
	r.json(t, &p)
	return p
}

// listarTodosPedidos sigue los cursores hasta agotar el listado: la base se comparte entre pruebas
// y una sola página de 100 podría dejar fuera los pedidos de esta.
func listarTodosPedidos(t *testing.T, query, tk string) []string {
	t.Helper()
	var ids []string
	cursor := ""
	for {
		q := query
		if cursor != "" {
			sep := "?"
			if strings.Contains(q, "?") {
				sep = "&"
			}
			q += sep + "cursor=" + cursor
		}
		p := listarPedidos(t, q, tk)
		ids = append(ids, p.ids()...)
		if p.NextCursor == nil {
			return ids
		}
		cursor = *p.NextCursor
	}
}

func (p pedidosPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, it := range p.Items {
		out = append(out, it.ID)
	}
	return out
}

func TestPedidosFiltrosYPaginacion(t *testing.T) {
	tk := cajeroPedidos(t)
	a := abrirLlevar(t, tk, "PRINCIPAL")
	b := abrirLlevar(t, tk, "PRINCIPAL")
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	r := post(t, "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":1}`, sedeID(t, "PRINCIPAL"), mesa), tk)
	esperar(t, r, http.StatusCreated, "")
	var s pedidoResp
	r.json(t, &s)

	if ids := listarTodosPedidos(t, "?tipo=SALON&mesa_id="+mesa, tk); !slices.Equal(ids, []string{s.ID}) {
		t.Fatalf("filtro por tipo y mesa: %v, quería [%s]", ids, s.ID)
	}
	if ids := listarTodosPedidos(t, "?tipo=LLEVAR", tk); !slices.Contains(ids, a.ID) || !slices.Contains(ids, b.ID) || slices.Contains(ids, s.ID) {
		t.Fatalf("filtro por tipo LLEVAR: %v", ids)
	}
	if ids := listarTodosPedidos(t, "?estado=CANCELADO&tipo=LLEVAR", tk); slices.Contains(ids, a.ID) {
		t.Fatalf("filtro por estado: %v", ids)
	}
	esperar(t, call(t, "GET", "/api/v1/pedidos?estado=QUIZAS", "", tk), http.StatusBadRequest, "estado_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?tipo=QUIZAS", "", tk), http.StatusBadRequest, "tipo_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?mesa_id=abc", "", tk), http.StatusBadRequest, "mesa_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")

	p1 := listarPedidos(t, "?limit=1", tk)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarPedidos(t, "?limit=1&cursor="+*p1.NextCursor, tk)
	if len(p2.Items) != 1 {
		t.Fatalf("la segunda página debe tener un elemento: %+v", p2)
	}
	id1, err1 := strconv.Atoi(p1.Items[0].ID)
	id2, err2 := strconv.Atoi(p2.Items[0].ID)
	if err1 != nil || err2 != nil {
		t.Fatalf("ids no numéricos: %q (%v) / %q (%v)", p1.Items[0].ID, err1, p2.Items[0].ID, err2)
	}
	if id1 == id2 || id1 >= id2 {
		t.Fatalf("las páginas no deben compartir id y deben continuar en orden ascendente: %+v / %+v", p1, p2)
	}
}
