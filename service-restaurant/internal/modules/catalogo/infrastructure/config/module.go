package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	"service-restaurant/internal/modules/catalogo/web/controller"
)

// Deps son las dependencias técnicas del módulo.
type Deps struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// Module es el módulo catalogo ya ensamblado: el directorio de clientes y proveedores y la carta.
type Module struct {
	deps     Deps
	handlers controller.Handlers
}

// New ensambla el módulo.
func New(d Deps) (*Module, error) {
	return &Module{deps: d, handlers: controller.Handlers{
		Clientes:    armarClientes(d.DB, d.Logger),
		Proveedores: armarProveedores(d.DB, d.Logger),
		Categorias:  armarCategorias(d.DB, d.Logger),
		Impuestos:   armarImpuestos(d.DB, d.Logger),
		Estaciones:  armarEstaciones(d.DB, d.Logger),
		Productos:   armarProductos(d.DB, d.Logger),
	}}, nil
}

// RegisterRoutes monta las rutas del módulo sobre mux (relativas a /api/v1). protect exige un
// actor autenticado; lo aporta el módulo identidad a través del bootstrap.
func (m *Module) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	controller.RegisterRoutes(mux, m.handlers, protect, m.deps.Logger)
}
