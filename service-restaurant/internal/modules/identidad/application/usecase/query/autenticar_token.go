// Package query contiene los casos de uso de lectura del módulo identidad.
package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// AutenticarToken valida el JWT y, además, la sesión en BD: el token firmado no basta.
// Así revocar, la inactividad, dar de baja al usuario o desactivar un rol surten efecto
// de inmediato (RES-RF-002, RES-RF-033).
type AutenticarToken struct {
	lector   outbound.SesionReader
	escritor outbound.SesionWriter
	tokens   outbound.TokenIssuer
	clock    outbound.Clock
	sesion   policy.PoliticaSesion
}

func NewAutenticarToken(
	lector outbound.SesionReader, escritor outbound.SesionWriter, tokens outbound.TokenIssuer,
	clock outbound.Clock, sesion policy.PoliticaSesion,
) *AutenticarToken {
	return &AutenticarToken{lector, escritor, tokens, clock, sesion}
}

func (uc *AutenticarToken) Execute(ctx context.Context, q qry.AutenticarToken) (appresult.Result[dtoresult.TokenAutenticado], error) {
	rechazo := appresult.Fail[dtoresult.TokenAutenticado](mapper.NoAutenticado())
	ahora := uc.clock.Now()

	claims, ok := uc.tokens.Verificar(q.Token, ahora)
	if !ok {
		return rechazo, nil
	}
	s, found, err := uc.lector.BuscarPorID(ctx, claims.SesionID)
	if err != nil {
		return appresult.Result[dtoresult.TokenAutenticado]{}, fmt.Errorf("autenticar token: sesión: %w", err)
	}
	if !found || s.Sesion.UsuarioID != claims.UsuarioID || !s.Sesion.Vigente(ahora) {
		return rechazo, nil
	}
	grants, err := uc.lector.PermisosEfectivos(ctx, s.Sesion.UsuarioID)
	if err != nil {
		return appresult.Result[dtoresult.TokenAutenticado]{}, fmt.Errorf("autenticar token: permisos: %w", err)
	}
	if uc.sesion.DebeRegistrarUso(s.Sesion, ahora) {
		if err := uc.escritor.RegistrarUso(ctx, uc.sesion.Extender(s.Sesion, ahora)); err != nil {
			return appresult.Result[dtoresult.TokenAutenticado]{}, fmt.Errorf("autenticar token: registrar uso: %w", err)
		}
	}
	return appresult.Ok(dtoresult.TokenAutenticado{
		Actor: appctx.NewActor(s.Sesion.UsuarioID, s.Username, s.Sesion.ID, grants),
	}), nil
}
