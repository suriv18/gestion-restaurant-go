package command

import (
	"context"
	"errors"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	"service-restaurant/internal/modules/identidad/domain/vo"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxNombreSucursal    = 120
	maxDireccionSucursal = 500
	maxCodigoSunat       = 10

	CodigoCodigoDuplicado   = "codigo_duplicado"
	CodigoUltimaSedeActiva  = "ultima_sede_activa"
	CodigoYaDadaDeBaja      = "ya_dada_de_baja"
	CodigoYaActiva          = "ya_activa"
	mensajeSinPermisoCrear  = "no tiene permiso para administrar sucursales"
	mensajeSedeNoAutorizada = "no administra esta sucursal"
)

// CrearSucursal da de alta una sede de la empresa única (RES-RF-003). Quien la crea no recibe
// acceso a ella: las asignaciones se hacen en Equipo y accesos.
type CrearSucursal struct{ sucursales outbound.SucursalStore }

func NewCrearSucursal(s outbound.SucursalStore) *CrearSucursal { return &CrearSucursal{s} }

func (uc *CrearSucursal) Execute(ctx context.Context, c cmd.CrearSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCrear, policy.SucursalAdministrar); ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	codigo, err := vo.NewCodigoSucursal(c.Codigo)
	if err != nil {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_invalido", "el código debe tener 2 a 20 caracteres: mayúsculas, dígitos, _ y -"))
	}
	nombre, err := vo.TextoObligatorio(c.Nombre, maxNombreSucursal)
	if err != nil {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 120 caracteres)"))
	}
	direccion, err := vo.TextoObligatorio(c.Direccion, maxDireccionSucursal)
	if err != nil {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("direccion_invalida", "la dirección es obligatoria (máximo 500 caracteres)"))
	}
	ubigeo, ok := valorOpcional(c.Ubigeo, normUbigeo)
	if !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("ubigeo_invalido", "el ubigeo debe tener 6 dígitos"))
	}
	sunat, ok := valorOpcional(c.CodigoEstablecimientoSunat, normMax(maxCodigoSunat))
	if !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_establecimiento_sunat_invalido", "el código de establecimiento SUNAT excede 10 caracteres"))
	}
	telefono, ok := valorOpcional(c.Telefono, normMax(maxTelefono))
	if !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("telefono_invalido", "el teléfono excede 30 caracteres"))
	}
	s, err := uc.sucursales.Crear(ctx, outbound.NuevaSucursal{
		Codigo: codigo.String(), Nombre: nombre, Direccion: direccion,
		Ubigeo: ubigeo, CodigoEstablecimientoSunat: sunat, Telefono: telefono,
	})
	if err != nil {
		return deSucursal(err, "crear sucursal")
	}
	return appresult.Ok(s), nil
}

// deSucursal traduce el código duplicado a un rechazo específico y delega lo demás.
func deSucursal(err error, contexto string) (appresult.Result[dtoresult.Sucursal], error) {
	if errors.Is(err, outbound.ErrCodigoSucursalDuplicado) {
		return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindConflict, CodigoCodigoDuplicado, "ya existe una sucursal con ese código"))
	}
	return deAlmacen[dtoresult.Sucursal](contexto, err)
}

// sedeGestionable carga la sede (bloqueándola) y verifica que el actor la ve (si no, 404) y que
// puede administrarla (si no, 403). Devuelve la sede o el rechazo; un error técnico va en err.
func sedeGestionable(ctx context.Context, store outbound.SucursalStore, actor appctx.Actor, id int64) (dtoresult.Sucursal, *apperror.AppError, error) {
	s, found, err := store.Obtener(ctx, id)
	if err != nil {
		return dtoresult.Sucursal{}, nil, err
	}
	if !found || !mapper.SucursalVisible(actor, s.ID, s.EsActivo) {
		return dtoresult.Sucursal{}, mapper.NoEncontrado("sucursal"), nil
	}
	if !mapper.PuedeAdministrarSucursal(actor, s.ID, s.EsActivo) {
		return dtoresult.Sucursal{}, apperror.New(apperror.KindForbidden, mapper.CodigoSedeNoAutorizada, mensajeSedeNoAutorizada), nil
	}
	return s, nil, nil
}

// EditarSucursal modifica solo los campos enviados.
type EditarSucursal struct{ sucursales outbound.SucursalStore }

func NewEditarSucursal(s outbound.SucursalStore) *EditarSucursal { return &EditarSucursal{s} }

func (uc *EditarSucursal) Execute(ctx context.Context, c cmd.EditarSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCrear, policy.SucursalAdministrar); ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	if _, ae, err := sedeGestionable(ctx, uc.sucursales, c.Actor, c.ID); err != nil {
		return tecnico[dtoresult.Sucursal]("buscar sucursal", err)
	} else if ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	if c.Codigo == nil && c.Nombre == nil && c.Direccion == nil &&
		!c.Ubigeo.Presente && !c.CodigoEstablecimientoSunat.Presente && !c.Telefono.Presente {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosSucursal
	if c.Codigo != nil {
		codigo, err := vo.NewCodigoSucursal(*c.Codigo)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_invalido", "el código debe tener 2 a 20 caracteres: mayúsculas, dígitos, _ y -"))
		}
		v := codigo.String()
		cambios.Codigo = &v
	}
	if c.Nombre != nil {
		v, err := vo.TextoObligatorio(*c.Nombre, maxNombreSucursal)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 120 caracteres)"))
		}
		cambios.Nombre = &v
	}
	if c.Direccion != nil {
		v, err := vo.TextoObligatorio(*c.Direccion, maxDireccionSucursal)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("direccion_invalida", "la dirección es obligatoria (máximo 500 caracteres)"))
		}
		cambios.Direccion = &v
	}
	var ok bool
	if cambios.Ubigeo, ok = opcionalValidado(c.Ubigeo, normUbigeo); !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("ubigeo_invalido", "el ubigeo debe tener 6 dígitos"))
	}
	if cambios.CodigoEstablecimientoSunat, ok = opcionalValidado(c.CodigoEstablecimientoSunat, normMax(maxCodigoSunat)); !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_establecimiento_sunat_invalido", "el código de establecimiento SUNAT excede 10 caracteres"))
	}
	if cambios.Telefono, ok = opcionalValidado(c.Telefono, normMax(maxTelefono)); !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("telefono_invalido", "el teléfono excede 30 caracteres"))
	}
	s, found, err := uc.sucursales.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deSucursal(err, "editar sucursal")
	}
	if !found {
		return rechazar[dtoresult.Sucursal](mapper.NoEncontrado("sucursal"))
	}
	return appresult.Ok(s), nil
}

// CambiarEstadoSucursal da de baja (con motivo) o restaura una sede. Nunca deja a la empresa
// sin sede activa; la restauración conserva el ID.
type CambiarEstadoSucursal struct{ sucursales outbound.SucursalStore }

func NewCambiarEstadoSucursal(s outbound.SucursalStore) *CambiarEstadoSucursal {
	return &CambiarEstadoSucursal{s}
}

func (uc *CambiarEstadoSucursal) Execute(ctx context.Context, c cmd.CambiarEstadoSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCrear, policy.SucursalAdministrar); ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	motivo := ""
	if !c.Activo {
		m, err := vo.TextoObligatorio(c.Motivo, maxMotivo)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("motivo_requerido", "la baja exige un motivo (máximo 2000 caracteres)"))
		}
		motivo = m
	}
	actual, ae, err := sedeGestionable(ctx, uc.sucursales, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Sucursal]("buscar sucursal", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	if c.Activo {
		if actual.EsActivo {
			return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindConflict, CodigoYaActiva, "la sucursal ya está activa"))
		}
		s, found, err := uc.sucursales.Restaurar(ctx, c.ID)
		if err != nil {
			return deSucursal(err, "restaurar sucursal")
		}
		if !found {
			return rechazar[dtoresult.Sucursal](mapper.NoEncontrado("sucursal"))
		}
		return appresult.Ok(s), nil
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la sucursal ya está dada de baja"))
	}
	if err := uc.sucursales.BloquearNegocio(ctx); err != nil {
		return tecnico[dtoresult.Sucursal]("bloquear negocio", err)
	}
	activas, err := uc.sucursales.ContarActivas(ctx)
	if err != nil {
		return tecnico[dtoresult.Sucursal]("contar sucursales activas", err)
	}
	if activas <= 1 {
		return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindBusinessRule, CodigoUltimaSedeActiva,
			"no se puede dar de baja la última sucursal activa del negocio"))
	}
	s, found, err := uc.sucursales.DarBaja(ctx, c.ID, motivo)
	if err != nil {
		return deSucursal(err, "dar de baja la sucursal")
	}
	if !found {
		return rechazar[dtoresult.Sucursal](mapper.NoEncontrado("sucursal"))
	}
	return appresult.Ok(s), nil
}
