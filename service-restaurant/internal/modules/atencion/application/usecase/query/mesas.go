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

// ListarMesas devuelve las mesas de las sedes visibles. Pide limite+1 filas para saber si hay
// más páginas.
type ListarMesas struct{ mesas outbound.MesaReader }

func NewListarMesas(m outbound.MesaReader) *ListarMesas { return &ListarMesas{m} }

func (uc *ListarMesas) Execute(ctx context.Context, q qry.ListarMesas) (appresult.Result[dtoresult.PaginaMesas], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaMesas](ae), nil
	}
	sedes, ok := sedesPedidas(q.Actor, q.SucursalID)
	if !ok {
		return appresult.Fail[dtoresult.PaginaMesas](mapper.NoEncontrado("sucursal")), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaMesas](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.mesas.ListarMesas(ctx, outbound.FiltroMesas{SedeIDs: sedes, Activo: q.Activo, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaMesas]{}, fmt.Errorf("listar mesas: %w", err)
	}
	pagina := dtoresult.PaginaMesas{Items: items}
	if len(items) > n && n > 0 {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Mesa{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerMesa lee una mesa; la de una sede que el actor no ve responde 404.
type ObtenerMesa struct{ mesas outbound.MesaReader }

func NewObtenerMesa(m outbound.MesaReader) *ObtenerMesa { return &ObtenerMesa{m} }

func (uc *ObtenerMesa) Execute(ctx context.Context, q qry.ObtenerMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Mesa](ae), nil
	}
	m, found, err := uc.mesas.ObtenerMesa(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Mesa]{}, fmt.Errorf("obtener mesa: %w", err)
	}
	if !found || !mapper.PuedeVer(q.Actor, m.SucursalID) {
		return appresult.Fail[dtoresult.Mesa](mapper.NoEncontrado("mesa")), nil
	}
	return appresult.Ok(m), nil
}
