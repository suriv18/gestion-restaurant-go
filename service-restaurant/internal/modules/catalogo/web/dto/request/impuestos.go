package request

// TasaImpuestoAlta: tasa y fechas viajan como cadenas (decimal exacto y AAAA-MM-DD).
type TasaImpuestoAlta struct {
	ImpuestoID   string  `json:"impuesto_id"`
	Tasa         string  `json:"tasa"`
	VigenteDesde string  `json:"vigente_desde"`
	VigenteHasta *string `json:"vigente_hasta"`
	Fundamento   *string `json:"fundamento"`
}

// TasaImpuestoEdicion no admite impuesto_id: la tasa no cambia de impuesto (se crea otra).
// vigente_hasta null deja la vigencia abierta; fundamento null lo vacía.
type TasaImpuestoEdicion struct {
	Tasa         *string          `json:"tasa,omitempty"`
	VigenteDesde *string          `json:"vigente_desde,omitempty"`
	VigenteHasta Opcional[string] `json:"vigente_hasta"`
	Fundamento   Opcional[string] `json:"fundamento"`
}
