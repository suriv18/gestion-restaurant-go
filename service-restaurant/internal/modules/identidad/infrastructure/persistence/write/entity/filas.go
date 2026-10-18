// Package entity describe filas persistidas; no lleva reglas ni se expone por HTTP.
package entity

import (
	"database/sql"
	"time"
)

// UsuarioRow es una fila de sch_restaurante.usuarios (solo las columnas que usa el módulo).
type UsuarioRow struct {
	ID               int64
	Username         string
	PasswordHash     sql.NullString
	EsSistema        bool
	EsActivo         bool
	IntentosFallidos int
	BloqueadoHasta   sql.NullTime
}

// SesionRow es una fila de sch_restaurante.sesiones.
type SesionRow struct {
	ID                  int64
	UsuarioID           int64
	CreadaEn            time.Time
	UltimoUsoEn         time.Time
	ExpiraInactividadEn time.Time
	ExpiraAbsolutaEn    time.Time
	RevocadaEn          sql.NullTime
}
