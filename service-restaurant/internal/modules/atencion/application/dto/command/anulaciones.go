package command

import appctx "service-restaurant/internal/shared/application/context"

// AnularDetalle anula una línea activa de una cuenta abierta junto con sus adicionales. El motivo
// es obligatorio (1 a 2000 caracteres).
type AnularDetalle struct {
	Intencion
	Actor     appctx.Actor
	PedidoID  int64
	DetalleID int64
	Motivo    string
}

// CancelarPedido cancela el pedido entero: sus cuentas abiertas, sus líneas activas y lo que cocina
// aún no entregó. El motivo es obligatorio (1 a 2000 caracteres).
type CancelarPedido struct {
	Intencion
	Actor    appctx.Actor
	PedidoID int64
	Motivo   string
}
