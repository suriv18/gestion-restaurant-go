package model

import "time"

// Sesion es una sesión de acceso. El JWT que la representa la identifica por su ID (claim sid).
type Sesion struct {
	ID                  int64
	UsuarioID           int64
	CreadaEn            time.Time
	UltimoUsoEn         time.Time
	ExpiraInactividadEn time.Time
	ExpiraAbsolutaEn    time.Time
	RevocadaEn          *time.Time
}

// Vigente: no revocada y dentro de ambos plazos.
func (s Sesion) Vigente(now time.Time) bool {
	return s.RevocadaEn == nil && now.Before(s.ExpiraInactividadEn) && now.Before(s.ExpiraAbsolutaEn)
}
