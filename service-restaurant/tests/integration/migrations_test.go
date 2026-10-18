package integration

import (
	"context"
	"strings"
	"testing"
)

func scalar(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := testDB.QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

func TestMigrationsCreateFullSchema(t *testing.T) {
	if n := scalar(t, `SELECT count(*) FROM information_schema.tables
		WHERE table_schema='sch_restaurante' AND table_type='BASE TABLE'`); n != 49 {
		t.Errorf("tablas = %d, quería 49", n)
	}
	if n := scalar(t, `SELECT count(*) FROM information_schema.columns c
		JOIN information_schema.tables t USING (table_schema, table_name)
		WHERE c.table_schema='sch_restaurante' AND t.table_type='BASE TABLE'`); n != 664 {
		t.Errorf("columnas = %d, quería 664", n)
	}
	if n := scalar(t, `SELECT count(*) FROM pg_trigger tg
		JOIN pg_class c ON c.oid=tg.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE n.nspname='sch_restaurante' AND NOT tg.tgisinternal`); n != 187 {
		t.Errorf("triggers = %d, quería 187", n)
	}
}

func TestMigrationsSeeds(t *testing.T) {
	cases := map[string]int{
		"roles": 5, "permisos": 10, "roles_permisos": 10,
		"unidades_medida": 5, "impuestos": 1, "metodos_pago": 6,
	}
	for table, want := range cases {
		if got := scalar(t, "SELECT count(*) FROM sch_restaurante."+table); got != want {
			t.Errorf("%s = %d filas, quería %d", table, got, want)
		}
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE es_sistema AND password_hash IS NULL`); n != 1 {
		t.Errorf("usuario __sistema__ = %d, quería 1", n)
	}
}

// Decisión 12 del SQL V4: sin actor en la transacción, la escritura se rechaza.
func TestWriteWithoutActorIsRejected(t *testing.T) {
	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SET LOCAL search_path = sch_restaurante, pg_catalog`); err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO categorias(nombre) VALUES ('SIN_ACTOR')`)
	if err != nil && !strings.Contains(err.Error(), "app.usuario_id") {
		t.Fatalf("el rechazo debía venir del chequeo de actor, no de otro error: %v", err)
	}
	if err == nil {
		t.Fatal("la escritura sin app.usuario_id debía fallar")
	}
}

func TestWriteWithSystemActorSucceedsAndIsAudited(t *testing.T) {
	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, q := range []string{
		`SET LOCAL search_path = sch_restaurante, pg_catalog`,
		`SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true)`,
		`INSERT INTO categorias(nombre) VALUES ('CON_ACTOR')`,
	} {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM auditoria_eventos WHERE tabla='categorias' AND accion='INSERT'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("eventos de auditoría = %d, quería 1", n)
	}
}
