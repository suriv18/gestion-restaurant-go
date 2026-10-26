package result

// Impuesto es un tributo aplicable a ventas. Solo se lee: no tiene API de escritura.
type Impuesto struct {
	ID          int64
	Codigo      string
	Nombre      string
	Descripcion *string
	EsActivo    bool
	Auditoria
}

// PaginaImpuestos: NextCursor vacío indica la última página.
type PaginaImpuestos struct {
	Items      []Impuesto
	NextCursor string
}

// TasaImpuesto es una vigencia [desde, hasta) de un impuesto. La tasa viaja como el decimal
// exacto que devuelve numeric(7,4)::text (siempre con 4 decimales, p. ej. "18.0000") y las
// fechas como YYYY-MM-DD, sin hora ni zona.
type TasaImpuesto struct {
	ID           int64
	ImpuestoID   int64
	Tasa         string
	VigenteDesde string
	VigenteHasta *string
	Fundamento   *string
	Auditoria
}

// PaginaTasas: NextCursor vacío indica la última página.
type PaginaTasas struct {
	Items      []TasaImpuesto
	NextCursor string
}
