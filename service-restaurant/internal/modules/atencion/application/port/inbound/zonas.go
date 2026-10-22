// Package inbound declara los casos de uso que ofrece el módulo atencion.
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarZonas interface {
	Execute(context.Context, qry.ListarZonas) (appresult.Result[dtoresult.PaginaZonas], error)
}
type ObtenerZona interface {
	Execute(context.Context, qry.ObtenerZona) (appresult.Result[dtoresult.Zona], error)
}
type CrearZona interface {
	Execute(context.Context, cmd.CrearZona) (appresult.Result[dtoresult.Zona], error)
}
type EditarZona interface {
	Execute(context.Context, cmd.EditarZona) (appresult.Result[dtoresult.Zona], error)
}
type CambiarEstadoZona interface {
	Execute(context.Context, cmd.CambiarEstadoZona) (appresult.Result[dtoresult.Zona], error)
}
