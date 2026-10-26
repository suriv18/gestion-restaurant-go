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
	// EnSucursal solo se emite si la consulta pidió sucursal_id: apunta a un puntero nulo (null)
	// cuando el producto no tiene fila en esa sede; nil (campo omitido) si no se pidió sede.
	EnSucursal **EnSucursal `json:"en_sucursal,omitempty"`
}

type ProductosPagina struct {
	Items      []Producto `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

// EnSucursal es el estado de un producto en la sede consultada. precio_sucursal es el precio propio
// de la sede (null = usa el general); precio_efectivo es el que se cobra (el de la sede o el general).
type EnSucursal struct {
	Disponible     bool    `json:"disponible"`
	PrecioSucursal *string `json:"precio_sucursal"`
	PrecioEfectivo string  `json:"precio_efectivo"`
	EstacionID     *string `json:"estacion_id"`
}

// ProductoSucursal es la fila de disponibilidad de un producto en una sede.
type ProductoSucursal struct {
	ID             string    `json:"id"`
	ProductoID     string    `json:"producto_id"`
	SucursalID     string    `json:"sucursal_id"`
	EstacionID     *string   `json:"estacion_id"`
	PrecioVenta    *string   `json:"precio_venta"`
	Disponible     bool      `json:"disponible"`
	EsActivo       bool      `json:"es_activo"`
	CreadoEn       time.Time `json:"creado_en"`
	CreadoPor      string    `json:"creado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
	ActualizadoPor string    `json:"actualizado_por"`
}
