// Package result agrupa los resultados de los casos de uso de produccion.
package result

import "time"

// ComandaLinea es una línea de comanda: una tanda de una línea de pedido enviada a una estación.
// Descripcion e Instrucciones son la instantánea de la línea al enviar.
type ComandaLinea struct {
	ID                int64
	ComandaID         int64
	PedidoID          int64
	DetallePedidoID   int64
	Cantidad          string
	Descripcion       string
	Instrucciones     *string
	Estado            string
	IniciadoEn        *time.Time
	ListoEn           *time.Time
	EntregadoEn       *time.Time
	MotivoCancelacion *string
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
}

// Comanda es un envío a una estación con sus líneas. No guarda estado propio: se deriva de ellas.
type Comanda struct {
	ID                int64
	PedidoID          int64
	SucursalID        int64
	EstacionID        int64
	MesaID            *int64
	ClaveIdempotencia string
	EnviadoEn         time.Time
	EnviadoPor        int64
	Notas             *string
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
	Lineas            []ComandaLinea
}

// PaginaComandas es una página de la cola; NextCursor vacío indica la última.
type PaginaComandas struct {
	Items      []Comanda
	NextCursor string
}
