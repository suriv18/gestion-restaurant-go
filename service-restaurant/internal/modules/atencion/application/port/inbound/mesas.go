package inbound

import (
	"context"

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
