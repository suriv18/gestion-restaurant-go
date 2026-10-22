package vo

import (
	"errors"
	"strings"
	"testing"
)

func TestUbigeo(t *testing.T) {
	if u, err := NewUbigeo(" 150101 "); err != nil || u.String() != "150101" {
		t.Fatalf("ubigeo válido: %v %q", err, u.String())
	}
	for _, malo := range []string{"", "15010", "1501011", "15a101", "150 101"} {
		if _, err := NewUbigeo(malo); !errors.Is(err, ErrFormatoInvalido) {
			t.Errorf("ubigeo %q debía rechazarse", malo)
		}
	}
}

func TestMoneda(t *testing.T) {
	if m, err := NewMoneda(" pen "); err != nil || m.String() != "PEN" {
		t.Fatalf("moneda válida: %v %q", err, m.String())
	}
	for _, mala := range []string{"", "PE", "PENN", "P3N", "12$", "pıt", "ſol"} {
		if _, err := NewMoneda(mala); !errors.Is(err, ErrFormatoInvalido) {
			t.Errorf("moneda %q debía rechazarse", mala)
		}
	}
}

func TestZonaHoraria(t *testing.T) {
	for _, ok := range []string{"America/Lima", "UTC", "Europe/Madrid"} {
		if z, err := NewZonaHoraria(ok); err != nil || z.String() != ok {
			t.Errorf("zona %q: %v", ok, err)
		}
	}
	for _, mala := range []string{"", "Local", "local", "Marte/Olimpo", "../etc/passwd"} {
		if _, err := NewZonaHoraria(mala); !errors.Is(err, ErrFormatoInvalido) {
			t.Errorf("zona %q debía rechazarse", mala)
		}
	}
}

func TestCodigoSucursal(t *testing.T) {
	if c, err := NewCodigoSucursal(" san-isidro_2 "); err != nil || c.String() != "SAN-ISIDRO_2" {
		t.Fatalf("código válido: %v %q", err, c.String())
	}
	for _, malo := range []string{"", "A", "-AB", "AB CD", "ÁB", "pıt", "ſol", strings.Repeat("A", 21)} {
		if _, err := NewCodigoSucursal(malo); !errors.Is(err, ErrCodigoInvalido) {
			t.Errorf("código %q debía rechazarse", malo)
		}
	}
}

func TestCodigoSucursalLongitudMaxima(t *testing.T) {
	if _, err := NewCodigoSucursal(strings.Repeat("a", 20)); err != nil {
		t.Fatalf("20 caracteres debía aceptarse: %v", err)
	}
	if _, err := NewCodigoSucursal(strings.Repeat("a", 21)); !errors.Is(err, ErrCodigoInvalido) {
		t.Fatalf("21 caracteres debía rechazarse: %v", err)
	}
}
