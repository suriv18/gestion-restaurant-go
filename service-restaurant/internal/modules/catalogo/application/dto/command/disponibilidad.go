package command

import appctx "service-restaurant/internal/shared/application/context"

// CambiarDisponibilidad habilita o pausa un producto en una sede (upsert de productos_sucursal).
// SucursalID 0 y Disponible nil significan "no enviado". Precio y Estacion son Campo: ausente =
// sin cambio; presente con Valor nil = volver al precio general / sin estación. El precio viaja
// como cadena decimal (sin float).
type CambiarDisponibilidad struct {
	Actor      appctx.Actor
	ProductoID int64
	SucursalID int64
	Disponible *bool
	Precio     Campo[string]
	Estacion   Campo[int64]
}
