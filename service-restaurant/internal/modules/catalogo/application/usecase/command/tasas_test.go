package command

import "testing"

func TestValidarTasa(t *testing.T) {
	for entrada, esperado := range map[string]string{
		"0": "0.0000", "18": "18.0000", "10.5": "10.5000", "100": "100.0000", "100.0000": "100.0000",
		"17.9999": "17.9999", "0.0001": "0.0001", "007.5": "7.5000",
	} {
		if got, ae := validarTasa(entrada); ae != nil || got != esperado {
			t.Errorf("validarTasa(%q) = %q, %v; quería %q", entrada, got, ae, esperado)
		}
	}
	for _, mala := range []string{"", "-1", "100.0001", "101", "1000", "1.12345", "abc", " 18", "1,5", ".5", "18.", "1e2", "+5", "0x10"} {
		if _, ae := validarTasa(mala); ae == nil || ae.Code != "tasa_invalida" {
			t.Errorf("validarTasa(%q) debía rechazarse con tasa_invalida: %v", mala, ae)
		}
	}
}

func TestVigencia(t *testing.T) {
	hasta := func(s string) *string { return &s }
	if d, h, ae := vigencia("2030-01-31", hasta("2030-02-01")); ae != nil || d != "2030-01-31" || *h != "2030-02-01" {
		t.Errorf("vigencia válida: %q %v %v", d, h, ae)
	}
	if _, h, ae := vigencia("2030-01-31", nil); ae != nil || h != nil {
		t.Errorf("vigencia abierta: %v %v", h, ae)
	}
	for _, c := range []struct {
		desde string
		hasta *string
		code  string
	}{
		{"2030-02-30", nil, "vigente_desde_invalido"},
		{"2030-1-1", nil, "vigente_desde_invalido"},
		{"0000-01-01", nil, "vigente_desde_invalido"},
		{"2030-01-01", hasta("x"), "vigente_hasta_invalido"},
		{"2030-01-01", hasta("2030-01-01"), "vigencia_invalida"},
		{"2030-01-02", hasta("2030-01-01"), "vigencia_invalida"},
	} {
		if _, _, ae := vigencia(c.desde, c.hasta); ae == nil || ae.Code != c.code {
			t.Errorf("vigencia(%q, %v): quería %s, obtuve %v", c.desde, c.hasta, c.code, ae)
		}
	}
}
