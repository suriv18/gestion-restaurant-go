package result

import "time"

// Pedido es la cabecera de un pedido. Los importes y cantidades de sus líneas son decimales
// exactos serializados como cadena.
type Pedido struct {
	ID                int64
	SucursalID        int64
	CodigoPublico     string
	ClaveIdempotencia string
	MesaID            *int64
	ReservaID         *int64
	ClienteID         *int64
	MeseroID          *int64
	Tipo              string
	Estado            string
	Moneda            string
	Comensales        int
	AbiertoEn         time.Time
	CerradoEn         *time.Time
	Notas             *string
	MotivoCancelacion *string
	VersionBloqueo    int
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
}

// Reparto son los datos de entrega de un pedido DELIVERY.
type Reparto struct {
	Destinatario string
	Telefono     string
	Direccion    string
	Referencia   *string
	Canal        string
	Estado       string
}

// Linea es un consumo del pedido como instantánea: descripción, precio y tasa quedan fijos aunque
// la carta cambie. ImporteTotal y BaseImponible los calcula la base (columnas generadas).
type Linea struct {
	ID               int64
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
	ImporteTotal     string
	BaseImponible    string
	Estado           string
	NotasPreparacion *string
	CreadoEn         time.Time
	CreadoPor        int64
	ActualizadoEn    time.Time
	ActualizadoPor   int64
}

// Cuenta agrupa líneas. Subtotal, Impuesto y Total suman solo las líneas activas.
type Cuenta struct {
	ID        int64
	Numero    int
	ClienteID *int64
	Etiqueta  *string
	Estado    string
	Subtotal  string
	Impuesto  string
	Total     string
	Lineas    []Linea
}

// PedidoDetalle es un pedido con su reparto (solo DELIVERY) y sus cuentas.
type PedidoDetalle struct {
	Pedido
	Reparto *Reparto
	Cuentas []Cuenta
}

// PaginaPedidos: NextCursor vacío indica la última página.
type PaginaPedidos struct {
	Items      []Pedido
	NextCursor string
}
