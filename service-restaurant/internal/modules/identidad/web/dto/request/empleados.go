package request

// EmpleadoDatos son los datos personales del empleado en el alta conjunta de usuario y empleado.
type EmpleadoDatos struct {
	TipoDocumento   string  `json:"tipo_documento"`
	NumeroDocumento string  `json:"numero_documento"`
	Nombres         string  `json:"nombres"`
	Apellidos       string  `json:"apellidos"`
	Telefono        *string `json:"telefono"`
	Cargo           *string `json:"cargo"`
}

// EmpleadoAlta: fecha_ingreso (AAAA-MM-DD) ausente = hoy; usuario_id (cadena) ausente = sin acceso.
type EmpleadoAlta struct {
	EmpleadoDatos
	FechaIngreso *string `json:"fecha_ingreso"`
	UsuarioID    *string `json:"usuario_id"`
}

// EmpleadoEdicion: usuario_id null desvincula al usuario; fecha_cese null borra el cese.
type EmpleadoEdicion struct {
	TipoDocumento   *string          `json:"tipo_documento,omitempty"`
	NumeroDocumento *string          `json:"numero_documento,omitempty"`
	Nombres         *string          `json:"nombres,omitempty"`
	Apellidos       *string          `json:"apellidos,omitempty"`
	FechaIngreso    *string          `json:"fecha_ingreso,omitempty"`
	Telefono        Opcional[string] `json:"telefono"`
	Cargo           Opcional[string] `json:"cargo"`
	FechaCese       Opcional[string] `json:"fecha_cese"`
	UsuarioID       Opcional[string] `json:"usuario_id"`
}
