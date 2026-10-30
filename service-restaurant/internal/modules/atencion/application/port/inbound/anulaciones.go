package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type AnularDetalle interface {
	Execute(context.Context, cmd.AnularDetalle) (appresult.Result[dtoresult.Linea], error)
}
type CancelarPedido interface {
	Execute(context.Context, cmd.CancelarPedido) (appresult.Result[dtoresult.PedidoDetalle], error)
}
