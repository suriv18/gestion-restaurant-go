package command

import (
	"context"

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
	estadoActivo     = "ACTIVO"
	estadoCuentaAbta = "ABIERTA"
	tipoAdicional    = "ADICIONAL"
	afectacionIGV    = "10"

	mensajeSinPermisoDescuento = "no tiene permiso para aplicar descuentos"
)

func sinPermisoDescuento() *apperror.AppError {
	return apperror.New(apperror.KindForbidden, mapper.CodigoSinPermiso, mensajeSinPermisoDescuento)
}

func descuentoExcede() *apperror.AppError {
	return mapper.Validacion("descuento_excede_importe", "el descuento no puede superar el importe de la línea")
}

func pedidoCerrado() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "pedido_cerrado", "el pedido ya está cerrado o cancelado")
}

func cuentaCerrada() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "cuenta_cerrada", "la cuenta ya no está abierta")
}

// pedidoEditable bloquea el pedido y comprueba que el actor pueda editarlo en su sede y que siga
// abierto. El rechazo esperado va en el segundo retorno; el fallo técnico, en el tercero.
func pedidoEditable(ctx context.Context, store outbound.PedidoStore, actor appctx.Actor, pedidoID int64) (outbound.PedidoRef, *apperror.AppError, error) {
	ped, found, err := store.BloquearPedido(ctx, pedidoID)
	if err != nil {
		return outbound.PedidoRef{}, nil, err
	}
	if !found {
		return outbound.PedidoRef{}, mapper.NoEncontradoM("pedido"), nil
	}
	if ae := mapper.AutorizarPedidos(actor, ped.SucursalID, policy.PedidoEditar, mapper.NoEncontradoM("pedido")); ae != nil {
		return outbound.PedidoRef{}, ae, nil
	}
	if ped.Cerrado {
		return outbound.PedidoRef{}, pedidoCerrado(), nil
	}
	return ped, nil, nil
}

// AgregarDetalle agrega una línea a una cuenta abierta fijando descripción, precio y tasa desde
// la carta en este instante (RES-RF-010).
type AgregarDetalle struct{ pedidos outbound.PedidoStore }

func NewAgregarDetalle(p outbound.PedidoStore) *AgregarDetalle { return &AgregarDetalle{p} }

func (uc *AgregarDetalle) Execute(ctx context.Context, c cmd.AgregarDetalle) (appresult.Result[dtoresult.Linea], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoPedidos, policy.PedidoEditar); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	cantidad, ae := validarCantidad(c.Cantidad)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	descuento, ae := validarDescuento(c.DescuentoMonto)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	instrucciones, ae := validarInstrucciones(c.Instrucciones)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}

	ped, ae, err := pedidoEditable(ctx, uc.pedidos, c.Actor, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if !descuento.EsCero() && !c.Actor.Can(policy.DescuentoAutorizar, ped.SucursalID) {
		return rechazar[dtoresult.Linea](sinPermisoDescuento())
	}

	cuenta, found, err := uc.pedidos.CuentaDelPedido(ctx, c.PedidoID, c.CuentaID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar cuenta", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.Validacion("cuenta_invalida", "la cuenta no existe o no pertenece al pedido"))
	}
	if cuenta.Estado != estadoCuentaAbta {
		return rechazar[dtoresult.Linea](cuentaCerrada())
	}

	prod, found, err := uc.pedidos.ProductoParaVenta(ctx, c.ProductoID, ped.SucursalID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar producto", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.Validacion("producto_invalido", "el producto no existe"))
	}
	if !prod.EsActivo {
		return rechazar[dtoresult.Linea](mapper.Validacion("producto_inactivo", "el producto está dado de baja"))
	}
	if !prod.Disponible {
		return rechazar[dtoresult.Linea](mapper.Validacion("producto_no_disponible", "el producto no está disponible en esta sucursal"))
	}
	tasa := "0"
	if prod.AfectacionIGV == afectacionIGV {
		if prod.Tasa == "" {
			return rechazar[dtoresult.Linea](mapper.Validacion("tasa_no_vigente", "el impuesto del producto no tiene una tasa vigente"))
		}
		tasa = prod.Tasa
	}

	ae, err = uc.validarPadre(ctx, c, prod.Tipo)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar línea padre", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if !vo.DescuentoCabe(cantidad, prod.Precio, descuento) {
		return rechazar[dtoresult.Linea](descuentoExcede())
	}

	l, err := uc.pedidos.InsertarDetalle(ctx, outbound.NuevoDetalle{
		PedidoID: c.PedidoID, CuentaID: c.CuentaID, ProductoID: prod.ID, DetallePadreID: c.DetallePadreID,
		Descripcion: prod.Nombre, Cantidad: cantidad.String(), PrecioUnitario: prod.Precio,
		DescuentoTotal: descuento.String(), AfectacionIGV: prod.AfectacionIGV, TasaImpuesto: tasa,
		NotasPreparacion: instrucciones,
	})
	if err != nil {
		return deAlmacen[dtoresult.Linea]("agregar detalle", err)
	}
	return appresult.Ok(l), nil
}

// validarPadre aplica la regla de los adicionales: un ADICIONAL cuelga de una línea activa de la
// misma cuenta, y solo un ADICIONAL puede colgar de otra línea.
func (uc *AgregarDetalle) validarPadre(ctx context.Context, c cmd.AgregarDetalle, tipoProducto string) (*apperror.AppError, error) {
	esAdicional := tipoProducto == tipoAdicional
	switch {
	case c.DetallePadreID == nil && esAdicional:
		return mapper.Validacion("adicional_sin_padre", "un adicional debe indicar la línea de la que cuelga"), nil
	case c.DetallePadreID != nil && !esAdicional:
		return mapper.Validacion("padre_no_aplica", "solo un adicional puede colgar de otra línea"), nil
	case c.DetallePadreID == nil:
		return nil, nil
	}
	padre, found, err := uc.pedidos.DetalleDelPedido(ctx, c.PedidoID, *c.DetallePadreID)
	if err != nil {
		return nil, err
	}
	if !found || padre.CuentaID != c.CuentaID || padre.Estado != estadoActivo {
		return mapper.Validacion("padre_invalido", "la línea padre no existe, está anulada o es de otra cuenta"), nil
	}
	return nil, nil
}

// EditarDetalle modifica cantidad, instrucciones o descuento de una línea activa de una cuenta
// abierta, mientras no se haya enviado a cocina.
type EditarDetalle struct{ pedidos outbound.PedidoStore }

func NewEditarDetalle(p outbound.PedidoStore) *EditarDetalle { return &EditarDetalle{p} }

func (uc *EditarDetalle) Execute(ctx context.Context, c cmd.EditarDetalle) (appresult.Result[dtoresult.Linea], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoPedidos, policy.PedidoEditar); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if c.Cantidad == nil && c.DescuentoMonto == nil && !c.Instrucciones.Presente {
		return rechazar[dtoresult.Linea](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosDetalle
	var nuevaCantidad *vo.Cantidad
	var nuevoDescuento *vo.Dinero
	if c.Cantidad != nil {
		q, ae := validarCantidad(*c.Cantidad)
		if ae != nil {
			return rechazar[dtoresult.Linea](ae)
		}
		s := q.String()
		cambios.Cantidad, nuevaCantidad = &s, &q
	}
	if c.DescuentoMonto != nil {
		d, ae := validarDescuento(c.DescuentoMonto)
		if ae != nil {
			return rechazar[dtoresult.Linea](ae)
		}
		s := d.String()
		cambios.DescuentoTotal, nuevoDescuento = &s, &d
	}
	if c.Instrucciones.Presente {
		v, ae := validarInstrucciones(c.Instrucciones.Valor)
		if ae != nil {
			return rechazar[dtoresult.Linea](ae)
		}
		cambios.Instrucciones = outbound.CampoOpcional{Presente: true, Valor: v}
	}

	ped, ae, err := pedidoEditable(ctx, uc.pedidos, c.Actor, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if nuevoDescuento != nil && !nuevoDescuento.EsCero() && !c.Actor.Can(policy.DescuentoAutorizar, ped.SucursalID) {
		return rechazar[dtoresult.Linea](sinPermisoDescuento())
	}

	det, found, err := uc.pedidos.DetalleDelPedido(ctx, c.PedidoID, c.DetalleID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar línea", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.NoEncontrado("línea"))
	}
	if det.Estado != estadoActivo {
		return rechazar[dtoresult.Linea](apperror.New(apperror.KindConflict, "linea_anulada", "la línea está anulada"))
	}
	if det.EnComanda {
		return rechazar[dtoresult.Linea](apperror.New(apperror.KindConflict, "linea_en_cocina", "la línea ya se envió a cocina"))
	}
	cuenta, found, err := uc.pedidos.CuentaDelPedido(ctx, c.PedidoID, det.CuentaID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar cuenta", err)
	}
	if !found || cuenta.Estado != estadoCuentaAbta {
		return rechazar[dtoresult.Linea](cuentaCerrada())
	}

	// El descuento resultante (nuevo o el que ya tenía) debe caber en el importe resultante.
	if nuevaCantidad == nil {
		q, err := vo.NewCantidad(det.Cantidad)
		if err != nil {
			return tecnico[dtoresult.Linea]("leer la cantidad de la línea", err)
		}
		nuevaCantidad = &q
	}
	if nuevoDescuento == nil {
		d, err := vo.NewDinero(det.DescuentoTotal)
		if err != nil {
			return tecnico[dtoresult.Linea]("leer el descuento de la línea", err)
		}
		nuevoDescuento = &d
	}
	if !vo.DescuentoCabe(*nuevaCantidad, det.PrecioUnitario, *nuevoDescuento) {
		return rechazar[dtoresult.Linea](descuentoExcede())
	}

	l, err := uc.pedidos.ActualizarDetalle(ctx, c.DetalleID, cambios)
	if err != nil {
		return deAlmacen[dtoresult.Linea]("editar detalle", err)
	}
	return appresult.Ok(l), nil
}
