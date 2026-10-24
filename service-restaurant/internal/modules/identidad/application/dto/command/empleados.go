package command

import appctx "service-restaurant/internal/shared/application/context"

// DatosEmpleado son los datos personales del empleado; los comparte el alta de empleado y el alta
// conjunta de usuario y empleado.
type DatosEmpleado struct {
	TipoDocumento   string
	NumeroDocumento string
	Nombres         string
	Apellidos       string
	Telefono        *string
	Cargo           *string
}

// CrearEmpleado da de alta a un empleado. FechaIngreso (AAAA-MM-DD) nil = hoy. UsuarioID nil =
// sin acceso al sistema.
type CrearEmpleado struct {
	Intencion
	Actor appctx.Actor
	DatosEmpleado
	FechaIngreso *string
	UsuarioID    *int64
}

// EditarEmpleado aplica solo los campos presentes. UsuarioID null desvincula al usuario;
// FechaCese null borra la fecha de cese.
type EditarEmpleado struct {
	Actor           appctx.Actor
	ID              int64
	TipoDocumento   *string
	NumeroDocumento *string
	Nombres         *string
	Apellidos       *string
	FechaIngreso    *string
	Telefono        Campo[string]
	Cargo           Campo[string]
	FechaCese       Campo[string]
	UsuarioID       Campo[int64]
}

// CambiarEstadoEmpleado da de baja (Activo=false, con motivo) o restaura (Activo=true).
type CambiarEstadoEmpleado struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}
