// Package columnas reúne las columnas y el escaneo de filas de zonas y mesas que comparten el
// lado de lectura y el de escritura. Vive bajo persistence/internal: solo la persistencia del
// módulo puede importarlo, y no acopla un lado con el otro.
package columnas

import (
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// Fila la cumplen *sql.Row y *sql.Rows.
type Fila interface{ Scan(...any) error }

// Las posiciones son numeric(10,2): se leen como texto para no perder exactitud ni depender de
// la codificación del driver (al escribir se envían vía ::text::numeric).
const (
	Mesa = `id, sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x::text, posicion_y::text,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

	Zona = `id, sucursal_id, nombre, orden_visualizacion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`
)

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

// ScanMesa lee una fila con las columnas de Mesa.
func ScanMesa(s Fila) (dtoresult.Mesa, error) {
	var m dtoresult.Mesa
	var px, py, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&m.ID, &m.SucursalID, &m.ZonaID, &m.NumeroMesa, &m.Capacidad, &m.Habilitada, &px, &py,
		&m.EsActivo, &motivo, &elimEn, &elimPor, &m.CreadoEn, &m.CreadoPor, &m.ActualizadoEn, &m.ActualizadoPor)
	m.PosicionX, m.PosicionY, m.MotivoEliminacion = textoNulo(px), textoNulo(py), textoNulo(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		m.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		m.EliminadoPor = &v
	}
	return m, err
}

// ScanZona lee una fila con las columnas de Zona.
func ScanZona(s Fila) (dtoresult.Zona, error) {
	var z dtoresult.Zona
	err := s.Scan(&z.ID, &z.SucursalID, &z.Nombre, &z.OrdenVisualizacion, &z.EsActivo,
		&z.CreadoEn, &z.CreadoPor, &z.ActualizadoEn, &z.ActualizadoPor)
	return z, err
}
