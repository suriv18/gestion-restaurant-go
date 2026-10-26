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

const mensajeSinPermisoCarta = "no tiene permiso para consultar la carta"

// ListarCategorias pide limite+1 filas para saber si hay más páginas.
type ListarCategorias struct{ categorias outbound.CategoriaReader }

func NewListarCategorias(c outbound.CategoriaReader) *ListarCategorias { return &ListarCategorias{c} }

func (uc *ListarCategorias) Execute(ctx context.Context, q qry.ListarCategorias) (appresult.Result[dtoresult.PaginaCategorias], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoCarta, policy.CartaVer, policy.CartaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaCategorias](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaCategorias](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.categorias.ListarCategorias(ctx, outbound.FiltroCategorias{
		Activo: q.Activo, Texto: textoBusqueda(q.Texto), DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaCategorias]{}, fmt.Errorf("listar categorías: %w", err)
	}
	pagina := dtoresult.PaginaCategorias{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Categoria{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerCategoria struct{ categorias outbound.CategoriaReader }

func NewObtenerCategoria(c outbound.CategoriaReader) *ObtenerCategoria { return &ObtenerCategoria{c} }

func (uc *ObtenerCategoria) Execute(ctx context.Context, q qry.ObtenerCategoria) (appresult.Result[dtoresult.Categoria], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoCarta, policy.CartaVer, policy.CartaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Categoria](ae), nil
	}
	c, found, err := uc.categorias.ObtenerCategoria(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Categoria]{}, fmt.Errorf("obtener categoría: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Categoria](mapper.NoEncontrada("categoría")), nil
	}
	return appresult.Ok(c), nil
}
