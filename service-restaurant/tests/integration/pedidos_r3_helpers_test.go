package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// supervisorPedidos: ver, crear, editar y anular pedidos y autorizar descuentos en PRINCIPAL y las
// sedes dadas. Es lo que el cajero real recibe tras las migraciones 00011 y 00013.
func supervisorPedidos(t *testing.T, sedes ...string) string {
	usuario := "ped.supervisor"
	if len(sedes) > 0 {
		usuario += "." + strings.Join(sedes, "+")
	}
	return tokenPedidos(t, usuario, "T_PED_SUPERVISOR",
		[]string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR", "PEDIDO_ANULAR", "DESCUENTO_AUTORIZAR"}, sedes...)
}

// agregarOK agrega una línea y la devuelve ya decodificada.
func agregarOK(t *testing.T, tk, pedido, cuenta, producto, cantidad, extras string) lineaResp {
	t.Helper()
	r := agregarConsumo(t, tk, pedido, cuenta, producto, cantidad, extras)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	return l
}

// pagoConfirmadoSQL registra, como sistema, un pago EFECTIVO confirmado sobre la cuenta. Abre antes
// una caja y una sesión propias; el monto no puede superar el saldo de la cuenta.
func pagoConfirmadoSQL(t *testing.T, pedido, cuenta, monto string) {
	t.Helper()
	cod := "CJ" + sufijo()
	asSystem(t,
		fmt.Sprintf(`INSERT INTO sch_restaurante.cajas(sucursal_id,codigo,nombre)
			SELECT p.sucursal_id,'%s','Caja %s' FROM sch_restaurante.pedidos p WHERE p.id=%s`, cod, cod, pedido),
		fmt.Sprintf(`INSERT INTO sch_restaurante.sesiones_caja(caja_id,sucursal_id,moneda,abierto_por)
			SELECT c.id,c.sucursal_id,(SELECT moneda FROM sch_restaurante.pedidos WHERE id=%s),
			       (SELECT id FROM sch_restaurante.usuarios WHERE es_sistema)
			FROM sch_restaurante.cajas c WHERE c.codigo='%s'`, pedido, cod),
		fmt.Sprintf(`INSERT INTO sch_restaurante.pagos(cuenta_id,sesion_caja_id,metodo_pago_id,monto,efectivo_recibido,estado,clave_idempotencia)
			SELECT %s, s.id, m.id, %s, %s, 'CONFIRMADO', gen_random_uuid()
			FROM sch_restaurante.sesiones_caja s JOIN sch_restaurante.cajas c ON c.id=s.caja_id,
			     sch_restaurante.metodos_pago m
			WHERE c.codigo='%s' AND m.codigo='EFECTIVO'`, cuenta, monto, monto, cod))
}

// dividirCuenta envía la división de las líneas dadas; extras se añade al cuerpo (p. ej.
// `,"nombre_cuenta":"Juan"`). Devuelve la respuesta cruda.
func dividirCuenta(t *testing.T, tk, pedido string, ids []string, extras string) resp {
	t.Helper()
	citados := make([]string, len(ids))
	for i, id := range ids {
		citados[i] = fmt.Sprintf("%q", id)
	}
	cuerpo := `{"detalle_ids":[` + strings.Join(citados, ",") + `]` + extras + `}`
	return post(t, "/api/v1/pedidos/"+pedido+"/dividir-cuenta", cuerpo, tk)
}

// sumaImportes devuelve la suma de importe_total de las líneas activas del pedido.
func sumaImportes(t *testing.T, pedido string) string {
	t.Helper()
	return scalarStr(t, `SELECT coalesce(sum(importe_total),0)::text FROM sch_restaurante.detalles_pedido
		WHERE pedido_id=$1::text::bigint AND estado='ACTIVO'`, pedido)
}
