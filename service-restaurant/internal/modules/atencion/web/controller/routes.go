package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterRoutes monta zonas y mesas (todas exigen autenticación; el permiso fino lo verifica
// cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, zonas *ZonasHandler, protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /zonas", zonas.Listar)
	ruta("GET /zonas/{id}", zonas.Obtener)
}
