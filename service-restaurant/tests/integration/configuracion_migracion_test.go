package integration

import (
	"context"
	"strings"
	"testing"

	"service-restaurant/migrations"
)

const permisosNuevos = `'NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_VER','MESA_ADMINISTRAR'`

func TestMigracion00008Permisos(t *testing.T) {
	app(t)
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo IN (`+permisosNuevos+`)`); n != 6 {
		t.Fatalf("permisos nuevos = %d, quería 6", n)
	}
	cuenta := func(roles, permisos string) int {
		return scalar(t, `SELECT count(*) FROM sch_restaurante.roles_permisos rp
			JOIN sch_restaurante.roles r ON r.id=rp.rol_id JOIN sch_restaurante.permisos p ON p.id=rp.permiso_id
			WHERE rp.es_activo AND r.codigo IN (`+roles+`) AND p.codigo IN (`+permisos+`)`)
	}
	if n := cuenta(`'ADMIN'`, permisosNuevos); n != 6 {
		t.Errorf("ADMIN recibe %d de los 6 permisos nuevos", n)
	}
	if n := cuenta(`'MESERO','CAJERO'`, `'MESA_VER'`); n != 2 {
		t.Errorf("MESERO y CAJERO reciben MESA_VER en %d de 2 roles", n)
	}
	if n := cuenta(`'MESERO','CAJERO'`, `'NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_ADMINISTRAR'`); n != 0 {
		t.Errorf("MESERO/CAJERO no deben recibir otros permisos nuevos (%d)", n)
	}
	if n := cuenta(`'COCINA','ALMACEN'`, permisosNuevos); n != 0 {
		t.Errorf("COCINA/ALMACEN no deben recibir permisos nuevos (%d)", n)
	}
}

func TestMesaDeBajaExigeMotivoYGuardaElActor(t *testing.T) {
	app(t)
	asSystem(t,
		`INSERT INTO sch_restaurante.zonas(sucursal_id,nombre)
		 SELECT id,'ZONA-MIG' FROM sch_restaurante.sucursales WHERE codigo='PRINCIPAL'
		 AND NOT EXISTS (SELECT 1 FROM sch_restaurante.zonas WHERE nombre='ZONA-MIG')`,
		`INSERT INTO sch_restaurante.mesas(sucursal_id,zona_id,numero_mesa,capacidad)
		 SELECT z.sucursal_id,z.id,'MIG-1',4 FROM sch_restaurante.zonas z WHERE z.nombre='ZONA-MIG'
		 AND NOT EXISTS (SELECT 1 FROM sch_restaurante.mesas WHERE numero_mesa='MIG-1')`)

	err := asSystemErr(`UPDATE sch_restaurante.mesas SET es_activo=false WHERE numero_mesa='MIG-1'`)
	if err == nil || !strings.Contains(err.Error(), "motivo_eliminacion") {
		t.Fatalf("la baja sin motivo debía rechazarse por motivo_eliminacion: %v", err)
	}
	asSystem(t, `UPDATE sch_restaurante.mesas SET es_activo=false, motivo_eliminacion='Mesa dañada' WHERE numero_mesa='MIG-1'`)
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.mesas m JOIN sch_restaurante.usuarios u ON u.id=m.eliminado_por
		WHERE m.numero_mesa='MIG-1' AND NOT m.es_activo AND m.eliminado_en IS NOT NULL AND u.es_sistema
		AND m.motivo_eliminacion='Mesa dañada'`); n != 1 {
		t.Fatal("la baja debe guardar motivo, fecha y actor")
	}
	asSystem(t, `UPDATE sch_restaurante.mesas SET es_activo=true WHERE numero_mesa='MIG-1'`)
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE numero_mesa='MIG-1' AND es_activo
		AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL`); n != 1 {
		t.Fatal("la restauración debe limpiar los marcadores")
	}
}

// La migración debe poder revertirse. Se ejecuta la sección Down dentro de una transacción
// que se descarta, para no alterar la base compartida por las demás pruebas.
func TestMigracion00008BajaLimpia(t *testing.T) {
	app(t)
	raw, err := migrations.FS.ReadFile("00008_configuracion_salon.sql")
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
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.permisos WHERE codigo IN (`+permisosNuevos+`)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down quedan %d permisos nuevos (%v)", n, err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_schema='sch_restaurante' AND table_name='mesas' AND column_name IN ('eliminado_en','eliminado_por','motivo_eliminacion')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("tras Down quedan %d columnas de baja en mesas (%v)", n, err)
	}
}

func sinComentarios(s string) string {
	var b strings.Builder
	for _, l := range strings.Split(s, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "--") {
			b.WriteString(l)
		}
	}
	return strings.TrimSpace(b.String())
}
