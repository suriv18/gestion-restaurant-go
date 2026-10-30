package command

import (
	"context"
	"errors"
	"slices"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoDividir = "no tiene permiso para dividir cuentas"
	maxEtiquetaCuenta        = 80
)

func cuentaConPago() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "cuenta_con_pago",
		"la cuenta tiene pagos confirmados: sus consumos ya no se pueden cambiar")
}

func lineaAnulada() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "linea_anulada", "la línea está anulada")
}

// rechazoSeleccion traduce los rechazos de la política de selección; nil si el error no es uno.
func rechazoSeleccion(err error) *apperror.AppError {
	switch {
	case errors.Is(err, policy.ErrSeleccionVacia):
		return mapper.Validacion("seleccion_vacia", "indique al menos una línea para mover")
	case errors.Is(err, policy.ErrLineaRepetida):
		return mapper.Validacion("detalle_duplicado", "una línea se repite en la selección")
	case errors.Is(err, policy.ErrLineaInexistente):
		return mapper.NoEncontrado("línea")
	case errors.Is(err, policy.ErrLineaAnulada):
		return lineaAnulada()
	case errors.Is(err, policy.ErrAdicionalSinPadre):
		return mapper.Validacion("adicional_sin_padre", "un adicional viaja con la línea de la que cuelga: seleccione también esa línea")
	}
	return nil
}

// DividirCuenta mueve líneas activas a una cuenta nueva del mismo pedido (RES-RF-011). No toca lo
// enviado a cocina: las comandas cuelgan de la línea, no de la cuenta.
type DividirCuenta struct{ cuentas outbound.DivisionStore }

func NewDividirCuenta(c outbound.DivisionStore) *DividirCuenta { return &DividirCuenta{c} }

func (uc *DividirCuenta) Execute(ctx context.Context, c cmd.DividirCuenta) (appresult.Result[dtoresult.Cuenta], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoDividir, policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.Cuenta](ae)
	}
	etiqueta, ae := textoOpcional(c.NombreCuenta, maxEtiquetaCuenta, false, "nombre_cuenta_invalido",
		"el nombre de la cuenta admite hasta 80 caracteres en una línea")
	if ae != nil {
		return rechazar[dtoresult.Cuenta](ae)
	}
	if len(c.DetalleIDs) == 0 {
		return rechazar[dtoresult.Cuenta](rechazoSeleccion(policy.ErrSeleccionVacia))
	}

	_, ae, err := pedidoAutorizado(ctx, uc.cuentas, c.Actor, c.PedidoID, policy.PedidoCrear)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Cuenta](ae)
	}

	lineas, err := uc.cuentas.LineasDelPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("leer las líneas del pedido", err)
	}
	vista := make([]policy.LineaPedido, len(lineas))
	cuentaDe := make(map[int64]int64, len(lineas))
	for i, l := range lineas {
		vista[i] = policy.LineaPedido{ID: l.ID, PadreID: l.PadreID, CuentaID: l.CuentaID, Activa: l.Estado == estadoActivo}
		cuentaDe[l.ID] = l.CuentaID
	}
	movidas, err := policy.SeleccionDeDivision(c.DetalleIDs, vista)
	if err != nil {
		if ae := rechazoSeleccion(err); ae != nil {
			return rechazar[dtoresult.Cuenta](ae)
		}
		return tecnico[dtoresult.Cuenta]("resolver la selección", err)
	}

	cuentas, err := uc.cuentas.CuentasDelPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("leer las cuentas del pedido", err)
	}
	estado := make(map[int64]string, len(cuentas))
	for _, cu := range cuentas {
		estado[cu.ID] = cu.Estado
	}
	var origen []int64
	for _, id := range movidas {
		if cu := cuentaDe[id]; !slices.Contains(origen, cu) {
			origen = append(origen, cu)
		}
	}
	slices.Sort(origen)
	for _, cu := range origen {
		if estado[cu] != estadoCuentaAbta {
			return rechazar[dtoresult.Cuenta](cuentaCerrada())
		}
	}
	conPago, err := uc.cuentas.CuentasConPago(ctx, origen)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("buscar pagos de las cuentas", err)
	}
	if len(conPago) > 0 {
		return rechazar[dtoresult.Cuenta](cuentaConPago())
	}

	id, err := uc.cuentas.CrearCuenta(ctx, outbound.NuevaCuenta{PedidoID: c.PedidoID, Etiqueta: etiqueta})
	if err != nil {
		return deAlmacen[dtoresult.Cuenta]("crear la cuenta", err)
	}
	if err := uc.cuentas.MoverLineas(ctx, c.PedidoID, id, movidas); err != nil {
		return deAlmacen[dtoresult.Cuenta]("mover las líneas", err)
	}
	cuenta, err := uc.cuentas.CuentaConLineas(ctx, c.PedidoID, id)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("leer la cuenta nueva", err)
	}
	return appresult.Ok(cuenta), nil
}
