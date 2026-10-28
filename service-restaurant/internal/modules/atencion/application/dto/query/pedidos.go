package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarPedidos pagina por cursor (id ascendente) los pedidos de las sedes que el actor puede ver.
// Estado y Tipo vacíos no filtran.
type ListarPedidos struct {
	Actor      appctx.Actor
	SucursalID *int64
	Estado     string
	Tipo       string
	MesaID     *int64
	Limite     int
	Cursor     string
}

type ObtenerPedido struct {
	Actor appctx.Actor
	ID    int64
}
