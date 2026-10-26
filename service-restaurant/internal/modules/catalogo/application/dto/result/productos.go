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
}

// PaginaProductos: NextCursor vacío indica la última página.
type PaginaProductos struct {
	Items      []Producto
	NextCursor string
}
