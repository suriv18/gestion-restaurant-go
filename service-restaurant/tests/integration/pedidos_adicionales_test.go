package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// Los adicionales tienen un solo nivel: un adicional puede colgar de una línea principal, no de otro
// adicional (dividir y anular solo arrastran un nivel).
func TestUnAdicionalNoPuedeColgarDeOtroAdicional(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	plato := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "20.00", true)
	extra := productoVentaSQL(t, "PRINCIPAL", "ADICIONAL", "3.00", true)

	padre := agregarOK(t, tk, p.ID, cuenta, plato, "1", "")
	hijo := agregarOK(t, tk, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID))

	r := agregarConsumo(t, tk, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, hijo.ID))
	esperar(t, r, http.StatusUnprocessableEntity, "padre_invalido")
}
