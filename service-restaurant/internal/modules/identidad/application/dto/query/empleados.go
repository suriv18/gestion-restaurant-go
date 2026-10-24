package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarEmpleados pagina por cursor (id ascendente). Activo y Texto filtran cuando no son nil o
// vacío; Texto busca por nombres, apellidos o inicio del número de documento.
type ListarEmpleados struct {
	Actor  appctx.Actor
	Activo *bool
	Texto  string
	Limite int
	Cursor string
}

type ObtenerEmpleado struct {
	Actor appctx.Actor
	ID    int64
}
