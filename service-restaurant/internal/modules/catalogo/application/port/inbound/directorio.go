package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarClientes interface {
	Execute(context.Context, qry.ListarClientes) (appresult.Result[dtoresult.PaginaClientes], error)
}
type ObtenerCliente interface {
	Execute(context.Context, qry.ObtenerCliente) (appresult.Result[dtoresult.Cliente], error)
}
type CrearCliente interface {
	Execute(context.Context, cmd.CrearCliente) (appresult.Result[dtoresult.Cliente], error)
}
type EditarCliente interface {
	Execute(context.Context, cmd.EditarCliente) (appresult.Result[dtoresult.Cliente], error)
}
type CambiarEstadoCliente interface {
	Execute(context.Context, cmd.CambiarEstadoCliente) (appresult.Result[dtoresult.Cliente], error)
}

type ListarProveedores interface {
	Execute(context.Context, qry.ListarProveedores) (appresult.Result[dtoresult.PaginaProveedores], error)
}
type ObtenerProveedor interface {
	Execute(context.Context, qry.ObtenerProveedor) (appresult.Result[dtoresult.Proveedor], error)
}
type CrearProveedor interface {
	Execute(context.Context, cmd.CrearProveedor) (appresult.Result[dtoresult.Proveedor], error)
}
type EditarProveedor interface {
	Execute(context.Context, cmd.EditarProveedor) (appresult.Result[dtoresult.Proveedor], error)
}
type CambiarEstadoProveedor interface {
	Execute(context.Context, cmd.CambiarEstadoProveedor) (appresult.Result[dtoresult.Proveedor], error)
}
