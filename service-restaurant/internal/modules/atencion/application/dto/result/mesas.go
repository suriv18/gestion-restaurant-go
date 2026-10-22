package result

import "time"

// Mesa es una mesa del salón. La ocupación no se expone: se deriva de los pedidos abiertos (I3).
// Las inactivas llevan motivo y actor de la baja.
type Mesa struct {
	ID                int64
	SucursalID        int64
	ZonaID            int64
	NumeroMesa        string
	Capacidad         int
	Habilitada        bool
	PosicionX         *string
	PosicionY         *string
	EsActivo          bool
	MotivoEliminacion *string
	EliminadoEn       *time.Time
	EliminadoPor      *int64
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
}

// PaginaMesas: NextCursor vacío indica la última página.
type PaginaMesas struct {
	Items      []Mesa
	NextCursor string
}
