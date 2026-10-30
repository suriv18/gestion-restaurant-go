package repository

import (
	"context"
	"database/sql"
	"fmt"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
)

// GetLineasDelPedido bloquea FOR UPDATE todas las líneas del pedido, por id ascendente.
func (r *PedidoSQLWriter) GetLineasDelPedido(ctx context.Context, pedidoID int64) ([]outbound.LineaCuenta, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, cuenta_id, detalle_padre_id, estado
		FROM sch_restaurante.detalles_pedido WHERE pedido_id = $1 ORDER BY id FOR UPDATE`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outbound.LineaCuenta
	for rows.Next() {
		var l outbound.LineaCuenta
		var padre sql.NullInt64
		if err := rows.Scan(&l.ID, &l.CuentaID, &padre, &l.Estado); err != nil {
			return nil, err
		}
		if padre.Valid {
			v := padre.Int64
			l.PadreID = &v
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetCuentasForUpdate bloquea las cuentas del pedido, por id ascendente.
func (r *PedidoSQLWriter) GetCuentasForUpdate(ctx context.Context, pedidoID int64) ([]outbound.CuentaRef, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, pedido_id, estado
		FROM sch_restaurante.cuentas WHERE pedido_id = $1 ORDER BY id FOR UPDATE`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outbound.CuentaRef
	for rows.Next() {
		var c outbound.CuentaRef
		if err := rows.Scan(&c.ID, &c.PedidoID, &c.Estado); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCuentasConPago devuelve, de las cuentas dadas, las que tienen algún pago CONFIRMADO.
func (r *PedidoSQLWriter) GetCuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT cuenta_id FROM sch_restaurante.pagos
		WHERE cuenta_id = ANY($1::bigint[]) AND estado = 'CONFIRMADO' ORDER BY cuenta_id`, cuentaIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// InsertCuenta crea la cuenta con el siguiente número del pedido. Es seguro sin carreras porque el
// pedido ya está bloqueado FOR UPDATE por el comando.
func (r *PedidoSQLWriter) InsertCuenta(ctx context.Context, n outbound.NuevaCuenta) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.cuentas (pedido_id, numero, etiqueta)
		SELECT $1::bigint, COALESCE(max(numero), 0) + 1, $2::text
		FROM sch_restaurante.cuentas WHERE pedido_id = $1
		RETURNING id`, n.PedidoID, n.Etiqueta).Scan(&id)
	return id, Translate(err)
}

// MoveLineas reasigna las líneas a la cuenta. Si no se mueven todas es un error técnico: revierte.
func (r *PedidoSQLWriter) MoveLineas(ctx context.Context, pedidoID, cuentaID int64, lineaIDs []int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE sch_restaurante.detalles_pedido SET cuenta_id = $2
		WHERE pedido_id = $1 AND id = ANY($3::bigint[])`, pedidoID, cuentaID, lineaIDs)
	if err != nil {
		return Translate(err)
	}
	if n, err := res.RowsAffected(); err != nil || n != int64(len(lineaIDs)) {
		return fmt.Errorf("se movieron %d líneas de %d (%v)", n, len(lineaIDs), err)
	}
	return nil
}

// GetCuentaConLineas lee la cuenta con sus líneas y totales a partir del pedido completo.
func (r *PedidoSQLWriter) GetCuentaConLineas(ctx context.Context, pedidoID, cuentaID int64) (dtoresult.Cuenta, error) {
	d, found, err := cargar.Pedido(ctx, r.db, pedidoID)
	if err != nil {
		return dtoresult.Cuenta{}, err
	}
	if found {
		for _, c := range d.Cuentas {
			if c.ID == cuentaID {
				return c, nil
			}
		}
	}
	return dtoresult.Cuenta{}, sql.ErrNoRows
}
