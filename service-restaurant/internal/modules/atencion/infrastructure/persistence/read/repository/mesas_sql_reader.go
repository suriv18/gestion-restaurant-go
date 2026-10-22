package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

// MesaSQLReader lee mesas.
type MesaSQLReader struct{ db database.DBTX }

func NewMesaSQLReader(db database.DBTX) *MesaSQLReader { return &MesaSQLReader{db} }

// Las posiciones son numeric(10,2): se leen como texto para no perder exactitud.
const mesaCols = `id, sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x::text, posicion_y::text,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func cadenaNula(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func leerMesa(s fila) (dtoresult.Mesa, error) {
	var m dtoresult.Mesa
	var px, py, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&m.ID, &m.SucursalID, &m.ZonaID, &m.NumeroMesa, &m.Capacidad, &m.Habilitada, &px, &py,
		&m.EsActivo, &motivo, &elimEn, &elimPor, &m.CreadoEn, &m.CreadoPor, &m.ActualizadoEn, &m.ActualizadoPor)
	m.PosicionX, m.PosicionY, m.MotivoEliminacion = cadenaNula(px), cadenaNula(py), cadenaNula(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		m.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		m.EliminadoPor = &v
	}
	return m, err
}

// ListMesas devuelve las mesas de f.SedeIDs ordenadas por id (cursor estable).
func (r *MesaSQLReader) ListMesas(ctx context.Context, f outbound.FiltroMesas) ([]dtoresult.Mesa, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+mesaCols+` FROM sch_restaurante.mesas
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2 AND ($3::boolean IS NULL OR es_activo = $3)
		ORDER BY id LIMIT $4`, f.SedeIDs, f.DespuesDeID, activo, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Mesa
	for rows.Next() {
		m, err := leerMesa(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MesaSQLReader) GetMesa(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := leerMesa(r.db.QueryRowContext(ctx, `SELECT `+mesaCols+` FROM sch_restaurante.mesas WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Mesa{}, false, nil
	}
	return m, err == nil, err
}
