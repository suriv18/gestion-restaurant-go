package vo

import "testing"

func TestUbigeo(t *testing.T) {
	if u, err := NewUbigeo(" 150101 "); err != nil || u.String() != "150101" {
		t.Fatalf("ubigeo válido: %v %q", err, u.String())
	}
	for _, malo := range []string{"", "15010", "1501011", "15a101", "150 101"} {
		if _, err := NewUbigeo(malo); err == nil {
			t.Errorf("ubigeo %q debía rechazarse", malo)
		}
	}
}

func TestMoneda(t *testing.T) {
	if m, err := NewMoneda(" pen "); err != nil || m.String() != "PEN" {
		t.Fatalf("moneda válida: %v %q", err, m.String())
	}
	for _, mala := range []string{"", "PE", "PENN", "P3N", "12$"} {
		if _, err := NewMoneda(mala); err == nil {
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
	for _, mala := range []string{"", "Local", "Marte/Olimpo", "../etc/passwd"} {
		if _, err := NewZonaHoraria(mala); err == nil {
			t.Errorf("zona %q debía rechazarse", mala)
		}
	}
}

func TestCodigoSucursal(t *testing.T) {
	if c, err := NewCodigoSucursal(" san-isidro_2 "); err != nil || c.String() != "SAN-ISIDRO_2" {
		t.Fatalf("código válido: %v %q", err, c.String())
	}
	for _, malo := range []string{"", "A", "-AB", "AB CD", "ÁB", "ABCDEFGHIJKLMNOPQRSTU"} {
		if _, err := NewCodigoSucursal(malo); err == nil {
			t.Errorf("código %q debía rechazarse", malo)
		}
	}
}
