package request

// NegocioEdicion: el RUC y el singleton no se editan (un campo desconocido responde 400).
// Los campos anulables usan Opcional para distinguir "no enviado" de null.
type NegocioEdicion struct {
	RazonSocial     *string          `json:"razon_social,omitempty"`
	NombreComercial *string          `json:"nombre_comercial,omitempty"`
	DireccionFiscal *string          `json:"direccion_fiscal,omitempty"`
	Ubigeo          Opcional[string] `json:"ubigeo"`
	Telefono        Opcional[string] `json:"telefono"`
	Email           Opcional[string] `json:"email"`
	Moneda          *string          `json:"moneda,omitempty"`
	ZonaHoraria     *string          `json:"zona_horaria,omitempty"`
}
