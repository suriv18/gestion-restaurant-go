// Package repository contiene el SQL parametrizado de lectura del módulo identidad.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/projection"
	"service-restaurant/internal/platform/database"
)

type SesionSQLReader struct{ db database.DBTX }

func NewSesionSQLReader(db database.DBTX) *SesionSQLReader { return &SesionSQLReader{db} }

// FindByID solo devuelve sesiones de usuarios activos y no técnicos: dar de baja a un
// usuario invalida sus sesiones de inmediato.
func (r *SesionSQLReader) FindByID(ctx context.Context, id int64) (projection.SesionUsuario, bool, error) {
	var p projection.SesionUsuario
	var revocada sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT s.id, s.usuario_id, u.username, s.creada_en, s.ultimo_uso_en,
		       s.expira_inactividad_en, s.expira_absoluta_en, s.revocada_en
		FROM sch_restaurante.sesiones s
		JOIN sch_restaurante.usuarios u ON u.id = s.usuario_id
		WHERE s.id = $1 AND u.es_activo AND NOT u.es_sistema`, id).
		Scan(&p.ID, &p.UsuarioID, &p.Username, &p.CreadaEn, &p.UltimoUsoEn,
			&p.ExpiraInactividadEn, &p.ExpiraAbsolutaEn, &revocada)
	if errors.Is(err, sql.ErrNoRows) {
		return projection.SesionUsuario{}, false, nil
	}
	if err != nil {
		return projection.SesionUsuario{}, false, err
	}
	if revocada.Valid {
		t := revocada.Time
		p.RevocadaEn = &t
	}
	return p, true, nil
}

// Grants lee los permisos efectivos (todos los eslabones activos: usuario, rol, permiso, sede, negocio).
func (r *SesionSQLReader) Grants(ctx context.Context, usuarioID int64) ([]projection.Grant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sucursal_id, permiso FROM sch_restaurante.v_permisos_efectivos
		WHERE usuario_id = $1 ORDER BY sucursal_id, permiso`, usuarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []projection.Grant
	for rows.Next() {
		var g projection.Grant
		if err := rows.Scan(&g.SucursalID, &g.Permiso); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
