package response

import "time"

// Producto: precio_venta y factor_consumo son cadenas normalizadas como las entrega PostgreSQL
// (precio con 2 decimales, factor con 6).
type Producto struct {
	ID                string     `json:"id"`
	CategoriaID       string     `json:"categoria_id"`
	Codigo            string     `json:"codigo"`
	Nombre            string     `json:"nombre"`
	Descripcion       *string    `json:"descripcion"`
	Tipo              string     `json:"tipo"`
	ArticuloID        *string    `json:"articulo_id"`
	FactorConsumo     string     `json:"factor_consumo"`
	PrecioVenta       string     `json:"precio_venta"`
	AfectacionIGV     string     `json:"afectacion_igv"`
	ImpuestoID        *string    `json:"impuesto_id"`
	ImagenURL         *string    `json:"imagen_url"`
	EsActivo          bool       `json:"es_activo"`
	MotivoEliminacion *string    `json:"motivo_eliminacion"`
	EliminadoEn       *time.Time `json:"eliminado_en"`
	EliminadoPor      *string    `json:"eliminado_por"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type ProductosPagina struct {
	Items      []Producto `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}
