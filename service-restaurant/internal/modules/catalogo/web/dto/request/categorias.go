package request

// CategoriaAlta: orden_visualizacion ausente equivale a 0.
type CategoriaAlta struct {
	Nombre             string  `json:"nombre"`
	Descripcion        *string `json:"descripcion"`
	OrdenVisualizacion *int    `json:"orden_visualizacion"`
}

// CategoriaEdicion: la descripción admite null para vaciarla.
type CategoriaEdicion struct {
	Nombre             *string          `json:"nombre,omitempty"`
	Descripcion        Opcional[string] `json:"descripcion"`
	OrdenVisualizacion *int             `json:"orden_visualizacion,omitempty"`
}
