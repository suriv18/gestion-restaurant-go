package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// Anulacion lleva el motivo ya validado y el usuario que anula.
type Anulacion struct {
	Motivo    string
	UsuarioID int64
}

// AnulacionStore escribe anulaciones y cancelaciones dentro de la transacción del comando. Los
// métodos que devuelven una referencia bloquean las filas hasta el final de la transacción.
type AnulacionStore interface {
	PedidoBloqueable
	// DetalleDelPedido lee la línea FOR UPDATE; found=false si no existe o es de otro pedido.
	DetalleDelPedido(ctx context.Context, pedidoID, detalleID int64) (DetalleRef, bool, error)
	// CuentaDelPedido lee la cuenta FOR UPDATE; found=false si no existe o es de otro pedido.
	CuentaDelPedido(ctx context.Context, pedidoID, cuentaID int64) (CuentaRef, bool, error)
	// CuentasDelPedido bloquea FOR UPDATE las cuentas del pedido, ordenadas por id.
	CuentasDelPedido(ctx context.Context, pedidoID int64) ([]CuentaRef, error)
	// CuentasConPago devuelve, de las cuentas dadas, las que tienen algún pago CONFIRMADO.
	CuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error)
	// CocinaEntregadaEnLinea indica si la línea o alguno de sus adicionales tiene una línea de
	// comanda ya ENTREGADO.
	CocinaEntregadaEnLinea(ctx context.Context, pedidoID, detalleID int64) (bool, error)
	// CocinaEntregadaEnPedido indica si alguna línea de comanda del pedido está ENTREGADO.
	CocinaEntregadaEnPedido(ctx context.Context, pedidoID int64) (bool, error)
	// AnularLinea anula la línea y sus adicionales activos, cancela sus líneas de comanda que aún no
	// se entregaron y devuelve la línea anulada.
	AnularLinea(ctx context.Context, pedidoID, detalleID int64, a Anulacion) (dtoresult.Linea, error)
	// CancelarPedido cancela el pedido, sus cuentas abiertas, sus líneas activas y sus líneas de
	// comanda no entregadas, y devuelve el pedido completo.
	CancelarPedido(ctx context.Context, pedidoID int64, a Anulacion) (dtoresult.PedidoDetalle, error)
}
