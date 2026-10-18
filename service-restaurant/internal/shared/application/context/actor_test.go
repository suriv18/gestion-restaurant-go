package context

import (
	"reflect"
	"testing"
)

func TestActorPermissionsPerSede(t *testing.T) {
	a := NewActor(7, "ana", 1, []Grant{
		{1, "PEDIDO_CREAR"}, {1, "CAJA_OPERAR"}, {2, "PEDIDO_CREAR"}, {1, "PEDIDO_CREAR"},
	})
	if !a.Can("PEDIDO_CREAR", 1) || !a.Can("PEDIDO_CREAR", 2) {
		t.Fatal("debe poder crear pedidos en 1 y 2")
	}
	if a.Can("CAJA_OPERAR", 2) {
		t.Fatal("CAJA_OPERAR solo en la sucursal 1")
	}
	if a.Can("PEDIDO_CREAR", 3) {
		t.Fatal("sucursal ajena")
	}
	if !a.CanAnywhere("CAJA_OPERAR") || a.CanAnywhere("USUARIO_ADMINISTRAR") {
		t.Fatal("CanAnywhere incorrecto")
	}
	if got := a.SucursalesCon("PEDIDO_CREAR"); !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Fatalf("SucursalesCon = %v", got)
	}
	if got := a.Permisos(); !reflect.DeepEqual(got, []string{"CAJA_OPERAR", "PEDIDO_CREAR"}) {
		t.Fatalf("Permisos = %v", got)
	}
}

func TestZeroActor(t *testing.T) {
	var a Actor
	if !a.IsZero() || a.Can("X", 1) || a.CanAnywhere("X") || len(a.Sucursales()) != 0 {
		t.Fatal("el actor cero no tiene acceso")
	}
}
