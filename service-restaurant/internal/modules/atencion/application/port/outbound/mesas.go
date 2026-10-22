package outbound

import (
	"context"
	"fmt"

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

// ErrNumeroMesaDuplicado distingue la colisión del número de mesa dentro de la sede.
var ErrNumeroMesaDuplicado = fmt.Errorf("número de mesa duplicado: %w", ErrDuplicado)

type NuevaMesa struct {
	SucursalID, ZonaID   int64
	NumeroMesa           string
	Capacidad            int
	Habilitada           bool
	PosicionX, PosicionY *string
}

// CampoOpcional es un cambio de un campo anulable: si Presente, se escribe Valor (nil = NULL).
type CampoOpcional struct {
	Presente bool
	Valor    *string
}

// CambiosMesa: solo los campos presentes se modifican.
type CambiosMesa struct {
	ZonaID               *int64
	NumeroMesa           *string
	Capacidad            *int
	Habilitada           *bool
	PosicionX, PosicionY CampoOpcional
}

// ZonaRef resume la zona que una mesa referencia.
type ZonaRef struct {
	SucursalID int64
	EsActiva   bool
}

// MesaStore escribe mesas dentro de la transacción del comando.
type MesaStore interface {
	Crear(ctx context.Context, n NuevaMesa) (dtoresult.Mesa, error)
	// Obtener lee la mesa bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Mesa, bool, error)
	// ZonaDeMesa lee la zona bloqueándola en modo compartido, para que no se desactive durante el comando.
	ZonaDeMesa(ctx context.Context, zonaID int64) (ZonaRef, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosMesa) (dtoresult.Mesa, bool, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Mesa, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Mesa, bool, error)
}
