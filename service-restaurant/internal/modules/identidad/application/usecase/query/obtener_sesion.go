package query

import (
	"context"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// ObtenerSesion describe al actor ya autenticado (usuario, sedes y permisos vigentes).
type ObtenerSesion struct{}

func NewObtenerSesion() *ObtenerSesion { return &ObtenerSesion{} }

func (uc *ObtenerSesion) Execute(_ context.Context, q qry.ObtenerSesion) (appresult.Result[dtoresult.SesionActual], error) {
	if q.Actor.IsZero() {
		return appresult.Fail[dtoresult.SesionActual](mapper.NoAutenticado()), nil
	}
	return appresult.Ok(dtoresult.SesionActual{
		UsuarioID:  q.Actor.UserID,
		Username:   q.Actor.Username,
		Sucursales: q.Actor.Sucursales(),
		Permisos:   q.Actor.Permisos(),
	}), nil
}
