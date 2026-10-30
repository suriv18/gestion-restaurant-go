package command

import (
	"context"
	"slices"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
)

// anulacionFake implementa outbound.AnulacionStore con respuestas fijadas por la prueba.
type anulacionFake struct {
	pedido          outbound.PedidoRef
	pedidoExiste    bool
	detalle         outbound.DetalleRef
	detalleExiste   bool
	cuenta          outbound.CuentaRef
	cuentaExiste    bool
	cuentas         []outbound.CuentaRef
	conPago         []int64
	entregadaLinea  bool
	entregadaPedido bool

	anulaciones   int
	anulada       int64
	cancelaciones int
	motivo        outbound.Anulacion
}

var _ outbound.AnulacionStore = (*anulacionFake)(nil)

func (f *anulacionFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *anulacionFake) DetalleDelPedido(context.Context, int64, int64) (outbound.DetalleRef, bool, error) {
	return f.detalle, f.detalleExiste, nil
}
func (f *anulacionFake) CuentaDelPedido(context.Context, int64, int64) (outbound.CuentaRef, bool, error) {
	return f.cuenta, f.cuentaExiste, nil
}
func (f *anulacionFake) CuentasDelPedido(context.Context, int64) ([]outbound.CuentaRef, error) {
	return f.cuentas, nil
}
func (f *anulacionFake) CuentasConPago(_ context.Context, ids []int64) ([]int64, error) {
	var out []int64
	for _, id := range ids {
		if slices.Contains(f.conPago, id) {
			out = append(out, id)
		}
	}
	return out, nil
}
func (f *anulacionFake) CocinaEntregadaEnLinea(context.Context, int64, int64) (bool, error) {
	return f.entregadaLinea, nil
}
func (f *anulacionFake) CocinaEntregadaEnPedido(context.Context, int64) (bool, error) {
	return f.entregadaPedido, nil
}
func (f *anulacionFake) AnularLinea(_ context.Context, _, detalleID int64, a outbound.Anulacion) (dtoresult.Linea, error) {
	f.anulaciones++
	f.anulada = detalleID
	f.motivo = a
	return dtoresult.Linea{ID: detalleID, Estado: "ANULADO"}, nil
}
func (f *anulacionFake) CancelarPedido(_ context.Context, pedidoID int64, a outbound.Anulacion) (dtoresult.PedidoDetalle, error) {
	f.cancelaciones++
	f.motivo = a
	return dtoresult.PedidoDetalle{Pedido: dtoresult.Pedido{ID: pedidoID, Estado: "CANCELADO"}}, nil
}

// fakeAnulacion es un pedido abierto con dos cuentas abiertas y una línea activa (55) en la 20.
func fakeAnulacion() *anulacionFake {
	return &anulacionFake{
		pedido:       outbound.PedidoRef{ID: 10, SucursalID: 1, Estado: "REGISTRADO"},
		pedidoExiste: true,
		detalle: outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, ProductoID: 30,
			Cantidad: "2.000", PrecioUnitario: "10.000000", DescuentoTotal: "0.00", Estado: "ACTIVO"},
		detalleExiste: true,
		cuenta:        outbound.CuentaRef{ID: 20, PedidoID: 10, Estado: "ABIERTA"},
		cuentaExiste:  true,
		cuentas: []outbound.CuentaRef{
			{ID: 20, PedidoID: 10, Estado: "ABIERTA"}, {ID: 21, PedidoID: 10, Estado: "ABIERTA"}},
	}
}
