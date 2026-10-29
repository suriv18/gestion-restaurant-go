package command

import appctx "service-restaurant/internal/shared/application/context"

// LineaEnvio es una línea del pedido y la cantidad que se envía ahora a cocina (decimal en cadena).
type LineaEnvio struct {
	DetallePedidoID int64
	Cantidad        string
}

// EnviarComanda envía a una estación las cantidades indicadas de las líneas de un pedido.
type EnviarComanda struct {
	Intencion
	Actor      appctx.Actor
	PedidoID   int64
	EstacionID int64
	Lineas     []LineaEnvio
}

// TransicionarLinea mueve una línea de comanda a su estado siguiente. Motivo se acepta por contrato
// y se ignora hasta la rebanada 3 (cancelaciones).
type TransicionarLinea struct {
	Actor   appctx.Actor
	LineaID int64
	Estado  string
	Motivo  *string
}
