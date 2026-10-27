package query

import (
	"context"
	"fmt"
	"slices"

	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoEstaciones = "no tiene permiso para consultar estaciones de preparación"

// ListarEstaciones devuelve las estaciones de las sedes visibles (o de la pedida, si la ve). Pide
// limite+1 filas para saber si hay más páginas.
type ListarEstaciones struct{ estaciones outbound.EstacionReader }

func NewListarEstaciones(e outbound.EstacionReader) *ListarEstaciones { return &ListarEstaciones{e} }

func (uc *ListarEstaciones) Execute(ctx context.Context, q qry.ListarEstaciones) (appresult.Result[dtoresult.PaginaEstaciones], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoEstaciones, policy.EstacionVer, policy.EstacionAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaEstaciones](ae), nil
	}
	sedes := mapper.SedesVisibles(q.Actor)
	if q.SucursalID != nil {
		if !slices.Contains(sedes, *q.SucursalID) {
			return appresult.Fail[dtoresult.PaginaEstaciones](mapper.NoEncontrada("sucursal")), nil
		}
		sedes = []int64{*q.SucursalID}
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaEstaciones](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.estaciones.ListarEstaciones(ctx, outbound.FiltroEstaciones{
		SedeIDs: sedes, Activo: q.Activo, Texto: textoBusqueda(q.Texto), DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaEstaciones]{}, fmt.Errorf("listar estaciones: %w", err)
	}
	pagina := dtoresult.PaginaEstaciones{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Estacion{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerEstacion lee una estación; la de una sede que el actor no ve responde 404.
type ObtenerEstacion struct{ estaciones outbound.EstacionReader }

func NewObtenerEstacion(e outbound.EstacionReader) *ObtenerEstacion { return &ObtenerEstacion{e} }

func (uc *ObtenerEstacion) Execute(ctx context.Context, q qry.ObtenerEstacion) (appresult.Result[dtoresult.Estacion], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoEstaciones, policy.EstacionVer, policy.EstacionAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Estacion](ae), nil
	}
	e, found, err := uc.estaciones.ObtenerEstacion(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Estacion]{}, fmt.Errorf("obtener estación: %w", err)
	}
	if !found || !mapper.PuedeVerSede(q.Actor, e.SucursalID) {
		return appresult.Fail[dtoresult.Estacion](mapper.NoEncontrada("estación")), nil
	}
	return appresult.Ok(e), nil
}
