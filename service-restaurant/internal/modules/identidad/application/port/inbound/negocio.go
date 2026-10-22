package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ObtenerNegocio interface {
	Execute(context.Context, qry.ObtenerNegocio) (appresult.Result[dtoresult.Negocio], error)
}
type EditarNegocio interface {
	Execute(context.Context, cmd.EditarNegocio) (appresult.Result[dtoresult.Negocio], error)
}
