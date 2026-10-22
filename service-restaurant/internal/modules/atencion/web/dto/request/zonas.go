package request

// ZonaAlta: sucursal_id viaja como cadena (contrato); un campo desconocido responde 400.
type ZonaAlta struct {
	SucursalID         string `json:"sucursal_id"`
	Nombre             string `json:"nombre"`
	OrdenVisualizacion *int   `json:"orden_visualizacion"`
}

// ZonaEdicion: la sede de una zona no se edita (sucursal_id responde 400).
type ZonaEdicion struct {
	Nombre             *string `json:"nombre,omitempty"`
	OrdenVisualizacion *int    `json:"orden_visualizacion,omitempty"`
}
