// Package mapper convierte resultados de aplicación en respuestas HTTP.
package mapper

import (
	"strconv"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/web/dto/response"
)

func id(n int64) string { return strconv.FormatInt(n, 10) }

func idOpcional(n *int64) *string {
	if n == nil {
		return nil
	}
	v := id(*n)
	return &v
}

func cursor(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}

func ToLinea(l dtoresult.ComandaLinea) response.ComandaLinea {
	return response.ComandaLinea{
		ID: id(l.ID), ComandaID: id(l.ComandaID), PedidoID: id(l.PedidoID), DetallePedidoID: id(l.DetallePedidoID),
		Cantidad: l.Cantidad, Descripcion: l.Descripcion, Instrucciones: l.Instrucciones, Estado: l.Estado,
		IniciadoEn: l.IniciadoEn, ListoEn: l.ListoEn, EntregadoEn: l.EntregadoEn, MotivoCancelacion: l.MotivoCancelacion,
		CreadoEn: l.CreadoEn, CreadoPor: id(l.CreadoPor), ActualizadoEn: l.ActualizadoEn, ActualizadoPor: id(l.ActualizadoPor),
	}
}

func ToComanda(c dtoresult.Comanda) response.Comanda {
	lineas := make([]response.ComandaLinea, len(c.Lineas))
	for i, l := range c.Lineas {
		lineas[i] = ToLinea(l)
	}
	return response.Comanda{
		ID: id(c.ID), PedidoID: id(c.PedidoID), SucursalID: id(c.SucursalID), EstacionID: id(c.EstacionID),
		MesaID: idOpcional(c.MesaID), ClaveIdempotencia: c.ClaveIdempotencia, EnviadoEn: c.EnviadoEn,
		EnviadoPor: id(c.EnviadoPor), Notas: c.Notas, Lineas: lineas,
		CreadoEn: c.CreadoEn, CreadoPor: id(c.CreadoPor), ActualizadoEn: c.ActualizadoEn, ActualizadoPor: id(c.ActualizadoPor),
	}
}

func ToComandasPagina(p dtoresult.PaginaComandas) response.ComandasPagina {
	items := make([]response.Comanda, len(p.Items))
	for i, c := range p.Items {
		items[i] = ToComanda(c)
	}
	return response.ComandasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
