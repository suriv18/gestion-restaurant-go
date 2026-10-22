package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarSucursales pagina por cursor (id ascendente) las sedes que el actor puede ver.
// Activo filtra por es_activo cuando no es nil.
type ListarSucursales struct {
	Actor  appctx.Actor
	Limite int
	Cursor string
	Activo *bool
}

type ObtenerSucursal struct {
	Actor appctx.Actor
	ID    int64
}
