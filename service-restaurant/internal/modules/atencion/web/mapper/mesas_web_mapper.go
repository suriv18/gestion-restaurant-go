package mapper

import (
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/web/dto/response"
)

func ToMesa(m dtoresult.Mesa) response.Mesa {
	var eliminadoPor *string
	if m.EliminadoPor != nil {
		v := id(*m.EliminadoPor)
		eliminadoPor = &v
	}
	return response.Mesa{
		ID: id(m.ID), SucursalID: id(m.SucursalID), ZonaID: id(m.ZonaID), NumeroMesa: m.NumeroMesa,
		Capacidad: m.Capacidad, Habilitada: m.Habilitada, PosicionX: m.PosicionX, PosicionY: m.PosicionY,
		EsActivo: m.EsActivo, MotivoEliminacion: m.MotivoEliminacion, EliminadoEn: m.EliminadoEn, EliminadoPor: eliminadoPor,
		CreadoEn: m.CreadoEn, CreadoPor: id(m.CreadoPor), ActualizadoEn: m.ActualizadoEn, ActualizadoPor: id(m.ActualizadoPor),
	}
}

func ToMesasPagina(p dtoresult.PaginaMesas) response.MesasPagina {
	items := make([]response.Mesa, len(p.Items))
	for i, m := range p.Items {
		items[i] = ToMesa(m)
	}
	return response.MesasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
