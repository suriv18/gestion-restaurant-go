package query

import (
	"context"
	"slices"
	"testing"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type sucursalReaderFake struct {
	filtro   outbound.FiltroSucursales
	llamadas int
	filas    []dtoresult.Sucursal
	una      dtoresult.Sucursal
}

func (f *sucursalReaderFake) ListarSucursales(_ context.Context, fl outbound.FiltroSucursales) ([]dtoresult.Sucursal, error) {
	f.llamadas++
	f.filtro = fl
	return f.filas, nil
}
func (f *sucursalReaderFake) ObtenerSucursal(_ context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	f.llamadas++
	return f.una, f.una.ID == id, nil
}

func actorSedes(g ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, g) }

func TestListarSucursalesAcotaPorLoQueElActorVe(t *testing.T) {
	f := &sucursalReaderFake{}
	a := actorSedes(
		appctx.Grant{SucursalID: 4, Permiso: policy.SucursalVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.SucursalAdministrar})
	res, err := NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: a, Limite: 10})
	if err != nil || !res.IsOk() {
		t.Fatalf("listar: %v %+v", err, res)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{2, 4}) || !f.filtro.IncluirInactivas || f.filtro.Limite != 11 {
		t.Fatalf("filtro: %+v (quien administra sedes ve también las inactivas; se pide limite+1)", f.filtro)
	}
	soloVer := actorSedes(appctx.Grant{SucursalID: 4, Permiso: policy.SucursalVer})
	if _, err := NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: soloVer}); err != nil {
		t.Fatal(err)
	}
	if f.filtro.IncluirInactivas {
		t.Error("quien solo tiene VER no debe ver sedes inactivas")
	}
}

func TestListarSucursalesPaginaYRechazos(t *testing.T) {
	f := &sucursalReaderFake{filas: []dtoresult.Sucursal{{ID: 1}, {ID: 2}, {ID: 3}}}
	a := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalVer})
	res, _ := NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: a, Limite: 2})
	p := res.Value()
	if len(p.Items) != 2 || p.NextCursor == "" {
		t.Fatalf("debía recortar a 2 y dar cursor: %+v", p)
	}

	sinPermiso := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.NegocioVer})
	f.llamadas = 0
	res, _ = NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: sinPermiso})
	if res.Err() == nil || res.Err().Kind != apperror.KindForbidden || f.llamadas != 0 {
		t.Fatalf("sin permiso debe ser 403 sin tocar el lector: %+v", res.Err())
	}
	res, _ = NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: a, Cursor: "%%%"})
	if res.Err() == nil || res.Err().Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido debe ser 400: %+v", res.Err())
	}
}

func TestObtenerSucursalOcultaLasAjenas(t *testing.T) {
	f := &sucursalReaderFake{una: dtoresult.Sucursal{ID: 9, EsActivo: true}}
	a := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalVer})
	res, _ := NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: a, ID: 9})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una sede activa ajena debe ser 404: %+v", res.Err())
	}
	f.una = dtoresult.Sucursal{ID: 1, EsActivo: true}
	res, _ = NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: a, ID: 1})
	if !res.IsOk() || res.Value().ID != 1 {
		t.Fatalf("una sede propia debe leerse: %+v", res.Err())
	}
	// Inactiva: la ve quien administra sedes; quien solo tiene VER no.
	f.una = dtoresult.Sucursal{ID: 5, EsActivo: false}
	res, _ = NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: a, ID: 5})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una sede inactiva es invisible para VER: %+v", res.Err())
	}
	admin := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalAdministrar})
	res, _ = NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: admin, ID: 5})
	if !res.IsOk() {
		t.Fatalf("quien administra sedes ve las inactivas: %+v", res.Err())
	}
}
