package command

import (
	"strings"
	"testing"
)

func TestValidarOrden(t *testing.T) {
	ent := func(n int) *int { return &n }
	for _, c := range []struct {
		in   *int
		want int
		ok   bool
	}{
		{nil, 0, true}, {ent(0), 0, true}, {ent(9999), 9999, true}, {ent(-1), 0, false}, {ent(10000), 0, false},
	} {
		got, ae := validarOrden(c.in)
		if (ae == nil) != c.ok || got != c.want {
			t.Errorf("validarOrden(%v) = %d, %v", c.in, got, ae)
		}
		if ae != nil && ae.Code != "orden_invalido" {
			t.Errorf("código %q", ae.Code)
		}
	}
}

func TestNombreDeCategoria(t *testing.T) {
	if n, ae := validarNombreHasta("  Bebidas  ", "nombre", maxNombreCategoria); ae != nil || n != "Bebidas" {
		t.Errorf("nombre válido: %q %v", n, ae)
	}
	for _, malo := range []string{"", "   ", "a\nb", strings.Repeat("x", maxNombreCategoria+1)} {
		if _, ae := validarNombreHasta(malo, "nombre", maxNombreCategoria); ae == nil || ae.Code != "nombre_invalido" {
			t.Errorf("%q debía rechazarse: %v", malo, ae)
		}
	}
}
