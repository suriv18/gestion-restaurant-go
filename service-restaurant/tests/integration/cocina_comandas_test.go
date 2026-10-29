package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// RES-CP-012: se envían las cantidades pendientes sin reenviar lo ya despachado.
func TestComandaSeEnviaEnTandasSinReenviar(t *testing.T) {
	e := nuevoEscenario(t, "3")

	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	if c.PedidoID != e.pedido.ID || c.EstacionID != e.estacion || c.SucursalID != sedeID(t, "PRINCIPAL") || c.MesaID != nil ||
		len(c.Lineas) != 1 {
		t.Fatalf("comanda: %+v", c)
	}
	l := c.Lineas[0]
	if l.DetallePedidoID != e.linea.ID || l.Cantidad != "2.000" || l.Estado != "PENDIENTE" ||
		l.Descripcion != e.linea.Descripcion || l.Instrucciones == nil || *l.Instrucciones != "sin ají" {
		t.Fatalf("línea de comanda: %+v", l)
	}

	esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"}),
		http.StatusConflict, "cantidad_excede_pendiente")
	enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "0.001"}),
		http.StatusConflict, "cantidad_excede_pendiente")
}

func TestComandaRepetidaConLaMismaClaveDevuelveLaOriginal(t *testing.T) {
	e := nuevoEscenario(t, "3")
	clave := nuevaClave()
	cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"1"}]}`, e.estacion, e.linea.ID)
	h := map[string]string{"Idempotency-Key": clave}
	ruta := "/api/v1/pedidos/" + e.pedido.ID + "/comandas"

	r1 := callH(t, "POST", ruta, cuerpo, e.mesero, h)
	esperar(t, r1, http.StatusCreated, "")
	var c1, c2 comandaResp
	r1.json(t, &c1)
	r2 := callH(t, "POST", ruta, cuerpo, e.mesero, h)
	esperar(t, r2, http.StatusCreated, "")
	r2.json(t, &c2)
	if c1.ID != c2.ID {
		t.Fatalf("la repetición debe devolver la comanda original: %s y %s", c1.ID, c2.ID)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comandas WHERE pedido_id=$1::text::bigint`, e.pedido.ID); n != 1 {
		t.Fatalf("la repetición no debe crear otra comanda (hay %d)", n)
	}

	otro := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"2"}]}`, e.estacion, e.linea.ID)
	esperar(t, callH(t, "POST", ruta, otro, e.mesero, h), http.StatusConflict, "idempotency_key_reutilizada")
}

// La clave de idempotencia es obligatoria: sin ella (o con una inválida) no se crea nada.
func TestComandaExigeIdempotencyKey(t *testing.T) {
	e := nuevoEscenario(t, "2")
	cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"1"}]}`, e.estacion, e.linea.ID)
	ruta := "/api/v1/pedidos/" + e.pedido.ID + "/comandas"

	esperar(t, callH(t, "POST", ruta, cuerpo, e.mesero, nil), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, callH(t, "POST", ruta, cuerpo, e.mesero, map[string]string{"Idempotency-Key": "no-es-uuid"}),
		http.StatusBadRequest, "idempotency_key_invalida")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comandas WHERE pedido_id=$1::text::bigint`, e.pedido.ID); n != 0 {
		t.Fatalf("sin clave válida no debe haber comandas (hay %d)", n)
	}
}

func TestComandaRechazaEnviosInvalidos(t *testing.T) {
	e := nuevoEscenario(t, "2")
	otraEstacion := estacionSQL(t, "PRINCIPAL", "EO"+sufijo())
	sinEstacion := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "5.00", true)
	r := agregarConsumo(t, e.mesero, e.pedido.ID, e.cuenta, sinEstacion, "1", "")
	esperar(t, r, http.StatusCreated, "")
	var lSin lineaResp
	r.json(t, &lSin)
	ajeno, ajenaCuenta := pedidoSQL(t, "PRINCIPAL")
	rA := agregarConsumo(t, e.mesero, ajeno, ajenaCuenta, e.producto, "1", "")
	esperar(t, rA, http.StatusCreated, "")
	var lAjena lineaResp
	rA.json(t, &lAjena)

	casos := []struct {
		nombre   string
		estacion string
		lineas   [][2]string
		status   int
		code     string
	}{
		{"estación que no corresponde al producto", otraEstacion, [][2]string{{e.linea.ID, "1"}}, 422, "estacion_no_corresponde"},
		{"producto sin estación", e.estacion, [][2]string{{lSin.ID, "1"}}, 422, "estacion_no_corresponde"},
		{"línea de otro pedido", e.estacion, [][2]string{{lAjena.ID, "1"}}, 422, "linea_invalida"},
		{"línea inexistente", e.estacion, [][2]string{{"999999999", "1"}}, 422, "linea_invalida"},
		{"línea repetida", e.estacion, [][2]string{{e.linea.ID, "1"}, {e.linea.ID, "1"}}, 422, "linea_duplicada"},
		{"cantidad cero", e.estacion, [][2]string{{e.linea.ID, "0"}}, 422, "cantidad_invalida"},
		{"cantidad mayor que la línea", e.estacion, [][2]string{{e.linea.ID, "2.001"}}, 409, "cantidad_excede_pendiente"},
		{"estación inexistente", "999999999", [][2]string{{e.linea.ID, "1"}}, 422, "estacion_invalida"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, tc.estacion, tc.lineas...), tc.status, tc.code)
		})
	}
	esperar(t, post(t, "/api/v1/pedidos/"+e.pedido.ID+"/comandas", fmt.Sprintf(`{"estacion_id":%q,"lineas":[]}`, e.estacion), e.mesero),
		http.StatusUnprocessableEntity, "lineas_invalidas")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comandas WHERE pedido_id=$1::text::bigint`, e.pedido.ID); n != 0 {
		t.Fatalf("ningún envío rechazado debe dejar comandas (hay %d)", n)
	}
}

func TestComandaNoSeEnviaDeUnaLineaAnulada(t *testing.T) {
	e := nuevoEscenario(t, "2")
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.detalles_pedido SET estado='ANULADO', anulado_en=now(),
		anulado_por=%d, motivo_anulacion='prueba' WHERE id=%s`, adminID(t), e.linea.ID))
	esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}),
		http.StatusConflict, "linea_anulada")
}

func TestComandaPermisosYSedes(t *testing.T) {
	e := nuevoEscenario(t, "2")

	t.Run("quien solo ve pedidos no envía", func(t *testing.T) {
		esperar(t, enviarComanda(t, lectorPedidos(t), e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusForbidden, "sin_permiso")
	})
	t.Run("el cocinero no crea envíos", func(t *testing.T) {
		esperar(t, enviarComanda(t, e.cocinero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusForbidden, "sin_permiso")
	})
	t.Run("sin sesión", func(t *testing.T) {
		esperar(t, enviarComanda(t, "", e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}), http.StatusUnauthorized, "")
	})
	t.Run("pedido inexistente", func(t *testing.T) {
		esperar(t, enviarComanda(t, e.mesero, "999999999", e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusNotFound, "no_encontrado")
	})
	t.Run("pedido de una sede que no ve", func(t *testing.T) {
		crearSedeSQL(t, "COC2")
		ajeno, _ := pedidoSQL(t, "COC2")
		esperar(t, enviarComanda(t, e.mesero, ajeno, e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusNotFound, "no_encontrado")
	})
}
