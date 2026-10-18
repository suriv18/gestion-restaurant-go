package command

import (
	"context"
	"fmt"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// CerrarSesion revoca la sesión del actor autenticado. Es idempotente.
type CerrarSesion struct {
	sesiones outbound.SesionWriter
	clock    outbound.Clock
}

func NewCerrarSesion(sesiones outbound.SesionWriter, clock outbound.Clock) *CerrarSesion {
	return &CerrarSesion{sesiones, clock}
}

func (uc *CerrarSesion) Execute(ctx context.Context, c cmd.CerrarSesion) (appresult.Result[dtoresult.CierreSesion], error) {
	if c.SesionID <= 0 {
		return appresult.Fail[dtoresult.CierreSesion](mapper.NoAutenticado()), nil
	}
	if err := uc.sesiones.Revocar(ctx, c.SesionID, uc.clock.Now()); err != nil {
		return appresult.Result[dtoresult.CierreSesion]{}, fmt.Errorf("cerrar sesión: %w", err)
	}
	return appresult.Ok(dtoresult.CierreSesion{}), nil
}
