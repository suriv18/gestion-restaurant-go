package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// ListarProductos pide limite+1 filas para saber si hay más páginas. Sin es_activo lista solo los
// activos (spec D6); la capa web pide "todos" explícitamente.
type ListarProductos struct{ productos outbound.ProductoReader }

func NewListarProductos(p outbound.ProductoReader) *ListarProductos { return &ListarProductos{p} }

func (uc *ListarProductos) Execute(ctx context.Context, q qry.ListarProductos) (appresult.Result[dtoresult.PaginaProductos], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoCarta, policy.CartaVer, policy.CartaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaProductos](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaProductos](ae), nil
	}
	activo := q.Activo
	if activo == nil && !q.Todos {
		t := true
		activo = &t
	}
	n := tamanio(q.Limite)
	items, err := uc.productos.ListarProductos(ctx, outbound.FiltroProductos{
		Activo: activo, CategoriaID: q.CategoriaID, Texto: textoBusqueda(q.Texto), DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaProductos]{}, fmt.Errorf("listar productos: %w", err)
	}
	pagina := dtoresult.PaginaProductos{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Producto{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerProducto struct{ productos outbound.ProductoReader }

func NewObtenerProducto(p outbound.ProductoReader) *ObtenerProducto { return &ObtenerProducto{p} }

func (uc *ObtenerProducto) Execute(ctx context.Context, q qry.ObtenerProducto) (appresult.Result[dtoresult.Producto], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoCarta, policy.CartaVer, policy.CartaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Producto](ae), nil
	}
	p, found, err := uc.productos.ObtenerProducto(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Producto]{}, fmt.Errorf("obtener producto: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Producto](mapper.NoEncontrado("producto")), nil
	}
	return appresult.Ok(p), nil
}
