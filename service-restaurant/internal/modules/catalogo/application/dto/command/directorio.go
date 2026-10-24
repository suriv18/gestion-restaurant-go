package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearCliente da de alta un cliente. El documento es opcional, pero tipo y número van juntos.
type CrearCliente struct {
	Intencion
	Actor           appctx.Actor
	TipoDocumento   *string
	NumeroDocumento *string
	Nombre          string
	Telefono        *string
	Email           *string
	Direccion       *string
}

// EditarCliente aplica solo los campos presentes. Tipo y número se envían juntos.
type EditarCliente struct {
	Actor           appctx.Actor
	ID              int64
	Nombre          *string
	TipoDocumento   Campo[string]
	NumeroDocumento Campo[string]
	Telefono        Campo[string]
	Email           Campo[string]
	Direccion       Campo[string]
}

// CambiarEstadoCliente da de baja (Activo=false, con motivo) o restaura (Activo=true).
type CambiarEstadoCliente struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}

// CrearProveedor da de alta un proveedor; el documento es obligatorio.
type CrearProveedor struct {
	Intencion
	Actor           appctx.Actor
	TipoDocumento   string
	NumeroDocumento string
	RazonSocial     string
	ContactoNombre  *string
	Telefono        *string
	Email           *string
	Direccion       *string
}

// EditarProveedor aplica solo los campos presentes.
type EditarProveedor struct {
	Actor           appctx.Actor
	ID              int64
	TipoDocumento   *string
	NumeroDocumento *string
	RazonSocial     *string
	ContactoNombre  Campo[string]
	Telefono        Campo[string]
	Email           Campo[string]
	Direccion       Campo[string]
}

// CambiarEstadoProveedor da de baja (con motivo) o restaura un proveedor.
type CambiarEstadoProveedor struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}
