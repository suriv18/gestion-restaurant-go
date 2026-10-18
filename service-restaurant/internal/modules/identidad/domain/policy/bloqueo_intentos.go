// Package policy contiene reglas de decisión del dominio de identidad.
package policy

import (
	"time"

	"service-restaurant/internal/modules/identidad/domain/model"
)

// BloqueoPorIntentos bloquea la cuenta tras demasiados fallos de contraseña.
type BloqueoPorIntentos struct {
	MaxIntentos int
	Duracion    time.Duration
}

// BloqueoPorDefecto: 5 intentos fallidos bloquean 15 minutos.
var BloqueoPorDefecto = BloqueoPorIntentos{MaxIntentos: 5, Duracion: 15 * time.Minute}

// RegistrarFallo suma un fallo. Si el bloqueo anterior ya venció, el contador reinicia.
func (b BloqueoPorIntentos) RegistrarFallo(u *model.Usuario, now time.Time) {
	if u.BloqueadoHasta != nil && !now.Before(*u.BloqueadoHasta) {
		u.IntentosFallidos = 0
		u.BloqueadoHasta = nil
	}
	u.IntentosFallidos++
	if u.IntentosFallidos >= b.MaxIntentos {
		hasta := now.Add(b.Duracion)
		u.BloqueadoHasta = &hasta
	}
}
