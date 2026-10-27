package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// CategoriaReadAdapter implementa outbound.CategoriaReader.
type CategoriaReadAdapter struct {
	repo *repository.CategoriaSQLReader
}

func NewCategoriaReadAdapter(db database.DBTX) *CategoriaReadAdapter {
	return &CategoriaReadAdapter{repo: repository.NewCategoriaSQLReader(db)}
}

var _ outbound.CategoriaReader = (*CategoriaReadAdapter)(nil)

func (a *CategoriaReadAdapter) ListarCategorias(ctx context.Context, f outbound.FiltroCategorias) ([]dtoresult.Categoria, error) {
	return a.repo.ListarCategorias(ctx, f)
}
func (a *CategoriaReadAdapter) ObtenerCategoria(ctx context.Context, id int64) (dtoresult.Categoria, bool, error) {
	return a.repo.ObtenerCategoria(ctx, id)
}
