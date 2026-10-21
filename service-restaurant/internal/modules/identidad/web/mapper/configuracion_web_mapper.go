package mapper

import (
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/web/dto/response"
)

func ToNegocio(n dtoresult.Negocio) response.Negocio {
	return response.Negocio{
		ID: id(n.ID), Singleton: true, RUC: n.RUC, RazonSocial: n.RazonSocial, NombreComercial: n.NombreComercial,
		DireccionFiscal: n.DireccionFiscal, Ubigeo: n.Ubigeo, Telefono: n.Telefono, Email: n.Email,
		Moneda: n.Moneda, ZonaHoraria: n.ZonaHoraria, EsActivo: n.EsActivo,
		CreadoEn: n.CreadoEn, CreadoPor: id(n.CreadoPor), ActualizadoEn: n.ActualizadoEn, ActualizadoPor: id(n.ActualizadoPor),
	}
}
