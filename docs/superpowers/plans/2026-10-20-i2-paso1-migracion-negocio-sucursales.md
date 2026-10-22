# I2 · Paso 1 — Migración 00008, permisos, negocio y sucursales — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar la migración `00008_configuracion_salon` (seis permisos nuevos y baja con motivo en `mesas`) y la API de negocio (`GET/PATCH /negocio`) y sucursales (listar, leer, crear, editar, baja, restauración) dentro del módulo `identidad`.

**Architecture:** Mismo esquema CQRS de I1 (`domain`, `application`, `infrastructure`, `web`). Los casos de uso reciben el `appctx.Actor` ya validado y autorizan en memoria con `actor.Can/CanAnywhere`; las escrituras se ejecutan dentro de la transacción decorada por `escritura[...]` (actor en `app.usuario_id`, idempotencia opcional); las lecturas van directo al pool.

**Tech Stack:** Go 1.26, `net/http` (mux 1.22), `database/sql` + pgx, PostgreSQL 18, goose, testcontainers (pruebas de integración), módulo `service-restaurant`.

**Spec:** `docs/superpowers/specs/2026-10-18-i2-configuracion-salon-design.md` (§1–§5, §8, §9 paso 1). Este plan **no** cubre zonas/mesas (paso 2), contrato/ADR/expediente (paso 3) ni frontend (paso 4).

## Global Constraints

- Todos los comandos se ejecutan desde `service-restaurant/` (raíz del módulo Go). Las pruebas de integración necesitan Docker activo.
- Los IDs viajan como **cadenas** en JSON (`"id": "7"`); internamente son `int64`.
- Errores en `application/problem+json` con el patrón de I1: 400 (`KindBadRequest`), 401, 403 (`CodigoSinPermiso`), 404 (`no_encontrado`), 409 (`KindConflict`), 422 (`KindValidation`/`KindBusinessRule`). El código del error nombra el campo (`moneda_invalida`, `ubigeo_invalido`, …).
- Los `POST` de creación, baja y restauración exigen `Idempotency-Key` (UUID); sin ella → 400 `idempotency_key_requerida`. Los `PATCH` no la exigen.
- `creado_por`, `actualizado_por`, `eliminado_*` y `negocio_id` **nunca** se aceptan del cliente: `DecodeJSON` rechaza campos desconocidos (400 `json_invalido`) y el actor sale de `app.usuario_id`.
- Permisos nuevos: `NEGOCIO_VER`, `NEGOCIO_ADMINISTRAR`, `SUCURSAL_VER`, `SUCURSAL_ADMINISTRAR`, `MESA_VER`, `MESA_ADMINISTRAR`. `ADMIN` recibe los seis; `MESERO` y `CAJERO` solo `MESA_VER`; `COCINA` y `ALMACEN` ninguno.
- Negocio: singleton; RUC y `singleton` no se editan; moneda `^[A-Z]{3}$`; zona horaria IANA; ubigeo 6 dígitos.
- Sucursales: código único; no se puede dar de baja la última sede activa (422 `ultima_sede_activa`); la baja exige `motivo` (máx. 2000); la restauración conserva el ID.
- Dependencias entre capas: `domain` no importa nada de infraestructura; `application` no importa `database/sql` ni `net/http`; lo valida `tests/architecture`.
- Estilo: comentarios y mensajes en español, igual que el código vecino; `gofmt` limpio; commits en español con el trailer `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.

## Decisión derivada del análisis (desvío menor del spec §3)

`v_permisos_efectivos` solo concede permisos sobre sedes **activas**. Al dar de baja una sede, el actor pierde sus permisos sobre ella y, siguiendo §3 al pie de la letra, nadie podría **restaurarla**. Regla adoptada (se anota en el spec en la Tarea 4):

| Sede | Visible para | Puede administrar |
|---|---|---|
| Activa | quien tenga `SUCURSAL_VER` o `SUCURSAL_ADMINISTRAR` **en esa sede** | `SUCURSAL_ADMINISTRAR` **en esa sede** |
| Inactiva | quien tenga `SUCURSAL_ADMINISTRAR` en **alguna** sede | quien tenga `SUCURSAL_ADMINISTRAR` en **alguna** sede |

Riesgo conocido: una persona cuya única sede asignada se da de baja pierde todo acceso hasta que otro administrador la restaure o la reasigne (se recoge en la sección de riesgos del spec).

## Estructura de archivos

Crear:

| Archivo | Responsabilidad |
|---|---|
| `migrations/00008_configuracion_salon.sql` | Permisos, asignaciones a roles, baja con motivo en `mesas` |
| `internal/modules/identidad/domain/vo/configuracion.go` (+ `_test.go`) | `Ubigeo`, `Moneda`, `ZonaHoraria`, `CodigoSucursal` |
| `internal/modules/identidad/application/mapper/autorizacion.go` (+ `_test.go`) | `Requiere`, `SedesVisibles`, `SucursalVisible`, `PuedeAdministrarSucursal` |
| `internal/modules/identidad/application/dto/result/negocio.go`, `sucursales.go` | Resultados `Negocio`, `Sucursal`, `PaginaSucursales` |
| `internal/modules/identidad/application/dto/command/negocio.go`, `sucursales.go` | `Campo[T]`, `EditarNegocio`, `CrearSucursal`, `EditarSucursal`, `CambiarEstadoSucursal` |
| `internal/modules/identidad/application/dto/query/negocio.go`, `sucursales.go` | `ObtenerNegocio`, `ListarSucursales`, `ObtenerSucursal` |
| `internal/modules/identidad/application/port/inbound/negocio.go`, `sucursales.go` | Interfaces de los casos de uso |
| `internal/modules/identidad/application/port/outbound/negocio.go`, `sucursales.go` | `NegocioStore/Reader`, `SucursalStore/Reader` |
| `internal/modules/identidad/application/usecase/query/negocio.go`, `sucursales.go` (+ tests) | Consultas |
| `internal/modules/identidad/application/usecase/command/negocio.go`, `sucursales.go`, `configuracion_helpers.go` (+ tests) | Comandos |
| `internal/modules/identidad/infrastructure/persistence/write/repository/configuracion_sql.go` | SQL de escritura |
| `internal/modules/identidad/infrastructure/persistence/write/adapter/configuracion.go` | Adaptadores de escritura |
| `internal/modules/identidad/infrastructure/persistence/read/repository/configuracion_sql_reader.go` | SQL de lectura |
| `internal/modules/identidad/infrastructure/persistence/read/adapter/configuracion_read_adapter.go` | Adaptador de lectura |
| `internal/modules/identidad/infrastructure/config/configuracion.go` | Composición de negocio y sucursales |
| `internal/modules/identidad/web/dto/request/configuracion.go`, `response/configuracion.go` | DTO HTTP |
| `internal/modules/identidad/web/mapper/configuracion_web_mapper.go` | Resultado → respuesta |
| `internal/modules/identidad/web/controller/negocio_handler.go`, `sucursales_handler.go`, `configuracion_routes.go`, `configuracion_helpers.go` | HTTP |
| `tests/integration/configuracion_helpers_test.go`, `configuracion_migracion_test.go`, `configuracion_negocio_test.go`, `configuracion_sucursales_test.go` | Integración |

Modificar: `internal/modules/identidad/domain/policy/permisos.go`, `internal/modules/identidad/infrastructure/config/module.go`, `tests/integration/migrations_test.go`, `docs/superpowers/specs/2026-10-18-i2-configuracion-salon-design.md`.

---

### Task 1: Migración 00008

**Files:**
- Create: `migrations/00008_configuracion_salon.sql`
- Create: `tests/integration/configuracion_helpers_test.go` (solo `asSystemErr` por ahora)
- Create: `tests/integration/configuracion_migracion_test.go`
- Modify: `tests/integration/migrations_test.go:25` (columnas 680 → 683)

**Interfaces:**
- Produces: permisos `NEGOCIO_*`, `SUCURSAL_*`, `MESA_*` en `permisos`; columnas `mesas.eliminado_en/eliminado_por/motivo_eliminacion`; restricción `ck_mesas_baja`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`tests/integration/configuracion_helpers_test.go`:

```go
package integration

import (
	"context"

	"service-restaurant/internal/platform/database"
)

// asSystemErr es asSystem pero devuelve el error de la sentencia en lugar de abortar la prueba:
// sirve para comprobar que la base rechaza una escritura.
func asSystemErr(stmts ...string) error {
	ctx := context.Background()
	tx, err := testDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.usuarios WHERE es_sistema`).Scan(&id); err != nil {
		return err
	}
	if err := database.SetActor(ctx, tx, id); err != nil {
		return err
	}
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return tx.Commit()
}
```

`tests/integration/configuracion_migracion_test.go`:

```go
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
```

En `tests/integration/migrations_test.go` cambiar el conteo de columnas:

```go
		WHERE c.table_schema='sch_restaurante' AND t.table_type='BASE TABLE'`); n != 683 {
		t.Errorf("columnas = %d, quería 683 (664 de la V4 + 7 de sesiones + 9 de idempotencia + 3 de baja en mesas)", n)
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `go test -count=1 ./tests/integration/ -run "TestMigracion00008|TestMesaDeBaja|TestMigrationsCreateFullSchema" -v`
Expected: FAIL (`permisos nuevos = 0`, columnas = 680, `column "motivo_eliminacion" does not exist`).

- [ ] **Step 3: Escribir la migración**

`migrations/00008_configuracion_salon.sql`:

```sql
-- Migracion 00008: configuracion y salon (I2, rebanada 1).
-- Seis permisos nuevos (negocio, sucursal, mesa) y baja con motivo en mesas (ADR 005).
-- Requiere PostgreSQL 18 y las migraciones 00001-00007.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO permisos(codigo,descripcion) VALUES
 ('NEGOCIO_VER','Consultar los datos del negocio'),
 ('NEGOCIO_ADMINISTRAR','Editar los datos del negocio'),
 ('SUCURSAL_VER','Consultar sucursales'),
 ('SUCURSAL_ADMINISTRAR','Crear, editar, dar de baja y restaurar sucursales'),
 ('MESA_VER','Consultar zonas y mesas'),
 ('MESA_ADMINISTRAR','Administrar zonas y mesas');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='ADMIN'
  AND p.codigo IN ('NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_VER','MESA_ADMINISTRAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO') AND p.codigo='MESA_VER';

-- Baja de mesa con motivo (decision del equipo; el expediente la definia como actividad simple).
ALTER TABLE mesas
    ADD COLUMN eliminado_en TIMESTAMPTZ,
    ADD COLUMN eliminado_por BIGINT,
    ADD COLUMN motivo_eliminacion TEXT;

-- Filas inactivas previas: se completan antes de exigir la restriccion. El trigger vigente
-- (sin baja logica) permite el UPDATE y la auditoria lo registra.
UPDATE mesas
SET eliminado_en = actualizado_en, eliminado_por = actualizado_por,
    motivo_eliminacion = 'Baja anterior a la migración 00008'
WHERE NOT es_activo;

ALTER TABLE mesas ADD CONSTRAINT ck_mesas_baja CHECK (
    (es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
    OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
        AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''));

ALTER TABLE mesas ADD CONSTRAINT fk_mesas_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

DROP TRIGGER a0_metadatos ON mesas;
CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

COMMENT ON COLUMN mesas.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha (ADR 005).';

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

DROP TRIGGER a0_metadatos ON mesas;
CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

ALTER TABLE mesas DROP CONSTRAINT fk_mesas_eliminador;
ALTER TABLE mesas DROP CONSTRAINT ck_mesas_baja;
ALTER TABLE mesas DROP COLUMN eliminado_en, DROP COLUMN eliminado_por, DROP COLUMN motivo_eliminacion;
COMMENT ON COLUMN mesas.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
ALTER TABLE permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos WHERE permiso_id IN (SELECT id FROM permisos WHERE codigo IN
 ('NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_VER','MESA_ADMINISTRAR'));
DELETE FROM permisos WHERE codigo IN
 ('NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_VER','MESA_ADMINISTRAR');
ALTER TABLE permisos ENABLE TRIGGER a_no_borrar;
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: `go test -count=1 ./tests/integration/ -run "TestMigracion00008|TestMesaDeBaja|TestMigrations|TestWrite" -v`
Expected: PASS. Si `TestMigrationsCreateFullSchema` reporta triggers ≠ 187, el `DROP/CREATE TRIGGER` no es neutro: revisar que no se haya creado un trigger extra.

- [ ] **Step 5: Commit**

```bash
git add migrations/00008_configuracion_salon.sql tests/integration/configuracion_helpers_test.go tests/integration/configuracion_migracion_test.go tests/integration/migrations_test.go
git commit -m "I2 paso 1: migración 00008 (permisos de negocio, sucursal y mesa; baja de mesa con motivo)

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Cimientos de dominio y autorización

**Files:**
- Modify: `internal/modules/identidad/domain/policy/permisos.go`
- Create: `internal/modules/identidad/domain/vo/configuracion.go`, `internal/modules/identidad/domain/vo/configuracion_test.go`
- Create: `internal/modules/identidad/application/mapper/autorizacion.go`, `internal/modules/identidad/application/mapper/autorizacion_test.go`

**Interfaces:**
- Produces:
  - `policy.NegocioVer`, `policy.NegocioAdministrar`, `policy.SucursalVer`, `policy.SucursalAdministrar` (string).
  - `vo.NewUbigeo(string) (Ubigeo, error)`, `vo.NewMoneda`, `vo.NewZonaHoraria`, `vo.NewCodigoSucursal`; cada tipo con `String() string`; error `vo.ErrFormatoInvalido` (y `vo.ErrCodigoInvalido` para el código).
  - `mapper.Requiere(a appctx.Actor, mensaje string, permisos ...string) *apperror.AppError`
  - `mapper.SedesVisibles(a appctx.Actor) []int64`
  - `mapper.SucursalVisible(a appctx.Actor, id int64, activa bool) bool`
  - `mapper.PuedeAdministrarSucursal(a appctx.Actor, id int64, activa bool) bool`

- [ ] **Step 1: Escribir las pruebas que fallan**

`internal/modules/identidad/domain/vo/configuracion_test.go`:

```go
package vo

import "testing"

func TestUbigeo(t *testing.T) {
	if u, err := NewUbigeo(" 150101 "); err != nil || u.String() != "150101" {
		t.Fatalf("ubigeo válido: %v %q", err, u.String())
	}
	for _, malo := range []string{"", "15010", "1501011", "15a101", "150 101"} {
		if _, err := NewUbigeo(malo); err == nil {
			t.Errorf("ubigeo %q debía rechazarse", malo)
		}
	}
}

func TestMoneda(t *testing.T) {
	if m, err := NewMoneda(" pen "); err != nil || m.String() != "PEN" {
		t.Fatalf("moneda válida: %v %q", err, m.String())
	}
	for _, mala := range []string{"", "PE", "PENN", "P3N", "12$"} {
		if _, err := NewMoneda(mala); err == nil {
			t.Errorf("moneda %q debía rechazarse", mala)
		}
	}
}

func TestZonaHoraria(t *testing.T) {
	for _, ok := range []string{"America/Lima", "UTC", "Europe/Madrid"} {
		if z, err := NewZonaHoraria(ok); err != nil || z.String() != ok {
			t.Errorf("zona %q: %v", ok, err)
		}
	}
	for _, mala := range []string{"", "Local", "Marte/Olimpo", "../etc/passwd"} {
		if _, err := NewZonaHoraria(mala); err == nil {
			t.Errorf("zona %q debía rechazarse", mala)
		}
	}
}

func TestCodigoSucursal(t *testing.T) {
	if c, err := NewCodigoSucursal(" san-isidro_2 "); err != nil || c.String() != "SAN-ISIDRO_2" {
		t.Fatalf("código válido: %v %q", err, c.String())
	}
	for _, malo := range []string{"", "A", "-AB", "AB CD", "ÁB", "ABCDEFGHIJKLMNOPQRSTU"} {
		if _, err := NewCodigoSucursal(malo); err == nil {
			t.Errorf("código %q debía rechazarse", malo)
		}
	}
}
```

`internal/modules/identidad/application/mapper/autorizacion_test.go`:

```go
package mapper

import (
	"slices"
	"testing"

	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func actorCon(grants ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, grants) }

func TestRequiere(t *testing.T) {
	if ae := Requiere(appctx.Actor{}, "x", policy.NegocioVer); ae == nil || ae.Kind != apperror.KindUnauthorized {
		t.Fatalf("sin actor debe ser 401: %v", ae)
	}
	a := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.NegocioVer})
	if ae := Requiere(a, "x", policy.NegocioAdministrar); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSinPermiso {
		t.Fatalf("sin permiso debe ser 403: %v", ae)
	}
	if ae := Requiere(a, "x", policy.NegocioAdministrar, policy.NegocioVer); ae != nil {
		t.Fatalf("alguno de los permisos basta: %v", ae)
	}
}

func TestSedesVisiblesEsLaUnionOrdenadaSinRepetir(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 5, Permiso: policy.SucursalVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.SucursalAdministrar},
		appctx.Grant{SucursalID: 5, Permiso: policy.SucursalAdministrar},
		appctx.Grant{SucursalID: 9, Permiso: policy.NegocioVer},
	)
	if got := SedesVisibles(a); !slices.Equal(got, []int64{2, 5}) {
		t.Fatalf("sedes visibles = %v, quería [2 5]", got)
	}
}

func TestVisibilidadYAdministracionDeSedeInactiva(t *testing.T) {
	soloVer := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalVer})
	admin := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalAdministrar})

	if !SucursalVisible(soloVer, 1, true) || SucursalVisible(soloVer, 2, true) {
		t.Error("una sede activa solo es visible donde hay VER o ADMINISTRAR")
	}
	if SucursalVisible(soloVer, 2, false) {
		t.Error("una sede inactiva no es visible sin SUCURSAL_ADMINISTRAR")
	}
	if !SucursalVisible(admin, 2, false) || !PuedeAdministrarSucursal(admin, 2, false) {
		t.Error("quien administra sedes ve y gestiona las inactivas (para poder restaurarlas)")
	}
	if PuedeAdministrarSucursal(admin, 2, true) {
		t.Error("una sede activa ajena no se administra")
	}
	if !PuedeAdministrarSucursal(admin, 1, true) {
		t.Error("una sede activa propia sí se administra")
	}
	if PuedeAdministrarSucursal(soloVer, 1, true) {
		t.Error("VER no permite administrar")
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `go test ./internal/modules/identidad/domain/vo/ ./internal/modules/identidad/application/mapper/ 2>&1 | head -20`
Expected: FAIL de compilación (`undefined: NewUbigeo`, `undefined: Requiere`, `undefined: policy.NegocioVer`).

- [ ] **Step 3: Implementar**

`internal/modules/identidad/domain/policy/permisos.go` (reemplazar el contenido):

```go
package policy

// UsuarioAdministrar es el permiso funcional que protege la administración de accesos.
// Es un permiso de la aplicación (semilla de la V4), no un privilegio de PostgreSQL.
const UsuarioAdministrar = "USUARIO_ADMINISTRAR"

// Permisos de configuración (migración 00008). Los de mesa los define el módulo atencion.
const (
	NegocioVer           = "NEGOCIO_VER"
	NegocioAdministrar   = "NEGOCIO_ADMINISTRAR"
	SucursalVer          = "SUCURSAL_VER"
	SucursalAdministrar  = "SUCURSAL_ADMINISTRAR"
)
```

(ejecutar `gofmt -w` sobre el archivo para alinear las constantes).

`internal/modules/identidad/domain/vo/configuracion.go`:

```go
package vo

import (
	"errors"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata" // zonas IANA embebidas: la imagen de ejecución puede no traerlas
)

// ErrFormatoInvalido señala un valor de configuración que no cumple su formato.
var ErrFormatoInvalido = errors.New("formato inválido")

var (
	ubigeoRe         = regexp.MustCompile(`^[0-9]{6}$`)
	monedaRe         = regexp.MustCompile(`^[A-Z]{3}$`)
	codigoSucursalRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,19}$`)
)

// Ubigeo es el código de ubicación de seis dígitos.
type Ubigeo struct{ value string }

func NewUbigeo(raw string) (Ubigeo, error) {
	v := strings.TrimSpace(raw)
	if !ubigeoRe.MatchString(v) {
		return Ubigeo{}, ErrFormatoInvalido
	}
	return Ubigeo{v}, nil
}

func (u Ubigeo) String() string { return u.value }

// Moneda es un código de tres letras mayúsculas (ISO 4217, p. ej. PEN).
type Moneda struct{ value string }

func NewMoneda(raw string) (Moneda, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if !monedaRe.MatchString(v) {
		return Moneda{}, ErrFormatoInvalido
	}
	return Moneda{v}, nil
}

func (m Moneda) String() string { return m.value }

// ZonaHoraria es un nombre de zona IANA conocido (America/Lima). Se rechaza "Local".
type ZonaHoraria struct{ value string }

func NewZonaHoraria(raw string) (ZonaHoraria, error) {
	v := strings.TrimSpace(raw)
	if v == "" || strings.EqualFold(v, "local") {
		return ZonaHoraria{}, ErrFormatoInvalido
	}
	if _, err := time.LoadLocation(v); err != nil {
		return ZonaHoraria{}, ErrFormatoInvalido
	}
	return ZonaHoraria{v}, nil
}

func (z ZonaHoraria) String() string { return z.value }

// CodigoSucursal identifica una sede: mayúsculas, dígitos, _ y - (2 a 20).
type CodigoSucursal struct{ value string }

func NewCodigoSucursal(raw string) (CodigoSucursal, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if !codigoSucursalRe.MatchString(v) {
		return CodigoSucursal{}, ErrCodigoInvalido
	}
	return CodigoSucursal{v}, nil
}

func (c CodigoSucursal) String() string { return c.value }
```

`internal/modules/identidad/application/mapper/autorizacion.go`:

```go
package mapper

import (
	"slices"

	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// Requiere exige un actor autenticado con alguno de los permisos en al menos una sede.
func Requiere(a appctx.Actor, mensaje string, permisos ...string) *apperror.AppError {
	if a.IsZero() {
		return NoAutenticado()
	}
	for _, p := range permisos {
		if a.CanAnywhere(p) {
			return nil
		}
	}
	return apperror.New(apperror.KindForbidden, CodigoSinPermiso, mensaje)
}

// SedesVisibles devuelve, ordenadas y sin repetir, las sedes activas donde el actor tiene
// SUCURSAL_VER o SUCURSAL_ADMINISTRAR (v_permisos_efectivos solo concede permisos sobre sedes activas).
func SedesVisibles(a appctx.Actor) []int64 {
	ids := append(a.SucursalesCon(policy.SucursalVer), a.SucursalesCon(policy.SucursalAdministrar)...)
	slices.Sort(ids)
	return slices.Compact(ids)
}

// SucursalVisible indica si el actor puede ver la sede. Las sedes inactivas no generan permisos
// (la vista de permisos las excluye), así que las ve quien administra sedes en alguna sede:
// es lo que permite restaurarlas.
func SucursalVisible(a appctx.Actor, id int64, activa bool) bool {
	if activa {
		return slices.Contains(SedesVisibles(a), id)
	}
	return a.CanAnywhere(policy.SucursalAdministrar)
}

// PuedeAdministrarSucursal aplica la misma regla para editar, dar de baja y restaurar.
func PuedeAdministrarSucursal(a appctx.Actor, id int64, activa bool) bool {
	if activa {
		return a.Can(policy.SucursalAdministrar, id)
	}
	return a.CanAnywhere(policy.SucursalAdministrar)
}
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: `gofmt -l internal/ && go test ./internal/modules/identidad/domain/... ./internal/modules/identidad/application/mapper/ -v 2>&1 | tail -20`
Expected: `gofmt -l` sin salida; PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/identidad/domain internal/modules/identidad/application/mapper
git commit -m "I2 paso 1: permisos de negocio y sucursal, objetos de valor y reglas de visibilidad de sedes

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Negocio (GET y PATCH /negocio)

**Files:**
- Create: `application/dto/result/negocio.go`, `application/dto/command/negocio.go`, `application/dto/query/negocio.go`
- Create: `application/port/inbound/negocio.go`, `application/port/outbound/negocio.go`
- Create: `application/usecase/query/negocio.go`, `application/usecase/command/negocio.go`, `application/usecase/command/configuracion_helpers.go`, `application/usecase/command/negocio_test.go`
- Create: `infrastructure/persistence/write/repository/configuracion_sql.go`, `.../write/adapter/configuracion.go`
- Create: `infrastructure/persistence/read/repository/configuracion_sql_reader.go`, `.../read/adapter/configuracion_read_adapter.go`
- Create: `infrastructure/config/configuracion.go`; Modify: `infrastructure/config/module.go`
- Create: `web/dto/request/configuracion.go`, `web/dto/response/configuracion.go`, `web/mapper/configuracion_web_mapper.go`, `web/controller/negocio_handler.go`, `web/controller/configuracion_routes.go`
- Create: `tests/integration/configuracion_helpers_test.go` (ampliar), `tests/integration/configuracion_negocio_test.go`

(Rutas relativas a `internal/modules/identidad/`, salvo `tests/`.)

**Interfaces:**
- Consumes: `mapper.Requiere`, `vo.NewMoneda/NewZonaHoraria/NewUbigeo`, `policy.NegocioVer/NegocioAdministrar`, `escritura[...]`, `rechazar/deAlmacen/tecnico`, `textoOpcional`, `emailOpcional`.
- Produces:
  - `result.Negocio` (campos: `ID int64, RUC, RazonSocial, NombreComercial, DireccionFiscal string, Ubigeo, Telefono, Email *string, Moneda, ZonaHoraria string, EsActivo bool, CreadoEn time.Time, CreadoPor int64, ActualizadoEn time.Time, ActualizadoPor int64`).
  - `command.Campo[T]{Presente bool; Valor *T}`; `command.EditarNegocio`; `query.ObtenerNegocio{Actor}`.
  - `inbound.ObtenerNegocio`, `inbound.EditarNegocio` (`Execute(ctx, X) (appresult.Result[result.Negocio], error)`).
  - `outbound.CampoOpcional{Presente bool; Valor *string}`, `outbound.CambiosNegocio`, `outbound.NegocioStore`, `outbound.NegocioReader`.
  - `controller.NewNegocioHandler(obtener, editar)`, `controller.RegisterConfiguracionRoutes(mux, negocio, protect, logger)`.
  - `readadapter.NewConfiguracionReadAdapter(db)`, `writeadapter.NewNegocioWriteAdapter(db)`.
  - Helpers del paquete `command`: `opcionalValidado`, `normUbigeo`, `normMax(max)`, `normEmail`.

- [ ] **Step 1: DTO, puertos y pruebas unitarias del caso de uso (fallan)**

`application/dto/result/negocio.go`:

```go
package result

import "time"

// Negocio es la empresa única del despliegue (RES-RF-003). El RUC no se edita.
type Negocio struct {
	ID              int64
	RUC             string
	RazonSocial     string
	NombreComercial string
	DireccionFiscal string
	Ubigeo          *string
	Telefono        *string
	Email           *string
	Moneda          string
	ZonaHoraria     string
	EsActivo        bool
	CreadoEn        time.Time
	CreadoPor       int64
	ActualizadoEn   time.Time
	ActualizadoPor  int64
}
```

`application/dto/command/negocio.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// Campo es un valor opcional de un PATCH: Presente distingue "no enviado" de "enviado como
// null" (vaciar). Valor es nil cuando se envió null.
type Campo[T any] struct {
	Presente bool
	Valor    *T
}

// EditarNegocio aplica solo los campos presentes. El RUC y el singleton no se editan.
type EditarNegocio struct {
	Actor           appctx.Actor
	RazonSocial     *string
	NombreComercial *string
	DireccionFiscal *string
	Moneda          *string
	ZonaHoraria     *string
	Ubigeo          Campo[string]
	Telefono        Campo[string]
	Email           Campo[string]
}
```

`application/dto/query/negocio.go`:

```go
package query

import appctx "service-restaurant/internal/shared/application/context"

// ObtenerNegocio lee la empresa única.
type ObtenerNegocio struct{ Actor appctx.Actor }
```

`application/port/inbound/negocio.go`:

```go
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ObtenerNegocio interface {
	Execute(context.Context, qry.ObtenerNegocio) (appresult.Result[dtoresult.Negocio], error)
}
type EditarNegocio interface {
	Execute(context.Context, cmd.EditarNegocio) (appresult.Result[dtoresult.Negocio], error)
}
```

`application/port/outbound/negocio.go`:

```go
package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
)

// CampoOpcional es un cambio de un campo anulable: si Presente, se escribe Valor (nil = NULL).
type CampoOpcional struct {
	Presente bool
	Valor    *string
}

// CambiosNegocio: solo los campos presentes se modifican.
type CambiosNegocio struct {
	RazonSocial, NombreComercial, DireccionFiscal, Moneda, ZonaHoraria *string
	Ubigeo, Telefono, Email                                            CampoOpcional
}

// NegocioStore escribe la empresa única. found=false si no existe.
type NegocioStore interface {
	Actualizar(ctx context.Context, c CambiosNegocio) (dtoresult.Negocio, bool, error)
}

// NegocioReader lee la empresa única.
type NegocioReader interface {
	ObtenerNegocio(ctx context.Context) (dtoresult.Negocio, bool, error)
}
```

`application/usecase/command/negocio_test.go`:

```go
package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type negocioStoreFake struct {
	recibido outbound.CambiosNegocio
	llamadas int
}

func (f *negocioStoreFake) Actualizar(_ context.Context, c outbound.CambiosNegocio) (dtoresult.Negocio, bool, error) {
	f.llamadas++
	f.recibido = c
	return dtoresult.Negocio{ID: 1, RUC: "20123456789"}, true, nil
}

func actorNegocio(permiso string) appctx.Actor {
	return appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: 1, Permiso: permiso}})
}

func texto(s string) *string { return &s }

func TestEditarNegocioRechazos(t *testing.T) {
	admin := actorNegocio(policy.NegocioAdministrar)
	casos := []struct {
		nombre string
		cmd    cmd.EditarNegocio
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", cmd.EditarNegocio{RazonSocial: texto("X")}, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", cmd.EditarNegocio{Actor: actorNegocio(policy.NegocioVer), RazonSocial: texto("X")}, apperror.KindForbidden, "sin_permiso"},
		{"vacío", cmd.EditarNegocio{Actor: admin}, apperror.KindValidation, "sin_cambios"},
		{"razón social en blanco", cmd.EditarNegocio{Actor: admin, RazonSocial: texto("   ")}, apperror.KindValidation, "razon_social_invalida"},
		{"moneda", cmd.EditarNegocio{Actor: admin, Moneda: texto("pe")}, apperror.KindValidation, "moneda_invalida"},
		{"zona horaria", cmd.EditarNegocio{Actor: admin, ZonaHoraria: texto("Marte/Olimpo")}, apperror.KindValidation, "zona_horaria_invalida"},
		{"ubigeo", cmd.EditarNegocio{Actor: admin, Ubigeo: cmd.Campo[string]{Presente: true, Valor: texto("12")}}, apperror.KindValidation, "ubigeo_invalido"},
		{"correo", cmd.EditarNegocio{Actor: admin, Email: cmd.Campo[string]{Presente: true, Valor: texto("no-es-correo")}}, apperror.KindValidation, "email_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			store := &negocioStoreFake{}
			res, err := NewEditarNegocio(store).Execute(context.Background(), c.cmd)
			if err != nil || !res.IsValid() || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v %+v", err, res)
			}
			if res.Err().Kind != c.kind || res.Err().Code != c.code {
				t.Fatalf("rechazo %s/%s, quería %s/%s", res.Err().Kind, res.Err().Code, c.kind, c.code)
			}
			if store.llamadas != 0 {
				t.Fatal("un rechazo no debe tocar el almacenamiento")
			}
		})
	}
}

func TestEditarNegocioNormalizaYVaciaCamposOpcionales(t *testing.T) {
	store := &negocioStoreFake{}
	res, err := NewEditarNegocio(store).Execute(context.Background(), cmd.EditarNegocio{
		Actor:           actorNegocio(policy.NegocioAdministrar),
		NombreComercial: texto("  La Casona  "),
		Moneda:          texto(" pen "),
		Telefono:        cmd.Campo[string]{Presente: true}, // null → vaciar
		Ubigeo:          cmd.Campo[string]{Presente: true, Valor: texto("150101")},
	})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía aceptarse: %v %+v", err, res)
	}
	c := store.recibido
	if *c.NombreComercial != "La Casona" || *c.Moneda != "PEN" {
		t.Errorf("no normalizó: %q %q", *c.NombreComercial, *c.Moneda)
	}
	if !c.Telefono.Presente || c.Telefono.Valor != nil {
		t.Errorf("null debe vaciar el teléfono: %+v", c.Telefono)
	}
	if !c.Ubigeo.Presente || *c.Ubigeo.Valor != "150101" {
		t.Errorf("ubigeo: %+v", c.Ubigeo)
	}
	if c.Email.Presente || c.RazonSocial != nil {
		t.Error("los campos no enviados no deben modificarse")
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `go test ./internal/modules/identidad/application/usecase/command/ -run TestEditarNegocio 2>&1 | head`
Expected: FAIL de compilación (`undefined: NewEditarNegocio`).

- [ ] **Step 3: Implementar los casos de uso**

`application/usecase/command/configuracion_helpers.go`:

```go
package command

import (
	"strings"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/vo"
)

// opcionalValidado normaliza un campo anulable de un PATCH: ausente → no se toca; null o en
// blanco → se vacía; con valor → se valida con norm. ok=false si el valor es inválido.
func opcionalValidado(c cmd.Campo[string], norm func(string) (string, bool)) (outbound.CampoOpcional, bool) {
	if !c.Presente {
		return outbound.CampoOpcional{}, true
	}
	if c.Valor == nil || strings.TrimSpace(*c.Valor) == "" {
		return outbound.CampoOpcional{Presente: true}, true
	}
	v, ok := norm(*c.Valor)
	if !ok {
		return outbound.CampoOpcional{}, false
	}
	return outbound.CampoOpcional{Presente: true, Valor: &v}, true
}

// valorOpcional valida un texto anulable de un alta: nil o en blanco → nil.
func valorOpcional(p *string, norm func(string) (string, bool)) (*string, bool) {
	c, ok := opcionalValidado(cmd.Campo[string]{Presente: true, Valor: p}, norm)
	return c.Valor, ok
}

func normUbigeo(s string) (string, bool) {
	u, err := vo.NewUbigeo(s)
	return u.String(), err == nil
}

func normMax(max int) func(string) (string, bool) {
	return func(s string) (string, bool) {
		v, err := vo.TextoObligatorio(s, max)
		return v, err == nil
	}
}

func normEmail(s string) (string, bool) {
	v := strings.TrimSpace(s)
	e, ok := emailOpcional(&v)
	if !ok || e == nil {
		return "", false
	}
	return *e, true
}
```

`application/usecase/command/negocio.go`:

```go
package command

import (
	"context"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	"service-restaurant/internal/modules/identidad/domain/vo"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxRazonSocial     = 250
	maxNombreComercial = 150
	maxDireccionFiscal = 500
	maxTelefono        = 30
)

// EditarNegocio modifica solo los campos enviados de la empresa única (RES-RF-003).
type EditarNegocio struct{ negocio outbound.NegocioStore }

func NewEditarNegocio(n outbound.NegocioStore) *EditarNegocio { return &EditarNegocio{n} }

func (uc *EditarNegocio) Execute(ctx context.Context, c cmd.EditarNegocio) (appresult.Result[dtoresult.Negocio], error) {
	if ae := mapper.Requiere(c.Actor, "no tiene permiso para editar el negocio", policy.NegocioAdministrar); ae != nil {
		return rechazar[dtoresult.Negocio](ae)
	}
	if c.RazonSocial == nil && c.NombreComercial == nil && c.DireccionFiscal == nil && c.Moneda == nil &&
		c.ZonaHoraria == nil && !c.Ubigeo.Presente && !c.Telefono.Presente && !c.Email.Presente {
		return rechazar[dtoresult.Negocio](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosNegocio
	if c.RazonSocial != nil {
		v, err := vo.TextoObligatorio(*c.RazonSocial, maxRazonSocial)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("razon_social_invalida", "la razón social es obligatoria (máximo 250 caracteres)"))
		}
		cambios.RazonSocial = &v
	}
	if c.NombreComercial != nil {
		v, err := vo.TextoObligatorio(*c.NombreComercial, maxNombreComercial)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("nombre_comercial_invalido", "el nombre comercial es obligatorio (máximo 150 caracteres)"))
		}
		cambios.NombreComercial = &v
	}
	if c.DireccionFiscal != nil {
		v, err := vo.TextoObligatorio(*c.DireccionFiscal, maxDireccionFiscal)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("direccion_fiscal_invalida", "la dirección fiscal es obligatoria (máximo 500 caracteres)"))
		}
		cambios.DireccionFiscal = &v
	}
	if c.Moneda != nil {
		m, err := vo.NewMoneda(*c.Moneda)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("moneda_invalida", "la moneda debe ser un código de tres letras, como PEN"))
		}
		v := m.String()
		cambios.Moneda = &v
	}
	if c.ZonaHoraria != nil {
		z, err := vo.NewZonaHoraria(*c.ZonaHoraria)
		if err != nil {
			return rechazar[dtoresult.Negocio](mapper.Validacion("zona_horaria_invalida", "la zona horaria debe ser un nombre IANA, como America/Lima"))
		}
		v := z.String()
		cambios.ZonaHoraria = &v
	}
	var ok bool
	if cambios.Ubigeo, ok = opcionalValidado(c.Ubigeo, normUbigeo); !ok {
		return rechazar[dtoresult.Negocio](mapper.Validacion("ubigeo_invalido", "el ubigeo debe tener 6 dígitos"))
	}
	if cambios.Telefono, ok = opcionalValidado(c.Telefono, normMax(maxTelefono)); !ok {
		return rechazar[dtoresult.Negocio](mapper.Validacion("telefono_invalido", "el teléfono excede 30 caracteres"))
	}
	if cambios.Email, ok = opcionalValidado(c.Email, normEmail); !ok {
		return rechazar[dtoresult.Negocio](mapper.Validacion("email_invalido", "el correo electrónico no es válido"))
	}
	n, found, err := uc.negocio.Actualizar(ctx, cambios)
	if err != nil {
		return deAlmacen[dtoresult.Negocio]("editar negocio", err)
	}
	if !found {
		return rechazar[dtoresult.Negocio](mapper.NoEncontrado("negocio"))
	}
	return appresult.Ok(n), nil
}
```

`application/usecase/query/negocio.go`:

```go
package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// ObtenerNegocio lee la empresa única: basta NEGOCIO_VER o NEGOCIO_ADMINISTRAR en alguna sede.
type ObtenerNegocio struct{ negocio outbound.NegocioReader }

func NewObtenerNegocio(n outbound.NegocioReader) *ObtenerNegocio { return &ObtenerNegocio{n} }

func (uc *ObtenerNegocio) Execute(ctx context.Context, q qry.ObtenerNegocio) (appresult.Result[dtoresult.Negocio], error) {
	if ae := mapper.Requiere(q.Actor, "no tiene permiso para consultar el negocio", policy.NegocioVer, policy.NegocioAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Negocio](ae), nil
	}
	n, found, err := uc.negocio.ObtenerNegocio(ctx)
	if err != nil {
		return appresult.Result[dtoresult.Negocio]{}, fmt.Errorf("obtener negocio: %w", err)
	}
	if !found {
		return appresult.Fail[dtoresult.Negocio](mapper.NoEncontrado("negocio")), nil
	}
	return appresult.Ok(n), nil
}
```

- [ ] **Step 4: Ejecutar las pruebas unitarias**

Run: `gofmt -l internal/ ; go test ./internal/modules/identidad/application/... 2>&1 | tail`
Expected: `ok` en `usecase/command` y `mapper`; sin salida de `gofmt -l`.

- [ ] **Step 5: Persistencia (escritura y lectura)**

`infrastructure/persistence/write/repository/configuracion_sql.go`:

```go
package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

const negocioCols = `id, ruc, razon_social, nombre_comercial, direccion_fiscal, ubigeo, telefono, email,
	moneda, zona_horaria, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func scanNegocio(s scanner) (dtoresult.Negocio, error) {
	var n dtoresult.Negocio
	var ubigeo, telefono, email sql.NullString
	err := s.Scan(&n.ID, &n.RUC, &n.RazonSocial, &n.NombreComercial, &n.DireccionFiscal, &ubigeo, &telefono, &email,
		&n.Moneda, &n.ZonaHoraria, &n.EsActivo, &n.CreadoEn, &n.CreadoPor, &n.ActualizadoEn, &n.ActualizadoPor)
	n.Ubigeo, n.Telefono, n.Email = textoNulo(ubigeo), textoNulo(telefono), textoNulo(email)
	return n, err
}

// NegocioSQLWriter opera sobre sch_restaurante.negocio (singleton).
type NegocioSQLWriter struct{ db database.DBTX }

func NewNegocioSQLWriter(db database.DBTX) *NegocioSQLWriter { return &NegocioSQLWriter{db} }

// Update aplica solo los campos presentes. El RUC y el singleton no forman parte del SQL.
func (r *NegocioSQLWriter) Update(ctx context.Context, c outbound.CambiosNegocio) (dtoresult.Negocio, bool, error) {
	n, err := scanNegocio(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.negocio SET
			razon_social = COALESCE($1, razon_social),
			nombre_comercial = COALESCE($2, nombre_comercial),
			direccion_fiscal = COALESCE($3, direccion_fiscal),
			ubigeo = CASE WHEN $4 THEN $5 ELSE ubigeo END,
			telefono = CASE WHEN $6 THEN $7 ELSE telefono END,
			email = CASE WHEN $8 THEN $9 ELSE email END,
			moneda = COALESCE($10, moneda),
			zona_horaria = COALESCE($11, zona_horaria)
		WHERE singleton RETURNING `+negocioCols,
		c.RazonSocial, c.NombreComercial, c.DireccionFiscal,
		c.Ubigeo.Presente, c.Ubigeo.Valor, c.Telefono.Presente, c.Telefono.Valor, c.Email.Presente, c.Email.Valor,
		c.Moneda, c.ZonaHoraria))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Negocio{}, false, Translate(err)
	}
	return n, true, nil
}
```

`infrastructure/persistence/write/adapter/configuracion.go`:

```go
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// NegocioWriteAdapter implementa outbound.NegocioStore.
type NegocioWriteAdapter struct{ repo *repository.NegocioSQLWriter }

func NewNegocioWriteAdapter(db database.DBTX) *NegocioWriteAdapter {
	return &NegocioWriteAdapter{repo: repository.NewNegocioSQLWriter(db)}
}

var _ outbound.NegocioStore = (*NegocioWriteAdapter)(nil)

func (a *NegocioWriteAdapter) Actualizar(ctx context.Context, c outbound.CambiosNegocio) (dtoresult.Negocio, bool, error) {
	return a.repo.Update(ctx, c)
}
```

`infrastructure/persistence/read/repository/configuracion_sql_reader.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/platform/database"
)

// ConfiguracionSQLReader lee negocio y sucursales.
type ConfiguracionSQLReader struct{ db database.DBTX }

func NewConfiguracionSQLReader(db database.DBTX) *ConfiguracionSQLReader {
	return &ConfiguracionSQLReader{db}
}

type filaConfig interface{ Scan(...any) error }

func cadenaNula(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

const negocioColsLectura = `id, ruc, razon_social, nombre_comercial, direccion_fiscal, ubigeo, telefono, email,
	moneda, zona_horaria, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

func leerNegocio(s filaConfig) (dtoresult.Negocio, error) {
	var n dtoresult.Negocio
	var ubigeo, telefono, email sql.NullString
	err := s.Scan(&n.ID, &n.RUC, &n.RazonSocial, &n.NombreComercial, &n.DireccionFiscal, &ubigeo, &telefono, &email,
		&n.Moneda, &n.ZonaHoraria, &n.EsActivo, &n.CreadoEn, &n.CreadoPor, &n.ActualizadoEn, &n.ActualizadoPor)
	n.Ubigeo, n.Telefono, n.Email = cadenaNula(ubigeo), cadenaNula(telefono), cadenaNula(email)
	return n, err
}

// GetNegocio devuelve la empresa única; found=false si aún no se instaló.
func (r *ConfiguracionSQLReader) GetNegocio(ctx context.Context) (dtoresult.Negocio, bool, error) {
	n, err := leerNegocio(r.db.QueryRowContext(ctx,
		`SELECT `+negocioColsLectura+` FROM sch_restaurante.negocio WHERE singleton`))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Negocio{}, false, nil
	}
	return n, err == nil, err
}
```

`infrastructure/persistence/read/adapter/configuracion_read_adapter.go`:

```go
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ConfiguracionReadAdapter implementa los lectores de negocio y sucursales.
type ConfiguracionReadAdapter struct{ repo *repository.ConfiguracionSQLReader }

func NewConfiguracionReadAdapter(db database.DBTX) *ConfiguracionReadAdapter {
	return &ConfiguracionReadAdapter{repo: repository.NewConfiguracionSQLReader(db)}
}

var _ outbound.NegocioReader = (*ConfiguracionReadAdapter)(nil)

func (a *ConfiguracionReadAdapter) ObtenerNegocio(ctx context.Context) (dtoresult.Negocio, bool, error) {
	return a.repo.GetNegocio(ctx)
}
```

- [ ] **Step 6: Capa web**

`web/dto/request/configuracion.go`:

```go
package request

// NegocioEdicion: el RUC y el singleton no se editan (un campo desconocido responde 400).
// Los campos anulables usan Opcional para distinguir "no enviado" de null.
type NegocioEdicion struct {
	RazonSocial     *string          `json:"razon_social,omitempty"`
	NombreComercial *string          `json:"nombre_comercial,omitempty"`
	DireccionFiscal *string          `json:"direccion_fiscal,omitempty"`
	Ubigeo          Opcional[string] `json:"ubigeo"`
	Telefono        Opcional[string] `json:"telefono"`
	Email           Opcional[string] `json:"email"`
	Moneda          *string          `json:"moneda,omitempty"`
	ZonaHoraria     *string          `json:"zona_horaria,omitempty"`
}
```

`web/dto/response/configuracion.go`:

```go
package response

import "time"

type Negocio struct {
	ID              string    `json:"id"`
	Singleton       bool      `json:"singleton"`
	RUC             string    `json:"ruc"`
	RazonSocial     string    `json:"razon_social"`
	NombreComercial string    `json:"nombre_comercial"`
	DireccionFiscal string    `json:"direccion_fiscal"`
	Ubigeo          *string   `json:"ubigeo"`
	Telefono        *string   `json:"telefono"`
	Email           *string   `json:"email"`
	Moneda          string    `json:"moneda"`
	ZonaHoraria     string    `json:"zona_horaria"`
	EsActivo        bool      `json:"es_activo"`
	CreadoEn        time.Time `json:"creado_en"`
	CreadoPor       string    `json:"creado_por"`
	ActualizadoEn   time.Time `json:"actualizado_en"`
	ActualizadoPor  string    `json:"actualizado_por"`
}
```

`web/mapper/configuracion_web_mapper.go`:

```go
package mapper

import (
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/web/dto/response"
)

func ToNegocio(n dtoresult.Negocio) response.Negocio {
	return response.Negocio{
		ID: id(n.ID), Singleton: true, RUC: n.RUC, RazonSocial: n.RazonSocial, NombreComercial: n.NombreComercial,
		DireccionFiscal: n.DireccionFiscal, Ubigeo: n.Ubigeo, Telefono: n.Telefono, Email: n.Email,
		Moneda: n.Moneda, ZonaHoraria: n.ZonaHoraria, EsActivo: n.EsActivo,
		CreadoEn: n.CreadoEn, CreadoPor: id(n.CreadoPor), ActualizadoEn: n.ActualizadoEn, ActualizadoPor: id(n.ActualizadoPor),
	}
}
```

`web/controller/negocio_handler.go`:

```go
package controller

import (
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// NegocioHandler: lectura y edición de la empresa única (RES-RF-003).
type NegocioHandler struct {
	obtener inbound.ObtenerNegocio
	editar  inbound.EditarNegocio
}

func NewNegocioHandler(o inbound.ObtenerNegocio, e inbound.EditarNegocio) *NegocioHandler {
	return &NegocioHandler{o, e}
}

// Obtener: GET /negocio
func (h *NegocioHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerNegocio{Actor: actor})
	return responder(w, res, err, http.StatusOK, webmapper.ToNegocio)
}

// Editar: PATCH /negocio
func (h *NegocioHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.NegocioEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarNegocio{
		Actor: actor, RazonSocial: body.RazonSocial, NombreComercial: body.NombreComercial,
		DireccionFiscal: body.DireccionFiscal, Moneda: body.Moneda, ZonaHoraria: body.ZonaHoraria,
		Ubigeo:   command.Campo[string]{Presente: body.Ubigeo.Presente, Valor: body.Ubigeo.Valor},
		Telefono: command.Campo[string]{Presente: body.Telefono.Presente, Valor: body.Telefono.Valor},
		Email:    command.Campo[string]{Presente: body.Email.Presente, Valor: body.Email.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToNegocio)
}
```

`web/controller/configuracion_routes.go`:

```go
package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterConfiguracionRoutes monta negocio y sucursales (todas exigen autenticación; el
// permiso fino lo verifica cada caso de uso).
func RegisterConfiguracionRoutes(mux *http.ServeMux, negocio *NegocioHandler,
	protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /negocio", negocio.Obtener)
	ruta("PATCH /negocio", negocio.Editar)
}
```

- [ ] **Step 7: Composición**

`infrastructure/config/configuracion.go`:

```go
package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/application/usecase/command"
	uquery "service-restaurant/internal/modules/identidad/application/usecase/query"
	readadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/identidad/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/identidad/web/controller"
)

// configuracion agrupa los casos de uso de negocio y sucursales (RES-RF-003) ya decorados.
type configuracion struct {
	negocio *controller.NegocioHandler
}

func armarConfiguracion(db *sql.DB, logger *slog.Logger) *configuracion {
	sinIntencion := func(actorID int64) (int64, cmd.Intencion) { return actorID, cmd.Intencion{} }

	editarNegocio := escritura[cmd.EditarNegocio, dtoresult.Negocio]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarNegocio, dtoresult.Negocio] {
			return command.NewEditarNegocio(writeadapter.NewNegocioWriteAdapter(q))
		},
		meta: func(c cmd.EditarNegocio) (int64, cmd.Intencion) { return sinIntencion(c.Actor.UserID) }}

	lector := readadapter.NewConfiguracionReadAdapter(db)
	return &configuracion{
		negocio: controller.NewNegocioHandler(uquery.NewObtenerNegocio(lector), inbound.EditarNegocio(editarNegocio)),
	}
}
```

En `infrastructure/config/module.go`: añadir el campo `conf *configuracion` a `Module` (junto a `admin`), la línea `m.conf = armarConfiguracion(d.DB, d.Logger)` justo después de `m.admin = armarAdministracion(...)`, y en `RegisterRoutes`:

```go
	controller.RegisterConfiguracionRoutes(mux, m.conf.negocio, protect, m.deps.Logger)
```

- [ ] **Step 8: Compilar**

Run: `go build ./... && go vet ./internal/modules/identidad/... && go test ./tests/architecture/`
Expected: sin errores; arquitectura en verde.

- [ ] **Step 9: Pruebas de integración de negocio (fallan antes de Step 7; deben pasar ahora)**

Ampliar `tests/integration/configuracion_helpers_test.go` (añadir imports `fmt`, `testing`, `service-restaurant/internal/modules/identidad/infrastructure/security`):

```go
// rolConPermisos garantiza un rol (por código) con los permisos indicados. Es idempotente.
func rolConPermisos(t *testing.T, codigo string, permisos ...string) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.roles(codigo,nombre)
		SELECT '`+codigo+`','`+codigo+`' WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.roles WHERE codigo='`+codigo+`')`)
	for _, p := range permisos {
		asSystem(t, `INSERT INTO sch_restaurante.roles_permisos(rol_id,permiso_id)
			SELECT r.id,p.id FROM sch_restaurante.roles r, sch_restaurante.permisos p
			WHERE r.codigo='`+codigo+`' AND p.codigo='`+p+`'
			AND NOT EXISTS (SELECT 1 FROM sch_restaurante.roles_permisos x WHERE x.rol_id=r.id AND x.permiso_id=p.id)`)
	}
}

// asegurarUsuario crea el usuario humano si no existe (con el rol dado) y le asigna las sedes
// indicadas (por código) que aún no tenga. Es idempotente: varias pruebas comparten la base.
func asegurarUsuario(t *testing.T, username, password, rol string, sedes ...string) {
	t.Helper()
	app(t)
	if scalar(t, `SELECT count(*) FROM sch_restaurante.usuarios WHERE username='`+username+`'`) == 0 {
		h, err := security.NewArgon2Hasher(security.Argon2ParamsTest)
		if err != nil {
			t.Fatal(err)
		}
		hash, _ := h.Hash(password)
		asSystem(t,
			`INSERT INTO sch_restaurante.usuarios(username, password_hash) VALUES ('`+username+`', '`+hash+`')`,
			`INSERT INTO sch_restaurante.usuarios_roles(usuario_id, rol_id)
			 SELECT u.id, r.id FROM sch_restaurante.usuarios u, sch_restaurante.roles r
			 WHERE u.username='`+username+`' AND r.codigo='`+rol+`'`)
	}
	for _, s := range sedes {
		asSystem(t, `INSERT INTO sch_restaurante.usuarios_sucursales(usuario_id, sucursal_id)
			SELECT u.id, s.id FROM sch_restaurante.usuarios u, sch_restaurante.sucursales s
			WHERE u.username='`+username+`' AND s.codigo='`+s+`'
			AND NOT EXISTS (SELECT 1 FROM sch_restaurante.usuarios_sucursales x WHERE x.usuario_id=u.id AND x.sucursal_id=s.id)`)
	}
}

// crearSedeSQL inserta una sede activa si no existe (para preparar escenarios).
func crearSedeSQL(t *testing.T, codigo string) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.sucursales(negocio_id, codigo, nombre, direccion)
		SELECT id, '`+codigo+`', 'Sede `+codigo+`', 'Calle de prueba 1' FROM sch_restaurante.negocio
		WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.sucursales WHERE codigo='`+codigo+`')`)
}

// sedeResp es la sucursal tal como la devuelve la API.
type sedeResp struct {
	ID           string  `json:"id"`
	NegocioID    string  `json:"negocio_id"`
	Codigo       string  `json:"codigo"`
	Nombre       string  `json:"nombre"`
	Direccion    string  `json:"direccion"`
	Ubigeo       *string `json:"ubigeo"`
	CodigoSunat  *string `json:"codigo_establecimiento_sunat"`
	Telefono     *string `json:"telefono"`
	EsActivo     bool    `json:"es_activo"`
	Motivo       *string `json:"motivo_eliminacion"`
	EliminadoPor *string `json:"eliminado_por"`
	CreadoPor    string  `json:"creado_por"`
}
```

(En este archivo no se necesita `fmt`: los imports son `testing` y `security`, además de `context` y `database` ya presentes.)

`tests/integration/configuracion_negocio_test.go`:

```go
package integration

import (
	"net/http"
	"testing"
)

type negocioResp struct {
	ID              string  `json:"id"`
	Singleton       bool    `json:"singleton"`
	RUC             string  `json:"ruc"`
	RazonSocial     string  `json:"razon_social"`
	NombreComercial string  `json:"nombre_comercial"`
	DireccionFiscal string  `json:"direccion_fiscal"`
	Ubigeo          *string `json:"ubigeo"`
	Telefono        *string `json:"telefono"`
	Email           *string `json:"email"`
	Moneda          string  `json:"moneda"`
	ZonaHoraria     string  `json:"zona_horaria"`
	ActualizadoPor  string  `json:"actualizado_por"`
}

func TestNegocioLecturaYPermisos(t *testing.T) {
	rolConPermisos(t, "T_NEGOCIO_VER", "NEGOCIO_VER")
	asegurarUsuario(t, "neg.ver", "NegVer2026Clave", "T_NEGOCIO_VER", "PRINCIPAL")
	asegurarUsuario(t, "neg.sin", "NegSin2026Clave", "CAJERO", "PRINCIPAL")

	esperar(t, call(t, "GET", "/api/v1/negocio", "", ""), http.StatusUnauthorized, "no_autenticado")
	esperar(t, call(t, "GET", "/api/v1/negocio", "", token(t, "neg.sin", "NegSin2026Clave")), http.StatusForbidden, "sin_permiso")

	for _, tk := range []string{adminToken(t), token(t, "neg.ver", "NegVer2026Clave")} {
		r := call(t, "GET", "/api/v1/negocio", "", tk)
		esperar(t, r, http.StatusOK, "")
		var n negocioResp
		r.json(t, &n)
		if n.RUC != "20123456789" || !n.Singleton || n.Moneda != "PEN" || n.ZonaHoraria != "America/Lima" || n.ID == "" {
			t.Fatalf("negocio: %+v", n)
		}
	}
	// Solo lectura: editar exige NEGOCIO_ADMINISTRAR.
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"nombre_comercial":"X"}`, token(t, "neg.ver", "NegVer2026Clave"), nil),
		http.StatusForbidden, "sin_permiso")
}

func TestNegocioEdicionParcialYValidaciones(t *testing.T) {
	tk := adminToken(t)
	t.Cleanup(func() {
		callH(t, "PATCH", "/api/v1/negocio", `{"nombre_comercial":"Prueba","telefono":null,"email":null,"ubigeo":null}`, tk, nil)
	})

	r := callH(t, "PATCH", "/api/v1/negocio",
		`{"nombre_comercial":"La Casona","telefono":"01-555-0100","email":"hola@lacasona.pe","ubigeo":"150101"}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var n negocioResp
	r.json(t, &n)
	if n.NombreComercial != "La Casona" || n.Telefono == nil || *n.Telefono != "01-555-0100" ||
		n.Email == nil || *n.Email != "hola@lacasona.pe" || n.Ubigeo == nil || *n.Ubigeo != "150101" ||
		n.RUC != "20123456789" || n.RazonSocial != "Restaurante de Prueba SAC" {
		t.Fatalf("edición parcial: %+v", n)
	}
	if n.ActualizadoPor != idDe(t, "usuarios", "username", "admin") {
		t.Errorf("actualizado_por = %s, debe ser el administrador de la sesión", n.ActualizadoPor)
	}

	// null vacía solo el campo indicado.
	r = callH(t, "PATCH", "/api/v1/negocio", `{"telefono":null}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &n)
	if n.Telefono != nil || n.NombreComercial != "La Casona" || n.Email == nil {
		t.Fatalf("null debe vaciar solo el teléfono: %+v", n)
	}

	// Normaliza la moneda.
	r = callH(t, "PATCH", "/api/v1/negocio", `{"moneda":" pen "}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &n)
	if n.Moneda != "PEN" {
		t.Fatalf("moneda = %q", n.Moneda)
	}

	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"ruc":"20999999999"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"singleton":false}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"moneda":"pe"}`, tk, nil), http.StatusUnprocessableEntity, "moneda_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"zona_horaria":"Marte/Olimpo"}`, tk, nil), http.StatusUnprocessableEntity, "zona_horaria_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"ubigeo":"12"}`, tk, nil), http.StatusUnprocessableEntity, "ubigeo_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"email":"no-es-correo"}`, tk, nil), http.StatusUnprocessableEntity, "email_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"razon_social":"   "}`, tk, nil), http.StatusUnprocessableEntity, "razon_social_invalida")

	// Los rechazos no modifican nada.
	if scalar(t, `SELECT count(*) FROM sch_restaurante.negocio WHERE ruc='20123456789' AND moneda='PEN' AND razon_social='Restaurante de Prueba SAC'`) != 1 {
		t.Fatal("un rechazo no debe alterar el negocio")
	}
}

func TestNegocioEdicionQuedaAuditadaConElActor(t *testing.T) {
	tk := adminToken(t)
	esperar(t, callH(t, "PATCH", "/api/v1/negocio", `{"nombre_comercial":"Prueba"}`, tk, nil), http.StatusOK, "")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos
		WHERE tabla='negocio' AND accion='UPDATE' AND creado_por=`+idDe(t, "usuarios", "username", "admin")); n == 0 {
		t.Fatal("la edición del negocio debe quedar en auditoria_eventos con el actor de la sesión")
	}
}
```

- [ ] **Step 10: Ejecutar integración de negocio**

Run: `go test -count=1 ./tests/integration/ -run "TestNegocio" -v`
Expected: PASS (3 pruebas).

- [ ] **Step 11: Regresión completa y commit**

Run: `gofmt -l . ; go vet ./... && go test ./... && go test -count=1 ./tests/integration/...`
Expected: todo en verde (las pruebas de I1 siguen pasando).

```bash
git add internal/modules/identidad tests/integration
git commit -m "I2 paso 1: GET y PATCH /negocio con permisos NEGOCIO_VER y NEGOCIO_ADMINISTRAR

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Sucursales — listar y consultar

**Files:**
- Create: `application/dto/result/sucursales.go`, `application/dto/query/sucursales.go`
- Create: `application/port/inbound/sucursales.go`, `application/port/outbound/sucursales.go`
- Create: `application/usecase/query/sucursales.go`, `application/usecase/query/sucursales_test.go`
- Modify: `infrastructure/persistence/read/repository/configuracion_sql_reader.go`, `.../read/adapter/configuracion_read_adapter.go`
- Create: `web/controller/sucursales_handler.go`, `web/controller/configuracion_helpers.go`
- Modify: `web/dto/response/configuracion.go`, `web/mapper/configuracion_web_mapper.go`, `web/controller/configuracion_routes.go`, `infrastructure/config/configuracion.go`
- Create: `tests/integration/configuracion_sucursales_test.go`
- Modify: `docs/superpowers/specs/2026-10-18-i2-configuracion-salon-design.md` (nota de §3)

**Interfaces:**
- Consumes: `mapper.Requiere`, `mapper.SedesVisibles`, `mapper.SucursalVisible`, `mapper.DecodeCursor/EncodeCursor`, `tamanio` (paquete `query`), helpers de integración de la Tarea 3.
- Produces:
  - `result.Sucursal{ID, NegocioID int64; Codigo, Nombre, Direccion string; Ubigeo, CodigoEstablecimientoSunat, Telefono *string; EsActivo bool; MotivoEliminacion *string; EliminadoEn *time.Time; EliminadoPor *int64; CreadoEn time.Time; CreadoPor int64; ActualizadoEn time.Time; ActualizadoPor int64}`, `result.PaginaSucursales{Items []Sucursal; NextCursor string}`.
  - `query.ListarSucursales{Actor; Limite int; Cursor string; Activo *bool}`, `query.ObtenerSucursal{Actor; ID int64}`.
  - `inbound.ListarSucursales`, `inbound.ObtenerSucursal`.
  - `outbound.FiltroSucursales{SedeIDs []int64; IncluirInactivas bool; Activo *bool; DespuesDeID int64; Limite int}`; `outbound.SucursalReader{ListarSucursales(ctx, FiltroSucursales) ([]Sucursal, error); ObtenerSucursal(ctx, id) (Sucursal, bool, error)}`.
  - `controller.NewSucursalesHandler(listar, obtener)`; `response.Sucursal`, `response.SucursalesPagina`; `webmapper.ToSucursal`, `ToSucursalesPagina`.

- [ ] **Step 1: DTO, puertos y pruebas unitarias (fallan)**

`application/dto/result/sucursales.go`:

```go
package result

import "time"

// Sucursal es un local operativo de la empresa (RES-RF-003). Las inactivas llevan motivo y
// actor de la baja.
type Sucursal struct {
	ID                         int64
	NegocioID                  int64
	Codigo                     string
	Nombre                     string
	Direccion                  string
	Ubigeo                     *string
	CodigoEstablecimientoSunat *string
	Telefono                   *string
	EsActivo                   bool
	MotivoEliminacion          *string
	EliminadoEn                *time.Time
	EliminadoPor               *int64
	CreadoEn                   time.Time
	CreadoPor                  int64
	ActualizadoEn              time.Time
	ActualizadoPor             int64
}

// PaginaSucursales: NextCursor vacío indica la última página.
type PaginaSucursales struct {
	Items      []Sucursal
	NextCursor string
}
```

`application/dto/query/sucursales.go`:

```go
package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarSucursales pagina por cursor (id ascendente) las sedes que el actor puede ver.
// Activo filtra por es_activo cuando no es nil.
type ListarSucursales struct {
	Actor  appctx.Actor
	Limite int
	Cursor string
	Activo *bool
}

type ObtenerSucursal struct {
	Actor appctx.Actor
	ID    int64
}
```

`application/port/inbound/sucursales.go`:

```go
package inbound

import (
	"context"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarSucursales interface {
	Execute(context.Context, qry.ListarSucursales) (appresult.Result[dtoresult.PaginaSucursales], error)
}
type ObtenerSucursal interface {
	Execute(context.Context, qry.ObtenerSucursal) (appresult.Result[dtoresult.Sucursal], error)
}
```

`application/port/outbound/sucursales.go`:

```go
package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
)

// FiltroSucursales acota el listado a lo que el actor puede ver: las sedes activas de SedeIDs
// y, si IncluirInactivas, todas las inactivas (quien administra sedes debe poder restaurarlas).
type FiltroSucursales struct {
	SedeIDs          []int64
	IncluirInactivas bool
	Activo           *bool
	DespuesDeID      int64
	Limite           int
}

// SucursalReader lee sedes sin bloquear filas.
type SucursalReader interface {
	ListarSucursales(ctx context.Context, f FiltroSucursales) ([]dtoresult.Sucursal, error)
	ObtenerSucursal(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error)
}
```

`application/usecase/query/sucursales_test.go`:

```go
package query

import (
	"context"
	"slices"
	"testing"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type sucursalReaderFake struct {
	filtro   outbound.FiltroSucursales
	llamadas int
	filas    []dtoresult.Sucursal
	una      dtoresult.Sucursal
}

func (f *sucursalReaderFake) ListarSucursales(_ context.Context, fl outbound.FiltroSucursales) ([]dtoresult.Sucursal, error) {
	f.llamadas++
	f.filtro = fl
	return f.filas, nil
}
func (f *sucursalReaderFake) ObtenerSucursal(_ context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	f.llamadas++
	return f.una, f.una.ID == id, nil
}

func actorSedes(g ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, g) }

func TestListarSucursalesAcotaPorLoQueElActorVe(t *testing.T) {
	f := &sucursalReaderFake{}
	a := actorSedes(
		appctx.Grant{SucursalID: 4, Permiso: policy.SucursalVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.SucursalAdministrar})
	res, err := NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: a, Limite: 10})
	if err != nil || !res.IsOk() {
		t.Fatalf("listar: %v %+v", err, res)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{2, 4}) || !f.filtro.IncluirInactivas || f.filtro.Limite != 11 {
		t.Fatalf("filtro: %+v (quien administra sedes ve también las inactivas; se pide limite+1)", f.filtro)
	}
	soloVer := actorSedes(appctx.Grant{SucursalID: 4, Permiso: policy.SucursalVer})
	if _, err := NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: soloVer}); err != nil {
		t.Fatal(err)
	}
	if f.filtro.IncluirInactivas {
		t.Error("quien solo tiene VER no debe ver sedes inactivas")
	}
}

func TestListarSucursalesPaginaYRechazos(t *testing.T) {
	f := &sucursalReaderFake{filas: []dtoresult.Sucursal{{ID: 1}, {ID: 2}, {ID: 3}}}
	a := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalVer})
	res, _ := NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: a, Limite: 2})
	p := res.Value()
	if len(p.Items) != 2 || p.NextCursor == "" {
		t.Fatalf("debía recortar a 2 y dar cursor: %+v", p)
	}

	sinPermiso := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.NegocioVer})
	f.llamadas = 0
	res, _ = NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: sinPermiso})
	if res.Err() == nil || res.Err().Kind != apperror.KindForbidden || f.llamadas != 0 {
		t.Fatalf("sin permiso debe ser 403 sin tocar el lector: %+v", res.Err())
	}
	res, _ = NewListarSucursales(f).Execute(context.Background(), qry.ListarSucursales{Actor: a, Cursor: "%%%"})
	if res.Err() == nil || res.Err().Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido debe ser 400: %+v", res.Err())
	}
}

func TestObtenerSucursalOcultaLasAjenas(t *testing.T) {
	f := &sucursalReaderFake{una: dtoresult.Sucursal{ID: 9, EsActivo: true}}
	a := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalVer})
	res, _ := NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: a, ID: 9})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una sede activa ajena debe ser 404: %+v", res.Err())
	}
	f.una = dtoresult.Sucursal{ID: 1, EsActivo: true}
	res, _ = NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: a, ID: 1})
	if !res.IsOk() || res.Value().ID != 1 {
		t.Fatalf("una sede propia debe leerse: %+v", res.Err())
	}
	// Inactiva: la ve quien administra sedes; quien solo tiene VER no.
	f.una = dtoresult.Sucursal{ID: 5, EsActivo: false}
	res, _ = NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: a, ID: 5})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una sede inactiva es invisible para VER: %+v", res.Err())
	}
	admin := actorSedes(appctx.Grant{SucursalID: 1, Permiso: policy.SucursalAdministrar})
	res, _ = NewObtenerSucursal(f).Execute(context.Background(), qry.ObtenerSucursal{Actor: admin, ID: 5})
	if !res.IsOk() {
		t.Fatalf("quien administra sedes ve las inactivas: %+v", res.Err())
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `go test ./internal/modules/identidad/application/usecase/query/ -run "Sucursal" 2>&1 | head`
Expected: FAIL de compilación (`undefined: NewListarSucursales`).

- [ ] **Step 3: Implementar los casos de uso**

`application/usecase/query/sucursales.go`:

```go
package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/identidad/application/dto/query"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// ListarSucursales devuelve solo las sedes que el actor puede ver. Pide limite+1 filas para
// saber si hay más páginas.
type ListarSucursales struct{ sucursales outbound.SucursalReader }

func NewListarSucursales(s outbound.SucursalReader) *ListarSucursales { return &ListarSucursales{s} }

func (uc *ListarSucursales) Execute(ctx context.Context, q qry.ListarSucursales) (appresult.Result[dtoresult.PaginaSucursales], error) {
	if ae := mapper.Requiere(q.Actor, "no tiene permiso para consultar sucursales", policy.SucursalVer, policy.SucursalAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaSucursales](ae), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaSucursales](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.sucursales.ListarSucursales(ctx, outbound.FiltroSucursales{
		SedeIDs:          mapper.SedesVisibles(q.Actor),
		IncluirInactivas: q.Actor.CanAnywhere(policy.SucursalAdministrar),
		Activo:           q.Activo,
		DespuesDeID:      desde,
		Limite:           n + 1,
	})
	if err != nil {
		return appresult.Result[dtoresult.PaginaSucursales]{}, fmt.Errorf("listar sucursales: %w", err)
	}
	pagina := dtoresult.PaginaSucursales{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Sucursal{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerSucursal lee una sede; una sede que el actor no ve responde 404 para no revelarla.
type ObtenerSucursal struct{ sucursales outbound.SucursalReader }

func NewObtenerSucursal(s outbound.SucursalReader) *ObtenerSucursal { return &ObtenerSucursal{s} }

func (uc *ObtenerSucursal) Execute(ctx context.Context, q qry.ObtenerSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(q.Actor, "no tiene permiso para consultar sucursales", policy.SucursalVer, policy.SucursalAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Sucursal](ae), nil
	}
	s, found, err := uc.sucursales.ObtenerSucursal(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Sucursal]{}, fmt.Errorf("obtener sucursal: %w", err)
	}
	if !found || !mapper.SucursalVisible(q.Actor, s.ID, s.EsActivo) {
		return appresult.Fail[dtoresult.Sucursal](mapper.NoEncontrado("sucursal")), nil
	}
	return appresult.Ok(s), nil
}
```

- [ ] **Step 4: Ejecutar las pruebas unitarias**

Run: `go test ./internal/modules/identidad/application/... 2>&1 | tail`
Expected: PASS.

- [ ] **Step 5: Lector SQL y adaptador**

Añadir a `infrastructure/persistence/read/repository/configuracion_sql_reader.go` (el parámetro `[]int64` con `$1::bigint[]` ya lo usa `ListSucursalesActivas` del lector de usuarios, así que no hace falta ninguna librería nueva):

```go
const sucursalColsLectura = `id, negocio_id, codigo, nombre, direccion, ubigeo, codigo_establecimiento_sunat, telefono,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func leerSucursal(s filaConfig) (dtoresult.Sucursal, error) {
	var x dtoresult.Sucursal
	var ubigeo, sunat, tel, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&x.ID, &x.NegocioID, &x.Codigo, &x.Nombre, &x.Direccion, &ubigeo, &sunat, &tel,
		&x.EsActivo, &motivo, &elimEn, &elimPor, &x.CreadoEn, &x.CreadoPor, &x.ActualizadoEn, &x.ActualizadoPor)
	x.Ubigeo, x.CodigoEstablecimientoSunat, x.Telefono, x.MotivoEliminacion =
		cadenaNula(ubigeo), cadenaNula(sunat), cadenaNula(tel), cadenaNula(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		x.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		x.EliminadoPor = &v
	}
	return x, err
}

// ListSucursales devuelve las sedes activas de f.SedeIDs y, si f.IncluirInactivas, todas las
// inactivas; ordenadas por id para un cursor estable.
func (r *ConfiguracionSQLReader) ListSucursales(ctx context.Context, f outbound.FiltroSucursales) ([]dtoresult.Sucursal, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+sucursalColsLectura+` FROM sch_restaurante.sucursales
		WHERE ((es_activo AND id = ANY($1::bigint[])) OR (NOT es_activo AND $2))
		  AND id > $3 AND ($4::boolean IS NULL OR es_activo = $4)
		ORDER BY id LIMIT $5`, f.SedeIDs, f.IncluirInactivas, f.DespuesDeID, activo, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Sucursal
	for rows.Next() {
		s, err := leerSucursal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ConfiguracionSQLReader) GetSucursal(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	s, err := leerSucursal(r.db.QueryRowContext(ctx,
		`SELECT `+sucursalColsLectura+` FROM sch_restaurante.sucursales WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Sucursal{}, false, nil
	}
	return s, err == nil, err
}
```

Añadir el import `"service-restaurant/internal/modules/identidad/application/port/outbound"` al archivo. **Nota:** el lector de usuarios del repo ya pasa `[]int64` con `ANY($1::bigint[])` (`ListSucursalesActivas`), así que el driver acepta ese parámetro tal cual; si `SedeIDs` es nil se envía `NULL` y `id = ANY(NULL)` no coincide con nada, lo cual es correcto.

Añadir a `infrastructure/persistence/read/adapter/configuracion_read_adapter.go`:

```go
var _ outbound.SucursalReader = (*ConfiguracionReadAdapter)(nil)

func (a *ConfiguracionReadAdapter) ListarSucursales(ctx context.Context, f outbound.FiltroSucursales) ([]dtoresult.Sucursal, error) {
	return a.repo.ListSucursales(ctx, f)
}
func (a *ConfiguracionReadAdapter) ObtenerSucursal(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return a.repo.GetSucursal(ctx, id)
}
```

- [ ] **Step 6: Capa web**

Añadir a `web/dto/response/configuracion.go`:

```go
type Sucursal struct {
	ID                         string     `json:"id"`
	NegocioID                  string     `json:"negocio_id"`
	Codigo                     string     `json:"codigo"`
	Nombre                     string     `json:"nombre"`
	Direccion                  string     `json:"direccion"`
	Ubigeo                     *string    `json:"ubigeo"`
	CodigoEstablecimientoSunat *string    `json:"codigo_establecimiento_sunat"`
	Telefono                   *string    `json:"telefono"`
	EsActivo                   bool       `json:"es_activo"`
	MotivoEliminacion          *string    `json:"motivo_eliminacion"`
	EliminadoEn                *time.Time `json:"eliminado_en"`
	EliminadoPor               *string    `json:"eliminado_por"`
	CreadoEn                   time.Time  `json:"creado_en"`
	CreadoPor                  string     `json:"creado_por"`
	ActualizadoEn              time.Time  `json:"actualizado_en"`
	ActualizadoPor             string     `json:"actualizado_por"`
}

type SucursalesPagina struct {
	Items      []Sucursal `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}
```

Añadir a `web/mapper/configuracion_web_mapper.go`:

```go
func ToSucursal(s dtoresult.Sucursal) response.Sucursal {
	var eliminadoPor *string
	if s.EliminadoPor != nil {
		v := id(*s.EliminadoPor)
		eliminadoPor = &v
	}
	return response.Sucursal{
		ID: id(s.ID), NegocioID: id(s.NegocioID), Codigo: s.Codigo, Nombre: s.Nombre, Direccion: s.Direccion,
		Ubigeo: s.Ubigeo, CodigoEstablecimientoSunat: s.CodigoEstablecimientoSunat, Telefono: s.Telefono,
		EsActivo: s.EsActivo, MotivoEliminacion: s.MotivoEliminacion, EliminadoEn: s.EliminadoEn, EliminadoPor: eliminadoPor,
		CreadoEn: s.CreadoEn, CreadoPor: id(s.CreadoPor), ActualizadoEn: s.ActualizadoEn, ActualizadoPor: id(s.ActualizadoPor),
	}
}

func ToSucursalesPagina(p dtoresult.PaginaSucursales) response.SucursalesPagina {
	items := make([]response.Sucursal, len(p.Items))
	for i, s := range p.Items {
		items[i] = ToSucursal(s)
	}
	return response.SucursalesPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
```

`web/controller/configuracion_helpers.go`:

```go
package controller

import (
	"net/url"

	"service-restaurant/internal/shared/kernel/apperror"
)

// activoDe lee el filtro opcional es_activo ("true" o "false").
func activoDe(q url.Values) (*bool, error) {
	switch v := q.Get("es_activo"); v {
	case "":
		return nil, nil
	case "true":
		t := true
		return &t, nil
	case "false":
		f := false
		return &f, nil
	}
	return nil, apperror.New(apperror.KindBadRequest, "es_activo_invalido", "es_activo debe ser true o false")
}
```

`web/controller/sucursales_handler.go`:

```go
package controller

import (
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
)

// SucursalesHandler: sedes del negocio (RES-RF-003).
type SucursalesHandler struct {
	listar  inbound.ListarSucursales
	obtener inbound.ObtenerSucursal
}

func NewSucursalesHandler(l inbound.ListarSucursales, o inbound.ObtenerSucursal) *SucursalesHandler {
	return &SucursalesHandler{l, o}
}

// Listar: GET /sucursales?limit=&cursor=&es_activo=
func (h *SucursalesHandler) Listar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	q, err := consulta(r)
	if err != nil {
		return err
	}
	limite, err := limiteDe(q)
	if err != nil {
		return err
	}
	activo, err := activoDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarSucursales{Actor: actor, Limite: limite, Cursor: q.Get("cursor"), Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursalesPagina)
}

// Obtener: GET /sucursales/{id}
func (h *SucursalesHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerSucursal{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}
```

Reemplazar `RegisterConfiguracionRoutes` en `web/controller/configuracion_routes.go`:

```go
func RegisterConfiguracionRoutes(mux *http.ServeMux, negocio *NegocioHandler, sucursales *SucursalesHandler,
	protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /negocio", negocio.Obtener)
	ruta("PATCH /negocio", negocio.Editar)

	ruta("GET /sucursales", sucursales.Listar)
	ruta("GET /sucursales/{id}", sucursales.Obtener)
}
```

En `infrastructure/config/configuracion.go`: añadir el campo `sucursales *controller.SucursalesHandler` a la estructura `configuracion` y, en el literal devuelto:

```go
		sucursales: controller.NewSucursalesHandler(uquery.NewListarSucursales(lector), uquery.NewObtenerSucursal(lector)),
```

En `module.go`: `controller.RegisterConfiguracionRoutes(mux, m.conf.negocio, m.conf.sucursales, protect, m.deps.Logger)`.

- [ ] **Step 7: Compilar**

Run: `go build ./... && go vet ./internal/modules/identidad/... && go test ./tests/architecture/ ./internal/...`
Expected: sin errores.

- [ ] **Step 8: Pruebas de integración de lectura**

`tests/integration/configuracion_sucursales_test.go`:

```go
package integration

import (
	"net/http"
	"slices"
	"sort"
	"testing"
)

type sedesPagina struct {
	Items      []sedeResp `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

func (p sedesPagina) codigos() []string {
	out := make([]string, 0, len(p.Items))
	for _, s := range p.Items {
		out = append(out, s.Codigo)
	}
	sort.Strings(out)
	return out
}

func listarSedes(t *testing.T, query, tk string) sedesPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/sucursales"+query, "", tk)
	esperar(t, r, http.StatusOK, "")
	var p sedesPagina
	r.json(t, &p)
	return p
}

func crearSedeInactivaSQL(t *testing.T, codigo string) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.sucursales(negocio_id, codigo, nombre, direccion, es_activo, motivo_eliminacion)
		SELECT id, '`+codigo+`', 'Sede `+codigo+`', 'Calle cerrada 1', false, 'Local cerrado' FROM sch_restaurante.negocio
		WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.sucursales WHERE codigo='`+codigo+`')`)
}

func TestSucursalesSoloMuestraLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "CFG-A")
	crearSedeSQL(t, "CFG-B")
	rolConPermisos(t, "T_SUC_VER", "SUCURSAL_VER")
	asegurarUsuario(t, "suc.ver", "SucVer2026Clave", "T_SUC_VER", "PRINCIPAL", "CFG-A")
	tk := token(t, "suc.ver", "SucVer2026Clave")

	if got := listarSedes(t, "", tk).codigos(); !slices.Equal(got, []string{"CFG-A", "PRINCIPAL"}) {
		t.Fatalf("sedes visibles = %v, quería [CFG-A PRINCIPAL] (CFG-B no es suya)", got)
	}
	// El administrador instalado solo tiene PRINCIPAL asignada.
	if got := listarSedes(t, "", adminToken(t)).codigos(); !slices.Equal(got, []string{"PRINCIPAL"}) {
		t.Fatalf("el admin ve %v, quería [PRINCIPAL]", got)
	}

	propia := idDe(t, "sucursales", "codigo", "CFG-A")
	r := call(t, "GET", "/api/v1/sucursales/"+propia, "", tk)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.Codigo != "CFG-A" || !s.EsActivo || s.NegocioID == "" || s.Motivo != nil {
		t.Fatalf("sucursal: %+v", s)
	}
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+idDe(t, "sucursales", "codigo", "CFG-B"), "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/sucursales/999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/sucursales/abc", "", tk), http.StatusBadRequest, "id_invalido")
}

func TestSucursalesExigeAlgunPermisoDeSucursal(t *testing.T) {
	asegurarUsuario(t, "suc.sin", "SucSin2026Clave", "CAJERO", "PRINCIPAL")
	tk := token(t, "suc.sin", "SucSin2026Clave")
	esperar(t, call(t, "GET", "/api/v1/sucursales", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/sucursales/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/sucursales", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func TestSucursalesFiltroPaginacionEInactivas(t *testing.T) {
	crearSedeSQL(t, "CFG-A")
	crearSedeInactivaSQL(t, "CFG-INACTIVA")
	rolConPermisos(t, "T_SUC_VER", "SUCURSAL_VER")
	rolConPermisos(t, "T_SUC_GESTOR", "SUCURSAL_VER", "SUCURSAL_ADMINISTRAR")
	asegurarUsuario(t, "suc.ver", "SucVer2026Clave", "T_SUC_VER", "PRINCIPAL", "CFG-A")
	asegurarUsuario(t, "suc.gestor", "SucGestor2026Clave", "T_SUC_GESTOR", "PRINCIPAL")
	ver, gestor := token(t, "suc.ver", "SucVer2026Clave"), token(t, "suc.gestor", "SucGestor2026Clave")

	// Quien solo ve no descubre sedes inactivas; quien administra sedes sí (para restaurarlas).
	if got := listarSedes(t, "?es_activo=false", ver).codigos(); len(got) != 0 {
		t.Fatalf("VER no debe ver inactivas: %v", got)
	}
	if got := listarSedes(t, "?es_activo=false", gestor).codigos(); !slices.Contains(got, "CFG-INACTIVA") {
		t.Fatalf("el gestor debe ver las inactivas: %v", got)
	}
	r := call(t, "GET", "/api/v1/sucursales/"+idDe(t, "sucursales", "codigo", "CFG-INACTIVA"), "", gestor)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.EsActivo || s.Motivo == nil || *s.Motivo != "Local cerrado" || s.EliminadoPor == nil {
		t.Fatalf("sede inactiva: %+v", s)
	}
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+s.ID, "", ver), http.StatusNotFound, "no_encontrado")
	for _, x := range listarSedes(t, "?es_activo=true", gestor).Items {
		if !x.EsActivo {
			t.Fatalf("es_activo=true devolvió una inactiva: %+v", x)
		}
	}
	esperar(t, call(t, "GET", "/api/v1/sucursales?es_activo=quizas", "", gestor), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/sucursales?limit=0", "", gestor), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/sucursales?cursor=%25%25", "", gestor), http.StatusBadRequest, "")

	// Paginación por cursor sobre las dos sedes de suc.ver.
	p1 := listarSedes(t, "?limit=1", ver)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarSedes(t, "?limit=1&cursor="+*p1.NextCursor, ver)
	if len(p2.Items) != 1 || p2.Items[0].ID == p1.Items[0].ID || p2.NextCursor != nil {
		t.Fatalf("segunda página: %+v", p2)
	}
}
```

- [ ] **Step 9: Ejecutar**

Run: `go test -count=1 ./tests/integration/ -run "TestSucursales|TestNegocio" -v`
Expected: PASS.

- [ ] **Step 10: Anotar la regla en el spec**

En `docs/superpowers/specs/2026-10-18-i2-configuracion-salon-design.md`, al final de §3 (después del párrafo «Quien crea una sede…»), añadir:

```markdown
**Sedes inactivas.** `v_permisos_efectivos` solo concede permisos sobre sedes activas, de modo que al dar de baja una sede el actor pierde sus permisos sobre ella. Para poder restaurarla, una sede inactiva la ven y la gestionan quienes tienen `SUCURSAL_ADMINISTRAR` en alguna sede; las activas siguen la regla de la tabla. Consecuencia: quien tenía asignada solo esa sede queda sin acceso hasta que otro administrador la restaure o lo reasigne.
```

y en §10 (Riesgos) una viñeta: `- **Administradores sin sede activa:** si se da de baja la única sede asignada a todos los administradores de sedes, nadie podría restaurarla por la API; se resuelve con cmd/admin o por SQL. Mitigación posible fuera de esta rebanada: impedir la baja si deja a la empresa sin administradores de sedes.`

- [ ] **Step 11: Regresión y commit**

Run: `gofmt -l . ; go vet ./... && go test ./... && go test -count=1 ./tests/integration/...`
Expected: todo en verde.

```bash
git add internal/modules/identidad tests/integration docs/superpowers/specs
git commit -m "I2 paso 1: GET /sucursales y /sucursales/{id} con visibilidad por sede

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Sucursales — crear, editar, baja y restauración

**Files:**
- Create: `application/dto/command/sucursales.go`
- Modify: `application/port/inbound/sucursales.go`, `application/port/outbound/sucursales.go`
- Create: `application/usecase/command/sucursales.go`, `application/usecase/command/sucursales_test.go`
- Modify: `infrastructure/persistence/write/repository/configuracion_sql.go`, `.../write/adapter/configuracion.go`
- Modify: `web/dto/request/configuracion.go`, `web/controller/sucursales_handler.go`, `web/controller/configuracion_routes.go`
- Modify: `infrastructure/config/configuracion.go`
- Modify: `tests/integration/configuracion_sucursales_test.go`

**Interfaces:**
- Consumes: todo lo de las Tareas 2–4; `escritura[...]`, `command.Intencion`, `intencion(r, operacion, cuerpo)`, `leerCuerpoOpcionalVacio`, `request.Motivo`, `request.Confirmacion`, `request.Opcional`.
- Produces:
  - `command.CrearSucursal{Intencion; Actor; Codigo, Nombre, Direccion string; Ubigeo, CodigoEstablecimientoSunat, Telefono *string}`
  - `command.EditarSucursal{Actor; ID int64; Codigo, Nombre, Direccion *string; Ubigeo, CodigoEstablecimientoSunat, Telefono Campo[string]}`
  - `command.CambiarEstadoSucursal{Intencion; Actor; ID int64; Activo bool; Motivo string}`
  - `inbound.CrearSucursal`, `inbound.EditarSucursal`, `inbound.CambiarEstadoSucursal` (`Execute(ctx, X) (Result[result.Sucursal], error)`).
  - `outbound.NuevaSucursal`, `outbound.CambiosSucursal`, `outbound.SucursalStore`, `outbound.ErrCodigoSucursalDuplicado`.
  - `writeadapter.NewSucursalWriteAdapter(db)`, `controller.NewSucursalesHandler(listar, obtener, crear, editar, estado)`.

- [ ] **Step 1: DTO, puertos y pruebas unitarias (fallan)**

`application/dto/command/sucursales.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearSucursal da de alta una sede del negocio. negocio_id no se acepta: es la empresa única.
type CrearSucursal struct {
	Intencion
	Actor                      appctx.Actor
	Codigo                     string
	Nombre                     string
	Direccion                  string
	Ubigeo                     *string
	CodigoEstablecimientoSunat *string
	Telefono                   *string
}

// EditarSucursal aplica solo los campos presentes.
type EditarSucursal struct {
	Actor                      appctx.Actor
	ID                         int64
	Codigo                     *string
	Nombre                     *string
	Direccion                  *string
	Ubigeo                     Campo[string]
	CodigoEstablecimientoSunat Campo[string]
	Telefono                   Campo[string]
}

// CambiarEstadoSucursal da de baja (Activo=false, exige Motivo) o restaura (Activo=true).
type CambiarEstadoSucursal struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}
```

Añadir a `application/port/inbound/sucursales.go` (importar además `cmd "…/application/dto/command"`):

```go
type CrearSucursal interface {
	Execute(context.Context, cmd.CrearSucursal) (appresult.Result[dtoresult.Sucursal], error)
}
type EditarSucursal interface {
	Execute(context.Context, cmd.EditarSucursal) (appresult.Result[dtoresult.Sucursal], error)
}
type CambiarEstadoSucursal interface {
	Execute(context.Context, cmd.CambiarEstadoSucursal) (appresult.Result[dtoresult.Sucursal], error)
}
```

Añadir a `application/port/outbound/sucursales.go` (importar además `"fmt"`):

```go
// ErrCodigoSucursalDuplicado distingue la colisión del código de sede de otras unicidades.
var ErrCodigoSucursalDuplicado = fmt.Errorf("código de sucursal duplicado: %w", ErrDuplicado)

type NuevaSucursal struct {
	Codigo, Nombre, Direccion                   string
	Ubigeo, CodigoEstablecimientoSunat, Telefono *string
}

type CambiosSucursal struct {
	Codigo, Nombre, Direccion                   *string
	Ubigeo, CodigoEstablecimientoSunat, Telefono CampoOpcional
}

// SucursalStore escribe sedes dentro de la transacción del comando.
type SucursalStore interface {
	// Crear inserta la sede en la empresa única.
	Crear(ctx context.Context, n NuevaSucursal) (dtoresult.Sucursal, error)
	// Obtener lee la sede bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosSucursal) (dtoresult.Sucursal, bool, error)
	// BloquearNegocio serializa las bajas concurrentes para que la regla de la última sede sea fiable.
	BloquearNegocio(ctx context.Context) error
	ContarActivas(ctx context.Context) (int, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Sucursal, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error)
}
```

`application/usecase/command/sucursales_test.go`:

```go
package command

import (
	"context"
	"errors"
	"testing"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type sucursalStoreFake struct {
	existente  dtoresult.Sucursal
	activas    int
	errCrear   error
	creada     outbound.NuevaSucursal
	cambios    outbound.CambiosSucursal
	bajas      int
	restaurada int
	bloqueos   int
}

func (f *sucursalStoreFake) Crear(_ context.Context, n outbound.NuevaSucursal) (dtoresult.Sucursal, error) {
	f.creada = n
	return dtoresult.Sucursal{ID: 3, Codigo: n.Codigo, EsActivo: true}, f.errCrear
}
func (f *sucursalStoreFake) Obtener(_ context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return f.existente, f.existente.ID == id, nil
}
func (f *sucursalStoreFake) Actualizar(_ context.Context, id int64, c outbound.CambiosSucursal) (dtoresult.Sucursal, bool, error) {
	f.cambios = c
	return f.existente, true, nil
}
func (f *sucursalStoreFake) BloquearNegocio(context.Context) error { f.bloqueos++; return nil }
func (f *sucursalStoreFake) ContarActivas(context.Context) (int, error) { return f.activas, nil }
func (f *sucursalStoreFake) DarBaja(_ context.Context, id int64, _ string) (dtoresult.Sucursal, bool, error) {
	f.bajas++
	s := f.existente
	s.EsActivo = false
	return s, true, nil
}
func (f *sucursalStoreFake) Restaurar(_ context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	f.restaurada++
	s := f.existente
	s.EsActivo = true
	return s, true, nil
}

func gestor(sedes ...int64) appctx.Actor {
	var g []appctx.Grant
	for _, s := range sedes {
		g = append(g, appctx.Grant{SucursalID: s, Permiso: policy.SucursalAdministrar})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

func TestCrearSucursal(t *testing.T) {
	ok := cmd.CrearSucursal{Actor: gestor(1), Codigo: " san-isidro ", Nombre: " San Isidro ", Direccion: "Av. Javier Prado 100",
		Ubigeo: texto("150131"), Telefono: texto("   ")}

	f := &sucursalStoreFake{}
	res, err := NewCrearSucursal(f).Execute(context.Background(), ok)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía crearse: %v %+v", err, res.Err())
	}
	if f.creada.Codigo != "SAN-ISIDRO" || f.creada.Nombre != "San Isidro" || *f.creada.Ubigeo != "150131" || f.creada.Telefono != nil {
		t.Fatalf("no normalizó: %+v", f.creada)
	}

	casos := []struct {
		nombre string
		mod    func(c *cmd.CrearSucursal)
		kind   apperror.Kind
		code   string
	}{
		{"sin permiso de sucursal", func(c *cmd.CrearSucursal) { c.Actor = actorNegocio(policy.NegocioAdministrar) }, apperror.KindForbidden, "sin_permiso"},
		{"código inválido", func(c *cmd.CrearSucursal) { c.Codigo = "x" }, apperror.KindValidation, "codigo_invalido"},
		{"nombre en blanco", func(c *cmd.CrearSucursal) { c.Nombre = " " }, apperror.KindValidation, "nombre_invalido"},
		{"dirección en blanco", func(c *cmd.CrearSucursal) { c.Direccion = "" }, apperror.KindValidation, "direccion_invalida"},
		{"ubigeo", func(c *cmd.CrearSucursal) { c.Ubigeo = texto("1") }, apperror.KindValidation, "ubigeo_invalido"},
		{"código SUNAT largo", func(c *cmd.CrearSucursal) { c.CodigoEstablecimientoSunat = texto("12345678901") }, apperror.KindValidation, "codigo_establecimiento_sunat_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := &sucursalStoreFake{}
			in := ok
			c.mod(&in)
			res, err := NewCrearSucursal(f).Execute(context.Background(), in)
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}

	t.Run("código duplicado", func(t *testing.T) {
		f := &sucursalStoreFake{errCrear: outbound.ErrCodigoSucursalDuplicado}
		res, err := NewCrearSucursal(f).Execute(context.Background(), ok)
		if err != nil || res.Err() == nil {
			t.Fatalf("debía rechazarse: %v", err)
		}
		esperarRechazo(t, res.Err(), apperror.KindConflict, "codigo_duplicado")
	})
	t.Run("error técnico", func(t *testing.T) {
		f := &sucursalStoreFake{errCrear: errors.New("conexión perdida")}
		if _, err := NewCrearSucursal(f).Execute(context.Background(), ok); err == nil {
			t.Fatal("un error técnico debe viajar por el canal error")
		}
	})
}

func TestEditarSucursalAutorizaPorSede(t *testing.T) {
	activa := dtoresult.Sucursal{ID: 5, EsActivo: true}
	edit := func(a appctx.Actor) cmd.EditarSucursal { return cmd.EditarSucursal{Actor: a, ID: 5, Nombre: texto("Nueva")} }

	casos := []struct {
		nombre string
		actor  appctx.Actor
		sede   dtoresult.Sucursal
		kind   apperror.Kind
		code   string
	}{
		{"sede ajena activa no se revela", gestor(1), activa, apperror.KindNotFound, "no_encontrado"},
		{"solo VER ve pero no edita", appctx.NewActor(7, "a", 1, []appctx.Grant{
			{SucursalID: 5, Permiso: policy.SucursalVer}, {SucursalID: 9, Permiso: policy.SucursalAdministrar}}), activa, apperror.KindForbidden, "sede_no_autorizada"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := &sucursalStoreFake{existente: c.sede}
			res, err := NewEditarSucursal(f).Execute(context.Background(), edit(c.actor))
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}

	f := &sucursalStoreFake{existente: activa}
	res, err := NewEditarSucursal(f).Execute(context.Background(), cmd.EditarSucursal{
		Actor: gestor(5), ID: 5, Nombre: texto("  Nueva  "), Telefono: cmd.Campo[string]{Presente: true}})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	if *f.cambios.Nombre != "Nueva" || !f.cambios.Telefono.Presente || f.cambios.Telefono.Valor != nil || f.cambios.Ubigeo.Presente {
		t.Fatalf("cambios: %+v", f.cambios)
	}
	res, _ = NewEditarSucursal(f).Execute(context.Background(), cmd.EditarSucursal{Actor: gestor(5), ID: 5})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "sin_cambios")
}

func TestDarDeBajaSucursal(t *testing.T) {
	baja := func(a appctx.Actor, motivo string) cmd.CambiarEstadoSucursal {
		return cmd.CambiarEstadoSucursal{Actor: a, ID: 5, Activo: false, Motivo: motivo}
	}

	t.Run("exige motivo", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 2}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(5), "  "))
		esperarRechazo(t, res.Err(), apperror.KindValidation, "motivo_requerido")
	})
	t.Run("no deja a la empresa sin sede activa", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 1}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(5), "Cierre"))
		esperarRechazo(t, res.Err(), apperror.KindBusinessRule, "ultima_sede_activa")
		if f.bajas != 0 || f.bloqueos != 1 {
			t.Fatalf("bajas=%d bloqueos=%d: debe bloquear antes de contar y no dar de baja", f.bajas, f.bloqueos)
		}
	})
	t.Run("baja correcta", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 2}
		res, err := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(5), "Cierre del local"))
		if err != nil || !res.IsOk() || res.Value().EsActivo || f.bajas != 1 {
			t.Fatalf("baja: %v %+v", err, res.Err())
		}
	})
	t.Run("ya estaba de baja", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: false}, activas: 3}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(1), "Otra vez"))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_dada_de_baja")
	})
	t.Run("sede activa ajena", func(t *testing.T) {
		f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}, activas: 3}
		res, _ := NewCambiarEstadoSucursal(f).Execute(context.Background(), baja(gestor(1), "No es mía"))
		esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	})
}

func TestRestaurarSucursal(t *testing.T) {
	rest := func(a appctx.Actor) cmd.CambiarEstadoSucursal {
		return cmd.CambiarEstadoSucursal{Actor: a, ID: 5, Activo: true}
	}
	// Una sede inactiva no genera permisos: la restaura quien administra sedes en cualquier otra.
	f := &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: false}}
	res, err := NewCambiarEstadoSucursal(f).Execute(context.Background(), rest(gestor(1)))
	if err != nil || !res.IsOk() || !res.Value().EsActivo || f.restaurada != 1 {
		t.Fatalf("restaurar: %v %+v", err, res.Err())
	}
	f = &sucursalStoreFake{existente: dtoresult.Sucursal{ID: 5, EsActivo: true}}
	res, _ = NewCambiarEstadoSucursal(f).Execute(context.Background(), rest(gestor(5)))
	esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_activa")
	res, _ = NewCambiarEstadoSucursal(&sucursalStoreFake{}).Execute(context.Background(), rest(appctx.Actor{}))
	esperarRechazo(t, res.Err(), apperror.KindUnauthorized, "no_autenticado")
}
```

(`texto` y `actorNegocio` vienen de `negocio_test.go`, mismo paquete.)

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `go test ./internal/modules/identidad/application/usecase/command/ 2>&1 | head`
Expected: FAIL de compilación (`undefined: NewCrearSucursal`).

- [ ] **Step 3: Implementar los casos de uso**

`application/usecase/command/sucursales.go`:

```go
package command

import (
	"context"
	"errors"

	cmd "service-restaurant/internal/modules/identidad/application/dto/command"
	dtoresult "service-restaurant/internal/modules/identidad/application/dto/result"
	"service-restaurant/internal/modules/identidad/application/mapper"
	"service-restaurant/internal/modules/identidad/application/port/outbound"
	"service-restaurant/internal/modules/identidad/domain/policy"
	"service-restaurant/internal/modules/identidad/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxNombreSucursal    = 120
	maxDireccionSucursal = 500
	maxCodigoSunat       = 10

	CodigoCodigoDuplicado   = "codigo_duplicado"
	CodigoUltimaSedeActiva  = "ultima_sede_activa"
	CodigoYaDadaDeBaja      = "ya_dada_de_baja"
	CodigoYaActiva          = "ya_activa"
	mensajeSinPermisoCrear  = "no tiene permiso para administrar sucursales"
	mensajeSedeNoAutorizada = "no administra esta sucursal"
)

// CrearSucursal da de alta una sede de la empresa única (RES-RF-003). Quien la crea no recibe
// acceso a ella: las asignaciones se hacen en Equipo y accesos.
type CrearSucursal struct{ sucursales outbound.SucursalStore }

func NewCrearSucursal(s outbound.SucursalStore) *CrearSucursal { return &CrearSucursal{s} }

func (uc *CrearSucursal) Execute(ctx context.Context, c cmd.CrearSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCrear, policy.SucursalAdministrar); ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	codigo, err := vo.NewCodigoSucursal(c.Codigo)
	if err != nil {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_invalido", "el código debe tener 2 a 20 caracteres: mayúsculas, dígitos, _ y -"))
	}
	nombre, err := vo.TextoObligatorio(c.Nombre, maxNombreSucursal)
	if err != nil {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 120 caracteres)"))
	}
	direccion, err := vo.TextoObligatorio(c.Direccion, maxDireccionSucursal)
	if err != nil {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("direccion_invalida", "la dirección es obligatoria (máximo 500 caracteres)"))
	}
	ubigeo, ok := valorOpcional(c.Ubigeo, normUbigeo)
	if !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("ubigeo_invalido", "el ubigeo debe tener 6 dígitos"))
	}
	sunat, ok := valorOpcional(c.CodigoEstablecimientoSunat, normMax(maxCodigoSunat))
	if !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_establecimiento_sunat_invalido", "el código de establecimiento SUNAT excede 10 caracteres"))
	}
	telefono, ok := valorOpcional(c.Telefono, normMax(maxTelefono))
	if !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("telefono_invalido", "el teléfono excede 30 caracteres"))
	}
	s, err := uc.sucursales.Crear(ctx, outbound.NuevaSucursal{
		Codigo: codigo.String(), Nombre: nombre, Direccion: direccion,
		Ubigeo: ubigeo, CodigoEstablecimientoSunat: sunat, Telefono: telefono,
	})
	if err != nil {
		return deSucursal(err, "crear sucursal")
	}
	return appresult.Ok(s), nil
}

// deSucursal traduce el código duplicado a un rechazo específico y delega lo demás.
func deSucursal(err error, contexto string) (appresult.Result[dtoresult.Sucursal], error) {
	if errors.Is(err, outbound.ErrCodigoSucursalDuplicado) {
		return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindConflict, CodigoCodigoDuplicado, "ya existe una sucursal con ese código"))
	}
	return deAlmacen[dtoresult.Sucursal](contexto, err)
}

```

Continúa en el mismo archivo con el helper de autorización por sede (lo usan editar y cambiar de estado). Añadir el import `appctx "service-restaurant/internal/shared/application/context"` al archivo:

```go
// sedeGestionable carga la sede (bloqueándola) y verifica que el actor la ve (si no, 404) y que
// puede administrarla (si no, 403). Devuelve la sede o el rechazo; un error técnico va en err.
func sedeGestionable(ctx context.Context, store outbound.SucursalStore, actor appctx.Actor, id int64) (dtoresult.Sucursal, *apperror.AppError, error) {
	s, found, err := store.Obtener(ctx, id)
	if err != nil {
		return dtoresult.Sucursal{}, nil, err
	}
	if !found || !mapper.SucursalVisible(actor, s.ID, s.EsActivo) {
		return dtoresult.Sucursal{}, mapper.NoEncontrado("sucursal"), nil
	}
	if !mapper.PuedeAdministrarSucursal(actor, s.ID, s.EsActivo) {
		return dtoresult.Sucursal{}, apperror.New(apperror.KindForbidden, mapper.CodigoSedeNoAutorizada, mensajeSedeNoAutorizada), nil
	}
	return s, nil, nil
}
```

(añadir el import `appctx "service-restaurant/internal/shared/application/context"`). Continuar el archivo con:

```go
// EditarSucursal modifica solo los campos enviados.
type EditarSucursal struct{ sucursales outbound.SucursalStore }

func NewEditarSucursal(s outbound.SucursalStore) *EditarSucursal { return &EditarSucursal{s} }

func (uc *EditarSucursal) Execute(ctx context.Context, c cmd.EditarSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCrear, policy.SucursalAdministrar); ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	if _, ae, err := sedeGestionable(ctx, uc.sucursales, c.Actor, c.ID); err != nil {
		return tecnico[dtoresult.Sucursal]("buscar sucursal", err)
	} else if ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	if c.Codigo == nil && c.Nombre == nil && c.Direccion == nil &&
		!c.Ubigeo.Presente && !c.CodigoEstablecimientoSunat.Presente && !c.Telefono.Presente {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosSucursal
	if c.Codigo != nil {
		codigo, err := vo.NewCodigoSucursal(*c.Codigo)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_invalido", "el código debe tener 2 a 20 caracteres: mayúsculas, dígitos, _ y -"))
		}
		v := codigo.String()
		cambios.Codigo = &v
	}
	if c.Nombre != nil {
		v, err := vo.TextoObligatorio(*c.Nombre, maxNombreSucursal)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 120 caracteres)"))
		}
		cambios.Nombre = &v
	}
	if c.Direccion != nil {
		v, err := vo.TextoObligatorio(*c.Direccion, maxDireccionSucursal)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("direccion_invalida", "la dirección es obligatoria (máximo 500 caracteres)"))
		}
		cambios.Direccion = &v
	}
	var ok bool
	if cambios.Ubigeo, ok = opcionalValidado(c.Ubigeo, normUbigeo); !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("ubigeo_invalido", "el ubigeo debe tener 6 dígitos"))
	}
	if cambios.CodigoEstablecimientoSunat, ok = opcionalValidado(c.CodigoEstablecimientoSunat, normMax(maxCodigoSunat)); !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("codigo_establecimiento_sunat_invalido", "el código de establecimiento SUNAT excede 10 caracteres"))
	}
	if cambios.Telefono, ok = opcionalValidado(c.Telefono, normMax(maxTelefono)); !ok {
		return rechazar[dtoresult.Sucursal](mapper.Validacion("telefono_invalido", "el teléfono excede 30 caracteres"))
	}
	s, found, err := uc.sucursales.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deSucursal(err, "editar sucursal")
	}
	if !found {
		return rechazar[dtoresult.Sucursal](mapper.NoEncontrado("sucursal"))
	}
	return appresult.Ok(s), nil
}

// CambiarEstadoSucursal da de baja (con motivo) o restaura una sede. Nunca deja a la empresa
// sin sede activa; la restauración conserva el ID.
type CambiarEstadoSucursal struct{ sucursales outbound.SucursalStore }

func NewCambiarEstadoSucursal(s outbound.SucursalStore) *CambiarEstadoSucursal {
	return &CambiarEstadoSucursal{s}
}

func (uc *CambiarEstadoSucursal) Execute(ctx context.Context, c cmd.CambiarEstadoSucursal) (appresult.Result[dtoresult.Sucursal], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoCrear, policy.SucursalAdministrar); ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	motivo := ""
	if !c.Activo {
		m, err := vo.TextoObligatorio(c.Motivo, maxMotivo)
		if err != nil {
			return rechazar[dtoresult.Sucursal](mapper.Validacion("motivo_requerido", "la baja exige un motivo (máximo 2000 caracteres)"))
		}
		motivo = m
	}
	actual, ae, err := sedeGestionable(ctx, uc.sucursales, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Sucursal]("buscar sucursal", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Sucursal](ae)
	}
	if c.Activo {
		if actual.EsActivo {
			return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindConflict, CodigoYaActiva, "la sucursal ya está activa"))
		}
		s, found, err := uc.sucursales.Restaurar(ctx, c.ID)
		if err != nil {
			return deSucursal(err, "restaurar sucursal")
		}
		if !found {
			return rechazar[dtoresult.Sucursal](mapper.NoEncontrado("sucursal"))
		}
		return appresult.Ok(s), nil
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la sucursal ya está dada de baja"))
	}
	if err := uc.sucursales.BloquearNegocio(ctx); err != nil {
		return tecnico[dtoresult.Sucursal]("bloquear negocio", err)
	}
	activas, err := uc.sucursales.ContarActivas(ctx)
	if err != nil {
		return tecnico[dtoresult.Sucursal]("contar sucursales activas", err)
	}
	if activas <= 1 {
		return rechazar[dtoresult.Sucursal](apperror.New(apperror.KindBusinessRule, CodigoUltimaSedeActiva,
			"no se puede dar de baja la última sucursal activa del negocio"))
	}
	s, found, err := uc.sucursales.DarBaja(ctx, c.ID, motivo)
	if err != nil {
		return deSucursal(err, "dar de baja la sucursal")
	}
	if !found {
		return rechazar[dtoresult.Sucursal](mapper.NoEncontrado("sucursal"))
	}
	return appresult.Ok(s), nil
}
```

Las constantes `CodigoUltimaSedeActiva`, etc., deben ser exportadas para que las pruebas del paquete `command` las vean; ya lo están. `maxMotivo` y `maxTelefono` provienen de `usuarios.go` y `negocio.go` (mismo paquete).

- [ ] **Step 4: Ejecutar las pruebas unitarias**

Run: `gofmt -l internal/ ; go vet ./internal/modules/identidad/... && go test ./internal/modules/identidad/... 2>&1 | tail`
Expected: PASS. Si la prueba «solo VER ve pero no edita» falla por orden de validaciones, recordar que `sedeGestionable` corre **antes** de validar los campos.

- [ ] **Step 5: Persistencia de escritura**

Añadir a `infrastructure/persistence/write/repository/configuracion_sql.go` (importar además `"strings"` y `"errors"`):

```go
const sucursalCols = `id, negocio_id, codigo, nombre, direccion, ubigeo, codigo_establecimiento_sunat, telefono,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func scanSucursal(s scanner) (dtoresult.Sucursal, error) {
	var x dtoresult.Sucursal
	var ubigeo, sunat, tel, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&x.ID, &x.NegocioID, &x.Codigo, &x.Nombre, &x.Direccion, &ubigeo, &sunat, &tel,
		&x.EsActivo, &motivo, &elimEn, &elimPor, &x.CreadoEn, &x.CreadoPor, &x.ActualizadoEn, &x.ActualizadoPor)
	x.Ubigeo, x.CodigoEstablecimientoSunat, x.Telefono, x.MotivoEliminacion =
		textoNulo(ubigeo), textoNulo(sunat), textoNulo(tel), textoNulo(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		x.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		x.EliminadoPor = &v
	}
	return x, err
}

// traducirSucursal distingue la colisión del código de sede (sucursales_codigo_key) de otras unicidades.
func traducirSucursal(err error) error {
	err = Translate(err)
	if errors.Is(err, outbound.ErrDuplicado) && strings.Contains(err.Error(), "sucursales_codigo_key") {
		return outbound.ErrCodigoSucursalDuplicado
	}
	return err
}

// SucursalSQLWriter opera sobre sch_restaurante.sucursales.
type SucursalSQLWriter struct{ db database.DBTX }

func NewSucursalSQLWriter(db database.DBTX) *SucursalSQLWriter { return &SucursalSQLWriter{db} }

// Insert crea la sede en la empresa única; negocio_id sale de la base, nunca del cliente.
func (r *SucursalSQLWriter) Insert(ctx context.Context, n outbound.NuevaSucursal) (dtoresult.Sucursal, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.sucursales (negocio_id, codigo, nombre, direccion, ubigeo, codigo_establecimiento_sunat, telefono)
		SELECT id, $1, $2, $3, $4, $5, $6 FROM sch_restaurante.negocio WHERE singleton
		RETURNING `+sucursalCols, n.Codigo, n.Nombre, n.Direccion, n.Ubigeo, n.CodigoEstablecimientoSunat, n.Telefono))
	return s, traducirSucursal(err)
}

// GetForUpdate lee la sede bloqueando su fila hasta el final de la transacción.
func (r *SucursalSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx,
		`SELECT `+sucursalCols+` FROM sch_restaurante.sucursales WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, err
	}
	return s, true, nil
}

func (r *SucursalSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosSucursal) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.sucursales SET
			codigo = COALESCE($2, codigo),
			nombre = COALESCE($3, nombre),
			direccion = COALESCE($4, direccion),
			ubigeo = CASE WHEN $5 THEN $6 ELSE ubigeo END,
			codigo_establecimiento_sunat = CASE WHEN $7 THEN $8 ELSE codigo_establecimiento_sunat END,
			telefono = CASE WHEN $9 THEN $10 ELSE telefono END
		WHERE id = $1 RETURNING `+sucursalCols, id, c.Codigo, c.Nombre, c.Direccion,
		c.Ubigeo.Presente, c.Ubigeo.Valor, c.CodigoEstablecimientoSunat.Presente, c.CodigoEstablecimientoSunat.Valor,
		c.Telefono.Presente, c.Telefono.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, traducirSucursal(err)
	}
	return s, true, nil
}

// LockNegocio bloquea la fila de la empresa: serializa las bajas (y las altas) de sedes.
func (r *SucursalSQLWriter) LockNegocio(ctx context.Context) error {
	var id int64
	return r.db.QueryRowContext(ctx, `SELECT id FROM sch_restaurante.negocio WHERE singleton FOR UPDATE`).Scan(&id)
}

func (r *SucursalSQLWriter) CountActivas(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM sch_restaurante.sucursales WHERE es_activo`).Scan(&n)
	return n, err
}

// SetBaja desactiva la sede; el trigger de metadatos completa actor y fecha de la baja.
func (r *SucursalSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.sucursales SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+sucursalCols, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, traducirSucursal(err)
	}
	return s, true, nil
}

// SetActiva reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja.
func (r *SucursalSQLWriter) SetActiva(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	s, err := scanSucursal(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.sucursales SET es_activo = true WHERE id = $1 RETURNING `+sucursalCols, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Sucursal{}, false, traducirSucursal(err)
	}
	return s, true, nil
}
```

Añadir a `infrastructure/persistence/write/adapter/configuracion.go`:

```go
// SucursalWriteAdapter implementa outbound.SucursalStore.
type SucursalWriteAdapter struct{ repo *repository.SucursalSQLWriter }

func NewSucursalWriteAdapter(db database.DBTX) *SucursalWriteAdapter {
	return &SucursalWriteAdapter{repo: repository.NewSucursalSQLWriter(db)}
}

var _ outbound.SucursalStore = (*SucursalWriteAdapter)(nil)

func (a *SucursalWriteAdapter) Crear(ctx context.Context, n outbound.NuevaSucursal) (dtoresult.Sucursal, error) {
	return a.repo.Insert(ctx, n)
}
func (a *SucursalWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *SucursalWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosSucursal) (dtoresult.Sucursal, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *SucursalWriteAdapter) BloquearNegocio(ctx context.Context) error { return a.repo.LockNegocio(ctx) }
func (a *SucursalWriteAdapter) ContarActivas(ctx context.Context) (int, error) {
	return a.repo.CountActivas(ctx)
}
func (a *SucursalWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Sucursal, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *SucursalWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Sucursal, bool, error) {
	return a.repo.SetActiva(ctx, id)
}
```

- [ ] **Step 6: Capa web**

Añadir a `web/dto/request/configuracion.go`:

```go
// SucursalAlta: negocio_id no existe aquí; un campo desconocido responde 400.
type SucursalAlta struct {
	Codigo                     string  `json:"codigo"`
	Nombre                     string  `json:"nombre"`
	Direccion                  string  `json:"direccion"`
	Ubigeo                     *string `json:"ubigeo"`
	CodigoEstablecimientoSunat *string `json:"codigo_establecimiento_sunat"`
	Telefono                   *string `json:"telefono"`
}

type SucursalEdicion struct {
	Codigo                     *string          `json:"codigo,omitempty"`
	Nombre                     *string          `json:"nombre,omitempty"`
	Direccion                  *string          `json:"direccion,omitempty"`
	Ubigeo                     Opcional[string] `json:"ubigeo"`
	CodigoEstablecimientoSunat Opcional[string] `json:"codigo_establecimiento_sunat"`
	Telefono                   Opcional[string] `json:"telefono"`
}
```

Reemplazar `web/controller/sucursales_handler.go` completo:

```go
package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/identidad/application/dto/command"
	"service-restaurant/internal/modules/identidad/application/dto/query"
	"service-restaurant/internal/modules/identidad/application/port/inbound"
	"service-restaurant/internal/modules/identidad/web/dto/request"
	webmapper "service-restaurant/internal/modules/identidad/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// SucursalesHandler: sedes del negocio (RES-RF-003).
type SucursalesHandler struct {
	listar  inbound.ListarSucursales
	obtener inbound.ObtenerSucursal
	crear   inbound.CrearSucursal
	editar  inbound.EditarSucursal
	estado  inbound.CambiarEstadoSucursal
}

func NewSucursalesHandler(l inbound.ListarSucursales, o inbound.ObtenerSucursal, c inbound.CrearSucursal,
	e inbound.EditarSucursal, s inbound.CambiarEstadoSucursal) *SucursalesHandler {
	return &SucursalesHandler{l, o, c, e, s}
}

// Listar: GET /sucursales?limit=&cursor=&es_activo=
func (h *SucursalesHandler) Listar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	q, err := consulta(r)
	if err != nil {
		return err
	}
	limite, err := limiteDe(q)
	if err != nil {
		return err
	}
	activo, err := activoDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarSucursales{Actor: actor, Limite: limite, Cursor: q.Get("cursor"), Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursalesPagina)
}

// Obtener: GET /sucursales/{id}
func (h *SucursalesHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerSucursal{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}

// Crear: POST /sucursales (Idempotency-Key obligatoria)
func (h *SucursalesHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.SucursalAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, "POST /sucursales", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearSucursal{
		Intencion: in, Actor: actor, Codigo: body.Codigo, Nombre: body.Nombre, Direccion: body.Direccion,
		Ubigeo: body.Ubigeo, CodigoEstablecimientoSunat: body.CodigoEstablecimientoSunat, Telefono: body.Telefono,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToSucursal)
}

// Editar: PATCH /sucursales/{id}
func (h *SucursalesHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.SucursalEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarSucursal{
		Actor: actor, ID: id, Codigo: body.Codigo, Nombre: body.Nombre, Direccion: body.Direccion,
		Ubigeo:                     command.Campo[string]{Presente: body.Ubigeo.Presente, Valor: body.Ubigeo.Valor},
		CodigoEstablecimientoSunat: command.Campo[string]{Presente: body.CodigoEstablecimientoSunat.Presente, Valor: body.CodigoEstablecimientoSunat.Valor},
		Telefono:                   command.Campo[string]{Presente: body.Telefono.Presente, Valor: body.Telefono.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}

// Baja: POST /sucursales/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *SucursalesHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.Motivo
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /sucursales/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoSucursal{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}

// Restauracion: POST /sucursales/{id}/restauracion (Idempotency-Key obligatoria)
func (h *SucursalesHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var c request.Confirmacion
	if err := leerCuerpoOpcionalVacio(r, &c); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /sucursales/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoSucursal{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToSucursal)
}
```

Añadir a `RegisterConfiguracionRoutes`:

```go
	ruta("POST /sucursales", sucursales.Crear)
	ruta("PATCH /sucursales/{id}", sucursales.Editar)
	ruta("POST /sucursales/{id}/baja", sucursales.Baja)
	ruta("POST /sucursales/{id}/restauracion", sucursales.Restauracion)
```

- [ ] **Step 7: Composición**

En `infrastructure/config/configuracion.go`, dentro de `armarConfiguracion` (antes de `lector :=`), añadir:

```go
	crearSucursal := escritura[cmd.CrearSucursal, dtoresult.Sucursal]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearSucursal, dtoresult.Sucursal] {
			return command.NewCrearSucursal(writeadapter.NewSucursalWriteAdapter(q))
		},
		meta: func(c cmd.CrearSucursal) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editarSucursal := escritura[cmd.EditarSucursal, dtoresult.Sucursal]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarSucursal, dtoresult.Sucursal] {
			return command.NewEditarSucursal(writeadapter.NewSucursalWriteAdapter(q))
		},
		meta: func(c cmd.EditarSucursal) (int64, cmd.Intencion) { return sinIntencion(c.Actor.UserID) }}
	estadoSucursal := escritura[cmd.CambiarEstadoSucursal, dtoresult.Sucursal]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoSucursal, dtoresult.Sucursal] {
			return command.NewCambiarEstadoSucursal(writeadapter.NewSucursalWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoSucursal) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
```

y reemplazar la línea del handler de sucursales por:

```go
		sucursales: controller.NewSucursalesHandler(uquery.NewListarSucursales(lector), uquery.NewObtenerSucursal(lector),
			inbound.CrearSucursal(crearSucursal), inbound.EditarSucursal(editarSucursal), inbound.CambiarEstadoSucursal(estadoSucursal)),
```

- [ ] **Step 8: Compilar**

Run: `go build ./... && go vet ./... && go test ./tests/architecture/ ./internal/...`
Expected: sin errores.

- [ ] **Step 9: Pruebas de integración de escritura**

Añadir a `tests/integration/configuracion_sucursales_test.go` (importar además `"fmt"`):

```go
const gestorUser, gestorPass = "suc.gestor", "SucGestor2026Clave"

func gestorToken(t *testing.T) string {
	t.Helper()
	rolConPermisos(t, "T_SUC_GESTOR", "SUCURSAL_VER", "SUCURSAL_ADMINISTRAR")
	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "PRINCIPAL")
	return token(t, gestorUser, gestorPass)
}

func altaSede(codigo string) string {
	return fmt.Sprintf(`{"codigo":"%s","nombre":"Sede %s","direccion":"Av. Nueva 123","ubigeo":"150101","telefono":"01-555-0000"}`, codigo, codigo)
}

func TestSucursalCrearConIdempotenciaYValidaciones(t *testing.T) {
	tk := gestorToken(t)
	clave := nuevaClave()
	h := map[string]string{"Idempotency-Key": clave}

	r1 := callH(t, "POST", "/api/v1/sucursales", altaSede("nueva-1"), tk, h)
	esperar(t, r1, http.StatusCreated, "")
	var s sedeResp
	r1.json(t, &s)
	if s.Codigo != "NUEVA-1" || !s.EsActivo || s.NegocioID == "" || s.CreadoPor != idDe(t, "usuarios", "username", gestorUser) ||
		s.Ubigeo == nil || *s.Ubigeo != "150101" {
		t.Fatalf("sede creada: %+v", s)
	}
	// Repetir la misma intención no duplica; otro contenido con la misma clave es conflicto.
	r2 := callH(t, "POST", "/api/v1/sucursales", altaSede("nueva-1"), tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var rep sedeResp
	r2.json(t, &rep)
	if rep.ID != s.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE codigo='NUEVA-1'`) != 1 {
		t.Fatalf("la repetición debe devolver la misma sede: %+v vs %+v", rep, s)
	}
	esperar(t, callH(t, "POST", "/api/v1/sucursales", altaSede("otra-9"), tk, h), http.StatusConflict, "idempotency_key_reutilizada")

	// Otra clave con el mismo código: conflicto de unicidad específico del campo.
	esperar(t, post(t, "/api/v1/sucursales", altaSede("NUEVA-1"), tk), http.StatusConflict, "codigo_duplicado")
	// Sin clave, con campos que el cliente no puede fijar, y validaciones por campo.
	esperar(t, call(t, "POST", "/api/v1/sucursales", altaSede("sin-clave"), tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"CON-NEG","nombre":"x","direccion":"y","negocio_id":"1"}`, tk), http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"creado_por","nombre":"x","direccion":"y","creado_por":"1"}`, tk), http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"x","nombre":"x","direccion":"y"}`, tk), http.StatusUnprocessableEntity, "codigo_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"BIEN-1","nombre":" ","direccion":"y"}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"BIEN-1","nombre":"x","direccion":""}`, tk), http.StatusUnprocessableEntity, "direccion_invalida")
	esperar(t, post(t, "/api/v1/sucursales", `{"codigo":"BIEN-1","nombre":"x","direccion":"y","ubigeo":"12"}`, tk), http.StatusUnprocessableEntity, "ubigeo_invalido")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE codigo IN ('BIEN-1','SIN-CLAVE','CON-NEG')`) != 0 {
		t.Fatal("los rechazos no deben dejar sedes")
	}

	// Sin SUCURSAL_ADMINISTRAR no se crea.
	esperar(t, post(t, "/api/v1/sucursales", altaSede("SIN-PODER"), token(t, "suc.ver", "SucVer2026Clave")), http.StatusForbidden, "sin_permiso")
}

func TestSucursalCreadaNoDaAccesoAlCreadorHastaAsignarla(t *testing.T) {
	tk := gestorToken(t)
	r := post(t, "/api/v1/sucursales", altaSede("SIN-ACCESO"), tk)
	esperar(t, r, http.StatusCreated, "")
	var s sedeResp
	r.json(t, &s)
	// Los permisos son por sede asignada: el creador aún no la ve ni la administra.
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+s.ID, "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+s.ID, `{"nombre":"X"}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/sucursales/"+s.ID+"/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")

	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "SIN-ACCESO")
	esperar(t, call(t, "GET", "/api/v1/sucursales/"+s.ID, "", tk), http.StatusOK, "")
}

func TestSucursalEditar(t *testing.T) {
	tk := gestorToken(t)
	crearSedeSQL(t, "EDITABLE")
	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "EDITABLE")
	id := idDe(t, "sucursales", "codigo", "EDITABLE")

	r := callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"nombre":"Sede Editada","telefono":"999","codigo_establecimiento_sunat":"0001"}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.Nombre != "Sede Editada" || s.Telefono == nil || *s.Telefono != "999" || s.CodigoSunat == nil || *s.CodigoSunat != "0001" || s.Codigo != "EDITABLE" {
		t.Fatalf("edición parcial: %+v", s)
	}
	r = callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"telefono":null}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &s)
	if s.Telefono != nil || s.CodigoSunat == nil || s.Nombre != "Sede Editada" {
		t.Fatalf("null debe vaciar solo el teléfono: %+v", s)
	}
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"codigo":"PRINCIPAL"}`, tk, nil), http.StatusConflict, "codigo_duplicado")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"negocio_id":"2"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/999999", `{"nombre":"x"}`, tk, nil), http.StatusNotFound, "no_encontrado")

	// Quien solo tiene SUCURSAL_VER no edita, aunque vea la sede.
	rolConPermisos(t, "T_SUC_VER", "SUCURSAL_VER")
	asegurarUsuario(t, "suc.ver", "SucVer2026Clave", "T_SUC_VER", "EDITABLE")
	esperar(t, callH(t, "PATCH", "/api/v1/sucursales/"+id, `{"nombre":"Hack"}`, token(t, "suc.ver", "SucVer2026Clave"), nil),
		http.StatusForbidden, "sin_permiso")
}

func TestSucursalBajaYRestauracion(t *testing.T) {
	tk := gestorToken(t)
	crearSedeSQL(t, "CICLO")
	asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", "CICLO")
	id := idDe(t, "sucursales", "codigo", "CICLO")

	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")

	clave := nuevaClave()
	h := map[string]string{"Idempotency-Key": clave}
	r := callH(t, "POST", "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Cierre del local"}`, tk, h)
	esperar(t, r, http.StatusOK, "")
	var s sedeResp
	r.json(t, &s)
	if s.EsActivo || s.Motivo == nil || *s.Motivo != "Cierre del local" || s.EliminadoPor == nil ||
		*s.EliminadoPor != idDe(t, "usuarios", "username", gestorUser) {
		t.Fatalf("baja: %+v", s)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='sucursales' AND accion='UPDATE'
		AND registro_id=`+id+` AND creado_por=`+idDe(t, "usuarios", "username", gestorUser)) == 0 {
		t.Fatal("la baja debe quedar auditada con el actor")
	}
	// La misma intención se repite sin nuevo efecto; con otra clave la sede ya está de baja.
	antes := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='sucursales' AND registro_id=`+id)
	esperar(t, callH(t, "POST", "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Cierre del local"}`, tk, h), http.StatusOK, "")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='sucursales' AND registro_id=`+id) != antes {
		t.Fatal("repetir la misma clave no debe volver a escribir")
	}
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Otra vez"}`, tk), http.StatusConflict, "ya_dada_de_baja")

	// Sin permisos sobre la sede inactiva, la restaura quien administra sedes en otra (PRINCIPAL).
	r = post(t, "/api/v1/sucursales/"+id+"/restauracion", `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rest sedeResp
	r.json(t, &rest)
	if rest.ID != s.ID || !rest.EsActivo || rest.Motivo != nil || rest.EliminadoPor != nil {
		t.Fatalf("restauración: %+v (debe conservar el ID y limpiar los marcadores)", rest)
	}
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/restauracion", "", tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/sucursales/999999/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")
}

func TestSucursalNoSePuedeDarDeBajaLaUltimaSedeActiva(t *testing.T) {
	tk := gestorToken(t)
	crearSedeSQL(t, "OTRA-SEDE")
	// Deja PRINCIPAL como única sede activa y lo revierte al terminar.
	const marca = "prueba-ultima-sede"
	asSystem(t, `UPDATE sch_restaurante.sucursales SET es_activo=false, motivo_eliminacion='`+marca+`'
		WHERE codigo<>'PRINCIPAL' AND es_activo`)
	t.Cleanup(func() {
		asSystem(t, `UPDATE sch_restaurante.sucursales SET es_activo=true WHERE motivo_eliminacion='`+marca+`'`)
	})

	id := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	esperar(t, post(t, "/api/v1/sucursales/"+id+"/baja", `{"motivo":"Intento"}`, tk), http.StatusUnprocessableEntity, "ultima_sede_activa")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.sucursales WHERE codigo='PRINCIPAL' AND es_activo`) != 1 {
		t.Fatal("PRINCIPAL debe seguir activa")
	}
}
```

- [ ] **Step 10: Ejecutar integración de sucursales**

Run: `go test -count=1 ./tests/integration/ -run "TestSucursal|TestSucursales|TestNegocio" -v`
Expected: PASS. Si `TestSucursalNoSePuedeDarDeBajaLaUltimaSedeActiva` interfiere con pruebas posteriores, comprobar que el `t.Cleanup` reactivó todas las sedes marcadas.

- [ ] **Step 11: Verificación completa del paso 1**

Run: `gofmt -l . ; go vet ./... && go test -race ./internal/... ./tests/architecture/ && go test -count=1 ./tests/integration/...`
Expected: sin salida de `gofmt`; todo PASS (incluidas las pruebas de I1 y las de este paso).

Comprobar manualmente en el contrato que no hay deriva: `grep -n "negocio_id" ../docs/planificacion/03_api/openapi.yaml | head -3` (el contrato todavía lista `negocio_id` en el alta de sucursales; se corrige en el paso 3, no aquí).

- [ ] **Step 12: Commit**

```bash
git add internal/modules/identidad tests/integration
git commit -m "I2 paso 1: crear, editar, dar de baja y restaurar sucursales

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

## Autorrevisión del plan

**Cobertura del spec (paso 1):**
- §2 migración: permisos, asignaciones, baja de mesa con motivo, trigger, relleno de filas previas → Tarea 1. La actualización de documentos del expediente es del paso 3.
- §3 autorización de negocio y sucursales (+ regla de sedes inactivas, anotada en el spec) → Tareas 2–5.
- §4 rutas de negocio y sucursales, idempotencia, filtro `es_activo`, `limit/cursor`, reglas (RUC inmutable, moneda/zona/ubigeo, código único, última sede, restauración conserva ID, `negocio_id` no aceptado, actor no aceptado) → Tareas 3–5. El filtro `sucursal_id` de los listados no aplica a `/sucursales` (la propia colección son las sedes) y se omite; sí aplicará a zonas y mesas en el paso 2.
- §5 estructura (casos de uso en `identidad`) → Tareas 3–5. `tests/architecture` ya cubre `identidad`; su ampliación a `atencion` es del paso 2.
- §8 pruebas del paso: unitarias de dominio/políticas, integración (404 sede ajena, 409 código duplicado, 422 última sede, restauración conserva ID, idempotencia, `*_VER` no escribe, permisos sembrados, motivo y actor en fila y auditoría, 00008 sube y baja) → cubiertas. Las pruebas de zonas/mesas son del paso 2.

**Placeholders:** no quedan TBD ni pasos sin código.

**Consistencia de tipos:** `outbound.CampoOpcional` / `command.Campo[T]` / `request.Opcional[T]` comparten la forma `{Presente, Valor *T}`; `mapper.SucursalVisible(a, id, activa)` y `PuedeAdministrarSucursal(a, id, activa)` se usan con la misma firma en consultas y comandos; `NewSucursalesHandler` cambia de 2 a 5 argumentos en la Tarea 5 (y su única llamada, en `configuracion.go`, se actualiza en ese mismo paso).
