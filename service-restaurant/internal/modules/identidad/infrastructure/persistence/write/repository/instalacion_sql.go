package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"service-restaurant/internal/platform/database"
)

// InstalacionSQLWriter soporta el alta inicial (negocio, sucursal y primer administrador).
type InstalacionSQLWriter struct{ db database.DBTX }

func NewInstalacionSQLWriter(db database.DBTX) *InstalacionSQLWriter {
	return &InstalacionSQLWriter{db}
}

func (r *InstalacionSQLWriter) ExisteUsuarioHumano(ctx context.Context) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM sch_restaurante.usuarios WHERE NOT es_sistema)`).Scan(&ok)
	return ok, err
}

// NegocioID devuelve el negocio único, si existe.
func (r *InstalacionSQLWriter) NegocioID(ctx context.Context) (int64, bool, error) {
	return first(r.db.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.negocio LIMIT 1`))
}

func (r *InstalacionSQLWriter) InsertNegocio(ctx context.Context, ruc, razon, comercial, direccion string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.negocio (ruc, razon_social, nombre_comercial, direccion_fiscal)
		VALUES ($1, $2, $3, $4) RETURNING id`, ruc, razon, comercial, direccion).Scan(&id)
	return id, err
}

func (r *InstalacionSQLWriter) SucursalPorCodigo(ctx context.Context, codigo string) (int64, bool, error) {
	return first(r.db.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.sucursales WHERE codigo = $1`, codigo))
}

func (r *InstalacionSQLWriter) InsertSucursal(ctx context.Context, negocioID int64, codigo, nombre, direccion string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.sucursales (negocio_id, codigo, nombre, direccion)
		VALUES ($1, $2, $3, $4) RETURNING id`, negocioID, codigo, nombre, direccion).Scan(&id)
	return id, err
}

// InsertAdministrador crea el usuario y le asigna el rol ADMIN y la sucursal.
func (r *InstalacionSQLWriter) InsertAdministrador(ctx context.Context, username, hash string, sucursalID int64) (int64, error) {
	var id int64
	if err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.usuarios (username, password_hash) VALUES ($1, $2) RETURNING id`,
		username, hash).Scan(&id); err != nil {
		return 0, err
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO sch_restaurante.usuarios_roles (usuario_id, rol_id)
		SELECT $1, id FROM sch_restaurante.roles WHERE codigo = 'ADMIN'`, id)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, fmt.Errorf("no existe el rol ADMIN (¿migraciones aplicadas?)")
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO sch_restaurante.usuarios_sucursales (usuario_id, sucursal_id) VALUES ($1, $2)`,
		id, sucursalID); err != nil {
		return 0, err
	}
	return id, nil
}

func first(row *sql.Row) (int64, bool, error) {
	var id int64
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}
