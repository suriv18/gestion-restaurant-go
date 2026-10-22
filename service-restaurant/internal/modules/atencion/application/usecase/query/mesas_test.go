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

type mesaReaderFake struct {
	filtro   outbound.FiltroMesas
	llamadas int
	filas    []dtoresult.Mesa
	una      dtoresult.Mesa
}

func (f *mesaReaderFake) ListarMesas(_ context.Context, fl outbound.FiltroMesas) ([]dtoresult.Mesa, error) {
	f.llamadas++
	f.filtro = fl
	return f.filas, nil
}
func (f *mesaReaderFake) ObtenerMesa(_ context.Context, id int64) (dtoresult.Mesa, bool, error) {
	f.llamadas++
	return f.una, f.una.ID == id, nil
}

func TestListarMesasAcotaPorLasSedesDelActor(t *testing.T) {
	f := &mesaReaderFake{filas: []dtoresult.Mesa{{ID: 1}, {ID: 2}, {ID: 3}}}
	a := actorMesas(
		appctx.Grant{SucursalID: 4, Permiso: policy.MesaVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.MesaAdministrar})
	res, err := NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a, Limite: 2})
	if err != nil || !res.IsOk() {
		t.Fatalf("listar: %v %+v", err, res)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{2, 4}) || f.filtro.Limite != 3 {
		t.Fatalf("filtro: %+v", f.filtro)
	}
	if p := res.Value(); len(p.Items) != 2 || p.NextCursor == "" {
		t.Fatalf("debía recortar a 2 y dar cursor: %+v", p)
	}
}

func TestListarMesasRechazos(t *testing.T) {
	f := &mesaReaderFake{}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	sinPermiso := actorMesas(appctx.Grant{SucursalID: 1, Permiso: "NEGOCIO_VER"})
	res, err := NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: sinPermiso})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindForbidden || f.llamadas != 0 {
		t.Fatalf("sin permiso debe ser 403 sin tocar el lector: %+v", res.Err())
	}
	ajena := int64(9)
	res, err = NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a, SucursalID: &ajena})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound || f.llamadas != 0 {
		t.Fatalf("una sede ajena debe ser 404: %+v", res.Err())
	}
	res, err = NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a, Cursor: "%%%"})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido debe ser 400: %+v", res.Err())
	}
	res, err = NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a})
	sinFallo(t, err)
	if !res.IsOk() || res.Value().Items == nil {
		t.Fatalf("una lista vacía debe ser [] y no nula: %+v", res.Value())
	}
}

func TestObtenerMesaOcultaLasAjenas(t *testing.T) {
	f := &mesaReaderFake{una: dtoresult.Mesa{ID: 9, SucursalID: 3}}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	res, err := NewObtenerMesa(f).Execute(context.Background(), qry.ObtenerMesa{Actor: a, ID: 9})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una mesa de sede ajena debe ser 404: %+v", res.Err())
	}
	f.una = dtoresult.Mesa{ID: 5, SucursalID: 1}
	res, err = NewObtenerMesa(f).Execute(context.Background(), qry.ObtenerMesa{Actor: a, ID: 5})
	sinFallo(t, err)
	if !res.IsOk() || res.Value().ID != 5 {
		t.Fatalf("una mesa propia debe leerse: %+v", res.Err())
	}
	res, err = NewObtenerMesa(f).Execute(context.Background(), qry.ObtenerMesa{Actor: a, ID: 77})
	sinFallo(t, err)
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una mesa inexistente debe ser 404: %+v", res.Err())
	}
}
