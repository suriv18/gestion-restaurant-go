package result

// Estacion es un puesto de preparación (cocina, barra...) de una sede. Su baja es una
// desactivación simple (sin motivo).
type Estacion struct {
	ID         int64
	SucursalID int64
	Codigo     string
	Nombre     string
	EsActivo   bool
	Auditoria
}

// PaginaEstaciones: NextCursor vacío indica la última página.
type PaginaEstaciones struct {
	Items      []Estacion
	NextCursor string
}
