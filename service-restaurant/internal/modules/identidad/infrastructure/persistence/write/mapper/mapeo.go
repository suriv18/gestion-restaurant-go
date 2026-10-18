// Package mapper convierte entre filas persistidas y el modelo de dominio.
package mapper

import (
	"database/sql"

	"service-restaurant/internal/modules/identidad/domain/model"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/entity"
)

func UsuarioAModelo(r entity.UsuarioRow) model.Usuario {
	u := model.Usuario{
		ID: r.ID, Username: r.Username, PasswordHash: r.PasswordHash.String,
		EsSistema: r.EsSistema, EsActivo: r.EsActivo, IntentosFallidos: r.IntentosFallidos,
	}
	if r.BloqueadoHasta.Valid {
		t := r.BloqueadoHasta.Time
		u.BloqueadoHasta = &t
	}
	return u
}

func SesionARow(s model.Sesion) entity.SesionRow {
	r := entity.SesionRow{
		ID: s.ID, UsuarioID: s.UsuarioID, CreadaEn: s.CreadaEn, UltimoUsoEn: s.UltimoUsoEn,
		ExpiraInactividadEn: s.ExpiraInactividadEn, ExpiraAbsolutaEn: s.ExpiraAbsolutaEn,
	}
	if s.RevocadaEn != nil {
		r.RevocadaEn = sql.NullTime{Time: *s.RevocadaEn, Valid: true}
	}
	return r
}

func SesionAModelo(r entity.SesionRow) model.Sesion {
	s := model.Sesion{
		ID: r.ID, UsuarioID: r.UsuarioID, CreadaEn: r.CreadaEn, UltimoUsoEn: r.UltimoUsoEn,
		ExpiraInactividadEn: r.ExpiraInactividadEn, ExpiraAbsolutaEn: r.ExpiraAbsolutaEn,
	}
	if r.RevocadaEn.Valid {
		t := r.RevocadaEn.Time
		s.RevocadaEn = &t
	}
	return s
}
