package integration

import (
	"context"
	"strings"
	"testing"

	"service-restaurant/migrations"
)

func TestMigracion00013Permisos(t *testing.T) {
	app(t)
	if n := cuentaRolPermiso(t, `'CAJERO'`, `'PEDIDO_ANULAR'`); n != 1 {
		t.Errorf("CAJERO debe recibir PEDIDO_ANULAR (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'ADMIN'`, `'PEDIDO_ANULAR'`); n != 1 {
		t.Errorf("ADMIN conserva PEDIDO_ANULAR (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'MESERO','COCINA','ALMACEN'`, `'PEDIDO_ANULAR'`); n != 0 {
		t.Errorf("MESERO, COCINA y ALMACEN no anulan pedidos (%d)", n)
	}
}

// La sección Down se ejecuta en una transacción que se descarta, para no alterar la base compartida.
func TestMigracion00013BajaLimpia(t *testing.T) {
	app(t)
	raw, err := migrations.FS.ReadFile("00013_anulacion_permisos.sql")
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
		WHERE r.codigo='CAJERO' AND p.codigo='PEDIDO_ANULAR'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down CAJERO conserva PEDIDO_ANULAR (%d, %v)", n, err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
		JOIN sch_restaurante.roles r ON r.id=rp.rol_id JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id
		WHERE r.codigo='ADMIN' AND p.codigo='PEDIDO_ANULAR'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("Down no debe tocar a ADMIN (%d, %v)", n, err)
	}
}
