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
	maxMotivo = 2000

	CodigoNumeroMesaDuplicado = "numero_mesa_duplicado"
	CodigoZonaInvalida        = "zona_invalida"
	CodigoMesaInactiva        = "mesa_inactiva"
)

// deMesa traduce el número duplicado a un rechazo específico y delega lo demás.
func deMesa(err error, contexto string) (appresult.Result[dtoresult.Mesa], error) {
	if errors.Is(err, outbound.ErrNumeroMesaDuplicado) {
		return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoNumeroMesaDuplicado, "ya existe una mesa con ese número en la sucursal"))
	}
	return deAlmacen[dtoresult.Mesa](contexto, err)
}

func zonaInvalida() *apperror.AppError {
	return mapper.Validacion(CodigoZonaInvalida, "la zona no existe, está dada de baja o pertenece a otra sucursal")
}

// zonaValida comprueba que la zona exista, esté activa y sea de la sede indicada.
func zonaValida(ctx context.Context, store outbound.MesaStore, zonaID, sedeID int64) (bool, error) {
	z, found, err := store.ZonaDeMesa(ctx, zonaID)
	if err != nil {
		return false, err
	}
	return found && z.EsActiva && z.SucursalID == sedeID, nil
}

// normPosicion valida un decimal de numeric(10,2) y lo devuelve normalizado.
func normPosicion(s string) (string, bool) {
	p, err := vo.NewPosicion(s)
	return p.String(), err == nil
}

// posicionOpcional valida una posición anulable de un PATCH: ausente → no se toca; null o en
// blanco → se vacía; con valor → se valida.
func posicionOpcional(c cmd.Campo[string]) (outbound.CampoOpcional, bool) {
	if !c.Presente {
		return outbound.CampoOpcional{}, true
	}
	if c.Valor == nil || *c.Valor == "" {
		return outbound.CampoOpcional{Presente: true}, true
	}
	v, ok := normPosicion(*c.Valor)
	if !ok {
		return outbound.CampoOpcional{}, false
	}
	return outbound.CampoOpcional{Presente: true, Valor: &v}, true
}

// CrearMesa da de alta una mesa en una sede donde el actor administra mesas (RES-RF-008).
type CrearMesa struct{ mesas outbound.MesaStore }

func NewCrearMesa(m outbound.MesaStore) *CrearMesa { return &CrearMesa{m} }

func (uc *CrearMesa) Execute(ctx context.Context, c cmd.CrearMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	if ae := mapper.AutorizarSede(c.Actor, c.SucursalID); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	numero, err := vo.NewNumeroMesa(c.NumeroMesa)
	if err != nil {
		return rechazar[dtoresult.Mesa](mapper.Validacion("numero_mesa_invalido", "el número de mesa es obligatorio (máximo 20 caracteres)"))
	}
	capacidad, err := vo.NewCapacidad(c.Capacidad)
	if err != nil {
		return rechazar[dtoresult.Mesa](mapper.Validacion("capacidad_invalida", "la capacidad debe estar entre 1 y 500"))
	}
	px, ok := posicionOpcional(cmd.Campo[string]{Presente: c.PosicionX != nil, Valor: c.PosicionX})
	if !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_x_invalida", "la posición X debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	py, ok := posicionOpcional(cmd.Campo[string]{Presente: c.PosicionY != nil, Valor: c.PosicionY})
	if !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_y_invalida", "la posición Y debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	valida, err := zonaValida(ctx, uc.mesas, c.ZonaID, c.SucursalID)
	if err != nil {
		return tecnico[dtoresult.Mesa]("buscar zona", err)
	}
	if !valida {
		return rechazar[dtoresult.Mesa](zonaInvalida())
	}
	habilitada := true
	if c.Habilitada != nil {
		habilitada = *c.Habilitada
	}
	m, err := uc.mesas.Crear(ctx, outbound.NuevaMesa{
		SucursalID: c.SucursalID, ZonaID: c.ZonaID, NumeroMesa: numero.String(), Capacidad: capacidad.Int(),
		Habilitada: habilitada, PosicionX: px.Valor, PosicionY: py.Valor,
	})
	if err != nil {
		return deMesa(err, "crear mesa")
	}
	return appresult.Ok(m), nil
}

// mesaGestionable carga la mesa (bloqueándola) y verifica que el actor ve su sede (si no, 404) y
// que la administra (si no, 403). Devuelve la mesa o el rechazo; un error técnico va en err.
func mesaGestionable(ctx context.Context, store outbound.MesaStore, actor appctx.Actor, id int64) (dtoresult.Mesa, *apperror.AppError, error) {
	m, found, err := store.Obtener(ctx, id)
	if err != nil {
		return dtoresult.Mesa{}, nil, err
	}
	if !found {
		return dtoresult.Mesa{}, mapper.NoEncontrado("mesa"), nil
	}
	if ae := mapper.AutorizarSede(actor, m.SucursalID); ae != nil {
		return dtoresult.Mesa{}, ae, nil
	}
	return m, nil, nil
}

// EditarMesa modifica solo los campos enviados de una mesa activa. Habilitar o deshabilitar no
// pide motivo.
type EditarMesa struct{ mesas outbound.MesaStore }

func NewEditarMesa(m outbound.MesaStore) *EditarMesa { return &EditarMesa{m} }

func (uc *EditarMesa) Execute(ctx context.Context, c cmd.EditarMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	actual, ae, err := mesaGestionable(ctx, uc.mesas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Mesa]("buscar mesa", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoMesaInactiva, "la mesa está dada de baja: restáurela para editarla"))
	}
	if c.ZonaID == nil && c.NumeroMesa == nil && c.Capacidad == nil && c.Habilitada == nil &&
		!c.PosicionX.Presente && !c.PosicionY.Presente {
		return rechazar[dtoresult.Mesa](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	cambios := outbound.CambiosMesa{ZonaID: c.ZonaID, Habilitada: c.Habilitada}
	if c.NumeroMesa != nil {
		n, err := vo.NewNumeroMesa(*c.NumeroMesa)
		if err != nil {
			return rechazar[dtoresult.Mesa](mapper.Validacion("numero_mesa_invalido", "el número de mesa es obligatorio (máximo 20 caracteres)"))
		}
		v := n.String()
		cambios.NumeroMesa = &v
	}
	if c.Capacidad != nil {
		capac, err := vo.NewCapacidad(*c.Capacidad)
		if err != nil {
			return rechazar[dtoresult.Mesa](mapper.Validacion("capacidad_invalida", "la capacidad debe estar entre 1 y 500"))
		}
		v := capac.Int()
		cambios.Capacidad = &v
	}
	var ok bool
	if cambios.PosicionX, ok = posicionOpcional(c.PosicionX); !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_x_invalida", "la posición X debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	if cambios.PosicionY, ok = posicionOpcional(c.PosicionY); !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_y_invalida", "la posición Y debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	if c.ZonaID != nil {
		valida, err := zonaValida(ctx, uc.mesas, *c.ZonaID, actual.SucursalID)
		if err != nil {
			return tecnico[dtoresult.Mesa]("buscar zona", err)
		}
		if !valida {
			return rechazar[dtoresult.Mesa](zonaInvalida())
		}
	}
	m, found, err := uc.mesas.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deMesa(err, "editar mesa")
	}
	if !found {
		return rechazar[dtoresult.Mesa](mapper.NoEncontrado("mesa"))
	}
	return appresult.Ok(m), nil
}

// CambiarEstadoMesa da de baja (con motivo) o restaura una mesa. La base impide dar de baja una
// mesa con pedido abierto y restaurar una mesa cuya zona está dada de baja; ambos casos llegan
// como regla_de_datos con el mensaje de la base.
type CambiarEstadoMesa struct{ mesas outbound.MesaStore }

func NewCambiarEstadoMesa(m outbound.MesaStore) *CambiarEstadoMesa { return &CambiarEstadoMesa{m} }

func (uc *CambiarEstadoMesa) Execute(ctx context.Context, c cmd.CambiarEstadoMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	actual, ae, err := mesaGestionable(ctx, uc.mesas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Mesa]("buscar mesa", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	if c.Activo {
		if actual.EsActivo {
			return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoYaActiva, "la mesa ya está activa"))
		}
		m, found, err := uc.mesas.Restaurar(ctx, c.ID)
		if err != nil {
			return deMesa(err, "restaurar mesa")
		}
		if !found {
			return rechazar[dtoresult.Mesa](mapper.NoEncontrado("mesa"))
		}
		return appresult.Ok(m), nil
	}
	motivo, err := vo.TextoObligatorio(c.Motivo, maxMotivo)
	if err != nil {
		return rechazar[dtoresult.Mesa](mapper.Validacion("motivo_requerido", "la baja exige un motivo (máximo 2000 caracteres)"))
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la mesa ya está dada de baja"))
	}
	m, found, err := uc.mesas.DarBaja(ctx, c.ID, motivo)
	if err != nil {
		return deMesa(err, "dar de baja la mesa")
	}
	if !found {
		return rechazar[dtoresult.Mesa](mapper.NoEncontrado("mesa"))
	}
	return appresult.Ok(m), nil
}
