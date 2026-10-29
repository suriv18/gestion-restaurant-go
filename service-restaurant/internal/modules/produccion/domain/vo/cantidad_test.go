package vo

import "testing"

func TestNewCantidad(t *testing.T) {
	validos := map[string]int64{"1": 1000, "2.5": 2500, " 0.001 ": 1, "10.250": 10250, "00003": 3000, "99999999999.999": 99999999999999}
	for raw, milis := range validos {
		c, err := NewCantidad(raw)
		if err != nil || c.Milis() != milis {
			t.Errorf("NewCantidad(%q) = %d, %v; quería %d", raw, c.Milis(), err, milis)
		}
	}
	for _, raw := range []string{"", "0", "0.000", "-1", "1.2345", "abc", "1e3", "1,5", ".5", "5.", "123456789012"} {
		if _, err := NewCantidad(raw); err == nil {
			t.Errorf("NewCantidad(%q) debía fallar", raw)
		}
	}
}

func TestCantidadDeAlmacenAceptaCero(t *testing.T) {
	for _, raw := range []string{"0", "0.000", "2.000"} {
		if _, err := CantidadDeAlmacen(raw); err != nil {
			t.Errorf("CantidadDeAlmacen(%q): %v", raw, err)
		}
	}
	if _, err := CantidadDeAlmacen("-1"); err == nil {
		t.Error("una cantidad negativa debía fallar")
	}
}

func TestCantidadAritmeticaYTexto(t *testing.T) {
	a, _ := NewCantidad("3")
	b, _ := NewCantidad("1.25")
	if got := a.Menos(b).String(); got != "1.750" {
		t.Errorf("3 - 1.25 = %s", got)
	}
	if got := b.Menos(a).Milis(); got != 0 {
		t.Errorf("la resta no baja de cero: %d", got)
	}
	if !a.Excede(b) || b.Excede(a) || a.Excede(a) {
		t.Error("Excede mal definido")
	}
	if got := (Cantidad{}).String(); got != "0.000" {
		t.Errorf("cero = %s", got)
	}
}
