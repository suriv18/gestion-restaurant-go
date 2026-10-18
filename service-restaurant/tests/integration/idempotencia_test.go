package integration

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"service-restaurant/internal/platform/idempotency"
	"service-restaurant/internal/platform/transaction"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type item struct {
	N int `json:"n"`
}

const (
	k1 = "11111111-1111-4111-8111-111111111111"
	k2 = "22222222-2222-4222-8222-222222222222"
	k3 = "33333333-3333-4333-8333-333333333333"
	k4 = "44444444-4444-4444-8444-444444444444"
	k5 = "55555555-5555-4555-8555-555555555555"
	k6 = "66666666-6666-4666-8666-666666666666"
)

func prepararIdem(t *testing.T) (usuarioID int64) {
	t.Helper()
	app(t) // asegura el administrador inicial
	if _, err := testDB.Exec(`CREATE TABLE IF NOT EXISTS public.idem_efectos (id serial PRIMARY KEY, clave text)`); err != nil {
		t.Fatal(err)
	}
	if err := testDB.QueryRow(`SELECT id FROM sch_restaurante.usuarios WHERE username='admin'`).Scan(&usuarioID); err != nil {
		t.Fatal(err)
	}
	return usuarioID
}

func req(clave string, usuario int64, cuerpo string) idempotency.Request {
	return idempotency.Request{
		Key: clave, UsuarioID: usuario, Operacion: "POST /prueba",
		Huella: idempotency.Fingerprint("POST /prueba", []byte(cuerpo)),
		Status: 201, Now: time.Now().UTC(),
	}
}

// efecto inserta una fila y devuelve cuántas hay para esa clave (el "efecto" observable).
func efecto(clave string, ejecuciones *atomic.Int32) func(context.Context, *sql.Tx) (appresult.Result[item], error) {
	return func(ctx context.Context, tx *sql.Tx) (appresult.Result[item], error) {
		ejecuciones.Add(1)
		if _, err := tx.ExecContext(ctx, `INSERT INTO public.idem_efectos(clave) VALUES ($1)`, clave); err != nil {
			return appresult.Result[item]{}, err
		}
		return appresult.Ok(item{N: int(ejecuciones.Load())}), nil
	}
}

func efectos(t *testing.T, clave string) int {
	return scalar(t, `SELECT count(*) FROM public.idem_efectos WHERE clave = $1`, clave)
}

func TestIdempotenciaRepiteElMismoEfecto(t *testing.T) {
	u := prepararIdem(t)
	var n atomic.Int32
	r := req(k1, u, `{"x":1}`)

	o1, err := idempotency.Run(context.Background(), testDB, discard, r, efecto(k1, &n))
	if err != nil || !o1.Result.IsOk() || o1.Replayed || o1.Status != 201 {
		t.Fatalf("primera: %+v err=%v", o1, err)
	}
	o2, err := idempotency.Run(context.Background(), testDB, discard, r, efecto(k1, &n))
	if err != nil || !o2.Result.IsOk() || !o2.Replayed || o2.Status != 201 {
		t.Fatalf("repetición: %+v err=%v", o2, err)
	}
	if o2.Result.Value() != o1.Result.Value() {
		t.Fatalf("debe repetir el mismo resultado: %v vs %v", o2.Result.Value(), o1.Result.Value())
	}
	if n.Load() != 1 || efectos(t, k1) != 1 {
		t.Fatalf("el efecto debe ocurrir una sola vez: ejecuciones=%d filas=%d", n.Load(), efectos(t, k1))
	}
}

func TestIdempotenciaContenidoDistintoEsConflicto(t *testing.T) {
	u := prepararIdem(t)
	var n atomic.Int32
	if _, err := idempotency.Run(context.Background(), testDB, discard, req(k2, u, `{"x":1}`), efecto(k2, &n)); err != nil {
		t.Fatal(err)
	}
	o, err := idempotency.Run(context.Background(), testDB, discard, req(k2, u, `{"x":2}`), efecto(k2, &n))
	if err != nil || o.Result.IsOk() || o.Result.Err() == nil ||
		o.Result.Err().Kind != apperror.KindConflict || o.Result.Err().Code != "idempotency_key_reutilizada" {
		t.Fatalf("debe ser conflicto: %+v err=%v", o, err)
	}
	if n.Load() != 1 || efectos(t, k2) != 1 {
		t.Fatal("el conflicto no debe ejecutar de nuevo")
	}
}

// Un rechazo (422) hace rollback y no deja rastro: el reintento se evalúa de nuevo.
func TestIdempotenciaRechazoNoDejaRastro(t *testing.T) {
	u := prepararIdem(t)
	var n atomic.Int32
	rechazo := func(ctx context.Context, tx *sql.Tx) (appresult.Result[item], error) {
		n.Add(1)
		_, _ = tx.ExecContext(ctx, `INSERT INTO public.idem_efectos(clave) VALUES ($1)`, k3)
		return appresult.Fail[item](apperror.New(apperror.KindValidation, "regla", "no cumple")), nil
	}
	o, err := idempotency.Run(context.Background(), testDB, discard, req(k3, u, `{}`), rechazo)
	if err != nil || o.Result.IsOk() || o.Result.Err().Code != "regla" {
		t.Fatalf("o=%+v err=%v", o, err)
	}
	if efectos(t, k3) != 0 || scalar(t, `SELECT count(*) FROM sch_restaurante.idempotencia WHERE clave=$1::uuid`, k3) != 0 {
		t.Fatal("el rechazo no debe dejar efecto ni registro")
	}
	// Reintento corregido: ahora se ejecuta y se confirma.
	o, err = idempotency.Run(context.Background(), testDB, discard, req(k3, u, `{}`), efecto(k3, &n))
	if err != nil || !o.Result.IsOk() || o.Replayed || efectos(t, k3) != 1 {
		t.Fatalf("reintento: %+v err=%v filas=%d", o, err, efectos(t, k3))
	}
}

func TestIdempotenciaErrorTecnicoHaceRollback(t *testing.T) {
	u := prepararIdem(t)
	boom := errors.New("fallo técnico")
	_, err := idempotency.Run(context.Background(), testDB, discard, req(k4, u, `{}`),
		func(ctx context.Context, tx *sql.Tx) (appresult.Result[item], error) {
			_, _ = tx.ExecContext(ctx, `INSERT INTO public.idem_efectos(clave) VALUES ($1)`, k4)
			return appresult.Result[item]{}, boom
		})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if efectos(t, k4) != 0 || scalar(t, `SELECT count(*) FROM sch_restaurante.idempotencia WHERE clave=$1::uuid`, k4) != 0 {
		t.Fatal("un error técnico no debe dejar efecto ni registro")
	}
}

func TestIdempotenciaCaducadaSeReejecuta(t *testing.T) {
	u := prepararIdem(t)
	var n atomic.Int32
	r := req(k5, u, `{}`)
	r.Now = time.Now().UTC().Add(-48 * time.Hour) // se registró hace dos días (TTL 24 h)
	if _, err := idempotency.Run(context.Background(), testDB, discard, r, efecto(k5, &n)); err != nil {
		t.Fatal(err)
	}
	o, err := idempotency.Run(context.Background(), testDB, discard, req(k5, u, `{}`), efecto(k5, &n))
	if err != nil || !o.Result.IsOk() || o.Replayed || n.Load() != 2 {
		t.Fatalf("caducada debe reejecutarse: %+v err=%v n=%d", o, err, n.Load())
	}
}

func TestIdempotenciaEstaAcotadaPorUsuario(t *testing.T) {
	u := prepararIdem(t)
	crearUsuario(t, "idem2", "IdemClave2026Ok", "CAJERO")
	var u2 int64
	if err := testDB.QueryRow(`SELECT id FROM sch_restaurante.usuarios WHERE username='idem2'`).Scan(&u2); err != nil {
		t.Fatal(err)
	}
	var n atomic.Int32
	clave := "77777777-7777-4777-8777-777777777777"
	for _, uid := range []int64{u, u2} {
		o, err := idempotency.Run(context.Background(), testDB, discard, req(clave, uid, `{}`), efecto(clave, &n))
		if err != nil || !o.Result.IsOk() || o.Replayed {
			t.Fatalf("usuario %d: %+v err=%v", uid, o, err)
		}
	}
	if n.Load() != 2 {
		t.Fatalf("la misma clave de otro usuario es otra intención: ejecuciones = %d", n.Load())
	}
}

// Solicitudes simultáneas con la misma clave: el efecto ocurre una vez y todas ven lo mismo.
func TestIdempotenciaConcurrenteEjecutaUnaVez(t *testing.T) {
	u := prepararIdem(t)
	var n atomic.Int32
	const workers = 8
	var wg sync.WaitGroup
	salidas := make([]idempotency.Outcome[item], workers)
	errs := make([]error, workers)
	arranque := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-arranque
			salidas[i], errs[i] = idempotency.Run(context.Background(), testDB, discard, req(k6, u, `{"c":1}`), efecto(k6, &n))
		}(i)
	}
	close(arranque)
	wg.Wait()

	replays := 0
	for i := 0; i < workers; i++ {
		if errs[i] != nil || !salidas[i].Result.IsOk() {
			t.Fatalf("worker %d: %+v err=%v", i, salidas[i], errs[i])
		}
		if salidas[i].Result.Value() != salidas[0].Result.Value() {
			t.Fatalf("todas deben ver el mismo resultado: %v vs %v", salidas[i].Result.Value(), salidas[0].Result.Value())
		}
		if salidas[i].Replayed {
			replays++
		}
	}
	if n.Load() != 1 || efectos(t, k6) != 1 || replays != workers-1 {
		t.Fatalf("ejecuciones=%d filas=%d repeticiones=%d (quería 1, 1, %d)", n.Load(), efectos(t, k6), replays, workers-1)
	}
}

// La opción de actor fija app.usuario_id para la V4 dentro de la misma transacción.
func TestTransaccionConActor(t *testing.T) {
	u := prepararIdem(t)
	var visto string
	_, err := transaction.Run(context.Background(), testDB, nil, discard,
		func(ctx context.Context, tx *sql.Tx) (bool, error) {
			return true, tx.QueryRowContext(ctx, `SELECT current_setting('app.usuario_id', true)`).Scan(&visto)
		},
		func(bool) bool { return false }, transaction.WithActor(u))
	if err != nil || visto == "" {
		t.Fatalf("visto=%q err=%v", visto, err)
	}
	// Fuera de la transacción el ajuste local no persiste.
	var despues sql.NullString
	_ = testDB.QueryRow(`SELECT current_setting('app.usuario_id', true)`).Scan(&despues)
	if despues.Valid && despues.String != "" {
		t.Fatalf("el actor no debe filtrarse a otras conexiones: %q", despues.String)
	}
}
