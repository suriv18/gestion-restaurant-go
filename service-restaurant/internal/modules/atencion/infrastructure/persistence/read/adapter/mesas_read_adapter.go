package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// MesaReadAdapter implementa outbound.MesaReader.
type MesaReadAdapter struct{ repo *repository.MesaSQLReader }

func NewMesaReadAdapter(db database.DBTX) *MesaReadAdapter {
	return &MesaReadAdapter{repo: repository.NewMesaSQLReader(db)}
}

var _ outbound.MesaReader = (*MesaReadAdapter)(nil)

func (a *MesaReadAdapter) ListarMesas(ctx context.Context, f outbound.FiltroMesas) ([]dtoresult.Mesa, error) {
	return a.repo.ListMesas(ctx, f)
}
func (a *MesaReadAdapter) ObtenerMesa(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return a.repo.GetMesa(ctx, id)
}
