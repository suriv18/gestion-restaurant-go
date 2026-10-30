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

// armarPedidos compone los casos de uso de pedidos y consumos, ya decorados con transacción e
// idempotencia. La edición de una línea no lleva clave: es una operación sobre un recurso ya
// identificado y repetirla produce el mismo estado.
func armarPedidos(db *sql.DB, logger *slog.Logger) *controller.PedidosHandler {
	abrir := escritura[cmd.AbrirPedido, dtoresult.PedidoDetalle]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.AbrirPedido, dtoresult.PedidoDetalle] {
			return command.NewAbrirPedido(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.AbrirPedido) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	agregar := escritura[cmd.AgregarDetalle, dtoresult.Linea]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.AgregarDetalle, dtoresult.Linea] {
			return command.NewAgregarDetalle(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.AgregarDetalle) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarDetalle, dtoresult.Linea]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarDetalle, dtoresult.Linea] {
			return command.NewEditarDetalle(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.EditarDetalle) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}

	dividir := escritura[cmd.DividirCuenta, dtoresult.Cuenta]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.DividirCuenta, dtoresult.Cuenta] {
			return command.NewDividirCuenta(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.DividirCuenta) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewPedidoReadAdapter(db)
	return controller.NewPedidosHandler(uquery.NewListarPedidos(lector), uquery.NewObtenerPedido(lector),
		inbound.AbrirPedido(abrir), inbound.AgregarDetalle(agregar), inbound.EditarDetalle(editar),
		inbound.DividirCuenta(dividir))
}
