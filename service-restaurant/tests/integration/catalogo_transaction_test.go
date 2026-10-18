package integration

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"service-restaurant/internal/platform/transaction"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func setupRunnerTables(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	stmts := []string{
		`DROP TABLE IF EXISTS public.runner_items`,
		`DROP TABLE IF EXISTS public.runner_deferred`,
		`CREATE TABLE public.runner_items (id int)`,
		`CREATE TABLE public.runner_deferred (id int, CONSTRAINT runner_deferred_uq UNIQUE (id) DEFERRABLE INITIALLY DEFERRED)`,
	}
	for _, s := range stmts {
		if _, err := testDB.ExecContext(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
}

func count(t *testing.T, table string) int {
	t.Helper()
	// Lectura independiente: otra conexión del pool, fuera de la transacción bajo prueba.
	return scalar(t, "SELECT count(*) FROM public."+table)
}

func insertOne(ctx context.Context, tx *sql.Tx) (bool, error) {
	_, err := tx.ExecContext(ctx, `INSERT INTO public.runner_items VALUES (1)`)
	return true, err
}

func TestRunnerCommitsWhenResultAllows(t *testing.T) {
	setupRunnerTables(t)
	ok, err := transaction.Run(context.Background(), testDB, nil, discard,
		func(ctx context.Context, tx *sql.Tx) (bool, error) { return insertOne(ctx, tx) },
		func(allow bool) bool { return allow })
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if n := count(t, "runner_items"); n != 1 {
		t.Fatalf("filas = %d, quería 1", n)
	}
}

// Rechazo esperado: sin error técnico, pero rollback verificado con otra lectura.
func TestRunnerRollsBackOnRejectedResult(t *testing.T) {
	setupRunnerTables(t)
	res, err := transaction.Run(context.Background(), testDB, nil, discard,
		func(ctx context.Context, tx *sql.Tx) (bool, error) {
			_, err := insertOne(ctx, tx)
			return false, err // el caso de uso devolvió un Result fallido
		},
		func(allow bool) bool { return allow })
	if err != nil || res {
		t.Fatalf("res=%v err=%v", res, err)
	}
	if n := count(t, "runner_items"); n != 0 {
		t.Fatalf("el rechazo debía hacer rollback; filas = %d", n)
	}
}

func TestRunnerRollsBackOnTechnicalError(t *testing.T) {
	setupRunnerTables(t)
	boom := errors.New("fallo técnico")
	_, err := transaction.Run(context.Background(), testDB, nil, discard,
		func(ctx context.Context, tx *sql.Tx) (bool, error) {
			if _, err := insertOne(ctx, tx); err != nil {
				return false, err
			}
			return true, boom
		},
		func(allow bool) bool { return allow })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if n := count(t, "runner_items"); n != 0 {
		t.Fatalf("filas = %d, quería 0", n)
	}
}

// Error de commit: jamás se devuelve éxito (restricción diferida que falla en COMMIT).
func TestRunnerNeverReportsSuccessWhenCommitFails(t *testing.T) {
	setupRunnerTables(t)
	_, err := transaction.Run(context.Background(), testDB, nil, discard,
		func(ctx context.Context, tx *sql.Tx) (bool, error) {
			for i := 0; i < 2; i++ {
				if _, err := tx.ExecContext(ctx, `INSERT INTO public.runner_deferred VALUES (1)`); err != nil {
					return false, err
				}
			}
			return true, nil
		},
		func(allow bool) bool { return allow })
	if err == nil {
		t.Fatal("el commit fallido debía devolver error")
	}
	if n := count(t, "runner_deferred"); n != 0 {
		t.Fatalf("filas = %d, quería 0", n)
	}
}
