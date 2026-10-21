package config

import (
	"database/sql"

	uquery "service-restaurant/internal/modules/atencion/application/usecase/query"
	readadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter"
	"service-restaurant/internal/modules/atencion/web/controller"
)

// armarZonas compone los casos de uso de zonas.
func armarZonas(db *sql.DB) *controller.ZonasHandler {
	lector := readadapter.NewZonaReadAdapter(db)
	return controller.NewZonasHandler(uquery.NewListarZonas(lector), uquery.NewObtenerZona(lector))
}
