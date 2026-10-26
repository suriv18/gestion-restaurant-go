package command

import (
	"context"
	"errors"
	"testing"

	cmd "service-restaurant/internal/modules/catalogo/application/dto/command"
	dtoresult "service-restaurant/internal/modules/catalogo/application/dto/result"
	"service-restaurant/internal/modules/catalogo/application/port/outbound"
	"service-restaurant/internal/modules/catalogo/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// storeDisp es un DisponibilidadStore en memoria de un solo producto; registra los guardados.
type storeDisp struct {
	producto   dtoresult.Producto
	existe     bool
	guardarErr error
	guardados  []outbound.CambiosDisponibilidad
}

func (s *storeDisp) BloquearProducto(context.Context, int64) (dtoresult.Producto, bool, error) {
	return s.producto, s.existe, nil
}
func (s *storeDisp) Guardar(_ context.Context, p, sede int64, c outbound.CambiosDisponibilidad) (dtoresult.ProductoSucursal, error) {
	if s.guardarErr != nil {
		return dtoresult.ProductoSucursal{}, s.guardarErr
	}
	s.guardados = append(s.guardados, c)
	return dtoresult.ProductoSucursal{ID: 1, ProductoID: p, SucursalID: sede, Disponible: c.Disponible, EsActivo: true}, nil
}

func nuevoStoreDisp() *storeDisp {
	return &storeDisp{producto: dtoresult.Producto{ID: 5, EsActivo: true}, existe: true}
}

func si() *bool { v := true; return &v }

// Los 403 sede_no_autorizada de la disponibilidad no se alcanzan por HTTP (v_permisos_efectivos
// cruza roles x sedes): se cubren aquí a nivel de caso de uso.
func TestDisponibilidadAutorizacionPorSede(t *testing.T) {
	ctx := context.Background()
	// Pausa en la sede 2, solo ve la 3, no tiene nada en la 4 y administra la 5.
	a := appctx.NewActor(7, "ana", 1, []appctx.Grant{
		{SucursalID: 2, Permiso: policy.CartaDisponibilidad},
		{SucursalID: 3, Permiso: policy.CartaVer},
		{SucursalID: 5, Permiso: policy.CartaAdministrar},
	})
	precio := "10"
	estacion := int64(9)
	casos := []struct {
		nombre string
		c      cmd.CambiarDisponibilidad
		kind   apperror.Kind
		code   string
	}{
		{"sede que no ve", cmd.CambiarDisponibilidad{SucursalID: 4, Disponible: si()}, apperror.KindNotFound, "no_encontrado"},
		{"ve sin poder pausar", cmd.CambiarDisponibilidad{SucursalID: 3, Disponible: si()}, apperror.KindForbidden, "sede_no_autorizada"},
		{"pausar no basta para precio", cmd.CambiarDisponibilidad{SucursalID: 2, Disponible: si(),
			Precio: cmd.Campo[string]{Presente: true, Valor: &precio}}, apperror.KindForbidden, "sede_no_autorizada"},
		{"pausar no basta para vaciar precio", cmd.CambiarDisponibilidad{SucursalID: 2, Disponible: si(),
			Precio: cmd.Campo[string]{Presente: true}}, apperror.KindForbidden, "sede_no_autorizada"},
		{"pausar no basta para estación", cmd.CambiarDisponibilidad{SucursalID: 2, Disponible: si(),
			Estacion: cmd.Campo[int64]{Presente: true, Valor: &estacion}}, apperror.KindForbidden, "sede_no_autorizada"},
	}
	for _, tc := range casos {
		st := nuevoStoreDisp()
		tc.c.Actor, tc.c.ProductoID = a, 5
		res, err := NewCambiarDisponibilidad(st).Execute(ctx, tc.c)
		esperaRechazo(t, err, res.Err(), tc.kind, tc.code)
		if len(st.guardados) != 0 {
			t.Errorf("%s: un rechazo de autorización no debe escribir", tc.nombre)
		}
	}

	// Autorizados: pausar donde tiene CARTA_DISPONIBILIDAD; precio y estación donde administra.
	st := nuevoStoreDisp()
	res, err := NewCambiarDisponibilidad(st).Execute(ctx, cmd.CambiarDisponibilidad{Actor: a, ProductoID: 5, SucursalID: 2, Disponible: si()})
	if err != nil || res.Err() != nil {
		t.Fatalf("pausar con CARTA_DISPONIBILIDAD: %v %v", err, res.Err())
	}
	res, err = NewCambiarDisponibilidad(st).Execute(ctx, cmd.CambiarDisponibilidad{Actor: a, ProductoID: 5, SucursalID: 5, Disponible: si(),
		Precio: cmd.Campo[string]{Presente: true, Valor: &precio}, Estacion: cmd.Campo[int64]{Presente: true, Valor: &estacion}})
	if err != nil || res.Err() != nil {
		t.Fatalf("precio y estación con CARTA_ADMINISTRAR: %v %v", err, res.Err())
	}
	c := st.guardados[1]
	if !c.Precio.Presente || c.Precio.Valor == nil || !c.Estacion.Presente || c.Estacion.Valor == nil || *c.Estacion.Valor != 9 {
		t.Errorf("cambios entregados al almacén: %+v", c)
	}
	if g := st.guardados[0]; g.Precio.Presente || g.Estacion.Presente {
		t.Errorf("solo disponible: precio y estación deben quedar ausentes: %+v", g)
	}

	// Sin ningún permiso de disponibilidad ni administración → 403 sin_permiso.
	soloVer := appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: 3, Permiso: policy.CartaVer}})
	res, err = NewCambiarDisponibilidad(st).Execute(ctx, cmd.CambiarDisponibilidad{Actor: soloVer, ProductoID: 5, SucursalID: 3, Disponible: si()})
	esperaRechazo(t, err, res.Err(), apperror.KindForbidden, "sin_permiso")
}

func TestDisponibilidadValidacionesYEstadoDelProducto(t *testing.T) {
	ctx := context.Background()
	admin := appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: 3, Permiso: policy.CartaAdministrar}})
	base := cmd.CambiarDisponibilidad{Actor: admin, ProductoID: 5, SucursalID: 3, Disponible: si()}

	st := nuevoStoreDisp()
	uc := NewCambiarDisponibilidad(st)
	c := base
	c.SucursalID = 0
	res, err := uc.Execute(ctx, c)
	esperaRechazo(t, err, res.Err(), apperror.KindValidation, "sucursal_requerida")
	c = base
	c.Disponible = nil
	res, err = uc.Execute(ctx, c)
	esperaRechazo(t, err, res.Err(), apperror.KindValidation, "disponible_requerido")
	for _, malo := range []string{"-1", "1.234", "abc", "", "1e3", "100000000000"} {
		c = base
		c.Precio = cmd.Campo[string]{Presente: true, Valor: &malo}
		res, err = uc.Execute(ctx, c)
		esperaRechazo(t, err, res.Err(), apperror.KindValidation, "precio_invalido")
	}
	cero := int64(0)
	c = base
	c.Estacion = cmd.Campo[int64]{Presente: true, Valor: &cero}
	res, err = uc.Execute(ctx, c)
	esperaRechazo(t, err, res.Err(), apperror.KindValidation, "estacion_invalida")
	if len(st.guardados) != 0 {
		t.Fatal("una validación fallida no debe escribir")
	}

	// Producto inexistente → 404; dado de baja → 409 producto_inactivo.
	st.existe = false
	res, err = uc.Execute(ctx, base)
	esperaRechazo(t, err, res.Err(), apperror.KindNotFound, "no_encontrado")
	st.existe, st.producto.EsActivo = true, false
	res, err = uc.Execute(ctx, base)
	esperaRechazo(t, err, res.Err(), apperror.KindConflict, CodigoProductoInactivo)
	if len(st.guardados) != 0 {
		t.Fatal("un producto inexistente o dado de baja no debe escribir")
	}
}

func TestDisponibilidadTraduceErroresDeAlmacenamiento(t *testing.T) {
	ctx := context.Background()
	admin := appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: 3, Permiso: policy.CartaAdministrar}})
	base := cmd.CambiarDisponibilidad{Actor: admin, ProductoID: 5, SucursalID: 3, Disponible: si()}
	casos := []struct {
		err     error
		kind    apperror.Kind
		code    string
		mensaje string
	}{
		{outbound.ErrEstacionOtraSede, apperror.KindValidation, "referencia_invalida", mensajeEstacionNoValida},
		{&outbound.ErrReglaDatos{Mensaje: "Referencia estacion_id=8: maestro estaciones_preparacion inexistente o dado de baja"},
			apperror.KindValidation, "referencia_invalida", mensajeEstacionNoValida},
		{&outbound.ErrReglaDatos{Mensaje: "Referencia sucursal_id=3: maestro sucursales inexistente o dado de baja"},
			apperror.KindValidation, "referencia_invalida", mensajeSedeNoValida},
	}
	for _, tc := range casos {
		st := nuevoStoreDisp()
		st.guardarErr = tc.err
		res, err := NewCambiarDisponibilidad(st).Execute(ctx, base)
		esperaRechazo(t, err, res.Err(), tc.kind, tc.code)
		if res.Err().Message != tc.mensaje {
			t.Errorf("mensaje = %q, quería %q", res.Err().Message, tc.mensaje)
		}
	}
	// Un error técnico no es un rechazo.
	st := nuevoStoreDisp()
	st.guardarErr = errors.New("conexión caída")
	if _, err := NewCambiarDisponibilidad(st).Execute(ctx, base); err == nil {
		t.Fatal("el error técnico debe salir por el canal error")
	}
}
