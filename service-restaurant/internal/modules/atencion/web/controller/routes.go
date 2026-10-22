package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterRoutes monta zonas y mesas (todas exigen autenticación; el permiso fino lo verifica
// cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, zonas *ZonasHandler, mesas *MesasHandler,
	protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /zonas", zonas.Listar)
	ruta("POST /zonas", zonas.Crear)
	ruta("GET /zonas/{id}", zonas.Obtener)
	ruta("PATCH /zonas/{id}", zonas.Editar)
	ruta("POST /zonas/{id}/baja", zonas.Baja)
	ruta("POST /zonas/{id}/restauracion", zonas.Restauracion)

	ruta("GET /mesas", mesas.Listar)
	ruta("GET /mesas/{id}", mesas.Obtener)
}
