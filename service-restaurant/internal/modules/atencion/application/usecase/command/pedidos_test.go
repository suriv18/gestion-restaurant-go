package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

const claveEjemplo = "6f1c0a52-3c1e-4a53-9a39-0b9a3a2f1a11"

func fakeApertura() *pedidoStoreFake {
	return &pedidoStoreFake{
		mesa:       outbound.MesaRef{SucursalID: 1, Capacidad: 4, Habilitada: true, EsActiva: true},
		mesaExiste: true,
	}
}

func abrirBase() cmd.AbrirPedido {
	return cmd.AbrirPedido{
		Intencion:  cmd.Intencion{Clave: claveEjemplo},
		Actor:      actorPedidos(1, policy.PedidoCrear),
		SucursalID: 1, Tipo: "SALON", MesaID: idPtr(7), Comensales: 2,
	}
}

func TestAbrirPedidoSalon(t *testing.T) {
	f := fakeApertura()
	f.empleado = idPtr(41)
	c := abrirBase()
	c.Notas = texto("  sin cebolla  ")
	c.ClienteID = idPtr(12)

	res, err := NewAbrirPedido(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía abrirse: %v %+v", err, res.Err())
	}
	n := f.creado
	if n.SucursalID != 1 || n.ClaveIdempotencia != claveEjemplo || n.Tipo != "SALON" || n.MesaID == nil || *n.MesaID != 7 ||
		n.Comensales != 2 || n.ClienteID == nil || *n.ClienteID != 12 || n.MeseroID == nil || *n.MeseroID != 41 ||
		n.Notas == nil || *n.Notas != "sin cebolla" || n.Reparto != nil {
		t.Fatalf("pedido mal armado: %+v", n)
	}
	if res.Value().ID != 5 {
		t.Fatalf("debe devolver el pedido creado: %+v", res.Value())
	}
}

func TestAbrirPedidoLlevarYDelivery(t *testing.T) {
	f := fakeApertura()
	llevar := abrirBase()
	llevar.Tipo, llevar.MesaID = "LLEVAR", nil
	res, err := NewAbrirPedido(f).Execute(context.Background(), llevar)
	if err != nil || !res.IsOk() || f.creado.MesaID != nil || f.creado.Reparto != nil || f.creado.MeseroID != nil {
		t.Fatalf("LLEVAR: %v %+v %+v", err, res.Err(), f.creado)
	}

	delivery := abrirBase()
	delivery.Tipo, delivery.MesaID = "DELIVERY", nil
	delivery.Reparto = &cmd.Reparto{Destinatario: " Ana Pérez ", Telefono: "999111222", Direccion: " Av. Lima 123 ", Canal: texto("PEDIDOSYA")}
	res, err = NewAbrirPedido(f).Execute(context.Background(), delivery)
	if err != nil || !res.IsOk() {
		t.Fatalf("DELIVERY: %v %+v", err, res.Err())
	}
	r := f.creado.Reparto
	if r == nil || r.Destinatario != "Ana Pérez" || r.Telefono != "999111222" || r.Direccion != "Av. Lima 123" ||
		r.Referencia != nil || r.Canal == nil || *r.Canal != "PEDIDOSYA" {
		t.Fatalf("reparto mal armado: %+v", r)
	}
}

func TestAbrirPedidoRechazos(t *testing.T) {
	deliveryOk := &cmd.Reparto{Destinatario: "Ana", Telefono: "999", Direccion: "Av. Lima 1"}
	casos := []struct {
		nombre string
		mod    func(f *pedidoStoreFake, c *cmd.AbrirPedido)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"sede ajena", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.SucursalID = 9 }, apperror.KindNotFound, "no_encontrado"},
		{"la sede se ve pero no se crea en ella", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Actor = appctx.NewActor(7, "ana", 1, []appctx.Grant{
				{SucursalID: 1, Permiso: policy.PedidoVer}, {SucursalID: 2, Permiso: policy.PedidoCrear}})
		}, apperror.KindForbidden, "sede_no_autorizada"},
		{"tipo desconocido", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Tipo = "MOSTRADOR" }, apperror.KindValidation, "tipo_invalido"},
		{"comensales cero", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Comensales = 0 }, apperror.KindValidation, "comensales_invalidos"},
		{"notas con caracteres de control", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Notas = texto("a\x00b") }, apperror.KindValidation, "notas_invalidas"},
		{"salón sin mesa", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.MesaID = nil }, apperror.KindValidation, "mesa_requerida"},
		{"llevar con mesa", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Tipo = "LLEVAR" }, apperror.KindValidation, "mesa_no_aplica"},
		{"reparto en un pedido de salón", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Reparto = deliveryOk }, apperror.KindValidation, "reparto_no_aplica"},
		{"delivery sin reparto", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Tipo, c.MesaID = "DELIVERY", nil }, apperror.KindValidation, "reparto_requerido"},
		{"delivery sin destinatario", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Tipo, c.MesaID = "DELIVERY", nil
			c.Reparto = &cmd.Reparto{Destinatario: " ", Telefono: "999", Direccion: "Av. Lima 1"}
		}, apperror.KindValidation, "destinatario_invalido"},
		{"delivery sin teléfono", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Tipo, c.MesaID = "DELIVERY", nil
			c.Reparto = &cmd.Reparto{Destinatario: "Ana", Telefono: "", Direccion: "Av. Lima 1"}
		}, apperror.KindValidation, "telefono_invalido"},
		{"delivery sin dirección", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Tipo, c.MesaID = "DELIVERY", nil
			c.Reparto = &cmd.Reparto{Destinatario: "Ana", Telefono: "999", Direccion: ""}
		}, apperror.KindValidation, "direccion_invalida"},
		{"mesa inexistente", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesaExiste = false }, apperror.KindValidation, "mesa_invalida"},
		{"mesa de otra sede", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesa.SucursalID = 2 }, apperror.KindValidation, "mesa_invalida"},
		{"mesa no habilitada", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesa.Habilitada = false }, apperror.KindValidation, "mesa_no_disponible"},
		{"mesa dada de baja", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesa.EsActiva = false }, apperror.KindValidation, "mesa_no_disponible"},
		{"capacidad excedida", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Comensales = 5 }, apperror.KindValidation, "capacidad_excedida"},
		{"mesa con pedido abierto", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.errCrear = outbound.ErrMesaOcupada }, apperror.KindConflict, "mesa_ocupada"},
		{"clave usada por otro pedido", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.errCrear = outbound.ErrClaveEnUso }, apperror.KindConflict, "clave_en_uso"},
		{"referencia inexistente", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.errCrear = outbound.ErrReferenciaInvalida }, apperror.KindValidation, "referencia_invalida"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := fakeApertura()
			in := abrirBase()
			c.mod(f, &in)
			res, err := NewAbrirPedido(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}
}

func TestAbrirPedidoRechazadoNoToca(t *testing.T) {
	f := fakeApertura()
	in := abrirBase()
	in.Comensales = 9
	_, err := NewAbrirPedido(f).Execute(context.Background(), in)
	sinFallo(t, err)
	if f.creaciones != 0 {
		t.Fatal("un rechazo no debe escribir")
	}
}
