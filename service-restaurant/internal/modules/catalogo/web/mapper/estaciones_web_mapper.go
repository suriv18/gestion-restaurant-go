package mapper

import (
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/web/dto/response"
)

func ToEstacion(e dtoresult.Estacion) response.Estacion {
	return response.Estacion{
		ID: id(e.ID), SucursalID: id(e.SucursalID), Codigo: e.Codigo, Nombre: e.Nombre, EsActivo: e.EsActivo,
		CreadoEn: e.CreadoEn, CreadoPor: id(e.CreadoPor), ActualizadoEn: e.ActualizadoEn, ActualizadoPor: id(e.ActualizadoPor),
	}
}

func ToEstacionesPagina(p dtoresult.PaginaEstaciones) response.EstacionesPagina {
	items := make([]response.Estacion, len(p.Items))
	for i, e := range p.Items {
		items[i] = ToEstacion(e)
	}
	return response.EstacionesPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
