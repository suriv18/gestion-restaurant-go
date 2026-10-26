package result

// Producto es un plato o artículo de la carta (maestro global). Precio y factor son decimales
// exactos como cadena, tal como los entrega PostgreSQL: precio con 2 decimales y factor con 6.
type Producto struct {
	ID            int64
	CategoriaID   int64
	Codigo        string
	Nombre        string
	Descripcion   *string
	Tipo          string
	ArticuloID    *int64
	FactorConsumo string
	PrecioVenta   string
	AfectacionIGV string
	ImpuestoID    *int64
	ImagenURL     *string
	EsActivo      bool
	Baja
	Auditoria
	// ConSede indica que la consulta pidió una sede: el ítem lleva entonces en_sucursal, que es
	// EnSucursal o null (nil) si el producto no tiene fila activa en esa sede.
	ConSede    bool
	EnSucursal *EnSucursal
}

// EnSucursal es el estado de un producto en una sede. PrecioSucursal es el precio propio de la
// sede (nil = usa el general); PrecioEfectivo es el que se cobra: el de la sede o, si no hay, el general.
type EnSucursal struct {
	Disponible     bool
	PrecioSucursal *string
	PrecioEfectivo string
	EstacionID     *int64
}

// ProductoSucursal es la fila de productos_sucursal: disponibilidad, precio opcional y estación
// de un producto en una sede. PrecioVenta nil = usa el precio general.
type ProductoSucursal struct {
	ID          int64
	ProductoID  int64
	SucursalID  int64
	EstacionID  *int64
	PrecioVenta *string
	Disponible  bool
	EsActivo    bool
	Auditoria
}

// PaginaProductos: NextCursor vacío indica la última página.
type PaginaProductos struct {
	Items      []Producto
	NextCursor string
}
