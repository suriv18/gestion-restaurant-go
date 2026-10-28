package mapper

import (
	"slices"

	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// SedesConPedidos devuelve, ordenadas y sin repetir, las sedes donde el actor tiene algún permiso
// de pedidos (ver, crear o editar). Nunca es nil.
func SedesConPedidos(a appctx.Actor) []int64 {
	ids := append(a.SucursalesCon(policy.PedidoVer), a.SucursalesCon(policy.PedidoCrear)...)
	ids = append(ids, a.SucursalesCon(policy.PedidoEditar)...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if ids == nil {
		return []int64{}
	}
	return ids
}

// PuedeVerPedidos indica si el actor tiene algún permiso de pedidos en la sede.
func PuedeVerPedidos(a appctx.Actor, sedeID int64) bool {
	return a.Can(policy.PedidoVer, sedeID) || a.Can(policy.PedidoCrear, sedeID) || a.Can(policy.PedidoEditar, sedeID)
}

// AutorizarPedidos decide si el actor puede ejecutar una operación de pedidos con el permiso en la
// sede: si no ve pedidos de esa sede responde ante404 (no se revela su existencia); si los ve pero
// no tiene el permiso, 403. ante404 lo elige quien llama para nombrar el recurso ("sucursal" en
// la apertura, "pedido" en las operaciones por id).
func AutorizarPedidos(a appctx.Actor, sedeID int64, permiso string, ante404 *apperror.AppError) *apperror.AppError {
	if !PuedeVerPedidos(a, sedeID) {
		return ante404
	}
	if !a.Can(permiso, sedeID) {
		return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada, "no tiene permiso para esta operación de pedidos en esta sucursal")
	}
	return nil
}
