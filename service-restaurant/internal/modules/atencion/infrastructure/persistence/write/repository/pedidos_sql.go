package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// Restricciones únicas que tienen un rechazo propio.
const (
	restriccionMesaOcupada = "uq_pedido_mesa_abierta"
	restriccionClavePedido = "pedidos_clave_idempotencia_key"
)

// traducirPedido distingue la mesa ocupada y la clave repetida por el nombre de la restricción.
func traducirPedido(err error) error {
	switch restriccionUnica(err) {
	case restriccionMesaOcupada:
		return outbound.ErrMesaOcupada
	case restriccionClavePedido:
		return outbound.ErrClaveEnUso
	}
	return Translate(err)
}

// PedidoSQLWriter opera sobre pedidos, cuentas, pedidos_delivery y detalles_pedido.
type PedidoSQLWriter struct{ db database.DBTX }

func NewPedidoSQLWriter(db database.DBTX) *PedidoSQLWriter { return &PedidoSQLWriter{db} }

// GetMesa lee la mesa en modo compartido: no puede deshabilitarse hasta que termine el comando.
func (r *PedidoSQLWriter) GetMesa(ctx context.Context, id int64) (outbound.MesaRef, bool, error) {
	var m outbound.MesaRef
	err := r.db.QueryRowContext(ctx, `SELECT sucursal_id, capacidad, habilitada, es_activo
		FROM sch_restaurante.mesas WHERE id = $1 FOR SHARE`, id).Scan(&m.SucursalID, &m.Capacidad, &m.Habilitada, &m.EsActiva)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.MesaRef{}, false, err
	}
	return m, true, nil
}

// GetEmpleadoDeUsuario devuelve el empleado activo vinculado al usuario, o nil.
func (r *PedidoSQLWriter) GetEmpleadoDeUsuario(ctx context.Context, usuarioID int64) (*int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM sch_restaurante.empleados WHERE usuario_id = $1 AND es_activo`, usuarioID).Scan(&id)
	if missing, err := notFound(err); missing || err != nil {
		return nil, err
	}
	return &id, nil
}

// InsertPedido crea pedido, cuenta nº 1 y reparto en la transacción en curso y devuelve el pedido
// armado. La moneda es la del negocio (PEN si no hay negocio activo).
func (r *PedidoSQLWriter) InsertPedido(ctx context.Context, n outbound.NuevoPedido) (dtoresult.PedidoDetalle, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.pedidos (sucursal_id, clave_idempotencia, mesa_id, cliente_id, mesero_id, tipo, comensales, notas, moneda)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8,
			COALESCE((SELECT moneda FROM sch_restaurante.negocio WHERE es_activo ORDER BY id LIMIT 1), 'PEN'))
		RETURNING id`,
		n.SucursalID, n.ClaveIdempotencia, n.MesaID, n.ClienteID, n.MeseroID, n.Tipo, n.Comensales, n.Notas).Scan(&id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, traducirPedido(err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO sch_restaurante.cuentas (pedido_id, numero, cliente_id) VALUES ($1, 1, $2)`, id, n.ClienteID); err != nil {
		return dtoresult.PedidoDetalle{}, traducirPedido(err)
	}
	if n.Reparto != nil {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO sch_restaurante.pedidos_delivery (pedido_id, destinatario, telefono, direccion, referencia, canal)
			VALUES ($1, $2, $3, $4, $5, COALESCE($6::text, 'PROPIO'))`,
			id, n.Reparto.Destinatario, n.Reparto.Telefono, n.Reparto.Direccion, n.Reparto.Referencia, n.Reparto.Canal); err != nil {
			return dtoresult.PedidoDetalle{}, traducirPedido(err)
		}
	}
	d, found, err := cargar.Pedido(ctx, r.db, id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, err
	}
	if !found {
		return dtoresult.PedidoDetalle{}, sql.ErrNoRows
	}
	return d, nil
}

// GetPedidoForUpdate bloquea el pedido hasta el final de la transacción.
func (r *PedidoSQLWriter) GetPedidoForUpdate(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	var p outbound.PedidoRef
	err := r.db.QueryRowContext(ctx, `SELECT id, sucursal_id, estado, cerrado_en IS NOT NULL
		FROM sch_restaurante.pedidos WHERE id = $1 FOR UPDATE`, id).Scan(&p.ID, &p.SucursalID, &p.Estado, &p.Cerrado)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.PedidoRef{}, false, err
	}
	return p, true, nil
}

// GetCuentaForUpdate bloquea la cuenta; no la encuentra si es de otro pedido.
func (r *PedidoSQLWriter) GetCuentaForUpdate(ctx context.Context, pedidoID, cuentaID int64) (outbound.CuentaRef, bool, error) {
	var c outbound.CuentaRef
	err := r.db.QueryRowContext(ctx, `SELECT id, pedido_id, estado
		FROM sch_restaurante.cuentas WHERE id = $1 AND pedido_id = $2 FOR UPDATE`, cuentaID, pedidoID).Scan(&c.ID, &c.PedidoID, &c.Estado)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.CuentaRef{}, false, err
	}
	return c, true, nil
}

// GetProductoVenta lee el producto con el precio de la sede (si lo tiene propio) y la tasa vigente
// hoy (hora de Lima) de su impuesto. El producto se bloquea en modo compartido para que su baja
// se serialice con la venta. Disponible es false si no hay fila activa del producto en la sede.
func (r *PedidoSQLWriter) GetProductoVenta(ctx context.Context, productoID, sedeID int64) (outbound.ProductoVenta, bool, error) {
	var p outbound.ProductoVenta
	var tasa sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.nombre, p.tipo, p.afectacion_igv, COALESCE(ps.precio_venta, p.precio_venta)::text,
		       t.tasa::text, p.es_activo, COALESCE(ps.disponible, false)
		FROM sch_restaurante.productos p
		LEFT JOIN sch_restaurante.productos_sucursal ps
		       ON ps.producto_id = p.id AND ps.sucursal_id = $2 AND ps.es_activo
		LEFT JOIN sch_restaurante.tasas_impuesto t
		       ON t.impuesto_id = p.impuesto_id
		      AND t.vigente_desde <= (now() AT TIME ZONE 'America/Lima')::date
		      AND (t.vigente_hasta IS NULL OR t.vigente_hasta > (now() AT TIME ZONE 'America/Lima')::date)
		WHERE p.id = $1
		FOR SHARE OF p`, productoID, sedeID).Scan(&p.ID, &p.Nombre, &p.Tipo, &p.AfectacionIGV, &p.Precio, &tasa, &p.EsActivo, &p.Disponible)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.ProductoVenta{}, false, err
	}
	if tasa.Valid {
		p.Tasa = tasa.String
	}
	return p, true, nil
}

// GetDetalleForUpdate bloquea la línea; no la encuentra si es de otro pedido. EnComanda indica que
// ya tiene un envío a cocina.
func (r *PedidoSQLWriter) GetDetalleForUpdate(ctx context.Context, pedidoID, detalleID int64) (outbound.DetalleRef, bool, error) {
	var d outbound.DetalleRef
	err := r.db.QueryRowContext(ctx, `
		SELECT id, pedido_id, cuenta_id, producto_id, cantidad::text, precio_unitario::text, descuento_total::text, estado,
		       EXISTS (SELECT 1 FROM sch_restaurante.comanda_detalles cd WHERE cd.detalle_pedido_id = detalles_pedido.id)
		FROM sch_restaurante.detalles_pedido WHERE id = $1 AND pedido_id = $2 FOR UPDATE`, detalleID, pedidoID).
		Scan(&d.ID, &d.PedidoID, &d.CuentaID, &d.ProductoID, &d.Cantidad, &d.PrecioUnitario, &d.DescuentoTotal, &d.Estado, &d.EnComanda)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.DetalleRef{}, false, err
	}
	return d, true, nil
}

func (r *PedidoSQLWriter) InsertDetalle(ctx context.Context, n outbound.NuevoDetalle) (dtoresult.Linea, error) {
	l, err := columnas.ScanLinea(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.detalles_pedido
			(pedido_id, cuenta_id, producto_id, detalle_padre_id, descripcion, cantidad, precio_unitario,
			 descuento_total, afectacion_igv, tasa_impuesto, notas_preparacion)
		VALUES ($1, $2, $3, $4, $5, $6::text::numeric, $7::text::numeric, $8::text::numeric, $9, $10::text::numeric, $11)
		RETURNING `+columnas.Linea,
		n.PedidoID, n.CuentaID, n.ProductoID, n.DetallePadreID, n.Descripcion, n.Cantidad, n.PrecioUnitario,
		n.DescuentoTotal, n.AfectacionIGV, n.TasaImpuesto, n.NotasPreparacion))
	return l, Translate(err)
}

func (r *PedidoSQLWriter) UpdateDetalle(ctx context.Context, id int64, c outbound.CambiosDetalle) (dtoresult.Linea, error) {
	l, err := columnas.ScanLinea(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.detalles_pedido SET
			cantidad = COALESCE($2::text::numeric, cantidad),
			descuento_total = COALESCE($3::text::numeric, descuento_total),
			notas_preparacion = CASE WHEN $4 THEN $5::text ELSE notas_preparacion END
		WHERE id = $1 RETURNING `+columnas.Linea, id, c.Cantidad, c.DescuentoTotal, c.Instrucciones.Presente, c.Instrucciones.Valor))
	return l, Translate(err)
}
