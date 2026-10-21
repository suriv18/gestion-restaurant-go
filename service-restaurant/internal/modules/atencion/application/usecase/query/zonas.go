// Package query implementa las consultas del módulo atencion.
package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	"service-restaurant/internal/shared/application/pagination"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoLectura = "no tiene permiso para consultar zonas y mesas"

// tamanio normaliza el límite pedido (por defecto 25, máximo 100).
func tamanio(limite int) int {
	return pagination.PageRequest{Size: limite}.Normalize().Size
}

// ListarZonas devuelve las zonas de las sedes visibles. Pide limite+1 filas para saber si hay
// más páginas.
type ListarZonas struct{ zonas outbound.ZonaReader }

func NewListarZonas(z outbound.ZonaReader) *ListarZonas { return &ListarZonas{z} }

func (uc *ListarZonas) Execute(ctx context.Context, q qry.ListarZonas) (appresult.Result[dtoresult.PaginaZonas], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaZonas](ae), nil
	}
	sedes, ok := sedesPedidas(q.Actor, q.SucursalID)
	if !ok {
		return appresult.Fail[dtoresult.PaginaZonas](mapper.NoEncontrado("sucursal")), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaZonas](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.zonas.ListarZonas(ctx, outbound.FiltroZonas{SedeIDs: sedes, Activo: q.Activo, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaZonas]{}, fmt.Errorf("listar zonas: %w", err)
	}
	pagina := dtoresult.PaginaZonas{Items: items}
	if len(items) > n && n > 0 {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Zona{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerZona lee una zona; la de una sede que el actor no ve responde 404.
type ObtenerZona struct{ zonas outbound.ZonaReader }

func NewObtenerZona(z outbound.ZonaReader) *ObtenerZona { return &ObtenerZona{z} }

func (uc *ObtenerZona) Execute(ctx context.Context, q qry.ObtenerZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Zona](ae), nil
	}
	z, found, err := uc.zonas.ObtenerZona(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Zona]{}, fmt.Errorf("obtener zona: %w", err)
	}
	if !found || !mapper.PuedeVer(q.Actor, z.SucursalID) {
		return appresult.Fail[dtoresult.Zona](mapper.NoEncontrado("zona")), nil
	}
	return appresult.Ok(z), nil
}
