package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearEstacion da de alta una estación de preparación en una sede.
type CrearEstacion struct {
	Intencion
	Actor      appctx.Actor
	SucursalID int64
	Codigo     string
	Nombre     string
}

// EditarEstacion aplica solo los campos presentes. La sede no se edita.
type EditarEstacion struct {
	Actor  appctx.Actor
	ID     int64
	Codigo *string
	Nombre *string
}

// CambiarEstadoEstacion da de baja (Activo=false, sin motivo) o restaura una estación.
type CambiarEstadoEstacion struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
}
