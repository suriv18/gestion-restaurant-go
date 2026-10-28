package command

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	appctx "service-restaurant/internal/shared/application/context"
)

// pedidoStoreFake implementa outbound.PedidoStore con respuestas fijadas por la prueba.
type pedidoStoreFake struct {
	mesa       outbound.MesaRef
	mesaExiste bool
	empleado   *int64
	errCrear   error
	creado     outbound.NuevoPedido
	creaciones int

	pedido         outbound.PedidoRef
	pedidoExiste   bool
	cuenta         outbound.CuentaRef
	cuentaExiste   bool
	producto       outbound.ProductoVenta
	productoExiste bool
	detalles       map[int64]outbound.DetalleRef

	insertado       outbound.NuevoDetalle
	inserciones     int
	cambios         outbound.CambiosDetalle
	actualizaciones int
}

var _ outbound.PedidoStore = (*pedidoStoreFake)(nil)

func (f *pedidoStoreFake) MesaParaPedido(context.Context, int64) (outbound.MesaRef, bool, error) {
	return f.mesa, f.mesaExiste, nil
}
func (f *pedidoStoreFake) EmpleadoDeUsuario(context.Context, int64) (*int64, error) {
	return f.empleado, nil
}
func (f *pedidoStoreFake) CrearPedido(_ context.Context, n outbound.NuevoPedido) (dtoresult.PedidoDetalle, error) {
	f.creaciones++
	f.creado = n
	return dtoresult.PedidoDetalle{Pedido: dtoresult.Pedido{ID: 5, SucursalID: n.SucursalID, Tipo: n.Tipo}}, f.errCrear
}
func (f *pedidoStoreFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *pedidoStoreFake) CuentaDelPedido(context.Context, int64, int64) (outbound.CuentaRef, bool, error) {
	return f.cuenta, f.cuentaExiste, nil
}
func (f *pedidoStoreFake) ProductoParaVenta(context.Context, int64, int64) (outbound.ProductoVenta, bool, error) {
	return f.producto, f.productoExiste, nil
}
func (f *pedidoStoreFake) DetalleDelPedido(_ context.Context, _ int64, id int64) (outbound.DetalleRef, bool, error) {
	d, ok := f.detalles[id]
	return d, ok, nil
}
func (f *pedidoStoreFake) InsertarDetalle(_ context.Context, n outbound.NuevoDetalle) (dtoresult.Linea, error) {
	f.inserciones++
	f.insertado = n
	return dtoresult.Linea{ID: 99, Cantidad: n.Cantidad}, nil
}
func (f *pedidoStoreFake) ActualizarDetalle(_ context.Context, id int64, c outbound.CambiosDetalle) (dtoresult.Linea, error) {
	f.actualizaciones++
	f.cambios = c
	return dtoresult.Linea{ID: id}, nil
}

// actorPedidos arma un actor con los permisos dados en una sola sede.
func actorPedidos(sede int64, permisos ...string) appctx.Actor {
	g := make([]appctx.Grant, 0, len(permisos))
	for _, p := range permisos {
		g = append(g, appctx.Grant{SucursalID: sede, Permiso: p})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func idPtr(n int64) *int64 { return &n }
