package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type comandaLineaResp struct {
	ID              string  `json:"id"`
	ComandaID       string  `json:"comanda_id"`
	PedidoID        string  `json:"pedido_id"`
	DetallePedidoID string  `json:"detalle_pedido_id"`
	Cantidad        string  `json:"cantidad"`
	Descripcion     string  `json:"descripcion"`
	Instrucciones   *string `json:"instrucciones"`
	Estado          string  `json:"estado"`
	IniciadoEn      *string `json:"iniciado_en"`
	ListoEn         *string `json:"listo_en"`
	EntregadoEn     *string `json:"entregado_en"`
}

type comandaResp struct {
	ID                string             `json:"id"`
	PedidoID          string             `json:"pedido_id"`
	SucursalID        string             `json:"sucursal_id"`
	EstacionID        string             `json:"estacion_id"`
	MesaID            *string            `json:"mesa_id"`
	ClaveIdempotencia string             `json:"clave_idempotencia"`
	EnviadoPor        string             `json:"enviado_por"`
	Lineas            []comandaLineaResp `json:"lineas"`
}

type comandasPagina struct {
	Items      []comandaResp `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

// estacionSQL crea, como sistema, una estación activa en la sede y devuelve su id.
func estacionSQL(t *testing.T, sede, codigo string) string {
	t.Helper()
	app(t)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.estaciones_preparacion(sucursal_id,codigo,nombre)
		SELECT s.id,'%s','Estación %s' FROM sch_restaurante.sucursales s WHERE s.codigo='%s'`, codigo, codigo, sede))
	return scalarStr(t, `SELECT e.id FROM sch_restaurante.estaciones_preparacion e
		JOIN sch_restaurante.sucursales s ON s.id=e.sucursal_id WHERE s.codigo=$1 AND e.codigo=$2`, sede, codigo)
}

// productoEnEstacionSQL crea un producto disponible en la sede y lo asigna a la estación en esa sede.
func productoEnEstacionSQL(t *testing.T, sede, estacion, precio string) string {
	t.Helper()
	prod := productoVentaSQL(t, sede, "PREPARADO", precio, true)
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos_sucursal SET estacion_id=%s
		WHERE producto_id=%s AND sucursal_id=(SELECT id FROM sch_restaurante.sucursales WHERE codigo='%s')`, estacion, prod, sede))
	return prod
}

// tokenCocinero: opera la cocina y ve pedidos en PRINCIPAL y las sedes dadas.
func tokenCocinero(t *testing.T, sedes ...string) string {
	return tokenPedidos(t, "coc.cocinero", "T_COC_COCINA", []string{"COCINA_OPERAR", "PEDIDO_VER"}, sedes...)
}

// escenarioCocina es un pedido LLEVAR de PRINCIPAL con una línea de la cantidad dada, cuyo producto
// se prepara en una estación propia de la prueba.
type escenarioCocina struct {
	mesero   string // token con PEDIDO_VER, PEDIDO_CREAR y PEDIDO_EDITAR
	cocinero string // token con COCINA_OPERAR y PEDIDO_VER
	pedido   pedidoResp
	cuenta   string
	estacion string
	producto string
	linea    lineaResp
}

func nuevoEscenario(t *testing.T, cantidad string) escenarioCocina {
	t.Helper()
	mesero := meseroPedidos(t)
	est := estacionSQL(t, "PRINCIPAL", "EC"+sufijo())
	prod := productoEnEstacionSQL(t, "PRINCIPAL", est, "10.00")
	p := abrirLlevar(t, mesero, "PRINCIPAL")
	r := agregarConsumo(t, mesero, p.ID, p.Cuentas[0].ID, prod, cantidad, `,"instrucciones":"sin ají"`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	return escenarioCocina{mesero: mesero, cocinero: tokenCocinero(t), pedido: p, cuenta: p.Cuentas[0].ID,
		estacion: est, producto: prod, linea: l}
}

// enviarComanda envía a la estación pares {id de línea, cantidad} y devuelve la respuesta cruda.
func enviarComanda(t *testing.T, tk, pedido, estacion string, lineas ...[2]string) resp {
	t.Helper()
	items := make([]string, len(lineas))
	for i, l := range lineas {
		items[i] = fmt.Sprintf(`{"detalle_pedido_id":%q,"cantidad":%q}`, l[0], l[1])
	}
	cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[%s]}`, estacion, strings.Join(items, ","))
	return post(t, "/api/v1/pedidos/"+pedido+"/comandas", cuerpo, tk)
}

// enviarOK envía y decodifica la comanda creada.
func enviarOK(t *testing.T, tk, pedido, estacion string, lineas ...[2]string) comandaResp {
	t.Helper()
	r := enviarComanda(t, tk, pedido, estacion, lineas...)
	esperar(t, r, http.StatusCreated, "")
	var c comandaResp
	r.json(t, &c)
	return c
}
