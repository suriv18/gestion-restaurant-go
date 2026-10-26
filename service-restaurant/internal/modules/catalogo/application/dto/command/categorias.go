package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearCategoria da de alta una categoría de la carta. Orden nil equivale a 0.
type CrearCategoria struct {
	Intencion
	Actor       appctx.Actor
	Nombre      string
	Descripcion *string
	Orden       *int
}

// EditarCategoria aplica solo los campos presentes; la descripción admite null para vaciarla.
type EditarCategoria struct {
	Actor       appctx.Actor
	ID          int64
	Nombre      *string
	Descripcion Campo[string]
	Orden       *int
}

// CambiarEstadoCategoria da de baja (Activo=false, sin motivo) o restaura una categoría.
type CambiarEstadoCategoria struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
}
