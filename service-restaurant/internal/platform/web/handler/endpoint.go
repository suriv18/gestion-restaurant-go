// Package handler adapta endpoints que devuelven error a http.Handler.
package handler

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/exception"
)

// Endpoint es un handler que retorna error.
type Endpoint func(w http.ResponseWriter, r *http.Request) error

// Adapt convierte un Endpoint en http.Handler; el error se traduce a problem+json.
func Adapt(logger *slog.Logger, e Endpoint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := e(w, r); err != nil {
			exception.Handle(w, r, logger, err)
		}
	})
}
