package command

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const maxDescripcionPermiso = 250

// contadorAdministradores es lo mínimo que necesita la regla del último administrador; la
// cumplen AccesosStore y SucursalStore (dar de baja una sede también quita permisos efectivos).
type contadorAdministradores interface {
	ContarAdministradoresEfectivos(ctx context.Context) (int, error)
}

// ultimoAdministrador comprueba, DESPUÉS de aplicar el cambio dentro de la transacción, que
// siga existiendo al menos un usuario con USUARIO_ADMINISTRAR efectivo. Si no, devuelve un
// rechazo y el decorador transaccional revierte el cambio.
func ultimoAdministrador(ctx context.Context, a contadorAdministradores) (*apperror.AppError, error) {
	n, err := a.ContarAdministradoresEfectivos(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return apperror.New(apperror.KindBusinessRule, mapper.CodigoUltimoAdministrador,
			"la operación dejaría al sistema sin ningún administrador activo"), nil
	}
	return nil, nil
}

// CrearPermiso da de alta un permiso funcional.
type CrearPermiso struct{ permisos outbound.PermisoStore }

func NewCrearPermiso(p outbound.PermisoStore) *CrearPermiso { return &CrearPermiso{p} }

func (uc *CrearPermiso) Execute(ctx context.Context, c cmd.CrearPermiso) (appresult.Result[dtoresult.Permiso], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.Permiso](ae)
	}
	codigo, err := vo.NewCodigoPermiso(c.Codigo)
	if err != nil {
		return rechazar[dtoresult.Permiso](mapper.Validacion("codigo_invalido", "el código del permiso debe tener 2 a 80 caracteres: mayúsculas, dígitos y _"))
	}
	desc, err := vo.TextoObligatorio(c.Descripcion, maxDescripcionPermiso)
	if err != nil {
		return rechazar[dtoresult.Permiso](mapper.Validacion("descripcion_invalida", "la descripción es obligatoria (máximo 250 caracteres)"))
	}
	p, err := uc.permisos.Crear(ctx, codigo.String(), desc)
	if err != nil {
		return deAlmacen[dtoresult.Permiso]("crear permiso", err)
	}
	return appresult.Ok(p), nil
}

// EditarPermiso modifica solo los campos enviados.
type EditarPermiso struct{ permisos outbound.PermisoStore }

func NewEditarPermiso(p outbound.PermisoStore) *EditarPermiso { return &EditarPermiso{p} }

func (uc *EditarPermiso) Execute(ctx context.Context, c cmd.EditarPermiso) (appresult.Result[dtoresult.Permiso], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.Permiso](ae)
	}
	if c.Codigo == nil && c.Descripcion == nil {
		return rechazar[dtoresult.Permiso](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosPermiso
	if c.Codigo != nil {
		codigo, err := vo.NewCodigoPermiso(*c.Codigo)
		if err != nil {
			return rechazar[dtoresult.Permiso](mapper.Validacion("codigo_invalido", "el código del permiso debe tener 2 a 80 caracteres: mayúsculas, dígitos y _"))
		}
		s := codigo.String()
		cambios.Codigo = &s
	}
	if c.Descripcion != nil {
		desc, err := vo.TextoObligatorio(*c.Descripcion, maxDescripcionPermiso)
		if err != nil {
			return rechazar[dtoresult.Permiso](mapper.Validacion("descripcion_invalida", "la descripción es obligatoria (máximo 250 caracteres)"))
		}
		cambios.Descripcion = &desc
	}
	p, found, err := uc.permisos.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deAlmacen[dtoresult.Permiso]("editar permiso", err)
	}
	if !found {
		return rechazar[dtoresult.Permiso](mapper.NoEncontrado("permiso"))
	}
	return appresult.Ok(p), nil
}

// CambiarEstadoPermiso desactiva o restaura un permiso; desactivarlo no puede dejar al
// sistema sin administrador (p. ej. desactivar USUARIO_ADMINISTRAR).
type CambiarEstadoPermiso struct {
	permisos outbound.PermisoStore
	accesos  outbound.AccesosStore
}

func NewCambiarEstadoPermiso(p outbound.PermisoStore, a outbound.AccesosStore) *CambiarEstadoPermiso {
	return &CambiarEstadoPermiso{p, a}
}

func (uc *CambiarEstadoPermiso) Execute(ctx context.Context, c cmd.CambiarEstadoPermiso) (appresult.Result[dtoresult.Permiso], error) {
	if ae := mapper.RequiereAdministrar(c.Actor); ae != nil {
		return rechazar[dtoresult.Permiso](ae)
	}
	p, found, err := uc.permisos.CambiarActivo(ctx, c.ID, c.Activo)
	if err != nil {
		return deAlmacen[dtoresult.Permiso]("cambiar estado del permiso", err)
	}
	if !found {
		return rechazar[dtoresult.Permiso](mapper.NoEncontrado("permiso"))
	}
	if !c.Activo {
		if ae, err := ultimoAdministrador(ctx, uc.accesos); err != nil {
			return tecnico[dtoresult.Permiso]("verificar administradores", err)
		} else if ae != nil {
			return rechazar[dtoresult.Permiso](ae)
		}
	}
	return appresult.Ok(p), nil
}
