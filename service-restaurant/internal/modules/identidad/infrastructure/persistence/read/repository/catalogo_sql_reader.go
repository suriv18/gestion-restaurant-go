package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/platform/database"
)

const (
	rolCols     = `id, codigo, nombre, descripcion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`
	permisoCols = `id, codigo, descripcion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`
)

type scanner interface{ Scan(...any) error }

func scanRol(s scanner) (dtoresult.Rol, error) {
	var r dtoresult.Rol
	var desc sql.NullString
	err := s.Scan(&r.ID, &r.Codigo, &r.Nombre, &desc, &r.EsActivo, &r.CreadoEn, &r.CreadoPor, &r.ActualizadoEn, &r.ActualizadoPor)
	if desc.Valid {
		r.Descripcion = &desc.String
	}
	return r, err
}

func scanPermiso(s scanner) (dtoresult.Permiso, error) {
	var p dtoresult.Permiso
	err := s.Scan(&p.ID, &p.Codigo, &p.Descripcion, &p.EsActivo, &p.CreadoEn, &p.CreadoPor, &p.ActualizadoEn, &p.ActualizadoPor)
	return p, err
}

// CatalogoSQLReader lee roles y permisos con orden estable por id y paginación por cursor.
type CatalogoSQLReader struct{ db database.DBTX }

func NewCatalogoSQLReader(db database.DBTX) *CatalogoSQLReader { return &CatalogoSQLReader{db} }

func (r *CatalogoSQLReader) ListRoles(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.Rol, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+rolCols+` FROM sch_restaurante.roles WHERE id > $1 ORDER BY id LIMIT $2`, despuesDeID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Rol
	for rows.Next() {
		x, err := scanRol(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *CatalogoSQLReader) GetRol(ctx context.Context, id int64) (dtoresult.Rol, bool, error) {
	x, err := scanRol(r.db.QueryRowContext(ctx, `SELECT `+rolCols+` FROM sch_restaurante.roles WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Rol{}, false, nil
	}
	return x, err == nil, err
}

func (r *CatalogoSQLReader) ListPermisos(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.Permiso, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+permisoCols+` FROM sch_restaurante.permisos WHERE id > $1 ORDER BY id LIMIT $2`, despuesDeID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Permiso
	for rows.Next() {
		x, err := scanPermiso(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *CatalogoSQLReader) GetPermiso(ctx context.Context, id int64) (dtoresult.Permiso, bool, error) {
	x, err := scanPermiso(r.db.QueryRowContext(ctx, `SELECT `+permisoCols+` FROM sch_restaurante.permisos WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Permiso{}, false, nil
	}
	return x, err == nil, err
}
