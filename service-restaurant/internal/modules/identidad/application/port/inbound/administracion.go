package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type CrearRol interface {
	Execute(context.Context, cmd.CrearRol) (appresult.Result[dtoresult.Rol], error)
}
type EditarRol interface {
	Execute(context.Context, cmd.EditarRol) (appresult.Result[dtoresult.Rol], error)
}
type CambiarEstadoRol interface {
	Execute(context.Context, cmd.CambiarEstadoRol) (appresult.Result[dtoresult.Rol], error)
}
type ListarRoles interface {
	Execute(context.Context, qry.ListarRoles) (appresult.Result[dtoresult.PaginaRoles], error)
}
type ObtenerRol interface {
	Execute(context.Context, qry.ObtenerRol) (appresult.Result[dtoresult.Rol], error)
}

type CrearPermiso interface {
	Execute(context.Context, cmd.CrearPermiso) (appresult.Result[dtoresult.Permiso], error)
}
type EditarPermiso interface {
	Execute(context.Context, cmd.EditarPermiso) (appresult.Result[dtoresult.Permiso], error)
}
type CambiarEstadoPermiso interface {
	Execute(context.Context, cmd.CambiarEstadoPermiso) (appresult.Result[dtoresult.Permiso], error)
}
type ListarPermisos interface {
	Execute(context.Context, qry.ListarPermisos) (appresult.Result[dtoresult.PaginaPermisos], error)
}
type ObtenerPermiso interface {
	Execute(context.Context, qry.ObtenerPermiso) (appresult.Result[dtoresult.Permiso], error)
}

type CrearUsuario interface {
	Execute(context.Context, cmd.CrearUsuario) (appresult.Result[dtoresult.UsuarioAdmin], error)
}
type AsignarAccesos interface {
	Execute(context.Context, cmd.AsignarAccesos) (appresult.Result[dtoresult.UsuarioAdmin], error)
}
type CambiarEstadoUsuario interface {
	Execute(context.Context, cmd.CambiarEstadoUsuario) (appresult.Result[dtoresult.UsuarioAdmin], error)
}

type ListarUsuarios interface {
	Execute(context.Context, qry.ListarUsuarios) (appresult.Result[dtoresult.PaginaUsuarios], error)
}

type ListarSucursalesAsignables interface {
	Execute(context.Context, qry.ListarSucursalesAsignables) (appresult.Result[[]dtoresult.SucursalResumen], error)
}
