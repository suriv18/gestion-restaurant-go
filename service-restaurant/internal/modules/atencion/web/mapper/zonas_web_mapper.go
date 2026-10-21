// Package mapper convierte resultados de aplicación en respuestas HTTP.
package mapper

import (
	"strconv"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/web/dto/response"
)

func id(n int64) string { return strconv.FormatInt(n, 10) }

func cursor(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}

func ToZona(z dtoresult.Zona) response.Zona {
	return response.Zona{
		ID: id(z.ID), SucursalID: id(z.SucursalID), Nombre: z.Nombre, OrdenVisualizacion: z.OrdenVisualizacion,
		EsActivo: z.EsActivo, CreadoEn: z.CreadoEn, CreadoPor: id(z.CreadoPor),
		ActualizadoEn: z.ActualizadoEn, ActualizadoPor: id(z.ActualizadoPor),
	}
}

func ToZonasPagina(p dtoresult.PaginaZonas) response.ZonasPagina {
	items := make([]response.Zona, len(p.Items))
	for i, z := range p.Items {
		items[i] = ToZona(z)
	}
	return response.ZonasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
