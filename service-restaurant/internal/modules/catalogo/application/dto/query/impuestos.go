package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarImpuestos pagina por cursor (id ascendente).
type ListarImpuestos struct {
	Actor  appctx.Actor
	Activo *bool
	Limite int
	Cursor string
}

type ObtenerImpuesto struct {
	Actor appctx.Actor
	ID    int64
}

// ListarTasasImpuesto pagina por cursor (id ascendente); ImpuestoID 0 no filtra.
type ListarTasasImpuesto struct {
	Actor      appctx.Actor
	ImpuestoID int64
	Limite     int
	Cursor     string
}

type ObtenerTasaImpuesto struct {
	Actor appctx.Actor
	ID    int64
}
