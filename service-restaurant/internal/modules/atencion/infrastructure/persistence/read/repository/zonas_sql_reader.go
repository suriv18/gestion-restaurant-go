// Package repository contiene el SQL parametrizado de lectura del módulo atencion.
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

// ZonaSQLReader lee zonas.
type ZonaSQLReader struct{ db database.DBTX }

func NewZonaSQLReader(db database.DBTX) *ZonaSQLReader { return &ZonaSQLReader{db} }

// ListZonas devuelve las zonas de f.SedeIDs ordenadas por id (cursor estable).
func (r *ZonaSQLReader) ListZonas(ctx context.Context, f outbound.FiltroZonas) ([]dtoresult.Zona, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Zona+` FROM sch_restaurante.zonas
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2 AND ($3::boolean IS NULL OR es_activo = $3)
		ORDER BY id LIMIT $4`, f.SedeIDs, f.DespuesDeID, activo, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Zona
	for rows.Next() {
		z, err := columnas.ScanZona(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

func (r *ZonaSQLReader) GetZona(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	z, err := columnas.ScanZona(r.db.QueryRowContext(ctx, `SELECT `+columnas.Zona+` FROM sch_restaurante.zonas WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Zona{}, false, nil
	}
	return z, err == nil, err
}
