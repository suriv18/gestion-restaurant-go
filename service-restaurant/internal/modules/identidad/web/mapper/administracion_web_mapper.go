package mapper

import (
	"strconv"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/web/dto/response"
)

func id(n int64) string { return strconv.FormatInt(n, 10) }

func ToRol(r dtoresult.Rol) response.Rol {
	return response.Rol{
		ID: id(r.ID), Codigo: r.Codigo, Nombre: r.Nombre, Descripcion: r.Descripcion, EsActivo: r.EsActivo,
		CreadoEn: r.CreadoEn, CreadoPor: id(r.CreadoPor), ActualizadoEn: r.ActualizadoEn, ActualizadoPor: id(r.ActualizadoPor),
	}
}

func ToPermiso(p dtoresult.Permiso) response.Permiso {
	return response.Permiso{
		ID: id(p.ID), Codigo: p.Codigo, Descripcion: p.Descripcion, EsActivo: p.EsActivo,
		CreadoEn: p.CreadoEn, CreadoPor: id(p.CreadoPor), ActualizadoEn: p.ActualizadoEn, ActualizadoPor: id(p.ActualizadoPor),
	}
}

func ToUsuario(u dtoresult.UsuarioAdmin) response.Usuario {
	return response.Usuario{
		ID: id(u.ID), Username: u.Username, Email: u.Email, EsSistema: u.EsSistema, EsActivo: u.EsActivo,
		UltimoAccesoEn: u.UltimoAccesoEn, BloqueadoHasta: u.BloqueadoHasta, IntentosFallidos: u.IntentosFallidos,
		CreadoEn: u.CreadoEn, CreadoPor: id(u.CreadoPor), ActualizadoEn: u.ActualizadoEn, ActualizadoPor: id(u.ActualizadoPor),
	}
}

func cursor(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}

func ToRolesLista(p dtoresult.PaginaRoles) response.RolesLista {
	items := make([]response.Rol, len(p.Items))
	for i, r := range p.Items {
		items[i] = ToRol(r)
	}
	return response.RolesLista{Items: items, NextCursor: cursor(p.NextCursor)}
}

func ToPermisosLista(p dtoresult.PaginaPermisos) response.PermisosLista {
	items := make([]response.Permiso, len(p.Items))
	for i, x := range p.Items {
		items[i] = ToPermiso(x)
	}
	return response.PermisosLista{Items: items, NextCursor: cursor(p.NextCursor)}
}
