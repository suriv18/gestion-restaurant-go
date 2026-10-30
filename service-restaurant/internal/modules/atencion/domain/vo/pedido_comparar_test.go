package vo

import "testing"

func TestCantidadMenor(t *testing.T) {
	casos := []struct {
		a, b   string
		quiere bool
	}{
		{"1", "2", true},
		{"2", "1", false},
		{"2", "2.000", false},
		{"1.999", "2", true},
	}
	for _, c := range casos {
		a, err := NewCantidad(c.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := NewCantidad(c.b)
		if err != nil {
			t.Fatal(err)
		}
		if got := a.Menor(b); got != c.quiere {
			t.Errorf("%s.Menor(%s) = %v, quería %v", c.a, c.b, got, c.quiere)
		}
	}
}
