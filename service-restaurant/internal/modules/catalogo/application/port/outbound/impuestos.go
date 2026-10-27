package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
)

type FiltroImpuestos struct {
	Activo      *bool
	DespuesDeID int64
	Limite      int
}

// FiltroTasas: ImpuestoID 0 no filtra por impuesto.
type FiltroTasas struct {
	ImpuestoID  int64
	DespuesDeID int64
	Limite      int
}

// ImpuestoReader lee impuestos y tasas sin bloquear filas.
type ImpuestoReader interface {
	ListarImpuestos(ctx context.Context, f FiltroImpuestos) ([]dtoresult.Impuesto, error)
	ObtenerImpuesto(ctx context.Context, id int64) (dtoresult.Impuesto, bool, error)
	ListarTasas(ctx context.Context, f FiltroTasas) ([]dtoresult.TasaImpuesto, error)
	ObtenerTasa(ctx context.Context, id int64) (dtoresult.TasaImpuesto, bool, error)
}

// NuevaTasa lleva la tasa y las fechas ya normalizadas (decimal en texto, YYYY-MM-DD).
type NuevaTasa struct {
	ImpuestoID   int64
	Tasa         string
	VigenteDesde string
	VigenteHasta *string
	Fundamento   *string
}

// CambiosTasa es el estado resultante de una edición: el caso de uso ya fusionó los campos
// presentes con la fila bloqueada, de modo que el almacén solo escribe valores completos.
type CambiosTasa struct {
	Tasa         string
	VigenteDesde string
	VigenteHasta *string
	Fundamento   *string
}

// TasaStore escribe tasas dentro de la transacción del comando.
type TasaStore interface {
	// ImpuestoActivo indica si el impuesto existe y está activo (bloqueándolo FOR SHARE para
	// que no se dé de baja durante el alta).
	ImpuestoActivo(ctx context.Context, impuestoID int64) (bool, error)
	Crear(ctx context.Context, n NuevaTasa) (dtoresult.TasaImpuesto, error)
	// Obtener lee la tasa bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.TasaImpuesto, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosTasa) (dtoresult.TasaImpuesto, bool, error)
}
