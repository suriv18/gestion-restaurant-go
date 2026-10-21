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
		WHERE table_schema='sch_restaurante' AND table_type='BASE TABLE'`); n != 51 {
		t.Errorf("tablas = %d, quería 51 (49 de la V4 + sesiones e idempotencia)", n)
	}
	if n := scalar(t, `SELECT count(*) FROM information_schema.columns c
		JOIN information_schema.tables t USING (table_schema, table_name)
		WHERE c.table_schema='sch_restaurante' AND t.table_type='BASE TABLE'`); n != 683 {
		t.Errorf("columnas = %d, quería 683 (664 de la V4 + 7 de sesiones + 9 de idempotencia + 3 de baja en mesas)", n)
	}
	if n := scalar(t, `SELECT count(*) FROM pg_trigger tg
		JOIN pg_class c ON c.oid=tg.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE n.nspname='sch_restaurante' AND NOT tg.tgisinternal`); n != 187 {
		t.Errorf("triggers = %d, quería 187", n)
	}
}

func TestMigrationsSeeds(t *testing.T) {
	// Por código y no por conteo total: otras pruebas crean roles y permisos en la misma base.
	semillas := []struct {
		tabla, codigos string
		esperadas      int
	}{
		{"roles", "'ADMIN','CAJERO','MESERO','COCINA','ALMACEN'", 5},
		{"permisos", "'PEDIDO_CREAR','PEDIDO_ANULAR','DESCUENTO_AUTORIZAR','CAJA_OPERAR','PAGO_DEVOLVER','COMPRA_REGISTRAR','INVENTARIO_AJUSTAR','COCINA_OPERAR','COMPROBANTE_EMITIR','USUARIO_ADMINISTRAR'", 10},
		{"unidades_medida", "'KG','G','L','ML','UND'", 5},
		{"metodos_pago", "'EFECTIVO','TARJETA','TRANSFERENCIA','YAPE','PLIN','OTROS'", 6},
		{"impuestos", "'IGV_TOTAL'", 1},
	}
	for _, s := range semillas {
		if got := scalar(t, "SELECT count(*) FROM sch_restaurante."+s.tabla+" WHERE codigo IN ("+s.codigos+")"); got != s.esperadas {
			t.Errorf("%s: %d semillas, quería %d", s.tabla, got, s.esperadas)
		}
	}
	// El rol ADMIN conserva sus diez permisos de la semilla.
	if got := scalar(t, "SELECT count(*) FROM sch_restaurante.roles_permisos rp JOIN sch_restaurante.roles r ON r.id=rp.rol_id WHERE r.codigo='ADMIN' AND rp.es_activo"); got < 10 {
		t.Errorf("ADMIN tiene %d permisos activos, quería al menos 10", got)
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
