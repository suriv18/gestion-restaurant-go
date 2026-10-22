package command

import appctx "service-restaurant/internal/shared/application/context"

// Campo es un valor opcional de un PATCH: Presente distingue "no enviado" de "enviado como
// null" (vaciar). Valor es nil cuando se envió null.
type Campo[T any] struct {
	Presente bool
	Valor    *T
}

// CrearMesa da de alta una mesa. Habilitada nil = true. Las posiciones son decimales exactos.
type CrearMesa struct {
	Intencion
	Actor      appctx.Actor
	SucursalID int64
	ZonaID     int64
	NumeroMesa string
	Capacidad  int
	Habilitada *bool
	PosicionX  *string
	PosicionY  *string
}

// EditarMesa aplica solo los campos presentes. La sede de la mesa no se edita; la zona puede
// cambiar dentro de la misma sede.
type EditarMesa struct {
	Actor      appctx.Actor
	ID         int64
	ZonaID     *int64
	NumeroMesa *string
	Capacidad  *int
	Habilitada *bool
	PosicionX  Campo[string]
	PosicionY  Campo[string]
}

// CambiarEstadoMesa da de baja (Activo=false, exige Motivo) o restaura (Activo=true).
type CambiarEstadoMesa struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}
