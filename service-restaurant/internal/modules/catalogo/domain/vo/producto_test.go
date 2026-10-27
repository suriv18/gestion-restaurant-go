package vo

import (
	"strings"
	"testing"
)

func TestNewPrecioVenta(t *testing.T) {
	ok := map[string]string{
		"0": "0.00", "0.00": "0.00", "10": "10.00", "10.5": "10.50", "10.55": "10.55", "007": "7.00",
		"99999999999.99": "99999999999.99", "99999999999": "99999999999.00",
	}
	for in, want := range ok {
		if got, err := NewPrecioVenta(in); err != nil || got != want {
			t.Errorf("NewPrecioVenta(%q) = %q, %v; quiero %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", " 5", "5 ", "-0", "-1", "+1", "1e3", "0.001", "10.555", ".5", "5.", "1,5", "100000000000.00",
		"100000000000", "abc", "1.2.3", "٣", "0x10", "NaN", "Infinity"} {
		if got, err := NewPrecioVenta(in); err == nil {
			t.Errorf("NewPrecioVenta(%q) = %q; debía fallar", in, got)
		}
	}
}

func TestNewFactorConsumo(t *testing.T) {
	ok := map[string]string{
		"1": "1.000000", "0.5": "0.500000", "0.000001": "0.000001", "2.123456": "2.123456",
		"999999999999.999999": "999999999999.999999",
	}
	for in, want := range ok {
		if got, err := NewFactorConsumo(in); err != nil || got != want {
			t.Errorf("NewFactorConsumo(%q) = %q, %v; quiero %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0", "0.0", "0.000000", "0.0000001", "-1", " 1", "1e3", "1000000000000", "1.2345678"} {
		if got, err := NewFactorConsumo(in); err == nil {
			t.Errorf("NewFactorConsumo(%q) = %q; debía fallar", in, got)
		}
	}
}

func TestNewCodigoProducto(t *testing.T) {
	ok := map[string]string{"ab-1": "AB-1", "  x ": "X", "A.b_c-9": "A.B_C-9", strings.Repeat("a", 40): strings.Repeat("A", 40)}
	for in, want := range ok {
		if got, err := NewCodigoProducto(in); err != nil || got != want {
			t.Errorf("NewCodigoProducto(%q) = %q, %v; quiero %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "   ", "A B", "A\nB", "Ñ1", "ı1", "A/B", "A*", strings.Repeat("A", 41)} {
		if got, err := NewCodigoProducto(in); err == nil {
			t.Errorf("NewCodigoProducto(%q) = %q; debía fallar", in, got)
		}
	}
}

func TestTipoYAfectacion(t *testing.T) {
	for _, v := range []string{"PREPARADO", "REVENTA", "ADICIONAL", "SERVICIO"} {
		if _, err := NewTipoProducto(v); err != nil {
			t.Errorf("tipo %q: %v", v, err)
		}
	}
	for _, v := range []string{"", "preparado", "OTRO", " REVENTA"} {
		if _, err := NewTipoProducto(v); err == nil {
			t.Errorf("tipo %q debía fallar", v)
		}
	}
	for _, v := range []string{"10", "20", "30"} {
		if _, err := NewAfectacionIGV(v); err != nil {
			t.Errorf("afectación %q: %v", v, err)
		}
	}
	for _, v := range []string{"", "1", "40", "010", "10 "} {
		if _, err := NewAfectacionIGV(v); err == nil {
			t.Errorf("afectación %q debía fallar", v)
		}
	}
}

func TestNewImagenURL(t *testing.T) {
	for _, v := range []string{"http://x.com/a.png", " https://cdn.example.com/a/b.jpg?x=1 "} {
		if _, err := NewImagenURL(v); err != nil {
			t.Errorf("url %q: %v", v, err)
		}
	}
	largo := "https://x.com/" + strings.Repeat("a", 1986) // 2000 exactos
	if _, err := NewImagenURL(largo); err != nil {
		t.Errorf("url de 2000: %v", err)
	}
	for _, v := range []string{"", "ftp://x.com/a", "javascript:alert(1)", "//x.com/a", "/a.png", "https://", "https:///a", "https://x .com",
		"https://x.com/a\nb", "data:image/png;base64,AAA", "https://x.com/" + strings.Repeat("a", 1987)} {
		if _, err := NewImagenURL(v); err == nil {
			t.Errorf("url %q debía fallar", v)
		}
	}
}
