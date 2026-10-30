// Package inbound declara los puertos de entrada del módulo produccion.
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	qry "service-restaurant/internal/modules/produccion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type EnviarComanda interface {
	Execute(context.Context, cmd.EnviarComanda) (appresult.Result[dtoresult.Comanda], error)
}

type TransicionarLinea interface {
	Execute(context.Context, cmd.TransicionarLinea) (appresult.Result[dtoresult.ComandaLinea], error)
}

type ListarComandas interface {
	Execute(context.Context, qry.ListarComandas) (appresult.Result[dtoresult.PaginaComandas], error)
}
