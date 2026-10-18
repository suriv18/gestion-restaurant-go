// Package command contiene los DTO de comandos del módulo identidad.
package command

// IniciarSesion: credenciales enviadas por el cliente.
type IniciarSesion struct {
	Username string
	Password string
}

// CerrarSesion revoca la sesión indicada (la del actor autenticado).
type CerrarSesion struct {
	SesionID int64
}

// InstalarAdministrador crea el primer administrador y, si faltan, el negocio y la sucursal.
// Los datos del negocio solo se usan cuando todavía no existe uno.
type InstalarAdministrador struct {
	Username string
	Password string

	RUC             string
	RazonSocial     string
	NombreComercial string
	DireccionFiscal string

	SucursalCodigo    string
	SucursalNombre    string
	SucursalDireccion string
}
