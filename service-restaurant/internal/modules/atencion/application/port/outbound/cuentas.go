package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// PedidoBloqueable bloquea un pedido FOR UPDATE. Lo comparten los puertos de escritura de pedidos,
// de división de cuentas y de anulación: todos empiezan por este bloqueo.
type PedidoBloqueable interface {
	// BloquearPedido lee el pedido FOR UPDATE; found=false si no existe.
	BloquearPedido(ctx context.Context, id int64) (PedidoRef, bool, error)
}

// LineaCuenta es una línea bloqueada con lo que la división de cuentas necesita.
type LineaCuenta struct {
	ID       int64
	CuentaID int64
	PadreID  *int64
	Estado   string
}

// NuevaCuenta es la cuenta que nace al dividir. Etiqueta nil la deja sin nombre.
type NuevaCuenta struct {
	PedidoID int64
	Etiqueta *string
}

// DivisionStore escribe la división de un pedido dentro de la transacción del comando. Los métodos
// que devuelven filas las bloquean hasta el final de la transacción.
type DivisionStore interface {
	PedidoBloqueable
	// LineasDelPedido bloquea FOR UPDATE todas las líneas del pedido, ordenadas por id.
	LineasDelPedido(ctx context.Context, pedidoID int64) ([]LineaCuenta, error)
	// CuentasDelPedido bloquea FOR UPDATE las cuentas del pedido, ordenadas por id.
	CuentasDelPedido(ctx context.Context, pedidoID int64) ([]CuentaRef, error)
	// CuentasConPago devuelve, de las cuentas dadas, las que tienen algún pago CONFIRMADO.
	CuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error)
	// CrearCuenta inserta la cuenta con el siguiente número del pedido y devuelve su id.
	CrearCuenta(ctx context.Context, n NuevaCuenta) (int64, error)
	// MoverLineas reasigna las líneas a la cuenta.
	MoverLineas(ctx context.Context, pedidoID, cuentaID int64, lineaIDs []int64) error
	// CuentaConLineas lee la cuenta con sus líneas y totales.
	CuentaConLineas(ctx context.Context, pedidoID, cuentaID int64) (dtoresult.Cuenta, error)
}
