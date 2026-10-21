package command

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	"service-restaurant/internal/modules/identidad/domain/vo"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxRazonSocial     = 250
	maxNombreComercial = 150
	maxDireccionFiscal = 500
	maxTelefono        = 30
)

// EditarNegocio modifica solo los campos enviados de la empresa única (RES-RF-003).
type EditarNegocio struct{ negocio outbound.NegocioStore }

func NewEditarNegocio(n outbound.NegocioStore) *EditarNegocio { return &EditarNegocio{n} }

func (uc *EditarNegocio) Execute(ctx context.Context, c cmd.EditarNegocio) (appresult.Result[dtoresult.Negocio], error) {
	if ae := mapper.Requiere(c.Actor, "no tiene permiso para editar el negocio", policy.NegocioAdministrar); ae != nil {
		return rechazar[dtoresult.Negocio](ae)
	}
	if c.RazonSocial == nil && c.NombreComercial == nil && c.DireccionFiscal == nil && c.Moneda == nil &&
		c.ZonaHoraria == nil && !c.Ubigeo.Presente && !c.Telefono.Presente && !c.Email.Presente {
		return rechazar[dtoresult.Negocio](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosNegocio
	if c.RazonSocial != nil {
		v, err := vo.TextoObligatorio(*c.RazonSocial, maxRazonSocial)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("razon_social_invalida", "la razón social es obligatoria (máximo 250 caracteres)"))
		}
		cambios.RazonSocial = &v
	}
	if c.NombreComercial != nil {
		v, err := vo.TextoObligatorio(*c.NombreComercial, maxNombreComercial)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("nombre_comercial_invalido", "el nombre comercial es obligatorio (máximo 150 caracteres)"))
		}
		cambios.NombreComercial = &v
	}
	if c.DireccionFiscal != nil {
		v, err := vo.TextoObligatorio(*c.DireccionFiscal, maxDireccionFiscal)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("direccion_fiscal_invalida", "la dirección fiscal es obligatoria (máximo 500 caracteres)"))
		}
		cambios.DireccionFiscal = &v
	}
	if c.Moneda != nil {
		m, err := vo.NewMoneda(*c.Moneda)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("moneda_invalida", "la moneda debe ser un código de tres letras, como PEN"))
		}
		v := m.String()
		cambios.Moneda = &v
	}
	if c.ZonaHoraria != nil {
		z, err := vo.NewZonaHoraria(*c.ZonaHoraria)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("zona_horaria_invalida", "la zona horaria debe ser un nombre IANA, como America/Lima"))
		}
		v := z.String()
		cambios.ZonaHoraria = &v
	}
	var ok bool
	if cambios.Ubigeo, ok = opcionalValidado(c.Ubigeo, normUbigeo); !ok {
		return rechazar[dtoresult.Negocio](mapper.Validacion("ubigeo_invalido", "el ubigeo debe tener 6 dígitos"))
	}
	if cambios.Telefono, ok = opcionalValidado(c.Telefono, normMax(maxTelefono)); !ok {
		return rechazar[dtoresult.Negocio](mapper.Validacion("telefono_invalido", "el teléfono excede 30 caracteres"))
	}
	if cambios.Email, ok = opcionalValidado(c.Email, normEmail); !ok {
		return rechazar[dtoresult.Negocio](mapper.Validacion("email_invalido", "el correo electrónico no es válido"))
	}
	n, found, err := uc.negocio.Actualizar(ctx, cambios)
	if err != nil {
		return deAlmacen[dtoresult.Negocio]("editar negocio", err)
	}
	if !found {
		return rechazar[dtoresult.Negocio](mapper.NoEncontrado("negocio"))
	}
	return appresult.Ok(n), nil
}
