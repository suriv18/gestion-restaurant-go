package command_test

import (
	"context"
	"testing"
	"time"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/application/usecase/command"
	"service-restaurant/internal/modules/identidad/domain/model"
	"service-restaurant/internal/modules/identidad/domain/policy"
)

var t0 = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type fakeUsuarios struct {
	byKey      map[string]*model.Usuario
	guardados  int
	accesos    int
	actorTrace *[]int64
}

func (f *fakeUsuarios) BuscarPorUsernameParaActualizar(_ context.Context, key string) (model.Usuario, bool, error) {
	u, ok := f.byKey[key]
	if !ok {
		return model.Usuario{}, false, nil
	}
	return *u, true, nil
}
func (f *fakeUsuarios) GuardarIntentos(_ context.Context, u model.Usuario) error {
	f.guardados++
	*f.byKey[u.Username] = u
	return nil
}
func (f *fakeUsuarios) RegistrarAcceso(_ context.Context, u model.Usuario, _ time.Time) error {
	f.accesos++
	*f.byKey[u.Username] = u
	return nil
}

type fakeActor struct{ fijados []int64 }

func (f *fakeActor) Fijar(_ context.Context, id int64) error {
	f.fijados = append(f.fijados, id)
	return nil
}
func (f *fakeActor) UsuarioSistemaID(context.Context) (int64, error) { return 1, nil }

type fakeSesiones struct{ creadas []model.Sesion }

func (f *fakeSesiones) Crear(_ context.Context, s model.Sesion) (model.Sesion, error) {
	s.ID = int64(len(f.creadas) + 100)
	f.creadas = append(f.creadas, s)
	return s, nil
}
func (f *fakeSesiones) Revocar(context.Context, int64, time.Time) error  { return nil }
func (f *fakeSesiones) RegistrarUso(context.Context, model.Sesion) error { return nil }

// fakeHasher: la "contraseña" correcta es el propio hash sin el prefijo.
type fakeHasher struct{ dummies int }

func (f *fakeHasher) Hash(p string) (string, error)    { return "h:" + p, nil }
func (f *fakeHasher) Verify(p, h string) (bool, error) { return h == "h:"+p, nil }
func (f *fakeHasher) Dummy(string)                     { f.dummies++ }

type fakeTokens struct{}

func (fakeTokens) Emitir(c outbound.TokenClaims, _, _ time.Time) (string, error) {
	return "jwt-sid", nil
}
func (fakeTokens) Verificar(string, time.Time) (outbound.TokenClaims, bool) {
	return outbound.TokenClaims{}, false
}

type harness struct {
	uc       *command.IniciarSesion
	usuarios *fakeUsuarios
	actor    *fakeActor
	sesiones *fakeSesiones
	hasher   *fakeHasher
	clock    *fakeClock
}

func newHarness() *harness {
	h := &harness{
		usuarios: &fakeUsuarios{byKey: map[string]*model.Usuario{
			"ana":         {ID: 7, Username: "ana", PasswordHash: "h:Correcta12345", EsActivo: true},
			"baja":        {ID: 8, Username: "baja", PasswordHash: "h:Correcta12345", EsActivo: false},
			"__sistema__": {ID: 1, Username: "__sistema__", EsSistema: true, EsActivo: true},
		}},
		actor: &fakeActor{}, sesiones: &fakeSesiones{}, hasher: &fakeHasher{}, clock: &fakeClock{now: t0},
	}
	h.uc = command.NewIniciarSesion(h.usuarios, h.actor, h.sesiones, h.hasher, fakeTokens{}, h.clock,
		policy.BloqueoPorDefecto, policy.SesionPorDefecto)
	return h
}

func TestLoginCorrecto(t *testing.T) {
	h := newHarness()
	res, err := h.uc.Execute(context.Background(), cmd.IniciarSesion{Username: " ANA ", Password: "Correcta12345"})
	if err != nil || !res.IsOk() {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if res.Value().AccessToken != "jwt-sid" || res.Value().ExpiresIn != 7200 {
		t.Fatalf("respuesta incorrecta: %+v", res.Value())
	}
	if len(h.sesiones.creadas) != 1 || h.sesiones.creadas[0].UsuarioID != 7 {
		t.Fatal("debe crear una sesión para el usuario")
	}
	if h.usuarios.accesos != 1 {
		t.Fatal("debe registrar el acceso")
	}
	if len(h.actor.fijados) != 1 || h.actor.fijados[0] != 7 {
		t.Fatalf("el actor del acceso es el propio usuario: %v", h.actor.fijados)
	}
}

// Cada causa de rechazo produce exactamente el mismo resultado.
func TestRechazosSonIndistinguibles(t *testing.T) {
	h := newHarness()
	bloqueado := t0.Add(time.Hour)
	h.usuarios.byKey["bloq"] = &model.Usuario{ID: 9, Username: "bloq", PasswordHash: "h:Correcta12345", EsActivo: true, BloqueadoHasta: &bloqueado}
	casos := []cmd.IniciarSesion{
		{Username: "ana", Password: "incorrecta"},
		{Username: "noexiste", Password: "Correcta12345"},
		{Username: "baja", Password: "Correcta12345"},
		{Username: "__sistema__", Password: "Correcta12345"},
		{Username: "bloq", Password: "Correcta12345"},
		{Username: "", Password: "Correcta12345"},
		{Username: "ana", Password: ""},
	}
	for _, c := range casos {
		res, err := h.uc.Execute(context.Background(), c)
		if err != nil || res.IsOk() || !res.IsValid() {
			t.Fatalf("%+v: res=%+v err=%v", c, res, err)
		}
		if got := res.Err(); got.Code != mapper.CodigoCredencialesInvalidas || got.Message != "credenciales inválidas" {
			t.Fatalf("%+v: rechazo distinguible: %+v", c, got)
		}
	}
	if len(h.sesiones.creadas) != 0 {
		t.Fatal("ningún rechazo crea sesión")
	}
}

func TestBloqueoTrasCincoFallos(t *testing.T) {
	h := newHarness()
	for i := 0; i < 5; i++ {
		_, _ = h.uc.Execute(context.Background(), cmd.IniciarSesion{Username: "ana", Password: "mala"})
	}
	u := h.usuarios.byKey["ana"]
	if u.IntentosFallidos != 5 || u.BloqueadoHasta == nil {
		t.Fatalf("debe quedar bloqueado: %+v", u)
	}
	for _, id := range h.actor.fijados {
		if id != 1 {
			t.Fatalf("los fallos los registra el actor técnico, no %d", id)
		}
	}
	// Bloqueado: ni la contraseña correcta entra, y no suma intentos.
	res, _ := h.uc.Execute(context.Background(), cmd.IniciarSesion{Username: "ana", Password: "Correcta12345"})
	if res.IsOk() || u.IntentosFallidos != 5 {
		t.Fatalf("bloqueado debe rechazar sin sumar: ok=%v intentos=%d", res.IsOk(), u.IntentosFallidos)
	}
	// Vencido el bloqueo, vuelve a entrar y se limpia el contador.
	h.clock.now = t0.Add(16 * time.Minute)
	res, _ = h.uc.Execute(context.Background(), cmd.IniciarSesion{Username: "ana", Password: "Correcta12345"})
	if !res.IsOk() || u.IntentosFallidos != 0 || u.BloqueadoHasta != nil {
		t.Fatalf("tras el bloqueo debe poder entrar: ok=%v %+v", res.IsOk(), u)
	}
}

func TestRechazoSinUsuarioGastaTiempoDeHash(t *testing.T) {
	h := newHarness()
	_, _ = h.uc.Execute(context.Background(), cmd.IniciarSesion{Username: "noexiste", Password: "x"})
	_, _ = h.uc.Execute(context.Background(), cmd.IniciarSesion{Username: "baja", Password: "x"})
	if h.hasher.dummies != 2 {
		t.Fatalf("dummies = %d, quería 2", h.hasher.dummies)
	}
}
