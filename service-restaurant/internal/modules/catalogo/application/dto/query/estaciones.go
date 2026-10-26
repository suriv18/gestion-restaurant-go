package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarEstaciones pagina por cursor (id ascendente) las estaciones de las sedes visibles;
// SucursalID nil no acota a una sede; Texto busca por código o nombre.
type ListarEstaciones struct {
	Actor      appctx.Actor
	SucursalID *int64
	Activo     *bool
	Texto      string
	Limite     int
	Cursor     string
}

type ObtenerEstacion struct {
	Actor appctx.Actor
	ID    int64
}
