package mapper

import (
	"errors"
	"fmt"
	"testing"

	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func TestCursorIdaYVuelta(t *testing.T) {
	for _, id := range []int64{0, 1, 42, 9223372036854775807} {
		got, ae := DecodeCursor(EncodeCursor(id))
		if ae != nil || got != id {
			t.Errorf("id=%d got=%d ae=%v", id, got, ae)
		}
	}
	if id, ae := DecodeCursor(""); ae != nil || id != 0 {
		t.Fatal("cursor vacío = primera página")
	}
	for _, malo := range []string{"abc", "!!!", EncodeCursor(0) + "x", "LTE"} { // "LTE" = "-1"
		if _, ae := DecodeCursor(malo); ae == nil || ae.Kind != apperror.KindBadRequest {
			t.Errorf("%q debía rechazarse", malo)
		}
	}
}

func TestDeAlmacen(t *testing.T) {
	casos := []struct {
		err  error
		kind apperror.Kind
		code string
	}{
		{fmt.Errorf("x: %w", outbound.ErrDuplicado), apperror.KindConflict, CodigoDuplicado},
		{fmt.Errorf("x: %w", outbound.ErrReferenciaInvalida), apperror.KindValidation, CodigoReferenciaInvalida},
		{&outbound.ErrReglaDatos{Mensaje: "regla"}, apperror.KindBusinessRule, CodigoReglaDatos},
	}
	for _, c := range casos {
		ae := DeAlmacen(c.err)
		if ae == nil || ae.Kind != c.kind || ae.Code != c.code {
			t.Errorf("%v -> %+v", c.err, ae)
		}
	}
	if DeAlmacen(nil) != nil || DeAlmacen(errors.New("fallo técnico")) != nil {
		t.Fatal("un error técnico no es un rechazo")
	}
}

func TestRequiereAdministrar(t *testing.T) {
	if ae := RequiereAdministrar(context.Actor{}); ae == nil || ae.Kind != apperror.KindUnauthorized {
		t.Fatalf("sin actor: %+v", ae)
	}
	sinPermiso := context.NewActor(1, "a", 1, []context.Grant{{SucursalID: 1, Permiso: "PEDIDO_CREAR"}})
	if ae := RequiereAdministrar(sinPermiso); ae == nil || ae.Kind != apperror.KindForbidden {
		t.Fatalf("sin permiso: %+v", ae)
	}
	admin := context.NewActor(1, "a", 1, []context.Grant{{SucursalID: 2, Permiso: "USUARIO_ADMINISTRAR"}})
	if ae := RequiereAdministrar(admin); ae != nil {
		t.Fatalf("administrador: %+v", ae)
	}
}
