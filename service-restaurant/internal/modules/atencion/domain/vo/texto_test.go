package vo

import (
	"errors"
	"strings"
	"testing"
)

func TestTextoCuentaRunasEnElLimite(t *testing.T) {
	for _, base := range []string{"ñ", "😀"} {
		justo := strings.Repeat(base, 80)
		if v, err := TextoUnaLinea(justo, 80); err != nil || v != justo {
			t.Errorf("80 runas de %q debían aceptarse: %v", base, err)
		}
		if _, err := TextoUnaLinea(justo+base, 80); !errors.Is(err, ErrTextoInvalido) {
			t.Errorf("81 runas de %q debían rechazarse: %v", base, err)
		}
		if _, err := TextoObligatorio(justo+base, 80); !errors.Is(err, ErrTextoInvalido) {
			t.Errorf("81 runas de %q debían rechazarse en multilínea: %v", base, err)
		}
	}
}

func TestTextoSaltosYTabulaciones(t *testing.T) {
	for _, s := range []string{"Salón\nPrincipal", "Salón\tPrincipal"} {
		if _, err := TextoUnaLinea(s, 80); !errors.Is(err, ErrTextoInvalido) {
			t.Errorf("nombre %q debía rechazarse", s)
		}
		if v, err := TextoObligatorio(s, 80); err != nil || v != s {
			t.Errorf("motivo %q debía aceptarse: %v", s, err)
		}
	}
}

func TestTextoRechazaOtrosControles(t *testing.T) {
	for _, s := range []string{"a\x00b", "a\x1bb", "a\x7fb", "a\u0085b", "a\u009fb", "a b", "a b"} {
		if _, err := TextoUnaLinea(s, 80); !errors.Is(err, ErrTextoInvalido) {
			t.Errorf("%q debía rechazarse en una línea", s)
		}
		if _, err := TextoObligatorio(s, 80); !errors.Is(err, ErrTextoInvalido) {
			t.Errorf("%q debía rechazarse en multilínea", s)
		}
	}
	if v, err := TextoUnaLinea("  Terraza ñandú 😀  ", 80); err != nil || v != "Terraza ñandú 😀" {
		t.Errorf("texto válido: %q %v", v, err)
	}
}
