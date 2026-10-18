// Package query contiene los DTO de consultas del módulo identidad.
package query

import appctx "service-restaurant/internal/shared/application/context"

// AutenticarToken valida el bearer JWT de una solicitud.
type AutenticarToken struct {
	Token string
}

// ObtenerSesion describe al actor ya autenticado.
type ObtenerSesion struct {
	Actor appctx.Actor
}
