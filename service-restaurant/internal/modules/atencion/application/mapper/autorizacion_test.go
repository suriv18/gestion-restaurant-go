package mapper

import (
	"errors"
	"slices"
	"testing"

	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func actorCon(g ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, g) }

func TestRequiere(t *testing.T) {
	if ae := Requiere(appctx.Actor{}, "x", policy.MesaVer); ae == nil || ae.Kind != apperror.KindUnauthorized || ae.Code != CodigoNoAutenticado {
		t.Fatalf("sin actor debe ser 401: %v", ae)
	}
	a := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	if ae := Requiere(a, "x", policy.MesaAdministrar); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSinPermiso {
		t.Fatalf("sin permiso debe ser 403: %v", ae)
	}
	if ae := Requiere(a, "x", policy.MesaAdministrar, policy.MesaVer); ae != nil {
		t.Fatalf("alguno de los permisos basta: %v", ae)
	}
}

func TestSedesVisiblesYAutorizarSede(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 5, Permiso: policy.MesaVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.MesaAdministrar},
		appctx.Grant{SucursalID: 5, Permiso: policy.MesaAdministrar},
		appctx.Grant{SucursalID: 9, Permiso: "NEGOCIO_VER"},
	)
	if got := SedesVisibles(a); !slices.Equal(got, []int64{2, 5}) {
		t.Fatalf("sedes visibles = %v, quería [2 5]", got)
	}
	if got := SedesVisibles(actorCon()); got == nil || len(got) != 0 {
		t.Fatalf("sin sedes debe devolver un slice vacío no nulo: %#v", got)
	}
	if ae := AutorizarSede(a, 2); ae != nil {
		t.Errorf("administra la sede 2: %v", ae)
	}
	if ae := AutorizarSede(a, 7); ae == nil || ae.Kind != apperror.KindNotFound {
		t.Errorf("una sede ajena responde 404: %v", ae)
	}
	soloVer := actorCon(appctx.Grant{SucursalID: 3, Permiso: policy.MesaVer})
	if ae := AutorizarSede(soloVer, 3); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSedeNoAutorizada {
		t.Errorf("ver sin administrar responde 403 sede_no_autorizada: %v", ae)
	}
}

func TestDeAlmacen(t *testing.T) {
	if DeAlmacen(nil) != nil || DeAlmacen(errors.New("técnico")) != nil {
		t.Fatal("nil y errores técnicos no son rechazos esperados")
	}
	if ae := DeAlmacen(outbound.ErrDuplicado); ae == nil || ae.Kind != apperror.KindConflict || ae.Code != CodigoDuplicado {
		t.Errorf("duplicado: %v", ae)
	}
	if ae := DeAlmacen(outbound.ErrReferenciaInvalida); ae == nil || ae.Kind != apperror.KindValidation {
		t.Errorf("referencia: %v", ae)
	}
	if ae := DeAlmacen(&outbound.ErrReglaDatos{Mensaje: "Cerrar el pedido"}); ae == nil || ae.Kind != apperror.KindBusinessRule ||
		ae.Code != CodigoReglaDatos || ae.Message != "Cerrar el pedido" {
		t.Errorf("regla de datos: %v", ae)
	}
}

func TestCursor(t *testing.T) {
	if id, ae := DecodeCursor(""); id != 0 || ae != nil {
		t.Fatalf("vacío es la primera página: %d %v", id, ae)
	}
	if id, ae := DecodeCursor(EncodeCursor(42)); id != 42 || ae != nil {
		t.Fatalf("ida y vuelta: %d %v", id, ae)
	}
	if _, ae := DecodeCursor("%%%"); ae == nil || ae.Code != CodigoCursorInvalido || ae.Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido: %v", ae)
	}
}
