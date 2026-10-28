package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoPedidos = "no tiene permiso para consultar pedidos"

// ListarPedidos devuelve los pedidos de las sedes visibles. Pide limite+1 filas para saber si hay
// más páginas.
type ListarPedidos struct{ pedidos outbound.PedidoReader }

func NewListarPedidos(p outbound.PedidoReader) *ListarPedidos { return &ListarPedidos{p} }

func (uc *ListarPedidos) Execute(ctx context.Context, q qry.ListarPedidos) (appresult.Result[dtoresult.PaginaPedidos], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoPedidos, policy.PedidoVer, policy.PedidoCrear, policy.PedidoEditar); ae != nil {
		return appresult.Fail[dtoresult.PaginaPedidos](ae), nil
	}
	sedes, ok := sedesDePedidos(q.Actor, q.SucursalID)
	if !ok {
		return appresult.Fail[dtoresult.PaginaPedidos](mapper.NoEncontrado("sucursal")), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaPedidos](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.pedidos.ListarPedidos(ctx, outbound.FiltroPedidos{
		SedeIDs: sedes, Estado: q.Estado, Tipo: q.Tipo, MesaID: q.MesaID, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaPedidos]{}, fmt.Errorf("listar pedidos: %w", err)
	}
	pagina := dtoresult.PaginaPedidos{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Pedido{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerPedido lee un pedido con sus cuentas y líneas; el de una sede que el actor no ve
// responde 404, igual que uno inexistente.
type ObtenerPedido struct{ pedidos outbound.PedidoReader }

func NewObtenerPedido(p outbound.PedidoReader) *ObtenerPedido { return &ObtenerPedido{p} }

func (uc *ObtenerPedido) Execute(ctx context.Context, q qry.ObtenerPedido) (appresult.Result[dtoresult.PedidoDetalle], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoPedidos, policy.PedidoVer, policy.PedidoCrear, policy.PedidoEditar); ae != nil {
		return appresult.Fail[dtoresult.PedidoDetalle](ae), nil
	}
	p, found, err := uc.pedidos.ObtenerPedido(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.PedidoDetalle]{}, fmt.Errorf("obtener pedido: %w", err)
	}
	if !found || !mapper.PuedeVerPedidos(q.Actor, p.SucursalID) {
		return appresult.Fail[dtoresult.PedidoDetalle](mapper.NoEncontradoM("pedido")), nil
	}
	return appresult.Ok(p), nil
}
