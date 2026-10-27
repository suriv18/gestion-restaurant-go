package response

import "time"

// Empleado: los ids viajan como cadenas y las fechas como AAAA-MM-DD.
type Empleado struct {
	ID                string     `json:"id"`
	UsuarioID         *string    `json:"usuario_id"`
	Username          *string    `json:"username"`
	TipoDocumento     string     `json:"tipo_documento"`
	NumeroDocumento   string     `json:"numero_documento"`
	Nombres           string     `json:"nombres"`
	Apellidos         string     `json:"apellidos"`
	Telefono          *string    `json:"telefono"`
	Cargo             *string    `json:"cargo"`
	FechaIngreso      string     `json:"fecha_ingreso"`
	FechaCese         *string    `json:"fecha_cese"`
	EsActivo          bool       `json:"es_activo"`
	MotivoEliminacion *string    `json:"motivo_eliminacion"`
	EliminadoEn       *time.Time `json:"eliminado_en"`
	EliminadoPor      *string    `json:"eliminado_por"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type EmpleadosLista struct {
	Items      []Empleado `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

// EmpleadoResumen identifica al empleado de una persona del equipo.
type EmpleadoResumen struct {
	ID        string  `json:"id"`
	Nombres   string  `json:"nombres"`
	Apellidos string  `json:"apellidos"`
	Cargo     *string `json:"cargo"`
}
