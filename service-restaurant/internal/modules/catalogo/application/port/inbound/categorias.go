package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarCategorias interface {
	Execute(context.Context, qry.ListarCategorias) (appresult.Result[dtoresult.PaginaCategorias], error)
}
type ObtenerCategoria interface {
	Execute(context.Context, qry.ObtenerCategoria) (appresult.Result[dtoresult.Categoria], error)
}
type CrearCategoria interface {
	Execute(context.Context, cmd.CrearCategoria) (appresult.Result[dtoresult.Categoria], error)
}
type EditarCategoria interface {
	Execute(context.Context, cmd.EditarCategoria) (appresult.Result[dtoresult.Categoria], error)
}
type CambiarEstadoCategoria interface {
	Execute(context.Context, cmd.CambiarEstadoCategoria) (appresult.Result[dtoresult.Categoria], error)
}
