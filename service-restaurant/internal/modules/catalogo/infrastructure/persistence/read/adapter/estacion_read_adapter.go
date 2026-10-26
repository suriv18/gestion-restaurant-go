package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// EstacionReadAdapter implementa outbound.EstacionReader.
type EstacionReadAdapter struct {
	repo *repository.EstacionSQLReader
}

func NewEstacionReadAdapter(db database.DBTX) *EstacionReadAdapter {
	return &EstacionReadAdapter{repo: repository.NewEstacionSQLReader(db)}
}

var _ outbound.EstacionReader = (*EstacionReadAdapter)(nil)

func (a *EstacionReadAdapter) ListarEstaciones(ctx context.Context, f outbound.FiltroEstaciones) ([]dtoresult.Estacion, error) {
	return a.repo.ListarEstaciones(ctx, f)
}
func (a *EstacionReadAdapter) ObtenerEstacion(ctx context.Context, id int64) (dtoresult.Estacion, bool, error) {
	return a.repo.ObtenerEstacion(ctx, id)
}
