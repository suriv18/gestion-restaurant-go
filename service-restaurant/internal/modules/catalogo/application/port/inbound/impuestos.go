package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarImpuestos interface {
	Execute(context.Context, qry.ListarImpuestos) (appresult.Result[dtoresult.PaginaImpuestos], error)
}
type ObtenerImpuesto interface {
	Execute(context.Context, qry.ObtenerImpuesto) (appresult.Result[dtoresult.Impuesto], error)
}
type ListarTasasImpuesto interface {
	Execute(context.Context, qry.ListarTasasImpuesto) (appresult.Result[dtoresult.PaginaTasas], error)
}
type ObtenerTasaImpuesto interface {
	Execute(context.Context, qry.ObtenerTasaImpuesto) (appresult.Result[dtoresult.TasaImpuesto], error)
}
type CrearTasaImpuesto interface {
	Execute(context.Context, cmd.CrearTasaImpuesto) (appresult.Result[dtoresult.TasaImpuesto], error)
}
type EditarTasaImpuesto interface {
	Execute(context.Context, cmd.EditarTasaImpuesto) (appresult.Result[dtoresult.TasaImpuesto], error)
}
