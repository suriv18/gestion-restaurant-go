package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// storeEstaciones es un EstacionStore en memoria de una sola estación; registra qué se invocó.
type storeEstaciones struct {
	est        dtoresult.Estacion
	enUso      bool
	crearErr   error
	escrituras int
}

func (s *storeEstaciones) Crear(context.Context, outbound.NuevaEstacion) (dtoresult.Estacion, error) {
	s.escrituras++
	return s.est, s.crearErr
}
func (s *storeEstaciones) Obtener(context.Context, int64) (dtoresult.Estacion, bool, error) {
	return s.est, true, nil
}
func (s *storeEstaciones) Actualizar(context.Context, int64, outbound.CambiosEstacion) (dtoresult.Estacion, bool, error) {
	s.escrituras++
	return s.est, true, nil
}
func (s *storeEstaciones) DarBaja(context.Context, int64) (dtoresult.Estacion, bool, error) {
	s.escrituras++
	return s.est, true, nil
}
func (s *storeEstaciones) Restaurar(context.Context, int64) (dtoresult.Estacion, bool, error) {
	s.escrituras++
	return s.est, true, nil
}
func (s *storeEstaciones) EnUso(context.Context, int64) (bool, error) { return s.enUso, nil }

func actorEstaciones(g ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, g) }

func esperaRechazo(t *testing.T, err error, e *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if err != nil {
		t.Fatalf("error técnico inesperado: %v", err)
	}

	if e == nil || e.Kind != kind || (code != "" && e.Code != code) {
		t.Fatalf("rechazo = %v, quería %v/%s", e, kind, code)
	}
}

// El 403 sede_no_autorizada no se puede provocar por HTTP (v_permisos_efectivos cruza roles x
// sedes), así que se cubre aquí a nivel de caso de uso.
func TestEstacionesAutorizacionPorSede(t *testing.T) {
	ctx := context.Background()
	st := &storeEstaciones{est: dtoresult.Estacion{ID: 1, SucursalID: 3, Codigo: "COC", Nombre: "Cocina", EsActivo: true}}
	// Administra la sede 2, solo ve la 3 y no tiene nada en la 4.
	a := actorEstaciones(
		appctx.Grant{SucursalID: 2, Permiso: policy.EstacionAdministrar},
		appctx.Grant{SucursalID: 3, Permiso: policy.EstacionVer},
	)

	// Alta: sede ajena → 404, visible sin administrar → 403 sede_no_autorizada, administrada → ok.
	res, err := NewCrearEstacion(st).Execute(ctx, cmd.CrearEstacion{Actor: a, SucursalID: 4, Codigo: "x", Nombre: "X"})
	esperaRechazo(t, err, res.Err(), apperror.KindNotFound, "no_encontrado")
	res, err = NewCrearEstacion(st).Execute(ctx, cmd.CrearEstacion{Actor: a, SucursalID: 3, Codigo: "x", Nombre: "X"})
	esperaRechazo(t, err, res.Err(), apperror.KindForbidden, "sede_no_autorizada")
	res, err = NewCrearEstacion(st).Execute(ctx, cmd.CrearEstacion{Actor: a, SucursalID: 2, Codigo: "x", Nombre: "X"})
	if err != nil || res.Err() != nil {
		t.Fatalf("alta en sede administrada: %v %v", err, res.Err())
	}

	// Editar / baja / restaurar de la estación de la sede 3 (visible, no administrada) → 403.
	st.escrituras = 0
	n := "Nuevo"
	res2, err := NewEditarEstacion(st).Execute(ctx, cmd.EditarEstacion{Actor: a, ID: 1, Nombre: &n})
	esperaRechazo(t, err, res2.Err(), apperror.KindForbidden, "sede_no_autorizada")
	for _, activo := range []bool{false, true} {
		res2, err = NewCambiarEstadoEstacion(st).Execute(ctx, cmd.CambiarEstadoEstacion{Actor: a, ID: 1, Activo: activo})
		esperaRechazo(t, err, res2.Err(), apperror.KindForbidden, "sede_no_autorizada")
	}
	if st.escrituras != 0 {
		t.Fatalf("un rechazo de autorización no debe escribir (%d escrituras)", st.escrituras)
	}

	// Estación de una sede que no ve → 404 de estación.
	st.est.SucursalID = 9
	res2, err = NewEditarEstacion(st).Execute(ctx, cmd.EditarEstacion{Actor: a, ID: 1, Nombre: &n})
	esperaRechazo(t, err, res2.Err(), apperror.KindNotFound, "no_encontrado")
	res2, err = NewCambiarEstadoEstacion(st).Execute(ctx, cmd.CambiarEstadoEstacion{Actor: a, ID: 1})
	esperaRechazo(t, err, res2.Err(), apperror.KindNotFound, "no_encontrado")

	// Sin ningún permiso de administración → 403 sin_permiso.
	soloVer := actorEstaciones(appctx.Grant{SucursalID: 3, Permiso: policy.EstacionVer})
	res, err = NewCrearEstacion(st).Execute(ctx, cmd.CrearEstacion{Actor: soloVer, SucursalID: 3, Codigo: "x", Nombre: "X"})
	esperaRechazo(t, err, res.Err(), apperror.KindForbidden, "sin_permiso")
}

func TestEstacionBajaYReglas(t *testing.T) {
	ctx := context.Background()
	admin := actorEstaciones(appctx.Grant{SucursalID: 3, Permiso: policy.EstacionAdministrar})
	st := &storeEstaciones{est: dtoresult.Estacion{ID: 1, SucursalID: 3, Codigo: "COC", Nombre: "Cocina", EsActivo: true}, enUso: true}

	res, err := NewCambiarEstadoEstacion(st).Execute(ctx, cmd.CambiarEstadoEstacion{Actor: admin, ID: 1})
	esperaRechazo(t, err, res.Err(), apperror.KindConflict, CodigoEstacionEnUso)
	if st.escrituras != 0 {
		t.Fatal("la baja bloqueada no debe escribir")
	}
	st.enUso = false
	if res, err = NewCambiarEstadoEstacion(st).Execute(ctx, cmd.CambiarEstadoEstacion{Actor: admin, ID: 1}); err != nil || res.Err() != nil {
		t.Fatalf("baja libre: %v %v", err, res.Err())
	}

	// Editar una inactiva y sin cambios.
	st.est.EsActivo = false
	n := "Otro"
	res2, err := NewEditarEstacion(st).Execute(ctx, cmd.EditarEstacion{Actor: admin, ID: 1, Nombre: &n})
	esperaRechazo(t, err, res2.Err(), apperror.KindConflict, CodigoEstacionInactiva)
	st.est.EsActivo = true
	res2, err = NewEditarEstacion(st).Execute(ctx, cmd.EditarEstacion{Actor: admin, ID: 1})
	esperaRechazo(t, err, res2.Err(), apperror.KindValidation, "sin_cambios")

	// El trigger de sede inactiva se traduce a un mensaje curado.
	st.crearErr = &outbound.ErrReglaDatos{Mensaje: "Referencia sucursal_id=3: maestro sucursales inexistente o dado de baja"}
	res, err = NewCrearEstacion(st).Execute(ctx, cmd.CrearEstacion{Actor: admin, SucursalID: 3, Codigo: "x", Nombre: "X"})
	esperaRechazo(t, err, res.Err(), apperror.KindValidation, "referencia_invalida")
	if res.Err().Message != mensajeSedeNoValida {
		t.Errorf("mensaje no curado: %v", res.Err())
	}
}
