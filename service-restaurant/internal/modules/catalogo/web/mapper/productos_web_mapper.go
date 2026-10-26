package mapper

import (
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/web/dto/response"
)

func idNulo(n *int64) *string {
	if n == nil {
		return nil
	}
	v := id(*n)
	return &v
}

func ToProducto(p dtoresult.Producto) response.Producto {
	r := response.Producto{
		ID: id(p.ID), CategoriaID: id(p.CategoriaID), Codigo: p.Codigo, Nombre: p.Nombre, Descripcion: p.Descripcion,
		Tipo: p.Tipo, ArticuloID: idNulo(p.ArticuloID), FactorConsumo: p.FactorConsumo, PrecioVenta: p.PrecioVenta,
		AfectacionIGV: p.AfectacionIGV, ImpuestoID: idNulo(p.ImpuestoID), ImagenURL: p.ImagenURL, EsActivo: p.EsActivo,
		MotivoEliminacion: p.MotivoEliminacion, EliminadoEn: p.EliminadoEn, EliminadoPor: eliminadoPor(p.Baja),
		CreadoEn: p.CreadoEn, CreadoPor: id(p.CreadoPor), ActualizadoEn: p.ActualizadoEn, ActualizadoPor: id(p.ActualizadoPor),
	}
	if p.ConSede {
		var e *response.EnSucursal
		if p.EnSucursal != nil {
			e = &response.EnSucursal{Disponible: p.EnSucursal.Disponible, PrecioSucursal: p.EnSucursal.PrecioSucursal,
				PrecioEfectivo: p.EnSucursal.PrecioEfectivo, EstacionID: idNulo(p.EnSucursal.EstacionID)}
		}
		r.EnSucursal = &e
	}
	return r
}

func ToProductosPagina(p dtoresult.PaginaProductos) response.ProductosPagina {
	items := make([]response.Producto, len(p.Items))
	for i, x := range p.Items {
		items[i] = ToProducto(x)
	}
	return response.ProductosPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}

func ToProductoSucursal(p dtoresult.ProductoSucursal) response.ProductoSucursal {
	return response.ProductoSucursal{
		ID: id(p.ID), ProductoID: id(p.ProductoID), SucursalID: id(p.SucursalID), EstacionID: idNulo(p.EstacionID),
		PrecioVenta: p.PrecioVenta, Disponible: p.Disponible, EsActivo: p.EsActivo,
		CreadoEn: p.CreadoEn, CreadoPor: id(p.CreadoPor), ActualizadoEn: p.ActualizadoEn, ActualizadoPor: id(p.ActualizadoPor),
	}
}
