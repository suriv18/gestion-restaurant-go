package request

// EstacionAlta: sucursal_id viaja como cadena (id BIGINT del contrato).
type EstacionAlta struct {
	SucursalID string `json:"sucursal_id"`
	Codigo     string `json:"codigo"`
	Nombre     string `json:"nombre"`
}

// EstacionEdicion no admite sucursal_id: una estación no cambia de sede (campo desconocido → 400).
type EstacionEdicion struct {
	Codigo *string `json:"codigo,omitempty"`
	Nombre *string `json:"nombre,omitempty"`
}
