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
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoPedidos = "no tiene permiso para gestionar pedidos"

	CodigoMesaOcupada = "mesa_ocupada"
	CodigoClaveEnUso  = "clave_en_uso"
)

// dePedido traduce la mesa ocupada y la clave repetida a rechazos específicos y delega lo demás.
func dePedido(err error, contexto string) (appresult.Result[dtoresult.PedidoDetalle], error) {
	switch {
	case errors.Is(err, outbound.ErrMesaOcupada):
		return rechazar[dtoresult.PedidoDetalle](apperror.New(apperror.KindConflict, CodigoMesaOcupada, "la mesa ya tiene un pedido abierto"))
	case errors.Is(err, outbound.ErrClaveEnUso):
		return rechazar[dtoresult.PedidoDetalle](apperror.New(apperror.KindConflict, CodigoClaveEnUso, "la clave de idempotencia ya fue usada por otro pedido"))
	}
	return deAlmacen[dtoresult.PedidoDetalle](contexto, err)
}

// AbrirPedido crea el pedido y su cuenta nº 1 en una sola transacción (RES-RF-009).
type AbrirPedido struct{ pedidos outbound.PedidoStore }

func NewAbrirPedido(p outbound.PedidoStore) *AbrirPedido { return &AbrirPedido{p} }

func (uc *AbrirPedido) Execute(ctx context.Context, c cmd.AbrirPedido) (appresult.Result[dtoresult.PedidoDetalle], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoPedidos, policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	if ae := mapper.AutorizarPedidos(c.Actor, c.SucursalID, policy.PedidoCrear, mapper.NoEncontrado("sucursal")); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	tipo, ae := validarTipoPedido(c.Tipo)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	comensales, ae := validarComensales(c.Comensales)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	notas, ae := validarNotas(c.Notas)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	if ae := validarMesaSegunTipo(tipo, c.MesaID); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	reparto, ae := validarReparto(tipo, c.Reparto)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}

	if tipo == vo.TipoSalon {
		mesa, found, err := uc.pedidos.MesaParaPedido(ctx, *c.MesaID)
		if err != nil {
			return tecnico[dtoresult.PedidoDetalle]("buscar mesa", err)
		}
		if !found || mesa.SucursalID != c.SucursalID {
			return rechazar[dtoresult.PedidoDetalle](mapper.Validacion("mesa_invalida", "la mesa no existe o pertenece a otra sucursal"))
		}
		if !mesa.EsActiva || !mesa.Habilitada {
			return rechazar[dtoresult.PedidoDetalle](mapper.Validacion("mesa_no_disponible", "la mesa está dada de baja o no está habilitada"))
		}
		if comensales.Int() > mesa.Capacidad {
			return rechazar[dtoresult.PedidoDetalle](mapper.Validacion("capacidad_excedida", "los comensales superan la capacidad de la mesa"))
		}
	}

	mesero, err := uc.pedidos.EmpleadoDeUsuario(ctx, c.Actor.UserID)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar empleado del usuario", err)
	}
	p, err := uc.pedidos.CrearPedido(ctx, outbound.NuevoPedido{
		SucursalID: c.SucursalID, ClaveIdempotencia: c.Intencion.Clave, Tipo: tipo.String(), MesaID: c.MesaID,
		MeseroID: mesero, ClienteID: c.ClienteID, Comensales: comensales.Int(), Notas: notas, Reparto: reparto,
	})
	if err != nil {
		return dePedido(err, "abrir pedido")
	}
	return appresult.Ok(p), nil
}
