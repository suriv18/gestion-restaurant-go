package command

import (
	"context"
	"errors"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoTasas  = "no tiene permiso para administrar las tasas de impuesto"
	mensajeVigenciaSolapada = "la vigencia se cruza con otra tasa del mismo impuesto (las vigencias son [desde, hasta) y no pueden solaparse)"
	mensajeImpuestoNoValido = "el impuesto no existe o está dado de baja"
)

func deTasa(err error, contexto string) (appresult.Result[dtoresult.TasaImpuesto], error) {
	if errors.Is(err, outbound.ErrVigenciaSolapada) {
		return rechazar[dtoresult.TasaImpuesto](apperror.New(apperror.KindConflict, CodigoVigenciaSolapada, mensajeVigenciaSolapada))
	}
	return deAlmacen[dtoresult.TasaImpuesto](contexto, "", err)
}

// CrearTasaImpuesto da de alta una vigencia para un impuesto activo.
type CrearTasaImpuesto struct{ tasas outbound.TasaStore }

func NewCrearTasaImpuesto(t outbound.TasaStore) *CrearTasaImpuesto { return &CrearTasaImpuesto{t} }

func (uc *CrearTasaImpuesto) Execute(ctx context.Context, c cmd.CrearTasaImpuesto) (appresult.Result[dtoresult.TasaImpuesto], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoTasas, policy.ImpuestoAdministrar); ae != nil {
		return rechazar[dtoresult.TasaImpuesto](ae)
	}
	// Ausente es 422 (obligatorio); presente pero mal formado es 400, siempre después del permiso.
	if c.ImpuestoID == "" {
		return rechazar[dtoresult.TasaImpuesto](mapper.Validacion("impuesto_id_invalido", "impuesto_id es obligatorio"))
	}
	impuestoID, ae := mapper.IDCadena(c.ImpuestoID, "impuesto_id")
	if ae != nil {
		return rechazar[dtoresult.TasaImpuesto](ae)
	}
	tasa, ae := validarTasa(c.Tasa)
	if ae != nil {
		return rechazar[dtoresult.TasaImpuesto](ae)
	}
	desde, hasta, ae := vigencia(c.VigenteDesde, c.VigenteHasta)
	if ae != nil {
		return rechazar[dtoresult.TasaImpuesto](ae)
	}
	fundamento, ae := opcional(c.Fundamento, "fundamento", fmtFundamento)
	if ae != nil {
		return rechazar[dtoresult.TasaImpuesto](ae)
	}
	activo, err := uc.tasas.ImpuestoActivo(ctx, impuestoID)
	if err != nil {
		return tecnico[dtoresult.TasaImpuesto]("comprobar impuesto", err)
	}
	if !activo {
		return rechazar[dtoresult.TasaImpuesto](apperror.New(apperror.KindValidation, mapper.CodigoReferenciaInvalida, mensajeImpuestoNoValido))
	}
	t, err := uc.tasas.Crear(ctx, outbound.NuevaTasa{ImpuestoID: impuestoID, Tasa: tasa, VigenteDesde: desde, VigenteHasta: hasta, Fundamento: fundamento})
	if err != nil {
		return deTasa(err, "crear tasa de impuesto")
	}
	return appresult.Ok(t), nil
}

// EditarTasaImpuesto modifica los campos enviados y valida la vigencia con el valor resultante,
// leyendo la fila con FOR UPDATE.
type EditarTasaImpuesto struct{ tasas outbound.TasaStore }

func NewEditarTasaImpuesto(t outbound.TasaStore) *EditarTasaImpuesto { return &EditarTasaImpuesto{t} }

func (uc *EditarTasaImpuesto) Execute(ctx context.Context, c cmd.EditarTasaImpuesto) (appresult.Result[dtoresult.TasaImpuesto], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoTasas, policy.ImpuestoAdministrar); ae != nil {
		return rechazar[dtoresult.TasaImpuesto](ae)
	}
	actual, found, err := uc.tasas.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.TasaImpuesto]("buscar tasa de impuesto", err)
	}
	if !found {
		return rechazar[dtoresult.TasaImpuesto](mapper.NoEncontrada("tasa de impuesto"))
	}
	if c.Tasa == nil && c.VigenteDesde == nil && !c.VigenteHasta.Presente && !c.Fundamento.Presente {
		return rechazar[dtoresult.TasaImpuesto](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	res := outbound.CambiosTasa{Tasa: actual.Tasa, VigenteDesde: actual.VigenteDesde, VigenteHasta: actual.VigenteHasta, Fundamento: actual.Fundamento}
	var ae *apperror.AppError
	if c.Tasa != nil {
		if res.Tasa, ae = validarTasa(*c.Tasa); ae != nil {
			return rechazar[dtoresult.TasaImpuesto](ae)
		}
	}
	if c.VigenteDesde != nil {
		res.VigenteDesde = *c.VigenteDesde
	}
	if c.VigenteHasta.Presente {
		res.VigenteHasta = c.VigenteHasta.Valor
	}
	// Se valida siempre el par resultante: editar solo una fecha puede invertir la vigencia.
	if res.VigenteDesde, res.VigenteHasta, ae = vigencia(res.VigenteDesde, res.VigenteHasta); ae != nil {
		return rechazar[dtoresult.TasaImpuesto](ae)
	}
	if c.Fundamento.Presente {
		if res.Fundamento, ae = opcional(c.Fundamento.Valor, "fundamento", fmtFundamento); ae != nil {
			return rechazar[dtoresult.TasaImpuesto](ae)
		}
	}
	t, found, err := uc.tasas.Actualizar(ctx, c.ID, res)
	if err != nil {
		return deTasa(err, "editar tasa de impuesto")
	}
	if !found {
		return rechazar[dtoresult.TasaImpuesto](mapper.NoEncontrada("tasa de impuesto"))
	}
	return appresult.Ok(t), nil
}
