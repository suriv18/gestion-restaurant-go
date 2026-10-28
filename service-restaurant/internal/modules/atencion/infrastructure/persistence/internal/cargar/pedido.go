// Package cargar arma un pedido completo (reparto, cuentas, líneas y totales) con consultas que
// comparten el lado de lectura y el de escritura. Vive bajo persistence/internal: solo la
// persistencia del módulo puede importarlo.
package cargar

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// Pedido lee el pedido con su reparto, sus cuentas, las líneas de cada cuenta y los totales de las
// líneas activas. found=false si no existe. Cada consulta se agota antes de lanzar la siguiente:
// sobre una transacción no puede haber dos resultados abiertos a la vez.
func Pedido(ctx context.Context, db database.DBTX, id int64) (dtoresult.PedidoDetalle, bool, error) {
	p, err := columnas.ScanPedido(db.QueryRowContext(ctx,
		`SELECT `+columnas.Pedido+` FROM sch_restaurante.pedidos WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.PedidoDetalle{}, false, nil
	}
	if err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	d := dtoresult.PedidoDetalle{Pedido: p, Cuentas: []dtoresult.Cuenta{}}
	if d.Reparto, err = leerReparto(ctx, db, id); err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	if d.Cuentas, err = leerCuentas(ctx, db, id); err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	lineas, err := leerLineas(ctx, db, id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	totales, err := leerTotales(ctx, db, id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	for i := range d.Cuentas {
		c := &d.Cuentas[i]
		c.Lineas = []dtoresult.Linea{}
		c.Subtotal, c.Impuesto, c.Total = "0.00", "0.00", "0.00"
		if t, ok := totales[c.ID]; ok {
			c.Subtotal, c.Impuesto, c.Total = t[0], t[1], t[2]
		}
		for _, l := range lineas {
			if l.CuentaID == c.ID {
				c.Lineas = append(c.Lineas, l)
			}
		}
	}
	return d, true, nil
}

func leerReparto(ctx context.Context, db database.DBTX, pedidoID int64) (*dtoresult.Reparto, error) {
	var r dtoresult.Reparto
	var ref sql.NullString
	err := db.QueryRowContext(ctx, `SELECT destinatario, telefono, direccion, referencia, canal, estado
		FROM sch_restaurante.pedidos_delivery WHERE pedido_id = $1`, pedidoID).
		Scan(&r.Destinatario, &r.Telefono, &r.Direccion, &ref, &r.Canal, &r.Estado)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if ref.Valid {
		v := ref.String
		r.Referencia = &v
	}
	return &r, nil
}

func leerCuentas(ctx context.Context, db database.DBTX, pedidoID int64) ([]dtoresult.Cuenta, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+columnas.Cuenta+` FROM sch_restaurante.cuentas WHERE pedido_id = $1 ORDER BY numero`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dtoresult.Cuenta{}
	for rows.Next() {
		c, err := columnas.ScanCuenta(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func leerLineas(ctx context.Context, db database.DBTX, pedidoID int64) ([]dtoresult.Linea, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+columnas.Linea+` FROM sch_restaurante.detalles_pedido WHERE pedido_id = $1 ORDER BY id`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Linea
	for rows.Next() {
		l, err := columnas.ScanLinea(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// leerTotales devuelve, por cuenta, [subtotal, impuesto, total] de las líneas activas.
func leerTotales(ctx context.Context, db database.DBTX, pedidoID int64) (map[int64][3]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT cuenta_id, sum(base_imponible)::text, (sum(importe_total) - sum(base_imponible))::text, sum(importe_total)::text
		FROM sch_restaurante.detalles_pedido WHERE pedido_id = $1 AND estado = 'ACTIVO' GROUP BY cuenta_id`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][3]string{}
	for rows.Next() {
		var cuenta int64
		var t [3]string
		if err := rows.Scan(&cuenta, &t[0], &t[1], &t[2]); err != nil {
			return nil, err
		}
		out[cuenta] = t
	}
	return out, rows.Err()
}
