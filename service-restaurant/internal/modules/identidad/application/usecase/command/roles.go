package command

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/vo"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxNombreRol      = 80
	maxDescripcionRol = 500
)

// CrearRol da de alta un rol de la aplicación (RES-RF-033).
type CrearRol struct{ roles outbound.RolStore }

func NewCrearRol(roles outbound.RolStore) *CrearRol { return &CrearRol{roles} }

func (uc *CrearRol) Execute(ctx context.Context, c cmd.CrearRol) (appresult.Result[dtoresult.Rol], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.Rol](ae)
	}
	codigo, err := vo.NewCodigoRol(c.Codigo)
	if err != nil {
		return rechazar[dtoresult.Rol](mapper.Validacion("codigo_invalido", "el código del rol debe tener 2 a 40 caracteres: mayúsculas, dígitos y _"))
	}
	nombre, err := vo.TextoObligatorio(c.Nombre, maxNombreRol)
	if err != nil {
		return rechazar[dtoresult.Rol](mapper.Validacion("nombre_invalido", "el nombre del rol es obligatorio (máximo 80 caracteres)"))
	}
	desc, ok := textoOpcional(c.Descripcion, maxDescripcionRol)
	if !ok {
		return rechazar[dtoresult.Rol](mapper.Validacion("descripcion_invalida", "la descripción excede 500 caracteres"))
	}
	rol, err := uc.roles.Crear(ctx, codigo.String(), nombre, desc)
	if err != nil {
		return deAlmacen[dtoresult.Rol]("crear rol", err)
	}
	return appresult.Ok(rol), nil
}

// EditarRol modifica solo los campos enviados.
type EditarRol struct{ roles outbound.RolStore }

func NewEditarRol(roles outbound.RolStore) *EditarRol { return &EditarRol{roles} }

func (uc *EditarRol) Execute(ctx context.Context, c cmd.EditarRol) (appresult.Result[dtoresult.Rol], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.Rol](ae)
	}
	if c.Codigo == nil && c.Nombre == nil && !c.DescripcionPresente {
		return rechazar[dtoresult.Rol](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosRol
	if c.Codigo != nil {
		codigo, err := vo.NewCodigoRol(*c.Codigo)
		if err != nil {
			return rechazar[dtoresult.Rol](mapper.Validacion("codigo_invalido", "el código del rol debe tener 2 a 40 caracteres: mayúsculas, dígitos y _"))
		}
		s := codigo.String()
		cambios.Codigo = &s
	}
	if c.Nombre != nil {
		nombre, err := vo.TextoObligatorio(*c.Nombre, maxNombreRol)
		if err != nil {
			return rechazar[dtoresult.Rol](mapper.Validacion("nombre_invalido", "el nombre del rol es obligatorio (máximo 80 caracteres)"))
		}
		cambios.Nombre = &nombre
	}
	if c.DescripcionPresente {
		desc, ok := textoOpcional(c.Descripcion, maxDescripcionRol)
		if !ok {
			return rechazar[dtoresult.Rol](mapper.Validacion("descripcion_invalida", "la descripción excede 500 caracteres"))
		}
		cambios.Descripcion, cambios.DescripcionPresente = desc, true
	}
	rol, found, err := uc.roles.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deAlmacen[dtoresult.Rol]("editar rol", err)
	}
	if !found {
		return rechazar[dtoresult.Rol](mapper.NoEncontrado("rol"))
	}
	return appresult.Ok(rol), nil
}

// CambiarEstadoRol desactiva o restaura un rol. Un rol inactivo deja de conceder permisos
// de inmediato; desactivarlo no puede dejar al sistema sin administrador.
type CambiarEstadoRol struct {
	roles   outbound.RolStore
	accesos outbound.AccesosStore
}

func NewCambiarEstadoRol(roles outbound.RolStore, accesos outbound.AccesosStore) *CambiarEstadoRol {
	return &CambiarEstadoRol{roles, accesos}
}

func (uc *CambiarEstadoRol) Execute(ctx context.Context, c cmd.CambiarEstadoRol) (appresult.Result[dtoresult.Rol], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.Rol](ae)
	}
	rol, found, err := uc.roles.CambiarActivo(ctx, c.ID, c.Activo)
	if err != nil {
		return deAlmacen[dtoresult.Rol]("cambiar estado del rol", err)
	}
	if !found {
		return rechazar[dtoresult.Rol](mapper.NoEncontrado("rol"))
	}
	if !c.Activo {
		if ae, err := ultimoAdministrador(ctx, uc.accesos); err != nil {
			return tecnico[dtoresult.Rol]("verificar administradores", err)
		} else if ae != nil {
			return rechazar[dtoresult.Rol](ae)
		}
	}
	return appresult.Ok(rol), nil
}
