package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// RES-CP-011: la suma de las cuentas antes y después de dividir coincide.
func TestDividirCuentaConservaLaSuma(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	agregarOK(t, tk, p.ID, cuenta, prod, "1", "")
	l2 := agregarOK(t, tk, p.ID, cuenta, prod, "2", "")
	agregarOK(t, tk, p.ID, cuenta, prod, "3", "")
	antes := sumaImportes(t, p.ID)
	if antes != "60.00" {
		t.Fatalf("suma inicial %s", antes)
	}

	r := dividirCuenta(t, tk, p.ID, []string{l2.ID}, `,"nombre_cuenta":" Juan "`)
	esperar(t, r, http.StatusOK, "")
	var nueva cuentaResp
	r.json(t, &nueva)
	if nueva.Numero != 2 || nueva.Etiqueta == nil || *nueva.Etiqueta != "Juan" || nueva.Estado != "ABIERTA" ||
		len(nueva.Lineas) != 1 || nueva.Lineas[0].ID != l2.ID || nueva.Total != "20.00" {
		t.Fatalf("cuenta nueva: %+v", nueva)
	}

	g := obtenerPedido(t, tk, p.ID)
	if len(g.Cuentas) != 2 || g.Cuentas[0].Total != "40.00" || g.Cuentas[1].Total != "20.00" || len(g.Cuentas[0].Lineas) != 2 {
		t.Fatalf("cuentas tras dividir: %+v", g.Cuentas)
	}
	if despues := sumaImportes(t, p.ID); despues != antes {
		t.Fatalf("la suma cambió: %s → %s", antes, despues)
	}
}

func TestDividirCuentaArrastraLosAdicionales(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	plato := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	extra := productoVentaSQL(t, "PRINCIPAL", "ADICIONAL", "2.00", true)
	padre := agregarOK(t, tk, p.ID, cuenta, plato, "1", "")
	hijo := agregarOK(t, tk, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID))
	otro := agregarOK(t, tk, p.ID, cuenta, plato, "1", "")

	esperar(t, dividirCuenta(t, tk, p.ID, []string{hijo.ID}, ""), http.StatusUnprocessableEntity, "adicional_sin_padre")

	r := dividirCuenta(t, tk, p.ID, []string{padre.ID}, "")
	esperar(t, r, http.StatusOK, "")
	var nueva cuentaResp
	r.json(t, &nueva)
	if len(nueva.Lineas) != 2 || nueva.Total != "12.00" {
		t.Fatalf("el adicional debe viajar con su padre: %+v", nueva)
	}
	g := obtenerPedido(t, tk, p.ID)
	if len(g.Cuentas[0].Lineas) != 1 || g.Cuentas[0].Lineas[0].ID != otro.ID {
		t.Fatalf("la cuenta de origen conserva solo la otra línea: %+v", g.Cuentas[0])
	}
}

// La cocina no se entera de la división: la comanda cuelga de la línea, no de la cuenta.
func TestDividirCuentaConservaLaTrazabilidadDeCocina(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})

	r := dividirCuenta(t, e.mesero, e.pedido.ID, []string{e.linea.ID}, "")
	esperar(t, r, http.StatusOK, "")

	if cuenta := scalarStr(t, `SELECT cuenta_id::text FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, e.linea.ID); cuenta == e.cuenta {
		t.Fatal("la línea debió cambiar de cuenta")
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint AND estado='PENDIENTE'`, c.Lineas[0].ID); n != 1 {
		t.Fatalf("la línea de comanda debe seguir intacta (%d)", n)
	}
	if got := colaDe(t, e.cocinero, url.Values{"estacion_id": {e.estacion}}).ids(); len(got) != 1 || got[0] != c.ID {
		t.Fatalf("la cola de cocina no cambia: %v", got)
	}
}

func TestDividirCuentaRechazos(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	l1 := agregarOK(t, tk, p.ID, cuenta, prod, "2", "")
	otro := abrirLlevar(t, tk, "PRINCIPAL")
	ajena := agregarOK(t, tk, otro.ID, otro.Cuentas[0].ID, prod, "1", "")

	esperar(t, dividirCuenta(t, lectorPedidos(t), p.ID, []string{l1.ID}, ""), http.StatusForbidden, "sin_permiso")
	esperar(t, dividirCuenta(t, tk, p.ID, nil, ""), http.StatusUnprocessableEntity, "seleccion_vacia")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{l1.ID, l1.ID}, ""), http.StatusUnprocessableEntity, "detalle_duplicado")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{ajena.ID}, ""), http.StatusNotFound, "no_encontrado")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{"abc"}, ""), http.StatusBadRequest, "detalle_ids_invalido")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{l1.ID}, `,"nombre_cuenta":"`+strings.Repeat("x", 81)+`"`),
		http.StatusUnprocessableEntity, "nombre_cuenta_invalido")

	pagoConfirmadoSQL(t, p.ID, cuenta, "1.00")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{l1.ID}, ""), http.StatusConflict, "cuenta_con_pago")
	if n := len(obtenerPedido(t, tk, p.ID).Cuentas); n != 1 {
		t.Fatalf("un rechazo no debe crear cuentas (%d)", n)
	}
}

func TestDividirCuentaEsIdempotente(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	l := agregarOK(t, tk, p.ID, p.Cuentas[0].ID, prod, "1", "")
	ruta := "/api/v1/pedidos/" + p.ID + "/dividir-cuenta"
	cuerpo := fmt.Sprintf(`{"detalle_ids":[%q]}`, l.ID)
	h := map[string]string{"Idempotency-Key": nuevaClave()}

	r1 := callH(t, "POST", ruta, cuerpo, tk, h)
	esperar(t, r1, http.StatusOK, "")
	r2 := callH(t, "POST", ruta, cuerpo, tk, h)
	esperar(t, r2, http.StatusOK, "")
	var c1, c2 cuentaResp
	r1.json(t, &c1)
	r2.json(t, &c2)
	if c1.ID != c2.ID {
		t.Fatalf("repetir la clave devuelve la misma cuenta: %s y %s", c1.ID, c2.ID)
	}
	if n := len(obtenerPedido(t, tk, p.ID).Cuentas); n != 2 {
		t.Fatalf("debe existir una sola cuenta nueva (%d cuentas)", n)
	}
}
