package vo

import (
	"strings"
	"testing"
)

func TestNewCodigoEstacion(t *testing.T) {
	validos := map[string]string{" cocina ": "COCINA", "a": "A", "bar-1_x": "BAR-1_X", strings.Repeat("z", 20): strings.Repeat("Z", 20)}
	for in, want := range validos {
		if got, err := NewCodigoEstacion(in); err != nil || got != want {
			t.Errorf("NewCodigoEstacion(%q) = %q, %v; quería %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "   ", "-A", "_A", "A B", "A.B", "ñ", "ı", "ſ", "A\nB", strings.Repeat("Z", 21)} {
		if _, err := NewCodigoEstacion(in); err == nil {
			t.Errorf("NewCodigoEstacion(%q) debía fallar", in)
		}
	}
}
