package result

import "time"

// Empleado es personal operativo del restaurante; no reemplaza un módulo de planillas. Username
// es el usuario vinculado, si lo hay. Las fechas viajan como AAAA-MM-DD.
type Empleado struct {
	ID                int64
	UsuarioID         *int64
	Username          *string
	TipoDocumento     string
	NumeroDocumento   string
	Nombres           string
	Apellidos         string
	Telefono          *string
	Cargo             *string
	FechaIngreso      string
	FechaCese         *string
	EsActivo          bool
	MotivoEliminacion *string
	EliminadoEn       *time.Time
	EliminadoPor      *int64
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
}

// PaginaEmpleados: NextCursor vacío indica la última página.
type PaginaEmpleados struct {
	Items      []Empleado
	NextCursor string
}

// EmpleadoResumen identifica al empleado de un usuario en la lista del equipo.
type EmpleadoResumen struct {
	ID        int64
	Nombres   string
	Apellidos string
	Cargo     *string
}
