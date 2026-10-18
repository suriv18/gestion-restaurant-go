// Package security define el contrato de autenticación; lo implementa el módulo identidad.
package security

import (
	"net/http"

	appctx "service-restaurant/internal/shared/application/context"
)

// Authenticator valida la solicitud y devuelve el actor verificado.
// Ante credenciales inválidas devuelve un *apperror.AppError de tipo KindUnauthorized.
type Authenticator interface {
	Authenticate(r *http.Request) (appctx.Actor, error)
}
