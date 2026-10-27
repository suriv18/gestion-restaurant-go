package result

// Categoria agrupa los productos de la carta. Su baja es una desactivación simple (sin motivo).
type Categoria struct {
	ID          int64
	Nombre      string
	Descripcion *string
	Orden       int
	EsActivo    bool
	Auditoria
}

// PaginaCategorias: NextCursor vacío indica la última página.
type PaginaCategorias struct {
	Items      []Categoria
	NextCursor string
}
