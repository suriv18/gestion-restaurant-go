package result

import "time"

// Rol y Permiso son catálogos de la aplicación (no roles de PostgreSQL).
type Rol struct {
	ID             int64
	Codigo         string
	Nombre         string
	Descripcion    *string
	EsActivo       bool
	CreadoEn       time.Time
	CreadoPor      int64
	ActualizadoEn  time.Time
	ActualizadoPor int64
}

type Permiso struct {
	ID             int64
	Codigo         string
	Descripcion    string
	EsActivo       bool
	CreadoEn       time.Time
	CreadoPor      int64
	ActualizadoEn  time.Time
	ActualizadoPor int64
}

// UsuarioAdmin es la vista administrativa de un usuario: nunca incluye el hash.
type UsuarioAdmin struct {
	ID               int64
	Username         string
	Email            *string
	EsSistema        bool
	EsActivo         bool
	UltimoAccesoEn   *time.Time
	BloqueadoHasta   *time.Time
	IntentosFallidos int
	CreadoEn         time.Time
	CreadoPor        int64
	ActualizadoEn    time.Time
	ActualizadoPor   int64
}

// PaginaRoles y PaginaPermisos: NextCursor vacío indica la última página.
type PaginaRoles struct {
	Items      []Rol
	NextCursor string
}

type PaginaPermisos struct {
	Items      []Permiso
	NextCursor string
}
