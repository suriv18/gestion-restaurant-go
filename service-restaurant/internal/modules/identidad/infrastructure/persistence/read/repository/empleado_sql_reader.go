package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/internal/empleadocol"
	"service-restaurant/internal/platform/database"
)

// EmpleadoSQLReader lee empleados con el usuario vinculado.
type EmpleadoSQLReader struct{ db database.DBTX }

func NewEmpleadoSQLReader(db database.DBTX) *EmpleadoSQLReader { return &EmpleadoSQLReader{db} }

// ListEmpleados ordena por id (cursor estable). El texto, ya con los comodines escapados, busca
// por nombres y apellidos (contiene) o por número de documento (empieza por).
func (r *EmpleadoSQLReader) ListEmpleados(ctx context.Context, f outbound.FiltroEmpleados) ([]dtoresult.Empleado, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+empleadocol.Columnas+` FROM `+empleadocol.Desde+`
		WHERE e.id > $1 AND ($2::boolean IS NULL OR e.es_activo = $2)
		  AND ($3::text = '' OR (e.nombres || ' ' || e.apellidos) ILIKE '%' || $3 || '%' ESCAPE '\'
		       OR e.numero_documento ILIKE $3 || '%' ESCAPE '\')
		ORDER BY e.id LIMIT $4`, f.DespuesDeID, activo, f.Texto, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Empleado
	for rows.Next() {
		e, err := empleadocol.Scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *EmpleadoSQLReader) GetEmpleado(ctx context.Context, id int64) (dtoresult.Empleado, bool, error) {
	e, err := empleadocol.Scan(r.db.QueryRowContext(ctx,
		`SELECT `+empleadocol.Columnas+` FROM `+empleadocol.Desde+` WHERE e.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Empleado{}, false, nil
	}
	return e, err == nil, err
}
