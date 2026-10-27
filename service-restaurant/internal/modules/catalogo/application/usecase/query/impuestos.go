package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/catalogo/application/dto/query"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoImpuestos = "no tiene permiso para consultar impuestos y tasas"

// puedeVerImpuestos: quien administra la carta también lee impuestos y tasas (formulario de productos).
func puedeVerImpuestos(a appctx.Actor) *apperror.AppError {
	return mapper.Requiere(a, mensajeSinPermisoImpuestos, policy.ImpuestoVer, policy.ImpuestoAdministrar, policy.CartaAdministrar)
}

type ListarImpuestos struct{ impuestos outbound.ImpuestoReader }

func NewListarImpuestos(r outbound.ImpuestoReader) *ListarImpuestos { return &ListarImpuestos{r} }

func (uc *ListarImpuestos) Execute(ctx context.Context, q qry.ListarImpuestos) (appresult.Result[dtoresult.PaginaImpuestos], error) {
	if ae := puedeVerImpuestos(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.PaginaImpuestos](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaImpuestos](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.impuestos.ListarImpuestos(ctx, outbound.FiltroImpuestos{Activo: q.Activo, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaImpuestos]{}, fmt.Errorf("listar impuestos: %w", err)
	}
	pagina := dtoresult.PaginaImpuestos{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Impuesto{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerImpuesto struct{ impuestos outbound.ImpuestoReader }

func NewObtenerImpuesto(r outbound.ImpuestoReader) *ObtenerImpuesto { return &ObtenerImpuesto{r} }

func (uc *ObtenerImpuesto) Execute(ctx context.Context, q qry.ObtenerImpuesto) (appresult.Result[dtoresult.Impuesto], error) {
	if ae := puedeVerImpuestos(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.Impuesto](ae), nil
	}
	i, found, err := uc.impuestos.ObtenerImpuesto(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Impuesto]{}, fmt.Errorf("obtener impuesto: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Impuesto](mapper.NoEncontrado("impuesto")), nil
	}
	return appresult.Ok(i), nil
}

type ListarTasasImpuesto struct{ impuestos outbound.ImpuestoReader }

func NewListarTasasImpuesto(r outbound.ImpuestoReader) *ListarTasasImpuesto {
	return &ListarTasasImpuesto{r}
}

func (uc *ListarTasasImpuesto) Execute(ctx context.Context, q qry.ListarTasasImpuesto) (appresult.Result[dtoresult.PaginaTasas], error) {
	if ae := puedeVerImpuestos(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.PaginaTasas](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaTasas](ae), nil
	}
	var impuestoID int64
	if q.ImpuestoID != "" {
		if impuestoID, ae = mapper.IDCadena(q.ImpuestoID, "impuesto_id"); ae != nil {
			return appresult.Fail[dtoresult.PaginaTasas](ae), nil
		}
	}
	n := tamanio(q.Limite)
	items, err := uc.impuestos.ListarTasas(ctx, outbound.FiltroTasas{ImpuestoID: impuestoID, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaTasas]{}, fmt.Errorf("listar tasas de impuesto: %w", err)
	}
	pagina := dtoresult.PaginaTasas{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.TasaImpuesto{}
	}
	return appresult.Ok(pagina), nil
}

type ObtenerTasaImpuesto struct{ impuestos outbound.ImpuestoReader }

func NewObtenerTasaImpuesto(r outbound.ImpuestoReader) *ObtenerTasaImpuesto {
	return &ObtenerTasaImpuesto{r}
}

func (uc *ObtenerTasaImpuesto) Execute(ctx context.Context, q qry.ObtenerTasaImpuesto) (appresult.Result[dtoresult.TasaImpuesto], error) {
	if ae := puedeVerImpuestos(q.Actor); ae != nil {
		return appresult.Fail[dtoresult.TasaImpuesto](ae), nil
	}
	t, found, err := uc.impuestos.ObtenerTasa(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.TasaImpuesto]{}, fmt.Errorf("obtener tasa de impuesto: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.TasaImpuesto](mapper.NoEncontrada("tasa de impuesto")), nil
	}
	return appresult.Ok(t), nil
}
