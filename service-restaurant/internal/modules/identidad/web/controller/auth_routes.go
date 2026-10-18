package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// errResultadoInvalido: un Result de valor cero es un error de programación, no un rechazo.
var errResultadoInvalido = errors.New("el caso de uso devolvió un Result inválido")

// RegisterAuthRoutes monta las rutas de acceso. protect exige un actor autenticado.
func RegisterAuthRoutes(mux *http.ServeMux, h *AuthHandler, protect func(http.Handler) http.Handler, logger *slog.Logger) {
	mux.Handle("POST /auth/login", handler.Adapt(logger, h.Login))
	mux.Handle("POST /auth/logout", protect(handler.Adapt(logger, h.Logout)))
	mux.Handle("GET /sesion", protect(handler.Adapt(logger, h.Sesion)))
}
