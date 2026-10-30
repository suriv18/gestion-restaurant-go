package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// anularLinea anula una línea con motivo y devuelve la respuesta cruda.
func anularLinea(t *testing.T, tk, pedido, detalle, motivo string) resp {
	t.Helper()
	return post(t, "/api/v1/pedidos/"+pedido+"/detalles/"+detalle+"/anulacion", fmt.Sprintf(`{"motivo":%q}`, motivo), tk)
}

// cancelarPedido cancela el pedido con motivo y devuelve la respuesta cruda.
func cancelarPedido(t *testing.T, tk, pedido, motivo string) resp {
	t.Helper()
	return post(t, "/api/v1/pedidos/"+pedido+"/cancelacion", fmt.Sprintf(`{"motivo":%q}`, motivo), tk)
}

// entregarTodo lleva la línea de comanda por PREPARACION, LISTO y ENTREGADO.
func entregarTodo(t *testing.T, cocinero, lineaComanda string) {
	t.Helper()
	for _, estado := range []string{"PREPARACION", "LISTO", "ENTREGADO"} {
		transicionarOK(t, cocinero, lineaComanda, estado)
	}
}

func TestAnularLineaEnviadaCancelaSuComanda(t *testing.T) {
	e := nuevoEscenario(t, "3")
	sup := supervisorPedidos(t)
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	transicionarOK(t, e.cocinero, c.Lineas[0].ID, "PREPARACION")

	r := anularLinea(t, sup, e.pedido.ID, e.linea.ID, "  el cliente se fue  ")
	esperar(t, r, http.StatusOK, "")
	var l lineaResp
	r.json(t, &l)
	if l.Estado != "ANULADO" || l.MotivoAnulacion == nil || *l.MotivoAnulacion != "el cliente se fue" {
		t.Fatalf("línea anulada: %+v", l)
	}

	g := obtenerPedido(t, sup, e.pedido.ID)
	if g.Cuentas[0].Total != "0.00" || len(g.Cuentas[0].Lineas) != 1 || g.Cuentas[0].Lineas[0].Estado != "ANULADO" {
		t.Fatalf("la línea se conserva anulada y no suma al total: %+v", g.Cuentas[0])
	}
	if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, c.Lineas[0].ID); estado != "CANCELADO" {
		t.Fatalf("la línea de comanda debe quedar CANCELADO (%s)", estado)
	}
	if got := colaDe(t, e.cocinero, url.Values{"estacion_id": {e.estacion}}).ids(); len(got) != 0 {
		t.Fatalf("lo cancelado sale de la cola de cocina: %v", got)
	}
}

func TestAnularLineaArrastraSusAdicionales(t *testing.T) {
	sup := supervisorPedidos(t)
	p := abrirLlevar(t, sup, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	plato := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	extra := productoVentaSQL(t, "PRINCIPAL", "ADICIONAL", "2.00", true)
	padre := agregarOK(t, sup, p.ID, cuenta, plato, "1", "")
	hijo := agregarOK(t, sup, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID))
	otro := agregarOK(t, sup, p.ID, cuenta, plato, "1", "")

	esperar(t, anularLinea(t, sup, p.ID, padre.ID, "se equivocó"), http.StatusOK, "")

	g := obtenerPedido(t, sup, p.ID)
	estados := map[string]string{}
	for _, l := range g.Cuentas[0].Lineas {
		estados[l.ID] = l.Estado
	}
	if estados[padre.ID] != "ANULADO" || estados[hijo.ID] != "ANULADO" || estados[otro.ID] != "ACTIVO" || g.Cuentas[0].Total != "10.00" {
		t.Fatalf("el adicional se anula con su padre: %v total %s", estados, g.Cuentas[0].Total)
	}
}

func TestAnularLineaRechazos(t *testing.T) {
	e := nuevoEscenario(t, "2")
	sup := supervisorPedidos(t)
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	esperar(t, anularLinea(t, e.mesero, e.pedido.ID, e.linea.ID, "x"), http.StatusForbidden, "sin_permiso")
	esperar(t, anularLinea(t, sup, e.pedido.ID, e.linea.ID, "   "), http.StatusUnprocessableEntity, "motivo_invalido")
	otro := abrirLlevar(t, sup, "PRINCIPAL")
	esperar(t, anularLinea(t, sup, otro.ID, e.linea.ID, "x"), http.StatusNotFound, "no_encontrado")

	// Una línea ya anulada no se anula dos veces.
	suelta := agregarOK(t, sup, e.pedido.ID, e.cuenta, prod, "1", "")
	esperar(t, anularLinea(t, sup, e.pedido.ID, suelta.ID, "primera"), http.StatusOK, "")
	esperar(t, anularLinea(t, sup, e.pedido.ID, suelta.ID, "segunda"), http.StatusConflict, "linea_anulada")

	// Lo servido no se anula, y el rechazo no cambia nada.
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	entregarTodo(t, e.cocinero, c.Lineas[0].ID)
	esperar(t, anularLinea(t, sup, e.pedido.ID, e.linea.ID, "x"), http.StatusConflict, "linea_entregada")
	if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, e.linea.ID); estado != "ACTIVO" {
		t.Fatalf("la línea entregada debe seguir ACTIVO (%s)", estado)
	}

	// Con un pago confirmado en la cuenta, la anulación se rechaza antes que por la entrega.
	pagoConfirmadoSQL(t, e.pedido.ID, e.cuenta, "1.00")
	esperar(t, anularLinea(t, sup, e.pedido.ID, e.linea.ID, "x"), http.StatusConflict, "cuenta_con_pago")
}

func TestAnulacionRespetaLasSedes(t *testing.T) {
	crearSedeSQL(t, "ANU7")
	pedido, _ := pedidoSQL(t, "ANU7")
	sup := supervisorPedidos(t) // solo PRINCIPAL
	esperar(t, cancelarPedido(t, sup, pedido, "x"), http.StatusNotFound, "no_encontrado")
	esperar(t, anularLinea(t, sup, pedido, "1", "x"), http.StatusNotFound, "no_encontrado")
}

func TestCancelarPedidoCancelaTodo(t *testing.T) {
	e := nuevoEscenario(t, "3")
	sup := supervisorPedidos(t)
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	transicionarOK(t, e.cocinero, c.Lineas[0].ID, "PREPARACION")

	r := cancelarPedido(t, sup, e.pedido.ID, " cliente se retiró ")
	esperar(t, r, http.StatusOK, "")
	var p pedidoResp
	r.json(t, &p)
	if p.Estado != "CANCELADO" || p.MotivoCancelacion == nil || *p.MotivoCancelacion != "cliente se retiró" || p.CerradoEn == nil ||
		len(p.Cuentas) != 1 || p.Cuentas[0].Estado != "CANCELADA" || p.Cuentas[0].Total != "0.00" ||
		len(p.Cuentas[0].Lineas) != 1 || p.Cuentas[0].Lineas[0].Estado != "ANULADO" {
		t.Fatalf("pedido cancelado: %+v", p)
	}
	if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, c.Lineas[0].ID); estado != "CANCELADO" {
		t.Fatalf("la línea de comanda debe quedar CANCELADO (%s)", estado)
	}
	if got := colaDe(t, e.cocinero, url.Values{"estacion_id": {e.estacion}}).ids(); len(got) != 0 {
		t.Fatalf("lo cancelado sale de la cola de cocina: %v", got)
	}
	if g := obtenerPedido(t, sup, e.pedido.ID); g.Estado != "CANCELADO" {
		t.Fatalf("el pedido sigue consultable y cancelado: %+v", g)
	}

	esperar(t, cancelarPedido(t, sup, e.pedido.ID, "otra vez"), http.StatusConflict, "pedido_cerrado")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	esperar(t, agregarConsumo(t, sup, e.pedido.ID, e.cuenta, prod, "1", ""), http.StatusConflict, "pedido_cerrado")
}

func TestCancelarPedidoLiberaLaMesa(t *testing.T) {
	sup := supervisorPedidos(t)
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	abrir := func() resp {
		return post(t, "/api/v1/pedidos",
			fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2}`, sedeID(t, "PRINCIPAL"), mesa), sup)
	}
	r := abrir()
	esperar(t, r, http.StatusCreated, "")
	var p pedidoResp
	r.json(t, &p)
	esperar(t, abrir(), http.StatusConflict, "")

	esperar(t, cancelarPedido(t, sup, p.ID, "el cliente no llegó"), http.StatusOK, "")
	esperar(t, abrir(), http.StatusCreated, "")
}

func TestCancelarPedidoRechazos(t *testing.T) {
	sup := supervisorPedidos(t)
	e := nuevoEscenario(t, "2")
	esperar(t, cancelarPedido(t, e.mesero, e.pedido.ID, "x"), http.StatusForbidden, "sin_permiso")
	esperar(t, cancelarPedido(t, sup, e.pedido.ID, ""), http.StatusUnprocessableEntity, "motivo_invalido")

	// Algo ya servido: no se cancela y no cambia nada.
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	entregarTodo(t, e.cocinero, c.Lineas[0].ID)
	esperar(t, cancelarPedido(t, sup, e.pedido.ID, "x"), http.StatusConflict, "linea_entregada")
	if p := obtenerPedido(t, sup, e.pedido.ID); p.Estado == "CANCELADO" || p.Cuentas[0].Estado != "ABIERTA" || p.Cuentas[0].Lineas[0].Estado != "ACTIVO" {
		t.Fatalf("un rechazo no cambia nada: %+v", p)
	}

	// El consumo cobrado no se devuelve automáticamente.
	f := nuevoEscenario(t, "2")
	pagoConfirmadoSQL(t, f.pedido.ID, f.cuenta, "1.00")
	esperar(t, cancelarPedido(t, sup, f.pedido.ID, "x"), http.StatusConflict, "pedido_con_pago")
}

func TestDividirCuentaRechazaLoAnuladoYLoCancelado(t *testing.T) {
	sup := supervisorPedidos(t)
	p := abrirLlevar(t, sup, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	l := agregarOK(t, sup, p.ID, p.Cuentas[0].ID, prod, "1", "")

	esperar(t, anularLinea(t, sup, p.ID, l.ID, "x"), http.StatusOK, "")
	esperar(t, dividirCuenta(t, sup, p.ID, []string{l.ID}, ""), http.StatusConflict, "linea_anulada")

	esperar(t, cancelarPedido(t, sup, p.ID, "x"), http.StatusOK, "")
	esperar(t, dividirCuenta(t, sup, p.ID, []string{l.ID}, ""), http.StatusConflict, "pedido_cerrado")
}

func TestAnularYCancelarSonIdempotentes(t *testing.T) {
	sup := supervisorPedidos(t)
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	p := abrirLlevar(t, sup, "PRINCIPAL")
	l := agregarOK(t, sup, p.ID, p.Cuentas[0].ID, prod, "1", "")
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles/" + l.ID + "/anulacion"
	h := map[string]string{"Idempotency-Key": nuevaClave()}
	esperar(t, callH(t, "POST", ruta, `{"motivo":"x"}`, sup, h), http.StatusOK, "")
	esperar(t, callH(t, "POST", ruta, `{"motivo":"x"}`, sup, h), http.StatusOK, "")
	esperar(t, callH(t, "POST", ruta, `{"motivo":"otro"}`, sup, h), http.StatusConflict, "idempotency_key_reutilizada")

	q := abrirLlevar(t, sup, "PRINCIPAL")
	rutaC := "/api/v1/pedidos/" + q.ID + "/cancelacion"
	hc := map[string]string{"Idempotency-Key": nuevaClave()}
	esperar(t, callH(t, "POST", rutaC, `{"motivo":"x"}`, sup, hc), http.StatusOK, "")
	esperar(t, callH(t, "POST", rutaC, `{"motivo":"x"}`, sup, hc), http.StatusOK, "")
}
