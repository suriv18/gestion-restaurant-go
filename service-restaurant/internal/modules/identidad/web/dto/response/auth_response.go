// Package response contiene los DTO de salida HTTP del módulo identidad.
// Los IDs viajan como cadenas, según el contrato.
package response

type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type SesionActual struct {
	UsuarioID  string   `json:"usuario_id"`
	Username   string   `json:"username"`
	Sucursales []string `json:"sucursales"`
	Permisos   []string `json:"permisos"`
}
