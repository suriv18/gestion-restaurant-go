package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
)

// FiltroEstaciones acota el listado a las sedes dadas (nunca vacío de significado: un arreglo
// vacío no devuelve nada), al estado y al texto (código o nombre, ya sin comodines de LIKE).
type FiltroEstaciones struct {
	SedeIDs     []int64
	Activo      *bool
	Texto       string
	DespuesDeID int64
	Limite      int
}

// EstacionReader lee estaciones sin bloquear filas.
type EstacionReader interface {
	ListarEstaciones(ctx context.Context, f FiltroEstaciones) ([]dtoresult.Estacion, error)
	ObtenerEstacion(ctx context.Context, id int64) (dtoresult.Estacion, bool, error)
}

type NuevaEstacion struct {
	SucursalID int64
	Codigo     string
	Nombre     string
}

// CambiosEstacion: solo los campos presentes se modifican.
type CambiosEstacion struct {
	Codigo *string
	Nombre *string
}

// EstacionStore escribe estaciones dentro de la transacción del comando.
type EstacionStore interface {
	Crear(ctx context.Context, n NuevaEstacion) (dtoresult.Estacion, error)
	// Obtener lee la estación bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Estacion, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosEstacion) (dtoresult.Estacion, bool, error)
	DarBaja(ctx context.Context, id int64) (dtoresult.Estacion, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Estacion, bool, error)
	// EnUso indica si alguna fila activa de productos_sucursal usa la estación. Se invoca con la
	// fila de la estación ya bloqueada, para que una asignación concurrente no la deje huérfana.
	EnUso(ctx context.Context, id int64) (bool, error)
}
