package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

// Translate convierte los errores de PostgreSQL conocidos en los errores de puerto que los
// casos de uso entienden como rechazos esperados. Cualquier otro error pasa sin cambios.
func Translate(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23505": // unique_violation
		return fmt.Errorf("%w (%s)", outbound.ErrDuplicado, pg.ConstraintName)
	case "23503": // foreign_key_violation
		return fmt.Errorf("%w (%s)", outbound.ErrReferenciaInvalida, pg.ConstraintName)
	case "23514": // check_violation
		return &outbound.ErrReglaDatos{Mensaje: "el valor no cumple una restricción de datos"}
	case "P0001": // RAISE EXCEPTION de las reglas de la V4: mensajes curados del esquema
		return &outbound.ErrReglaDatos{Mensaje: pg.Message}
	}
	return err
}

const (
	rolCols     = `id, codigo, nombre, descripcion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`
	permisoCols = `id, codigo, descripcion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`
	usuarioCols = `id, username, email, es_sistema, es_activo, ultimo_acceso_en, bloqueado_hasta, intentos_fallidos,
		creado_en, creado_por, actualizado_en, actualizado_por`
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

func scanUsuario(s scanner) (dtoresult.UsuarioAdmin, error) {
	var u dtoresult.UsuarioAdmin
	var email sql.NullString
	var acceso, bloqueo sql.NullTime
	err := s.Scan(&u.ID, &u.Username, &email, &u.EsSistema, &u.EsActivo, &acceso, &bloqueo, &u.IntentosFallidos,
		&u.CreadoEn, &u.CreadoPor, &u.ActualizadoEn, &u.ActualizadoPor)
	if email.Valid {
		u.Email = &email.String
	}
	if acceso.Valid {
		t := acceso.Time
		u.UltimoAccesoEn = &t
	}
	if bloqueo.Valid {
		t := bloqueo.Time
		u.BloqueadoHasta = &t
	}
	return u, err
}

func notFound(err error) (bool, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return false, err
}

// RolSQLWriter y PermisoSQLWriter escriben los catálogos; AccesosSQLWriter, usuarios y asignaciones.
type RolSQLWriter struct{ db database.DBTX }

func NewRolSQLWriter(db database.DBTX) *RolSQLWriter { return &RolSQLWriter{db} }

func (r *RolSQLWriter) Insert(ctx context.Context, codigo, nombre string, descripcion *string) (dtoresult.Rol, error) {
	rol, err := scanRol(r.db.QueryRowContext(ctx,
		`INSERT INTO sch_restaurante.roles (codigo, nombre, descripcion) VALUES ($1, $2, $3) RETURNING `+rolCols,
		codigo, nombre, descripcion))
	return rol, Translate(err)
}

func (r *RolSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosRol) (dtoresult.Rol, bool, error) {
	rol, err := scanRol(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.roles
		SET codigo = COALESCE($2, codigo), nombre = COALESCE($3, nombre),
		    descripcion = CASE WHEN $4 THEN $5 ELSE descripcion END
		WHERE id = $1 RETURNING `+rolCols, id, c.Codigo, c.Nombre, c.DescripcionPresente, c.Descripcion))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Rol{}, false, Translate(err)
	}
	return rol, true, nil
}

func (r *RolSQLWriter) SetActivo(ctx context.Context, id int64, activo bool) (dtoresult.Rol, bool, error) {
	rol, err := scanRol(r.db.QueryRowContext(ctx,
		`UPDATE sch_restaurante.roles SET es_activo = $2 WHERE id = $1 RETURNING `+rolCols, id, activo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Rol{}, false, Translate(err)
	}
	return rol, true, nil
}

type PermisoSQLWriter struct{ db database.DBTX }

func NewPermisoSQLWriter(db database.DBTX) *PermisoSQLWriter { return &PermisoSQLWriter{db} }

func (r *PermisoSQLWriter) Insert(ctx context.Context, codigo, descripcion string) (dtoresult.Permiso, error) {
	p, err := scanPermiso(r.db.QueryRowContext(ctx,
		`INSERT INTO sch_restaurante.permisos (codigo, descripcion) VALUES ($1, $2) RETURNING `+permisoCols, codigo, descripcion))
	return p, Translate(err)
}

func (r *PermisoSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosPermiso) (dtoresult.Permiso, bool, error) {
	p, err := scanPermiso(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.permisos
		SET codigo = COALESCE($2, codigo), descripcion = COALESCE($3, descripcion)
		WHERE id = $1 RETURNING `+permisoCols, id, c.Codigo, c.Descripcion))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Permiso{}, false, Translate(err)
	}
	return p, true, nil
}

func (r *PermisoSQLWriter) SetActivo(ctx context.Context, id int64, activo bool) (dtoresult.Permiso, bool, error) {
	p, err := scanPermiso(r.db.QueryRowContext(ctx,
		`UPDATE sch_restaurante.permisos SET es_activo = $2 WHERE id = $1 RETURNING `+permisoCols, id, activo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Permiso{}, false, Translate(err)
	}
	return p, true, nil
}

type AccesosSQLWriter struct{ db database.DBTX }

func NewAccesosSQLWriter(db database.DBTX) *AccesosSQLWriter { return &AccesosSQLWriter{db} }

func (r *AccesosSQLWriter) InsertUsuario(ctx context.Context, username string, email *string, hash string) (dtoresult.UsuarioAdmin, error) {
	u, err := scanUsuario(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.usuarios (username, email, password_hash) VALUES ($1, $2, $3)
		RETURNING `+usuarioCols, username, email, hash))
	return u, Translate(err)
}

func (r *AccesosSQLWriter) GetUsuario(ctx context.Context, id int64) (dtoresult.UsuarioAdmin, bool, error) {
	u, err := scanUsuario(r.db.QueryRowContext(ctx,
		`SELECT `+usuarioCols+` FROM sch_restaurante.usuarios WHERE id = $1`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.UsuarioAdmin{}, false, err
	}
	return u, true, nil
}

func (r *AccesosSQLWriter) SucursalesActivas(ctx context.Context, usuarioID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sucursal_id FROM sch_restaurante.usuarios_sucursales WHERE usuario_id = $1 AND es_activo ORDER BY 1`, usuarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReemplazarAsignaciones deja activas exactamente las filas indicadas, sin duplicar:
// desactiva las que sobran, y reactiva o inserta las pedidas (UNIQUE usuario+rol / usuario+sede).
func (r *AccesosSQLWriter) ReemplazarAsignaciones(ctx context.Context, usuarioID int64, rolIDs, sucursalIDs []int64) error {
	stmts := []struct {
		sql  string
		args []any
	}{
		{`UPDATE sch_restaurante.usuarios_roles SET es_activo = false
		  WHERE usuario_id = $1 AND es_activo AND NOT (rol_id = ANY($2::bigint[]))`, []any{usuarioID, rolIDs}},
		{`INSERT INTO sch_restaurante.usuarios_roles (usuario_id, rol_id)
		  SELECT $1, x FROM unnest($2::bigint[]) AS x
		  ON CONFLICT (usuario_id, rol_id) DO UPDATE SET es_activo = true
		  WHERE sch_restaurante.usuarios_roles.es_activo = false`, []any{usuarioID, rolIDs}},
		{`UPDATE sch_restaurante.usuarios_sucursales SET es_activo = false
		  WHERE usuario_id = $1 AND es_activo AND NOT (sucursal_id = ANY($2::bigint[]))`, []any{usuarioID, sucursalIDs}},
		{`INSERT INTO sch_restaurante.usuarios_sucursales (usuario_id, sucursal_id)
		  SELECT $1, x FROM unnest($2::bigint[]) AS x
		  ON CONFLICT (usuario_id, sucursal_id) DO UPDATE SET es_activo = true
		  WHERE sch_restaurante.usuarios_sucursales.es_activo = false`, []any{usuarioID, sucursalIDs}},
	}
	for _, s := range stmts {
		if _, err := r.db.ExecContext(ctx, s.sql, s.args...); err != nil {
			return Translate(err)
		}
	}
	return nil
}

func (r *AccesosSQLWriter) SetActivoUsuario(ctx context.Context, id int64, activo bool, motivo string) (dtoresult.UsuarioAdmin, error) {
	u, err := scanUsuario(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.usuarios
		SET es_activo = $2, motivo_eliminacion = CASE WHEN $2 THEN NULL ELSE $3::text END
		WHERE id = $1 RETURNING `+usuarioCols, id, activo, motivo))
	return u, Translate(err)
}

func (r *AccesosSQLWriter) RevocarSesiones(ctx context.Context, usuarioID int64, ahora time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sch_restaurante.sesiones SET revocada_en = $2 WHERE usuario_id = $1 AND revocada_en IS NULL`, usuarioID, ahora)
	return err
}

func (r *AccesosSQLWriter) ContarAdministradores(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT count(DISTINCT usuario_id) FROM sch_restaurante.v_permisos_efectivos WHERE permiso = 'USUARIO_ADMINISTRAR'`).Scan(&n)
	return n, err
}
