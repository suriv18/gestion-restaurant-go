package response

import "time"

type Impuesto struct {
	ID             string    `json:"id"`
	Codigo         string    `json:"codigo"`
	Nombre         string    `json:"nombre"`
	Descripcion    *string   `json:"descripcion"`
	EsActivo       bool      `json:"es_activo"`
	CreadoEn       time.Time `json:"creado_en"`
	CreadoPor      string    `json:"creado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
	ActualizadoPor string    `json:"actualizado_por"`
}

type ImpuestosPagina struct {
	Items      []Impuesto `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

// TasaImpuesto: tasa con 4 decimales ("18.0000") y fechas AAAA-MM-DD.
type TasaImpuesto struct {
	ID             string    `json:"id"`
	ImpuestoID     string    `json:"impuesto_id"`
	Tasa           string    `json:"tasa"`
	VigenteDesde   string    `json:"vigente_desde"`
	VigenteHasta   *string   `json:"vigente_hasta"`
	Fundamento     *string   `json:"fundamento"`
	CreadoEn       time.Time `json:"creado_en"`
	CreadoPor      string    `json:"creado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
	ActualizadoPor string    `json:"actualizado_por"`
}

type TasasPagina struct {
	Items      []TasaImpuesto `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}
