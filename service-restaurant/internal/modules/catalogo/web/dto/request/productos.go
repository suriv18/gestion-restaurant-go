package request

// ProductoAlta: los ids y los decimales viajan como cadenas (contrato). categoria_id y
// precio_venta son obligatorios; factor_consumo ausente equivale a "1".
type ProductoAlta struct {
	CategoriaID   string  `json:"categoria_id"`
	Codigo        string  `json:"codigo"`
	Nombre        string  `json:"nombre"`
	Descripcion   *string `json:"descripcion"`
	Tipo          string  `json:"tipo"`
	ArticuloID    *string `json:"articulo_id"`
	FactorConsumo *string `json:"factor_consumo"`
	PrecioVenta   string  `json:"precio_venta"`
	AfectacionIGV string  `json:"afectacion_igv"`
	ImpuestoID    *string `json:"impuesto_id"`
	ImagenURL     *string `json:"imagen_url"`
}

// ProductoEdicion: cualquier campo del alta. descripcion, articulo_id, impuesto_id e imagen_url
// admiten null para vaciarse.
type ProductoEdicion struct {
	CategoriaID   *string          `json:"categoria_id,omitempty"`
	Codigo        *string          `json:"codigo,omitempty"`
	Nombre        *string          `json:"nombre,omitempty"`
	Descripcion   Opcional[string] `json:"descripcion"`
	Tipo          *string          `json:"tipo,omitempty"`
	ArticuloID    Opcional[string] `json:"articulo_id"`
	FactorConsumo *string          `json:"factor_consumo,omitempty"`
	PrecioVenta   *string          `json:"precio_venta,omitempty"`
	AfectacionIGV *string          `json:"afectacion_igv,omitempty"`
	ImpuestoID    Opcional[string] `json:"impuesto_id"`
	ImagenURL     Opcional[string] `json:"imagen_url"`
}
