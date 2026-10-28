package mapper

import (
	"slices"
	"testing"

	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func TestSedesConPedidos(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 5, Permiso: policy.PedidoVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoCrear},
		appctx.Grant{SucursalID: 5, Permiso: policy.PedidoEditar},
		appctx.Grant{SucursalID: 9, Permiso: policy.MesaVer},
	)
	if got := SedesConPedidos(a); !slices.Equal(got, []int64{2, 5}) {
		t.Fatalf("sedes con pedidos = %v, quería [2 5]", got)
	}
	if got := SedesConPedidos(actorCon()); got == nil || len(got) != 0 {
		t.Fatalf("sin sedes debe devolver un slice vacío no nulo: %#v", got)
	}
}

func TestAutorizarPedidos(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 1, Permiso: policy.PedidoVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoCrear},
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoEditar},
	)
	ante404 := NoEncontradoM("pedido")

	// Sede que no ve: 404 con el rechazo que se le pasó.
	if ae := AutorizarPedidos(a, 9, policy.PedidoEditar, ante404); ae != ante404 {
		t.Fatalf("sede ajena debe devolver el 404 recibido: %v", ae)
	}
	// La ve pero sin el permiso del comando: 403.
	if ae := AutorizarPedidos(a, 1, policy.PedidoEditar, ante404); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSedeNoAutorizada {
		t.Fatalf("solo lectura debe ser 403 sede_no_autorizada: %v", ae)
	}
	// Con el permiso: autorizado. CREAR y EDITAR implican ver.
	if ae := AutorizarPedidos(a, 2, policy.PedidoEditar, ante404); ae != nil {
		t.Fatalf("debía autorizar: %v", ae)
	}
	if !PuedeVerPedidos(a, 2) || !PuedeVerPedidos(a, 1) || PuedeVerPedidos(a, 9) {
		t.Fatal("PuedeVerPedidos: debe ver la 1 y la 2, no la 9")
	}
}

func TestNoEncontradoM(t *testing.T) {
	ae := NoEncontradoM("pedido")
	if ae.Kind != apperror.KindNotFound || ae.Code != CodigoNoEncontrado || ae.Message != "pedido no encontrado" {
		t.Fatalf("NoEncontradoM = %+v", ae)
	}
}
