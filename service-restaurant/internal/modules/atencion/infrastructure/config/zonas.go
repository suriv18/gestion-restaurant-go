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

// armarZonas compone los casos de uso de zonas, ya decorados con transacción e idempotencia.
func armarZonas(db *sql.DB, logger *slog.Logger) *controller.ZonasHandler {
	crear := escritura[cmd.CrearZona, dtoresult.Zona]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearZona, dtoresult.Zona] {
			return command.NewCrearZona(writeadapter.NewZonaWriteAdapter(q))
		},
		meta: func(c cmd.CrearZona) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarZona, dtoresult.Zona]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarZona, dtoresult.Zona] {
			return command.NewEditarZona(writeadapter.NewZonaWriteAdapter(q))
		},
		meta: func(c cmd.EditarZona) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoZona, dtoresult.Zona]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoZona, dtoresult.Zona] {
			return command.NewCambiarEstadoZona(writeadapter.NewZonaWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoZona) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewZonaReadAdapter(db)
	return controller.NewZonasHandler(uquery.NewListarZonas(lector), uquery.NewObtenerZona(lector),
		inbound.CrearZona(crear), inbound.EditarZona(editar), inbound.CambiarEstadoZona(estado))
}
