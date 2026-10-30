// Package query agrupa los DTO de consultas del módulo produccion.
package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarComandas pagina por cursor (id ascendente) las comandas de las sedes donde el actor opera la
// cocina. Estado vacío significa «con alguna línea pendiente de entregar» (PENDIENTE, PREPARACION o
// LISTO); con un estado, las comandas que tienen alguna línea en él.
type ListarComandas struct {
	Actor      appctx.Actor
	SucursalID *int64
	EstacionID *int64
	Estado     string
	Limite     int
	Cursor     string
}
