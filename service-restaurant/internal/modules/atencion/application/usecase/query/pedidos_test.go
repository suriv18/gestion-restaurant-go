package query

import (
	"context"
	"slices"
	"testing"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type pedidoReaderFake struct {
	items    []dtoresult.Pedido
	detalle  dtoresult.PedidoDetalle
	existe   bool
	filtro   outbound.FiltroPedidos
	llamadas int
}

func (f *pedidoReaderFake) ListarPedidos(_ context.Context, fl outbound.FiltroPedidos) ([]dtoresult.Pedido, error) {
	f.llamadas++
	f.filtro = fl
	return f.items, nil
}
func (f *pedidoReaderFake) ObtenerPedido(context.Context, int64) (dtoresult.PedidoDetalle, bool, error) {
	return f.detalle, f.existe, nil
}

func lectorDePedidos(grants ...appctx.Grant) appctx.Actor {
	return appctx.NewActor(7, "ana", 1, grants)
}

func TestListarPedidos(t *testing.T) {
	actor := lectorDePedidos(
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoVer},
		appctx.Grant{SucursalID: 1, Permiso: policy.PedidoCrear},
	)

	t.Run("sin permiso", func(t *testing.T) {
		res, err := NewListarPedidos(&pedidoReaderFake{}).Execute(context.Background(),
			qry.ListarPedidos{Actor: lectorDePedidos(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer}), Limite: 10})
		if err != nil || res.Err() == nil || res.Err().Kind != apperror.KindForbidden || res.Err().Code != mapper.CodigoSinPermiso {
			t.Fatalf("debía ser 403: %v %+v", err, res.Err())
		}
	})

	t.Run("sede que no ve", func(t *testing.T) {
		f := &pedidoReaderFake{}
		res, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, SucursalID: idPtr(9), Limite: 10})
		if err != nil || res.Err() == nil || res.Err().Kind != apperror.KindNotFound || f.llamadas != 0 {
			t.Fatalf("debía ser 404 sin consultar: %v %+v", err, res.Err())
		}
	})

	t.Run("filtra por las sedes visibles y pasa los filtros", func(t *testing.T) {
		f := &pedidoReaderFake{}
		_, err := NewListarPedidos(f).Execute(context.Background(),
			qry.ListarPedidos{Actor: actor, Estado: "REGISTRADO", Tipo: "SALON", MesaID: idPtr(4), Limite: 10})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(f.filtro.SedeIDs, []int64{1, 2}) || f.filtro.Estado != "REGISTRADO" || f.filtro.Tipo != "SALON" ||
			f.filtro.MesaID == nil || *f.filtro.MesaID != 4 || f.filtro.Limite != 11 {
			t.Fatalf("filtro: %+v", f.filtro)
		}
	})

	t.Run("una sede concreta", func(t *testing.T) {
		f := &pedidoReaderFake{}
		_, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, SucursalID: idPtr(2), Limite: 10})
		if err != nil || !slices.Equal(f.filtro.SedeIDs, []int64{2}) {
			t.Fatalf("debía acotar a la sede 2: %v %+v", err, f.filtro)
		}
	})

	t.Run("pagina con cursor", func(t *testing.T) {
		f := &pedidoReaderFake{items: []dtoresult.Pedido{{ID: 1}, {ID: 2}, {ID: 3}}}
		res, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Limite: 2})
		if err != nil || !res.IsOk() {
			t.Fatalf("%v %+v", err, res.Err())
		}
		p := res.Value()
		if len(p.Items) != 2 || p.NextCursor != mapper.EncodeCursor(2) {
			t.Fatalf("página: %+v", p)
		}
		desde, _ := mapper.DecodeCursor(p.NextCursor)
		if _, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Limite: 2, Cursor: p.NextCursor}); err != nil || f.filtro.DespuesDeID != desde {
			t.Fatalf("el cursor debe pasar al filtro: %v %+v", err, f.filtro)
		}
	})

	t.Run("cursor inválido", func(t *testing.T) {
		res, err := NewListarPedidos(&pedidoReaderFake{}).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Cursor: "!!!", Limite: 10})
		if err != nil || res.Err() == nil || res.Err().Code != mapper.CodigoCursorInvalido {
			t.Fatalf("debía rechazar el cursor: %v %+v", err, res.Err())
		}
	})

	t.Run("sin resultados devuelve lista vacía", func(t *testing.T) {
		res, _ := NewListarPedidos(&pedidoReaderFake{}).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Limite: 10})
		if res.Value().Items == nil || len(res.Value().Items) != 0 || res.Value().NextCursor != "" {
			t.Fatalf("página vacía: %+v", res.Value())
		}
	})
}

func TestObtenerPedido(t *testing.T) {
	actor := lectorDePedidos(appctx.Grant{SucursalID: 1, Permiso: policy.PedidoVer})
	f := &pedidoReaderFake{existe: true, detalle: dtoresult.PedidoDetalle{Pedido: dtoresult.Pedido{ID: 5, SucursalID: 1}}}

	if res, err := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: actor, ID: 5}); err != nil || !res.IsOk() || res.Value().ID != 5 {
		t.Fatalf("debía devolver el pedido: %v %+v", err, res.Err())
	}
	// Pedido de una sede que el actor no ve: 404, igual que uno inexistente.
	f.detalle.SucursalID = 2
	if res, err := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: actor, ID: 5}); err != nil || res.Err() == nil ||
		res.Err().Kind != apperror.KindNotFound || res.Err().Message != "pedido no encontrado" {
		t.Fatalf("sede ajena debía ser 404: %v %+v", err, res.Err())
	}
	f.existe = false
	if res, _ := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: actor, ID: 5}); res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("inexistente debía ser 404: %+v", res.Err())
	}
	// Sin permiso de pedidos: 403.
	otro := lectorDePedidos(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	if res, _ := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: otro, ID: 5}); res.Err() == nil || res.Err().Kind != apperror.KindForbidden {
		t.Fatalf("sin permiso debía ser 403: %+v", res.Err())
	}
}

func idPtr(n int64) *int64 { return &n }
