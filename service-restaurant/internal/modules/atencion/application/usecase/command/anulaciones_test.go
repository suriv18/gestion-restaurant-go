package command

import (
	"context"
	"strings"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func supervisor() appctx.Actor { return actorPedidos(1, policy.PedidoVer, policy.PedidoAnular) }

// anularEnOtraSede: ve pedidos en la sede 1 pero anula solo en la 2.
func anularEnOtraSede() appctx.Actor {
	return appctx.NewActor(7, "ana", 1, []appctx.Grant{
		{SucursalID: 1, Permiso: policy.PedidoVer}, {SucursalID: 2, Permiso: policy.PedidoAnular}})
}

func anularBase() cmd.AnularDetalle {
	return cmd.AnularDetalle{
		Intencion: cmd.Intencion{Clave: claveEjemplo}, Actor: supervisor(),
		PedidoID: 10, DetalleID: 55, Motivo: "  el cliente se fue  ",
	}
}

func cancelarBase() cmd.CancelarPedido {
	return cmd.CancelarPedido{
		Intencion: cmd.Intencion{Clave: claveEjemplo}, Actor: supervisor(), PedidoID: 10, Motivo: "  cliente se retiró  ",
	}
}

func TestAnularDetalle(t *testing.T) {
	f := fakeAnulacion()
	res, err := NewAnularDetalle(f).Execute(context.Background(), anularBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía anularse: %v %+v", err, res.Err())
	}
	if f.anulaciones != 1 || f.anulada != 55 || f.motivo.Motivo != "el cliente se fue" || f.motivo.UsuarioID != 7 {
		t.Fatalf("anulación mal armada: %+v", f)
	}
	if res.Value().ID != 55 || res.Value().Estado != "ANULADO" {
		t.Fatalf("debe devolver la línea anulada: %+v", res.Value())
	}
}

func TestAnularDetalleRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *anulacionFake, c *cmd.AnularDetalle)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"sin permiso de anular", func(_ *anulacionFake, c *cmd.AnularDetalle) {
			c.Actor = actorPedidos(1, policy.PedidoVer, policy.PedidoEditar)
		}, apperror.KindForbidden, "sin_permiso"},
		{"permiso solo en otra sede", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Actor = anularEnOtraSede() }, apperror.KindForbidden, "sede_no_autorizada"},
		{"motivo vacío", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Motivo = "   " }, apperror.KindValidation, "motivo_invalido"},
		{"motivo demasiado largo", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Motivo = strings.Repeat("x", 2001) }, apperror.KindValidation, "motivo_invalido"},
		{"pedido inexistente", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"línea inexistente", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.detalleExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"línea ya anulada", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.detalle.Estado = "ANULADO" }, apperror.KindConflict, "linea_anulada"},
		{"cuenta cerrada", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.cuenta.Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"cuenta con pago confirmado", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.conPago = []int64{20} }, apperror.KindConflict, "cuenta_con_pago"},
		{"línea ya entregada", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.entregadaLinea = true }, apperror.KindConflict, "linea_entregada"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeAnulacion()
			in := anularBase()
			tc.mod(f, &in)
			res, err := NewAnularDetalle(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.anulaciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}

func TestCancelarPedido(t *testing.T) {
	f := fakeAnulacion()
	res, err := NewCancelarPedido(f).Execute(context.Background(), cancelarBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía cancelarse: %v %+v", err, res.Err())
	}
	if f.cancelaciones != 1 || f.motivo.Motivo != "cliente se retiró" || f.motivo.UsuarioID != 7 {
		t.Fatalf("cancelación mal armada: %+v", f)
	}
	if res.Value().ID != 10 || res.Value().Estado != "CANCELADO" {
		t.Fatalf("debe devolver el pedido cancelado: %+v", res.Value())
	}
}

func TestCancelarPedidoRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *anulacionFake, c *cmd.CancelarPedido)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *anulacionFake, c *cmd.CancelarPedido) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"sin permiso de anular", func(_ *anulacionFake, c *cmd.CancelarPedido) {
			c.Actor = actorPedidos(1, policy.PedidoVer, policy.PedidoEditar)
		}, apperror.KindForbidden, "sin_permiso"},
		{"permiso solo en otra sede", func(_ *anulacionFake, c *cmd.CancelarPedido) { c.Actor = anularEnOtraSede() }, apperror.KindForbidden, "sede_no_autorizada"},
		{"motivo vacío", func(_ *anulacionFake, c *cmd.CancelarPedido) { c.Motivo = "" }, apperror.KindValidation, "motivo_invalido"},
		{"pedido inexistente", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido ya cerrado o cancelado", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"alguna cuenta con pago confirmado", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.conPago = []int64{21} }, apperror.KindConflict, "pedido_con_pago"},
		{"alguna línea ya entregada", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.entregadaPedido = true }, apperror.KindConflict, "linea_entregada"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeAnulacion()
			in := cancelarBase()
			tc.mod(f, &in)
			res, err := NewCancelarPedido(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.cancelaciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}
