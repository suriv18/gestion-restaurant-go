package command

import (
	"context"
	"errors"
	"strings"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/mapper"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoEstaciones = "no tiene permiso para administrar estaciones de preparación"
	mensajeEstacionDuplicada    = "ya existe una estación con ese código en la sucursal (aunque esté dada de baja)"
	mensajeEstacionEnUso        = "la estación está asignada a productos activos de la sucursal: reasígnelos o desactívelos antes"
	mensajeSedeNoValida         = "la sucursal no existe o está dada de baja"
)

// deEstacion traduce los errores de almacenamiento propios de las estaciones. La sede inactiva
// (alta o restauración) la rechaza el trigger de referencias activas con un mensaje técnico:
// aquí se sustituye por uno curado.
func deEstacion(err error, contexto string) (appresult.Result[dtoresult.Estacion], error) {
	var regla *outbound.ErrReglaDatos
	switch {
	case errors.Is(err, outbound.ErrEstacionDuplicada):
		return rechazar[dtoresult.Estacion](apperror.New(apperror.KindConflict, CodigoEstacionDuplicada, mensajeEstacionDuplicada))
	case errors.As(err, &regla) && strings.Contains(regla.Mensaje, "maestro sucursales"):
		return rechazar[dtoresult.Estacion](apperror.New(apperror.KindValidation, mapper.CodigoReferenciaInvalida, mensajeSedeNoValida))
	}
	return deAlmacen[dtoresult.Estacion](contexto, "", err)
}

// CrearEstacion da de alta una estación en una sede que el actor administra.
type CrearEstacion struct{ estaciones outbound.EstacionStore }

func NewCrearEstacion(e outbound.EstacionStore) *CrearEstacion { return &CrearEstacion{e} }

func (uc *CrearEstacion) Execute(ctx context.Context, c cmd.CrearEstacion) (appresult.Result[dtoresult.Estacion], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEstaciones, policy.EstacionAdministrar); ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	// Primero la sede: si el actor no la ve, no se revela si existe.
	if ae := mapper.AutorizarSede(c.Actor, c.SucursalID); ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	codigo, ae := validarCodigoEstacion(c.Codigo)
	if ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	nombre, ae := validarNombreHasta(c.Nombre, "nombre", maxNombreEstacion)
	if ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	e, err := uc.estaciones.Crear(ctx, outbound.NuevaEstacion{SucursalID: c.SucursalID, Codigo: codigo, Nombre: nombre})
	if err != nil {
		return deEstacion(err, "crear estación")
	}
	return appresult.Ok(e), nil
}

// EditarEstacion modifica código y/o nombre de una estación activa. La sede no se edita.
type EditarEstacion struct{ estaciones outbound.EstacionStore }

func NewEditarEstacion(e outbound.EstacionStore) *EditarEstacion { return &EditarEstacion{e} }

func (uc *EditarEstacion) Execute(ctx context.Context, c cmd.EditarEstacion) (appresult.Result[dtoresult.Estacion], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEstaciones, policy.EstacionAdministrar); ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	actual, found, err := uc.estaciones.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Estacion]("buscar estación", err)
	}
	if !found {
		return rechazar[dtoresult.Estacion](mapper.NoEncontrada("estación"))
	}
	if ae := mapper.AutorizarEstacion(c.Actor, actual.SucursalID); ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Estacion](apperror.New(apperror.KindConflict, CodigoEstacionInactiva, "la estación está dada de baja: restáurela para editarla"))
	}
	if c.Codigo == nil && c.Nombre == nil {
		return rechazar[dtoresult.Estacion](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosEstacion
	if c.Codigo != nil {
		v, ae := validarCodigoEstacion(*c.Codigo)
		if ae != nil {
			return rechazar[dtoresult.Estacion](ae)
		}
		cambios.Codigo = &v
	}
	if c.Nombre != nil {
		v, ae := validarNombreHasta(*c.Nombre, "nombre", maxNombreEstacion)
		if ae != nil {
			return rechazar[dtoresult.Estacion](ae)
		}
		cambios.Nombre = &v
	}
	e, found, err := uc.estaciones.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deEstacion(err, "editar estación")
	}
	if !found {
		return rechazar[dtoresult.Estacion](mapper.NoEncontrada("estación"))
	}
	return appresult.Ok(e), nil
}

// CambiarEstadoEstacion da de baja (sin motivo) o restaura una estación. La baja se rechaza si
// aún la usan filas activas de productos_sucursal: la comprobación ocurre con la fila de la
// estación bloqueada. Restaurar exige la sede activa (lo valida el trigger).
type CambiarEstadoEstacion struct{ estaciones outbound.EstacionStore }

func NewCambiarEstadoEstacion(e outbound.EstacionStore) *CambiarEstadoEstacion {
	return &CambiarEstadoEstacion{e}
}

func (uc *CambiarEstadoEstacion) Execute(ctx context.Context, c cmd.CambiarEstadoEstacion) (appresult.Result[dtoresult.Estacion], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEstaciones, policy.EstacionAdministrar); ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	actual, found, err := uc.estaciones.Obtener(ctx, c.ID)
	if err != nil {
		return tecnico[dtoresult.Estacion]("buscar estación", err)
	}
	if !found {
		return rechazar[dtoresult.Estacion](mapper.NoEncontrada("estación"))
	}
	if ae := mapper.AutorizarEstacion(c.Actor, actual.SucursalID); ae != nil {
		return rechazar[dtoresult.Estacion](ae)
	}
	var e dtoresult.Estacion
	switch {
	case c.Activo && actual.EsActivo:
		return rechazar[dtoresult.Estacion](apperror.New(apperror.KindConflict, CodigoYaActiva, "la estación ya está activa"))
	case !c.Activo && !actual.EsActivo:
		return rechazar[dtoresult.Estacion](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la estación ya está dada de baja"))
	case c.Activo:
		e, found, err = uc.estaciones.Restaurar(ctx, c.ID)
	default:
		enUso, uerr := uc.estaciones.EnUso(ctx, c.ID)
		if uerr != nil {
			return tecnico[dtoresult.Estacion]("comprobar uso de la estación", uerr)
		}
		if enUso {
			return rechazar[dtoresult.Estacion](apperror.New(apperror.KindConflict, CodigoEstacionEnUso, mensajeEstacionEnUso))
		}
		e, found, err = uc.estaciones.DarBaja(ctx, c.ID)
	}
	if err != nil {
		return deEstacion(err, "cambiar estado de la estación")
	}
	if !found {
		return rechazar[dtoresult.Estacion](mapper.NoEncontrada("estación"))
	}
	return appresult.Ok(e), nil
}
