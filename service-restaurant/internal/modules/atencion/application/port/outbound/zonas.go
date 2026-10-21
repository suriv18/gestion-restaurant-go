package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// FiltroZonas acota el listado a las sedes visibles (SedeIDs) y, opcionalmente, a su estado.
type FiltroZonas struct {
	SedeIDs     []int64
	Activo      *bool
	DespuesDeID int64
	Limite      int
}

// ZonaReader lee zonas sin bloquear filas.
type ZonaReader interface {
	ListarZonas(ctx context.Context, f FiltroZonas) ([]dtoresult.Zona, error)
	ObtenerZona(ctx context.Context, id int64) (dtoresult.Zona, bool, error)
}
