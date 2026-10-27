package mapper

import (
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/web/dto/response"
)

func ToImpuesto(i dtoresult.Impuesto) response.Impuesto {
	return response.Impuesto{
		ID: id(i.ID), Codigo: i.Codigo, Nombre: i.Nombre, Descripcion: i.Descripcion, EsActivo: i.EsActivo,
		CreadoEn: i.CreadoEn, CreadoPor: id(i.CreadoPor), ActualizadoEn: i.ActualizadoEn, ActualizadoPor: id(i.ActualizadoPor),
	}
}

func ToImpuestosPagina(p dtoresult.PaginaImpuestos) response.ImpuestosPagina {
	items := make([]response.Impuesto, len(p.Items))
	for k, i := range p.Items {
		items[k] = ToImpuesto(i)
	}
	return response.ImpuestosPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}

func ToTasaImpuesto(t dtoresult.TasaImpuesto) response.TasaImpuesto {
	return response.TasaImpuesto{
		ID: id(t.ID), ImpuestoID: id(t.ImpuestoID), Tasa: t.Tasa, VigenteDesde: t.VigenteDesde,
		VigenteHasta: t.VigenteHasta, Fundamento: t.Fundamento,
		CreadoEn: t.CreadoEn, CreadoPor: id(t.CreadoPor), ActualizadoEn: t.ActualizadoEn, ActualizadoPor: id(t.ActualizadoPor),
	}
}

func ToTasasPagina(p dtoresult.PaginaTasas) response.TasasPagina {
	items := make([]response.TasaImpuesto, len(p.Items))
	for k, t := range p.Items {
		items[k] = ToTasaImpuesto(t)
	}
	return response.TasasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
