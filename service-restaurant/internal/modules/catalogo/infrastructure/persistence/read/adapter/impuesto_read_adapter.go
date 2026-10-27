package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ImpuestoReadAdapter implementa outbound.ImpuestoReader.
type ImpuestoReadAdapter struct {
	repo *repository.ImpuestoSQLReader
}

func NewImpuestoReadAdapter(db database.DBTX) *ImpuestoReadAdapter {
	return &ImpuestoReadAdapter{repo: repository.NewImpuestoSQLReader(db)}
}

var _ outbound.ImpuestoReader = (*ImpuestoReadAdapter)(nil)

func (a *ImpuestoReadAdapter) ListarImpuestos(ctx context.Context, f outbound.FiltroImpuestos) ([]dtoresult.Impuesto, error) {
	return a.repo.ListarImpuestos(ctx, f)
}
func (a *ImpuestoReadAdapter) ObtenerImpuesto(ctx context.Context, id int64) (dtoresult.Impuesto, bool, error) {
	return a.repo.ObtenerImpuesto(ctx, id)
}
func (a *ImpuestoReadAdapter) ListarTasas(ctx context.Context, f outbound.FiltroTasas) ([]dtoresult.TasaImpuesto, error) {
	return a.repo.ListarTasas(ctx, f)
}
func (a *ImpuestoReadAdapter) ObtenerTasa(ctx context.Context, id int64) (dtoresult.TasaImpuesto, bool, error) {
	return a.repo.ObtenerTasa(ctx, id)
}
