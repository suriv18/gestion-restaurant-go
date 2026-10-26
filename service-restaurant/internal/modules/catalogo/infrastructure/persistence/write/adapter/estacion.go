package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// EstacionWriteAdapter implementa outbound.EstacionStore.
type EstacionWriteAdapter struct {
	repo *repository.EstacionSQLWriter
}

func NewEstacionWriteAdapter(db database.DBTX) *EstacionWriteAdapter {
	return &EstacionWriteAdapter{repo: repository.NewEstacionSQLWriter(db)}
}

var _ outbound.EstacionStore = (*EstacionWriteAdapter)(nil)

func (a *EstacionWriteAdapter) Crear(ctx context.Context, n outbound.NuevaEstacion) (dtoresult.Estacion, error) {
	return a.repo.Insert(ctx, n)
}
func (a *EstacionWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Estacion, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *EstacionWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosEstacion) (dtoresult.Estacion, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *EstacionWriteAdapter) DarBaja(ctx context.Context, id int64) (dtoresult.Estacion, bool, error) {
	return a.repo.SetActivo(ctx, id, false)
}
func (a *EstacionWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Estacion, bool, error) {
	return a.repo.SetActivo(ctx, id, true)
}
func (a *EstacionWriteAdapter) EnUso(ctx context.Context, id int64) (bool, error) {
	return a.repo.EnUso(ctx, id)
}
