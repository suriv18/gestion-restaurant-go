package model

import (
	"testing"
	"time"
)

func TestUsuarioPuedeAutenticar(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	future, past := now.Add(time.Minute), now.Add(-time.Minute)
	cases := []struct {
		name string
		u    Usuario
		want bool
	}{
		{"activo", Usuario{EsActivo: true}, true},
		{"inactivo", Usuario{EsActivo: false}, false},
		{"sistema", Usuario{EsActivo: true, EsSistema: true}, false},
		{"bloqueado", Usuario{EsActivo: true, BloqueadoHasta: &future}, false},
		{"bloqueo vencido", Usuario{EsActivo: true, BloqueadoHasta: &past}, true},
		{"bloqueo termina ahora", Usuario{EsActivo: true, BloqueadoHasta: &now}, true},
	}
	for _, c := range cases {
		if got := c.u.PuedeAutenticar(now); got != c.want {
			t.Errorf("%s: %v, quería %v", c.name, got, c.want)
		}
	}
}

func TestSesionVigente(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	s := Sesion{ExpiraInactividadEn: now.Add(time.Hour), ExpiraAbsolutaEn: now.Add(2 * time.Hour)}
	if !s.Vigente(now) {
		t.Fatal("debe estar vigente")
	}
	if s.Vigente(now.Add(time.Hour)) {
		t.Fatal("vencida por inactividad")
	}
	s.ExpiraInactividadEn = now.Add(3 * time.Hour)
	if s.Vigente(now.Add(2 * time.Hour)) {
		t.Fatal("vencida por plazo absoluto")
	}
	r := now
	s.RevocadaEn = &r
	if s.Vigente(now) {
		t.Fatal("revocada")
	}
}
