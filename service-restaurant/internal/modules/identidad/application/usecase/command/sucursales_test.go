package command

import (
	"context"
	"errors"
	"testing"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type sucursalStoreFake struct {
	existente  dtoresult.Sucursal
	activas    int
	errCrear   error
	creada     outbound.NuevaSucursal
	cambios    outbound.CambiosSucursal
	bajas      int
	restaurada int
	bloqueos   int
}

func (f *sucursalStoreFake) Crear(_ context.Context, n outbound.NuevaSucursal) (dtoresult.Sucursal, error) {
	f.creada = n
	return dtoresult.Sucursal{ID: 3, Codigo: n.Codigo, EsActivo: true}, f.errCrear
}
func (f *sucursalStoreFake) Obtener(_ context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return f.existente, f.existente.ID == id, nil
}
func (f *sucursalStoreFake) Actualizar(_ context.Context, id int64, c outbound.CambiosSucursal) (dtoresult.Sucursal, bool, error) {
	f.cambios = c
	return f.existente, true, nil
}
func (f *sucursalStoreFake) BloquearNegocio(context.Context) error      { f.bloqueos++; return nil }
func (f *sucursalStoreFake) ContarActivas(context.Context) (int, error) { return f.activas, nil }
func (f *sucursalStoreFake) DarBaja(_ context.Context, id int64, _ string) (dtoresult.Sucursal, bool, error) {
	f.bajas++
	s := f.existente
	s.EsActivo = false
	return s, true, nil
}
func (f *sucursalStoreFake) Restaurar(_ context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	f.restaurada++
	s := f.existente
	s.EsActivo = true
	return s, true, nil
}

func gestor(sedes ...int64) appctx.Actor {
	var g []appctx.Grant
	for _, s := range sedes {
		g = append(g, appctx.Grant{SucursalID: s, Permiso: policy.SucursalAdministrar})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

func TestCrearSucursal(t *testing.T) {
	ok := cmd.CrearSucursal{Actor: gestor(1), Codigo: " san-isidro ", Nombre: " San Isidro ", Direccion: "Av. Javier Prado 100",
		Ubigeo: texto("150131"), Telefono: texto("   ")}

	f := &sucursalStoreFake{}
	res, err := NewCrearSucursal(f).Execute(context.Background(), ok)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía crearse: %v %+v", err, res.Err())
	}
	if f.creada.Codigo != "SAN-ISIDRO" || f.creada.Nombre != "San Isidro" || *f.creada.Ubigeo != "150131" || f.creada.Telefono != nil {
		t.Fatalf("no normalizó: %+v", f.creada)
	}

	casos := []struct {
		nombre string
		mod    func(c *cmd.CrearSucursal)
		kind   apperror.Kind
		code   string
	}{
		{"sin permiso de sucursal", func(c *cmd.CrearSucursal) { c.Actor = actorNegocio(policy.NegocioAdministrar) }, apperror.KindForbidden, "sin_permiso"},
		{"código inválido", func(c *cmd.CrearSucursal) { c.Codigo = "x" }, apperror.KindValidation, "codigo_invalido"},
		{"nombre en blanco", func(c *cmd.CrearSucursal) { c.Nombre = " " }, apperror.KindValidation, "nombre_invalido"},
		{"dirección en blanco", func(c *cmd.CrearSucursal) { c.Direccion = "" }, apperror.KindValidation, "direccion_invalida"},
		{"ubigeo", func(c *cmd.CrearSucursal) { c.Ubigeo = texto("1") }, apperror.KindValidation, "ubigeo_invalido"},
		{"código SUNAT largo", func(c *cmd.CrearSucursal) { c.CodigoEstablecimientoSunat = texto("12345678901") }, apperror.KindValidation, "codigo_establecimiento_sunat_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := &sucursalStoreFake{}
			in := ok
			c.mod(&in)
			res, err := NewCrearSucursal(f).Execute(context.Background(), in)
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}

	t.Run("código duplicado", func(t *testing.T) {
		f := &sucursalStoreFake{errCrear: outbound.ErrCodigoSucursalDuplicado}
		res, err := NewCrearSucursal(f).Execute(context.Background(), ok)
		if err != nil || res.Err() == nil {
			t.Fatalf("debía rechazarse: %v", err)
		}
		esperarRechazo(t, res.Err(), apperror.KindConflict, "codigo_duplicado")
	})
	t.Run("error técnico", func(t *testing.T) {
		f := &sucursalStoreFake{errCrear: errors.New("conexión perdida")}
		if _, err := NewCrearSucursal(f).Execute(context.Background(), ok); err == nil {
			t.Fatal("un error técnico debe viajar por el canal error")
		}
	})
}

func TestEditarSucursalAutorizaPorSede(t *testing.T) {
	activa := dtoresult.Sucursal{ID: 5, EsActivo: true}
	edit := func(a appctx.Actor) cmd.EditarSucursal {
		return cmd.EditarSucursal{Actor: a, ID: 5, Nombre: texto("Nueva")}
	}

	casos := []struct {
		nombre string
		actor  appctx.Actor
		sede   dtoresult.Sucursal
		kind   apperror.Kind
		code   string
	}{
		{"sede ajena activa no se revela", gestor(1), activa, apperror.KindNotFound, "no_encontrado"},
		{"solo VER ve pero no edita", appctx.NewActor(7, "a", 1, []appctx.Grant{
			{SucursalID: 5, Permiso: policy.SucursalVer}, {SucursalID: 9, Permiso: policy.SucursalAdministrar}}), activa, apperror.KindForbidden, "sede_no_autorizada"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := &sucursalStoreFake{existente: c.sede}
			res, err := NewEditarSucursal(f).Execute(context.Background(), edit(c.actor))
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}

	f := &sucursalStoreFake{existente: activa}
	res, err := NewEditarSucursal(f).Execute(context.Background(), cmd.EditarSucursal{
		Actor: gestor(5), ID: 5, Nombre: texto("  Nueva  "), Telefono: cmd.Campo[string]{Presente: true}})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	if *f.cambios.Nombre != "Nueva" || !f.cambios.Telefono.Presente || f.cambios.Telefono.Valor != nil || f.cambios.Ubigeo.Presente {
		t.Fatalf("cambios: %+v", f.cambios)
	}
	res, _ = NewEditarSucursal(f).Execute(context.Background(), cmd.EditarSucursal{Actor: gestor(5), ID: 5})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "sin_cambios")
}

func TestDarDeBajaSucursal(t *testing.T) {
	baja := func(a appctx.Actor, motivo string) cmd.CambiarEstadoSucursal {
		return cmd.CambiarEstadoSucursal{Actor: a, ID: 5, Activo: false, Motivo: motivo}
	}

	t.Run("exige motivo", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 2}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(5), "  "))
		esperarRechazo(t, res.Err(), apperror.KindValidation, "motivo_requerido")
	})
	t.Run("no deja a la empresa sin sede activa", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 1}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(5), "Cierre"))
		esperarRechazo(t, res.Err(), apperror.KindBusinessRule, "ultima_sede_activa")
		if f.bajas != 0 || f.bloqueos != 1 {
			t.Fatalf("bajas=%d bloqueos=%d: debe bloquear antes de contar y no dar de baja", f.bajas, f.bloqueos)
		}
	})
	t.Run("baja correcta", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 2}
		res, err := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(5), "Cierre del local"))
		if err != nil || !res.IsOk() || res.Value().EsActivo || f.bajas != 1 {
			t.Fatalf("baja: %v %+v", err, res.Err())
		}
	})
	t.Run("ya estaba de baja", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: false}, activas: 3}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(1), "Otra vez"))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_dada_de_baja")
	})
	t.Run("sede activa ajena", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 3}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(1), "No es mía"))
		esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	})
}

func TestRestaurarSucursal(t *testing.T) {
	rest := func(a appctx.Actor) cmd.CambiarEstadoSucursal {
		return cmd.CambiarEstadoSucursal{Actor: a, ID: 5, Activo: true}
	}
	// Una sede inactiva no genera permisos: la restaura quien administra sedes en cualquier otra.
	f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: false}}
	res, err := NewCambiarEstadoSucursal(f).Execute(context.Background(), rest(gestor(1)))
	if err != nil || !res.IsOk() || !res.Value().EsActivo || f.restaurada != 1 {
		t.Fatalf("restaurar: %v %+v", err, res.Err())
	}
	f = &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}}
	res, _ = NewCambiarEstadoSucursal(f).Execute(context.Background(), rest(gestor(5)))
	esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_activa")
	res, _ = NewCambiarEstadoSucursal(&sucursalStoreFake{}).Execute(context.Background(), rest(appctx.Actor{}))
	esperarRechazo(t, res.Err(), apperror.KindUnauthorized, "no_autenticado")
}
