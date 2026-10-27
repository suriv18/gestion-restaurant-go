package mapper

import (
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/web/dto/response"
)

func idOpcional(n *int64) *string {
	if n == nil {
		return nil
	}
	v := id(*n)
	return &v
}

func ToEmpleado(e dtoresult.Empleado) response.Empleado {
	return response.Empleado{
		ID: id(e.ID), UsuarioID: idOpcional(e.UsuarioID), Username: e.Username,
		TipoDocumento: e.TipoDocumento, NumeroDocumento: e.NumeroDocumento, Nombres: e.Nombres, Apellidos: e.Apellidos,
		Telefono: e.Telefono, Cargo: e.Cargo, FechaIngreso: e.FechaIngreso, FechaCese: e.FechaCese,
		EsActivo: e.EsActivo, MotivoEliminacion: e.MotivoEliminacion, EliminadoEn: e.EliminadoEn, EliminadoPor: idOpcional(e.EliminadoPor),
		CreadoEn: e.CreadoEn, CreadoPor: id(e.CreadoPor), ActualizadoEn: e.ActualizadoEn, ActualizadoPor: id(e.ActualizadoPor),
	}
}

func ToEmpleadosLista(p dtoresult.PaginaEmpleados) response.EmpleadosLista {
	items := make([]response.Empleado, len(p.Items))
	for i, e := range p.Items {
		items[i] = ToEmpleado(e)
	}
	return response.EmpleadosLista{Items: items, NextCursor: cursor(p.NextCursor)}
}

func toEmpleadoResumen(e *dtoresult.EmpleadoResumen) *response.EmpleadoResumen {
	if e == nil {
		return nil
	}
	return &response.EmpleadoResumen{ID: id(e.ID), Nombres: e.Nombres, Apellidos: e.Apellidos, Cargo: e.Cargo}
}
