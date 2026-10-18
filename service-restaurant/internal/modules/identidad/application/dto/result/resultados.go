// Package result contiene los DTO de resultado del módulo identidad.
package result

import appctx "service-restaurant/internal/shared/application/context"

// SesionIniciada es la respuesta del login.
type SesionIniciada struct {
	AccessToken string
	ExpiresIn   int // segundos hasta el vencimiento por inactividad
}

// SesionActual es la vista del actor autenticado.
type SesionActual struct {
	UsuarioID  int64
	Username   string
	Sucursales []int64
	Permisos   []string
}

// TokenAutenticado es el actor reconstruido a partir de un token válido.
type TokenAutenticado struct {
	Actor appctx.Actor
}

// CierreSesion confirma que la sesión quedó revocada.
type CierreSesion struct{}

// AdministradorInstalado confirma la instalación inicial.
type AdministradorInstalado struct {
	UsuarioID  int64
	SucursalID int64
}
