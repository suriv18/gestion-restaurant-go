package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// FiltroMesas acota el listado a las sedes visibles (SedeIDs) y, opcionalmente, a su estado.
type FiltroMesas struct {
	SedeIDs     []int64
	Activo      *bool
	DespuesDeID int64
	Limite      int
}

// MesaReader lee mesas sin bloquear filas.
type MesaReader interface {
	ListarMesas(ctx context.Context, f FiltroMesas) ([]dtoresult.Mesa, error)
	ObtenerMesa(ctx context.Context, id int64) (dtoresult.Mesa, bool, error)
}
