// Package mapper convierte resultados de aplicación en respuestas HTTP.
package mapper

import (
	"strconv"

	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/web/dto/response"
)

func id(n int64) string { return strconv.FormatInt(n, 10) }

func cursor(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}

func eliminadoPor(b dtoresult.Baja) *string {
	if b.EliminadoPor == nil {
		return nil
	}
	v := id(*b.EliminadoPor)
	return &v
}

func ToCliente(c dtoresult.Cliente) response.Cliente {
	return response.Cliente{
		ID: id(c.ID), TipoDocumento: c.TipoDocumento, NumeroDocumento: c.NumeroDocumento,
		NombreRazonSocial: c.Nombre, Telefono: c.Telefono, Email: c.Email, Direccion: c.Direccion,
		EsActivo: c.EsActivo, MotivoEliminacion: c.MotivoEliminacion, EliminadoEn: c.EliminadoEn, EliminadoPor: eliminadoPor(c.Baja),
		CreadoEn: c.CreadoEn, CreadoPor: id(c.CreadoPor), ActualizadoEn: c.ActualizadoEn, ActualizadoPor: id(c.ActualizadoPor),
	}
}

func ToClientesPagina(p dtoresult.PaginaClientes) response.ClientesPagina {
	items := make([]response.Cliente, len(p.Items))
	for i, c := range p.Items {
		items[i] = ToCliente(c)
	}
	return response.ClientesPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}

func ToProveedor(p dtoresult.Proveedor) response.Proveedor {
	return response.Proveedor{
		ID: id(p.ID), TipoDocumento: p.TipoDocumento, NumeroDocumento: p.NumeroDocumento, RazonSocial: p.RazonSocial,
		ContactoNombre: p.ContactoNombre, Telefono: p.Telefono, Email: p.Email, Direccion: p.Direccion,
		EsActivo: p.EsActivo, MotivoEliminacion: p.MotivoEliminacion, EliminadoEn: p.EliminadoEn, EliminadoPor: eliminadoPor(p.Baja),
		CreadoEn: p.CreadoEn, CreadoPor: id(p.CreadoPor), ActualizadoEn: p.ActualizadoEn, ActualizadoPor: id(p.ActualizadoPor),
	}
}

func ToProveedoresPagina(p dtoresult.PaginaProveedores) response.ProveedoresPagina {
	items := make([]response.Proveedor, len(p.Items))
	for i, x := range p.Items {
		items[i] = ToProveedor(x)
	}
	return response.ProveedoresPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
