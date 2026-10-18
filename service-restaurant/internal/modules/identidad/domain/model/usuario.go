// Package model contiene los agregados y entidades del dominio de identidad.
package model

import "time"

// Usuario es la cuenta de acceso. PasswordHash es un hash PHC, nunca texto plano.
type Usuario struct {
	ID               int64
	Username         string
	PasswordHash     string
	EsSistema        bool
	EsActivo         bool
	IntentosFallidos int
	BloqueadoHasta   *time.Time
}

// PuedeAutenticar: activo, humano (no el actor técnico) y sin bloqueo vigente.
func (u Usuario) PuedeAutenticar(now time.Time) bool {
	if !u.EsActivo || u.EsSistema {
		return false
	}
	return u.BloqueadoHasta == nil || !now.Before(*u.BloqueadoHasta)
}

// RegistrarAcceso limpia el contador de fallos tras un acceso correcto.
func (u *Usuario) RegistrarAcceso() {
	u.IntentosFallidos = 0
	u.BloqueadoHasta = nil
}
