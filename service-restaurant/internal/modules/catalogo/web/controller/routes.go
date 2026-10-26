package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// Handlers reúne los controladores del módulo; cada entidad nueva de la carta añade un campo.
type Handlers struct {
	Clientes    *ClientesHandler
	Proveedores *ProveedoresHandler
	Categorias  *CategoriasHandler
	Impuestos   *ImpuestosHandler
	Estaciones  *EstacionesHandler
}

// RegisterRoutes monta clientes, proveedores y la carta (todas exigen autenticación; el permiso
// fino lo verifica cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, h Handlers, protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /clientes", h.Clientes.Listar)
	ruta("POST /clientes", h.Clientes.Crear)
	ruta("GET /clientes/{id}", h.Clientes.Obtener)
	ruta("PATCH /clientes/{id}", h.Clientes.Editar)
	ruta("POST /clientes/{id}/baja", h.Clientes.Baja)
	ruta("POST /clientes/{id}/restauracion", h.Clientes.Restauracion)

	ruta("GET /proveedores", h.Proveedores.Listar)
	ruta("POST /proveedores", h.Proveedores.Crear)
	ruta("GET /proveedores/{id}", h.Proveedores.Obtener)
	ruta("PATCH /proveedores/{id}", h.Proveedores.Editar)
	ruta("POST /proveedores/{id}/baja", h.Proveedores.Baja)
	ruta("POST /proveedores/{id}/restauracion", h.Proveedores.Restauracion)

	ruta("GET /categorias", h.Categorias.Listar)
	ruta("POST /categorias", h.Categorias.Crear)
	ruta("GET /categorias/{id}", h.Categorias.Obtener)
	ruta("PATCH /categorias/{id}", h.Categorias.Editar)
	ruta("POST /categorias/{id}/baja", h.Categorias.Baja)
	ruta("POST /categorias/{id}/restauracion", h.Categorias.Restauracion)

	ruta("GET /impuestos", h.Impuestos.ListarImpuestos)
	ruta("GET /impuestos/{id}", h.Impuestos.ObtenerImpuesto)
	ruta("GET /tasas-impuesto", h.Impuestos.ListarTasas)
	ruta("POST /tasas-impuesto", h.Impuestos.CrearTasa)
	ruta("GET /tasas-impuesto/{id}", h.Impuestos.ObtenerTasa)
	ruta("PATCH /tasas-impuesto/{id}", h.Impuestos.EditarTasa)

	ruta("GET /estaciones", h.Estaciones.Listar)
	ruta("POST /estaciones", h.Estaciones.Crear)
	ruta("GET /estaciones/{id}", h.Estaciones.Obtener)
	ruta("PATCH /estaciones/{id}", h.Estaciones.Editar)
	ruta("POST /estaciones/{id}/baja", h.Estaciones.Baja)
	ruta("POST /estaciones/{id}/restauracion", h.Estaciones.Restauracion)
}
