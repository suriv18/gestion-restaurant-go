package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarEstaciones interface {
	Execute(context.Context, qry.ListarEstaciones) (appresult.Result[dtoresult.PaginaEstaciones], error)
}
type ObtenerEstacion interface {
	Execute(context.Context, qry.ObtenerEstacion) (appresult.Result[dtoresult.Estacion], error)
}
type CrearEstacion interface {
	Execute(context.Context, cmd.CrearEstacion) (appresult.Result[dtoresult.Estacion], error)
}
type EditarEstacion interface {
	Execute(context.Context, cmd.EditarEstacion) (appresult.Result[dtoresult.Estacion], error)
}
type CambiarEstadoEstacion interface {
	Execute(context.Context, cmd.CambiarEstadoEstacion) (appresult.Result[dtoresult.Estacion], error)
}
