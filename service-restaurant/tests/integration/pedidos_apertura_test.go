package integration

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"testing"
)

func TestPedidosAbrirSalonCreaPedidoYCuenta(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	tk := cajeroPedidos(t)
	sede := sedeID(t, "PRINCIPAL")

	r := post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2,"notas":" sin cebolla "}`, sede, mesa), tk)
	esperar(t, r, http.StatusCreated, "")
	var p pedidoResp
	r.json(t, &p)
	if p.Tipo != "SALON" || p.Estado != "REGISTRADO" || p.Moneda != "PEN" || p.Comensales != 2 || p.SucursalID != sede ||
		p.MesaID == nil || *p.MesaID != mesa || p.Notas == nil || *p.Notas != "sin cebolla" || p.Reparto != nil {
		t.Fatalf("pedido: %+v", p)
	}
	if len(p.Cuentas) != 1 || p.Cuentas[0].Numero != 1 || p.Cuentas[0].Estado != "ABIERTA" ||
		p.Cuentas[0].Total != "0.00" || len(p.Cuentas[0].Lineas) != 0 {
		t.Fatalf("debe traer la cuenta nº 1 abierta y vacía: %+v", p.Cuentas)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.cuentas WHERE pedido_id=$1::text::bigint`, p.ID); n != 1 {
		t.Fatalf("cuentas del pedido = %d, quería 1", n)
	}
	// El pedido abierto ocupa la mesa.
	if got := scalarStr(t, `SELECT estado_actual FROM sch_restaurante.v_mesas_estado WHERE mesa_id=$1::text::bigint`, mesa); got != "OCUPADA" {
		t.Fatalf("la mesa debe quedar OCUPADA, está %s", got)
	}
	// GET devuelve lo mismo.
	if g := obtenerPedido(t, tk, p.ID); g.ID != p.ID || len(g.Cuentas) != 1 {
		t.Fatalf("GET: %+v", g)
	}
}

func TestPedidosAbrirLlevarYDelivery(t *testing.T) {
	tk := cajeroPedidos(t)
	sede := sedeID(t, "PRINCIPAL")

	p := abrirLlevar(t, tk, "PRINCIPAL")
	if p.Tipo != "LLEVAR" || p.MesaID != nil || p.Reparto != nil {
		t.Fatalf("LLEVAR: %+v", p)
	}

	r := post(t, "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"DELIVERY","comensales":1,
		"reparto":{"destinatario":" Ana Pérez ","telefono":"999111222","direccion":"Av. Lima 123","referencia":"Frente al parque"}}`, sede), tk)
	esperar(t, r, http.StatusCreated, "")
	var d pedidoResp
	r.json(t, &d)
	if d.Tipo != "DELIVERY" || d.Reparto == nil || d.Reparto.Destinatario != "Ana Pérez" || d.Reparto.Canal != "PROPIO" ||
		d.Reparto.Estado != "PENDIENTE" || d.Reparto.Referencia == nil || *d.Reparto.Referencia != "Frente al parque" {
		t.Fatalf("DELIVERY: %+v", d)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos_delivery WHERE pedido_id=$1::text::bigint`, d.ID); n != 1 {
		t.Fatalf("pedidos_delivery = %d, quería 1", n)
	}
}

func TestPedidosAbrirRechazos(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 2)
	tk := cajeroPedidos(t)
	sede := sedeID(t, "PRINCIPAL")
	casos := []struct {
		nombre, cuerpo string
		status         int
		code           string
	}{
		{"capacidad excedida", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":3}`, sede, mesa), 422, "capacidad_excedida"},
		{"salón sin mesa", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","comensales":1}`, sede), 422, "mesa_requerida"},
		{"llevar con mesa", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","mesa_id":%q,"comensales":1}`, sede, mesa), 422, "mesa_no_aplica"},
		{"mesa inexistente", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":"999999","comensales":1}`, sede), 422, "mesa_invalida"},
		{"delivery sin reparto", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"DELIVERY","comensales":1}`, sede), 422, "reparto_requerido"},
		{"tipo desconocido", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"MOSTRADOR","comensales":1}`, sede), 422, "tipo_invalido"},
		{"comensales cero", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":0}`, sede), 422, "comensales_invalidos"},
		{"campo desconocido", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1,"total":"10"}`, sede), 400, ""},
		{"id mal formado", `{"sucursal_id":"abc","tipo":"LLEVAR","comensales":1}`, 400, "sucursal_id_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) { esperar(t, post(t, "/api/v1/pedidos", c.cuerpo, tk), c.status, c.code) })
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE mesa_id=$1::text::bigint`, mesa); n != 0 {
		t.Fatalf("los rechazos no deben dejar pedidos (%d)", n)
	}
	// Sin Idempotency-Key.
	esperar(t, call(t, "POST", "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sede), tk),
		http.StatusBadRequest, "idempotency_key_requerida")
}

func TestPedidosMesaOcupada(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	tk := cajeroPedidos(t)
	cuerpo := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2}`, sedeID(t, "PRINCIPAL"), mesa)
	esperar(t, post(t, "/api/v1/pedidos", cuerpo, tk), http.StatusCreated, "")
	esperar(t, post(t, "/api/v1/pedidos", cuerpo, tk), http.StatusConflict, "mesa_ocupada")
}

// Dos aperturas simultáneas de la misma mesa: solo una gana.
func TestPedidosDosAperturasSimultaneasDeUnaMesa(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	tk := cajeroPedidos(t)
	cuerpo := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2}`, sedeID(t, "PRINCIPAL"), mesa)

	codigos := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codigos {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codigos[i] = post(t, "/api/v1/pedidos", cuerpo, tk).Code
		}(i)
	}
	wg.Wait()
	sort.Ints(codigos)
	if codigos[0] != http.StatusCreated || codigos[1] != http.StatusConflict {
		t.Fatalf("estados %v, quería [201 409]", codigos)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE mesa_id=$1::text::bigint AND cerrado_en IS NULL`, mesa); n != 1 {
		t.Fatalf("pedidos abiertos en la mesa = %d, quería 1", n)
	}
}

// RES-CP-009: si falla la creación de la cuenta no queda una cabecera de pedido huérfana.
func TestPedidosAperturaSinCabeceraHuerfana(t *testing.T) {
	app(t)
	asSystem(t,
		`CREATE OR REPLACE FUNCTION sch_restaurante.fn_test_falla_cuenta() RETURNS trigger LANGUAGE plpgsql AS $$
		 BEGIN
		   IF EXISTS (SELECT 1 FROM sch_restaurante.pedidos WHERE id = NEW.pedido_id AND notas = 'FALLA-CUENTA') THEN
		     RAISE EXCEPTION 'falla simulada de la cuenta';
		   END IF;
		   RETURN NEW;
		 END $$`,
		`CREATE TRIGGER zz_test_falla_cuenta BEFORE INSERT ON sch_restaurante.cuentas
		 FOR EACH ROW EXECUTE FUNCTION sch_restaurante.fn_test_falla_cuenta()`)
	t.Cleanup(func() {
		asSystem(t, `DROP TRIGGER IF EXISTS zz_test_falla_cuenta ON sch_restaurante.cuentas`,
			`DROP FUNCTION IF EXISTS sch_restaurante.fn_test_falla_cuenta()`)
	})

	tk := cajeroPedidos(t)
	r := post(t, "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1,"notas":"FALLA-CUENTA"}`, sedeID(t, "PRINCIPAL")), tk)
	esperar(t, r, http.StatusUnprocessableEntity, "regla_de_datos")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE notas='FALLA-CUENTA'`); n != 0 {
		t.Fatalf("quedaron %d cabeceras huérfanas", n)
	}
}

func TestPedidosAperturaIdempotente(t *testing.T) {
	tk := cajeroPedidos(t)
	cuerpo := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, "PRINCIPAL"))
	clave := nuevaClave()
	h := map[string]string{"Idempotency-Key": clave}

	r1 := callH(t, http.MethodPost, "/api/v1/pedidos", cuerpo, tk, h)
	esperar(t, r1, http.StatusCreated, "")
	r2 := callH(t, http.MethodPost, "/api/v1/pedidos", cuerpo, tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var p1, p2 pedidoResp
	r1.json(t, &p1)
	r2.json(t, &p2)
	if p1.ID != p2.ID || p1.ClaveIdempotencia != clave {
		t.Fatalf("el reintento debe devolver el mismo pedido: %s / %s (clave %s)", p1.ID, p2.ID, p1.ClaveIdempotencia)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE clave_idempotencia=$1::uuid`, clave); n != 1 {
		t.Fatalf("pedidos con esa clave = %d, quería 1", n)
	}
	// La misma clave con otro contenido es un conflicto.
	otro := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":3}`, sedeID(t, "PRINCIPAL"))
	esperar(t, callH(t, http.MethodPost, "/api/v1/pedidos", otro, tk, h), http.StatusConflict, "idempotency_key_reutilizada")
}
