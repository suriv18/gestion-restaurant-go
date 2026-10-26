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

// armarProductos compone los casos de uso de productos, ya decorados con transacción e idempotencia.
func armarProductos(db *sql.DB, logger *slog.Logger) *controller.ProductosHandler {
	crear := escritura[cmd.CrearProducto, dtoresult.Producto]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearProducto, dtoresult.Producto] {
			return command.NewCrearProducto(writeadapter.NewProductoWriteAdapter(q))
		},
		meta: func(c cmd.CrearProducto) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarProducto, dtoresult.Producto]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarProducto, dtoresult.Producto] {
			return command.NewEditarProducto(writeadapter.NewProductoWriteAdapter(q))
		},
		meta: func(c cmd.EditarProducto) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoProducto, dtoresult.Producto]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoProducto, dtoresult.Producto] {
			return command.NewCambiarEstadoProducto(writeadapter.NewProductoWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoProducto) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	disponibilidad := escritura[cmd.CambiarDisponibilidad, dtoresult.ProductoSucursal]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarDisponibilidad, dtoresult.ProductoSucursal] {
			return command.NewCambiarDisponibilidad(writeadapter.NewDisponibilidadWriteAdapter(q))
		},
		meta: func(c cmd.CambiarDisponibilidad) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}

	lector := readadapter.NewProductoLectorAdapter(db)
	return controller.NewProductosHandler(uquery.NewListarProductos(lector), uquery.NewObtenerProducto(lector),
		inbound.CrearProducto(crear), inbound.EditarProducto(editar), inbound.CambiarEstadoProducto(estado), inbound.CambiarDisponibilidad(disponibilidad))
}
