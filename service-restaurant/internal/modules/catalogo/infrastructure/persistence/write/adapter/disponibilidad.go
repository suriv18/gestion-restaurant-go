package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// DisponibilidadWriteAdapter implementa outbound.DisponibilidadStore.
type DisponibilidadWriteAdapter struct {
	repo *repository.DisponibilidadSQLWriter
}

func NewDisponibilidadWriteAdapter(db database.DBTX) *DisponibilidadWriteAdapter {
	return &DisponibilidadWriteAdapter{repo: repository.NewDisponibilidadSQLWriter(db)}
}

var _ outbound.DisponibilidadStore = (*DisponibilidadWriteAdapter)(nil)

func (a *DisponibilidadWriteAdapter) BloquearProducto(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	return a.repo.LockProducto(ctx, id)
}
func (a *DisponibilidadWriteAdapter) Guardar(ctx context.Context, productoID, sucursalID int64, c outbound.CambiosDisponibilidad) (dtoresult.ProductoSucursal, error) {
	return a.repo.Upsert(ctx, productoID, sucursalID, c)
}
