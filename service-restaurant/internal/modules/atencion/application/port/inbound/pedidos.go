package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarPedidos interface {
	Execute(context.Context, qry.ListarPedidos) (appresult.Result[dtoresult.PaginaPedidos], error)
}
type ObtenerPedido interface {
	Execute(context.Context, qry.ObtenerPedido) (appresult.Result[dtoresult.PedidoDetalle], error)
}
type AbrirPedido interface {
	Execute(context.Context, cmd.AbrirPedido) (appresult.Result[dtoresult.PedidoDetalle], error)
}
type AgregarDetalle interface {
	Execute(context.Context, cmd.AgregarDetalle) (appresult.Result[dtoresult.Linea], error)
}
type EditarDetalle interface {
	Execute(context.Context, cmd.EditarDetalle) (appresult.Result[dtoresult.Linea], error)
}
