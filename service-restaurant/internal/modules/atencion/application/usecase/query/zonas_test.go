package query

import (
	"context"
	"slices"
	"testing"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type zonaReaderFake struct {
	filtro   outbound.FiltroZonas
	llamadas int
	filas    []dtoresult.Zona
	una      dtoresult.Zona
}

func (f *zonaReaderFake) ListarZonas(_ context.Context, fl outbound.FiltroZonas) ([]dtoresult.Zona, error) {
	f.llamadas++
	f.filtro = fl
	return f.filas, nil
}
func (f *zonaReaderFake) ObtenerZona(_ context.Context, id int64) (dtoresult.Zona, bool, error) {
	f.llamadas++
	return f.una, f.una.ID == id, nil
}

func actorMesas(g ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, g) }

func TestListarZonasAcotaPorLasSedesDelActor(t *testing.T) {
	f := &zonaReaderFake{}
	a := actorMesas(
		appctx.Grant{SucursalID: 4, Permiso: policy.MesaVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.MesaAdministrar})
	res, err := NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, Limite: 10})
	if err != nil || !res.IsOk() {
		t.Fatalf("listar: %v %+v", err, res)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{2, 4}) || f.filtro.Limite != 11 {
		t.Fatalf("filtro: %+v (se pide limite+1)", f.filtro)
	}
	sede := int64(4)
	if _, err := NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, SucursalID: &sede}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{4}) {
		t.Fatalf("un sucursal_id visible restringe a esa sede: %+v", f.filtro)
	}
}

func TestListarZonasRechazos(t *testing.T) {
	f := &zonaReaderFake{filas: []dtoresult.Zona{{ID: 1}, {ID: 2}, {ID: 3}}}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	res, err := NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, Limite: 2})
	sinFallo(t, err)
	if p := res.Value(); len(p.Items) != 2 || p.NextCursor == "" {
		t.Fatalf("debía recortar a 2 y dar cursor: %+v", p)
	}

	f.llamadas = 0
	sinPermiso := actorMesas(appctx.Grant{SucursalID: 1, Permiso: "NEGOCIO_VER"})
	res, err = NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: sinPermiso})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindForbidden || f.llamadas != 0 {
		t.Fatalf("sin permiso debe ser 403 sin tocar el lector: %+v", res.Err())
	}
	ajena := int64(9)
	res, err = NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, SucursalID: &ajena})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound || f.llamadas != 0 {
		t.Fatalf("una sede ajena debe ser 404 sin tocar el lector: %+v", res.Err())
	}
	res, err = NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, Cursor: "%%%"})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido debe ser 400: %+v", res.Err())
	}
}

func TestObtenerZonaOcultaLasAjenas(t *testing.T) {
	f := &zonaReaderFake{una: dtoresult.Zona{ID: 9, SucursalID: 3}}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	res, err := NewObtenerZona(f).Execute(context.Background(), qry.ObtenerZona{Actor: a, ID: 9})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una zona de sede ajena debe ser 404: %+v", res.Err())
	}
	f.una = dtoresult.Zona{ID: 5, SucursalID: 1}
	res, err = NewObtenerZona(f).Execute(context.Background(), qry.ObtenerZona{Actor: a, ID: 5})
	sinFallo(t, err)
	if !res.IsOk() || res.Value().ID != 5 {
		t.Fatalf("una zona propia debe leerse: %+v", res.Err())
	}
	res, err = NewObtenerZona(f).Execute(context.Background(), qry.ObtenerZona{Actor: a, ID: 77})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una zona inexistente debe ser 404: %+v", res.Err())
	}
}

// sinFallo exige que un rechazo esperado no haya viajado por el canal de error técnico.
func sinFallo(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("un rechazo esperado no debe ser un error técnico: %v", err)
	}
}
