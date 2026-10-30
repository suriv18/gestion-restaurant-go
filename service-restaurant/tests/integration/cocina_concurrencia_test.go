package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// Dos envíos simultáneos de la misma línea no pueden superar su cantidad: el pedido se bloquea
// FOR UPDATE, así que el segundo envío ve lo que dejó el primero.
func TestDosEnviosSimultaneosNoSuperanLaCantidad(t *testing.T) {
	e := nuevoEscenario(t, "3")
	cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"2"}]}`, e.estacion, e.linea.ID)
	ruta := "/api/v1/pedidos/" + e.pedido.ID + "/comandas"
	app(t) // inicializa la aplicación antes de lanzar goroutines

	const n = 6
	codigos := make([]int, n)
	var inicio, fin sync.WaitGroup
	inicio.Add(1)
	for i := 0; i < n; i++ {
		fin.Add(1)
		go func(i int) {
			defer fin.Done()
			inicio.Wait()
			r := callH(t, "POST", ruta, cuerpo, e.mesero, map[string]string{"Idempotency-Key": nuevaClave()})
			codigos[i] = r.Code
		}(i)
	}
	inicio.Done()
	fin.Wait()

	creadas, rechazadas := 0, 0
	for _, c := range codigos {
		switch c {
		case http.StatusCreated:
			creadas++
		case http.StatusConflict:
			rechazadas++
		default:
			t.Fatalf("código inesperado %d en %v", c, codigos)
		}
	}
	if creadas != 1 || rechazadas != n-1 {
		t.Fatalf("de %d envíos de 2 sobre una línea de 3 debía crearse exactamente 1: %v", n, codigos)
	}
	if enviado := scalar(t, `SELECT sum(cantidad)::int FROM sch_restaurante.comanda_detalles WHERE detalle_pedido_id=$1::text::bigint`, e.linea.ID); enviado != 2 {
		t.Fatalf("lo enviado debe ser 2 (hay %d)", enviado)
	}
}
