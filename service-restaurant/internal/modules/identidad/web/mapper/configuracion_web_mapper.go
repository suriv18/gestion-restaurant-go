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

func ToSucursal(s dtoresult.Sucursal) response.Sucursal {
	var eliminadoPor *string
	if s.EliminadoPor != nil {
		v := id(*s.EliminadoPor)
		eliminadoPor = &v
	}
	return response.Sucursal{
		ID: id(s.ID), NegocioID: id(s.NegocioID), Codigo: s.Codigo, Nombre: s.Nombre, Direccion: s.Direccion,
		Ubigeo: s.Ubigeo, CodigoEstablecimientoSunat: s.CodigoEstablecimientoSunat, Telefono: s.Telefono,
		EsActivo: s.EsActivo, MotivoEliminacion: s.MotivoEliminacion, EliminadoEn: s.EliminadoEn, EliminadoPor: eliminadoPor,
		CreadoEn: s.CreadoEn, CreadoPor: id(s.CreadoPor), ActualizadoEn: s.ActualizadoEn, ActualizadoPor: id(s.ActualizadoPor),
	}
}

func ToSucursalesPagina(p dtoresult.PaginaSucursales) response.SucursalesPagina {
	items := make([]response.Sucursal, len(p.Items))
	for i, s := range p.Items {
		items[i] = ToSucursal(s)
	}
	return response.SucursalesPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
