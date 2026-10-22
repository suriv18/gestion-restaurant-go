package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarMesas pagina por cursor (id ascendente) las mesas de las sedes que el actor puede ver.
type ListarMesas struct {
	Actor      appctx.Actor
	SucursalID *int64
	Activo     *bool
	Limite     int
	Cursor     string
}

type ObtenerMesa struct {
	Actor appctx.Actor
	ID    int64
}
