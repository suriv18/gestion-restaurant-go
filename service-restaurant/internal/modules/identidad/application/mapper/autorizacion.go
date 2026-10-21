package mapper

import (
	"slices"

	"service-restaurant/internal/modules/identidad/domain/policy"
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

// SedesVisibles devuelve, ordenadas y sin repetir, las sedes activas donde el actor tiene
// SUCURSAL_VER o SUCURSAL_ADMINISTRAR (v_permisos_efectivos solo concede permisos sobre sedes activas).
func SedesVisibles(a appctx.Actor) []int64 {
	ids := append(a.SucursalesCon(policy.SucursalVer), a.SucursalesCon(policy.SucursalAdministrar)...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if ids == nil {
		return []int64{}
	}
	return ids
}

// SucursalVisible indica si el actor puede ver la sede. Las sedes inactivas no generan permisos
// (la vista de permisos las excluye), así que las ve quien administra sedes en alguna sede:
// es lo que permite restaurarlas.
func SucursalVisible(a appctx.Actor, id int64, activa bool) bool {
	if activa {
		return slices.Contains(SedesVisibles(a), id)
	}
	return a.CanAnywhere(policy.SucursalAdministrar)
}

// PuedeAdministrarSucursal aplica la misma regla para editar, dar de baja y restaurar.
func PuedeAdministrarSucursal(a appctx.Actor, id int64, activa bool) bool {
	if activa {
		return a.Can(policy.SucursalAdministrar, id)
	}
	return a.CanAnywhere(policy.SucursalAdministrar)
}
