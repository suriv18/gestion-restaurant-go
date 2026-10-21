package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
)

// FiltroSucursales acota el listado a lo que el actor puede ver: las sedes activas de SedeIDs
// y, si IncluirInactivas, todas las inactivas (quien administra sedes debe poder restaurarlas).
type FiltroSucursales struct {
	SedeIDs          []int64
	IncluirInactivas bool
	Activo           *bool
	DespuesDeID      int64
	Limite           int
}

// SucursalReader lee sedes sin bloquear filas.
type SucursalReader interface {
	ListarSucursales(ctx context.Context, f FiltroSucursales) ([]dtoresult.Sucursal, error)
	ObtenerSucursal(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error)
}
