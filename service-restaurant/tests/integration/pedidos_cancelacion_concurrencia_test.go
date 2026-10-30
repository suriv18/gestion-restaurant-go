package integration

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
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

// carreraEntregaContra prepara un pedido con una línea LISTO en cocina y lanza a la vez la
// transición a ENTREGADO y la sentencia (ruta y cuerpo) que quiere anular o cancelar. Devuelve el
// escenario, la línea de comanda y los dos códigos HTTP.
func carreraEntregaContra(t *testing.T, sup string, ruta func(e escenarioCocina) string) (e escenarioCocina, lc string, codTrans, codAnul int, cuerpoAnul string) {
	t.Helper()
	e = nuevoEscenario(t, "1")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	lc = c.Lineas[0].ID
	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	transicionarOK(t, e.cocinero, lc, "LISTO")

	var inicio, fin sync.WaitGroup
	inicio.Add(1)
	fin.Add(2)
	go func() {
		defer fin.Done()
		inicio.Wait()
		codTrans = callH(t, "POST", "/api/v1/comanda-detalles/"+lc+"/transiciones", `{"estado":"ENTREGADO"}`, e.cocinero, nil).Code
	}()
	go func() {
		defer fin.Done()
		inicio.Wait()
		r := callH(t, "POST", ruta(e), `{"motivo":"carrera"}`, sup, map[string]string{"Idempotency-Key": nuevaClave()})
		codAnul = r.Code
		cuerpoAnul = r.Body.String()
	}()
	inicio.Done()
	fin.Wait()
	return
}

// Cancelar el pedido y entregar el plato a la vez: si la cancelación gana (200) la línea de cocina
// queda CANCELADA; si pierde (409 linea_entregada) el pedido sigue vivo con el plato ENTREGADO. Nunca
// un pedido cancelado con un plato servido.
func TestCancelarContraEntregaSimultaneaNoAnulaUnPlatoServido(t *testing.T) {
	sup := supervisorPedidos(t)
	app(t)
	for i := 0; i < 5; i++ {
		e, lc, codTrans, codCancel, cuerpo := carreraEntregaContra(t, sup, func(e escenarioCocina) string {
			return "/api/v1/pedidos/" + e.pedido.ID + "/cancelacion"
		})
		if codTrans != http.StatusOK && codTrans != http.StatusConflict {
			t.Fatalf("vuelta %d: la transición da 200 o 409, no %d", i, codTrans)
		}
		estadoCocina := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, lc)
		estadoPedido := scalarStr(t, `SELECT estado FROM sch_restaurante.pedidos WHERE id=$1::text::bigint`, e.pedido.ID)
		estadoLinea := scalarStr(t, `SELECT estado FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, e.linea.ID)
		switch codCancel {
		case http.StatusOK:
			if estadoCocina != "CANCELADO" || estadoPedido != "CANCELADO" {
				t.Fatalf("vuelta %d: cancelación 200 (transición %d) pero cocina=%s pedido=%s", i, codTrans, estadoCocina, estadoPedido)
			}
		case http.StatusConflict:
			if estadoCocina != "ENTREGADO" || estadoPedido == "CANCELADO" || estadoLinea != "ACTIVO" {
				t.Fatalf("vuelta %d: cancelación 409 pero cocina=%s pedido=%s línea=%s (%s)", i, estadoCocina, estadoPedido, estadoLinea, cuerpo)
			}
		default:
			t.Fatalf("vuelta %d: la cancelación da 200 o 409, no %d (%s)", i, codCancel, cuerpo)
		}
	}
}

// Lo mismo para anular una sola línea.
func TestAnularLineaContraEntregaSimultaneaNoAnulaUnPlatoServido(t *testing.T) {
	sup := supervisorPedidos(t)
	app(t)
	for i := 0; i < 5; i++ {
		e, lc, codTrans, codAnul, cuerpo := carreraEntregaContra(t, sup, func(e escenarioCocina) string {
			return "/api/v1/pedidos/" + e.pedido.ID + "/detalles/" + e.linea.ID + "/anulacion"
		})
		if codTrans != http.StatusOK && codTrans != http.StatusConflict {
			t.Fatalf("vuelta %d: la transición da 200 o 409, no %d", i, codTrans)
		}
		estadoCocina := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, lc)
		estadoLinea := scalarStr(t, `SELECT estado FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, e.linea.ID)
		switch codAnul {
		case http.StatusOK:
			if estadoCocina != "CANCELADO" || estadoLinea != "ANULADO" {
				t.Fatalf("vuelta %d: anulación 200 (transición %d) pero cocina=%s línea=%s", i, codTrans, estadoCocina, estadoLinea)
			}
		case http.StatusConflict:
			if estadoCocina != "ENTREGADO" || estadoLinea != "ACTIVO" {
				t.Fatalf("vuelta %d: anulación 409 pero cocina=%s línea=%s (%s)", i, estadoCocina, estadoLinea, cuerpo)
			}
		default:
			t.Fatalf("vuelta %d: la anulación da 200 o 409, no %d (%s)", i, codAnul, cuerpo)
		}
	}
}

// entregaEnVueloContra deja la línea de comanda LISTO entregándose en una transacción de cocina aún
// abierta (fila bloqueada), lanza la anulación o cancelación, espera a que quede bloqueada y recién
// entonces confirma la entrega. Determinista: reproduce el orden verificar-luego-bloquear.
func entregaEnVueloContra(t *testing.T, sup string, ruta func(e escenarioCocina) string) (e escenarioCocina, lc string, r resp) {
	t.Helper()
	app(t)
	e = nuevoEscenario(t, "1")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	lc = c.Lineas[0].ID
	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	transicionarOK(t, e.cocinero, lc, "LISTO")

	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.usuario_id', $1::text, true)`, strconv.FormatInt(adminID(t), 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sch_restaurante.comanda_detalles
		SET estado='ENTREGADO', entregado_en=clock_timestamp() WHERE id=$1::text::bigint`, lc); err != nil {
		t.Fatal(err)
	}
	hecho := make(chan resp, 1)
	go func() {
		hecho <- callH(t, "POST", ruta(e), `{"motivo":"carrera"}`, sup, map[string]string{"Idempotency-Key": nuevaClave()})
	}()
	for i := 0; i < 100; i++ { // hasta ~5 s esperando que la petición quede bloqueada en un lock
		if scalar(t, `SELECT count(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND datname=current_database()`) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return e, lc, <-hecho
}

func TestCancelarPedidoConEntregaEnVueloEsConflicto(t *testing.T) {
	sup := supervisorPedidos(t)
	e, lc, r := entregaEnVueloContra(t, sup, func(e escenarioCocina) string {
		return "/api/v1/pedidos/" + e.pedido.ID + "/cancelacion"
	})
	esperar(t, r, http.StatusConflict, "linea_entregada")
	if c := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, lc); c != "ENTREGADO" {
		t.Fatalf("cocina = %s", c)
	}
	if p := scalarStr(t, `SELECT estado FROM sch_restaurante.pedidos WHERE id=$1::text::bigint`, e.pedido.ID); p == "CANCELADO" {
		t.Fatal("el pedido no puede quedar cancelado con un plato servido")
	}
}

func TestAnularLineaConEntregaEnVueloEsConflicto(t *testing.T) {
	sup := supervisorPedidos(t)
	e, lc, r := entregaEnVueloContra(t, sup, func(e escenarioCocina) string {
		return "/api/v1/pedidos/" + e.pedido.ID + "/detalles/" + e.linea.ID + "/anulacion"
	})
	esperar(t, r, http.StatusConflict, "linea_entregada")
	if c := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, lc); c != "ENTREGADO" {
		t.Fatalf("cocina = %s", c)
	}
	if l := scalarStr(t, `SELECT estado FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, e.linea.ID); l != "ACTIVO" {
		t.Fatalf("la línea servida no puede anularse (%s)", l)
	}
}
