package request

// Reparto: datos de entrega de un pedido DELIVERY (esquema DeliveryDatos del contrato).
type Reparto struct {
	Destinatario string  `json:"destinatario"`
	Telefono     string  `json:"telefono"`
	Direccion    string  `json:"direccion"`
	Referencia   *string `json:"referencia,omitempty"`
	Canal        *string `json:"canal,omitempty"`
}

// PedidoAlta: los ids viajan como cadenas (contrato). Un campo desconocido responde 400.
type PedidoAlta struct {
	SucursalID string   `json:"sucursal_id"`
	Tipo       string   `json:"tipo"`
	MesaID     *string  `json:"mesa_id,omitempty"`
	Comensales int      `json:"comensales"`
	ClienteID  *string  `json:"cliente_id,omitempty"`
	Notas      *string  `json:"notas,omitempty"`
	Reparto    *Reparto `json:"reparto,omitempty"`
}

// ConsumoAlta: cantidad y descuento_monto son decimales exactos en cadena.
type ConsumoAlta struct {
	ProductoID     string  `json:"producto_id"`
	CuentaID       string  `json:"cuenta_id"`
	Cantidad       string  `json:"cantidad"`
	DetallePadreID *string `json:"detalle_padre_id,omitempty"`
	Instrucciones  *string `json:"instrucciones,omitempty"`
	DescuentoMonto *string `json:"descuento_monto,omitempty"`
}

// ConsumoEdicion: instrucciones distingue ausente de null (vaciar).
type ConsumoEdicion struct {
	Cantidad       *string          `json:"cantidad,omitempty"`
	Instrucciones  Opcional[string] `json:"instrucciones"`
	DescuentoMonto *string          `json:"descuento_monto,omitempty"`
}
