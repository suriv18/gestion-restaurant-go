package command

import (
	"context"
	"errors"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type zonaStoreFake struct {
	existente     dtoresult.Zona
	mesas         int
	errCrear      error
	creada        outbound.NuevaZona
	cambios       outbound.CambiosZona
	cambiosHechos int
}

func (f *zonaStoreFake) Crear(_ context.Context, n outbound.NuevaZona) (dtoresult.Zona, error) {
	f.creada = n
	return dtoresult.Zona{ID: 3, SucursalID: n.SucursalID, Nombre: n.Nombre, EsActivo: true}, f.errCrear
}
func (f *zonaStoreFake) Obtener(_ context.Context, id int64) (dtoresult.Zona, bool, error) {
	return f.existente, f.existente.ID == id, nil
}
func (f *zonaStoreFake) Actualizar(_ context.Context, _ int64, c outbound.CambiosZona) (dtoresult.Zona, bool, error) {
	f.cambios = c
	return f.existente, true, nil
}
func (f *zonaStoreFake) ContarMesasActivas(context.Context, int64) (int, error) { return f.mesas, nil }
func (f *zonaStoreFake) CambiarActivo(_ context.Context, _ int64, activo bool) (dtoresult.Zona, bool, error) {
	f.cambiosHechos++
	z := f.existente
	z.EsActivo = activo
	return z, true, nil
}

func gestorDe(sedes ...int64) appctx.Actor {
	var g []appctx.Grant
	for _, s := range sedes {
		g = append(g, appctx.Grant{SucursalID: s, Permiso: policy.MesaAdministrar})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func soloVer(sede int64) appctx.Actor {
	return appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: sede, Permiso: policy.MesaVer}})
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

func entero(n int) *int      { return &n }
func texto(s string) *string { return &s }

func TestCrearZona(t *testing.T) {
	ok := cmd.CrearZona{Actor: gestorDe(1), SucursalID: 1, Nombre: "  Terraza  ", OrdenVisualizacion: entero(2)}
	f := &zonaStoreFake{}
	res, err := NewCrearZona(f).Execute(context.Background(), ok)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía crearse: %v %+v", err, res.Err())
	}
	if f.creada.Nombre != "Terraza" || f.creada.Orden != 2 || f.creada.SucursalID != 1 {
		t.Fatalf("no normalizó: %+v", f.creada)
	}
	_, _ = NewCrearZona(f).Execute(context.Background(), cmd.CrearZona{Actor: gestorDe(1), SucursalID: 1, Nombre: "Sin orden"})
	if f.creada.Orden != 0 {
		t.Errorf("orden por defecto = %d, quería 0", f.creada.Orden)
	}

	casos := []struct {
		nombre string
		mod    func(c *cmd.CrearZona)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(c *cmd.CrearZona) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(c *cmd.CrearZona) { c.Actor = soloVer(1) }, apperror.KindForbidden, "sin_permiso"},
		{"sede ajena", func(c *cmd.CrearZona) { c.SucursalID = 9 }, apperror.KindNotFound, "no_encontrado"},
		{"nombre en blanco", func(c *cmd.CrearZona) { c.Nombre = "  " }, apperror.KindValidation, "nombre_invalido"},
		{"orden fuera de rango", func(c *cmd.CrearZona) { c.OrdenVisualizacion = entero(10000) }, apperror.KindValidation, "orden_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			in := ok
			c.mod(&in)
			res, err := NewCrearZona(&zonaStoreFake{}).Execute(context.Background(), in)
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}
	t.Run("nombre duplicado", func(t *testing.T) {
		res, _ := NewCrearZona(&zonaStoreFake{errCrear: outbound.ErrNombreZonaDuplicado}).Execute(context.Background(), ok)
		esperarRechazo(t, res.Err(), apperror.KindConflict, "nombre_duplicado")
	})
	t.Run("error técnico", func(t *testing.T) {
		if _, err := NewCrearZona(&zonaStoreFake{errCrear: errors.New("conexión perdida")}).Execute(context.Background(), ok); err == nil {
			t.Fatal("un error técnico debe viajar por el canal error")
		}
	})
}

func TestEditarZona(t *testing.T) {
	activa := dtoresult.Zona{ID: 5, SucursalID: 1, EsActivo: true}
	edit := func(a appctx.Actor) cmd.EditarZona {
		return cmd.EditarZona{Actor: a, ID: 5, Nombre: texto("  Nueva  ")}
	}

	res, _ := NewEditarZona(&zonaStoreFake{existente: activa}).Execute(context.Background(), edit(gestorDe(9)))
	esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado") // sede 1 no es del actor
	res, _ = NewEditarZona(&zonaStoreFake{existente: activa}).Execute(context.Background(),
		cmd.EditarZona{Actor: appctx.NewActor(7, "a", 1, []appctx.Grant{
			{SucursalID: 1, Permiso: policy.MesaVer}, {SucursalID: 9, Permiso: policy.MesaAdministrar}}), ID: 5, Nombre: texto("x")})
	esperarRechazo(t, res.Err(), apperror.KindForbidden, "sede_no_autorizada")
	res, _ = NewEditarZona(&zonaStoreFake{existente: dtoresult.Zona{ID: 5, SucursalID: 1}}).Execute(context.Background(), edit(gestorDe(1)))
	esperarRechazo(t, res.Err(), apperror.KindConflict, "zona_inactiva")

	f := &zonaStoreFake{existente: activa}
	res, err := NewEditarZona(f).Execute(context.Background(), cmd.EditarZona{Actor: gestorDe(1), ID: 5, Nombre: texto("  Nueva  "), OrdenVisualizacion: entero(4)})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	if *f.cambios.Nombre != "Nueva" || *f.cambios.Orden != 4 {
		t.Fatalf("cambios: %+v", f.cambios)
	}
	res, _ = NewEditarZona(f).Execute(context.Background(), cmd.EditarZona{Actor: gestorDe(1), ID: 5})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "sin_cambios")
	res, _ = NewEditarZona(f).Execute(context.Background(), cmd.EditarZona{Actor: gestorDe(1), ID: 5, Nombre: texto(" ")})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "nombre_invalido")
}

func TestCambiarEstadoZona(t *testing.T) {
	baja := func(a appctx.Actor) cmd.CambiarEstadoZona {
		return cmd.CambiarEstadoZona{Actor: a, ID: 5, Activo: false}
	}
	activa := dtoresult.Zona{ID: 5, SucursalID: 1, EsActivo: true}

	t.Run("con mesas activas no se da de baja", func(t *testing.T) {
		f := &zonaStoreFake{existente: activa, mesas: 2}
		res, _ := NewCambiarEstadoZona(f).Execute(context.Background(), baja(gestorDe(1)))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "zona_con_mesas")
		if f.cambiosHechos != 0 {
			t.Fatal("no debe cambiar el estado")
		}
	})
	t.Run("baja sin mesas", func(t *testing.T) {
		f := &zonaStoreFake{existente: activa}
		res, err := NewCambiarEstadoZona(f).Execute(context.Background(), baja(gestorDe(1)))
		if err != nil || !res.IsOk() || res.Value().EsActivo || f.cambiosHechos != 1 {
			t.Fatalf("baja: %v %+v", err, res.Err())
		}
	})
	t.Run("ya estaba de baja", func(t *testing.T) {
		f := &zonaStoreFake{existente: dtoresult.Zona{ID: 5, SucursalID: 1}}
		res, _ := NewCambiarEstadoZona(f).Execute(context.Background(), baja(gestorDe(1)))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_dada_de_baja")
	})
	t.Run("restaurar", func(t *testing.T) {
		f := &zonaStoreFake{existente: dtoresult.Zona{ID: 5, SucursalID: 1}}
		res, err := NewCambiarEstadoZona(f).Execute(context.Background(), cmd.CambiarEstadoZona{Actor: gestorDe(1), ID: 5, Activo: true})
		if err != nil || !res.IsOk() || !res.Value().EsActivo {
			t.Fatalf("restaurar: %v %+v", err, res.Err())
		}
		f = &zonaStoreFake{existente: activa}
		res, _ = NewCambiarEstadoZona(f).Execute(context.Background(), cmd.CambiarEstadoZona{Actor: gestorDe(1), ID: 5, Activo: true})
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_activa")
	})
	t.Run("sede ajena", func(t *testing.T) {
		res, _ := NewCambiarEstadoZona(&zonaStoreFake{existente: activa}).Execute(context.Background(), baja(gestorDe(9)))
		esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	})
}
