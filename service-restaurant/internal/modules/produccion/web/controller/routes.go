package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterRoutes monta las rutas de cocina (todas exigen autenticación; el permiso fino lo verifica
// cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, comandas *ComandasHandler, protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("POST /pedidos/{id}/comandas", comandas.Enviar)
	ruta("POST /comanda-detalles/{id}/transiciones", comandas.Transicionar)
}
