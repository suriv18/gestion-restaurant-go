// Package request contiene los DTO de entrada HTTP del módulo identidad.
package request

// Login es el cuerpo de POST /auth/login.
type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
