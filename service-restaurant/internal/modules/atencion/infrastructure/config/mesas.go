package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	"service-restaurant/internal/modules/atencion/application/usecase/command"
	uquery "service-restaurant/internal/modules/atencion/application/usecase/query"
	readadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/atencion/web/controller"
)

// armarMesas compone los casos de uso de mesas, ya decorados con transacción e idempotencia.
func armarMesas(db *sql.DB, logger *slog.Logger) *controller.MesasHandler {
	crear := escritura[cmd.CrearMesa, dtoresult.Mesa]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearMesa, dtoresult.Mesa] {
			return command.NewCrearMesa(writeadapter.NewMesaWriteAdapter(q))
		},
		meta: func(c cmd.CrearMesa) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarMesa, dtoresult.Mesa]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarMesa, dtoresult.Mesa] {
			return command.NewEditarMesa(writeadapter.NewMesaWriteAdapter(q))
		},
		meta: func(c cmd.EditarMesa) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoMesa, dtoresult.Mesa]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoMesa, dtoresult.Mesa] {
			return command.NewCambiarEstadoMesa(writeadapter.NewMesaWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoMesa) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewMesaReadAdapter(db)
	return controller.NewMesasHandler(uquery.NewListarMesas(lector), uquery.NewObtenerMesa(lector),
		inbound.CrearMesa(crear), inbound.EditarMesa(editar), inbound.CambiarEstadoMesa(estado))
}
