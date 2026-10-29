package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/platform/database"
)

// restriccionClaveComanda es la restricción única de comandas.clave_idempotencia.
const restriccionClaveComanda = "comandas_clave_idempotencia_key"

// traducirComanda distingue la clave repetida por el nombre de la restricción.
func traducirComanda(err error) error {
	if restriccionUnica(err) == restriccionClaveComanda {
		return outbound.ErrClaveEnUso
	}
	return Translate(err)
}

// ComandaSQLWriter opera sobre comandas y comanda_detalles, y lee pedidos, líneas y estaciones.
type ComandaSQLWriter struct{ db database.DBTX }

func NewComandaSQLWriter(db database.DBTX) *ComandaSQLWriter { return &ComandaSQLWriter{db} }

// GetPedidoForUpdate bloquea el pedido hasta el final de la transacción.
func (r *ComandaSQLWriter) GetPedidoForUpdate(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	var p outbound.PedidoRef
	err := r.db.QueryRowContext(ctx, `SELECT id, sucursal_id, cerrado_en IS NOT NULL
		FROM sch_restaurante.pedidos WHERE id = $1 FOR UPDATE`, id).Scan(&p.ID, &p.SucursalID, &p.Cerrado)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.PedidoRef{}, false, err
	}
	return p, true, nil
}

// GetEstacion lee la estación en modo compartido: no puede darse de baja hasta que termine el comando.
func (r *ComandaSQLWriter) GetEstacion(ctx context.Context, id int64) (outbound.EstacionRef, bool, error) {
	var e outbound.EstacionRef
	err := r.db.QueryRowContext(ctx, `SELECT id, sucursal_id, es_activo
		FROM sch_restaurante.estaciones_preparacion WHERE id = $1 FOR SHARE`, id).Scan(&e.ID, &e.SucursalID, &e.EsActiva)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.EstacionRef{}, false, err
	}
	return e, true, nil
}

// GetLineasParaEnvio bloquea las líneas del pedido con esos ids, por id ascendente. Enviado suma lo
// ya enviado de cada línea sin contar las líneas de comanda canceladas. La estación sale de la fila
// activa del producto en la sede del pedido. El pedido ya está bloqueado: dos envíos del mismo
// pedido nunca llegan aquí a la vez.
func (r *ComandaSQLWriter) GetLineasParaEnvio(ctx context.Context, pedidoID, sucursalID int64, ids []int64) ([]outbound.LineaPedidoRef, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.cuenta_id, c.estado, d.estado, d.cantidad::text, d.descripcion, d.notas_preparacion, ps.estacion_id,
		       COALESCE((SELECT sum(cd.cantidad) FROM sch_restaurante.comanda_detalles cd
		                  WHERE cd.detalle_pedido_id = d.id AND cd.estado <> 'CANCELADO'), 0)::text
		FROM sch_restaurante.detalles_pedido d
		JOIN sch_restaurante.cuentas c ON c.id = d.cuenta_id
		LEFT JOIN sch_restaurante.productos_sucursal ps
		       ON ps.producto_id = d.producto_id AND ps.sucursal_id = $3 AND ps.es_activo
		WHERE d.pedido_id = $1 AND d.id = ANY($2::bigint[])
		ORDER BY d.id
		FOR UPDATE OF d`, pedidoID, ids, sucursalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outbound.LineaPedidoRef
	for rows.Next() {
		var l outbound.LineaPedidoRef
		var instr sql.NullString
		var estacion sql.NullInt64
		if err := rows.Scan(&l.ID, &l.CuentaID, &l.CuentaEstado, &l.Estado, &l.Cantidad, &l.Descripcion, &instr, &estacion, &l.Enviado); err != nil {
			return nil, err
		}
		if instr.Valid {
			s := instr.String
			l.Instrucciones = &s
		}
		if estacion.Valid {
			n := estacion.Int64
			l.EstacionID = &n
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// InsertComanda crea la comanda y sus líneas en la transacción en curso y la devuelve armada.
// creado_por, actualizado_por y las marcas de tiempo los fija el trigger de metadatos.
func (r *ComandaSQLWriter) InsertComanda(ctx context.Context, n outbound.NuevaComanda) (dtoresult.Comanda, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.comandas (pedido_id, sucursal_id, estacion_id, clave_idempotencia, enviado_por)
		VALUES ($1, $2, $3, $4::uuid, $5) RETURNING id`,
		n.PedidoID, n.SucursalID, n.EstacionID, n.ClaveIdempotencia, n.EnviadoPor).Scan(&id)
	if err != nil {
		return dtoresult.Comanda{}, traducirComanda(err)
	}
	for _, l := range n.Lineas {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO sch_restaurante.comanda_detalles (comanda_id, pedido_id, detalle_pedido_id, cantidad, descripcion, instrucciones)
			VALUES ($1, $2, $3, $4::text::numeric, $5, $6)`,
			id, n.PedidoID, l.DetallePedidoID, l.Cantidad, l.Descripcion, l.Instrucciones); err != nil {
			return dtoresult.Comanda{}, traducirComanda(err)
		}
	}
	c, found, err := cargar.Comanda(ctx, r.db, id)
	if err != nil {
		return dtoresult.Comanda{}, err
	}
	if !found {
		return dtoresult.Comanda{}, sql.ErrNoRows
	}
	return c, nil
}
