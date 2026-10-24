package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/platform/database"
)

// UsuarioSQLReader lee el equipo con sus asignaciones activas en una sola consulta.
type UsuarioSQLReader struct{ db database.DBTX }

func NewUsuarioSQLReader(db database.DBTX) *UsuarioSQLReader { return &UsuarioSQLReader{db} }

type rolJSON struct {
	ID       int64  `json:"id"`
	Codigo   string `json:"codigo"`
	Nombre   string `json:"nombre"`
	EsActivo bool   `json:"es_activo"`
}

type sucursalJSON struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo"`
	Nombre string `json:"nombre"`
}

// ListUsuarios excluye siempre al actor técnico (es_sistema) y ordena por id (cursor estable).
// Las asignaciones inactivas no se devuelven: son historial, no acceso vigente.
func (r *UsuarioSQLReader) ListUsuarios(ctx context.Context, despuesDeID int64, limite int) ([]dtoresult.UsuarioConAccesos, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.es_sistema, u.es_activo, u.ultimo_acceso_en, u.bloqueado_hasta,
		       u.intentos_fallidos, u.creado_en, u.creado_por, u.actualizado_en, u.actualizado_por,
		       u.motivo_eliminacion, emp.id, emp.nombres, emp.apellidos, emp.cargo,
		       COALESCE((SELECT json_agg(json_build_object('id', r.id, 'codigo', r.codigo, 'nombre', r.nombre, 'es_activo', r.es_activo) ORDER BY r.id)
		                 FROM sch_restaurante.usuarios_roles ur JOIN sch_restaurante.roles r ON r.id = ur.rol_id
		                 WHERE ur.usuario_id = u.id AND ur.es_activo), '[]'::json),
		       COALESCE((SELECT json_agg(json_build_object('id', s.id, 'codigo', s.codigo, 'nombre', s.nombre) ORDER BY s.id)
		                 FROM sch_restaurante.usuarios_sucursales us JOIN sch_restaurante.sucursales s ON s.id = us.sucursal_id
		                 WHERE us.usuario_id = u.id AND us.es_activo), '[]'::json)
		FROM sch_restaurante.usuarios u
		LEFT JOIN sch_restaurante.empleados emp ON emp.usuario_id = u.id AND emp.es_activo
		WHERE NOT u.es_sistema AND u.id > $1
		ORDER BY u.id LIMIT $2`, despuesDeID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []dtoresult.UsuarioConAccesos
	for rows.Next() {
		var u dtoresult.UsuarioConAccesos
		var email, motivo, empNombres, empApellidos, empCargo sql.NullString
		var empID sql.NullInt64
		var acceso, bloqueo sql.NullTime
		var roles, sedes []byte
		if err := rows.Scan(&u.ID, &u.Username, &email, &u.EsSistema, &u.EsActivo, &acceso, &bloqueo,
			&u.IntentosFallidos, &u.CreadoEn, &u.CreadoPor, &u.ActualizadoEn, &u.ActualizadoPor,
			&motivo, &empID, &empNombres, &empApellidos, &empCargo, &roles, &sedes); err != nil {
			return nil, err
		}
		if empID.Valid {
			e := &dtoresult.EmpleadoResumen{ID: empID.Int64, Nombres: empNombres.String, Apellidos: empApellidos.String}
			if empCargo.Valid {
				e.Cargo = &empCargo.String
			}
			u.Empleado = e
		}
		if email.Valid {
			u.Email = &email.String
		}
		if motivo.Valid {
			u.MotivoBaja = &motivo.String
		}
		if acceso.Valid {
			t := acceso.Time
			u.UltimoAccesoEn = &t
		}
		if bloqueo.Valid {
			t := bloqueo.Time
			u.BloqueadoHasta = &t
		}
		var rs []rolJSON
		var ss []sucursalJSON
		if err := json.Unmarshal(roles, &rs); err != nil {
			return nil, fmt.Errorf("roles del usuario %d: %w", u.ID, err)
		}
		if err := json.Unmarshal(sedes, &ss); err != nil {
			return nil, fmt.Errorf("sucursales del usuario %d: %w", u.ID, err)
		}
		u.Roles = make([]dtoresult.RolResumen, len(rs))
		for i, x := range rs {
			u.Roles[i] = dtoresult.RolResumen{ID: x.ID, Codigo: x.Codigo, Nombre: x.Nombre, EsActivo: x.EsActivo}
		}
		u.Sucursales = make([]dtoresult.SucursalResumen, len(ss))
		for i, x := range ss {
			u.Sucursales[i] = dtoresult.SucursalResumen{ID: x.ID, Codigo: x.Codigo, Nombre: x.Nombre}
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UsuarioSQLReader) ListSucursalesActivas(ctx context.Context, ids []int64) ([]dtoresult.SucursalResumen, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, codigo, nombre FROM sch_restaurante.sucursales
		WHERE id = ANY($1::bigint[]) AND es_activo ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.SucursalResumen
	for rows.Next() {
		var s dtoresult.SucursalResumen
		if err := rows.Scan(&s.ID, &s.Codigo, &s.Nombre); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
