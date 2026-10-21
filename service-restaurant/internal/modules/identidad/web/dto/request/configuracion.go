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

// SucursalAlta: negocio_id no existe aquí; un campo desconocido responde 400.
type SucursalAlta struct {
	Codigo                     string  `json:"codigo"`
	Nombre                     string  `json:"nombre"`
	Direccion                  string  `json:"direccion"`
	Ubigeo                     *string `json:"ubigeo"`
	CodigoEstablecimientoSunat *string `json:"codigo_establecimiento_sunat"`
	Telefono                   *string `json:"telefono"`
}

type SucursalEdicion struct {
	Codigo                     *string          `json:"codigo,omitempty"`
	Nombre                     *string          `json:"nombre,omitempty"`
	Direccion                  *string          `json:"direccion,omitempty"`
	Ubigeo                     Opcional[string] `json:"ubigeo"`
	CodigoEstablecimientoSunat Opcional[string] `json:"codigo_establecimiento_sunat"`
	Telefono                   Opcional[string] `json:"telefono"`
}
