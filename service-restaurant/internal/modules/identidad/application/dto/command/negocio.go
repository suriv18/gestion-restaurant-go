package command

import appctx "service-restaurant/internal/shared/application/context"

// Campo es un valor opcional de un PATCH: Presente distingue "no enviado" de "enviado como
// null" (vaciar). Valor es nil cuando se envió null.
type Campo[T any] struct {
	Presente bool
	Valor    *T
}

// EditarNegocio aplica solo los campos presentes. El RUC y el singleton no se editan.
type EditarNegocio struct {
	Actor           appctx.Actor
	RazonSocial     *string
	NombreComercial *string
	DireccionFiscal *string
	Moneda          *string
	ZonaHoraria     *string
	Ubigeo          Campo[string]
	Telefono        Campo[string]
	Email           Campo[string]
}
