package mapper

import (
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/web/dto/response"
)

func ToCategoria(c dtoresult.Categoria) response.Categoria {
	return response.Categoria{
		ID: id(c.ID), Nombre: c.Nombre, Descripcion: c.Descripcion, OrdenVisualizacion: c.Orden, EsActivo: c.EsActivo,
		CreadoEn: c.CreadoEn, CreadoPor: id(c.CreadoPor), ActualizadoEn: c.ActualizadoEn, ActualizadoPor: id(c.ActualizadoPor),
	}
}

func ToCategoriasPagina(p dtoresult.PaginaCategorias) response.CategoriasPagina {
	items := make([]response.Categoria, len(p.Items))
	for i, c := range p.Items {
		items[i] = ToCategoria(c)
	}
	return response.CategoriasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
