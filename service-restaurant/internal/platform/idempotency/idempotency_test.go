package idempotency

import "testing"

func TestParseKey(t *testing.T) {
	k, err := ParseKey(" 3F2504E0-4F89-41D3-9A0C-0305E82C3301 ")
	if err != nil || k != "3f2504e0-4f89-41d3-9a0c-0305e82c3301" {
		t.Fatalf("k=%q err=%v", k, err)
	}
	for _, bad := range []string{"", "   ", "no-es-uuid", "3f2504e0-4f89-41d3-9a0c-0305e82c330", "3f2504e0-4f89-41d3-9a0c-0305e82c3301x", "'; drop table x;--"} {
		if _, err := ParseKey(bad); err == nil {
			t.Errorf("%q debía rechazarse", bad)
		}
	}
}

func TestFingerprint(t *testing.T) {
	a := Fingerprint("POST /usuarios", []byte(`{"a":1}`))
	if a != Fingerprint("POST /usuarios", []byte(`{"a":1}`)) {
		t.Fatal("misma entrada, misma huella")
	}
	if a == Fingerprint("POST /usuarios", []byte(`{"a":2}`)) || a == Fingerprint("POST /roles", []byte(`{"a":1}`)) {
		t.Fatal("cuerpo u operación distintos deben cambiar la huella")
	}
	// El separador evita colisiones entre (operación, cuerpo) desplazados.
	if Fingerprint("ab", []byte("c")) == Fingerprint("a", []byte("bc")) {
		t.Fatal("colisión por concatenación")
	}
}
