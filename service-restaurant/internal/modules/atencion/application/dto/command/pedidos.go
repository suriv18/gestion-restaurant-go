package command

import appctx "service-restaurant/internal/shared/application/context"

// Reparto son los datos de entrega que acompañan a un pedido DELIVERY.
type Reparto struct {
	Destinatario string
	Telefono     string
	Direccion    string
	Referencia   *string
	Canal        *string
}

// AbrirPedido crea el pedido y su cuenta nº 1. Tipo es SALON, LLEVAR o DELIVERY; MesaID solo en
// SALON y Reparto solo en DELIVERY.
type AbrirPedido struct {
	Intencion
	Actor      appctx.Actor
	SucursalID int64
	Tipo       string
	MesaID     *int64
	Comensales int
	ClienteID  *int64
	Notas      *string
	Reparto    *Reparto
}

// AgregarDetalle agrega una línea a una cuenta abierta. Cantidad y DescuentoMonto son decimales
// exactos en cadena; el servidor fija descripción, precio y tasa desde la carta.
type AgregarDetalle struct {
	Intencion
	Actor          appctx.Actor
	PedidoID       int64
	CuentaID       int64
	ProductoID     int64
	Cantidad       string
	DetallePadreID *int64
	Instrucciones  *string
	DescuentoMonto *string
}

// EditarDetalle modifica solo lo enviado de una línea activa. Instrucciones distingue "no
// enviado" de "enviado como null" (vaciar).
type EditarDetalle struct {
	Actor          appctx.Actor
	PedidoID       int64
	DetalleID      int64
	Cantidad       *string
	Instrucciones  Campo[string]
	DescuentoMonto *string
}
