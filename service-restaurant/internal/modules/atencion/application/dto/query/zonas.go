// Package query agrupa los DTO de consultas del módulo atencion.
package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarZonas pagina por cursor (id ascendente) las zonas de las sedes que el actor puede ver.
// SucursalID y Activo filtran cuando no son nil.
type ListarZonas struct {
	Actor      appctx.Actor
	SucursalID *int64
	Activo     *bool
	Limite     int
	Cursor     string
}

type ObtenerZona struct {
	Actor appctx.Actor
	ID    int64
}
