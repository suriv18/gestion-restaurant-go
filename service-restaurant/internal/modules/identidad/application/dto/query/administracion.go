package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarRoles y ListarPermisos paginan por cursor (id ascendente). Cursor vacío = primera página.
type ListarRoles struct {
	Actor  appctx.Actor
	Limite int
	Cursor string
}

type ObtenerRol struct {
	Actor appctx.Actor
	ID    int64
}

type ListarPermisos struct {
	Actor  appctx.Actor
	Limite int
	Cursor string
}

type ObtenerPermiso struct {
	Actor appctx.Actor
	ID    int64
}

// ListarUsuarios pagina por cursor (id ascendente) a las personas del equipo.
type ListarUsuarios struct {
	Actor  appctx.Actor
	Limite int
	Cursor string
}

// ListarSucursalesAsignables devuelve las sedes que el actor puede asignar a otros usuarios
// (aquellas donde él tiene USUARIO_ADMINISTRAR).
type ListarSucursalesAsignables struct {
	Actor appctx.Actor
}
