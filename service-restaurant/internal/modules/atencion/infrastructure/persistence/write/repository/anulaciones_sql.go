package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
)

// estadosCocinaCancelables son los estados de una línea de comanda que aún se pueden cancelar: el
// plato todavía no llegó a la mesa. ENTREGADO y CANCELADO quedan como están.
const estadosCocinaCancelables = `'PENDIENTE','PREPARACION','LISTO'`

// ExistsCocinaEntregadaEnLinea indica si la línea o alguno de sus adicionales tiene una línea de
// comanda ya ENTREGADO.
func (r *PedidoSQLWriter) ExistsCocinaEntregadaEnLinea(ctx context.Context, pedidoID, detalleID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM sch_restaurante.comanda_detalles cd
		JOIN sch_restaurante.detalles_pedido d ON d.id = cd.detalle_pedido_id
		WHERE d.pedido_id = $1 AND (d.id = $2 OR d.detalle_padre_id = $2) AND cd.estado = 'ENTREGADO')`,
		pedidoID, detalleID).Scan(&ok)
	return ok, err
}

// ExistsCocinaEntregadaEnPedido indica si alguna línea de comanda del pedido está ENTREGADO.
func (r *PedidoSQLWriter) ExistsCocinaEntregadaEnPedido(ctx context.Context, pedidoID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM sch_restaurante.comanda_detalles WHERE pedido_id = $1 AND estado = 'ENTREGADO')`,
		pedidoID).Scan(&ok)
	return ok, err
}

// UpdateAnularLinea cancela las líneas de comanda de la línea y sus adicionales que aún no se
// entregaron, anula la línea y sus adicionales activos y devuelve la línea. La línea se conserva.
func (r *PedidoSQLWriter) UpdateAnularLinea(ctx context.Context, pedidoID, detalleID int64, a outbound.Anulacion) (dtoresult.Linea, error) {
	if _, err := r.db.ExecContext(ctx, `UPDATE sch_restaurante.comanda_detalles cd
		SET estado = 'CANCELADO', motivo_cancelacion = $3
		FROM sch_restaurante.detalles_pedido d
		WHERE d.id = cd.detalle_pedido_id AND d.pedido_id = $1 AND (d.id = $2 OR d.detalle_padre_id = $2)
		  AND cd.estado IN (`+estadosCocinaCancelables+`)`, pedidoID, detalleID, a.Motivo); err != nil {
		return dtoresult.Linea{}, Translate(err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE sch_restaurante.detalles_pedido
		SET estado = 'ANULADO', anulado_en = clock_timestamp(), anulado_por = $3, motivo_anulacion = $4
		WHERE pedido_id = $1 AND (id = $2 OR detalle_padre_id = $2) AND estado = 'ACTIVO'`,
		pedidoID, detalleID, a.UsuarioID, a.Motivo); err != nil {
		return dtoresult.Linea{}, Translate(err)
	}
	l, err := columnas.ScanLinea(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Linea+` FROM sch_restaurante.detalles_pedido WHERE id = $1`, detalleID))
	return l, Translate(err)
}

// UpdateCancelarPedido cancela, en este orden (los triggers del esquema lo exigen): las líneas de
// comanda no entregadas, las líneas activas, las cuentas abiertas y por último el pedido, que queda
// cerrado y libera la mesa. Devuelve el pedido completo.
func (r *PedidoSQLWriter) UpdateCancelarPedido(ctx context.Context, pedidoID int64, a outbound.Anulacion) (dtoresult.PedidoDetalle, error) {
	pasos := []struct {
		sentencia string
		args      []any
	}{
		{`UPDATE sch_restaurante.comanda_detalles SET estado = 'CANCELADO', motivo_cancelacion = $2
			WHERE pedido_id = $1 AND estado IN (` + estadosCocinaCancelables + `)`, []any{pedidoID, a.Motivo}},
		{`UPDATE sch_restaurante.detalles_pedido
			SET estado = 'ANULADO', anulado_en = clock_timestamp(), anulado_por = $3, motivo_anulacion = $2
			WHERE pedido_id = $1 AND estado = 'ACTIVO'`, []any{pedidoID, a.Motivo, a.UsuarioID}},
		{`UPDATE sch_restaurante.cuentas SET estado = 'CANCELADA', motivo_cancelacion = $2
			WHERE pedido_id = $1 AND estado = 'ABIERTA'`, []any{pedidoID, a.Motivo}},
		{`UPDATE sch_restaurante.pedidos
			SET estado = 'CANCELADO', cerrado_en = clock_timestamp(), motivo_cancelacion = $2
			WHERE id = $1`, []any{pedidoID, a.Motivo}},
	}
	for _, p := range pasos {
		if _, err := r.db.ExecContext(ctx, p.sentencia, p.args...); err != nil {
			return dtoresult.PedidoDetalle{}, Translate(err)
		}
	}
	d, found, err := cargar.Pedido(ctx, r.db, pedidoID)
	if err != nil {
		return dtoresult.PedidoDetalle{}, err
	}
	if !found {
		return dtoresult.PedidoDetalle{}, sql.ErrNoRows
	}
	return d, nil
}
