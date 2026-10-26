package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/inbound"
	"service-restaurant/internal/modules/catalogo/application/usecase/command"
	uquery "service-restaurant/internal/modules/catalogo/application/usecase/query"
	readadapter "service-restaurant/internal/modules/catalogo/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/catalogo/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/catalogo/web/controller"
)

// armarImpuestos compone la lectura de impuestos y la lectura/escritura de sus tasas.
func armarImpuestos(db *sql.DB, logger *slog.Logger) *controller.ImpuestosHandler {
	crear := escritura[cmd.CrearTasaImpuesto, dtoresult.TasaImpuesto]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearTasaImpuesto, dtoresult.TasaImpuesto] {
			return command.NewCrearTasaImpuesto(writeadapter.NewTasaWriteAdapter(q))
		},
		meta: func(c cmd.CrearTasaImpuesto) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarTasaImpuesto, dtoresult.TasaImpuesto]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarTasaImpuesto, dtoresult.TasaImpuesto] {
			return command.NewEditarTasaImpuesto(writeadapter.NewTasaWriteAdapter(q))
		},
		meta: func(c cmd.EditarTasaImpuesto) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}

	lector := readadapter.NewImpuestoReadAdapter(db)
	return controller.NewImpuestosHandler(uquery.NewListarImpuestos(lector), uquery.NewObtenerImpuesto(lector),
		uquery.NewListarTasasImpuesto(lector), uquery.NewObtenerTasaImpuesto(lector),
		inbound.CrearTasaImpuesto(crear), inbound.EditarTasaImpuesto(editar))
}
