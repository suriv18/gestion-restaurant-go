package policy

import (
	"time"

	"service-restaurant/internal/modules/identidad/domain/model"
)

// PoliticaSesion fija los plazos de una sesión.
type PoliticaSesion struct {
	Inactividad  time.Duration // sin uso, la sesión vence
	Absoluta     time.Duration // vida máxima aunque se use
	IntervaloUso time.Duration // frecuencia máxima de escritura de ultimo_uso_en
}

// SesionPorDefecto: pensada para un turno de trabajo.
var SesionPorDefecto = PoliticaSesion{Inactividad: 2 * time.Hour, Absoluta: 12 * time.Hour, IntervaloUso: time.Minute}

// Nueva crea una sesión que empieza en now.
func (p PoliticaSesion) Nueva(usuarioID int64, now time.Time) model.Sesion {
	absoluta := now.Add(p.Absoluta)
	return model.Sesion{
		UsuarioID:           usuarioID,
		CreadaEn:            now,
		UltimoUsoEn:         now,
		ExpiraInactividadEn: minTime(now.Add(p.Inactividad), absoluta),
		ExpiraAbsolutaEn:    absoluta,
	}
}

// DebeRegistrarUso evita una escritura por cada petición.
func (p PoliticaSesion) DebeRegistrarUso(s model.Sesion, now time.Time) bool {
	return now.Sub(s.UltimoUsoEn) >= p.IntervaloUso
}

// Extender devuelve la sesión con el uso registrado y el plazo de inactividad renovado
// (sin superar el plazo absoluto).
func (p PoliticaSesion) Extender(s model.Sesion, now time.Time) model.Sesion {
	s.UltimoUsoEn = now
	s.ExpiraInactividadEn = minTime(now.Add(p.Inactividad), s.ExpiraAbsolutaEn)
	return s
}

// ExpiraEnSegundos es el valor de expires_in para una sesión recién creada.
func (p PoliticaSesion) ExpiraEnSegundos(s model.Sesion, now time.Time) int {
	return int(s.ExpiraInactividadEn.Sub(now) / time.Second)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
