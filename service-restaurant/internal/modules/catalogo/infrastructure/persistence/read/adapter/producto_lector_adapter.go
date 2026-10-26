package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ProductoLectorAdapter implementa outbound.ProductoReader.
type ProductoLectorAdapter struct {
	repo *repository.ProductoSQLReader
}

func NewProductoLectorAdapter(db database.DBTX) *ProductoLectorAdapter {
	return &ProductoLectorAdapter{repo: repository.NewProductoSQLReader(db)}
}

var _ outbound.ProductoReader = (*ProductoLectorAdapter)(nil)

func (a *ProductoLectorAdapter) ListarProductos(ctx context.Context, f outbound.FiltroProductos) ([]dtoresult.Producto, error) {
	return a.repo.ListarProductos(ctx, f)
}
func (a *ProductoLectorAdapter) ObtenerProducto(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	return a.repo.ObtenerProducto(ctx, id)
}
