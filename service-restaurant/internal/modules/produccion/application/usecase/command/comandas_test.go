package command

import (
	"context"
	"errors"
	"slices"
	"testing"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

const claveEjemplo = "6f1c0a52-3c1e-4a53-9a39-0b9a3a2f1a11"

func idPtr(n int64) *int64   { return &n }
func texto(s string) *string { return &s }

func actorEn(sede int64, permisos ...string) appctx.Actor {
	g := make([]appctx.Grant, 0, len(permisos))
	for _, p := range permisos {
		g = append(g, appctx.Grant{SucursalID: sede, Permiso: p})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

// envioStoreFake implementa outbound.EnvioStore con respuestas fijadas por la prueba.
type envioStoreFake struct {
	pedido         outbound.PedidoRef
	pedidoExiste   bool
	estacion       outbound.EstacionRef
	estacionExiste bool
	lineas         []outbound.LineaPedidoRef
	errCrear       error

	idsPedidos []int64
	creada     outbound.NuevaComanda
	creaciones int
}

var _ outbound.EnvioStore = (*envioStoreFake)(nil)

func (f *envioStoreFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *envioStoreFake) Estacion(context.Context, int64) (outbound.EstacionRef, bool, error) {
	return f.estacion, f.estacionExiste, nil
}
func (f *envioStoreFake) LineasParaEnvio(_ context.Context, _, _ int64, ids []int64) ([]outbound.LineaPedidoRef, error) {
	f.idsPedidos = ids
	var out []outbound.LineaPedidoRef
	for _, l := range f.lineas {
		if slices.Contains(ids, l.ID) {
			out = append(out, l)
		}
	}
	return out, nil
}
func (f *envioStoreFake) CrearComanda(_ context.Context, n outbound.NuevaComanda) (dtoresult.Comanda, error) {
	f.creaciones++
	f.creada = n
	return dtoresult.Comanda{ID: 9, PedidoID: n.PedidoID}, f.errCrear
}

func fakeEnvio() *envioStoreFake {
	return &envioStoreFake{
		pedido:         outbound.PedidoRef{ID: 10, SucursalID: 1},
		pedidoExiste:   true,
		estacion:       outbound.EstacionRef{ID: 5, SucursalID: 1, EsActiva: true},
		estacionExiste: true,
		lineas: []outbound.LineaPedidoRef{{
			ID: 100, CuentaID: 20, CuentaEstado: "ABIERTA", Estado: "ACTIVO", Cantidad: "3.000", Enviado: "0",
			Descripcion: "Lomo saltado", Instrucciones: texto("sin ají"), EstacionID: idPtr(5),
		}},
	}
}

func envioBase() cmd.EnviarComanda {
	return cmd.EnviarComanda{
		Intencion: cmd.Intencion{Clave: claveEjemplo},
		Actor:     actorEn(1, policy.PedidoCrear),
		PedidoID:  10, EstacionID: 5,
		Lineas: []cmd.LineaEnvio{{DetallePedidoID: 100, Cantidad: "2"}},
	}
}

func TestEnviarComandaCreaLaTandaConLaInstantanea(t *testing.T) {
	f := fakeEnvio()
	res, err := NewEnviarComanda(f).Execute(context.Background(), envioBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía enviarse: %v %v", err, res.Err())
	}
	n := f.creada
	if f.creaciones != 1 || n.PedidoID != 10 || n.SucursalID != 1 || n.EstacionID != 5 ||
		n.ClaveIdempotencia != claveEjemplo || n.EnviadoPor != 7 || len(n.Lineas) != 1 {
		t.Fatalf("comanda mal armada: %+v", n)
	}
	l := n.Lineas[0]
	if l.DetallePedidoID != 100 || l.Cantidad != "2.000" || l.Descripcion != "Lomo saltado" ||
		l.Instrucciones == nil || *l.Instrucciones != "sin ají" {
		t.Fatalf("línea mal armada: %+v", l)
	}
}

func TestEnviarComandaAceptaElRestoPendiente(t *testing.T) {
	f := fakeEnvio()
	f.lineas[0].Enviado = "2.000"
	c := envioBase()
	c.Lineas[0].Cantidad = "1"
	if res, err := NewEnviarComanda(f).Execute(context.Background(), c); err != nil || !res.IsOk() {
		t.Fatalf("el resto pendiente debía enviarse: %v %v", err, res.Err())
	}
}

func TestEnviarComandaPideLasLineasOrdenadasYSinRepetir(t *testing.T) {
	f := fakeEnvio()
	f.lineas = append(f.lineas, outbound.LineaPedidoRef{ID: 50, CuentaID: 20, CuentaEstado: "ABIERTA", Estado: "ACTIVO",
		Cantidad: "1.000", Enviado: "0", Descripcion: "Chicha", EstacionID: idPtr(5)})
	c := envioBase()
	c.Lineas = []cmd.LineaEnvio{{DetallePedidoID: 100, Cantidad: "1"}, {DetallePedidoID: 50, Cantidad: "1"}}
	if res, err := NewEnviarComanda(f).Execute(context.Background(), c); err != nil || !res.IsOk() {
		t.Fatalf("debía enviarse: %v %v", err, res.Err())
	}
	if !slices.Equal(f.idsPedidos, []int64{50, 100}) {
		t.Fatalf("los ids deben pedirse ascendentes (orden de bloqueo): %v", f.idsPedidos)
	}
}

func TestEnviarComandaRechaza(t *testing.T) {
	casos := []struct {
		nombre string
		ajusta func(f *envioStoreFake, c *cmd.EnviarComanda)
		kind   apperror.Kind
		code   string
	}{
		{"sin permiso en ninguna sede", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Actor = actorEn(1, policy.PedidoVer) },
			apperror.KindForbidden, "sin_permiso"},
		{"sin actor", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Actor = appctx.Actor{} },
			apperror.KindUnauthorized, "no_autenticado"},
		{"sin líneas", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas = nil },
			apperror.KindValidation, "lineas_invalidas"},
		{"demasiadas líneas", func(_ *envioStoreFake, c *cmd.EnviarComanda) {
			c.Lineas = make([]cmd.LineaEnvio, 101)
			for i := range c.Lineas {
				c.Lineas[i] = cmd.LineaEnvio{DetallePedidoID: int64(i + 1), Cantidad: "1"}
			}
		}, apperror.KindValidation, "lineas_invalidas"},
		{"línea repetida", func(_ *envioStoreFake, c *cmd.EnviarComanda) {
			c.Lineas = append(c.Lineas, cmd.LineaEnvio{DetallePedidoID: 100, Cantidad: "1"})
		}, apperror.KindValidation, "linea_duplicada"},
		{"id de línea inválido", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].DetallePedidoID = 0 },
			apperror.KindValidation, "linea_invalida"},
		{"cantidad cero", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].Cantidad = "0" },
			apperror.KindValidation, "cantidad_invalida"},
		{"cantidad con demasiados decimales", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].Cantidad = "1.2345" },
			apperror.KindValidation, "cantidad_invalida"},
		{"pedido inexistente", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.pedidoExiste = false },
			apperror.KindNotFound, "no_encontrado"},
		{"sede que el actor no ve", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.pedido.SucursalID = 2 },
			apperror.KindNotFound, "no_encontrado"},
		{"ve la sede pero no puede crear", func(f *envioStoreFake, c *cmd.EnviarComanda) {
			f.pedido.SucursalID = 2
			c.Actor = appctx.NewActor(7, "ana", 1, []appctx.Grant{
				{SucursalID: 1, Permiso: policy.PedidoCrear}, {SucursalID: 2, Permiso: policy.PedidoVer}})
		}, apperror.KindForbidden, "sede_no_autorizada"},
		{"pedido cerrado", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.pedido.Cerrado = true },
			apperror.KindConflict, "pedido_cerrado"},
		{"estación inexistente", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.estacionExiste = false },
			apperror.KindValidation, "estacion_invalida"},
		{"estación de otra sede", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.estacion.SucursalID = 2 },
			apperror.KindValidation, "estacion_invalida"},
		{"estación dada de baja", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.estacion.EsActiva = false },
			apperror.KindValidation, "estacion_invalida"},
		{"línea que no es del pedido", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas = nil },
			apperror.KindValidation, "linea_invalida"},
		{"línea anulada", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].Estado = "ANULADO" },
			apperror.KindConflict, "linea_anulada"},
		{"cuenta cerrada", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].CuentaEstado = "CERRADA" },
			apperror.KindConflict, "cuenta_cerrada"},
		{"producto sin estación", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].EstacionID = nil },
			apperror.KindValidation, "estacion_no_corresponde"},
		{"producto de otra estación", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].EstacionID = idPtr(6) },
			apperror.KindValidation, "estacion_no_corresponde"},
		{"excede el pendiente", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].Cantidad = "3.001" },
			apperror.KindConflict, "cantidad_excede_pendiente"},
		{"excede lo que queda tras un envío previo", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].Enviado = "1.500" },
			apperror.KindConflict, "cantidad_excede_pendiente"},
		{"clave de idempotencia usada por otra comanda", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.errCrear = outbound.ErrClaveEnUso },
			apperror.KindConflict, "clave_en_uso"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeEnvio()
			c := envioBase()
			tc.ajusta(f, &c)
			res, err := NewEnviarComanda(f).Execute(context.Background(), c)
			if err != nil {
				t.Fatalf("un rechazo esperado no es un error técnico: %v", err)
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if tc.code != "clave_en_uso" && f.creaciones != 0 {
				t.Fatalf("un envío rechazado no debe crear la comanda")
			}
		})
	}
}

func TestEnviarComandaPropagaUnErrorTecnico(t *testing.T) {
	f := fakeEnvio()
	f.errCrear = errors.New("se cayó la base")
	res, err := NewEnviarComanda(f).Execute(context.Background(), envioBase())
	if err == nil || res.IsValid() {
		t.Fatalf("un error técnico debe salir por el canal error: %v %+v", err, res)
	}
}
