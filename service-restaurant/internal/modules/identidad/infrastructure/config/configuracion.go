package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/application/usecase/command"
	uquery "service-restaurant/internal/modules/identidad/application/usecase/query"
	readadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/identidad/web/controller"
)

// configuracion agrupa los casos de uso de negocio y sucursales (RES-RF-003) ya decorados.
type configuracion struct {
	negocio *controller.NegocioHandler
}

func armarConfiguracion(db *sql.DB, logger *slog.Logger) *configuracion {
	sinIntencion := func(actorID int64) (int64, cmd.Intencion) { return actorID, cmd.Intencion{} }

	editarNegocio := escritura[cmd.EditarNegocio, dtoresult.Negocio]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarNegocio, dtoresult.Negocio] {
			return command.NewEditarNegocio(writeadapter.NewNegocioWriteAdapter(q))
		},
		meta: func(c cmd.EditarNegocio) (int64, cmd.Intencion) { return sinIntencion(c.Actor.UserID) }}

	lector := readadapter.NewConfiguracionReadAdapter(db)
	return &configuracion{
		negocio: controller.NewNegocioHandler(uquery.NewObtenerNegocio(lector), inbound.EditarNegocio(editarNegocio)),
	}
}
