package query

import (
	"slices"

	"service-restaurant/internal/modules/atencion/application/mapper"
	appctx "service-restaurant/internal/shared/application/context"
)

// sedesPedidas devuelve las sedes que alcanza un listado de mesas: las visibles del actor o, si
// pidió una sucursal concreta, solo esa. ok=false si pidió una sucursal que no ve.
func sedesPedidas(a appctx.Actor, sucursalID *int64) ([]int64, bool) {
	return acotarSedes(mapper.SedesVisibles(a), sucursalID)
}

// sedesDePedidos es como sedesPedidas para los permisos de pedidos.
func sedesDePedidos(a appctx.Actor, sucursalID *int64) ([]int64, bool) {
	return acotarSedes(mapper.SedesConPedidos(a), sucursalID)
}

func acotarSedes(visibles []int64, sucursalID *int64) ([]int64, bool) {
	if sucursalID == nil {
		return visibles, true
	}
	if !slices.Contains(visibles, *sucursalID) {
		return nil, false
	}
	return []int64{*sucursalID}, true
}
