package query

import (
	"context"
	"testing"

	qry "service-restaurant/internal/modules/produccion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type lectorFake struct {
	comandas []dtoresult.Comanda
	filtro   outbound.FiltroComandas
	llamadas int
}

var _ outbound.ComandaReader = (*lectorFake)(nil)

func (f *lectorFake) ListarComandas(_ context.Context, fl outbound.FiltroComandas) ([]dtoresult.Comanda, error) {
	f.llamadas++
	f.filtro = fl
	return f.comandas, nil
}

func cocinero(sedes ...int64) appctx.Actor {
	g := make([]appctx.Grant, len(sedes))
	for i, s := range sedes {
		g[i] = appctx.Grant{SucursalID: s, Permiso: policy.CocinaOperar}
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

func TestListarComandasUsaSoloLasSedesDeCocina(t *testing.T) {
	f := &lectorFake{}
	res, err := NewListarComandas(f).Execute(context.Background(), qry.ListarComandas{Actor: cocinero(3, 1), Limite: 10})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía listar: %v %v", err, res.Err())
	}
	if got := f.filtro.SedeIDs; len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("sedes del filtro: %v", got)
	}
	if f.filtro.Limite != 11 || res.Value().Items == nil {
		t.Fatalf("pide limite+1 y nunca devuelve items nil: %+v %v", f.filtro, res.Value())
	}
}

func TestListarComandasPaginaConCursor(t *testing.T) {
	f := &lectorFake{comandas: []dtoresult.Comanda{{ID: 4}, {ID: 7}, {ID: 9}}}
	res, _ := NewListarComandas(f).Execute(context.Background(), qry.ListarComandas{Actor: cocinero(1), Limite: 2})
	p := res.Value()
	if len(p.Items) != 2 || p.Items[1].ID != 7 || p.NextCursor != mapper.EncodeCursor(7) {
		t.Fatalf("página: %+v", p)
	}
	res, _ = NewListarComandas(f).Execute(context.Background(), qry.ListarComandas{Actor: cocinero(1), Limite: 2, Cursor: p.NextCursor})
	if f.filtro.DespuesDeID != 7 {
		t.Fatalf("el cursor debe pasar al filtro: %d", f.filtro.DespuesDeID)
	}
}

func TestListarComandasRechaza(t *testing.T) {
	sede2 := int64(2)
	casos := []struct {
		nombre string
		q      qry.ListarComandas
		kind   apperror.Kind
		code   string
	}{
		{"sin actor", qry.ListarComandas{}, apperror.KindUnauthorized, "no_autenticado"},
		{"sin permiso de cocina", qry.ListarComandas{Actor: appctx.NewActor(7, "ana", 1,
			[]appctx.Grant{{SucursalID: 1, Permiso: policy.PedidoVer}})}, apperror.KindForbidden, "sin_permiso"},
		{"sucursal que no opera", qry.ListarComandas{Actor: cocinero(1), SucursalID: &sede2}, apperror.KindNotFound, "no_encontrado"},
		{"cursor inválido", qry.ListarComandas{Actor: cocinero(1), Cursor: "###"}, apperror.KindBadRequest, "cursor_invalido"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := &lectorFake{}
			res, err := NewListarComandas(f).Execute(context.Background(), tc.q)
			if err != nil {
				t.Fatal(err)
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.llamadas != 0 {
				t.Fatal("un rechazo no debe consultar la base")
			}
		})
	}
}
