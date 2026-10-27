package mapper

import (
	"slices"
	"testing"

	"service-restaurant/internal/modules/catalogo/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func TestSedesVisiblesYAutorizacionDeEstaciones(t *testing.T) {
	a := appctx.NewActor(7, "ana", 1, []appctx.Grant{
		{SucursalID: 5, Permiso: policy.EstacionVer},
		{SucursalID: 2, Permiso: policy.EstacionAdministrar},
		{SucursalID: 5, Permiso: policy.EstacionAdministrar},
		{SucursalID: 9, Permiso: policy.CartaVer},
	})
	if got := SedesVisibles(a); !slices.Equal(got, []int64{2, 5}) {
		t.Fatalf("sedes visibles = %v, quería [2 5]", got)
	}
	if got := SedesVisibles(appctx.NewActor(7, "ana", 1, nil)); got == nil || len(got) != 0 {
		t.Fatalf("sin sedes debe ser un slice vacío no nulo: %#v", got)
	}
	if ae := AutorizarSede(a, 2); ae != nil {
		t.Errorf("administra la sede 2: %v", ae)
	}
	if ae := AutorizarSede(a, 7); ae == nil || ae.Kind != apperror.KindNotFound {
		t.Errorf("una sede ajena responde 404: %v", ae)
	}
	if ae := AutorizarEstacion(a, 9); ae == nil || ae.Kind != apperror.KindNotFound {
		t.Errorf("un permiso de otra clase no da visibilidad: %v", ae)
	}
	soloVer := appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: 3, Permiso: policy.EstacionVer}})
	for _, f := range []func() *apperror.AppError{
		func() *apperror.AppError { return AutorizarSede(soloVer, 3) },
		func() *apperror.AppError { return AutorizarEstacion(soloVer, 3) },
	} {
		if ae := f(); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSedeNoAutorizada {
			t.Errorf("ver sin administrar responde 403 sede_no_autorizada: %v", ae)
		}
	}
	if ae := AutorizarEstacion(soloVer, 4); ae == nil || ae.Kind != apperror.KindNotFound {
		t.Errorf("estación de sede ajena responde 404: %v", ae)
	}
}
