package outbound

import (
	"context"
	"fmt"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// Rechazos de almacenamiento propios de pedidos; ambos envuelven ErrDuplicado.
var (
	ErrMesaOcupada = fmt.Errorf("la mesa ya tiene un pedido abierto: %w", ErrDuplicado)
	ErrClaveEnUso  = fmt.Errorf("la clave de idempotencia ya fue usada por otro pedido: %w", ErrDuplicado)
)

// FiltroPedidos acota el listado a las sedes visibles (SedeIDs) y, opcionalmente, por estado,
// tipo y mesa.
type FiltroPedidos struct {
	SedeIDs     []int64
	Estado      string
	Tipo        string
	MesaID      *int64
	DespuesDeID int64
	Limite      int
}

// PedidoReader lee pedidos sin bloquear filas.
type PedidoReader interface {
	ListarPedidos(ctx context.Context, f FiltroPedidos) ([]dtoresult.Pedido, error)
	ObtenerPedido(ctx context.Context, id int64) (dtoresult.PedidoDetalle, bool, error)
}

// NuevoReparto son los datos de entrega a persistir. Canal nil usa el valor por defecto (PROPIO).
type NuevoReparto struct {
	Destinatario string
	Telefono     string
	Direccion    string
	Referencia   *string
	Canal        *string
}

// NuevoPedido: la cuenta nº 1 y, si Reparto no es nil, los datos de entrega se crean con él.
type NuevoPedido struct {
	SucursalID        int64
	ClaveIdempotencia string
	Tipo              string
	MesaID            *int64
	MeseroID          *int64
	ClienteID         *int64
	Comensales        int
	Notas             *string
	Reparto           *NuevoReparto
}

// MesaRef resume la mesa que un pedido de salón ocupa.
type MesaRef struct {
	SucursalID int64
	Capacidad  int
	Habilitada bool
	EsActiva   bool
}

// PedidoRef resume un pedido bloqueado. Cerrado incluye los cancelados.
type PedidoRef struct {
	ID         int64
	SucursalID int64
	Estado     string
	Cerrado    bool
}

// CuentaRef resume una cuenta bloqueada.
type CuentaRef struct {
	ID       int64
	PedidoID int64
	Estado   string
}

// ProductoVenta es lo que hace falta de la carta para fijar una línea. Precio ya incluye el precio
// propio de la sede si existe. Tasa es la vigente hoy del impuesto del producto ("" si no hay).
// Disponible es false si el producto no tiene fila activa en la sede.
type ProductoVenta struct {
	ID            int64
	Nombre        string
	Tipo          string
	AfectacionIGV string
	Precio        string
	Tasa          string
	EsActivo      bool
	Disponible    bool
}

// DetalleRef es una línea bloqueada. EnComanda indica que ya se envió a cocina.
type DetalleRef struct {
	ID             int64
	PedidoID       int64
	CuentaID       int64
	ProductoID     int64
	Cantidad       string
	PrecioUnitario string
	DescuentoTotal string
	Estado         string
	EnComanda      bool
	PadreID        *int64 // detalle_padre_id: no nulo si la línea es un adicional
}

// NuevoDetalle es la instantánea de una línea: todo salvo importe_total y base_imponible.
type NuevoDetalle struct {
	PedidoID         int64
	CuentaID         int64
	ProductoID       int64
	DetallePadreID   *int64
	Descripcion      string
	Cantidad         string
	PrecioUnitario   string
	DescuentoTotal   string
	AfectacionIGV    string
	TasaImpuesto     string
	NotasPreparacion *string
}

// CambiosDetalle: solo los campos presentes se modifican.
type CambiosDetalle struct {
	Cantidad       *string
	DescuentoTotal *string
	Instrucciones  CampoOpcional
}

// PedidoStore escribe pedidos dentro de la transacción del comando. Los métodos que devuelven una
// referencia bloquean la fila hasta el final de la transacción.
type PedidoStore interface {
	// MesaParaPedido lee la mesa en modo compartido, para que no se deshabilite durante el comando.
	MesaParaPedido(ctx context.Context, mesaID int64) (MesaRef, bool, error)
	// EmpleadoDeUsuario devuelve el empleado activo vinculado al usuario, o nil.
	EmpleadoDeUsuario(ctx context.Context, usuarioID int64) (*int64, error)
	// CrearPedido inserta pedido, cuenta nº 1 y reparto, y devuelve el pedido ya armado.
	CrearPedido(ctx context.Context, n NuevoPedido) (dtoresult.PedidoDetalle, error)
	// BloquearPedido lee el pedido FOR UPDATE.
	BloquearPedido(ctx context.Context, id int64) (PedidoRef, bool, error)
	// CuentaDelPedido lee la cuenta FOR UPDATE; found=false si no existe o es de otro pedido.
	CuentaDelPedido(ctx context.Context, pedidoID, cuentaID int64) (CuentaRef, bool, error)
	// ProductoParaVenta lee el producto con su precio y tasa vigentes en la sede (FOR SHARE).
	ProductoParaVenta(ctx context.Context, productoID, sedeID int64) (ProductoVenta, bool, error)
	// DetalleDelPedido lee la línea FOR UPDATE; found=false si no existe o es de otro pedido.
	DetalleDelPedido(ctx context.Context, pedidoID, detalleID int64) (DetalleRef, bool, error)
	InsertarDetalle(ctx context.Context, n NuevoDetalle) (dtoresult.Linea, error)
	ActualizarDetalle(ctx context.Context, id int64, c CambiosDetalle) (dtoresult.Linea, error)
}
