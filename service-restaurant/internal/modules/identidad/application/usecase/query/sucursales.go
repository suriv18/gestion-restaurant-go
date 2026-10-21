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

// ListarSucursales devuelve solo las sedes que el actor puede ver. Pide limite+1 filas para
// saber si hay más páginas.
type ListarSucursales struct{ sucursales outbound.SucursalReader }

func NewListarSucursales(s outbound.SucursalReader) *ListarSucursales { return &ListarSucursales{s} }

func (uc *ListarSucursales) Execute(ctx context.Context, q qry.ListarSucursales) (appresult.Result[dtoresult.PaginaSucursales], error) {
	if ae := mapper.Requiere(q.Actor, "no tiene permiso para consultar sucursales", policy.SucursalVer, policy.SucursalAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaSucursales](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaSucursales](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.sucursales.ListarSucursales(ctx, outbound.FiltroSucursales{
		SedeIDs:          mapper.SedesVisibles(q.Actor),
		IncluirInactivas: q.Actor.CanAnywhere(policy.SucursalAdministrar),
		Activo:           q.Activo,
		DespuesDeID:      desde,
		Limite:           n + 1,
	})
	if err != nil {
		return appresult.Result[dtoresult.PaginaSucursales]{}, fmt.Errorf("listar sucursales: %w", err)
	}
	pagina := dtoresult.PaginaSucursales{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Sucursal{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerSucursal lee una sede; una sede que el actor no ve responde 404 para no revelarla.
type ObtenerSucursal struct{ sucursales outbound.SucursalReader }

func NewObtenerSucursal(s outbound.SucursalReader) *ObtenerSucursal { return &ObtenerSucursal{s} }

func (uc *ObtenerSucursal) Execute(ctx context.Context, q qry.ObtenerSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(q.Actor, "no tiene permiso para consultar sucursales", policy.SucursalVer, policy.SucursalAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Sucursal](ae), nil
	}
	s, found, err := uc.sucursales.ObtenerSucursal(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Sucursal]{}, fmt.Errorf("obtener sucursal: %w", err)
	}
	if !found || !mapper.SucursalVisible(q.Actor, s.ID, s.EsActivo) {
		return appresult.Fail[dtoresult.Sucursal](mapper.NoEncontrado("sucursal")), nil
	}
	return appresult.Ok(s), nil
}
