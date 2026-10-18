package policy

import (
	"strings"
	"testing"
	"time"

	"service-restaurant/internal/modules/identidad/domain/model"
)

func TestBloqueoPorIntentos(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	u := &model.Usuario{EsActivo: true}
	for i := 1; i < BloqueoPorDefecto.MaxIntentos; i++ {
		BloqueoPorDefecto.RegistrarFallo(u, now)
		if u.BloqueadoHasta != nil {
			t.Fatalf("no debe bloquear en el intento %d", i)
		}
	}
	BloqueoPorDefecto.RegistrarFallo(u, now)
	if u.BloqueadoHasta == nil || !u.BloqueadoHasta.Equal(now.Add(15*time.Minute)) {
		t.Fatalf("debe bloquear 15 min tras 5 fallos: %v", u.BloqueadoHasta)
	}
	// Vencido el bloqueo, el contador reinicia.
	later := now.Add(16 * time.Minute)
	BloqueoPorDefecto.RegistrarFallo(u, later)
	if u.IntentosFallidos != 1 || u.BloqueadoHasta != nil {
		t.Fatalf("reinicio incorrecto: %d %v", u.IntentosFallidos, u.BloqueadoHasta)
	}
	u.RegistrarAcceso()
	if u.IntentosFallidos != 0 {
		t.Fatal("RegistrarAcceso debe limpiar")
	}
}

func TestPoliticaSesion(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	p := SesionPorDefecto
	s := p.Nueva(9, now)
	if !s.ExpiraInactividadEn.Equal(now.Add(2*time.Hour)) || !s.ExpiraAbsolutaEn.Equal(now.Add(12*time.Hour)) {
		t.Fatalf("plazos incorrectos: %+v", s)
	}
	if p.ExpiraEnSegundos(s, now) != 7200 {
		t.Fatal("expires_in debe ser 7200")
	}
	if p.DebeRegistrarUso(s, now.Add(30*time.Second)) || !p.DebeRegistrarUso(s, now.Add(time.Minute)) {
		t.Fatal("el uso se registra como máximo una vez por minuto")
	}
	// La inactividad nunca supera el plazo absoluto.
	e := p.Extender(s, now.Add(11*time.Hour+30*time.Minute))
	if !e.ExpiraInactividadEn.Equal(s.ExpiraAbsolutaEn) {
		t.Fatalf("debe limitarse al plazo absoluto: %v", e.ExpiraInactividadEn)
	}
}

func TestValidarClaveNueva(t *testing.T) {
	ok := []string{"Correcta12345", "una frase larga 2026"}
	bad := []string{"", "corta1", "soloLetrasSinNumeros", "123456789012", strings.Repeat("a1", 70)}
	for _, c := range ok {
		if err := ValidarClaveNueva(c); err != nil {
			t.Errorf("%q: %v", c, err)
		}
	}
	for _, c := range bad {
		if ValidarClaveNueva(c) == nil {
			t.Errorf("%q debía rechazarse", c)
		}
	}
	if !ClaveAdmisibleParaLogin("x") || ClaveAdmisibleParaLogin("") || ClaveAdmisibleParaLogin(strings.Repeat("a", 129)) {
		t.Fatal("ClaveAdmisibleParaLogin incorrecta")
	}
}
