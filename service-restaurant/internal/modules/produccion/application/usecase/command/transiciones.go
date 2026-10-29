package command

import (
	"context"
	"fmt"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoTransicion = "no tiene permiso para cambiar el estado de preparación"

// TransicionarLinea mueve una línea de comanda al estado siguiente (RES-RF-012). Solo bloquea la
// línea: no toma ningún otro bloqueo después, así que no puede cruzarse con las escrituras de
// pedido de R1 o R3 (que bloquean pedido y después líneas).
type TransicionarLinea struct{ store outbound.TransicionStore }

func NewTransicionarLinea(s outbound.TransicionStore) *TransicionarLinea {
	return &TransicionarLinea{s}
}

func (uc *TransicionarLinea) Execute(ctx context.Context, c cmd.TransicionarLinea) (appresult.Result[dtoresult.ComandaLinea], error) {
	if !policy.EstadoDestinoValido(c.Estado) {
		return rechazar[dtoresult.ComandaLinea](mapper.Validacion("estado_invalido", "el estado indicado no es válido"))
	}
	permisos := policy.PermisosDe(c.Estado)
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoTransicion, permisos...); ae != nil {
		return rechazar[dtoresult.ComandaLinea](ae)
	}

	ref, found, err := uc.store.BloquearLineaComanda(ctx, c.LineaID)
	if err != nil {
		return tecnico[dtoresult.ComandaLinea]("buscar línea de comanda", err)
	}
	if !found {
		return rechazar[dtoresult.ComandaLinea](mapper.NoEncontrado("línea de comanda"))
	}
	if ae := mapper.Autorizar(c.Actor, ref.SucursalID, mapper.NoEncontrado("línea de comanda"), permisos...); ae != nil {
		return rechazar[dtoresult.ComandaLinea](ae)
	}
	if !policy.Permite(ref.Estado, c.Estado) {
		return rechazar[dtoresult.ComandaLinea](apperror.New(apperror.KindConflict, "transicion_invalida",
			fmt.Sprintf("no se puede pasar de %s a %s", ref.Estado, c.Estado)))
	}

	l, err := uc.store.TransicionarLinea(ctx, c.LineaID, c.Estado)
	if err != nil {
		return deAlmacen[dtoresult.ComandaLinea]("transicionar línea", err)
	}
	return appresult.Ok(l), nil
}
