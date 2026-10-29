package command

import (
	"context"
	"errors"
	"slices"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	"service-restaurant/internal/modules/produccion/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoEnvio = "no tiene permiso para enviar comandas"
	maxLineasEnvio         = 100

	estadoActivo        = "ACTIVO"
	estadoCuentaAbierta = "ABIERTA"

	CodigoClaveEnUso = "clave_en_uso"
)

func pedidoCerrado() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "pedido_cerrado", "el pedido ya está cerrado o cancelado")
}

func cuentaCerrada() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "cuenta_cerrada", "la cuenta ya no está abierta")
}

// deComanda traduce la clave repetida a un rechazo específico y delega lo demás.
func deComanda(err error, contexto string) (appresult.Result[dtoresult.Comanda], error) {
	if errors.Is(err, outbound.ErrClaveEnUso) {
		return rechazar[dtoresult.Comanda](apperror.New(apperror.KindConflict, CodigoClaveEnUso, "la clave de idempotencia ya fue usada por otra comanda"))
	}
	return deAlmacen[dtoresult.Comanda](contexto, err)
}

// validarLineas comprueba el cuerpo del envío y devuelve, por id de línea, la cantidad pedida y los
// ids ordenados de menor a mayor (el orden de bloqueo).
func validarLineas(lineas []cmd.LineaEnvio) (map[int64]vo.Cantidad, []int64, *apperror.AppError) {
	if len(lineas) == 0 || len(lineas) > maxLineasEnvio {
		return nil, nil, mapper.Validacion("lineas_invalidas", "el envío debe llevar entre 1 y 100 líneas")
	}
	pedidas := make(map[int64]vo.Cantidad, len(lineas))
	ids := make([]int64, 0, len(lineas))
	for _, l := range lineas {
		if l.DetallePedidoID <= 0 {
			return nil, nil, mapper.Validacion("linea_invalida", "el id de una línea no es válido")
		}
		if _, repetida := pedidas[l.DetallePedidoID]; repetida {
			return nil, nil, mapper.Validacion("linea_duplicada", "una línea aparece más de una vez en el envío")
		}
		q, err := vo.NewCantidad(l.Cantidad)
		if err != nil {
			return nil, nil, mapper.Validacion("cantidad_invalida", "la cantidad debe ser mayor que cero y tener hasta 3 decimales")
		}
		pedidas[l.DetallePedidoID] = q
		ids = append(ids, l.DetallePedidoID)
	}
	slices.Sort(ids)
	return pedidas, ids, nil
}

// EnviarComanda envía a una estación las cantidades pendientes de varias líneas de un pedido
// (RES-RF-012). Reenviar con la misma Idempotency-Key devuelve la comanda original: lo resuelve el
// decorador de idempotencia, no este caso de uso.
type EnviarComanda struct{ store outbound.EnvioStore }

func NewEnviarComanda(s outbound.EnvioStore) *EnviarComanda { return &EnviarComanda{s} }

func (uc *EnviarComanda) Execute(ctx context.Context, c cmd.EnviarComanda) (appresult.Result[dtoresult.Comanda], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEnvio, policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.Comanda](ae)
	}
	pedidas, ids, ae := validarLineas(c.Lineas)
	if ae != nil {
		return rechazar[dtoresult.Comanda](ae)
	}

	// Regla de R1: el pedido se bloquea FOR UPDATE antes que cualquier línea.
	ped, found, err := uc.store.BloquearPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Comanda]("buscar pedido", err)
	}
	if !found {
		return rechazar[dtoresult.Comanda](mapper.NoEncontradoM("pedido"))
	}
	if ae := mapper.Autorizar(c.Actor, ped.SucursalID, mapper.NoEncontradoM("pedido"), policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.Comanda](ae)
	}
	if ped.Cerrado {
		return rechazar[dtoresult.Comanda](pedidoCerrado())
	}

	est, found, err := uc.store.Estacion(ctx, c.EstacionID)
	if err != nil {
		return tecnico[dtoresult.Comanda]("buscar estación", err)
	}
	if !found || est.SucursalID != ped.SucursalID || !est.EsActiva {
		return rechazar[dtoresult.Comanda](mapper.Validacion("estacion_invalida", "la estación no existe, está dada de baja o es de otra sucursal"))
	}

	lineas, err := uc.store.LineasParaEnvio(ctx, c.PedidoID, ped.SucursalID, ids)
	if err != nil {
		return tecnico[dtoresult.Comanda]("buscar líneas", err)
	}
	if len(lineas) != len(ids) {
		return rechazar[dtoresult.Comanda](mapper.Validacion("linea_invalida", "alguna línea no existe o no pertenece al pedido"))
	}

	nuevas := make([]outbound.NuevaComandaLinea, 0, len(lineas))
	for _, l := range lineas {
		if l.Estado != estadoActivo {
			return rechazar[dtoresult.Comanda](apperror.New(apperror.KindConflict, "linea_anulada", "una línea está anulada"))
		}
		if l.CuentaEstado != estadoCuentaAbierta {
			return rechazar[dtoresult.Comanda](cuentaCerrada())
		}
		if l.EstacionID == nil || *l.EstacionID != est.ID {
			return rechazar[dtoresult.Comanda](mapper.Validacion("estacion_no_corresponde", "el producto de una línea no se prepara en esta estación"))
		}
		total, err := vo.CantidadDeAlmacen(l.Cantidad)
		if err != nil {
			return tecnico[dtoresult.Comanda]("leer la cantidad de la línea", err)
		}
		enviado, err := vo.CantidadDeAlmacen(l.Enviado)
		if err != nil {
			return tecnico[dtoresult.Comanda]("leer la cantidad enviada", err)
		}
		cantidad := pedidas[l.ID]
		if cantidad.Excede(policy.Pendiente(total, enviado)) {
			return rechazar[dtoresult.Comanda](apperror.New(apperror.KindConflict, "cantidad_excede_pendiente",
				"la cantidad supera lo que queda pendiente de enviar de una línea"))
		}
		nuevas = append(nuevas, outbound.NuevaComandaLinea{
			DetallePedidoID: l.ID, Cantidad: cantidad.String(), Descripcion: l.Descripcion, Instrucciones: l.Instrucciones,
		})
	}

	com, err := uc.store.CrearComanda(ctx, outbound.NuevaComanda{
		PedidoID: c.PedidoID, SucursalID: ped.SucursalID, EstacionID: est.ID, ClaveIdempotencia: c.Intencion.Clave,
		EnviadoPor: c.Actor.UserID, Lineas: nuevas,
	})
	if err != nil {
		return deComanda(err, "crear comanda")
	}
	return appresult.Ok(com), nil
}
