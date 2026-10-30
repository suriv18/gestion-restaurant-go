package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/inbound"
	"service-restaurant/internal/modules/produccion/application/usecase/command"
	uquery "service-restaurant/internal/modules/produccion/application/usecase/query"
	readadapter "service-restaurant/internal/modules/produccion/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/produccion/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/produccion/web/controller"
)

// armarComandas compone los casos de uso de cocina, ya decorados con transacción e idempotencia.
func armarComandas(db *sql.DB, logger *slog.Logger) *controller.ComandasHandler {
	enviar := escritura[cmd.EnviarComanda, dtoresult.Comanda]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.EnviarComanda, dtoresult.Comanda] {
			return command.NewEnviarComanda(writeadapter.NewComandaWriteAdapter(q))
		},
		meta: func(c cmd.EnviarComanda) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	transicionar := escritura[cmd.TransicionarLinea, dtoresult.ComandaLinea]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.TransicionarLinea, dtoresult.ComandaLinea] {
			return command.NewTransicionarLinea(writeadapter.NewComandaWriteAdapter(q))
		},
		meta: func(c cmd.TransicionarLinea) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}

	lector := readadapter.NewComandaReadAdapter(db)
	return controller.NewComandasHandler(uquery.NewListarComandas(lector),
		inbound.EnviarComanda(enviar), inbound.TransicionarLinea(transicionar))
}
