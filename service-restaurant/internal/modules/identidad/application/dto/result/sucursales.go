package result

import "time"

// Sucursal es un local operativo de la empresa (RES-RF-003). Las inactivas llevan motivo y
// actor de la baja.
type Sucursal struct {
	ID                         int64
	NegocioID                  int64
	Codigo                     string
	Nombre                     string
	Direccion                  string
	Ubigeo                     *string
	CodigoEstablecimientoSunat *string
	Telefono                   *string
	EsActivo                   bool
	MotivoEliminacion          *string
	EliminadoEn                *time.Time
	EliminadoPor               *int64
	CreadoEn                   time.Time
	CreadoPor                  int64
	ActualizadoEn              time.Time
	ActualizadoPor             int64
}

// PaginaSucursales: NextCursor vacío indica la última página.
type PaginaSucursales struct {
	Items      []Sucursal
	NextCursor string
}
