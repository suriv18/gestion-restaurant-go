package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// MesaSQLReader lee mesas.
type MesaSQLReader struct{ db database.DBTX }

func NewMesaSQLReader(db database.DBTX) *MesaSQLReader { return &MesaSQLReader{db} }

// ListMesas devuelve las mesas de f.SedeIDs ordenadas por id (cursor estable).
func (r *MesaSQLReader) ListMesas(ctx context.Context, f outbound.FiltroMesas) ([]dtoresult.Mesa, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Mesa+` FROM sch_restaurante.mesas
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2 AND ($3::boolean IS NULL OR es_activo = $3)
		ORDER BY id LIMIT $4`, f.SedeIDs, f.DespuesDeID, activo, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Mesa
	for rows.Next() {
		m, err := columnas.ScanMesa(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MesaSQLReader) GetMesa(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := columnas.ScanMesa(r.db.QueryRowContext(ctx, `SELECT `+columnas.Mesa+` FROM sch_restaurante.mesas WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Mesa{}, false, nil
	}
	return m, err == nil, err
}
