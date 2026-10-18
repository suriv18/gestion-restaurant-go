package outbound

import (
	"context"
	"errors"
	"time"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
)

// Errores de almacenamiento que los casos de uso traducen a rechazos esperados.
var (
	ErrDuplicado          = errors.New("registro duplicado")
	ErrReferenciaInvalida = errors.New("referencia inexistente")
)

// ErrReglaDatos señala una regla de integridad de la base (triggers de la V4).
type ErrReglaDatos struct{ Mensaje string }

func (e *ErrReglaDatos) Error() string { return "regla de datos: " + e.Mensaje }

// CambiosRol: solo los campos presentes se modifican.
type CambiosRol struct {
	Codigo, Nombre      *string
	Descripcion         *string
	DescripcionPresente bool
}

type CambiosPermiso struct{ Codigo, Descripcion *string }

// RolStore escribe el catálogo de roles. Los métodos que buscan un id devuelven found=false si no existe.
type RolStore interface {
	Crear(ctx context.Context, codigo, nombre string, descripcion *string) (dtoresult.Rol, error)
	Actualizar(ctx context.Context, id int64, c CambiosRol) (dtoresult.Rol, bool, error)
	CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Rol, bool, error)
}

type PermisoStore interface {
	Crear(ctx context.Context, codigo, descripcion string) (dtoresult.Permiso, error)
	Actualizar(ctx context.Context, id int64, c CambiosPermiso) (dtoresult.Permiso, bool, error)
	CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Permiso, bool, error)
}

// AccesosStore administra usuarios y sus asignaciones de roles y sedes.
type AccesosStore interface {
	CrearUsuario(ctx context.Context, username string, email *string, passwordHash string) (dtoresult.UsuarioAdmin, error)
	ObtenerUsuario(ctx context.Context, id int64) (dtoresult.UsuarioAdmin, bool, error)
	SucursalesActivas(ctx context.Context, usuarioID int64) ([]int64, error)
	// ReemplazarAsignaciones deja activos exactamente los roles y sedes indicados:
	// reactiva los existentes, crea los nuevos y desactiva el resto (sin duplicar filas).
	ReemplazarAsignaciones(ctx context.Context, usuarioID int64, rolIDs, sucursalIDs []int64) error
	CambiarActivoUsuario(ctx context.Context, id int64, activo bool, motivo string) (dtoresult.UsuarioAdmin, error)
	RevocarSesionesDeUsuario(ctx context.Context, usuarioID int64, ahora time.Time) error
	// ContarAdministradoresEfectivos cuenta usuarios con USUARIO_ADMINISTRAR efectivo
	// (toda la cadena activa). Se consulta tras aplicar un cambio para no dejar el sistema sin administrador.
	ContarAdministradoresEfectivos(ctx context.Context) (int, error)
}

// CatalogoReader lee roles y permisos paginados por id.
type CatalogoReader interface {
	ListarRoles(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.Rol, error)
	ObtenerRol(ctx context.Context, id int64) (dtoresult.Rol, bool, error)
	ListarPermisos(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.Permiso, error)
	ObtenerPermiso(ctx context.Context, id int64) (dtoresult.Permiso, bool, error)
}

// UsuarioReader lee el equipo con sus asignaciones; nunca devuelve al actor técnico.
type UsuarioReader interface {
	ListarUsuarios(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.UsuarioConAccesos, error)
	// SucursalesActivasPorID devuelve, ordenadas por id, las sucursales activas indicadas.
	SucursalesActivasPorID(ctx context.Context, ids []int64) ([]dtoresult.SucursalResumen, error)
}
