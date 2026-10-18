// Package projection describe los resultados de lectura; permanece dentro del adaptador.
package projection

import "time"

// SesionUsuario es la sesión unida al usuario que la posee.
type SesionUsuario struct {
	ID                  int64
	UsuarioID           int64
	Username            string
	CreadaEn            time.Time
	UltimoUsoEn         time.Time
	ExpiraInactividadEn time.Time
	ExpiraAbsolutaEn    time.Time
	RevocadaEn          *time.Time
}

// Grant es una fila de v_permisos_efectivos.
type Grant struct {
	SucursalID int64
	Permiso    string
}
