package request

// Disponibilidad es el cuerpo de PUT /productos/{id}/disponibilidad. sucursal_id y disponible son
// obligatorios (si faltan, el caso de uso responde 422). precio_venta y estacion_id son una
// extensión del contrato: ausente = sin cambio; null = precio general / sin estación.
type Disponibilidad struct {
	SucursalID  string           `json:"sucursal_id"`
	Disponible  *bool            `json:"disponible"`
	PrecioVenta Opcional[string] `json:"precio_venta"`
	EstacionID  Opcional[string] `json:"estacion_id"`
}
