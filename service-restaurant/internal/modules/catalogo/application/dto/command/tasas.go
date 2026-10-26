package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearTasaImpuesto da de alta una vigencia. Tasa y fechas llegan como texto: se validan sin float.
type CrearTasaImpuesto struct {
	Intencion
	Actor        appctx.Actor
	ImpuestoID   int64
	Tasa         string
	VigenteDesde string
	VigenteHasta *string
	Fundamento   *string
}

// EditarTasaImpuesto aplica solo los campos presentes. VigenteHasta null deja la vigencia abierta
// y Fundamento null lo vacía. El impuesto no se cambia.
type EditarTasaImpuesto struct {
	Actor        appctx.Actor
	ID           int64
	Tasa         *string
	VigenteDesde *string
	VigenteHasta Campo[string]
	Fundamento   Campo[string]
}
