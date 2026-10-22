package config

import (
	"database/sql"

	uquery "service-restaurant/internal/modules/atencion/application/usecase/query"
	readadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter"
	"service-restaurant/internal/modules/atencion/web/controller"
)

// armarMesas compone los casos de uso de mesas.
func armarMesas(db *sql.DB) *controller.MesasHandler {
	lector := readadapter.NewMesaReadAdapter(db)
	return controller.NewMesasHandler(uquery.NewListarMesas(lector), uquery.NewObtenerMesa(lector))
}
