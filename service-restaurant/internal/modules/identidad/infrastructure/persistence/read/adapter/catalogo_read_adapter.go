package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// CatalogoReadAdapter implementa outbound.CatalogoReader.
type CatalogoReadAdapter struct{ repo *repository.CatalogoSQLReader }

func NewCatalogoReadAdapter(db database.DBTX) *CatalogoReadAdapter {
	return &CatalogoReadAdapter{repo: repository.NewCatalogoSQLReader(db)}
}

var _ outbound.CatalogoReader = (*CatalogoReadAdapter)(nil)

func (a *CatalogoReadAdapter) ListarRoles(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.Rol, error) {
	return a.repo.ListRoles(ctx, despuesDeID, limite)
}
func (a *CatalogoReadAdapter) ObtenerRol(ctx context.Context, id int64) (dtoresult.Rol, bool, error) {
	return a.repo.GetRol(ctx, id)
}
func (a *CatalogoReadAdapter) ListarPermisos(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.Permiso, error) {
	return a.repo.ListPermisos(ctx, despuesDeID, limite)
}
func (a *CatalogoReadAdapter) ObtenerPermiso(ctx context.Context, id int64) (dtoresult.Permiso, bool, error) {
	return a.repo.GetPermiso(ctx, id)
}
