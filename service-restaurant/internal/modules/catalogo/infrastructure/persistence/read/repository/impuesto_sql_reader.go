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

// ImpuestoSQLReader lee impuestos y tasas de impuesto.
type ImpuestoSQLReader struct{ db database.DBTX }

func NewImpuestoSQLReader(db database.DBTX) *ImpuestoSQLReader { return &ImpuestoSQLReader{db} }

// ListarImpuestos ordena por id (cursor estable).
func (r *ImpuestoSQLReader) ListarImpuestos(ctx context.Context, f outbound.FiltroImpuestos) ([]dtoresult.Impuesto, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Impuesto+` FROM sch_restaurante.impuestos
		WHERE id > $1 AND ($2::boolean IS NULL OR es_activo = $2)
		ORDER BY id LIMIT $3`, f.DespuesDeID, activoSQL(f.Activo), f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Impuesto
	for rows.Next() {
		i, err := columnas.ScanImpuesto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *ImpuestoSQLReader) ObtenerImpuesto(ctx context.Context, id int64) (dtoresult.Impuesto, bool, error) {
	i, err := columnas.ScanImpuesto(r.db.QueryRowContext(ctx, `SELECT `+columnas.Impuesto+` FROM sch_restaurante.impuestos WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Impuesto{}, false, nil
	}
	return i, err == nil, err
}

// ListarTasas ordena por id; ImpuestoID 0 no filtra.
func (r *ImpuestoSQLReader) ListarTasas(ctx context.Context, f outbound.FiltroTasas) ([]dtoresult.TasaImpuesto, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.TasaImpuesto+` FROM sch_restaurante.tasas_impuesto
		WHERE id > $1 AND ($2::bigint = 0 OR impuesto_id = $2)
		ORDER BY id LIMIT $3`, f.DespuesDeID, f.ImpuestoID, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.TasaImpuesto
	for rows.Next() {
		t, err := columnas.ScanTasaImpuesto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *ImpuestoSQLReader) ObtenerTasa(ctx context.Context, id int64) (dtoresult.TasaImpuesto, bool, error) {
	t, err := columnas.ScanTasaImpuesto(r.db.QueryRowContext(ctx, `SELECT `+columnas.TasaImpuesto+` FROM sch_restaurante.tasas_impuesto WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.TasaImpuesto{}, false, nil
	}
	return t, err == nil, err
}
