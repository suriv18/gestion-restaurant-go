package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarClientes pagina por cursor (id ascendente). Activo y Texto filtran cuando no son nil o
// vacío; Texto busca por nombre o por inicio del número de documento.
type ListarClientes struct {
	Actor  appctx.Actor
	Activo *bool
	Texto  string
	Limite int
	Cursor string
}

type ObtenerCliente struct {
	Actor appctx.Actor
	ID    int64
}

// ListarProveedores es el equivalente para proveedores.
type ListarProveedores struct {
	Actor  appctx.Actor
	Activo *bool
	Texto  string
	Limite int
	Cursor string
}

type ObtenerProveedor struct {
	Actor appctx.Actor
	ID    int64
}
