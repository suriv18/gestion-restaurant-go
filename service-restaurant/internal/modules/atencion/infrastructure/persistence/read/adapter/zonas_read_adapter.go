// Package adapter implementa los puertos de salida de lectura del módulo atencion.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ZonaReadAdapter implementa outbound.ZonaReader.
type ZonaReadAdapter struct{ repo *repository.ZonaSQLReader }

func NewZonaReadAdapter(db database.DBTX) *ZonaReadAdapter {
	return &ZonaReadAdapter{repo: repository.NewZonaSQLReader(db)}
}

var _ outbound.ZonaReader = (*ZonaReadAdapter)(nil)

func (a *ZonaReadAdapter) ListarZonas(ctx context.Context, f outbound.FiltroZonas) ([]dtoresult.Zona, error) {
	return a.repo.ListZonas(ctx, f)
}
func (a *ZonaReadAdapter) ObtenerZona(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	return a.repo.GetZona(ctx, id)
}
