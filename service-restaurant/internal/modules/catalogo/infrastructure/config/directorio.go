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

// armarClientes compone los casos de uso de clientes, ya decorados con transacción e idempotencia.
func armarClientes(db *sql.DB, logger *slog.Logger) *controller.ClientesHandler {
	crear := escritura[cmd.CrearCliente, dtoresult.Cliente]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearCliente, dtoresult.Cliente] {
			return command.NewCrearCliente(writeadapter.NewClienteWriteAdapter(q))
		},
		meta: func(c cmd.CrearCliente) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarCliente, dtoresult.Cliente]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarCliente, dtoresult.Cliente] {
			return command.NewEditarCliente(writeadapter.NewClienteWriteAdapter(q))
		},
		meta: func(c cmd.EditarCliente) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoCliente, dtoresult.Cliente]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoCliente, dtoresult.Cliente] {
			return command.NewCambiarEstadoCliente(writeadapter.NewClienteWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoCliente) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewClienteReadAdapter(db)
	return controller.NewClientesHandler(uquery.NewListarClientes(lector), uquery.NewObtenerCliente(lector),
		inbound.CrearCliente(crear), inbound.EditarCliente(editar), inbound.CambiarEstadoCliente(estado))
}

// armarProveedores compone los casos de uso de proveedores, ya decorados.
func armarProveedores(db *sql.DB, logger *slog.Logger) *controller.ProveedoresHandler {
	crear := escritura[cmd.CrearProveedor, dtoresult.Proveedor]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearProveedor, dtoresult.Proveedor] {
			return command.NewCrearProveedor(writeadapter.NewProveedorWriteAdapter(q))
		},
		meta: func(c cmd.CrearProveedor) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarProveedor, dtoresult.Proveedor]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarProveedor, dtoresult.Proveedor] {
			return command.NewEditarProveedor(writeadapter.NewProveedorWriteAdapter(q))
		},
		meta: func(c cmd.EditarProveedor) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoProveedor, dtoresult.Proveedor]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoProveedor, dtoresult.Proveedor] {
			return command.NewCambiarEstadoProveedor(writeadapter.NewProveedorWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoProveedor) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewProveedorReadAdapter(db)
	return controller.NewProveedoresHandler(uquery.NewListarProveedores(lector), uquery.NewObtenerProveedor(lector),
		inbound.CrearProveedor(crear), inbound.EditarProveedor(editar), inbound.CambiarEstadoProveedor(estado))
}
