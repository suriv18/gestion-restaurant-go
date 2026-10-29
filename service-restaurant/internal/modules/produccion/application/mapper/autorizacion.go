package mapper

import (
	"service-restaurant/internal/modules/produccion/domain/policy"
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

// Visible indica si el actor tiene algún permiso de pedidos o de cocina en la sede: si no, no se
// le revela que el recurso existe.
func Visible(a appctx.Actor, sedeID int64) bool {
	return a.Can(policy.PedidoVer, sedeID) || a.Can(policy.PedidoCrear, sedeID) ||
		a.Can(policy.PedidoEditar, sedeID) || a.Can(policy.CocinaOperar, sedeID)
}

// Autorizar decide si el actor puede operar un recurso de la sede con alguno de los permisos: si no
// ve la sede responde ante404; si la ve pero no tiene ninguno de los permisos, 403 sede_no_autorizada.
func Autorizar(a appctx.Actor, sedeID int64, ante404 *apperror.AppError, permisos ...string) *apperror.AppError {
	if !Visible(a, sedeID) {
		return ante404
	}
	for _, p := range permisos {
		if a.Can(p, sedeID) {
			return nil
		}
	}
	return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada, "no tiene permiso para esta operación de cocina en esta sucursal")
}

// SedesCocina devuelve, ordenadas, las sedes donde el actor opera la cocina. Nunca es nil, para que
// un parámetro SQL vacío sea un arreglo vacío.
func SedesCocina(a appctx.Actor) []int64 {
	ids := a.SucursalesCon(policy.CocinaOperar)
	if ids == nil {
		return []int64{}
	}
	return ids
}
