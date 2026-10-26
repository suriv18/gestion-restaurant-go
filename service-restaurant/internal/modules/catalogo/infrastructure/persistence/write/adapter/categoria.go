package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// CategoriaWriteAdapter implementa outbound.CategoriaStore.
type CategoriaWriteAdapter struct {
	repo *repository.CategoriaSQLWriter
}

func NewCategoriaWriteAdapter(db database.DBTX) *CategoriaWriteAdapter {
	return &CategoriaWriteAdapter{repo: repository.NewCategoriaSQLWriter(db)}
}

var _ outbound.CategoriaStore = (*CategoriaWriteAdapter)(nil)

func (a *CategoriaWriteAdapter) Crear(ctx context.Context, n outbound.NuevaCategoria) (dtoresult.Categoria, error) {
	return a.repo.Insert(ctx, n)
}
func (a *CategoriaWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Categoria, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *CategoriaWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosCategoria) (dtoresult.Categoria, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *CategoriaWriteAdapter) DarBaja(ctx context.Context, id int64) (dtoresult.Categoria, bool, error) {
	return a.repo.SetActivo(ctx, id, false)
}
func (a *CategoriaWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Categoria, bool, error) {
	return a.repo.SetActivo(ctx, id, true)
}
func (a *CategoriaWriteAdapter) TieneProductosActivos(ctx context.Context, id int64) (bool, error) {
	return a.repo.TieneProductosActivos(ctx, id)
}
