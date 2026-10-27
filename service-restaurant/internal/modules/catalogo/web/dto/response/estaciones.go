package response

import "time"

type Estacion struct {
	ID             string    `json:"id"`
	SucursalID     string    `json:"sucursal_id"`
	Codigo         string    `json:"codigo"`
	Nombre         string    `json:"nombre"`
	EsActivo       bool      `json:"es_activo"`
	CreadoEn       time.Time `json:"creado_en"`
	CreadoPor      string    `json:"creado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
	ActualizadoPor string    `json:"actualizado_por"`
}

type EstacionesPagina struct {
	Items      []Estacion `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}
