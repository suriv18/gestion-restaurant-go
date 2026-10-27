package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type CambiarDisponibilidad interface {
	Execute(context.Context, cmd.CambiarDisponibilidad) (appresult.Result[dtoresult.ProductoSucursal], error)
}
