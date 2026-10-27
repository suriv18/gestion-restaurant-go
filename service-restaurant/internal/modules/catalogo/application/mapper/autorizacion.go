package mapper

import (
	"slices"

	"service-restaurant/internal/modules/catalogo/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// CodigoSedeNoAutorizada: el actor ve la sede pero no administra sus estaciones.
const CodigoSedeNoAutorizada = "sede_no_autorizada"

// Las estaciones son por sede (D2): además de Requiere (permiso en alguna sede) se decide por
// sede con los permisos ESTACION_VER / ESTACION_ADMINISTRAR.

// SedesVisibles devuelve, ordenadas y sin repetir, las sedes donde el actor tiene ESTACION_VER o
// ESTACION_ADMINISTRAR. Nunca es nil, para que un parámetro SQL vacío sea un arreglo vacío.
func SedesVisibles(a appctx.Actor) []int64 {
	ids := append(a.SucursalesCon(policy.EstacionVer), a.SucursalesCon(policy.EstacionAdministrar)...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if ids == nil {
		return []int64{}
	}
	return ids
}

// PuedeVerSede indica si el actor tiene algún permiso de estación en la sede.
func PuedeVerSede(a appctx.Actor, sedeID int64) bool {
	return a.Can(policy.EstacionVer, sedeID) || a.Can(policy.EstacionAdministrar, sedeID)
}

// AutorizarSede decide si el actor puede escribir estaciones en la sede: sede que no ve → 404
// (no se revela su existencia); la ve pero no la administra → 403.
func AutorizarSede(a appctx.Actor, sedeID int64) *apperror.AppError {
	if !PuedeVerSede(a, sedeID) {
		return NoEncontrada("sucursal")
	}
	return exigirAdministrar(a, sedeID)
}

// AutorizarEstacion es AutorizarSede para una estación ya leída: si el actor no ve la sede, la
// estación "no existe" (404 de estación, sin pistas); si la ve sin administrarla, 403.
func AutorizarEstacion(a appctx.Actor, sedeID int64) *apperror.AppError {
	if !PuedeVerSede(a, sedeID) {
		return NoEncontrada("estación")
	}
	return exigirAdministrar(a, sedeID)
}

func exigirAdministrar(a appctx.Actor, sedeID int64) *apperror.AppError {
	if !a.Can(policy.EstacionAdministrar, sedeID) {
		return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada, "no administra las estaciones de esta sucursal")
	}
	return nil
}
