package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type lineaResp struct {
	ID             string  `json:"id"`
	PedidoID       string  `json:"pedido_id"`
	CuentaID       string  `json:"cuenta_id"`
	ProductoID     string  `json:"producto_id"`
	DetallePadreID *string `json:"detalle_padre_id"`
	Descripcion    string  `json:"descripcion"`
	Cantidad       string  `json:"cantidad"`
	PrecioUnitario string  `json:"precio_unitario"`
	DescuentoTotal string  `json:"descuento_total"`
	AfectacionIGV  string  `json:"afectacion_igv"`
	TasaImpuesto   string  `json:"tasa_impuesto"`
	ImporteTotal   string  `json:"importe_total"`
	BaseImponible  string  `json:"base_imponible"`
	Estado         string  `json:"estado"`
	Notas          *string `json:"notas_preparacion"`
}

type cuentaResp struct {
	ID       string      `json:"id"`
	Numero   int         `json:"numero"`
	Estado   string      `json:"estado"`
	Subtotal string      `json:"subtotal"`
	Impuesto string      `json:"impuesto"`
	Total    string      `json:"total"`
	Lineas   []lineaResp `json:"lineas"`
}

type repartoResp struct {
	Destinatario string  `json:"destinatario"`
	Telefono     string  `json:"telefono"`
	Direccion    string  `json:"direccion"`
	Referencia   *string `json:"referencia"`
	Canal        string  `json:"canal"`
	Estado       string  `json:"estado"`
}

type pedidoResp struct {
	ID                string       `json:"id"`
	SucursalID        string       `json:"sucursal_id"`
	ClaveIdempotencia string       `json:"clave_idempotencia"`
	MesaID            *string      `json:"mesa_id"`
	MeseroID          *string      `json:"mesero_id"`
	Tipo              string       `json:"tipo"`
	Estado            string       `json:"estado"`
	Moneda            string       `json:"moneda"`
	Comensales        int          `json:"comensales"`
	Notas             *string      `json:"notas"`
	Reparto           *repartoResp `json:"reparto"`
	Cuentas           []cuentaResp `json:"cuentas"`
}

type pedidosPagina struct {
	Items      []pedidoResp `json:"items"`
	NextCursor *string      `json:"nextCursor"`
}

// impuestoPedidosSQL garantiza un impuesto propio de estas pruebas con tasa del 18 % vigente desde
// el año 2000: no choca con las tasas que crean las pruebas de tasas_impuesto.
func impuestoPedidosSQL(t *testing.T) string {
	t.Helper()
	app(t)
	asSystem(t,
		`INSERT INTO sch_restaurante.impuestos(codigo,nombre) SELECT 'IGV_PED','IGV de las pruebas de pedidos'
		 WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.impuestos WHERE codigo='IGV_PED')`,
		`INSERT INTO sch_restaurante.tasas_impuesto(impuesto_id,tasa,vigente_desde)
		 SELECT i.id,18,DATE '2000-01-01' FROM sch_restaurante.impuestos i
		 WHERE i.codigo='IGV_PED' AND NOT EXISTS (SELECT 1 FROM sch_restaurante.tasas_impuesto x WHERE x.impuesto_id=i.id)`)
	return scalarStr(t, `SELECT id FROM sch_restaurante.impuestos WHERE codigo='IGV_PED'`)
}

func categoriaPedidosSQL(t *testing.T) string {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.categorias(nombre) SELECT 'Pedidos IT'
		WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.categorias WHERE nombre='Pedidos IT')`)
	return scalarStr(t, `SELECT id FROM sch_restaurante.categorias WHERE nombre='Pedidos IT'`)
}

// productoVentaSQL crea un producto gravado con IGV_PED (18 %) del tipo y precio dados y, si sede
// no está vacía, lo deja en esa sede con la disponibilidad indicada. Devuelve su id.
func productoVentaSQL(t *testing.T, sede, tipo, precio string, disponible bool) string {
	t.Helper()
	cat, imp := categoriaPedidosSQL(t), impuestoPedidosSQL(t)
	codigo := "PED-" + sufijo()
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.productos(categoria_id,codigo,nombre,tipo,precio_venta,afectacion_igv,impuesto_id)
		VALUES (%s,'%s','Producto %s','%s',%s,'10',%s)`, cat, codigo, codigo, tipo, precio, imp))
	if sede != "" {
		asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.productos_sucursal(producto_id,sucursal_id,disponible)
			SELECT p.id,s.id,%t FROM sch_restaurante.productos p, sch_restaurante.sucursales s
			WHERE p.codigo='%s' AND s.codigo='%s'`, disponible, codigo, sede))
	}
	return scalarStr(t, `SELECT id FROM sch_restaurante.productos WHERE codigo=$1`, codigo)
}

func tokenPedidos(t *testing.T, usuario, rol string, permisos []string, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, rol, permisos...)
	asegurarUsuario(t, usuario, "Pedidos2026Clave", rol, append([]string{"PRINCIPAL"}, sedes...)...)
	return token(t, usuario, "Pedidos2026Clave")
}

// cajeroPedidos: ver, crear y editar pedidos y autorizar descuentos en PRINCIPAL y las sedes dadas.
func cajeroPedidos(t *testing.T, sedes ...string) string {
	usuario := "ped.cajero"
	if len(sedes) > 0 {
		// Un usuario propio por conjunto de sedes: no se muta el cajero compartido.
		usuario += "." + strings.Join(sedes, "+")
	}
	return tokenPedidos(t, usuario, "T_PED_CAJERO",
		[]string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR", "DESCUENTO_AUTORIZAR"}, sedes...)
}

// meseroPedidos: ver, crear y editar pedidos, sin autorizar descuentos.
func meseroPedidos(t *testing.T, sedes ...string) string {
	return tokenPedidos(t, "ped.mesero", "T_PED_MESERO", []string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR"}, sedes...)
}

// lectorPedidos: solo ver pedidos.
func lectorPedidos(t *testing.T, sedes ...string) string {
	return tokenPedidos(t, "ped.lector", "T_PED_LECTOR", []string{"PEDIDO_VER"}, sedes...)
}

// mesaPedidosSQL crea una mesa nueva de la capacidad dada en la sede y devuelve su id.
func mesaPedidosSQL(t *testing.T, sede string, capacidad int) string {
	t.Helper()
	zona := "Zona pedidos " + sede
	crearZonaSQL(t, sede, zona)
	return crearMesaSQL(t, sede, zona, "PD-"+sufijo(), capacidad)
}

func sedeID(t *testing.T, codigo string) string { return idDe(t, "sucursales", "codigo", codigo) }

// abrirLlevar abre un pedido LLEVAR en la sede y devuelve la respuesta ya decodificada.
func abrirLlevar(t *testing.T, tk, sede string) pedidoResp {
	t.Helper()
	r := post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, sede)), tk)
	esperar(t, r, http.StatusCreated, "")
	var p pedidoResp
	r.json(t, &p)
	return p
}

func obtenerPedido(t *testing.T, tk, id string) pedidoResp {
	t.Helper()
	r := call(t, "GET", "/api/v1/pedidos/"+id, "", tk)
	esperar(t, r, http.StatusOK, "")
	var p pedidoResp
	r.json(t, &p)
	return p
}

// agregarConsumo agrega una línea y devuelve la respuesta cruda.
func agregarConsumo(t *testing.T, tk, pedido, cuenta, producto, cantidad string, extras string) resp {
	t.Helper()
	cuerpo := fmt.Sprintf(`{"producto_id":%q,"cuenta_id":%q,"cantidad":%q%s}`, producto, cuenta, cantidad, extras)
	return post(t, "/api/v1/pedidos/"+pedido+"/detalles", cuerpo, tk)
}

// pedidoSQL inserta, como sistema, un pedido LLEVAR con su cuenta nº 1 en la sede; devuelve ids.
func pedidoSQL(t *testing.T, sede string) (pedidoID, cuentaID string) {
	t.Helper()
	app(t)
	asSystem(t,
		fmt.Sprintf(`INSERT INTO sch_restaurante.pedidos(sucursal_id,clave_idempotencia,tipo,comensales)
			SELECT s.id,gen_random_uuid(),'LLEVAR',1 FROM sch_restaurante.sucursales s WHERE s.codigo='%s'`, sede))
	pedidoID = scalarStr(t, `SELECT p.id FROM sch_restaurante.pedidos p JOIN sch_restaurante.sucursales s ON s.id=p.sucursal_id
		WHERE s.codigo=$1 ORDER BY p.id DESC LIMIT 1`, sede)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.cuentas(pedido_id,numero) VALUES (%s,1)`, pedidoID))
	cuentaID = scalarStr(t, `SELECT id FROM sch_restaurante.cuentas WHERE pedido_id=$1::text::bigint`, pedidoID)
	return pedidoID, cuentaID
}
