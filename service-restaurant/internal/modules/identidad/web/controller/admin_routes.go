package controller

import (
	"crypto/sha256"
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

func sha256Of(s string) [32]byte { return sha256.Sum256([]byte(s)) }

// RegisterAdminRoutes monta la administración de accesos (todas exigen autenticación;
// el permiso USUARIO_ADMINISTRAR lo verifica cada caso de uso).
func RegisterAdminRoutes(mux *http.ServeMux, roles *RolesHandler, permisos *PermisosHandler, usuarios *UsuariosHandler,
	protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /roles", roles.Listar)
	ruta("POST /roles", roles.Crear)
	ruta("GET /roles/{id}", roles.Obtener)
	ruta("PATCH /roles/{id}", roles.Editar)
	ruta("POST /roles/{id}/baja", roles.Baja)
	ruta("POST /roles/{id}/restauracion", roles.Restauracion)

	ruta("GET /permisos", permisos.Listar)
	ruta("POST /permisos", permisos.Crear)
	ruta("GET /permisos/{id}", permisos.Obtener)
	ruta("PATCH /permisos/{id}", permisos.Editar)
	ruta("POST /permisos/{id}/baja", permisos.Baja)
	ruta("POST /permisos/{id}/restauracion", permisos.Restauracion)

	ruta("GET /usuarios", usuarios.Listar)
	ruta("GET /sucursales-asignables", usuarios.SucursalesAsignables)
	ruta("POST /usuarios", usuarios.Crear)
	ruta("PUT /usuarios/{id}/asignaciones", usuarios.Asignaciones)
	ruta("POST /usuarios/{id}/baja", usuarios.Baja)
	ruta("POST /usuarios/{id}/restauracion", usuarios.Restauracion)
}
