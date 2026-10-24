package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/internal/empleadocol"
	"service-restaurant/internal/platform/database"
)

// Restricciones únicas de empleados que tienen un rechazo propio.
const (
	restriccionDocumentoEmpleado = "empleados_tipo_documento_numero_documento_key"
	restriccionUsuarioEmpleado   = "empleados_usuario_id_key"
)

// traducirEmpleado distingue las colisiones del documento y del usuario por el nombre de la
// restricción violada; lo demás pasa por Translate.
func traducirEmpleado(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		switch pg.ConstraintName {
		case restriccionDocumentoEmpleado:
			return outbound.ErrDocumentoEmpleadoDuplicado
		case restriccionUsuarioEmpleado:
			return outbound.ErrUsuarioYaVinculado
		}
	}
	return Translate(err)
}

// EmpleadoSQLWriter opera sobre sch_restaurante.empleados. Actor y fechas de auditoría los
// completa el trigger de metadatos desde app.usuario_id. Cada escritura devuelve la fila ya unida
// con su usuario mediante un CTE.
type EmpleadoSQLWriter struct{ db database.DBTX }

func NewEmpleadoSQLWriter(db database.DBTX) *EmpleadoSQLWriter { return &EmpleadoSQLWriter{db} }

const selectDeCTE = `SELECT ` + empleadocol.Columnas + ` FROM ` + empleadocol.DesdeCTE

func (r *EmpleadoSQLWriter) Insert(ctx context.Context, n outbound.NuevoEmpleado) (dtoresult.Empleado, error) {
	e, err := empleadocol.Scan(r.db.QueryRowContext(ctx, `
		WITH e AS (
			INSERT INTO sch_restaurante.empleados (usuario_id, tipo_documento, numero_documento, nombres, apellidos, telefono, cargo, fecha_ingreso)
			VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8::text::date, CURRENT_DATE)) RETURNING *)
		`+selectDeCTE,
		n.UsuarioID, n.TipoDocumento, n.NumeroDocumento, n.Nombres, n.Apellidos, n.Telefono, n.Cargo, n.FechaIngreso))
	return e, traducirEmpleado(err)
}

// GetForUpdate lee el empleado bloqueando su fila hasta el final de la transacción.
func (r *EmpleadoSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Empleado, bool, error) {
	e, err := empleadocol.Scan(r.db.QueryRowContext(ctx,
		`SELECT `+empleadocol.Columnas+` FROM `+empleadocol.Desde+` WHERE e.id = $1 FOR UPDATE OF e`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Empleado{}, false, err
	}
	return e, true, nil
}

func (r *EmpleadoSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosEmpleado) (dtoresult.Empleado, bool, error) {
	e, err := empleadocol.Scan(r.db.QueryRowContext(ctx, `
		WITH e AS (
			UPDATE sch_restaurante.empleados SET
				tipo_documento = COALESCE($2, tipo_documento),
				numero_documento = COALESCE($3, numero_documento),
				nombres = COALESCE($4, nombres),
				apellidos = COALESCE($5, apellidos),
				fecha_ingreso = COALESCE($6::text::date, fecha_ingreso),
				telefono = CASE WHEN $7 THEN $8::text ELSE telefono END,
				cargo = CASE WHEN $9 THEN $10::text ELSE cargo END,
				fecha_cese = CASE WHEN $11 THEN $12::text::date ELSE fecha_cese END,
				usuario_id = CASE WHEN $13 THEN $14::bigint ELSE usuario_id END
			WHERE id = $1 RETURNING *)
		`+selectDeCTE, id, c.TipoDocumento, c.NumeroDocumento, c.Nombres, c.Apellidos, c.FechaIngreso,
		c.Telefono.Presente, c.Telefono.Valor, c.Cargo.Presente, c.Cargo.Valor,
		c.FechaCese.Presente, c.FechaCese.Valor, c.UsuarioID.Presente, c.UsuarioID.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Empleado{}, false, traducirEmpleado(err)
	}
	return e, true, nil
}

// SetBaja desactiva al empleado; el trigger de metadatos completa actor y fecha de la baja.
func (r *EmpleadoSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Empleado, bool, error) {
	e, err := empleadocol.Scan(r.db.QueryRowContext(ctx, `
		WITH e AS (
			UPDATE sch_restaurante.empleados SET es_activo = false, motivo_eliminacion = $2
			WHERE id = $1 RETURNING *)
		`+selectDeCTE, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Empleado{}, false, traducirEmpleado(err)
	}
	return e, true, nil
}

// SetActivo reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja.
func (r *EmpleadoSQLWriter) SetActivo(ctx context.Context, id int64) (dtoresult.Empleado, bool, error) {
	e, err := empleadocol.Scan(r.db.QueryRowContext(ctx, `
		WITH e AS (
			UPDATE sch_restaurante.empleados SET es_activo = true WHERE id = $1 RETURNING *)
		`+selectDeCTE, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Empleado{}, false, traducirEmpleado(err)
	}
	return e, true, nil
}

// GetUsuario lee el usuario en modo compartido: no puede darse de baja hasta que termine el comando.
func (r *EmpleadoSQLWriter) GetUsuario(ctx context.Context, id int64) (outbound.UsuarioRef, bool, error) {
	var u outbound.UsuarioRef
	err := r.db.QueryRowContext(ctx,
		`SELECT es_sistema, es_activo FROM sch_restaurante.usuarios WHERE id = $1 FOR SHARE`, id).Scan(&u.EsSistema, &u.EsActivo)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.UsuarioRef{}, false, err
	}
	return u, true, nil
}

// UsuarioActivoDe indica si el empleado tiene un usuario vinculado y activo.
func (r *EmpleadoSQLWriter) UsuarioActivoDe(ctx context.Context, empleadoID int64) (bool, error) {
	var activo bool
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(u.es_activo, false) FROM sch_restaurante.empleados e
		LEFT JOIN sch_restaurante.usuarios u ON u.id = e.usuario_id WHERE e.id = $1`, empleadoID).Scan(&activo)
	if missing, err := notFound(err); missing || err != nil {
		return false, err
	}
	return activo, nil
}
