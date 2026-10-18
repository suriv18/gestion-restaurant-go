// Package outbound define los puertos de salida que la infraestructura implementa.
// Los define el consumidor (los casos de uso), con solo lo que necesita.
package outbound

import (
	"context"
	"time"

	"service-restaurant/internal/modules/identidad/domain/model"
	appctx "service-restaurant/internal/shared/application/context"
)

// UsuarioStore: acceso a usuarios dentro de la unidad de trabajo del caso de uso.
type UsuarioStore interface {
	// BuscarPorUsernameParaActualizar bloquea la fila hasta el fin de la transacción.
	// key es Username.Key() (minúsculas), igual que el índice único.
	BuscarPorUsernameParaActualizar(ctx context.Context, key string) (model.Usuario, bool, error)
	GuardarIntentos(ctx context.Context, u model.Usuario) error
	RegistrarAcceso(ctx context.Context, u model.Usuario, ahora time.Time) error
}

// ActorContext fija quién escribe (app.usuario_id) para la auditoría de la V4.
type ActorContext interface {
	Fijar(ctx context.Context, usuarioID int64) error
	UsuarioSistemaID(ctx context.Context) (int64, error)
}

// SesionWriter persiste el ciclo de vida de las sesiones.
type SesionWriter interface {
	Crear(ctx context.Context, s model.Sesion) (model.Sesion, error)
	Revocar(ctx context.Context, sesionID int64, ahora time.Time) error
	RegistrarUso(ctx context.Context, s model.Sesion) error
}

// SesionAutenticada une la sesión con los datos del usuario que la posee.
type SesionAutenticada struct {
	Sesion   model.Sesion
	Username string
}

// SesionReader consulta sesiones y permisos efectivos.
type SesionReader interface {
	// BuscarPorID devuelve solo sesiones de usuarios activos y no técnicos.
	BuscarPorID(ctx context.Context, id int64) (SesionAutenticada, bool, error)
	PermisosEfectivos(ctx context.Context, usuarioID int64) ([]appctx.Grant, error)
}

// PasswordHasher abstrae el hash de contraseñas (argon2id).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hash string) (bool, error)
	// Dummy gasta el mismo tiempo que Verify cuando no hay hash real que comprobar.
	Dummy(plain string)
}

// TokenClaims son los datos mínimos que viajan en el JWT.
type TokenClaims struct {
	SesionID  int64
	UsuarioID int64
}

// TokenIssuer emite y verifica los JWT de acceso.
type TokenIssuer interface {
	Emitir(c TokenClaims, ahora, expiraEn time.Time) (string, error)
	// Verificar comprueba firma, algoritmo, emisor y vencimiento. Cualquier fallo es ok=false.
	Verificar(token string, ahora time.Time) (TokenClaims, bool)
}

// Clock permite fijar el tiempo en pruebas.
type Clock interface {
	Now() time.Time
}

// DatosInstalacion son los datos del negocio y la sucursal iniciales.
type DatosInstalacion struct {
	RUC, RazonSocial, NombreComercial, DireccionFiscal string
	SucursalCodigo, SucursalNombre, SucursalDireccion  string
}

// InstalacionStore soporta el alta del primer administrador.
type InstalacionStore interface {
	ExisteUsuarioHumano(ctx context.Context) (bool, error)
	// AsegurarSucursal usa el negocio y la sucursal existentes o los crea con los datos.
	AsegurarSucursal(ctx context.Context, d DatosInstalacion) (int64, error)
	// CrearAdministrador crea el usuario y le asigna el rol ADMIN y la sucursal.
	CrearAdministrador(ctx context.Context, username, passwordHash string, sucursalID int64) (int64, error)
}
