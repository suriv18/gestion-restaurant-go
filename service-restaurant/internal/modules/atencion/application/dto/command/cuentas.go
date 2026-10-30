package command

import appctx "service-restaurant/internal/shared/application/context"

// DividirCuenta mueve líneas activas de las cuentas abiertas de un pedido a una cuenta nueva del
// mismo pedido. Los adicionales viajan con su línea padre. NombreCuenta es la etiqueta opcional.
type DividirCuenta struct {
	Intencion
	Actor        appctx.Actor
	PedidoID     int64
	DetalleIDs   []int64
	NombreCuenta *string
}
