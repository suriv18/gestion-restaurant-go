package command

import (
	"context"
	"slices"
	"strings"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func dividirBase() cmd.DividirCuenta {
	return cmd.DividirCuenta{
		Intencion: cmd.Intencion{Clave: claveEjemplo},
		Actor:     actorPedidos(1, policy.PedidoCrear),
		PedidoID:  10, DetalleIDs: []int64{1},
	}
}

func TestDividirCuentaMueveLasLineasYSusAdicionales(t *testing.T) {
	f := fakeDivision()
	c := dividirBase()
	c.NombreCuenta = texto("  Juan  ")
	res, err := NewDividirCuenta(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía dividirse: %v %+v", err, res.Err())
	}
	if f.creaciones != 1 || f.creada.PedidoID != 10 || f.creada.Etiqueta == nil || *f.creada.Etiqueta != "Juan" {
		t.Fatalf("cuenta mal creada: %+v", f.creada)
	}
	if f.movimientos != 1 || f.destino != 21 || !slices.Equal(f.movidas, []int64{1, 3}) {
		t.Fatalf("el padre arrastra a su adicional: destino %d, movidas %v", f.destino, f.movidas)
	}
	if res.Value().ID != 21 {
		t.Fatalf("debe devolver la cuenta nueva: %+v", res.Value())
	}
}

func TestDividirCuentaSinNombreYDeVariasCuentas(t *testing.T) {
	f := fakeDivision()
	f.lineas[1].CuentaID = 22
	f.cuentas = append(f.cuentas, outbound.CuentaRef{ID: 22, PedidoID: 10, Estado: "ABIERTA"})
	c := dividirBase()
	c.DetalleIDs = []int64{2, 1}
	res, err := NewDividirCuenta(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() || f.creada.Etiqueta != nil || !slices.Equal(f.movidas, []int64{1, 2, 3}) {
		t.Fatalf("líneas de dos cuentas a una nueva: %v %+v %+v %v", err, res.Err(), f.creada, f.movidas)
	}
}

func TestDividirCuentaRechazos(t *testing.T) {
	otraSede := appctx.NewActor(7, "ana", 1, []appctx.Grant{
		{SucursalID: 1, Permiso: policy.PedidoVer}, {SucursalID: 2, Permiso: policy.PedidoCrear}})
	casos := []struct {
		nombre string
		mod    func(f *divisionFake, c *cmd.DividirCuenta)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *divisionFake, c *cmd.DividirCuenta) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *divisionFake, c *cmd.DividirCuenta) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"permiso solo en otra sede", func(_ *divisionFake, c *cmd.DividirCuenta) { c.Actor = otraSede }, apperror.KindForbidden, "sede_no_autorizada"},
		{"nombre demasiado largo", func(_ *divisionFake, c *cmd.DividirCuenta) { c.NombreCuenta = texto(strings.Repeat("x", 81)) }, apperror.KindValidation, "nombre_cuenta_invalido"},
		{"selección vacía", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = nil }, apperror.KindValidation, "seleccion_vacia"},
		{"pedido inexistente", func(f *divisionFake, _ *cmd.DividirCuenta) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *divisionFake, _ *cmd.DividirCuenta) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado", func(f *divisionFake, _ *cmd.DividirCuenta) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"línea repetida", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = []int64{2, 2} }, apperror.KindValidation, "detalle_duplicado"},
		{"línea que no es del pedido", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = []int64{99} }, apperror.KindNotFound, "no_encontrado"},
		{"línea anulada", func(f *divisionFake, c *cmd.DividirCuenta) {
			f.lineas[1].Estado = "ANULADO"
			c.DetalleIDs = []int64{2}
		}, apperror.KindConflict, "linea_anulada"},
		{"adicional sin su padre", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = []int64{3} }, apperror.KindValidation, "adicional_sin_padre"},
		{"cuenta de origen cerrada", func(f *divisionFake, _ *cmd.DividirCuenta) { f.cuentas[0].Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"cuenta de origen con pago confirmado", func(f *divisionFake, _ *cmd.DividirCuenta) { f.conPago = []int64{20} }, apperror.KindConflict, "cuenta_con_pago"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeDivision()
			in := dividirBase()
			tc.mod(f, &in)
			res, err := NewDividirCuenta(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.creaciones != 0 || f.movimientos != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}
