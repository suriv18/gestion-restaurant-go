package command

import (
	"context"
	"slices"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
)

// divisionFake implementa outbound.DivisionStore con respuestas fijadas por la prueba.
type divisionFake struct {
	pedido       outbound.PedidoRef
	pedidoExiste bool
	lineas       []outbound.LineaCuenta
	cuentas      []outbound.CuentaRef
	conPago      []int64
	nuevaID      int64

	creada      outbound.NuevaCuenta
	creaciones  int
	movidas     []int64
	destino     int64
	movimientos int
}

var _ outbound.DivisionStore = (*divisionFake)(nil)

func (f *divisionFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *divisionFake) LineasDelPedido(context.Context, int64) ([]outbound.LineaCuenta, error) {
	return f.lineas, nil
}
func (f *divisionFake) CuentasDelPedido(context.Context, int64) ([]outbound.CuentaRef, error) {
	return f.cuentas, nil
}
func (f *divisionFake) CuentasConPago(_ context.Context, ids []int64) ([]int64, error) {
	var out []int64
	for _, id := range ids {
		if slices.Contains(f.conPago, id) {
			out = append(out, id)
		}
	}
	return out, nil
}
func (f *divisionFake) CrearCuenta(_ context.Context, n outbound.NuevaCuenta) (int64, error) {
	f.creaciones++
	f.creada = n
	return f.nuevaID, nil
}
func (f *divisionFake) MoverLineas(_ context.Context, _, cuentaID int64, ids []int64) error {
	f.movimientos++
	f.destino = cuentaID
	f.movidas = ids
	return nil
}
func (f *divisionFake) CuentaConLineas(_ context.Context, _, cuentaID int64) (dtoresult.Cuenta, error) {
	return dtoresult.Cuenta{ID: cuentaID, Numero: 2}, nil
}

// fakeDivision es un pedido abierto con una cuenta y tres líneas: la 3 es un adicional de la 1.
func fakeDivision() *divisionFake {
	return &divisionFake{
		pedido:       outbound.PedidoRef{ID: 10, SucursalID: 1, Estado: "REGISTRADO"},
		pedidoExiste: true,
		lineas: []outbound.LineaCuenta{
			{ID: 1, CuentaID: 20, Estado: "ACTIVO"},
			{ID: 2, CuentaID: 20, Estado: "ACTIVO"},
			{ID: 3, CuentaID: 20, PadreID: idPtr(1), Estado: "ACTIVO"},
		},
		cuentas: []outbound.CuentaRef{{ID: 20, PedidoID: 10, Estado: "ABIERTA"}},
		nuevaID: 21,
	}
}
