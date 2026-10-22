package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarMesas interface {
	Execute(context.Context, qry.ListarMesas) (appresult.Result[dtoresult.PaginaMesas], error)
}
type ObtenerMesa interface {
	Execute(context.Context, qry.ObtenerMesa) (appresult.Result[dtoresult.Mesa], error)
}
type CrearMesa interface {
	Execute(context.Context, cmd.CrearMesa) (appresult.Result[dtoresult.Mesa], error)
}
type EditarMesa interface {
	Execute(context.Context, cmd.EditarMesa) (appresult.Result[dtoresult.Mesa], error)
}
type CambiarEstadoMesa interface {
	Execute(context.Context, cmd.CambiarEstadoMesa) (appresult.Result[dtoresult.Mesa], error)
}
