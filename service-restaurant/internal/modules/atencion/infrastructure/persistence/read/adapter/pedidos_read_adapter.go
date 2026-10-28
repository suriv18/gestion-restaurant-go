package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// PedidoReadAdapter implementa outbound.PedidoReader.
type PedidoReadAdapter struct{ repo *repository.PedidoSQLReader }

func NewPedidoReadAdapter(db database.DBTX) *PedidoReadAdapter {
	return &PedidoReadAdapter{repo: repository.NewPedidoSQLReader(db)}
}

var _ outbound.PedidoReader = (*PedidoReadAdapter)(nil)

func (a *PedidoReadAdapter) ListarPedidos(ctx context.Context, f outbound.FiltroPedidos) ([]dtoresult.Pedido, error) {
	return a.repo.ListarPedidos(ctx, f)
}
func (a *PedidoReadAdapter) ObtenerPedido(ctx context.Context, id int64) (dtoresult.PedidoDetalle, bool, error) {
	return a.repo.ObtenerPedido(ctx, id)
}
