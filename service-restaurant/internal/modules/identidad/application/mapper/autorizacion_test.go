package mapper

import (
	"slices"
	"testing"

	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func actorCon(grants ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, grants) }

func TestRequiere(t *testing.T) {
	if ae := Requiere(appctx.Actor{}, "x", policy.NegocioVer); ae == nil || ae.Kind != apperror.KindUnauthorized {
		t.Fatalf("sin actor debe ser 401: %v", ae)
	}
	a := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.NegocioVer})
	if ae := Requiere(a, "x", policy.NegocioAdministrar); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSinPermiso {
		t.Fatalf("sin permiso debe ser 403: %v", ae)
	}
	if ae := Requiere(a, "x", policy.NegocioAdministrar, policy.NegocioVer); ae != nil {
		t.Fatalf("alguno de los permisos basta: %v", ae)
	}
}

func TestSedesVisiblesEsLaUnionOrdenadaSinRepetir(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 5, Permiso: policy.SucursalVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.SucursalAdministrar},
		appctx.Grant{SucursalID: 5, Permiso: policy.SucursalAdministrar},
		appctx.Grant{SucursalID: 9, Permiso: policy.NegocioVer},
	)
	if got := SedesVisibles(a); !slices.Equal(got, []int64{2, 5}) {
		t.Fatalf("sedes visibles = %v, quería [2 5]", got)
	}
}

func TestVisibilidadYAdministracionDeSedeInactiva(t *testing.T) {
	soloVer := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalVer})
	admin := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalAdministrar})

	if !SucursalVisible(soloVer, 1, true) || SucursalVisible(soloVer, 2, true) {
		t.Error("una sede activa solo es visible donde hay VER o ADMINISTRAR")
	}
	if SucursalVisible(soloVer, 2, false) {
		t.Error("una sede inactiva no es visible sin SUCURSAL_ADMINISTRAR")
	}
	if !SucursalVisible(admin, 2, false) || !PuedeAdministrarSucursal(admin, 2, false) {
		t.Error("quien administra sedes ve y gestiona las inactivas (para poder restaurarlas)")
	}
	if PuedeAdministrarSucursal(admin, 2, true) {
		t.Error("una sede activa ajena no se administra")
	}
	if !PuedeAdministrarSucursal(admin, 1, true) {
		t.Error("una sede activa propia sí se administra")
	}
	if PuedeAdministrarSucursal(soloVer, 1, true) {
		t.Error("VER no permite administrar")
	}
}
