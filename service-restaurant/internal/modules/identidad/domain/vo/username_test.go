package vo

import (
	"strings"
	"testing"
)

func TestUsername(t *testing.T) {
	u, err := NewUsername("  Ana.Perez ")
	if err != nil || u.String() != "Ana.Perez" || u.Key() != "ana.perez" {
		t.Fatalf("u=%v err=%v", u, err)
	}
	for _, bad := range []string{"", "   ", "a\x00b", "x\ny", strings.Repeat("a", MaxUsername+1)} {
		if _, err := NewUsername(bad); err == nil {
			t.Errorf("%q debía rechazarse", bad)
		}
	}
	if _, err := NewUsername(strings.Repeat("a", MaxUsername)); err != nil {
		t.Errorf("el máximo debe aceptarse: %v", err)
	}
}
