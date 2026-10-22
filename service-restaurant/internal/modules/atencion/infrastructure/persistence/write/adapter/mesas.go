package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// MesaWriteAdapter implementa outbound.MesaStore.
type MesaWriteAdapter struct{ repo *repository.MesaSQLWriter }

func NewMesaWriteAdapter(db database.DBTX) *MesaWriteAdapter {
	return &MesaWriteAdapter{repo: repository.NewMesaSQLWriter(db)}
}

var _ outbound.MesaStore = (*MesaWriteAdapter)(nil)

func (a *MesaWriteAdapter) Crear(ctx context.Context, n outbound.NuevaMesa) (dtoresult.Mesa, error) {
	return a.repo.Insert(ctx, n)
}
func (a *MesaWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *MesaWriteAdapter) ZonaDeMesa(ctx context.Context, zonaID int64) (outbound.ZonaRef, bool, error) {
	return a.repo.GetZona(ctx, zonaID)
}
func (a *MesaWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosMesa) (dtoresult.Mesa, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *MesaWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Mesa, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *MesaWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return a.repo.SetActiva(ctx, id)
}
