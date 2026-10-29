package policy

import (
	"slices"
	"testing"

	"service-restaurant/internal/modules/produccion/domain/vo"
)

func TestPermiteSoloElSiguientePaso(t *testing.T) {
	estados := []string{EstadoPendiente, EstadoPreparacion, EstadoListo, EstadoEntregado, EstadoCancelado}
	validas := map[[2]string]bool{
		{EstadoPendiente, EstadoPreparacion}: true,
		{EstadoPreparacion, EstadoListo}:     true,
		{EstadoListo, EstadoEntregado}:       true,
	}
	for _, a := range estados {
		for _, d := range estados {
			if got := Permite(a, d); got != validas[[2]string{a, d}] {
				t.Errorf("Permite(%s, %s) = %v", a, d, got)
			}
		}
	}
	if Permite("INEXISTENTE", EstadoListo) {
		t.Error("un estado desconocido no transiciona")
	}
}

func TestEstadoDestinoValido(t *testing.T) {
	for _, s := range []string{EstadoPreparacion, EstadoListo, EstadoEntregado, EstadoCancelado} {
		if !EstadoDestinoValido(s) {
			t.Errorf("%s es un destino válido", s)
		}
	}
	for _, s := range []string{"", EstadoPendiente, "preparacion", "OTRO"} {
		if EstadoDestinoValido(s) {
			t.Errorf("%q no es un destino válido", s)
		}
	}
}

func TestPermisosDelDestino(t *testing.T) {
	if got := PermisosDe(EstadoPreparacion); !slices.Equal(got, []string{CocinaOperar}) {
		t.Errorf("PREPARACION: %v", got)
	}
	if got := PermisosDe(EstadoListo); !slices.Equal(got, []string{CocinaOperar}) {
		t.Errorf("LISTO: %v", got)
	}
	if got := PermisosDe(EstadoEntregado); !slices.Equal(got, []string{CocinaOperar, PedidoEditar}) {
		t.Errorf("ENTREGADO: %v", got)
	}
}

func TestPendiente(t *testing.T) {
	total, _ := vo.NewCantidad("3")
	enviado, _ := vo.CantidadDeAlmacen("1.5")
	if got := Pendiente(total, enviado).String(); got != "1.500" {
		t.Errorf("pendiente = %s", got)
	}
	todo, _ := vo.CantidadDeAlmacen("3.000")
	if got := Pendiente(total, todo).Milis(); got != 0 {
		t.Errorf("sin pendiente = %d", got)
	}
}
