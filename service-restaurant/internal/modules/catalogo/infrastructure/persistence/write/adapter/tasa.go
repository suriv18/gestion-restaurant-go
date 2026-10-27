package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// TasaWriteAdapter implementa outbound.TasaStore.
type TasaWriteAdapter struct {
	repo *repository.TasaSQLWriter
}

func NewTasaWriteAdapter(db database.DBTX) *TasaWriteAdapter {
	return &TasaWriteAdapter{repo: repository.NewTasaSQLWriter(db)}
}

var _ outbound.TasaStore = (*TasaWriteAdapter)(nil)

func (a *TasaWriteAdapter) ImpuestoActivo(ctx context.Context, impuestoID int64) (bool, error) {
	return a.repo.ImpuestoActivo(ctx, impuestoID)
}
func (a *TasaWriteAdapter) Crear(ctx context.Context, n outbound.NuevaTasa) (dtoresult.TasaImpuesto, error) {
	return a.repo.Insert(ctx, n)
}
func (a *TasaWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.TasaImpuesto, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *TasaWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosTasa) (dtoresult.TasaImpuesto, bool, error) {
	return a.repo.Update(ctx, id, c)
}
