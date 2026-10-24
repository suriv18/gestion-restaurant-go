package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterRoutes monta clientes y proveedores (todas exigen autenticación; el permiso fino lo
// verifica cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, clientes *ClientesHandler, proveedores *ProveedoresHandler,
	protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /clientes", clientes.Listar)
	ruta("POST /clientes", clientes.Crear)
	ruta("GET /clientes/{id}", clientes.Obtener)
	ruta("PATCH /clientes/{id}", clientes.Editar)
	ruta("POST /clientes/{id}/baja", clientes.Baja)
	ruta("POST /clientes/{id}/restauracion", clientes.Restauracion)

	ruta("GET /proveedores", proveedores.Listar)
	ruta("POST /proveedores", proveedores.Crear)
	ruta("GET /proveedores/{id}", proveedores.Obtener)
	ruta("PATCH /proveedores/{id}", proveedores.Editar)
	ruta("POST /proveedores/{id}/baja", proveedores.Baja)
	ruta("POST /proveedores/{id}/restauracion", proveedores.Restauracion)
}
