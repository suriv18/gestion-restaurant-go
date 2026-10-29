// Package response define los cuerpos de respuesta HTTP del módulo produccion.
package response

import "time"

type ComandaLinea struct {
	ID                string     `json:"id"`
	ComandaID         string     `json:"comanda_id"`
	PedidoID          string     `json:"pedido_id"`
	DetallePedidoID   string     `json:"detalle_pedido_id"`
	Cantidad          string     `json:"cantidad"`
	Descripcion       string     `json:"descripcion"`
	Instrucciones     *string    `json:"instrucciones"`
	Estado            string     `json:"estado"`
	IniciadoEn        *time.Time `json:"iniciado_en"`
	ListoEn           *time.Time `json:"listo_en"`
	EntregadoEn       *time.Time `json:"entregado_en"`
	MotivoCancelacion *string    `json:"motivo_cancelacion"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type Comanda struct {
	ID                string         `json:"id"`
	PedidoID          string         `json:"pedido_id"`
	SucursalID        string         `json:"sucursal_id"`
	EstacionID        string         `json:"estacion_id"`
	MesaID            *string        `json:"mesa_id"`
	ClaveIdempotencia string         `json:"clave_idempotencia"`
	EnviadoEn         time.Time      `json:"enviado_en"`
	EnviadoPor        string         `json:"enviado_por"`
	Notas             *string        `json:"notas"`
	Lineas            []ComandaLinea `json:"lineas"`
	CreadoEn          time.Time      `json:"creado_en"`
	CreadoPor         string         `json:"creado_por"`
	ActualizadoEn     time.Time      `json:"actualizado_en"`
	ActualizadoPor    string         `json:"actualizado_por"`
}

type ComandasPagina struct {
	Items      []Comanda `json:"items"`
	NextCursor *string   `json:"nextCursor"`
}
