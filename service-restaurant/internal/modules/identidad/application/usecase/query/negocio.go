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

// ObtenerNegocio lee la empresa única: basta NEGOCIO_VER o NEGOCIO_ADMINISTRAR en alguna sede.
type ObtenerNegocio struct{ negocio outbound.NegocioReader }

func NewObtenerNegocio(n outbound.NegocioReader) *ObtenerNegocio { return &ObtenerNegocio{n} }

func (uc *ObtenerNegocio) Execute(ctx context.Context, q qry.ObtenerNegocio) (appresult.Result[dtoresult.Negocio], error) {
	if ae := mapper.Requiere(q.Actor, "no tiene permiso para consultar el negocio", policy.NegocioVer, policy.NegocioAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Negocio](ae), nil
	}
	n, found, err := uc.negocio.ObtenerNegocio(ctx)
	if err != nil {
		return appresult.Result[dtoresult.Negocio]{}, fmt.Errorf("obtener negocio: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Negocio](mapper.NoEncontrado("negocio")), nil
	}
	return appresult.Ok(n), nil
}
