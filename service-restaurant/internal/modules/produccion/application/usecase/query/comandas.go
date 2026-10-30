// Package query implementa las consultas del módulo produccion.
package query

import (
	"context"
	"fmt"
	"slices"

	qry "service-restaurant/internal/modules/produccion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoCola = "no tiene permiso para consultar la cola de preparación"

// tamanio acota el tamaño de página a 1..100 (por defecto 25).
func tamanio(l int) int {
	switch {
	case l < 1:
		return 25
	case l > 100:
		return 100
	}
	return l
}

// ListarComandas devuelve la cola de las sedes donde el actor opera la cocina. Pide limite+1 filas
// para saber si hay más páginas.
type ListarComandas struct{ comandas outbound.ComandaReader }

func NewListarComandas(c outbound.ComandaReader) *ListarComandas { return &ListarComandas{c} }

func (uc *ListarComandas) Execute(ctx context.Context, q qry.ListarComandas) (appresult.Result[dtoresult.PaginaComandas], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoCola, policy.CocinaOperar); ae != nil {
		return appresult.Fail[dtoresult.PaginaComandas](ae), nil
	}
	sedes := mapper.SedesCocina(q.Actor)
	if q.SucursalID != nil {
		if !slices.Contains(sedes, *q.SucursalID) {
			return appresult.Fail[dtoresult.PaginaComandas](mapper.NoEncontrado("sucursal")), nil
		}
		sedes = []int64{*q.SucursalID}
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaComandas](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.comandas.ListarComandas(ctx, outbound.FiltroComandas{
		SedeIDs: sedes, EstacionID: q.EstacionID, Estado: q.Estado, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaComandas]{}, fmt.Errorf("listar comandas: %w", err)
	}
	pagina := dtoresult.PaginaComandas{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Comanda{}
	}
	return appresult.Ok(pagina), nil
}
