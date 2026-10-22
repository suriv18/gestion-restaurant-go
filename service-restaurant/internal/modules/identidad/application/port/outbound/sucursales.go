package outbound

import (
	"context"
	"fmt"

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

// ErrCodigoSucursalDuplicado distingue la colisión del código de sede de otras unicidades.
var ErrCodigoSucursalDuplicado = fmt.Errorf("código de sucursal duplicado: %w", ErrDuplicado)

type NuevaSucursal struct {
	Codigo, Nombre, Direccion                    string
	Ubigeo, CodigoEstablecimientoSunat, Telefono *string
}

type CambiosSucursal struct {
	Codigo, Nombre, Direccion                    *string
	Ubigeo, CodigoEstablecimientoSunat, Telefono CampoOpcional
}

// SucursalStore escribe sedes dentro de la transacción del comando.
type SucursalStore interface {
	// Crear inserta la sede en la empresa única.
	Crear(ctx context.Context, n NuevaSucursal) (dtoresult.Sucursal, error)
	// Obtener lee la sede bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosSucursal) (dtoresult.Sucursal, bool, error)
	// BloquearNegocio serializa las bajas concurrentes para que la regla de la última sede sea
	// fiable (las altas solo esperan por los locks de FK sobre la fila de la empresa).
	BloquearNegocio(ctx context.Context) error
	ContarActivas(ctx context.Context) (int, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Sucursal, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error)
	// ContarAdministradoresEfectivos cuenta usuarios con USUARIO_ADMINISTRAR efectivo (tras la baja).
	ContarAdministradoresEfectivos(ctx context.Context) (int, error)
}
