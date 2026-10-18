package query_test

import (
	"context"
	"testing"
	"time"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/application/usecase/query"
	"service-restaurant/internal/modules/identidad/domain/model"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
)

var t0 = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type fakeLector struct {
	sesion outbound.SesionAutenticada
	found  bool
	grants []appctx.Grant
}

func (f *fakeLector) BuscarPorID(context.Context, int64) (outbound.SesionAutenticada, bool, error) {
	return f.sesion, f.found, nil
}
func (f *fakeLector) PermisosEfectivos(context.Context, int64) ([]appctx.Grant, error) {
	return f.grants, nil
}

type fakeEscritor struct{ usos int }

func (f *fakeEscritor) Crear(context.Context, model.Sesion) (model.Sesion, error) {
	return model.Sesion{}, nil
}
func (f *fakeEscritor) Revocar(context.Context, int64, time.Time) error  { return nil }
func (f *fakeEscritor) RegistrarUso(context.Context, model.Sesion) error { f.usos++; return nil }

type fakeTokens struct {
	claims outbound.TokenClaims
	ok     bool
}

func (f fakeTokens) Emitir(outbound.TokenClaims, time.Time, time.Time) (string, error) {
	return "", nil
}
func (f fakeTokens) Verificar(string, time.Time) (outbound.TokenClaims, bool) { return f.claims, f.ok }

func nueva() (*query.AutenticarToken, *fakeLector, *fakeEscritor, *fakeClock, *fakeTokens) {
	ses := policy.SesionPorDefecto.Nueva(7, t0)
	ses.ID = 50
	lector := &fakeLector{
		sesion: outbound.SesionAutenticada{Sesion: ses, Username: "ana"},
		found:  true,
		grants: []appctx.Grant{{SucursalID: 1, Permiso: "PEDIDO_CREAR"}},
	}
	esc := &fakeEscritor{}
	clock := &fakeClock{now: t0.Add(time.Second)}
	tok := &fakeTokens{claims: outbound.TokenClaims{SesionID: 50, UsuarioID: 7}, ok: true}
	return query.NewAutenticarToken(lector, esc, tok, clock, policy.SesionPorDefecto), lector, esc, clock, tok
}

func TestAutenticaConTokenYSesionVigentes(t *testing.T) {
	uc, _, esc, _, _ := nueva()
	res, err := uc.Execute(context.Background(), qry.AutenticarToken{Token: "x"})
	if err != nil || !res.IsOk() {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	a := res.Value().Actor
	if a.UserID != 7 || a.Username != "ana" || a.SesionID != 50 || !a.Can("PEDIDO_CREAR", 1) || a.Can("PEDIDO_CREAR", 2) {
		t.Fatalf("actor incorrecto: %+v", a)
	}
	if esc.usos != 0 {
		t.Fatal("no registra uso antes de un minuto")
	}
}

func TestRegistraUsoAlPasarElIntervalo(t *testing.T) {
	uc, _, esc, clock, _ := nueva()
	clock.now = t0.Add(2 * time.Minute)
	if res, _ := uc.Execute(context.Background(), qry.AutenticarToken{}); !res.IsOk() || esc.usos != 1 {
		t.Fatalf("ok=%v usos=%d", res.IsOk(), esc.usos)
	}
}

func TestRechazos(t *testing.T) {
	revocada := t0
	casos := map[string]func(*fakeLector, *fakeClock, *fakeTokens){
		"firma o vencimiento del jwt inválidos":   func(_ *fakeLector, _ *fakeClock, t *fakeTokens) { t.ok = false },
		"sesión inexistente (o usuario inactivo)": func(l *fakeLector, _ *fakeClock, _ *fakeTokens) { l.found = false },
		"sesión de otro usuario":                  func(_ *fakeLector, _ *fakeClock, t *fakeTokens) { t.claims.UsuarioID = 99 },
		"sesión revocada":                         func(l *fakeLector, _ *fakeClock, _ *fakeTokens) { l.sesion.Sesion.RevocadaEn = &revocada },
		"inactividad vencida":                     func(_ *fakeLector, c *fakeClock, _ *fakeTokens) { c.now = t0.Add(3 * time.Hour) },
		"plazo absoluto vencido":                  func(_ *fakeLector, c *fakeClock, _ *fakeTokens) { c.now = t0.Add(13 * time.Hour) },
	}
	for name, mutar := range casos {
		uc, l, _, c, tk := nueva()
		mutar(l, c, tk)
		res, err := uc.Execute(context.Background(), qry.AutenticarToken{Token: "x"})
		if err != nil || res.IsOk() || res.Err() == nil || res.Err().Code != "no_autenticado" {
			t.Errorf("%s: res=%+v err=%v", name, res, err)
		}
	}
}
