package vo

import "testing"

func TestNewDocumento(t *testing.T) {
	casos := []struct {
		nombre, tipo, numero string
		okTipo, okNumero     string
		valido               bool
	}{
		{"DNI válido en minúsculas", " dni ", " 12345678 ", "DNI", "12345678", true},
		{"RUC válido", "RUC", "20123456789", "RUC", "20123456789", true},
		{"DNI corto", "DNI", "1234567", "", "", false},
		{"DNI con letras", "DNI", "1234567A", "", "", false},
		{"RUC de 10", "RUC", "2012345678", "", "", false},
		{"carné libre con guion", "ce", "ab-123", "CE", "AB-123", true},
		{"tipo vacío", "", "123", "", "", false},
		{"número vacío", "DNI", "  ", "", "", false},
		{"tipo demasiado largo", "ABCDEFGHIJK", "1", "", "", false},
		{"número demasiado largo", "OTRO", "123456789012345678901", "", "", false},
		{"símbolos", "DNI", "12 345678", "", "", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			tipo, numero, err := NewDocumento(c.tipo, c.numero)
			if (err == nil) != c.valido {
				t.Fatalf("error = %v, válido esperado %v", err, c.valido)
			}
			if c.valido && (tipo != c.okTipo || numero != c.okNumero) {
				t.Fatalf("(%q, %q), quería (%q, %q)", tipo, numero, c.okTipo, c.okNumero)
			}
		})
	}
}

func TestNewEmail(t *testing.T) {
	for _, ok := range []string{"a@b.pe", "ventas.lima@proveedor.com.pe"} {
		if _, err := NewEmail(ok); err != nil {
			t.Errorf("%q debería ser válido: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "sin-arroba", "a@b", "Ana <a@b.pe>", "a@@b.pe", "a b@c.pe"} {
		if _, err := NewEmail(mal); err == nil {
			t.Errorf("%q debería ser inválido", mal)
		}
	}
}

func TestNewTelefono(t *testing.T) {
	for _, ok := range []string{"999888777", "+51 999 888 777", "(01) 555-1234"} {
		if _, err := NewTelefono(ok); err != nil {
			t.Errorf("%q debería ser válido: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "1234", "abc12345", "999;888"} {
		if _, err := NewTelefono(mal); err == nil {
			t.Errorf("%q debería ser inválido", mal)
		}
	}
}
