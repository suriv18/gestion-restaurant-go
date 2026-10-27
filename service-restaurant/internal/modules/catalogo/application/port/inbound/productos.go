package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarProductos interface {
	Execute(context.Context, qry.ListarProductos) (appresult.Result[dtoresult.PaginaProductos], error)
}
type ObtenerProducto interface {
	Execute(context.Context, qry.ObtenerProducto) (appresult.Result[dtoresult.Producto], error)
}
type CrearProducto interface {
	Execute(context.Context, cmd.CrearProducto) (appresult.Result[dtoresult.Producto], error)
}
type EditarProducto interface {
	Execute(context.Context, cmd.EditarProducto) (appresult.Result[dtoresult.Producto], error)
}
type CambiarEstadoProducto interface {
	Execute(context.Context, cmd.CambiarEstadoProducto) (appresult.Result[dtoresult.Producto], error)
}
