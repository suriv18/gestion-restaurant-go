// Package adapter implementa los puertos de lectura del módulo produccion sobre SQL.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ComandaReadAdapter implementa outbound.ComandaReader.
type ComandaReadAdapter struct{ repo *repository.ComandaSQLReader }

func NewComandaReadAdapter(db database.DBTX) *ComandaReadAdapter {
	return &ComandaReadAdapter{repo: repository.NewComandaSQLReader(db)}
}

var _ outbound.ComandaReader = (*ComandaReadAdapter)(nil)

func (a *ComandaReadAdapter) ListarComandas(ctx context.Context, f outbound.FiltroComandas) ([]dtoresult.Comanda, error) {
	return a.repo.ListarComandas(ctx, f)
}
