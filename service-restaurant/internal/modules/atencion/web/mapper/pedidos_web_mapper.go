package mapper

import (
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/web/dto/response"
)

func idOpcional(n *int64) *string {
	if n == nil {
		return nil
	}
	v := id(*n)
	return &v
}

func ToPedido(p dtoresult.Pedido) response.Pedido {
	return response.Pedido{
		ID: id(p.ID), SucursalID: id(p.SucursalID), CodigoPublico: p.CodigoPublico, ClaveIdempotencia: p.ClaveIdempotencia,
		MesaID: idOpcional(p.MesaID), ReservaID: idOpcional(p.ReservaID), ClienteID: idOpcional(p.ClienteID), MeseroID: idOpcional(p.MeseroID),
		Tipo: p.Tipo, Estado: p.Estado, Moneda: p.Moneda, Comensales: p.Comensales, AbiertoEn: p.AbiertoEn, CerradoEn: p.CerradoEn,
		Notas: p.Notas, MotivoCancelacion: p.MotivoCancelacion, VersionBloqueo: p.VersionBloqueo,
		CreadoEn: p.CreadoEn, CreadoPor: id(p.CreadoPor), ActualizadoEn: p.ActualizadoEn, ActualizadoPor: id(p.ActualizadoPor),
	}
}

func ToLinea(l dtoresult.Linea) response.Linea {
	return response.Linea{
		ID: id(l.ID), PedidoID: id(l.PedidoID), CuentaID: id(l.CuentaID), ProductoID: id(l.ProductoID),
		DetallePadreID: idOpcional(l.DetallePadreID), Descripcion: l.Descripcion, Cantidad: l.Cantidad,
		PrecioUnitario: l.PrecioUnitario, DescuentoTotal: l.DescuentoTotal, AfectacionIGV: l.AfectacionIGV,
		TasaImpuesto: l.TasaImpuesto, ImporteTotal: l.ImporteTotal, BaseImponible: l.BaseImponible, Estado: l.Estado,
		NotasPreparacion: l.NotasPreparacion, CreadoEn: l.CreadoEn, CreadoPor: id(l.CreadoPor),
		ActualizadoEn: l.ActualizadoEn, ActualizadoPor: id(l.ActualizadoPor),
	}
}

func ToPedidoDetalle(d dtoresult.PedidoDetalle) response.PedidoDetalle {
	out := response.PedidoDetalle{Pedido: ToPedido(d.Pedido), Cuentas: make([]response.Cuenta, len(d.Cuentas))}
	if d.Reparto != nil {
		out.Reparto = &response.Reparto{
			Destinatario: d.Reparto.Destinatario, Telefono: d.Reparto.Telefono, Direccion: d.Reparto.Direccion,
			Referencia: d.Reparto.Referencia, Canal: d.Reparto.Canal, Estado: d.Reparto.Estado,
		}
	}
	for i, c := range d.Cuentas {
		lineas := make([]response.Linea, len(c.Lineas))
		for j, l := range c.Lineas {
			lineas[j] = ToLinea(l)
		}
		out.Cuentas[i] = response.Cuenta{
			ID: id(c.ID), Numero: c.Numero, ClienteID: idOpcional(c.ClienteID), Etiqueta: c.Etiqueta, Estado: c.Estado,
			Subtotal: c.Subtotal, Impuesto: c.Impuesto, Total: c.Total, Lineas: lineas,
		}
	}
	return out
}

func ToPedidosPagina(p dtoresult.PaginaPedidos) response.PedidosPagina {
	items := make([]response.Pedido, len(p.Items))
	for i, it := range p.Items {
		items[i] = ToPedido(it)
	}
	return response.PedidosPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
