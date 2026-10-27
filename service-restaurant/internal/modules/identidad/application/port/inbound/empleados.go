package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarEmpleados interface {
	Execute(context.Context, qry.ListarEmpleados) (appresult.Result[dtoresult.PaginaEmpleados], error)
}
type ObtenerEmpleado interface {
	Execute(context.Context, qry.ObtenerEmpleado) (appresult.Result[dtoresult.Empleado], error)
}
type CrearEmpleado interface {
	Execute(context.Context, cmd.CrearEmpleado) (appresult.Result[dtoresult.Empleado], error)
}
type EditarEmpleado interface {
	Execute(context.Context, cmd.EditarEmpleado) (appresult.Result[dtoresult.Empleado], error)
}
type CambiarEstadoEmpleado interface {
	Execute(context.Context, cmd.CambiarEstadoEmpleado) (appresult.Result[dtoresult.Empleado], error)
}
