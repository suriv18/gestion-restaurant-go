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

// armarEstaciones compone los casos de uso de estaciones, ya decorados con transacción e idempotencia.
func armarEstaciones(db *sql.DB, logger *slog.Logger) *controller.EstacionesHandler {
	crear := escritura[cmd.CrearEstacion, dtoresult.Estacion]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearEstacion, dtoresult.Estacion] {
			return command.NewCrearEstacion(writeadapter.NewEstacionWriteAdapter(q))
		},
		meta: func(c cmd.CrearEstacion) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarEstacion, dtoresult.Estacion]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarEstacion, dtoresult.Estacion] {
			return command.NewEditarEstacion(writeadapter.NewEstacionWriteAdapter(q))
		},
		meta: func(c cmd.EditarEstacion) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoEstacion, dtoresult.Estacion]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoEstacion, dtoresult.Estacion] {
			return command.NewCambiarEstadoEstacion(writeadapter.NewEstacionWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoEstacion) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewEstacionReadAdapter(db)
	return controller.NewEstacionesHandler(uquery.NewListarEstaciones(lector), uquery.NewObtenerEstacion(lector),
		inbound.CrearEstacion(crear), inbound.EditarEstacion(editar), inbound.CambiarEstadoEstacion(estado))
}
