// Package adapter implementa los puertos de salida del módulo produccion sobre SQL.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// ComandaWriteAdapter implementa outbound.EnvioStore y outbound.TransicionStore.
type ComandaWriteAdapter struct{ repo *repository.ComandaSQLWriter }

func NewComandaWriteAdapter(db database.DBTX) *ComandaWriteAdapter {
	return &ComandaWriteAdapter{repo: repository.NewComandaSQLWriter(db)}
}

var (
	_ outbound.EnvioStore      = (*ComandaWriteAdapter)(nil)
	_ outbound.TransicionStore = (*ComandaWriteAdapter)(nil)
)

func (a *ComandaWriteAdapter) BloquearPedido(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	return a.repo.GetPedidoForUpdate(ctx, id)
}
func (a *ComandaWriteAdapter) Estacion(ctx context.Context, id int64) (outbound.EstacionRef, bool, error) {
	return a.repo.GetEstacion(ctx, id)
}
func (a *ComandaWriteAdapter) LineasParaEnvio(ctx context.Context, pedidoID, sucursalID int64, ids []int64) ([]outbound.LineaPedidoRef, error) {
	return a.repo.GetLineasParaEnvio(ctx, pedidoID, sucursalID, ids)
}
func (a *ComandaWriteAdapter) CrearComanda(ctx context.Context, n outbound.NuevaComanda) (dtoresult.Comanda, error) {
	return a.repo.InsertComanda(ctx, n)
}

func (a *ComandaWriteAdapter) BloquearLineaComanda(ctx context.Context, id int64) (outbound.LineaComandaRef, bool, error) {
	return a.repo.GetLineaComandaForUpdate(ctx, id)
}
func (a *ComandaWriteAdapter) TransicionarLinea(ctx context.Context, id int64, destino string) (dtoresult.ComandaLinea, error) {
	return a.repo.UpdateLineaComanda(ctx, id, destino)
}
