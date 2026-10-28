package response

import "time"

// Pedido es la cabecera (esquema Pedidos del contrato). version_bloqueo es de solo lectura.
type Pedido struct {
	ID                string     `json:"id"`
	SucursalID        string     `json:"sucursal_id"`
	CodigoPublico     string     `json:"codigo_publico"`
	ClaveIdempotencia string     `json:"clave_idempotencia"`
	MesaID            *string    `json:"mesa_id"`
	ReservaID         *string    `json:"reserva_id"`
	ClienteID         *string    `json:"cliente_id"`
	MeseroID          *string    `json:"mesero_id"`
	Tipo              string     `json:"tipo"`
	Estado            string     `json:"estado"`
	Moneda            string     `json:"moneda"`
	Comensales        int        `json:"comensales"`
	AbiertoEn         time.Time  `json:"abierto_en"`
	CerradoEn         *time.Time `json:"cerrado_en"`
	Notas             *string    `json:"notas"`
	MotivoCancelacion *string    `json:"motivo_cancelacion"`
	VersionBloqueo    int        `json:"version_bloqueo"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

// Linea es un consumo (esquema DetallesPedido del contrato).
type Linea struct {
	ID               string    `json:"id"`
	PedidoID         string    `json:"pedido_id"`
	CuentaID         string    `json:"cuenta_id"`
	ProductoID       string    `json:"producto_id"`
	DetallePadreID   *string   `json:"detalle_padre_id"`
	Descripcion      string    `json:"descripcion"`
	Cantidad         string    `json:"cantidad"`
	PrecioUnitario   string    `json:"precio_unitario"`
	DescuentoTotal   string    `json:"descuento_total"`
	AfectacionIGV    string    `json:"afectacion_igv"`
	TasaImpuesto     string    `json:"tasa_impuesto"`
	ImporteTotal     string    `json:"importe_total"`
	BaseImponible    string    `json:"base_imponible"`
	Estado           string    `json:"estado"`
	NotasPreparacion *string   `json:"notas_preparacion"`
	CreadoEn         time.Time `json:"creado_en"`
	CreadoPor        string    `json:"creado_por"`
	ActualizadoEn    time.Time `json:"actualizado_en"`
	ActualizadoPor   string    `json:"actualizado_por"`
}

type Cuenta struct {
	ID        string  `json:"id"`
	Numero    int     `json:"numero"`
	ClienteID *string `json:"cliente_id"`
	Etiqueta  *string `json:"etiqueta"`
	Estado    string  `json:"estado"`
	Subtotal  string  `json:"subtotal"`
	Impuesto  string  `json:"impuesto"`
	Total     string  `json:"total"`
	Lineas    []Linea `json:"lineas"`
}

type Reparto struct {
	Destinatario string  `json:"destinatario"`
	Telefono     string  `json:"telefono"`
	Direccion    string  `json:"direccion"`
	Referencia   *string `json:"referencia"`
	Canal        string  `json:"canal"`
	Estado       string  `json:"estado"`
}

// PedidoDetalle: la cabecera más el reparto (null salvo DELIVERY) y las cuentas con sus líneas.
type PedidoDetalle struct {
	Pedido
	Reparto *Reparto `json:"reparto"`
	Cuentas []Cuenta `json:"cuentas"`
}

type PedidosPagina struct {
	Items      []Pedido `json:"items"`
	NextCursor *string  `json:"nextCursor"`
}
