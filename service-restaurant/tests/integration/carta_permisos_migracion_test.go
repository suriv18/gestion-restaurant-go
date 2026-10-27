package integration

import (
	"context"
	"strings"
	"testing"

	"service-restaurant/migrations"
)

const permisosCarta = `'CARTA_VER','CARTA_ADMINISTRAR','CARTA_DISPONIBILIDAD','ESTACION_VER','ESTACION_ADMINISTRAR','IMPUESTO_VER','IMPUESTO_ADMINISTRAR'`

func cuentaRolPermiso(t *testing.T, roles, permisos string) int {
	t.Helper()
	return scalar(t, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
		JOIN sch_restaurante.roles r ON r.id=rp.rol_id JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id
		WHERE rp.es_activo AND r.codigo IN (`+roles+`) AND p.codigo IN (`+permisos+`)`)
}

func TestMigracion00010Permisos(t *testing.T) {
	app(t)
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo IN (`+permisosCarta+`)`); n != 7 {
		t.Fatalf("permisos de carta = %d, quería 7", n)
	}
	if n := cuentaRolPermiso(t, `'ADMIN'`, permisosCarta); n != 7 {
		t.Errorf("ADMIN recibe %d de los 7 permisos", n)
	}
	if n := cuentaRolPermiso(t, `'COCINA'`, `'CARTA_VER','CARTA_DISPONIBILIDAD','ESTACION_VER'`); n != 3 {
		t.Errorf("COCINA recibe %d de CARTA_VER, CARTA_DISPONIBILIDAD y ESTACION_VER", n)
	}
	if n := cuentaRolPermiso(t, `'COCINA'`, `'CARTA_ADMINISTRAR','ESTACION_ADMINISTRAR','IMPUESTO_VER','IMPUESTO_ADMINISTRAR'`); n != 0 {
		t.Errorf("COCINA no debe administrar ni ver impuestos (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'MESERO'`, `'CARTA_VER','ESTACION_VER'`); n != 2 {
		t.Errorf("MESERO recibe %d de CARTA_VER y ESTACION_VER", n)
	}
	if n := cuentaRolPermiso(t, `'MESERO'`, `'CARTA_ADMINISTRAR','CARTA_DISPONIBILIDAD','ESTACION_ADMINISTRAR','IMPUESTO_VER','IMPUESTO_ADMINISTRAR'`); n != 0 {
		t.Errorf("MESERO recibe permisos de más (%d)", n)
	}
	if n := cuentaRolPermiso(t, `'CAJERO','ALMACEN'`, `'CARTA_VER'`); n != 2 {
		t.Errorf("CAJERO y ALMACEN reciben CARTA_VER en %d de 2 roles", n)
	}
	if n := cuentaRolPermiso(t, `'CAJERO','ALMACEN'`, `'CARTA_ADMINISTRAR','CARTA_DISPONIBILIDAD','ESTACION_VER','ESTACION_ADMINISTRAR','IMPUESTO_VER','IMPUESTO_ADMINISTRAR'`); n != 0 {
		t.Errorf("CAJERO/ALMACEN reciben permisos de más (%d)", n)
	}
}

// La sección Down se ejecuta en una transacción que se descarta, para no alterar la base compartida.
func TestMigracion00010BajaLimpia(t *testing.T) {
	app(t)
	raw, err := migrations.FS.ReadFile("00010_carta_permisos.sql")
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
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo IN (`+permisosCarta+`)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down quedan %d permisos de carta (%v)", n, err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
		JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id WHERE p.codigo IN (`+permisosCarta+`)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down quedan %d asignaciones (%v)", n, err)
	}
}
