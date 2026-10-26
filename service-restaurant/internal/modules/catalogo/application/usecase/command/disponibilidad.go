package command

import (
	"context"
	"errors"
	"strings"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	CodigoEstacionDeOtraSede = "estacion_de_otra_sede"

	mensajeSinPermisoDisponibilidad = "no tiene permiso para cambiar la disponibilidad de productos"
	mensajeEstacionDeOtraSede       = "la estación indicada pertenece a otra sucursal"
	mensajeEstacionNoValida         = "la estación indicada no existe o está dada de baja"
)

// deDisponibilidad traduce los errores de almacenamiento de productos_sucursal: la estación de
// otra sede (FK compuesta), la estación o la sede inexistentes o inactivas (trigger de
// referencias activas, con mensaje técnico) y el resto de reglas de datos.
func deDisponibilidad(err error) (appresult.Result[dtoresult.ProductoSucursal], error) {
	var regla *outbound.ErrReglaDatos
	switch {
	case errors.Is(err, outbound.ErrEstacionOtraSede):
		return rechazar[dtoresult.ProductoSucursal](mapper.Validacion(CodigoEstacionDeOtraSede, mensajeEstacionDeOtraSede))
	case errors.As(err, &regla) && strings.HasPrefix(regla.Mensaje, "Referencia estacion_id="):
		return rechazar[dtoresult.ProductoSucursal](mapper.Validacion(mapper.CodigoReferenciaInvalida, mensajeEstacionNoValida))
	case errors.As(err, &regla) && strings.Contains(regla.Mensaje, "maestro sucursales"):
		return rechazar[dtoresult.ProductoSucursal](mapper.Validacion(mapper.CodigoReferenciaInvalida, mensajeSedeNoValida))
	}
	if ae := mapper.DeAlmacen(err); ae != nil {
		return rechazar[dtoresult.ProductoSucursal](ae)
	}
	return tecnico[dtoresult.ProductoSucursal]("guardar disponibilidad", err)
}

// CambiarDisponibilidad habilita o pausa un producto en una sede (upsert por producto y sede).
// Es idempotente por naturaleza: repetirlo deja el mismo estado. No escribe nada en productos:
// pausar un plato no toca el maestro ni la historia.
type CambiarDisponibilidad struct{ store outbound.DisponibilidadStore }

func NewCambiarDisponibilidad(s outbound.DisponibilidadStore) *CambiarDisponibilidad {
	return &CambiarDisponibilidad{s}
}

func (uc *CambiarDisponibilidad) Execute(ctx context.Context, c cmd.CambiarDisponibilidad) (appresult.Result[dtoresult.ProductoSucursal], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoDisponibilidad, policy.CartaDisponibilidad, policy.CartaAdministrar); ae != nil {
		return rechazar[dtoresult.ProductoSucursal](ae)
	}
	if c.SucursalID <= 0 {
		return rechazar[dtoresult.ProductoSucursal](mapper.Validacion("sucursal_requerida", "sucursal_id es obligatorio"))
	}
	// Primero la sede: si el actor no la ve, no se revela si existe; luego lo que puede cambiar.
	if ae := mapper.AutorizarDisponibilidad(c.Actor, c.SucursalID, c.Precio.Presente || c.Estacion.Presente); ae != nil {
		return rechazar[dtoresult.ProductoSucursal](ae)
	}
	if c.Disponible == nil {
		return rechazar[dtoresult.ProductoSucursal](mapper.Validacion("disponible_requerido", "disponible es obligatorio (true o false)"))
	}
	cambios := outbound.CambiosDisponibilidad{Disponible: *c.Disponible}
	if c.Precio.Presente {
		cambios.Precio.Presente = true
		if c.Precio.Valor != nil {
			p, ae := validarPrecio(*c.Precio.Valor)
			if ae != nil {
				return rechazar[dtoresult.ProductoSucursal](ae)
			}
			cambios.Precio.Valor = &p
		}
	}
	if c.Estacion.Presente {
		if c.Estacion.Valor != nil && *c.Estacion.Valor <= 0 {
			return rechazar[dtoresult.ProductoSucursal](mapper.Validacion("estacion_invalida", "estacion_id no es válido"))
		}
		cambios.Estacion = outbound.CampoID{Presente: true, Valor: c.Estacion.Valor}
	}

	p, found, err := uc.store.BloquearProducto(ctx, c.ProductoID)
	if err != nil {
		return tecnico[dtoresult.ProductoSucursal]("buscar producto", err)
	}
	if !found {
		return rechazar[dtoresult.ProductoSucursal](mapper.NoEncontrado("producto"))
	}
	if !p.EsActivo {
		return rechazar[dtoresult.ProductoSucursal](apperror.New(apperror.KindConflict, CodigoProductoInactivo,
			"el producto está dado de baja: restáurelo para cambiar su disponibilidad"))
	}
	ps, err := uc.store.Guardar(ctx, c.ProductoID, c.SucursalID, cambios)
	if err != nil {
		return deDisponibilidad(err)
	}
	return appresult.Ok(ps), nil
}
