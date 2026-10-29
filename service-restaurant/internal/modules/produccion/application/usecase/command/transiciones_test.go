package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// transicionStoreFake implementa outbound.TransicionStore.
type transicionStoreFake struct {
	linea       outbound.LineaComandaRef
	lineaExiste bool

	destino      string
	transiciones int
}

var _ outbound.TransicionStore = (*transicionStoreFake)(nil)

func (f *transicionStoreFake) BloquearLineaComanda(context.Context, int64) (outbound.LineaComandaRef, bool, error) {
	return f.linea, f.lineaExiste, nil
}
func (f *transicionStoreFake) TransicionarLinea(_ context.Context, id int64, destino string) (dtoresult.ComandaLinea, error) {
	f.transiciones++
	f.destino = destino
	return dtoresult.ComandaLinea{ID: id, Estado: destino}, nil
}

func fakeTransicion(estado string) *transicionStoreFake {
	return &transicionStoreFake{linea: outbound.LineaComandaRef{ID: 40, SucursalID: 1, Estado: estado}, lineaExiste: true}
}

func transicionBase(destino string, permisos ...string) cmd.TransicionarLinea {
	return cmd.TransicionarLinea{Actor: actorEn(1, permisos...), LineaID: 40, Estado: destino}
}

func TestTransicionarLineaAvanzaUnPaso(t *testing.T) {
	pasos := []struct{ desde, hacia string }{
		{"PENDIENTE", "PREPARACION"}, {"PREPARACION", "LISTO"}, {"LISTO", "ENTREGADO"},
	}
	for _, p := range pasos {
		f := fakeTransicion(p.desde)
		res, err := NewTransicionarLinea(f).Execute(context.Background(), transicionBase(p.hacia, policy.CocinaOperar))
		if err != nil || !res.IsOk() || f.destino != p.hacia || res.Value().Estado != p.hacia {
			t.Fatalf("%s→%s debía avanzar: %v %v", p.desde, p.hacia, err, res.Err())
		}
	}
}

func TestLaEntregaLaPuedeConfirmarQuienEditaPedidos(t *testing.T) {
	f := fakeTransicion("LISTO")
	res, err := NewTransicionarLinea(f).Execute(context.Background(), transicionBase("ENTREGADO", policy.PedidoEditar))
	if err != nil || !res.IsOk() {
		t.Fatalf("el mesero con PEDIDO_EDITAR debía confirmar la entrega: %v %v", err, res.Err())
	}
}

func TestTransicionarLineaRechaza(t *testing.T) {
	casos := []struct {
		nombre string
		store  *transicionStoreFake
		cmd    cmd.TransicionarLinea
		kind   apperror.Kind
		code   string
	}{
		{"sin actor", fakeTransicion("PENDIENTE"), cmd.TransicionarLinea{LineaID: 40, Estado: "PREPARACION"},
			apperror.KindUnauthorized, "no_autenticado"},
		{"estado desconocido", fakeTransicion("PENDIENTE"), transicionBase("COCINANDO", policy.CocinaOperar),
			apperror.KindValidation, "estado_invalido"},
		{"destino PENDIENTE", fakeTransicion("PENDIENTE"), transicionBase("PENDIENTE", policy.CocinaOperar),
			apperror.KindValidation, "estado_invalido"},
		{"el mesero no prepara", fakeTransicion("PENDIENTE"), transicionBase("PREPARACION", policy.PedidoEditar),
			apperror.KindForbidden, "sin_permiso"},
		{"el lector no entrega", fakeTransicion("LISTO"), transicionBase("ENTREGADO", policy.PedidoVer),
			apperror.KindForbidden, "sin_permiso"},
		{"línea inexistente", &transicionStoreFake{}, transicionBase("PREPARACION", policy.CocinaOperar),
			apperror.KindNotFound, "no_encontrado"},
		{"sede que el actor no ve", func() *transicionStoreFake {
			f := fakeTransicion("PENDIENTE")
			f.linea.SucursalID = 2
			return f
		}(), transicionBase("PREPARACION", policy.CocinaOperar), apperror.KindNotFound, "no_encontrado"},
		{"ve la sede pero no opera allí", func() *transicionStoreFake {
			f := fakeTransicion("PENDIENTE")
			f.linea.SucursalID = 2
			return f
		}(), cmd.TransicionarLinea{LineaID: 40, Estado: "PREPARACION", Actor: appctx.NewActor(7, "ana", 1, []appctx.Grant{
			{SucursalID: 1, Permiso: policy.CocinaOperar}, {SucursalID: 2, Permiso: policy.PedidoVer}})},
			apperror.KindForbidden, "sede_no_autorizada"},
		{"salto PENDIENTE→LISTO", fakeTransicion("PENDIENTE"), transicionBase("LISTO", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"retroceso LISTO→PREPARACION", fakeTransicion("LISTO"), transicionBase("PREPARACION", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"repetir el mismo estado", fakeTransicion("PREPARACION"), transicionBase("PREPARACION", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"CANCELADO es de la rebanada 3", fakeTransicion("PENDIENTE"), transicionBase("CANCELADO", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"nada se avanza desde ENTREGADO", fakeTransicion("ENTREGADO"), transicionBase("ENTREGADO", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			res, err := NewTransicionarLinea(tc.store).Execute(context.Background(), tc.cmd)
			if err != nil {
				t.Fatalf("un rechazo esperado no es un error técnico: %v", err)
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if tc.store.transiciones != 0 {
				t.Fatalf("una transición rechazada no debe escribir")
			}
		})
	}
}
