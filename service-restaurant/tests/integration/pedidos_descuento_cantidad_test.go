package integration

import (
	"net/http"
	"testing"
)

// D4: sin DESCUENTO_AUTORIZAR no se puede bajar la cantidad de una línea que conserva descuento.
func TestBajarLaCantidadConDescuentoExigeElPermiso(t *testing.T) {
	cajero := cajeroPedidos(t)
	mesero := meseroPedidos(t)
	p := abrirLlevar(t, cajero, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	r := agregarConsumo(t, cajero, p.ID, p.Cuentas[0].ID, prod, "3", `,"descuento_monto":"1.00"`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles/" + l.ID

	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"2"}`, mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"4"}`, mesero), http.StatusOK, "")
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"2"}`, cajero), http.StatusOK, "")
}
