package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// EstacionSQLReader lee estaciones de preparación.
type EstacionSQLReader struct{ db database.DBTX }

func NewEstacionSQLReader(db database.DBTX) *EstacionSQLReader { return &EstacionSQLReader{db} }

// ListarEstaciones devuelve las de f.SedeIDs ordenadas por id (cursor estable); el texto, ya con
// los comodines escapados, busca por código o nombre (contiene).
func (r *EstacionSQLReader) ListarEstaciones(ctx context.Context, f outbound.FiltroEstaciones) ([]dtoresult.Estacion, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Estacion+` FROM sch_restaurante.estaciones_preparacion
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2 AND ($3::boolean IS NULL OR es_activo = $3)
		  AND ($4::text = '' OR codigo ILIKE '%' || $4 || '%' ESCAPE '\' OR nombre ILIKE '%' || $4 || '%' ESCAPE '\')
		ORDER BY id LIMIT $5`, f.SedeIDs, f.DespuesDeID, activoSQL(f.Activo), f.Texto, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Estacion
	for rows.Next() {
		e, err := columnas.ScanEstacion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EstacionSQLReader) ObtenerEstacion(ctx context.Context, id int64) (dtoresult.Estacion, bool, error) {
	e, err := columnas.ScanEstacion(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Estacion+` FROM sch_restaurante.estaciones_preparacion WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Estacion{}, false, nil
	}
	return e, err == nil, err
}
