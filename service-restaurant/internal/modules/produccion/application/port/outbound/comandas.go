package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
)

// PedidoRef resume un pedido bloqueado. Cerrado incluye los cancelados.
type PedidoRef struct {
	ID         int64
	SucursalID int64
	Cerrado    bool
}

// EstacionRef resume una estación de preparación.
type EstacionRef struct {
	ID         int64
	SucursalID int64
	EsActiva   bool
}

// LineaPedidoRef es una línea de pedido bloqueada, con lo que hace falta para enviarla.
// Enviado es lo ya enviado a cocina sin contar lo cancelado (decimal en cadena). EstacionID es la
// estación del producto en la sede del pedido, o nil si no tiene (p. ej. un SERVICIO).
type LineaPedidoRef struct {
	ID            int64
	CuentaID      int64
	CuentaEstado  string
	Estado        string
	Cantidad      string
	Enviado       string
	Descripcion   string
	Instrucciones *string
	EstacionID    *int64
}

// NuevaComandaLinea es la tanda de una línea que se envía: cantidad más la instantánea.
type NuevaComandaLinea struct {
	DetallePedidoID int64
	Cantidad        string
	Descripcion     string
	Instrucciones   *string
}

// NuevaComanda es un envío completo. ClaveIdempotencia es el UUID de la solicitud.
type NuevaComanda struct {
	PedidoID          int64
	SucursalID        int64
	EstacionID        int64
	ClaveIdempotencia string
	EnviadoPor        int64
	Lineas            []NuevaComandaLinea
}

// EnvioStore escribe un envío a cocina dentro de la transacción del comando. Los métodos que
// devuelven una referencia bloquean las filas hasta el final de la transacción.
type EnvioStore interface {
	// BloquearPedido lee el pedido FOR UPDATE.
	BloquearPedido(ctx context.Context, id int64) (PedidoRef, bool, error)
	// Estacion lee la estación en modo compartido.
	Estacion(ctx context.Context, id int64) (EstacionRef, bool, error)
	// LineasParaEnvio bloquea FOR UPDATE las líneas del pedido con esos ids, ordenadas por id. Las
	// que no existen o son de otro pedido simplemente no vienen en el resultado.
	LineasParaEnvio(ctx context.Context, pedidoID, sucursalID int64, ids []int64) ([]LineaPedidoRef, error)
	// CrearComanda inserta la comanda y sus líneas y la devuelve ya armada.
	CrearComanda(ctx context.Context, n NuevaComanda) (dtoresult.Comanda, error)
}
