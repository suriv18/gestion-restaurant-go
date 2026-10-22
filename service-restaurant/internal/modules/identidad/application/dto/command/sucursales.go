package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearSucursal da de alta una sede del negocio. negocio_id no se acepta: es la empresa única.
type CrearSucursal struct {
	Intencion
	Actor                      appctx.Actor
	Codigo                     string
	Nombre                     string
	Direccion                  string
	Ubigeo                     *string
	CodigoEstablecimientoSunat *string
	Telefono                   *string
}

// EditarSucursal aplica solo los campos presentes.
type EditarSucursal struct {
	Actor                      appctx.Actor
	ID                         int64
	Codigo                     *string
	Nombre                     *string
	Direccion                  *string
	Ubigeo                     Campo[string]
	CodigoEstablecimientoSunat Campo[string]
	Telefono                   Campo[string]
}

// CambiarEstadoSucursal da de baja (Activo=false, exige Motivo) o restaura (Activo=true).
type CambiarEstadoSucursal struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}
