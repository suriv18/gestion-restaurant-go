package command

import (
	"context"
	"errors"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type mesaStoreFake struct {
	existente  dtoresult.Mesa
	zona       outbound.ZonaRef
	zonaExiste bool
	errCrear   error
	errBaja    error
	creada     outbound.NuevaMesa
	cambios    outbound.CambiosMesa
	llamadas   int
	bajas      int
}

func (f *mesaStoreFake) Crear(_ context.Context, n outbound.NuevaMesa) (dtoresult.Mesa, error) {
	f.llamadas++
	f.creada = n
	return dtoresult.Mesa{ID: 3, SucursalID: n.SucursalID, ZonaID: n.ZonaID, NumeroMesa: n.NumeroMesa, EsActivo: true}, f.errCrear
}
func (f *mesaStoreFake) Obtener(_ context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return f.existente, f.existente.ID == id, nil
}
func (f *mesaStoreFake) ZonaDeMesa(context.Context, int64) (outbound.ZonaRef, bool, error) {
	return f.zona, f.zonaExiste, nil
}
func (f *mesaStoreFake) Actualizar(_ context.Context, _ int64, c outbound.CambiosMesa) (dtoresult.Mesa, bool, error) {
	f.llamadas++
	f.cambios = c
	return f.existente, true, nil
}
func (f *mesaStoreFake) DarBaja(_ context.Context, _ int64, _ string) (dtoresult.Mesa, bool, error) {
	f.bajas++
	m := f.existente
	m.EsActivo = false
	return m, true, f.errBaja
}
func (f *mesaStoreFake) Restaurar(_ context.Context, _ int64) (dtoresult.Mesa, bool, error) {
	f.llamadas++
	m := f.existente
	m.EsActivo = true
	return m, true, nil
}

func booleano(b bool) *bool { return &b }

func TestCrearMesa(t *testing.T) {
	zonaOk := outbound.ZonaRef{SucursalID: 1, EsActiva: true}
	ok := cmd.CrearMesa{Actor: gestorDe(1), SucursalID: 1, ZonaID: 8, NumeroMesa: " A-1 ", Capacidad: 4, PosicionX: texto(" 12.50 ")}

	f := &mesaStoreFake{zona: zonaOk, zonaExiste: true}
	res, err := NewCrearMesa(f).Execute(context.Background(), ok)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía crearse: %v %+v", err, res.Err())
	}
	if f.creada.NumeroMesa != "A-1" || f.creada.Capacidad != 4 || !f.creada.Habilitada ||
		f.creada.PosicionX == nil || *f.creada.PosicionX != "12.50" || f.creada.PosicionY != nil {
		t.Fatalf("no normalizó: %+v", f.creada)
	}
	deshabilitada := ok
	deshabilitada.Habilitada = booleano(false)
	_, _ = NewCrearMesa(f).Execute(context.Background(), deshabilitada)
	if f.creada.Habilitada {
		t.Error("habilitada=false debe respetarse")
	}

	casos := []struct {
		nombre string
		mod    func(c *cmd.CrearMesa)
		zona   outbound.ZonaRef
		existe bool
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(c *cmd.CrearMesa) { c.Actor = appctx.Actor{} }, zonaOk, true, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(c *cmd.CrearMesa) { c.Actor = soloVer(1) }, zonaOk, true, apperror.KindForbidden, "sin_permiso"},
		{"sede ajena", func(c *cmd.CrearMesa) { c.SucursalID = 9 }, zonaOk, true, apperror.KindNotFound, "no_encontrado"},
		{"número en blanco", func(c *cmd.CrearMesa) { c.NumeroMesa = " " }, zonaOk, true, apperror.KindValidation, "numero_mesa_invalido"},
		{"capacidad cero", func(c *cmd.CrearMesa) { c.Capacidad = 0 }, zonaOk, true, apperror.KindValidation, "capacidad_invalida"},
		{"posición x inválida", func(c *cmd.CrearMesa) { c.PosicionX = texto("abc") }, zonaOk, true, apperror.KindValidation, "posicion_x_invalida"},
		{"posición y inválida", func(c *cmd.CrearMesa) { c.PosicionY = texto("1.234") }, zonaOk, true, apperror.KindValidation, "posicion_y_invalida"},
		{"zona inexistente", func(c *cmd.CrearMesa) {}, zonaOk, false, apperror.KindValidation, "zona_invalida"},
		{"zona de otra sede", func(c *cmd.CrearMesa) {}, outbound.ZonaRef{SucursalID: 2, EsActiva: true}, true, apperror.KindValidation, "zona_invalida"},
		{"zona inactiva", func(c *cmd.CrearMesa) {}, outbound.ZonaRef{SucursalID: 1, EsActiva: false}, true, apperror.KindValidation, "zona_invalida"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			in := ok
			c.mod(&in)
			f := &mesaStoreFake{zona: c.zona, zonaExiste: c.existe}
			res, err := NewCrearMesa(f).Execute(context.Background(), in)
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
			if f.llamadas != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
	t.Run("número duplicado", func(t *testing.T) {
		res, _ := NewCrearMesa(&mesaStoreFake{zona: zonaOk, zonaExiste: true, errCrear: outbound.ErrNumeroMesaDuplicado}).Execute(context.Background(), ok)
		esperarRechazo(t, res.Err(), apperror.KindConflict, "numero_mesa_duplicado")
	})
	t.Run("error técnico", func(t *testing.T) {
		if _, err := NewCrearMesa(&mesaStoreFake{zona: zonaOk, zonaExiste: true, errCrear: errors.New("conexión perdida")}).Execute(context.Background(), ok); err == nil {
			t.Fatal("un error técnico debe viajar por el canal error")
		}
	})
}

func TestEditarMesa(t *testing.T) {
	activa := dtoresult.Mesa{ID: 5, SucursalID: 1, ZonaID: 8, EsActivo: true}
	zonaOk := outbound.ZonaRef{SucursalID: 1, EsActiva: true}
	nueva := int64(9)

	res, _ := NewEditarMesa(&mesaStoreFake{existente: activa}).Execute(context.Background(),
		cmd.EditarMesa{Actor: gestorDe(9), ID: 5, Capacidad: entero(2)})
	esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	res, _ = NewEditarMesa(&mesaStoreFake{existente: dtoresult.Mesa{ID: 5, SucursalID: 1}}).Execute(context.Background(),
		cmd.EditarMesa{Actor: gestorDe(1), ID: 5, Habilitada: booleano(true)})
	esperarRechazo(t, res.Err(), apperror.KindConflict, "mesa_inactiva")

	f := &mesaStoreFake{existente: activa, zona: zonaOk, zonaExiste: true}
	res, err := NewEditarMesa(f).Execute(context.Background(), cmd.EditarMesa{
		Actor: gestorDe(1), ID: 5, ZonaID: &nueva, NumeroMesa: texto(" B-2 "), Capacidad: entero(6), Habilitada: booleano(false),
		PosicionX: cmd.Campo[string]{Presente: true}, PosicionY: cmd.Campo[string]{Presente: true, Valor: texto("3.5")}})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	c := f.cambios
	if *c.ZonaID != 9 || *c.NumeroMesa != "B-2" || *c.Capacidad != 6 || *c.Habilitada ||
		!c.PosicionX.Presente || c.PosicionX.Valor != nil || !c.PosicionY.Presente || *c.PosicionY.Valor != "3.5" {
		t.Fatalf("cambios: %+v", c)
	}

	res, _ = NewEditarMesa(f).Execute(context.Background(), cmd.EditarMesa{Actor: gestorDe(1), ID: 5})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "sin_cambios")
	res, _ = NewEditarMesa(f).Execute(context.Background(), cmd.EditarMesa{Actor: gestorDe(1), ID: 5, Capacidad: entero(0)})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "capacidad_invalida")
	otraSede := &mesaStoreFake{existente: activa, zona: outbound.ZonaRef{SucursalID: 2, EsActiva: true}, zonaExiste: true}
	res, _ = NewEditarMesa(otraSede).Execute(context.Background(), cmd.EditarMesa{Actor: gestorDe(1), ID: 5, ZonaID: &nueva})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "zona_invalida")
	if otraSede.llamadas != 0 {
		t.Fatal("una zona inválida no debe escribir")
	}
}

func TestCambiarEstadoMesa(t *testing.T) {
	activa := dtoresult.Mesa{ID: 5, SucursalID: 1, EsActivo: true}
	baja := func(a appctx.Actor, motivo string) cmd.CambiarEstadoMesa {
		return cmd.CambiarEstadoMesa{Actor: a, ID: 5, Activo: false, Motivo: motivo}
	}

	t.Run("exige motivo", func(t *testing.T) {
		f := &mesaStoreFake{existente: activa}
		res, _ := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "  "))
		esperarRechazo(t, res.Err(), apperror.KindValidation, "motivo_requerido")
		if f.bajas != 0 {
			t.Fatal("no debe dar de baja sin motivo")
		}
	})
	t.Run("la sede ajena se resuelve antes que el motivo", func(t *testing.T) {
		res, _ := NewCambiarEstadoMesa(&mesaStoreFake{existente: activa}).Execute(context.Background(), baja(gestorDe(9), ""))
		esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	})
	t.Run("baja correcta", func(t *testing.T) {
		f := &mesaStoreFake{existente: activa}
		res, err := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "Mesa dañada"))
		if err != nil || !res.IsOk() || res.Value().EsActivo || f.bajas != 1 {
			t.Fatalf("baja: %v %+v", err, res.Err())
		}
	})
	t.Run("regla de la base (pedido abierto)", func(t *testing.T) {
		f := &mesaStoreFake{existente: activa, errBaja: &outbound.ErrReglaDatos{Mensaje: "Cerrar el pedido de la mesa antes de darla de baja"}}
		res, err := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "Mesa dañada"))
		if err != nil {
			t.Fatalf("una regla de la base es un rechazo, no un error técnico: %v", err)
		}
		esperarRechazo(t, res.Err(), apperror.KindBusinessRule, "regla_de_datos")
	})
	t.Run("ya estaba de baja", func(t *testing.T) {
		f := &mesaStoreFake{existente: dtoresult.Mesa{ID: 5, SucursalID: 1}}
		res, _ := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "Otra vez"))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_dada_de_baja")
	})
	t.Run("restaurar", func(t *testing.T) {
		f := &mesaStoreFake{existente: dtoresult.Mesa{ID: 5, SucursalID: 1}}
		res, err := NewCambiarEstadoMesa(f).Execute(context.Background(), cmd.CambiarEstadoMesa{Actor: gestorDe(1), ID: 5, Activo: true})
		if err != nil || !res.IsOk() || !res.Value().EsActivo {
			t.Fatalf("restaurar: %v %+v", err, res.Err())
		}
		res, _ = NewCambiarEstadoMesa(&mesaStoreFake{existente: activa}).Execute(context.Background(), cmd.CambiarEstadoMesa{Actor: gestorDe(1), ID: 5, Activo: true})
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_activa")
	})
}
