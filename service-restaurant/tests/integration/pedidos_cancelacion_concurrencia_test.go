package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// Una cancelación y un envío a cocina simultáneos sobre el mismo pedido: ambos bloquean el pedido
// FOR UPDATE primero, así que el resultado es siempre coherente. O el envío ocurrió antes y la
// cancelación canceló su comanda, o la cancelación ocurrió antes y el envío se rechazó con 409.
func TestCancelarYEnviarSimultaneosDejanUnEstadoCoherente(t *testing.T) {
	sup := supervisorPedidos(t)
	app(t) // inicializa la aplicación antes de lanzar goroutines

	for i := 0; i < 5; i++ {
		e := nuevoEscenario(t, "3")
		cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"2"}]}`, e.estacion, e.linea.ID)
		rutaEnvio := "/api/v1/pedidos/" + e.pedido.ID + "/comandas"
		rutaCancel := "/api/v1/pedidos/" + e.pedido.ID + "/cancelacion"

		var codEnvio, codCancel int
		var inicio, fin sync.WaitGroup
		inicio.Add(1)
		fin.Add(2)
		go func() {
			defer fin.Done()
			inicio.Wait()
			codEnvio = callH(t, "POST", rutaEnvio, cuerpo, e.mesero, map[string]string{"Idempotency-Key": nuevaClave()}).Code
		}()
		go func() {
			defer fin.Done()
			inicio.Wait()
			codCancel = callH(t, "POST", rutaCancel, `{"motivo":"carrera"}`, sup, map[string]string{"Idempotency-Key": nuevaClave()}).Code
		}()
		inicio.Done()
		fin.Wait()

		if codCancel != http.StatusOK {
			t.Fatalf("vuelta %d: la cancelación debe tener éxito (%d)", i, codCancel)
		}
		if codEnvio != http.StatusCreated && codEnvio != http.StatusConflict {
			t.Fatalf("vuelta %d: el envío da 201 o 409, no %d", i, codEnvio)
		}
		if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.pedidos WHERE id=$1::text::bigint`, e.pedido.ID); estado != "CANCELADO" {
			t.Fatalf("vuelta %d: el pedido debe quedar CANCELADO (%s)", i, estado)
		}
		if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comanda_detalles
			WHERE pedido_id=$1::text::bigint AND estado <> 'CANCELADO'`, e.pedido.ID); n != 0 {
			t.Fatalf("vuelta %d: no puede quedar cocina viva en un pedido cancelado (%d líneas)", i, n)
		}
		enviadas := scalar(t, `SELECT count(*) FROM sch_restaurante.comanda_detalles WHERE pedido_id=$1::text::bigint`, e.pedido.ID)
		if (codEnvio == http.StatusCreated) != (enviadas == 1) {
			t.Fatalf("vuelta %d: envío %d pero %d líneas de comanda", i, codEnvio, enviadas)
		}
	}
}
