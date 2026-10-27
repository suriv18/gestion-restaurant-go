package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// ProductoWriteAdapter implementa outbound.ProductoStore.
type ProductoWriteAdapter struct {
	repo *repository.ProductoSQLWriter
}

func NewProductoWriteAdapter(db database.DBTX) *ProductoWriteAdapter {
	return &ProductoWriteAdapter{repo: repository.NewProductoSQLWriter(db)}
}

var _ outbound.ProductoStore = (*ProductoWriteAdapter)(nil)

func (a *ProductoWriteAdapter) Crear(ctx context.Context, n outbound.NuevoProducto) (dtoresult.Producto, error) {
	return a.repo.Insert(ctx, n)
}
func (a *ProductoWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *ProductoWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosProducto) (dtoresult.Producto, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *ProductoWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Producto, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *ProductoWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Producto, bool, error) {
	return a.repo.SetActivo(ctx, id)
}
