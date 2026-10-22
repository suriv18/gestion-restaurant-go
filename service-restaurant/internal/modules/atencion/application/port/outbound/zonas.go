package outbound

import (
	"context"
	"fmt"

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

// ErrNombreZonaDuplicado distingue la colisión del nombre de zona dentro de la sede.
var ErrNombreZonaDuplicado = fmt.Errorf("nombre de zona duplicado: %w", ErrDuplicado)

type NuevaZona struct {
	SucursalID int64
	Nombre     string
	Orden      int
}

type CambiosZona struct {
	Nombre *string
	Orden  *int
}

// ZonaStore escribe zonas dentro de la transacción del comando.
type ZonaStore interface {
	Crear(ctx context.Context, n NuevaZona) (dtoresult.Zona, error)
	// Obtener lee la zona bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Zona, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosZona) (dtoresult.Zona, bool, error)
	ContarMesasActivas(ctx context.Context, zonaID int64) (int, error)
	CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Zona, bool, error)
}
