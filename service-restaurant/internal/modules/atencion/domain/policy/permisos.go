// Package policy contiene las reglas de permiso del módulo atencion.
package policy

// Permisos de zonas y mesas (migración 00008). Se declaran aquí porque atencion no importa
// identidad; el catálogo de permisos vive en la base.
const (
	MesaVer         = "MESA_VER"
	MesaAdministrar = "MESA_ADMINISTRAR"
)
