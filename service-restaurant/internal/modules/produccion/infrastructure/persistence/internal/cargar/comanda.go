// Package cargar arma comandas con sus líneas con consultas que comparten el lado de lectura y el
// de escritura. Vive bajo persistence/internal: solo la persistencia del módulo puede importarlo.
package cargar

import (
	"context"
	"database/sql"
	"time"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/platform/database"
)

// Las cantidades son numeric: se leen como texto para no perder exactitud.
const (
	colComanda = `c.id, c.pedido_id, c.sucursal_id, c.estacion_id, p.mesa_id, c.clave_idempotencia::text, c.enviado_en,
		c.enviado_por, c.notas, c.creado_en, c.creado_por, c.actualizado_en, c.actualizado_por`

	// ColLinea son las columnas de comanda_detalles que lee ScanLinea.
	ColLinea = `id, comanda_id, pedido_id, detalle_pedido_id, cantidad::text, descripcion, instrucciones, estado,
		iniciado_en, listo_en, entregado_en, motivo_cancelacion, creado_en, creado_por, actualizado_en, actualizado_por`
)

type fila interface{ Scan(...any) error }

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func fechaNula(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

// ScanLinea lee una fila con las columnas ColLinea.
func ScanLinea(s fila) (dtoresult.ComandaLinea, error) {
	var l dtoresult.ComandaLinea
	var instr, motivo sql.NullString
	var ini, listo, entrega sql.NullTime
	err := s.Scan(&l.ID, &l.ComandaID, &l.PedidoID, &l.DetallePedidoID, &l.Cantidad, &l.Descripcion, &instr, &l.Estado,
		&ini, &listo, &entrega, &motivo, &l.CreadoEn, &l.CreadoPor, &l.ActualizadoEn, &l.ActualizadoPor)
	l.Instrucciones, l.MotivoCancelacion = textoNulo(instr), textoNulo(motivo)
	l.IniciadoEn, l.ListoEn, l.EntregadoEn = fechaNula(ini), fechaNula(listo), fechaNula(entrega)
	return l, err
}

// Comandas lee las comandas con esos ids, ordenadas por id, cada una con todas sus líneas. Cada
// consulta se agota antes de lanzar la siguiente: sobre una transacción no puede haber dos
// resultados abiertos a la vez.
func Comandas(ctx context.Context, db database.DBTX, ids []int64) ([]dtoresult.Comanda, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+colComanda+` FROM sch_restaurante.comandas c
		JOIN sch_restaurante.pedidos p ON p.id = c.pedido_id
		WHERE c.id = ANY($1::bigint[]) ORDER BY c.id`, ids)
	if err != nil {
		return nil, err
	}
	out := []dtoresult.Comanda{}
	for rows.Next() {
		var c dtoresult.Comanda
		var mesa sql.NullInt64
		var notas sql.NullString
		if err := rows.Scan(&c.ID, &c.PedidoID, &c.SucursalID, &c.EstacionID, &mesa, &c.ClaveIdempotencia, &c.EnviadoEn,
			&c.EnviadoPor, &notas, &c.CreadoEn, &c.CreadoPor, &c.ActualizadoEn, &c.ActualizadoPor); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if mesa.Valid {
			m := mesa.Int64
			c.MesaID = &m
		}
		c.Notas = textoNulo(notas)
		c.Lineas = []dtoresult.ComandaLinea{}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	if len(out) == 0 {
		return out, nil
	}

	posicion := make(map[int64]int, len(out))
	for i, c := range out {
		posicion[c.ID] = i
	}
	lrows, err := db.QueryContext(ctx, `SELECT `+ColLinea+` FROM sch_restaurante.comanda_detalles
		WHERE comanda_id = ANY($1::bigint[]) ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	defer lrows.Close()
	for lrows.Next() {
		l, err := ScanLinea(lrows)
		if err != nil {
			return nil, err
		}
		i := posicion[l.ComandaID]
		out[i].Lineas = append(out[i].Lineas, l)
	}
	return out, lrows.Err()
}

// Comanda lee una comanda con sus líneas; found=false si no existe.
func Comanda(ctx context.Context, db database.DBTX, id int64) (dtoresult.Comanda, bool, error) {
	cs, err := Comandas(ctx, db, []int64{id})
	if err != nil || len(cs) == 0 {
		return dtoresult.Comanda{}, false, err
	}
	return cs[0], true, nil
}
