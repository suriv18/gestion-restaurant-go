package policy

import (
	"errors"
	"slices"
	"testing"
)

func idp(n int64) *int64 { return &n }

func lineasDePrueba() []LineaPedido {
	return []LineaPedido{
		{ID: 1, CuentaID: 20, Activa: true},
		{ID: 2, CuentaID: 20, Activa: true},
		{ID: 3, CuentaID: 20, PadreID: idp(1), Activa: true},  // adicional activo de la 1
		{ID: 4, CuentaID: 20, PadreID: idp(1), Activa: false}, // adicional anulado de la 1
		{ID: 5, CuentaID: 21, Activa: false},                  // línea anulada
	}
}

func TestSeleccionDeDivision(t *testing.T) {
	casos := []struct {
		nombre  string
		pedidas []int64
		quiere  []int64
	}{
		{"una línea simple", []int64{2}, []int64{2}},
		{"el padre arrastra a sus adicionales activos", []int64{1}, []int64{1, 3}},
		{"padre e hijo explícitos no se duplican", []int64{3, 1}, []int64{1, 3}},
		{"varias líneas salen ordenadas", []int64{2, 1}, []int64{1, 2, 3}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, err := SeleccionDeDivision(c.pedidas, lineasDePrueba())
			if err != nil || !slices.Equal(got, c.quiere) {
				t.Fatalf("selección %v (%v), quería %v", got, err, c.quiere)
			}
		})
	}
}

func TestSeleccionDeDivisionRechaza(t *testing.T) {
	casos := []struct {
		nombre  string
		pedidas []int64
		quiere  error
	}{
		{"vacía", nil, ErrSeleccionVacia},
		{"línea repetida", []int64{2, 2}, ErrLineaRepetida},
		{"línea que no es del pedido", []int64{99}, ErrLineaInexistente},
		{"línea anulada", []int64{5}, ErrLineaAnulada},
		{"adicional sin su padre", []int64{3}, ErrAdicionalSinPadre},
		{"adicional anulado", []int64{4, 1}, ErrLineaAnulada},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if _, err := SeleccionDeDivision(c.pedidas, lineasDePrueba()); !errors.Is(err, c.quiere) {
				t.Fatalf("error %v, quería %v", err, c.quiere)
			}
		})
	}
}
