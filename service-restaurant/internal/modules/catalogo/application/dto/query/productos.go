package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarProductos pagina por cursor (id ascendente). Activo nil = solo activos (spec D6); Todos
// anula el filtro de estado. Texto busca por nombre (contiene) o por inicio del código.
type ListarProductos struct {
	Actor       appctx.Actor
	Activo      *bool
	Todos       bool
	CategoriaID *int64
	Texto       string
	Limite      int
	Cursor      string
}

type ObtenerProducto struct {
	Actor appctx.Actor
	ID    int64
}
