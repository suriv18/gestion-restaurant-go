package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type DividirCuenta interface {
	Execute(context.Context, cmd.DividirCuenta) (appresult.Result[dtoresult.Cuenta], error)
}
