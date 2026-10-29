package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	"service-restaurant/internal/modules/produccion/web/controller"
)

// Deps son las dependencias técnicas del módulo.
type Deps struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// Module es el módulo produccion ya ensamblado.
type Module struct {
	deps     Deps
	comandas *controller.ComandasHandler
}

// New ensambla el módulo.
func New(d Deps) (*Module, error) {
	return &Module{deps: d, comandas: armarComandas(d.DB, d.Logger)}, nil
}

// RegisterRoutes monta las rutas del módulo sobre mux (relativas a /api/v1). protect exige un actor
// autenticado; lo aporta el módulo identidad a través del bootstrap.
func (m *Module) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	controller.RegisterRoutes(mux, m.comandas, protect, m.deps.Logger)
}
