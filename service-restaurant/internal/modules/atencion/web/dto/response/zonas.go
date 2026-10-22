// Package response define las respuestas HTTP del módulo atencion; los ids viajan como cadenas.
package response

import "time"

type Zona struct {
	ID                 string    `json:"id"`
	SucursalID         string    `json:"sucursal_id"`
	Nombre             string    `json:"nombre"`
	OrdenVisualizacion int       `json:"orden_visualizacion"`
	EsActivo           bool      `json:"es_activo"`
	CreadoEn           time.Time `json:"creado_en"`
	CreadoPor          string    `json:"creado_por"`
	ActualizadoEn      time.Time `json:"actualizado_en"`
	ActualizadoPor     string    `json:"actualizado_por"`
}

type ZonasPagina struct {
	Items      []Zona  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
