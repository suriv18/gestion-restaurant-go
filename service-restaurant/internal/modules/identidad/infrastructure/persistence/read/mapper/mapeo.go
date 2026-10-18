// Package mapper convierte proyecciones de lectura en tipos de aplicación.
package mapper

import (
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/model"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/projection"
	appctx "service-restaurant/internal/shared/application/context"
)

func SesionAAplicacion(p projection.SesionUsuario) outbound.SesionAutenticada {
	return outbound.SesionAutenticada{
		Username: p.Username,
		Sesion: model.Sesion{
			ID: p.ID, UsuarioID: p.UsuarioID, CreadaEn: p.CreadaEn, UltimoUsoEn: p.UltimoUsoEn,
			ExpiraInactividadEn: p.ExpiraInactividadEn, ExpiraAbsolutaEn: p.ExpiraAbsolutaEn, RevocadaEn: p.RevocadaEn,
		},
	}
}

func GrantsAAplicacion(gs []projection.Grant) []appctx.Grant {
	out := make([]appctx.Grant, len(gs))
	for i, g := range gs {
		out[i] = appctx.Grant{SucursalID: g.SucursalID, Permiso: g.Permiso}
	}
	return out
}
