package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// PedidoWriteAdapter implementa outbound.PedidoStore.
type PedidoWriteAdapter struct{ repo *repository.PedidoSQLWriter }

func NewPedidoWriteAdapter(db database.DBTX) *PedidoWriteAdapter {
	return &PedidoWriteAdapter{repo: repository.NewPedidoSQLWriter(db)}
}

var _ outbound.PedidoStore = (*PedidoWriteAdapter)(nil)

func (a *PedidoWriteAdapter) MesaParaPedido(ctx context.Context, mesaID int64) (outbound.MesaRef, bool, error) {
	return a.repo.GetMesa(ctx, mesaID)
}
func (a *PedidoWriteAdapter) EmpleadoDeUsuario(ctx context.Context, usuarioID int64) (*int64, error) {
	return a.repo.GetEmpleadoDeUsuario(ctx, usuarioID)
}
func (a *PedidoWriteAdapter) CrearPedido(ctx context.Context, n outbound.NuevoPedido) (dtoresult.PedidoDetalle, error) {
	return a.repo.InsertPedido(ctx, n)
}
func (a *PedidoWriteAdapter) BloquearPedido(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	return a.repo.GetPedidoForUpdate(ctx, id)
}
func (a *PedidoWriteAdapter) CuentaDelPedido(ctx context.Context, pedidoID, cuentaID int64) (outbound.CuentaRef, bool, error) {
	return a.repo.GetCuentaForUpdate(ctx, pedidoID, cuentaID)
}
func (a *PedidoWriteAdapter) ProductoParaVenta(ctx context.Context, productoID, sedeID int64) (outbound.ProductoVenta, bool, error) {
	return a.repo.GetProductoVenta(ctx, productoID, sedeID)
}
func (a *PedidoWriteAdapter) DetalleDelPedido(ctx context.Context, pedidoID, detalleID int64) (outbound.DetalleRef, bool, error) {
	return a.repo.GetDetalleForUpdate(ctx, pedidoID, detalleID)
}
func (a *PedidoWriteAdapter) InsertarDetalle(ctx context.Context, n outbound.NuevoDetalle) (dtoresult.Linea, error) {
	return a.repo.InsertDetalle(ctx, n)
}
func (a *PedidoWriteAdapter) ActualizarDetalle(ctx context.Context, id int64, c outbound.CambiosDetalle) (dtoresult.Linea, error) {
	return a.repo.UpdateDetalle(ctx, id, c)
}

var _ outbound.DivisionStore = (*PedidoWriteAdapter)(nil)

func (a *PedidoWriteAdapter) LineasDelPedido(ctx context.Context, pedidoID int64) ([]outbound.LineaCuenta, error) {
	return a.repo.GetLineasDelPedido(ctx, pedidoID)
}
func (a *PedidoWriteAdapter) CuentasDelPedido(ctx context.Context, pedidoID int64) ([]outbound.CuentaRef, error) {
	return a.repo.GetCuentasForUpdate(ctx, pedidoID)
}
func (a *PedidoWriteAdapter) CuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error) {
	return a.repo.GetCuentasConPago(ctx, cuentaIDs)
}
func (a *PedidoWriteAdapter) CrearCuenta(ctx context.Context, n outbound.NuevaCuenta) (int64, error) {
	return a.repo.InsertCuenta(ctx, n)
}
func (a *PedidoWriteAdapter) MoverLineas(ctx context.Context, pedidoID, cuentaID int64, lineaIDs []int64) error {
	return a.repo.MoveLineas(ctx, pedidoID, cuentaID, lineaIDs)
}
func (a *PedidoWriteAdapter) CuentaConLineas(ctx context.Context, pedidoID, cuentaID int64) (dtoresult.Cuenta, error) {
	return a.repo.GetCuentaConLineas(ctx, pedidoID, cuentaID)
}
