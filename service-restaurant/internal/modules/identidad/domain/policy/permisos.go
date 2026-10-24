package policy

// UsuarioAdministrar es el permiso funcional que protege la administración de accesos.
// Es un permiso de la aplicación (semilla de la V4), no un privilegio de PostgreSQL.
const UsuarioAdministrar = "USUARIO_ADMINISTRAR"

// Permisos del personal operativo (migración 00009).
const (
	EmpleadoVer         = "EMPLEADO_VER"
	EmpleadoAdministrar = "EMPLEADO_ADMINISTRAR"
)

// Permisos de configuración (migración 00008). Los de mesa los define el módulo atencion.
const (
	NegocioVer          = "NEGOCIO_VER"
	NegocioAdministrar  = "NEGOCIO_ADMINISTRAR"
	SucursalVer         = "SUCURSAL_VER"
	SucursalAdministrar = "SUCURSAL_ADMINISTRAR"
)
