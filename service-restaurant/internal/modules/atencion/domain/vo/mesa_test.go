package vo

import (
	"errors"
	"strings"
	"testing"
)

func TestNumeroMesa(t *testing.T) {
	if n, err := NewNumeroMesa("  A-12 "); err != nil || n.String() != "A-12" {
		t.Fatalf("número válido: %v %q", err, n.String())
	}
	for _, malo := range []string{"", "   ", strings.Repeat("9", 21), "A\n1", "A\t1"} {
		if _, err := NewNumeroMesa(malo); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("número %q debía rechazarse con ErrValorInvalido: %v", malo, err)
		}
	}
	if _, err := NewNumeroMesa(strings.Repeat("9", 20)); err != nil {
		t.Errorf("20 caracteres es el máximo permitido: %v", err)
	}
}

func TestCapacidadYOrden(t *testing.T) {
	if c, err := NewCapacidad(4); err != nil || c.Int() != 4 {
		t.Fatalf("capacidad válida: %v", err)
	}
	for _, malo := range []int{0, -1, 501} {
		if _, err := NewCapacidad(malo); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("capacidad %d debía rechazarse", malo)
		}
	}
	if _, err := NewCapacidad(500); err != nil {
		t.Errorf("500 es el máximo: %v", err)
	}
	if o, err := NewOrden(0); err != nil || o.Int() != 0 {
		t.Fatalf("orden 0 es válido: %v", err)
	}
	for _, malo := range []int{-1, 10000} {
		if _, err := NewOrden(malo); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("orden %d debía rechazarse", malo)
		}
	}
}

func TestPosicion(t *testing.T) {
	for _, ok := range []string{"0", "12.5", "-3.25", "99999999.99", " 7 "} {
		if p, err := NewPosicion(ok); err != nil || p.String() != strings.TrimSpace(ok) {
			t.Errorf("posición %q: %v", ok, err)
		}
	}
	for _, mala := range []string{"", "abc", "1.234", "123456789", ".5", "1e3", "--1", "1,5"} {
		if _, err := NewPosicion(mala); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("posición %q debía rechazarse", mala)
		}
	}
}
