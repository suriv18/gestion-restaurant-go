package integration

import (
	"context"
	"strings"
	"testing"

	"service-restaurant/migrations"
)

func TestMigracion00012Permisos(t *testing.T) {
	app(t)
	if n := cuentaRolPermiso(t, `'COCINA'`, `'COCINA_OPERAR'`); n != 1 {
		t.Errorf("COCINA debe recibir COCINA_OPERAR (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'ADMIN'`, `'COCINA_OPERAR'`); n != 1 {
		t.Errorf("ADMIN conserva COCINA_OPERAR (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'MESERO','CAJERO','ALMACEN'`, `'COCINA_OPERAR'`); n != 0 {
		t.Errorf("MESERO, CAJERO y ALMACEN no operan la cocina (%d)", n)
	}
}

// La sección Down se ejecuta en una transacción que se descarta, para no alterar la base compartida.
func TestMigracion00012BajaLimpia(t *testing.T) {
	app(t)
	raw, err := migrations.FS.ReadFile("00012_cocina_permisos.sql")
	if err != nil {
		t.Fatal(err)
	}
	partes := strings.SplitN(string(raw), "-- +goose Down", 2)
	if len(partes) != 2 {
		t.Fatal("la migración no tiene sección Down")
	}
	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, s := range strings.Split(partes[1], ";\n") {
		if sinComentarios(s) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, s); err != nil {
			t.Fatalf("Down: %v\n%s", err, s)
		}
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
		JOIN sch_restaurante.roles r ON r.id=rp.rol_id JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id
		WHERE r.codigo='COCINA' AND p.codigo='COCINA_OPERAR'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down COCINA conserva COCINA_OPERAR (%d, %v)", n, err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
		JOIN sch_restaurante.roles r ON r.id=rp.rol_id JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id
		WHERE r.codigo='ADMIN' AND p.codigo='COCINA_OPERAR'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("Down no debe tocar a ADMIN (%d, %v)", n, err)
	}
}
