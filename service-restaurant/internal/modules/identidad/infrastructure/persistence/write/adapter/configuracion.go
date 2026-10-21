package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// NegocioWriteAdapter implementa outbound.NegocioStore.
type NegocioWriteAdapter struct{ repo *repository.NegocioSQLWriter }

func NewNegocioWriteAdapter(db database.DBTX) *NegocioWriteAdapter {
	return &NegocioWriteAdapter{repo: repository.NewNegocioSQLWriter(db)}
}

var _ outbound.NegocioStore = (*NegocioWriteAdapter)(nil)

func (a *NegocioWriteAdapter) Actualizar(ctx context.Context, c outbound.CambiosNegocio) (dtoresult.Negocio, bool, error) {
	return a.repo.Update(ctx, c)
}
