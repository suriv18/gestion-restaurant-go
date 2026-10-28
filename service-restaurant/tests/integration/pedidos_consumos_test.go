package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// RES-CP-010: cambiar el precio de la carta no modifica un detalle existente.
func TestConsumoConservaElPrecioHistorico(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	r := agregarConsumo(t, tk, p.ID, cuenta, prod, "2", `,"instrucciones":" sin ají "`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	if l.Cantidad != "2.000" || l.PrecioUnitario != "10.000000" || l.DescuentoTotal != "0.00" || l.AfectacionIGV != "10" ||
		l.TasaImpuesto != "18.0000" || l.ImporteTotal != "20.00" || l.BaseImponible != "16.95" || l.Estado != "ACTIVO" ||
		l.Notas == nil || *l.Notas != "sin ají" || l.CuentaID != cuenta || l.PedidoID != p.ID {
		t.Fatalf("línea: %+v", l)
	}

	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos SET precio_venta = 15 WHERE id = %s`, prod))
	r2 := agregarConsumo(t, tk, p.ID, cuenta, prod, "2", "")
	esperar(t, r2, http.StatusCreated, "")
	var l2 lineaResp
	r2.json(t, &l2)
	if l2.PrecioUnitario != "15.000000" {
		t.Fatalf("la línea nueva usa el precio vigente: %+v", l2)
	}

	g := obtenerPedido(t, tk, p.ID)
	c := g.Cuentas[0]
	if len(c.Lineas) != 2 || c.Lineas[0].PrecioUnitario != "10.000000" || c.Lineas[0].ImporteTotal != "20.00" {
		t.Fatalf("la línea existente no debe cambiar: %+v", c.Lineas)
	}
	if c.Subtotal != "42.37" || c.Impuesto != "7.63" || c.Total != "50.00" {
		t.Fatalf("totales de la cuenta: %+v", c)
	}
}

func TestConsumoUsaElPrecioPropioDeLaSede(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos_sucursal SET precio_venta = 12.50 WHERE producto_id = %s`, prod))
	r := agregarConsumo(t, tk, p.ID, p.Cuentas[0].ID, prod, "1", "")
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	if l.PrecioUnitario != "12.500000" {
		t.Fatalf("debe usar el precio de la sede: %+v", l)
	}
}

func TestConsumoRechazos(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	ok := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	t.Run("cantidad no positiva", func(t *testing.T) {
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, ok, "0", ""), 422, "cantidad_invalida")
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, ok, "-1", ""), 422, "cantidad_invalida")
	})
	t.Run("producto inexistente", func(t *testing.T) {
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, "999999", "1", ""), 422, "producto_invalido")
	})
	t.Run("producto dado de baja", func(t *testing.T) {
		baja := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
		asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos SET es_activo=false, motivo_eliminacion='prueba' WHERE id=%s`, baja))
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, baja, "1", ""), 422, "producto_inactivo")
	})
	t.Run("producto pausado en la sede", func(t *testing.T) {
		pausado := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", false)
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, pausado, "1", ""), 422, "producto_no_disponible")
	})
	t.Run("producto sin fila en la sede", func(t *testing.T) {
		sinSede := productoVentaSQL(t, "", "PREPARADO", "10.00", true)
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, sinSede, "1", ""), 422, "producto_no_disponible")
	})
	t.Run("cuenta de otro pedido", func(t *testing.T) {
		otro := abrirLlevar(t, tk, "PRINCIPAL")
		esperar(t, agregarConsumo(t, tk, p.ID, otro.Cuentas[0].ID, ok, "1", ""), 422, "cuenta_invalida")
	})
	t.Run("pedido inexistente", func(t *testing.T) {
		esperar(t, agregarConsumo(t, tk, "999999", cuenta, ok, "1", ""), 404, "no_encontrado")
	})
	t.Run("campo desconocido", func(t *testing.T) {
		esperar(t, post(t, "/api/v1/pedidos/"+p.ID+"/detalles",
			fmt.Sprintf(`{"producto_id":%q,"cuenta_id":%q,"cantidad":"1","precio_unitario":"1.00"}`, ok, cuenta), tk), 400, "")
	})
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, p.ID); n != 0 {
		t.Fatalf("los rechazos no deben dejar líneas (%d)", n)
	}
}

func TestConsumoDescuentoExigePermiso(t *testing.T) {
	cajero, mesero := cajeroPedidos(t), meseroPedidos(t)
	p := abrirLlevar(t, cajero, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	extra := `,"descuento_monto":"5.00"`

	esperar(t, agregarConsumo(t, mesero, p.ID, cuenta, prod, "2", extra), http.StatusForbidden, "sin_permiso")
	r := agregarConsumo(t, cajero, p.ID, cuenta, prod, "2", extra)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	if l.DescuentoTotal != "5.00" || l.ImporteTotal != "15.00" {
		t.Fatalf("línea con descuento: %+v", l)
	}
	// El mesero sí agrega sin descuento.
	esperar(t, agregarConsumo(t, mesero, p.ID, cuenta, prod, "1", ""), http.StatusCreated, "")
	// El descuento no puede superar el importe.
	esperar(t, agregarConsumo(t, cajero, p.ID, cuenta, prod, "1", `,"descuento_monto":"10.01"`), 422, "descuento_excede_importe")
}

func TestConsumoAdicionalCuelgaDeUnaLinea(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	plato := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "20.00", true)
	extra := productoVentaSQL(t, "PRINCIPAL", "ADICIONAL", "3.00", true)

	esperar(t, agregarConsumo(t, tk, p.ID, cuenta, extra, "1", ""), 422, "adicional_sin_padre")

	r := agregarConsumo(t, tk, p.ID, cuenta, plato, "1", "")
	esperar(t, r, http.StatusCreated, "")
	var padre lineaResp
	r.json(t, &padre)

	esperar(t, agregarConsumo(t, tk, p.ID, cuenta, plato, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID)), 422, "padre_no_aplica")
	esperar(t, agregarConsumo(t, tk, p.ID, cuenta, extra, "1", `,"detalle_padre_id":"999999"`), 422, "padre_invalido")

	r = agregarConsumo(t, tk, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID))
	esperar(t, r, http.StatusCreated, "")
	var hija lineaResp
	r.json(t, &hija)
	if hija.DetallePadreID == nil || *hija.DetallePadreID != padre.ID {
		t.Fatalf("la línea hija debe apuntar a su padre: %+v", hija)
	}
}

func TestConsumoAgregarEsIdempotente(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	cuerpo := fmt.Sprintf(`{"producto_id":%q,"cuenta_id":%q,"cantidad":"1"}`, prod, p.Cuentas[0].ID)
	h := map[string]string{"Idempotency-Key": nuevaClave()}
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles"

	r1 := callH(t, http.MethodPost, ruta, cuerpo, tk, h)
	r2 := callH(t, http.MethodPost, ruta, cuerpo, tk, h)
	esperar(t, r1, http.StatusCreated, "")
	esperar(t, r2, http.StatusCreated, "")
	var l1, l2 lineaResp
	r1.json(t, &l1)
	r2.json(t, &l2)
	if l1.ID != l2.ID {
		t.Fatalf("el reintento debe devolver la misma línea: %s / %s", l1.ID, l2.ID)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, p.ID); n != 1 {
		t.Fatalf("líneas = %d, quería 1", n)
	}
	// Misma clave con otro cuerpo: conflicto, sin línea nueva.
	otro := fmt.Sprintf(`{"producto_id":%q,"cuenta_id":%q,"cantidad":"2"}`, prod, p.Cuentas[0].ID)
	esperar(t, callH(t, http.MethodPost, ruta, otro, tk, h), http.StatusConflict, "idempotency_key_reutilizada")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, p.ID); n != 1 {
		t.Fatalf("líneas tras la clave reutilizada = %d, quería 1", n)
	}
}

func TestConsumoEditar(t *testing.T) {
	cajero, mesero := cajeroPedidos(t), meseroPedidos(t)
	p := abrirLlevar(t, cajero, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	r := agregarConsumo(t, cajero, p.ID, p.Cuentas[0].ID, prod, "2", `,"instrucciones":"sin sal"`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles/" + l.ID

	// Cantidad.
	r = call(t, "PATCH", ruta, `{"cantidad":"3"}`, mesero)
	esperar(t, r, http.StatusOK, "")
	var tras lineaResp
	r.json(t, &tras)
	if tras.Cantidad != "3.000" || tras.ImporteTotal != "30.00" || tras.Notas == nil || *tras.Notas != "sin sal" {
		t.Fatalf("tras editar la cantidad: %+v", tras)
	}
	// Vaciar las instrucciones (null).
	r = call(t, "PATCH", ruta, `{"instrucciones":null}`, mesero)
	esperar(t, r, http.StatusOK, "")
	var vacia lineaResp
	r.json(t, &vacia)
	if vacia.Notas != nil {
		t.Fatalf("las instrucciones debían vaciarse: %+v", vacia)
	}
	// Sin cambios, cantidad inválida y desconocidos.
	esperar(t, call(t, "PATCH", ruta, `{}`, mesero), 422, "sin_cambios")
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"0"}`, mesero), 422, "cantidad_invalida")
	esperar(t, call(t, "PATCH", ruta, `{"precio_unitario":"1.00"}`, mesero), 400, "")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+p.ID+"/detalles/999999", `{"cantidad":"1"}`, mesero), 404, "no_encontrado")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+p.ID+"/detalles/abc", `{"cantidad":"1"}`, mesero), 400, "detalle_id_invalido")
	// Descuento: sin permiso 403; con permiso, 200; y no puede superar el importe.
	esperar(t, call(t, "PATCH", ruta, `{"descuento_monto":"1.00"}`, mesero), http.StatusForbidden, "sin_permiso")
	r = call(t, "PATCH", ruta, `{"descuento_monto":"10.00"}`, cajero)
	esperar(t, r, http.StatusOK, "")
	var desc lineaResp
	r.json(t, &desc)
	if desc.DescuentoTotal != "10.00" || desc.ImporteTotal != "20.00" {
		t.Fatalf("tras el descuento: %+v", desc)
	}
	esperar(t, call(t, "PATCH", ruta, `{"descuento_monto":"30.01"}`, cajero), 422, "descuento_excede_importe")
	// Una línea anulada ya no se edita.
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.detalles_pedido SET estado='ANULADO', anulado_en=now(),
		anulado_por=(SELECT id FROM sch_restaurante.usuarios WHERE es_sistema), motivo_anulacion='prueba' WHERE id=%s`, l.ID))
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"1"}`, mesero), http.StatusConflict, "linea_anulada")
}
