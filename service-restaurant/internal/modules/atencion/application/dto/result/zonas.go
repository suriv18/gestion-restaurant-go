// Package result agrupa los resultados de aplicación del módulo atencion.
package result

import "time"

// Zona es un salón, terraza o nivel del local.
type Zona struct {
	ID                 int64
	SucursalID         int64
	Nombre             string
	OrdenVisualizacion int
	EsActivo           bool
	CreadoEn           time.Time
	CreadoPor          int64
	ActualizadoEn      time.Time
	ActualizadoPor     int64
}

// PaginaZonas: NextCursor vacío indica la última página.
type PaginaZonas struct {
	Items      []Zona
	NextCursor string
}
