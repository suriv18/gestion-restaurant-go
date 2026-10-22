package mapper

import (
	"encoding/base64"
	"strings"
	"testing"

	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/shared/kernel/apperror"
)

func TestReglaDeReferenciaSeTraduceAUnMensajeParaElUsuario(t *testing.T) {
	regla := func(msg string) error { return &outbound.ErrReglaDatos{Mensaje: msg} }
	casos := []struct {
		nombre   string
		err      error
		traducir func(error) *apperror.AppError
		quiere   string
	}{
		{"zona inactiva al guardar", regla("Referencia zona_id=12: maestro zonas inexistente o dado de baja"), DeAlmacen,
			"la zona indicada no existe o está dada de baja"},
		{"sucursal inactiva al guardar", regla("Referencia sucursal_id=3: maestro sucursales inexistente o dado de baja"), DeAlmacen,
			"la sucursal indicada no existe o está dada de baja"},
		{"restaurar mesa con la zona de baja", regla("Referencia zona_id=12: maestro zonas inexistente o dado de baja"),
			func(e error) *apperror.AppError { return DeRestauracion(e, "la mesa") },
			"no se puede restaurar la mesa porque su zona está dada de baja"},
		{"restaurar zona con la sucursal de baja", regla("Referencia sucursal_id=3: maestro sucursales inexistente o dado de baja"),
			func(e error) *apperror.AppError { return DeRestauracion(e, "la zona") },
			"no se puede restaurar la zona porque su sucursal está dada de baja"},
		{"referencia a un maestro desconocido", regla("Referencia rol_id=4: maestro roles inexistente o dado de baja"), DeAlmacen,
			"una referencia no existe o está dada de baja"},
		{"referencia con formato inesperado", regla("Referencia rara"), DeAlmacen,
			"una referencia no existe o está dada de baja"},
		{"otro mensaje curado pasa tal cual", regla("Cerrar el pedido de la mesa antes de darla de baja"), DeAlmacen,
			"Cerrar el pedido de la mesa antes de darla de baja"},
		{"otro mensaje curado al restaurar", regla("Cerrar el pedido de la mesa antes de darla de baja"),
			func(e error) *apperror.AppError { return DeRestauracion(e, "la mesa") },
			"Cerrar el pedido de la mesa antes de darla de baja"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ae := c.traducir(c.err)
			if ae == nil || ae.Kind != apperror.KindBusinessRule || ae.Code != CodigoReglaDatos {
				t.Fatalf("debía ser regla_de_datos (422): %v", ae)
			}
			if ae.Message != c.quiere {
				t.Fatalf("mensaje %q, quería %q", ae.Message, c.quiere)
			}
			if strings.Contains(ae.Message, "_id=") || strings.Contains(ae.Message, "maestro") {
				t.Fatalf("el mensaje no debe exponer nombres internos: %q", ae.Message)
			}
		})
	}
}

func TestCursorSoloAceptaDecimalCanonico(t *testing.T) {
	codificar := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	for entrada, quiere := range map[string]int64{"0": 0, "5": 5, "42": 42} {
		if id, ae := DecodeCursor(codificar(entrada)); ae != nil || id != quiere {
			t.Errorf("%q: obtuvo %d, %v; quería %d", entrada, id, ae, quiere)
		}
	}
	for _, entrada := range []string{"+5", "007", "05", "-0", "-1", "1.0", " 5", "5 ", "abc", "9223372036854775808"} {
		c := codificar(entrada)
		if _, ae := DecodeCursor(c); ae == nil || ae.Code != CodigoCursorInvalido {
			t.Errorf("%q debía rechazarse como cursor inválido: %v", entrada, ae)
		}
	}
}
