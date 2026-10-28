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

func fakeConsumo() *pedidoStoreFake {
	return &pedidoStoreFake{
		pedido:       outbound.PedidoRef{ID: 10, SucursalID: 1, Estado: "REGISTRADO"},
		pedidoExiste: true,
		cuenta:       outbound.CuentaRef{ID: 20, PedidoID: 10, Estado: "ABIERTA"},
		cuentaExiste: true,
		producto: outbound.ProductoVenta{ID: 30, Nombre: "Lomo saltado", Tipo: "PREPARADO", AfectacionIGV: "10",
			Precio: "10.00", Tasa: "18.0000", EsActivo: true, Disponible: true},
		productoExiste: true,
		detalles:       map[int64]outbound.DetalleRef{},
	}
}

func agregarBase() cmd.AgregarDetalle {
	return cmd.AgregarDetalle{
		Intencion: cmd.Intencion{Clave: claveEjemplo},
		Actor:     actorPedidos(1, policy.PedidoEditar),
		PedidoID:  10, CuentaID: 20, ProductoID: 30, Cantidad: "2",
	}
}

func conDescuento() appctx.Actor {
	return actorPedidos(1, policy.PedidoEditar, policy.DescuentoAutorizar)
}

func TestAgregarDetalleFijaLaInstantanea(t *testing.T) {
	f := fakeConsumo()
	c := agregarBase()
	c.Cantidad = " 2.5 "
	c.Instrucciones = texto("  sin ají ")
	res, err := NewAgregarDetalle(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía agregarse: %v %+v", err, res.Err())
	}
	n := f.insertado
	if n.PedidoID != 10 || n.CuentaID != 20 || n.ProductoID != 30 || n.Descripcion != "Lomo saltado" ||
		n.Cantidad != "2.5" || n.PrecioUnitario != "10.00" || n.DescuentoTotal != "0" ||
		n.AfectacionIGV != "10" || n.TasaImpuesto != "18.0000" || n.DetallePadreID != nil ||
		n.NotasPreparacion == nil || *n.NotasPreparacion != "sin ají" {
		t.Fatalf("instantánea mal armada: %+v", n)
	}
}

func TestAgregarDetalleExoneradoVaConTasaCero(t *testing.T) {
	f := fakeConsumo()
	f.producto.AfectacionIGV, f.producto.Tasa = "20", ""
	res, err := NewAgregarDetalle(f).Execute(context.Background(), agregarBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("un producto exonerado no necesita tasa: %v %+v", err, res.Err())
	}
	if f.insertado.AfectacionIGV != "20" || f.insertado.TasaImpuesto != "0" {
		t.Fatalf("tasa: %+v", f.insertado)
	}
}

func TestAgregarDetalleConDescuentoAutorizado(t *testing.T) {
	f := fakeConsumo()
	c := agregarBase()
	c.Actor = conDescuento()
	c.DescuentoMonto = texto("5.00")
	res, err := NewAgregarDetalle(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() || f.insertado.DescuentoTotal != "5.00" {
		t.Fatalf("descuento autorizado: %v %+v %+v", err, res.Err(), f.insertado)
	}
}

func TestAgregarAdicionalConPadre(t *testing.T) {
	f := fakeConsumo()
	f.producto.Tipo = "ADICIONAL"
	f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, Estado: "ACTIVO"}
	c := agregarBase()
	c.DetallePadreID = idPtr(55)
	res, err := NewAgregarDetalle(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() || f.insertado.DetallePadreID == nil || *f.insertado.DetallePadreID != 55 {
		t.Fatalf("adicional con padre: %v %+v %+v", err, res.Err(), f.insertado)
	}
}

func TestAgregarDetalleRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *pedidoStoreFake, c *cmd.AgregarDetalle)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"cantidad cero", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Cantidad = "0" }, apperror.KindValidation, "cantidad_invalida"},
		{"cantidad negativa", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Cantidad = "-1" }, apperror.KindValidation, "cantidad_invalida"},
		{"descuento ilegible", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.DescuentoMonto = texto("abc") }, apperror.KindValidation, "descuento_invalido"},
		{"instrucciones con caracteres de control", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Instrucciones = texto("a\x00") }, apperror.KindValidation, "instrucciones_invalidas"},
		{"pedido inexistente", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado o cancelado", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"descuento sin permiso", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.DescuentoMonto = texto("1.00") }, apperror.KindForbidden, "sin_permiso"},
		{"cuenta inexistente o ajena", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.cuentaExiste = false }, apperror.KindValidation, "cuenta_invalida"},
		{"cuenta cerrada", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.cuenta.Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"producto inexistente", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.productoExiste = false }, apperror.KindValidation, "producto_invalido"},
		{"producto dado de baja", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.EsActivo = false }, apperror.KindValidation, "producto_inactivo"},
		{"producto no disponible en la sede", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.Disponible = false }, apperror.KindValidation, "producto_no_disponible"},
		{"gravado sin tasa vigente", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.Tasa = "" }, apperror.KindValidation, "tasa_no_vigente"},
		{"descuento mayor que el importe", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), "1", texto("10.01")
		}, apperror.KindValidation, "descuento_excede_importe"},
		{"adicional sin padre", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.Tipo = "ADICIONAL" }, apperror.KindValidation, "adicional_sin_padre"},
		{"padre en un producto que no es adicional", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, Estado: "ACTIVO"}
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_no_aplica"},
		{"padre inexistente", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.producto.Tipo = "ADICIONAL"
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_invalido"},
		{"padre de otra cuenta", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.producto.Tipo = "ADICIONAL"
			f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 21, Estado: "ACTIVO"}
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_invalido"},
		{"padre anulado", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.producto.Tipo = "ADICIONAL"
			f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, Estado: "ANULADO"}
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := fakeConsumo()
			in := agregarBase()
			c.mod(f, &in)
			res, err := NewAgregarDetalle(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
			if f.inserciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}

func detalleActivo() outbound.DetalleRef {
	return outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, ProductoID: 30,
		Cantidad: "2.000", PrecioUnitario: "10.000000", DescuentoTotal: "0.00", Estado: "ACTIVO"}
}

func fakeEdicion() *pedidoStoreFake {
	f := fakeConsumo()
	f.detalles[55] = detalleActivo()
	return f
}

func editarBase() cmd.EditarDetalle {
	return cmd.EditarDetalle{Actor: actorPedidos(1, policy.PedidoEditar), PedidoID: 10, DetalleID: 55, Cantidad: texto("3")}
}

func TestEditarDetalle(t *testing.T) {
	f := fakeEdicion()
	res, err := NewEditarDetalle(f).Execute(context.Background(), editarBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	if f.actualizaciones != 1 || f.cambios.Cantidad == nil || *f.cambios.Cantidad != "3" ||
		f.cambios.DescuentoTotal != nil || f.cambios.Instrucciones.Presente {
		t.Fatalf("cambios mal armados: %+v", f.cambios)
	}

	// Vaciar las instrucciones (null) se distingue de no enviarlas.
	f = fakeEdicion()
	c := editarBase()
	c.Cantidad = nil
	c.Instrucciones = cmd.Campo[string]{Presente: true}
	if res, err := NewEditarDetalle(f).Execute(context.Background(), c); err != nil || !res.IsOk() ||
		!f.cambios.Instrucciones.Presente || f.cambios.Instrucciones.Valor != nil {
		t.Fatalf("vaciar instrucciones: %v %+v %+v", err, res.Err(), f.cambios)
	}

	// Con permiso de descuento se puede fijar uno.
	f = fakeEdicion()
	c = editarBase()
	c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), nil, texto("5.00")
	if res, err := NewEditarDetalle(f).Execute(context.Background(), c); err != nil || !res.IsOk() ||
		f.cambios.DescuentoTotal == nil || *f.cambios.DescuentoTotal != "5.00" {
		t.Fatalf("descuento: %v %+v %+v", err, res.Err(), f.cambios)
	}
}

func TestEditarDetalleRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *pedidoStoreFake, c *cmd.EditarDetalle)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"sin cambios", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Cantidad = nil }, apperror.KindValidation, "sin_cambios"},
		{"cantidad cero", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Cantidad = texto("0") }, apperror.KindValidation, "cantidad_invalida"},
		{"pedido inexistente", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"línea inexistente", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { delete(f.detalles, 55) }, apperror.KindNotFound, "no_encontrado"},
		{"línea anulada", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) {
			d := f.detalles[55]
			d.Estado = "ANULADO"
			f.detalles[55] = d
		}, apperror.KindConflict, "linea_anulada"},
		{"línea enviada a cocina", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) {
			d := f.detalles[55]
			d.EnComanda = true
			f.detalles[55] = d
		}, apperror.KindConflict, "linea_en_cocina"},
		{"cuenta cerrada", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.cuenta.Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"descuento sin permiso", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Cantidad, c.DescuentoMonto = nil, texto("1.00") }, apperror.KindForbidden, "sin_permiso"},
		{"descuento ilegible", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) {
			c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), nil, texto("x")
		}, apperror.KindValidation, "descuento_invalido"},
		{"descuento mayor que el importe", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) {
			c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), nil, texto("20.01")
		}, apperror.KindValidation, "descuento_excede_importe"},
		{"bajar la cantidad deja el descuento por encima del importe", func(f *pedidoStoreFake, c *cmd.EditarDetalle) {
			d := f.detalles[55]
			d.DescuentoTotal = "15.00"
			f.detalles[55] = d
			c.Cantidad = texto("1")
		}, apperror.KindValidation, "descuento_excede_importe"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := fakeEdicion()
			in := editarBase()
			c.mod(f, &in)
			res, err := NewEditarDetalle(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
			if f.actualizaciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}
