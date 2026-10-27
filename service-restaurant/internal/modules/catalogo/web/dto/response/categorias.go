package response

import "time"

type Categoria struct {
	ID                 string    `json:"id"`
	Nombre             string    `json:"nombre"`
	Descripcion        *string   `json:"descripcion"`
	OrdenVisualizacion int       `json:"orden_visualizacion"`
	EsActivo           bool      `json:"es_activo"`
	CreadoEn           time.Time `json:"creado_en"`
	CreadoPor          string    `json:"creado_por"`
	ActualizadoEn      time.Time `json:"actualizado_en"`
	ActualizadoPor     string    `json:"actualizado_por"`
}

type CategoriasPagina struct {
	Items      []Categoria `json:"items"`
	NextCursor *string     `json:"nextCursor"`
}
