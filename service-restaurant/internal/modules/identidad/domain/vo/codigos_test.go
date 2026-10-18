package vo

import (
	"strings"
	"testing"
)

func TestCodigoRol(t *testing.T) {
	c, err := NewCodigoRol(" jefe_cocina ")
	if err != nil || c.String() != "JEFE_COCINA" {
		t.Fatalf("c=%v err=%v", c, err)
	}
	for _, bad := range []string{"", "A", "1ABC", "con espacio", "ñandú", "A-B", strings.Repeat("A", 41)} {
		if _, err := NewCodigoRol(bad); err == nil {
			t.Errorf("%q debía rechazarse", bad)
		}
	}
	if _, err := NewCodigoRol(strings.Repeat("A", 40)); err != nil {
		t.Errorf("40 caracteres es válido: %v", err)
	}
}

func TestCodigoPermiso(t *testing.T) {
	if c, err := NewCodigoPermiso("pedido_crear"); err != nil || c.String() != "PEDIDO_CREAR" {
		t.Fatalf("c=%v err=%v", c, err)
	}
	if _, err := NewCodigoPermiso(strings.Repeat("A", 81)); err == nil {
		t.Error("81 caracteres debe rechazarse")
	}
	if _, err := NewCodigoPermiso("a b"); err == nil {
		t.Error("espacios deben rechazarse")
	}
}

func TestTextoObligatorio(t *testing.T) {
	if v, err := TextoObligatorio("  Hola  ", 10); err != nil || v != "Hola" {
		t.Fatalf("v=%q err=%v", v, err)
	}
	for _, bad := range []string{"", "   ", "a\x00b", strings.Repeat("x", 11)} {
		if _, err := TextoObligatorio(bad, 10); err == nil {
			t.Errorf("%q debía rechazarse", bad)
		}
	}
}
