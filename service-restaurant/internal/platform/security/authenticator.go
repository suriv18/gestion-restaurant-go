// Package security define el contrato de autenticación; la implementación llega en I1.
package security

import (
	"net/http"

	appctx "service-restaurant/internal/shared/application/context"
)

// Authenticator valida la solicitud y devuelve el actor verificado.
type Authenticator interface {
	Authenticate(r *http.Request) (appctx.Actor, error)
}
