// Package empleadocol reúne las columnas y el escaneo de filas de empleados que comparten el lado
// de lectura y el de escritura. Vive bajo persistence/internal: solo la persistencia del módulo
// puede importarlo, y no acopla un lado con el otro.
package empleadocol

import (
	"database/sql"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
)

// Fila la cumplen *sql.Row y *sql.Rows.
type Fila interface{ Scan(...any) error }

// Columnas espera la tabla de empleados con alias e y la de usuarios con alias u (LEFT JOIN).
// Las fechas se leen como texto (AAAA-MM-DD) para no depender de la zona horaria del driver.
const Columnas = `e.id, e.usuario_id, u.username, e.tipo_documento, e.numero_documento, e.nombres, e.apellidos,
	e.telefono, e.cargo, e.fecha_ingreso::text, e.fecha_cese::text, e.es_activo, e.motivo_eliminacion,
	e.eliminado_en, e.eliminado_por, e.creado_en, e.creado_por, e.actualizado_en, e.actualizado_por`

// Desde es el origen común de la lectura: empleados con su usuario vinculado, si lo hay.
const Desde = `sch_restaurante.empleados e LEFT JOIN sch_restaurante.usuarios u ON u.id = e.usuario_id`

// DesdeCTE une el resultado de una escritura (CTE llamado e, con RETURNING *) con su usuario.
const DesdeCTE = `e LEFT JOIN sch_restaurante.usuarios u ON u.id = e.usuario_id`

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

// Scan lee una fila con las columnas de Columnas.
func Scan(s Fila) (dtoresult.Empleado, error) {
	var e dtoresult.Empleado
	var usuarioID, elimPor sql.NullInt64
	var username, tel, cargo, cese, motivo sql.NullString
	var elimEn sql.NullTime
	err := s.Scan(&e.ID, &usuarioID, &username, &e.TipoDocumento, &e.NumeroDocumento, &e.Nombres, &e.Apellidos,
		&tel, &cargo, &e.FechaIngreso, &cese, &e.EsActivo, &motivo,
		&elimEn, &elimPor, &e.CreadoEn, &e.CreadoPor, &e.ActualizadoEn, &e.ActualizadoPor)
	if usuarioID.Valid {
		v := usuarioID.Int64
		e.UsuarioID = &v
	}
	if elimPor.Valid {
		v := elimPor.Int64
		e.EliminadoPor = &v
	}
	if elimEn.Valid {
		t := elimEn.Time
		e.EliminadoEn = &t
	}
	e.Username, e.Telefono, e.Cargo = textoNulo(username), textoNulo(tel), textoNulo(cargo)
	e.FechaCese, e.MotivoEliminacion = textoNulo(cese), textoNulo(motivo)
	return e, err
}
