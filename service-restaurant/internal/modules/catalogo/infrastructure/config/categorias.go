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

// armarCategorias compone los casos de uso de categorías, ya decorados con transacción e idempotencia.
func armarCategorias(db *sql.DB, logger *slog.Logger) *controller.CategoriasHandler {
	crear := escritura[cmd.CrearCategoria, dtoresult.Categoria]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearCategoria, dtoresult.Categoria] {
			return command.NewCrearCategoria(writeadapter.NewCategoriaWriteAdapter(q))
		},
		meta: func(c cmd.CrearCategoria) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarCategoria, dtoresult.Categoria]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarCategoria, dtoresult.Categoria] {
			return command.NewEditarCategoria(writeadapter.NewCategoriaWriteAdapter(q))
		},
		meta: func(c cmd.EditarCategoria) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoCategoria, dtoresult.Categoria]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoCategoria, dtoresult.Categoria] {
			return command.NewCambiarEstadoCategoria(writeadapter.NewCategoriaWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoCategoria) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewCategoriaReadAdapter(db)
	return controller.NewCategoriasHandler(uquery.NewListarCategorias(lector), uquery.NewObtenerCategoria(lector),
		inbound.CrearCategoria(crear), inbound.EditarCategoria(editar), inbound.CambiarEstadoCategoria(estado))
}
