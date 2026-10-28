package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterRoutes monta zonas, mesas y pedidos (todas exigen autenticación; el permiso fino lo
// verifica cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, zonas *ZonasHandler, mesas *MesasHandler, pedidos *PedidosHandler,
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
	ruta("POST /mesas", mesas.Crear)
	ruta("PATCH /mesas/{id}", mesas.Editar)
	ruta("POST /mesas/{id}/baja", mesas.Baja)
	ruta("POST /mesas/{id}/restauracion", mesas.Restauracion)

	ruta("GET /pedidos", pedidos.Listar)
	ruta("POST /pedidos", pedidos.Abrir)
	ruta("GET /pedidos/{id}", pedidos.Obtener)
	ruta("POST /pedidos/{id}/detalles", pedidos.AgregarDetalle)
	ruta("PATCH /pedidos/{id}/detalles/{detalle_id}", pedidos.EditarDetalle)
}
