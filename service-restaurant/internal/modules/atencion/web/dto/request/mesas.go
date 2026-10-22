package request

// MesaAlta: los ids viajan como cadenas (contrato); las posiciones, como decimales exactos en
// cadena. Un campo desconocido responde 400.
type MesaAlta struct {
	SucursalID string  `json:"sucursal_id"`
	ZonaID     string  `json:"zona_id"`
	NumeroMesa string  `json:"numero_mesa"`
	Capacidad  int     `json:"capacidad"`
	Habilitada *bool   `json:"habilitada"`
	PosicionX  *string `json:"posicion_x"`
	PosicionY  *string `json:"posicion_y"`
}

// MesaEdicion: la sede de una mesa no se edita (sucursal_id responde 400).
type MesaEdicion struct {
	ZonaID     *string          `json:"zona_id,omitempty"`
	NumeroMesa *string          `json:"numero_mesa,omitempty"`
	Capacidad  *int             `json:"capacidad,omitempty"`
	Habilitada *bool            `json:"habilitada,omitempty"`
	PosicionX  Opcional[string] `json:"posicion_x"`
	PosicionY  Opcional[string] `json:"posicion_y"`
}
