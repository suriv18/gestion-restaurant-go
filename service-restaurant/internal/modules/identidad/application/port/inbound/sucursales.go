package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarSucursales interface {
	Execute(context.Context, qry.ListarSucursales) (appresult.Result[dtoresult.PaginaSucursales], error)
}
type ObtenerSucursal interface {
	Execute(context.Context, qry.ObtenerSucursal) (appresult.Result[dtoresult.Sucursal], error)
}

type CrearSucursal interface {
	Execute(context.Context, cmd.CrearSucursal) (appresult.Result[dtoresult.Sucursal], error)
}
type EditarSucursal interface {
	Execute(context.Context, cmd.EditarSucursal) (appresult.Result[dtoresult.Sucursal], error)
}
type CambiarEstadoSucursal interface {
	Execute(context.Context, cmd.CambiarEstadoSucursal) (appresult.Result[dtoresult.Sucursal], error)
}
