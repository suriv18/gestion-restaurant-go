package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterConfiguracionRoutes monta negocio y sucursales (todas exigen autenticación; el
// permiso fino lo verifica cada caso de uso).
func RegisterConfiguracionRoutes(mux *http.ServeMux, negocio *NegocioHandler, sucursales *SucursalesHandler,
	protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /negocio", negocio.Obtener)
	ruta("PATCH /negocio", negocio.Editar)

	ruta("GET /sucursales", sucursales.Listar)
	ruta("GET /sucursales/{id}", sucursales.Obtener)
}
