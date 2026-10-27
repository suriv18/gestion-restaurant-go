package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarCategorias pagina por cursor (id ascendente); Texto busca por nombre.
type ListarCategorias struct {
	Actor  appctx.Actor
	Activo *bool
	Texto  string
	Limite int
	Cursor string
}

type ObtenerCategoria struct {
	Actor appctx.Actor
	ID    int64
}
