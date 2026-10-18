// Package repository contiene el SQL parametrizado de escritura del módulo identidad.
// Recibe un database.DBTX: en operaciones de escritura es el *sql.Tx de la unidad de trabajo.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/entity"
	"service-restaurant/internal/platform/database"
)

// UsuarioSQLWriter opera sobre sch_restaurante.usuarios.
type UsuarioSQLWriter struct{ db database.DBTX }

func NewUsuarioSQLWriter(db database.DBTX) *UsuarioSQLWriter { return &UsuarioSQLWriter{db} }

// FindByKeyForUpdate busca por lower(btrim(username)) y bloquea la fila.
func (r *UsuarioSQLWriter) FindByKeyForUpdate(ctx context.Context, key string) (entity.UsuarioRow, bool, error) {
	var u entity.UsuarioRow
	err := r.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, es_sistema, es_activo, intentos_fallidos, bloqueado_hasta
		FROM sch_restaurante.usuarios
		WHERE lower(btrim(username)) = $1
		FOR UPDATE`, key).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.EsSistema, &u.EsActivo, &u.IntentosFallidos, &u.BloqueadoHasta)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.UsuarioRow{}, false, nil
	}
	return u, err == nil, err
}

func (r *UsuarioSQLWriter) UpdateIntentos(ctx context.Context, id int64, intentos int, bloqueadoHasta sql.NullTime) error {
	return exactlyOne(r.db.ExecContext(ctx, `
		UPDATE sch_restaurante.usuarios SET intentos_fallidos = $2, bloqueado_hasta = $3 WHERE id = $1`,
		id, intentos, bloqueadoHasta))
}

func (r *UsuarioSQLWriter) UpdateAcceso(ctx context.Context, id int64, ahora time.Time) error {
	return exactlyOne(r.db.ExecContext(ctx, `
		UPDATE sch_restaurante.usuarios
		SET intentos_fallidos = 0, bloqueado_hasta = NULL, ultimo_acceso_en = $2 WHERE id = $1`, id, ahora))
}

// SistemaID devuelve el actor técnico creado por la instalación de la V4.
func (r *UsuarioSQLWriter) SistemaID(ctx context.Context) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.usuarios WHERE es_sistema`).Scan(&id)
	return id, err
}

// SesionSQLWriter opera sobre sch_restaurante.sesiones.
type SesionSQLWriter struct{ db database.DBTX }

func NewSesionSQLWriter(db database.DBTX) *SesionSQLWriter { return &SesionSQLWriter{db} }

func (r *SesionSQLWriter) Insert(ctx context.Context, s entity.SesionRow) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.sesiones
			(usuario_id, creada_en, ultimo_uso_en, expira_inactividad_en, expira_absoluta_en)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		s.UsuarioID, s.CreadaEn, s.UltimoUsoEn, s.ExpiraInactividadEn, s.ExpiraAbsolutaEn).Scan(&id)
	return id, err
}

// Revoke es idempotente: revocar una sesión ya revocada no es un error.
func (r *SesionSQLWriter) Revoke(ctx context.Context, id int64, ahora time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sch_restaurante.sesiones SET revocada_en = $2 WHERE id = $1 AND revocada_en IS NULL`, id, ahora)
	return err
}

func (r *SesionSQLWriter) Touch(ctx context.Context, id int64, ultimoUso, expiraInactividad time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sch_restaurante.sesiones SET ultimo_uso_en = $2, expira_inactividad_en = $3
		WHERE id = $1 AND revocada_en IS NULL`, id, ultimoUso, expiraInactividad)
	return err
}

func exactlyOne(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("se esperaba actualizar exactamente una fila")
	}
	return nil
}
