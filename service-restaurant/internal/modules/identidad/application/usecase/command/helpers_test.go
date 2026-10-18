package command

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestPositivosUnicos(t *testing.T) {
	got, ok := positivosUnicos([]int64{3, 1, 3, 2, 1})
	if !ok || !reflect.DeepEqual(got, []int64{3, 1, 2}) {
		t.Fatalf("got=%v ok=%v", got, ok)
	}
	for _, malo := range [][]int64{{0}, {-1, 2}, {1, 0}} {
		if _, ok := positivosUnicos(malo); ok {
			t.Errorf("%v debía rechazarse", malo)
		}
	}
	if got, ok := positivosUnicos(nil); !ok || len(got) != 0 {
		t.Errorf("vacío: %v %v", got, ok)
	}
}

func TestDiferenciaSimetrica(t *testing.T) {
	got := diferenciaSimetrica([]int64{1, 2, 3}, []int64{2, 3, 4})
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	if !reflect.DeepEqual(got, []int64{1, 4}) {
		t.Fatalf("got=%v", got)
	}
	if got := diferenciaSimetrica([]int64{1, 2}, []int64{2, 1}); len(got) != 0 {
		t.Fatalf("conjuntos iguales: %v", got)
	}
}

func TestTextoOpcional(t *testing.T) {
	s := "  hola "
	if v, ok := textoOpcional(&s, 10); !ok || v == nil || *v != "hola" {
		t.Fatalf("v=%v ok=%v", v, ok)
	}
	vacio := "   "
	if v, ok := textoOpcional(&vacio, 10); !ok || v != nil {
		t.Fatalf("vacío debe ser nil: v=%v ok=%v", v, ok)
	}
	if v, ok := textoOpcional(nil, 10); !ok || v != nil {
		t.Fatal("nil debe ser nil")
	}
	largo := strings.Repeat("x", 11)
	if _, ok := textoOpcional(&largo, 10); ok {
		t.Fatal("demasiado largo debe rechazarse")
	}
}

func TestEmailOpcional(t *testing.T) {
	ok := "ana@example.com"
	if v, valido := emailOpcional(&ok); !valido || v == nil || *v != ok {
		t.Fatalf("v=%v valido=%v", v, valido)
	}
	for _, malo := range []string{"sin-arroba", "Ana <ana@example.com>", "a b@example.com", "@example.com"} {
		m := malo
		if _, valido := emailOpcional(&m); valido {
			t.Errorf("%q debía rechazarse", malo)
		}
	}
	if v, valido := emailOpcional(nil); !valido || v != nil {
		t.Fatal("ausente es válido")
	}
}
