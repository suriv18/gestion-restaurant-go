package command

import (
	"context"
	"errors"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	"service-restaurant/internal/modules/atencion/domain/vo"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxNombreZona          = 80
	mensajeSinPermisoAdmin = "no tiene permiso para administrar zonas y mesas"

	CodigoNombreDuplicado = "nombre_duplicado"
	CodigoZonaInactiva    = "zona_inactiva"
	CodigoZonaConMesas    = "zona_con_mesas"
	CodigoYaDadaDeBaja    = "ya_dada_de_baja"
	CodigoYaActiva        = "ya_activa"
)

// CrearZona da de alta una zona en una sede donde el actor administra mesas (RES-RF-008).
type CrearZona struct{ zonas outbound.ZonaStore }

func NewCrearZona(z outbound.ZonaStore) *CrearZona { return &CrearZona{z} }

func (uc *CrearZona) Execute(ctx context.Context, c cmd.CrearZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	if ae := mapper.AutorizarSede(c.Actor, c.SucursalID); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	nombre, err := vo.TextoObligatorio(c.Nombre, maxNombreZona)
	if err != nil {
		return rechazar[dtoresult.Zona](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 80 caracteres)"))
	}
	orden := 0
	if c.OrdenVisualizacion != nil {
		o, err := vo.NewOrden(*c.OrdenVisualizacion)
		if err != nil {
			return rechazar[dtoresult.Zona](mapper.Validacion("orden_invalido", "el orden de visualización debe estar entre 0 y 9999"))
		}
		orden = o.Int()
	}
	z, err := uc.zonas.Crear(ctx, outbound.NuevaZona{SucursalID: c.SucursalID, Nombre: nombre, Orden: orden})
	if err != nil {
		return deZona(err, "crear zona")
	}
	return appresult.Ok(z), nil
}

// deZona traduce el nombre duplicado a un rechazo específico y delega lo demás.
func deZona(err error, contexto string) (appresult.Result[dtoresult.Zona], error) {
	if errors.Is(err, outbound.ErrNombreZonaDuplicado) {
		return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoNombreDuplicado, "ya existe una zona con ese nombre en la sucursal"))
	}
	return deAlmacen[dtoresult.Zona](contexto, err)
}

// zonaGestionable carga la zona (bloqueándola) y verifica que el actor ve su sede (si no, 404) y
// que la administra (si no, 403). Devuelve la zona o el rechazo; un error técnico va en err.
func zonaGestionable(ctx context.Context, store outbound.ZonaStore, actor appctx.Actor, id int64) (dtoresult.Zona, *apperror.AppError, error) {
	z, found, err := store.Obtener(ctx, id)
	if err != nil {
		return dtoresult.Zona{}, nil, err
	}
	if !found {
		return dtoresult.Zona{}, mapper.NoEncontrado("zona"), nil
	}
	if ae := mapper.AutorizarSede(actor, z.SucursalID); ae != nil {
		return dtoresult.Zona{}, ae, nil
	}
	return z, nil, nil
}

// EditarZona modifica solo los campos enviados de una zona activa.
type EditarZona struct{ zonas outbound.ZonaStore }

func NewEditarZona(z outbound.ZonaStore) *EditarZona { return &EditarZona{z} }

func (uc *EditarZona) Execute(ctx context.Context, c cmd.EditarZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	actual, ae, err := zonaGestionable(ctx, uc.zonas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Zona]("buscar zona", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoZonaInactiva, "la zona está dada de baja: restáurela para editarla"))
	}
	if c.Nombre == nil && c.OrdenVisualizacion == nil {
		return rechazar[dtoresult.Zona](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosZona
	if c.Nombre != nil {
		n, err := vo.TextoObligatorio(*c.Nombre, maxNombreZona)
		if err != nil {
			return rechazar[dtoresult.Zona](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 80 caracteres)"))
		}
		cambios.Nombre = &n
	}
	if c.OrdenVisualizacion != nil {
		o, err := vo.NewOrden(*c.OrdenVisualizacion)
		if err != nil {
			return rechazar[dtoresult.Zona](mapper.Validacion("orden_invalido", "el orden de visualización debe estar entre 0 y 9999"))
		}
		v := o.Int()
		cambios.Orden = &v
	}
	z, found, err := uc.zonas.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deZona(err, "editar zona")
	}
	if !found {
		return rechazar[dtoresult.Zona](mapper.NoEncontrado("zona"))
	}
	return appresult.Ok(z), nil
}

// CambiarEstadoZona desactiva (sin motivo) o restaura una zona. No se desactiva una zona con
// mesas activas: primero hay que dar de baja o mover las mesas.
type CambiarEstadoZona struct{ zonas outbound.ZonaStore }

func NewCambiarEstadoZona(z outbound.ZonaStore) *CambiarEstadoZona { return &CambiarEstadoZona{z} }

func (uc *CambiarEstadoZona) Execute(ctx context.Context, c cmd.CambiarEstadoZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	actual, ae, err := zonaGestionable(ctx, uc.zonas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Zona]("buscar zona", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	if c.Activo && actual.EsActivo {
		return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoYaActiva, "la zona ya está activa"))
	}
	if !c.Activo {
		if !actual.EsActivo {
			return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la zona ya está dada de baja"))
		}
		n, err := uc.zonas.ContarMesasActivas(ctx, c.ID)
		if err != nil {
			return tecnico[dtoresult.Zona]("contar mesas activas", err)
		}
		if n > 0 {
			return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoZonaConMesas,
				"la zona tiene mesas activas: dé de baja o mueva las mesas antes de desactivarla"))
		}
	}
	z, found, err := uc.zonas.CambiarActivo(ctx, c.ID, c.Activo)
	if err != nil {
		return deZona(err, "cambiar estado de la zona")
	}
	if !found {
		return rechazar[dtoresult.Zona](mapper.NoEncontrado("zona"))
	}
	return appresult.Ok(z), nil
}
