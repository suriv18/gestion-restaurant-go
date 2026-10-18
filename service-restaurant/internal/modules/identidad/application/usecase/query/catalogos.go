package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/shared/application/pagination"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// tamanio normaliza el límite pedido (por defecto 25, máximo 100).
func tamanio(limite int) int {
	return pagination.PageRequest{Size: limite}.Normalize().Size
}

// ListarRoles pagina por cursor (id ascendente). Pide limite+1 filas para saber si hay más.
type ListarRoles struct{ catalogo outbound.CatalogoReader }

func NewListarRoles(c outbound.CatalogoReader) *ListarRoles { return &ListarRoles{c} }

func (uc *ListarRoles) Execute(ctx context.Context, q qry.ListarRoles) (appresult.Result[dtoresult.PaginaRoles], error) {
	if ae := mapper.RequiereAdministrar(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.PaginaRoles](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaRoles](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.catalogo.ListarRoles(ctx, desde, n+1)
	if err != nil {
		return appresult.Result[dtoresult.PaginaRoles]{}, fmt.Errorf("listar roles: %w", err)
	}
	pagina := dtoresult.PaginaRoles{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Rol{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerRol struct{ catalogo outbound.CatalogoReader }

func NewObtenerRol(c outbound.CatalogoReader) *ObtenerRol { return &ObtenerRol{c} }

func (uc *ObtenerRol) Execute(ctx context.Context, q qry.ObtenerRol) (appresult.Result[dtoresult.Rol], error) {
	if ae := mapper.RequiereAdministrar(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.Rol](ae), nil
	}
	r, found, err := uc.catalogo.ObtenerRol(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Rol]{}, fmt.Errorf("obtener rol: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Rol](mapper.NoEncontrado("rol")), nil
	}
	return appresult.Ok(r), nil
}

type ListarPermisos struct{ catalogo outbound.CatalogoReader }

func NewListarPermisos(c outbound.CatalogoReader) *ListarPermisos { return &ListarPermisos{c} }

func (uc *ListarPermisos) Execute(ctx context.Context, q qry.ListarPermisos) (appresult.Result[dtoresult.PaginaPermisos], error) {
	if ae := mapper.RequiereAdministrar(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.PaginaPermisos](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaPermisos](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.catalogo.ListarPermisos(ctx, desde, n+1)
	if err != nil {
		return appresult.Result[dtoresult.PaginaPermisos]{}, fmt.Errorf("listar permisos: %w", err)
	}
	pagina := dtoresult.PaginaPermisos{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Permiso{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerPermiso struct{ catalogo outbound.CatalogoReader }

func NewObtenerPermiso(c outbound.CatalogoReader) *ObtenerPermiso { return &ObtenerPermiso{c} }

func (uc *ObtenerPermiso) Execute(ctx context.Context, q qry.ObtenerPermiso) (appresult.Result[dtoresult.Permiso], error) {
	if ae := mapper.RequiereAdministrar(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.Permiso](ae), nil
	}
	p, found, err := uc.catalogo.ObtenerPermiso(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Permiso]{}, fmt.Errorf("obtener permiso: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Permiso](mapper.NoEncontrado("permiso")), nil
	}
	return appresult.Ok(p), nil
}
