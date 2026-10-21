package inbound

import (
	"context"

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
