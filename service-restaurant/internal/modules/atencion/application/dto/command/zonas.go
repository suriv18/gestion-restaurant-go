package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearZona da de alta una zona en una sede. OrdenVisualizacion nil = 0.
type CrearZona struct {
	Intencion
	Actor              appctx.Actor
	SucursalID         int64
	Nombre             string
	OrdenVisualizacion *int
}

// EditarZona aplica solo los campos presentes. La sede de la zona no se edita.
type EditarZona struct {
	Actor              appctx.Actor
	ID                 int64
	Nombre             *string
	OrdenVisualizacion *int
}

// CambiarEstadoZona desactiva (Activo=false, sin motivo) o restaura (Activo=true).
type CambiarEstadoZona struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
}
