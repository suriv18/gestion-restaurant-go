package mapper

import (
	"slices"

	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// Requiere exige un actor autenticado con alguno de los permisos en al menos una sede.
func Requiere(a appctx.Actor, mensaje string, permisos ...string) *apperror.AppError {
	if a.IsZero() {
		return NoAutenticado()
	}
	for _, p := range permisos {
		if a.CanAnywhere(p) {
			return nil
		}
	}
	return apperror.New(apperror.KindForbidden, CodigoSinPermiso, mensaje)
}

// SedesVisibles devuelve, ordenadas y sin repetir, las sedes donde el actor tiene MESA_VER o
// MESA_ADMINISTRAR. Nunca es nil, para que un parámetro SQL vacío sea un arreglo vacío.
func SedesVisibles(a appctx.Actor) []int64 {
	ids := append(a.SucursalesCon(policy.MesaVer), a.SucursalesCon(policy.MesaAdministrar)...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if ids == nil {
		return []int64{}
	}
	return ids
}

// PuedeVer indica si el actor tiene algún permiso de mesa en la sede.
func PuedeVer(a appctx.Actor, sedeID int64) bool { return slices.Contains(SedesVisibles(a), sedeID) }

// AutorizarSede decide si el actor puede escribir en la sede: sede que no ve → 404 (no se
// revela su existencia); la ve pero no la administra → 403.
func AutorizarSede(a appctx.Actor, sedeID int64) *apperror.AppError {
	if !PuedeVer(a, sedeID) {
		return NoEncontrado("sucursal")
	}
	if !a.Can(policy.MesaAdministrar, sedeID) {
		return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada, "no administra las mesas de esta sucursal")
	}
	return nil
}
