// Package request define los cuerpos HTTP del módulo produccion.
package request

// ComandaLinea: id de la línea del pedido y cantidad a enviar (decimal en cadena).
type ComandaLinea struct {
	DetallePedidoID string `json:"detalle_pedido_id"`
	Cantidad        string `json:"cantidad"`
}

// ComandaAlta: los ids viajan como cadenas (contrato). Un campo desconocido responde 400.
type ComandaAlta struct {
	EstacionID string         `json:"estacion_id"`
	Lineas     []ComandaLinea `json:"lineas"`
}

// PreparacionTransicion: estado destino y, por contrato, un motivo que hoy se ignora.
type PreparacionTransicion struct {
	Estado string  `json:"estado"`
	Motivo *string `json:"motivo,omitempty"`
}
