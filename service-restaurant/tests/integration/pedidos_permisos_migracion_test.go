package integration

import (
	"context"
	"strings"
	"testing"

	"service-restaurant/migrations"
)

const permisosPedidos = `'PEDIDO_VER','PEDIDO_CREAR','PEDIDO_EDITAR','DESCUENTO_AUTORIZAR'`

func TestMigracion00011Permisos(t *testing.T) {
	app(t)
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo IN ('PEDIDO_VER','PEDIDO_EDITAR')`); n != 2 {
		t.Fatalf("permisos nuevos = %d, quería 2", n)
	}
	if n := cuentaRolPermiso(t, `'ADMIN'`, permisosPedidos); n != 4 {
		t.Errorf("ADMIN recibe %d de los 4 permisos", n)
	}
	if n := cuentaRolPermiso(t, `'MESERO'`, `'PEDIDO_VER','PEDIDO_CREAR','PEDIDO_EDITAR'`); n != 3 {
		t.Errorf("MESERO recibe %d de PEDIDO_VER, PEDIDO_CREAR y PEDIDO_EDITAR", n)
	}
	if n := cuentaRolPermiso(t, `'MESERO'`, `'DESCUENTO_AUTORIZAR'`); n != 0 {
		t.Errorf("MESERO no debe autorizar descuentos (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'CAJERO'`, permisosPedidos); n != 4 {
		t.Errorf("CAJERO recibe %d de los 4 permisos", n)
	}
	if n := cuentaRolPermiso(t, `'COCINA'`, `'PEDIDO_VER'`); n != 1 {
		t.Errorf("COCINA debe ver pedidos (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'COCINA'`, `'PEDIDO_CREAR','PEDIDO_EDITAR','DESCUENTO_AUTORIZAR'`); n != 0 {
		t.Errorf("COCINA recibe permisos de más (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'ALMACEN'`, permisosPedidos); n != 0 {
		t.Errorf("ALMACEN no debe tocar pedidos (%d)", n)
	}
}

// La sección Down se ejecuta en una transacción que se descarta, para no alterar la base compartida.
func TestMigracion00011BajaLimpia(t *testing.T) {
	app(t)
	raw, err := migrations.FS.ReadFile("00011_pedidos_permisos.sql")
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
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo IN ('PEDIDO_VER','PEDIDO_EDITAR')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down quedan %d permisos nuevos (%v)", n, err)
	}
	// Los permisos de 00001 siguen existiendo y ADMIN conserva los suyos; MESERO y CAJERO pierden lo que 00011 les dio.
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo IN ('PEDIDO_CREAR','DESCUENTO_AUTORIZAR')`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("Down no debe borrar los permisos de 00001 (%d, %v)", n, err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
		JOIN sch_restaurante.roles r ON r.id=rp.rol_id JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id
		WHERE r.codigo IN ('MESERO','CAJERO') AND p.codigo IN ('PEDIDO_CREAR','DESCUENTO_AUTORIZAR')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down MESERO/CAJERO conservan %d asignaciones de 00011 (%v)", n, err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
		JOIN sch_restaurante.roles r ON r.id=rp.rol_id JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id
		WHERE r.codigo='ADMIN' AND p.codigo IN ('PEDIDO_CREAR','DESCUENTO_AUTORIZAR')`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("ADMIN debe conservar los permisos de 00001 (%d, %v)", n, err)
	}
}
