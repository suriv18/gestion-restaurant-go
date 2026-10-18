package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// ListarUsuarios devuelve el equipo con sus roles y sedes vigentes. Pide limite+1 filas para
// saber si hay más páginas. El actor técnico nunca aparece (lo garantiza el lector).
type ListarUsuarios struct{ usuarios outbound.UsuarioReader }

func NewListarUsuarios(u outbound.UsuarioReader) *ListarUsuarios { return &ListarUsuarios{u} }

func (uc *ListarUsuarios) Execute(ctx context.Context, q qry.ListarUsuarios) (appresult.Result[dtoresult.PaginaUsuarios], error) {
	if ae := mapper.RequiereAdministrar(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.PaginaUsuarios](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaUsuarios](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.usuarios.ListarUsuarios(ctx, desde, n+1)
	if err != nil {
		return appresult.Result[dtoresult.PaginaUsuarios]{}, fmt.Errorf("listar usuarios: %w", err)
	}
	pagina := dtoresult.PaginaUsuarios{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.UsuarioConAccesos{}
	}
	return appresult.Ok(pagina), nil
}

// ListarSucursalesAsignables devuelve solo las sedes donde el actor administra accesos: es
// exactamente el conjunto que la regla de AsignarAccesos le permite agregar o quitar.
type ListarSucursalesAsignables struct{ usuarios outbound.UsuarioReader }

func NewListarSucursalesAsignables(u outbound.UsuarioReader) *ListarSucursalesAsignables {
	return &ListarSucursalesAsignables{u}
}

func (uc *ListarSucursalesAsignables) Execute(ctx context.Context, q qry.ListarSucursalesAsignables) (appresult.Result[[]dtoresult.SucursalResumen], error) {
	if ae := mapper.RequiereAdministrar(q.Actor); ae != nil {
		return appresult.Fail[[]dtoresult.SucursalResumen](ae), nil
	}
	sedes, err := uc.usuarios.SucursalesActivasPorID(ctx, q.Actor.SucursalesCon(policy.UsuarioAdministrar))
	if err != nil {
		return appresult.Result[[]dtoresult.SucursalResumen]{}, fmt.Errorf("listar sucursales asignables: %w", err)
	}
	if sedes == nil {
		sedes = []dtoresult.SucursalResumen{}
	}
	return appresult.Ok(sedes), nil
}
