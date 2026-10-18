// Package inbound define los puertos de entrada (casos de uso) del módulo identidad.
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type IniciarSesion interface {
	Execute(context.Context, cmd.IniciarSesion) (appresult.Result[dtoresult.SesionIniciada], error)
}

type CerrarSesion interface {
	Execute(context.Context, cmd.CerrarSesion) (appresult.Result[dtoresult.CierreSesion], error)
}

type InstalarAdministrador interface {
	Execute(context.Context, cmd.InstalarAdministrador) (appresult.Result[dtoresult.AdministradorInstalado], error)
}

type AutenticarToken interface {
	Execute(context.Context, qry.AutenticarToken) (appresult.Result[dtoresult.TokenAutenticado], error)
}

type ObtenerSesion interface {
	Execute(context.Context, qry.ObtenerSesion) (appresult.Result[dtoresult.SesionActual], error)
}
