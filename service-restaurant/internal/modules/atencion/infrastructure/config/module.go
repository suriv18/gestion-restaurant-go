package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	"service-restaurant/internal/modules/atencion/web/controller"
)

// Deps son las dependencias técnicas del módulo.
type Deps struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// Module es el módulo atencion ya ensamblado.
type Module struct {
	deps    Deps
	zonas   *controller.ZonasHandler
	mesas   *controller.MesasHandler
	pedidos *controller.PedidosHandler
}

// New ensambla el módulo.
func New(d Deps) (*Module, error) {
	return &Module{deps: d, zonas: armarZonas(d.DB, d.Logger), mesas: armarMesas(d.DB, d.Logger),
		pedidos: armarPedidos(d.DB, d.Logger)}, nil
}

// RegisterRoutes monta las rutas del módulo sobre mux (relativas a /api/v1). protect exige un
// actor autenticado; lo aporta el módulo identidad a través del bootstrap.
func (m *Module) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	controller.RegisterRoutes(mux, m.zonas, m.mesas, m.pedidos, protect, m.deps.Logger)
}
