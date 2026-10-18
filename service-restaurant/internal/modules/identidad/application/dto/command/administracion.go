package command

import (
	appctx "service-restaurant/internal/shared/application/context"
)

// Intencion identifica una operación para la idempotencia (ADR-04). La rellena la capa
// web con la clave del encabezado, la operación y la huella del contenido; los casos de uso la ignoran.
type Intencion struct {
	Clave     string
	Operacion string // p. ej. "POST /roles/7/baja" (con el id concreto)
	Huella    [32]byte
}

// --- Roles

type CrearRol struct {
	Intencion
	Actor       appctx.Actor
	Codigo      string
	Nombre      string
	Descripcion *string
}

// EditarRol aplica solo los campos presentes. DescripcionPresente distingue "no enviada"
// de "enviada como null" (vaciar).
type EditarRol struct {
	Actor               appctx.Actor
	ID                  int64
	Codigo              *string
	Nombre              *string
	Descripcion         *string
	DescripcionPresente bool
}

// CambiarEstadoRol desactiva (Activo=false) o restaura (Activo=true).
type CambiarEstadoRol struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
}

// --- Permisos

type CrearPermiso struct {
	Intencion
	Actor       appctx.Actor
	Codigo      string
	Descripcion string
}

type EditarPermiso struct {
	Actor       appctx.Actor
	ID          int64
	Codigo      *string
	Descripcion *string
}

type CambiarEstadoPermiso struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
}

// --- Usuarios

// CrearUsuario recibe la contraseña en texto solo para derivar su hash; nunca se guarda ni se registra.
type CrearUsuario struct {
	Intencion
	Actor    appctx.Actor
	Username string
	Password string
	Email    *string
}

// AsignarAccesos reemplaza el conjunto de roles y sedes del usuario.
type AsignarAccesos struct {
	Actor       appctx.Actor
	UsuarioID   int64
	RolIDs      []int64
	SucursalIDs []int64
}

// CambiarEstadoUsuario da de baja (Activo=false, exige Motivo) o restaura (Activo=true).
type CambiarEstadoUsuario struct {
	Intencion
	Actor     appctx.Actor
	UsuarioID int64
	Activo    bool
	Motivo    string
}
