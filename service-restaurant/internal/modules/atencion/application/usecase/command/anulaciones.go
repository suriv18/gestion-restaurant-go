package command

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	"service-restaurant/internal/modules/atencion/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoAnular = "no tiene permiso para anular pedidos"

// validarMotivo recorta el motivo y exige 1..2000 caracteres.
func validarMotivo(raw string) (string, *apperror.AppError) {
	m, err := vo.TextoObligatorio(raw, maxMotivo)
	if err != nil {
		return "", mapper.Validacion("motivo_invalido", "el motivo es obligatorio (máximo 2000 caracteres)")
	}
	return m, nil
}

func pedidoConPago() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "pedido_con_pago",
		"el pedido tiene pagos confirmados: el consumo cobrado no se anula, se registra una devolución")
}

func lineaEntregada(mensaje string) *apperror.AppError {
	return apperror.New(apperror.KindConflict, "linea_entregada", mensaje)
}

// AnularDetalle anula una línea activa y sus adicionales, y cancela en cocina lo que aún no se
// entregó (RES-RF-014). La línea se conserva con su motivo y su auditoría.
type AnularDetalle struct{ pedidos outbound.AnulacionStore }

func NewAnularDetalle(p outbound.AnulacionStore) *AnularDetalle { return &AnularDetalle{p} }

func (uc *AnularDetalle) Execute(ctx context.Context, c cmd.AnularDetalle) (appresult.Result[dtoresult.Linea], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAnular, policy.PedidoAnular); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	motivo, ae := validarMotivo(c.Motivo)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	_, ae, err := pedidoAutorizado(ctx, uc.pedidos, c.Actor, c.PedidoID, policy.PedidoAnular)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}

	det, found, err := uc.pedidos.DetalleDelPedido(ctx, c.PedidoID, c.DetalleID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar línea", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.NoEncontrado("línea"))
	}
	if det.Estado != estadoActivo {
		return rechazar[dtoresult.Linea](lineaAnulada())
	}
	cuenta, found, err := uc.pedidos.CuentaDelPedido(ctx, c.PedidoID, det.CuentaID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar cuenta", err)
	}
	if !found || cuenta.Estado != estadoCuentaAbta {
		return rechazar[dtoresult.Linea](cuentaCerrada())
	}
	conPago, err := uc.pedidos.CuentasConPago(ctx, []int64{det.CuentaID})
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pagos de la cuenta", err)
	}
	if len(conPago) > 0 {
		return rechazar[dtoresult.Linea](cuentaConPago())
	}
	entregada, err := uc.pedidos.CocinaEntregadaEnLinea(ctx, c.PedidoID, c.DetalleID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar entregas de la línea", err)
	}
	if entregada {
		return rechazar[dtoresult.Linea](lineaEntregada("la línea ya se entregó a la mesa: lo servido no se anula"))
	}

	l, err := uc.pedidos.AnularLinea(ctx, c.PedidoID, c.DetalleID, outbound.Anulacion{Motivo: motivo, UsuarioID: c.Actor.UserID})
	if err != nil {
		return deAlmacen[dtoresult.Linea]("anular línea", err)
	}
	return appresult.Ok(l), nil
}

// CancelarPedido cancela el pedido entero: cuentas abiertas, líneas activas y lo que cocina aún no
// entregó, con un solo motivo. El consumo ya cobrado no se devuelve aquí (RES-RF-014).
type CancelarPedido struct{ pedidos outbound.AnulacionStore }

func NewCancelarPedido(p outbound.AnulacionStore) *CancelarPedido { return &CancelarPedido{p} }

func (uc *CancelarPedido) Execute(ctx context.Context, c cmd.CancelarPedido) (appresult.Result[dtoresult.PedidoDetalle], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAnular, policy.PedidoAnular); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	motivo, ae := validarMotivo(c.Motivo)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	_, ae, err := pedidoAutorizado(ctx, uc.pedidos, c.Actor, c.PedidoID, policy.PedidoAnular)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}

	cuentas, err := uc.pedidos.CuentasDelPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("leer las cuentas del pedido", err)
	}
	ids := make([]int64, len(cuentas))
	for i, cu := range cuentas {
		ids[i] = cu.ID
	}
	conPago, err := uc.pedidos.CuentasConPago(ctx, ids)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar pagos del pedido", err)
	}
	if len(conPago) > 0 {
		return rechazar[dtoresult.PedidoDetalle](pedidoConPago())
	}
	entregada, err := uc.pedidos.CocinaEntregadaEnPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar entregas del pedido", err)
	}
	if entregada {
		return rechazar[dtoresult.PedidoDetalle](lineaEntregada("el pedido tiene líneas ya entregadas a la mesa: lo servido no se anula"))
	}

	d, err := uc.pedidos.CancelarPedido(ctx, c.PedidoID, outbound.Anulacion{Motivo: motivo, UsuarioID: c.Actor.UserID})
	if err != nil {
		return deAlmacen[dtoresult.PedidoDetalle]("cancelar pedido", err)
	}
	return appresult.Ok(d), nil
}
