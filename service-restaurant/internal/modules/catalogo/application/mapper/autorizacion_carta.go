package mapper

import (
	"service-restaurant/internal/modules/catalogo/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// La disponibilidad de productos es por sede (D2, D7): además de Requiere (permiso en alguna
// sede) se decide por sede con CARTA_VER, CARTA_DISPONIBILIDAD y CARTA_ADMINISTRAR.

// PuedeVerSedeCarta indica si el actor tiene algún permiso de carta en la sede ("ver" la sede).
func PuedeVerSedeCarta(a appctx.Actor, sedeID int64) bool {
	return a.Can(policy.CartaVer, sedeID) || a.Can(policy.CartaDisponibilidad, sedeID) || a.Can(policy.CartaAdministrar, sedeID)
}

// SedeCartaVisible devuelve 404 de sucursal si el actor no ve la sede (no se revela si existe).
func SedeCartaVisible(a appctx.Actor, sedeID int64) *apperror.AppError {
	if !PuedeVerSedeCarta(a, sedeID) {
		return NoEncontrada("sucursal")
	}
	return nil
}

// AutorizarDisponibilidad decide si el actor puede escribir productos_sucursal en la sede: sede
// que no ve → 404; solo cambiar `disponible` pide CARTA_DISPONIBILIDAD o CARTA_ADMINISTRAR; cambiar
// precio o estación (cambiaPrecioOEstacion) pide CARTA_ADMINISTRAR. La ve pero no alcanza → 403.
func AutorizarDisponibilidad(a appctx.Actor, sedeID int64, cambiaPrecioOEstacion bool) *apperror.AppError {
	if ae := SedeCartaVisible(a, sedeID); ae != nil {
		return ae
	}
	if cambiaPrecioOEstacion {
		if !a.Can(policy.CartaAdministrar, sedeID) {
			return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada,
				"cambiar el precio o la estación de un producto en una sucursal exige administrar la carta de esa sucursal")
		}
		return nil
	}
	if !a.Can(policy.CartaDisponibilidad, sedeID) && !a.Can(policy.CartaAdministrar, sedeID) {
		return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada,
			"no puede pausar ni habilitar productos en esta sucursal")
	}
	return nil
}
