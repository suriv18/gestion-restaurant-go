// Package command contiene los casos de uso de escritura del módulo identidad.
package command

import (
	"context"
	"fmt"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	"service-restaurant/internal/modules/identidad/domain/vo"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// IniciarSesion autentica a un usuario humano y abre una sesión (RES-RF-001).
//
// Todo rechazo de credenciales es idéntico (mapper.CredencialesInvalidas). Un fallo de
// contraseña incrementa el contador de intentos: quien orquesta la transacción debe
// confirmar ese rechazo (ver infrastructure/config), o el bloqueo nunca se acumularía.
type IniciarSesion struct {
	usuarios outbound.UsuarioStore
	actor    outbound.ActorContext
	sesiones outbound.SesionWriter
	hasher   outbound.PasswordHasher
	tokens   outbound.TokenIssuer
	clock    outbound.Clock
	bloqueo  policy.BloqueoPorIntentos
	sesion   policy.PoliticaSesion
}

func NewIniciarSesion(
	usuarios outbound.UsuarioStore, actor outbound.ActorContext, sesiones outbound.SesionWriter,
	hasher outbound.PasswordHasher, tokens outbound.TokenIssuer, clock outbound.Clock,
	bloqueo policy.BloqueoPorIntentos, sesion policy.PoliticaSesion,
) *IniciarSesion {
	return &IniciarSesion{usuarios, actor, sesiones, hasher, tokens, clock, bloqueo, sesion}
}

func (uc *IniciarSesion) Execute(ctx context.Context, c cmd.IniciarSesion) (appresult.Result[dtoresult.SesionIniciada], error) {
	rechazo := appresult.Fail[dtoresult.SesionIniciada](mapper.CredencialesInvalidas())

	username, err := vo.NewUsername(c.Username)
	if err != nil || !policy.ClaveAdmisibleParaLogin(c.Password) {
		uc.hasher.Dummy(c.Password)
		return rechazo, nil
	}

	u, found, err := uc.usuarios.BuscarPorUsernameParaActualizar(ctx, username.Key())
	if err != nil {
		return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: buscar usuario: %w", err)
	}
	ahora := uc.clock.Now()
	if !found || !u.PuedeAutenticar(ahora) {
		// Mismo costo que una verificación real: el tiempo no revela el motivo.
		uc.hasher.Dummy(c.Password)
		return rechazo, nil
	}

	ok, err := uc.hasher.Verify(c.Password, u.PasswordHash)
	if err != nil {
		return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: verificar contraseña: %w", err)
	}
	if !ok {
		sistema, err := uc.actor.UsuarioSistemaID(ctx)
		if err != nil {
			return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: actor técnico: %w", err)
		}
		// Aún no hay usuario autenticado: el conteo de fallos lo registra el actor técnico.
		if err := uc.actor.Fijar(ctx, sistema); err != nil {
			return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: fijar actor: %w", err)
		}
		uc.bloqueo.RegistrarFallo(&u, ahora)
		if err := uc.usuarios.GuardarIntentos(ctx, u); err != nil {
			return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: guardar intentos: %w", err)
		}
		return rechazo, nil
	}

	if err := uc.actor.Fijar(ctx, u.ID); err != nil {
		return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: fijar actor: %w", err)
	}
	u.RegistrarAcceso()
	if err := uc.usuarios.RegistrarAcceso(ctx, u, ahora); err != nil {
		return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: registrar acceso: %w", err)
	}

	sesion, err := uc.sesiones.Crear(ctx, uc.sesion.Nueva(u.ID, ahora))
	if err != nil {
		return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: crear sesión: %w", err)
	}
	token, err := uc.tokens.Emitir(outbound.TokenClaims{SesionID: sesion.ID, UsuarioID: u.ID}, ahora, sesion.ExpiraAbsolutaEn)
	if err != nil {
		return appresult.Result[dtoresult.SesionIniciada]{}, fmt.Errorf("iniciar sesión: emitir token: %w", err)
	}
	return appresult.Ok(dtoresult.SesionIniciada{
		AccessToken: token,
		ExpiresIn:   uc.sesion.ExpiraEnSegundos(sesion, ahora),
	}), nil
}
