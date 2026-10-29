# I3 rebanada 2 — Comandas y estados de cocina: plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implementar RES-RF-012 en el módulo nuevo `produccion`: enviar a una estación las cantidades pendientes de un pedido, consultar la cola de preparación y avanzar cada línea de comanda por `PENDIENTE → PREPARACION → LISTO → ENTREGADO`.

**Architecture:** Módulo `internal/modules/produccion` con las mismas capas que `atencion` (domain/application/infrastructure/web), sin importar otro módulo: lee pedido, líneas y estación de producto con su propio SQL. Usa las tablas `comandas` y `comanda_detalles` de la migración `00003`; solo añade la migración `00012` (permiso). Cada escritura es un caso de uso decorado con transacción e idempotencia (`escritura[C,T]`), como en `atencion`.

**Tech Stack:** Go 1.26, `net/http` (ServeMux), `database/sql` + pgx, PostgreSQL 18 (testcontainers en las pruebas de integración), goose.

Spec: `docs/superpowers/specs/2026-10-28-i3-rebanada2-comandas-design.md`. Rama: se continúa en `i3-rebanada1-pedidos` (la actual) salvo que el usuario indique otra.

## Global Constraints

- Toda ruta y comando se ejecuta desde `service-restaurant/` salvo que el paso diga otra cosa. Las pruebas de integración necesitan Docker (testcontainers).
- `produccion` **no importa** `atencion`, `catalogo` ni `identidad` (lo verifica `tests/architecture`). Dominio y aplicación no importan `database/sql`, `net/http` ni drivers. `web` no importa dominio ni infraestructura.
- Tablas siempre con el esquema: `sch_restaurante.<tabla>`.
- Cantidades y ids viajan como cadenas en JSON (contrato). Cantidades: `numeric(14,3)`, tres decimales como máximo.
- Orden de bloqueo (regla de R1): pedido `FOR UPDATE` primero, después las líneas por `id` ascendente.
- Errores (`apperror`): `422` validación, `409` conflicto de estado, `403`/`404` por sede. Sin ningún permiso del conjunto exigido: `403 sin_permiso`. Con el permiso en otra sede pero no en la del recurso: `403 sede_no_autorizada`. Recurso de una sede que el actor no ve: `404 no_encontrado`.
- Mensajes de usuario y comentarios en español, con tildes, como el resto del repositorio.
- Mensaje de cada commit termina con la línea `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>` (se pasa con un segundo `-m`).

## Ajustes respecto al spec (decididos al planificar)

| Spec | Plan | Motivo |
|---|---|---|
| §1: la migración asigna `COCINA_OPERAR` a COCINA "y a ADMIN si no lo tiene" | Solo a COCINA | `00001` asigna todos los permisos existentes a ADMIN con un `CROSS JOIN`, así que ADMIN ya lo tiene |
| §2: `POST …/transiciones` con `Idempotency-Key` | La transición **no** lleva clave | Mismo criterio que el `PATCH` de consumos de R1: operación sobre un recurso ya identificado; repetirla da 409 `transicion_invalida` |
| §3.2: «Sin ninguno de los dos: 403 `sin_permiso`» | `sin_permiso` si no tiene ninguno de los permisos en ninguna sede; `sede_no_autorizada` si lo tiene en otra sede pero no en la del recurso | Mismo contrato que `AutorizarPedidos` de R1 |
| §3.2: el cuerpo trae `estado` y `motivo` (contrato) | `estado` desconocido → 422 `estado_invalido`; `CANCELADO` → 409 `transicion_invalida`; `motivo` se acepta y se ignora | `CANCELADO` es de R3 |
| §3.3: la cola no define el valor por defecto de `estado` | Sin `estado`, la cola devuelve las comandas con alguna línea `PENDIENTE`, `PREPARACION` o `LISTO`; con `estado=X`, las que tienen alguna línea en `X`. Siempre se devuelven todas las líneas de la comanda | Una cola de cocina no debe mostrar el historial entregado |
| §3.3: «mesa si existe» | `mesa_id` (cadena o `null`) en cada comanda | El nombre de la mesa lo resuelve el frontend (R4) |
| §3.1: respuesta del envío | Comanda con `lineas[]` (contrato `Comandas` + `lineas` y `mesa_id`, se añaden en la Task 8) | La pantalla necesita las líneas creadas |

Fuera de alcance, y así queda dicho: `pedidos.estado` (`REGISTRADO/PREPARACION/LISTO/ENTREGADO`) **no** se actualiza desde las comandas en R2; el estado de cocina vive en las líneas de comanda. Derivar `pedidos.estado` se decidirá con R3/R4 o I4 (cierre de pedido).

## Mapa de archivos

Todo bajo `service-restaurant/` salvo que se indique.

Crear:

| Archivo | Responsabilidad |
|---|---|
| `migrations/00012_cocina_permisos.sql` | Asigna `COCINA_OPERAR` al rol COCINA |
| `internal/modules/produccion/domain/vo/cantidad.go` | Cantidad `numeric(14,3)` en milésimas |
| `internal/modules/produccion/domain/policy/permisos.go` | Constantes de permisos |
| `internal/modules/produccion/domain/policy/transiciones.go` | Tabla de transiciones, permisos por destino, cantidad pendiente |
| `internal/modules/produccion/application/dto/command/{intencion,comandas}.go` | Comandos `EnviarComanda`, `TransicionarLinea` |
| `internal/modules/produccion/application/dto/query/comandas.go` | Consulta `ListarComandas` |
| `internal/modules/produccion/application/dto/result/comandas.go` | `Comanda`, `ComandaLinea`, `PaginaComandas` |
| `internal/modules/produccion/application/port/inbound/comandas.go` | Puertos de entrada |
| `internal/modules/produccion/application/port/outbound/{errores,comandas}.go` | Puertos de salida y errores de almacenamiento |
| `internal/modules/produccion/application/mapper/{errores,autorizacion}.go` | Rechazos esperados y autorización por sede |
| `internal/modules/produccion/application/usecase/command/{helpers,comandas,transiciones}.go` | Casos de uso de escritura |
| `internal/modules/produccion/application/usecase/query/comandas.go` | Caso de uso de la cola |
| `internal/modules/produccion/infrastructure/persistence/internal/cargar/comanda.go` | Arma comandas con líneas (lectura y escritura) |
| `internal/modules/produccion/infrastructure/persistence/write/repository/{sql,comandas_sql}.go` | SQL de escritura |
| `internal/modules/produccion/infrastructure/persistence/write/adapter/comandas.go` | Adaptador de escritura |
| `internal/modules/produccion/infrastructure/persistence/read/repository/comandas_sql_reader.go` | SQL de la cola |
| `internal/modules/produccion/infrastructure/persistence/read/adapter/comandas_read_adapter.go` | Adaptador de lectura |
| `internal/modules/produccion/infrastructure/config/{escritura,module,comandas}.go` | Composición del módulo |
| `internal/modules/produccion/web/dto/request/comandas.go`, `web/dto/response/comandas.go` | Cuerpos HTTP |
| `internal/modules/produccion/web/mapper/comandas_web_mapper.go` | Resultado → respuesta |
| `internal/modules/produccion/web/controller/{helpers,comandas_handler,routes}.go` | Adaptador HTTP |
| `tests/integration/cocina_permisos_migracion_test.go`, `cocina_helpers_test.go`, `cocina_comandas_test.go`, `cocina_transiciones_test.go`, `cocina_cola_test.go` | Pruebas de integración |

Modificar: `internal/bootstrap/app.go`, `internal/bootstrap/router.go`, `docs/planificacion/03_api/openapi.yaml`, el expediente en `docs/planificacion/`, `service-restaurant/docs/module-map.md` y el spec de R2 (sección de ajustes).

---

### Task 1: Migración 00012 — permiso de cocina

**Files:**
- Create: `migrations/00012_cocina_permisos.sql`
- Create: `tests/integration/cocina_permisos_migracion_test.go`

**Interfaces:**
- Produces: el rol COCINA tiene `COCINA_OPERAR`. Las Tasks 5 y 6 lo usan en sus pruebas.

- [ ] **Step 1: Escribir la prueba que falla**

`tests/integration/cocina_permisos_migracion_test.go`:

```go
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
```

- [ ] **Step 2: Ejecutar y comprobar que falla**

Run: `go test ./tests/integration/ -run TestMigracion00012 -count=1 -v`
Expected: FAIL (`COCINA debe recibir COCINA_OPERAR (0)` y la lectura de `00012_cocina_permisos.sql` falla).

- [ ] **Step 3: Escribir la migración**

`migrations/00012_cocina_permisos.sql`:

```sql
-- Migracion 00012: permiso de cocina (I3, rebanada 2).
-- COCINA_OPERAR existe desde 00001 y solo lo tiene ADMIN (00001 asigna todos los permisos a ADMIN):
-- aqui se asigna al rol COCINA. PEDIDO_VER para COCINA ya lo dio 00011.
-- Requiere PostgreSQL 18 y las migraciones 00001-00011.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='COCINA' AND p.codigo='COCINA_OPERAR'
  AND NOT EXISTS (SELECT 1 FROM roles_permisos x WHERE x.rol_id=r.id AND x.permiso_id=p.id);

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo='COCINA')
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='COCINA_OPERAR');
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
```

- [ ] **Step 4: Ejecutar y comprobar que pasa**

Run: `go test ./tests/integration/ -run "TestMigracion0001[0-2]|TestMigrations" -count=1 -v`
Expected: PASS (incluye las pruebas de migraciones anteriores; si `migrations_test.go` cuenta migraciones y falla por el número, actualizar ese conteo a 12 y decirlo en el commit).

- [ ] **Step 5: Commit**

```bash
git add migrations/00012_cocina_permisos.sql tests/integration/cocina_permisos_migracion_test.go
git commit -m "I3 rebanada 2: migración 00012, COCINA_OPERAR para el rol COCINA" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Dominio — cantidad, permisos y transiciones

**Files:**
- Create: `internal/modules/produccion/domain/vo/cantidad.go`, `cantidad_test.go`
- Create: `internal/modules/produccion/domain/policy/permisos.go`
- Create: `internal/modules/produccion/domain/policy/transiciones.go`, `transiciones_test.go`

**Interfaces:**
- Produces (`vo`): `type Cantidad struct`; `ErrValorInvalido`; `NewCantidad(raw string) (Cantidad, error)` (positiva); `CantidadDeAlmacen(raw string) (Cantidad, error)` (acepta cero); métodos `Milis() int64`, `Excede(o Cantidad) bool`, `Menos(o Cantidad) Cantidad` (nunca negativa), `String() string` (siempre tres decimales).
- Produces (`policy`): constantes `PedidoVer, PedidoCrear, PedidoEditar, CocinaOperar` y `EstadoPendiente, EstadoPreparacion, EstadoListo, EstadoEntregado, EstadoCancelado`; `EstadoDestinoValido(string) bool`; `Permite(actual, destino string) bool`; `PermisosDe(destino string) []string`; `Pendiente(total, enviado vo.Cantidad) vo.Cantidad`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`internal/modules/produccion/domain/vo/cantidad_test.go`:

```go
package vo

import "testing"

func TestNewCantidad(t *testing.T) {
	validos := map[string]int64{"1": 1000, "2.5": 2500, " 0.001 ": 1, "10.250": 10250, "00003": 3000, "99999999999.999": 99999999999999}
	for raw, milis := range validos {
		c, err := NewCantidad(raw)
		if err != nil || c.Milis() != milis {
			t.Errorf("NewCantidad(%q) = %d, %v; quería %d", raw, c.Milis(), err, milis)
		}
	}
	for _, raw := range []string{"", "0", "0.000", "-1", "1.2345", "abc", "1e3", "1,5", ".5", "5.", "123456789012"} {
		if _, err := NewCantidad(raw); err == nil {
			t.Errorf("NewCantidad(%q) debía fallar", raw)
		}
	}
}

func TestCantidadDeAlmacenAceptaCero(t *testing.T) {
	for _, raw := range []string{"0", "0.000", "2.000"} {
		if _, err := CantidadDeAlmacen(raw); err != nil {
			t.Errorf("CantidadDeAlmacen(%q): %v", raw, err)
		}
	}
	if _, err := CantidadDeAlmacen("-1"); err == nil {
		t.Error("una cantidad negativa debía fallar")
	}
}

func TestCantidadAritmeticaYTexto(t *testing.T) {
	a, _ := NewCantidad("3")
	b, _ := NewCantidad("1.25")
	if got := a.Menos(b).String(); got != "1.750" {
		t.Errorf("3 - 1.25 = %s", got)
	}
	if got := b.Menos(a).Milis(); got != 0 {
		t.Errorf("la resta no baja de cero: %d", got)
	}
	if !a.Excede(b) || b.Excede(a) || a.Excede(a) {
		t.Error("Excede mal definido")
	}
	if got := (Cantidad{}).String(); got != "0.000" {
		t.Errorf("cero = %s", got)
	}
}
```

`internal/modules/produccion/domain/policy/transiciones_test.go`:

```go
package policy

import (
	"slices"
	"testing"

	"service-restaurant/internal/modules/produccion/domain/vo"
)

func TestPermiteSoloElSiguientePaso(t *testing.T) {
	estados := []string{EstadoPendiente, EstadoPreparacion, EstadoListo, EstadoEntregado, EstadoCancelado}
	validas := map[[2]string]bool{
		{EstadoPendiente, EstadoPreparacion}: true,
		{EstadoPreparacion, EstadoListo}:     true,
		{EstadoListo, EstadoEntregado}:       true,
	}
	for _, a := range estados {
		for _, d := range estados {
			if got := Permite(a, d); got != validas[[2]string{a, d}] {
				t.Errorf("Permite(%s, %s) = %v", a, d, got)
			}
		}
	}
	if Permite("INEXISTENTE", EstadoListo) {
		t.Error("un estado desconocido no transiciona")
	}
}

func TestEstadoDestinoValido(t *testing.T) {
	for _, s := range []string{EstadoPreparacion, EstadoListo, EstadoEntregado, EstadoCancelado} {
		if !EstadoDestinoValido(s) {
			t.Errorf("%s es un destino válido", s)
		}
	}
	for _, s := range []string{"", EstadoPendiente, "preparacion", "OTRO"} {
		if EstadoDestinoValido(s) {
			t.Errorf("%q no es un destino válido", s)
		}
	}
}

func TestPermisosDelDestino(t *testing.T) {
	if got := PermisosDe(EstadoPreparacion); !slices.Equal(got, []string{CocinaOperar}) {
		t.Errorf("PREPARACION: %v", got)
	}
	if got := PermisosDe(EstadoListo); !slices.Equal(got, []string{CocinaOperar}) {
		t.Errorf("LISTO: %v", got)
	}
	if got := PermisosDe(EstadoEntregado); !slices.Equal(got, []string{CocinaOperar, PedidoEditar}) {
		t.Errorf("ENTREGADO: %v", got)
	}
}

func TestPendiente(t *testing.T) {
	total, _ := vo.NewCantidad("3")
	enviado, _ := vo.CantidadDeAlmacen("1.5")
	if got := Pendiente(total, enviado).String(); got != "1.500" {
		t.Errorf("pendiente = %s", got)
	}
	todo, _ := vo.CantidadDeAlmacen("3.000")
	if got := Pendiente(total, todo).Milis(); got != 0 {
		t.Errorf("sin pendiente = %d", got)
	}
}
```

- [ ] **Step 2: Ejecutar y comprobar que falla**

Run: `go test ./internal/modules/produccion/domain/... -count=1`
Expected: FAIL de compilación (paquetes sin código).

- [ ] **Step 3: Implementar**

`internal/modules/produccion/domain/vo/cantidad.go`:

```go
// Package vo contiene los objetos de valor del módulo produccion; solo usa la biblioteca estándar.
package vo

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ErrValorInvalido = errors.New("valor inválido")

var cantidadRe = regexp.MustCompile(`^[0-9]{1,11}(\.[0-9]{1,3})?$`)

// Cantidad es una cantidad decimal exacta de numeric(14,3), guardada en milésimas para sumar y
// restar sin coma flotante.
type Cantidad struct{ milis int64 }

func parsear(raw string) (Cantidad, error) {
	v := strings.TrimSpace(raw)
	if !cantidadRe.MatchString(v) {
		return Cantidad{}, ErrValorInvalido
	}
	entera, fraccion, _ := strings.Cut(v, ".")
	fraccion += strings.Repeat("0", 3-len(fraccion))
	n, err := strconv.ParseInt(entera+fraccion, 10, 64)
	if err != nil {
		return Cantidad{}, ErrValorInvalido
	}
	return Cantidad{n}, nil
}

// NewCantidad exige una cantidad estrictamente positiva: la que llega en una solicitud.
func NewCantidad(raw string) (Cantidad, error) {
	c, err := parsear(raw)
	if err != nil || c.milis <= 0 {
		return Cantidad{}, ErrValorInvalido
	}
	return c, nil
}

// CantidadDeAlmacen lee una cantidad que viene de la base; acepta cero (nada enviado todavía).
func CantidadDeAlmacen(raw string) (Cantidad, error) { return parsear(raw) }

func (c Cantidad) Milis() int64           { return c.milis }
func (c Cantidad) Excede(o Cantidad) bool { return c.milis > o.milis }

// Menos resta o de c sin bajar de cero.
func (c Cantidad) Menos(o Cantidad) Cantidad {
	if o.milis >= c.milis {
		return Cantidad{}
	}
	return Cantidad{c.milis - o.milis}
}

// String devuelve la cantidad con tres decimales, igual que numeric(14,3).
func (c Cantidad) String() string {
	return fmt.Sprintf("%d.%03d", c.milis/1000, c.milis%1000)
}
```

`internal/modules/produccion/domain/policy/permisos.go`:

```go
// Package policy contiene las reglas de permiso y de estado del módulo produccion.
package policy

// Permisos que usa el módulo. Se declaran aquí porque produccion no importa identidad ni
// atencion; el catálogo de permisos vive en la base (00001, 00011 y 00012).
const (
	PedidoVer     = "PEDIDO_VER"
	PedidoCrear   = "PEDIDO_CREAR"
	PedidoEditar  = "PEDIDO_EDITAR"
	CocinaOperar  = "COCINA_OPERAR"
)
```

(Ejecutar `gofmt -w` sobre el paquete en el Step 4 alinea las constantes.)

`internal/modules/produccion/domain/policy/transiciones.go`:

```go
package policy

import "service-restaurant/internal/modules/produccion/domain/vo"

// Estados de una línea de comanda (comanda_detalles.estado).
const (
	EstadoPendiente   = "PENDIENTE"
	EstadoPreparacion = "PREPARACION"
	EstadoListo       = "LISTO"
	EstadoEntregado   = "ENTREGADO"
	EstadoCancelado   = "CANCELADO"
)

// siguiente es el único paso permitido desde cada estado. CANCELADO no se alcanza desde aquí: es
// de la rebanada 3.
var siguiente = map[string]string{
	EstadoPendiente:   EstadoPreparacion,
	EstadoPreparacion: EstadoListo,
	EstadoListo:       EstadoEntregado,
}

// EstadoDestinoValido indica si el valor es un destino que el contrato admite en una transición.
func EstadoDestinoValido(destino string) bool {
	switch destino {
	case EstadoPreparacion, EstadoListo, EstadoEntregado, EstadoCancelado:
		return true
	}
	return false
}

// Permite indica si de actual se puede pasar a destino: solo al paso siguiente, sin saltos ni
// retrocesos.
func Permite(actual, destino string) bool {
	s, ok := siguiente[actual]
	return ok && s == destino
}

// PermisosDe devuelve los permisos que habilitan la transición hacia destino (basta uno). La
// entrega también la puede confirmar quien edita pedidos: el mesero que lleva el plato.
func PermisosDe(destino string) []string {
	if destino == EstadoEntregado {
		return []string{CocinaOperar, PedidoEditar}
	}
	return []string{CocinaOperar}
}

// Pendiente es lo que falta por enviar de una línea: su cantidad menos lo ya enviado (sin contar
// lo cancelado).
func Pendiente(total, enviado vo.Cantidad) vo.Cantidad { return total.Menos(enviado) }
```

- [ ] **Step 4: Formatear, ejecutar y comprobar que pasa**

Run: `gofmt -w internal/modules/produccion && go test ./internal/modules/produccion/domain/... -count=1 -v`
Expected: PASS (`TestNewCantidad`, `TestCantidadDeAlmacenAceptaCero`, `TestCantidadAritmeticaYTexto`, `TestPermiteSoloElSiguientePaso`, `TestEstadoDestinoValido`, `TestPermisosDelDestino`, `TestPendiente`).

- [ ] **Step 5: Commit**

```bash
git add internal/modules/produccion/domain
git commit -m "I3 rebanada 2: dominio de produccion (cantidad, permisos y transiciones de cocina)" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Aplicación — caso de uso EnviarComanda

**Files:**
- Create: `internal/modules/produccion/application/dto/command/intencion.go`, `comandas.go`
- Create: `internal/modules/produccion/application/dto/result/comandas.go`
- Create: `internal/modules/produccion/application/port/outbound/errores.go`, `comandas.go`
- Create: `internal/modules/produccion/application/port/inbound/comandas.go`
- Create: `internal/modules/produccion/application/mapper/errores.go`, `autorizacion.go`
- Create: `internal/modules/produccion/application/usecase/command/helpers.go`, `comandas.go`, `comandas_test.go`

**Interfaces:**
- Consumes: Task 2 (`vo.NewCantidad`, `vo.CantidadDeAlmacen`, `policy.*`).
- Produces:
  - `command.Intencion{Clave string; Operacion string; Huella [32]byte}`; `command.LineaEnvio{DetallePedidoID int64; Cantidad string}`; `command.EnviarComanda{Intencion; Actor appctx.Actor; PedidoID, EstacionID int64; Lineas []LineaEnvio}`.
  - `result.ComandaLinea`, `result.Comanda`, `result.PaginaComandas` (campos exactos abajo).
  - `outbound.EnvioStore` (métodos abajo), `outbound.ErrClaveEnUso`, `outbound.ErrDuplicado`, `outbound.ErrReferenciaInvalida`, `outbound.ErrReglaDatos`.
  - `inbound.EnviarComanda`: `Execute(context.Context, cmd.EnviarComanda) (appresult.Result[dtoresult.Comanda], error)`.
  - `command.NewEnviarComanda(outbound.EnvioStore) *EnviarComanda`.
  - `mapper`: `Requiere`, `Autorizar`, `Visible`, `SedesCocina`, `NoAutenticado`, `NoEncontrado`, `NoEncontradoM`, `Validacion`, `DeAlmacen`, `EncodeCursor`, `DecodeCursor`, constantes `CodigoSinPermiso`, `CodigoSedeNoAutorizada`.

- [ ] **Step 1: Crear los DTO, los puertos y el mapper (compilan solos)**

`application/dto/command/intencion.go`:

```go
// Package command agrupa los DTO de comandos del módulo produccion.
package command

// Intencion identifica una operación para la idempotencia (ADR-04). La rellena la capa web con
// la clave del encabezado, la operación y la huella del contenido; los casos de uso solo leen Clave.
type Intencion struct {
	Clave     string
	Operacion string // p. ej. "POST /pedidos/7/comandas" (con el id concreto)
	Huella    [32]byte
}
```

`application/dto/command/comandas.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// LineaEnvio es una línea del pedido y la cantidad que se envía ahora a cocina (decimal en cadena).
type LineaEnvio struct {
	DetallePedidoID int64
	Cantidad        string
}

// EnviarComanda envía a una estación las cantidades indicadas de las líneas de un pedido.
type EnviarComanda struct {
	Intencion
	Actor      appctx.Actor
	PedidoID   int64
	EstacionID int64
	Lineas     []LineaEnvio
}

// TransicionarLinea mueve una línea de comanda a su estado siguiente. Motivo se acepta por contrato
// y se ignora hasta la rebanada 3 (cancelaciones).
type TransicionarLinea struct {
	Actor   appctx.Actor
	LineaID int64
	Estado  string
	Motivo  *string
}
```

`application/dto/result/comandas.go`:

```go
// Package result agrupa los resultados de los casos de uso de produccion.
package result

import "time"

// ComandaLinea es una línea de comanda: una tanda de una línea de pedido enviada a una estación.
// Descripcion e Instrucciones son la instantánea de la línea al enviar.
type ComandaLinea struct {
	ID                int64
	ComandaID         int64
	PedidoID          int64
	DetallePedidoID   int64
	Cantidad          string
	Descripcion       string
	Instrucciones     *string
	Estado            string
	IniciadoEn        *time.Time
	ListoEn           *time.Time
	EntregadoEn       *time.Time
	MotivoCancelacion *string
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
}

// Comanda es un envío a una estación con sus líneas. No guarda estado propio: se deriva de ellas.
type Comanda struct {
	ID                int64
	PedidoID          int64
	SucursalID        int64
	EstacionID        int64
	MesaID            *int64
	ClaveIdempotencia string
	EnviadoEn         time.Time
	EnviadoPor        int64
	Notas             *string
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
	Lineas            []ComandaLinea
}

// PaginaComandas es una página de la cola; NextCursor vacío indica la última.
type PaginaComandas struct {
	Items      []Comanda
	NextCursor string
}
```

`application/port/outbound/errores.go`:

```go
// Package outbound declara los puertos de salida del módulo produccion.
package outbound

import (
	"errors"
	"fmt"
)

// Errores de almacenamiento que los casos de uso traducen a rechazos esperados.
var (
	ErrDuplicado          = errors.New("registro duplicado")
	ErrReferenciaInvalida = errors.New("referencia inexistente")
	ErrClaveEnUso         = fmt.Errorf("la clave de idempotencia ya fue usada por otra comanda: %w", ErrDuplicado)
)

// ErrReglaDatos señala una regla de integridad de la base (triggers de la V4).
type ErrReglaDatos struct{ Mensaje string }

func (e *ErrReglaDatos) Error() string { return "regla de datos: " + e.Mensaje }
```

`application/port/outbound/comandas.go`:

```go
package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
)

// PedidoRef resume un pedido bloqueado. Cerrado incluye los cancelados.
type PedidoRef struct {
	ID         int64
	SucursalID int64
	Cerrado    bool
}

// EstacionRef resume una estación de preparación.
type EstacionRef struct {
	ID         int64
	SucursalID int64
	EsActiva   bool
}

// LineaPedidoRef es una línea de pedido bloqueada, con lo que hace falta para enviarla.
// Enviado es lo ya enviado a cocina sin contar lo cancelado (decimal en cadena). EstacionID es la
// estación del producto en la sede del pedido, o nil si no tiene (p. ej. un SERVICIO).
type LineaPedidoRef struct {
	ID            int64
	CuentaID      int64
	CuentaEstado  string
	Estado        string
	Cantidad      string
	Enviado       string
	Descripcion   string
	Instrucciones *string
	EstacionID    *int64
}

// NuevaComandaLinea es la tanda de una línea que se envía: cantidad más la instantánea.
type NuevaComandaLinea struct {
	DetallePedidoID int64
	Cantidad        string
	Descripcion     string
	Instrucciones   *string
}

// NuevaComanda es un envío completo. ClaveIdempotencia es el UUID de la solicitud.
type NuevaComanda struct {
	PedidoID          int64
	SucursalID        int64
	EstacionID        int64
	ClaveIdempotencia string
	EnviadoPor        int64
	Lineas            []NuevaComandaLinea
}

// EnvioStore escribe un envío a cocina dentro de la transacción del comando. Los métodos que
// devuelven una referencia bloquean las filas hasta el final de la transacción.
type EnvioStore interface {
	// BloquearPedido lee el pedido FOR UPDATE.
	BloquearPedido(ctx context.Context, id int64) (PedidoRef, bool, error)
	// Estacion lee la estación en modo compartido.
	Estacion(ctx context.Context, id int64) (EstacionRef, bool, error)
	// LineasParaEnvio bloquea FOR UPDATE las líneas del pedido con esos ids, ordenadas por id. Las
	// que no existen o son de otro pedido simplemente no vienen en el resultado.
	LineasParaEnvio(ctx context.Context, pedidoID, sucursalID int64, ids []int64) ([]LineaPedidoRef, error)
	// CrearComanda inserta la comanda y sus líneas y la devuelve ya armada.
	CrearComanda(ctx context.Context, n NuevaComanda) (dtoresult.Comanda, error)
}
```

`application/port/inbound/comandas.go`:

```go
// Package inbound declara los puertos de entrada del módulo produccion.
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type EnviarComanda interface {
	Execute(context.Context, cmd.EnviarComanda) (appresult.Result[dtoresult.Comanda], error)
}
```

`application/mapper/errores.go`:

```go
// Package mapper convierte errores en rechazos esperados de aplicación y resuelve la
// autorización por sede del módulo produccion.
package mapper

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	CodigoNoAutenticado      = "no_autenticado"
	CodigoSinPermiso         = "sin_permiso"
	CodigoNoEncontrado       = "no_encontrado"
	CodigoDuplicado          = "registro_duplicado"
	CodigoReferenciaInvalida = "referencia_invalida"
	CodigoReglaDatos         = "regla_de_datos"
	CodigoSedeNoAutorizada   = "sede_no_autorizada"
	CodigoCursorInvalido     = "cursor_invalido"
)

func NoAutenticado() *apperror.AppError {
	return apperror.New(apperror.KindUnauthorized, CodigoNoAutenticado, "autenticación requerida")
}

// NoEncontrado: para sustantivos femeninos ("sucursal", "línea de comanda").
func NoEncontrado(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrada")
}

// NoEncontradoM es como NoEncontrado para sustantivos masculinos ("pedido").
func NoEncontradoM(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrado")
}

func Validacion(code, msg string) *apperror.AppError {
	return apperror.New(apperror.KindValidation, code, msg)
}

// DeAlmacen traduce los errores de almacenamiento conocidos a rechazos esperados. Devuelve nil si
// err no es uno de ellos (error técnico: debe seguir por el canal error).
func DeAlmacen(err error) *apperror.AppError {
	var regla *outbound.ErrReglaDatos
	switch {
	case err == nil:
		return nil
	case errors.Is(err, outbound.ErrDuplicado):
		return apperror.New(apperror.KindConflict, CodigoDuplicado, "ya existe un registro con ese valor único")
	case errors.Is(err, outbound.ErrReferenciaInvalida):
		return apperror.New(apperror.KindValidation, CodigoReferenciaInvalida, "una referencia no existe")
	case errors.As(err, &regla):
		return apperror.New(apperror.KindBusinessRule, CodigoReglaDatos, mensajeRegla(regla.Mensaje))
	}
	return nil
}

// referenciaRe reconoce el rechazo del trigger fn_referencias_activas:
// "Referencia estacion_id=12: maestro estaciones_preparacion inexistente o dado de baja".
var referenciaRe = regexp.MustCompile(`^Referencia \w+=\d+: maestro (\w+) inexistente o dado de baja$`)

var maestros = map[string]string{"estaciones_preparacion": "estación", "sucursales": "sucursal"}

// mensajeRegla deja pasar los mensajes curados del esquema y reemplaza por un texto para el
// usuario los que exponen nombres internos del trigger de referencias.
func mensajeRegla(msg string) string {
	if !strings.HasPrefix(msg, "Referencia ") {
		return msg
	}
	if m := referenciaRe.FindStringSubmatch(msg); m != nil {
		if nombre, ok := maestros[m[1]]; ok {
			return "la " + nombre + " indicada no existe o está dada de baja"
		}
	}
	return "una referencia no existe o está dada de baja"
}

// EncodeCursor: el cursor es el último id entregado, opaco para el cliente.
func EncodeCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

// DecodeCursor acepta solo el id en decimal canónico (sin "+" ni ceros a la izquierda): así cada
// página tiene un único cursor válido.
func DecodeCursor(c string) (int64, *apperror.AppError) {
	if c == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err == nil {
		if id, perr := strconv.ParseInt(string(raw), 10, 64); perr == nil && id >= 0 && strconv.FormatInt(id, 10) == string(raw) {
			return id, nil
		}
	}
	return 0, apperror.New(apperror.KindBadRequest, CodigoCursorInvalido, "cursor inválido")
}
```

`application/mapper/autorizacion.go`:

```go
package mapper

import (
	"service-restaurant/internal/modules/produccion/domain/policy"
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

// Visible indica si el actor tiene algún permiso de pedidos o de cocina en la sede: si no, no se
// le revela que el recurso existe.
func Visible(a appctx.Actor, sedeID int64) bool {
	return a.Can(policy.PedidoVer, sedeID) || a.Can(policy.PedidoCrear, sedeID) ||
		a.Can(policy.PedidoEditar, sedeID) || a.Can(policy.CocinaOperar, sedeID)
}

// Autorizar decide si el actor puede operar un recurso de la sede con alguno de los permisos: si no
// ve la sede responde ante404; si la ve pero no tiene ninguno de los permisos, 403 sede_no_autorizada.
func Autorizar(a appctx.Actor, sedeID int64, ante404 *apperror.AppError, permisos ...string) *apperror.AppError {
	if !Visible(a, sedeID) {
		return ante404
	}
	for _, p := range permisos {
		if a.Can(p, sedeID) {
			return nil
		}
	}
	return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada, "no tiene permiso para esta operación de cocina en esta sucursal")
}

// SedesCocina devuelve, ordenadas, las sedes donde el actor opera la cocina. Nunca es nil, para que
// un parámetro SQL vacío sea un arreglo vacío.
func SedesCocina(a appctx.Actor) []int64 {
	ids := a.SucursalesCon(policy.CocinaOperar)
	if ids == nil {
		return []int64{}
	}
	return ids
}
```

`application/usecase/command/helpers.go`:

```go
// Package command implementa los comandos del módulo produccion.
package command

import (
	"fmt"

	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

// rechazar devuelve un rechazo esperado (no un error técnico).
func rechazar[T any](ae *apperror.AppError) (appresult.Result[T], error) {
	return appresult.Fail[T](ae), nil
}

// tecnico envuelve un error inesperado; el Result queda en su valor cero (inválido).
func tecnico[T any](contexto string, err error) (appresult.Result[T], error) {
	return appresult.Result[T]{}, fmt.Errorf("%s: %w", contexto, err)
}

// deAlmacen separa un error de almacenamiento conocido (rechazo) de uno técnico.
func deAlmacen[T any](contexto string, err error) (appresult.Result[T], error) {
	if ae := mapper.DeAlmacen(err); ae != nil {
		return rechazar[T](ae)
	}
	return tecnico[T](contexto, err)
}
```

- [ ] **Step 2: Escribir las pruebas del caso de uso (fallan: no existe `NewEnviarComanda`)**

`application/usecase/command/comandas_test.go`:

```go
package command

import (
	"context"
	"errors"
	"slices"
	"testing"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

const claveEjemplo = "6f1c0a52-3c1e-4a53-9a39-0b9a3a2f1a11"

func idPtr(n int64) *int64     { return &n }
func texto(s string) *string   { return &s }

func actorEn(sede int64, permisos ...string) appctx.Actor {
	g := make([]appctx.Grant, 0, len(permisos))
	for _, p := range permisos {
		g = append(g, appctx.Grant{SucursalID: sede, Permiso: p})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

// envioStoreFake implementa outbound.EnvioStore con respuestas fijadas por la prueba.
type envioStoreFake struct {
	pedido         outbound.PedidoRef
	pedidoExiste   bool
	estacion       outbound.EstacionRef
	estacionExiste bool
	lineas         []outbound.LineaPedidoRef
	errCrear       error

	idsPedidos []int64
	creada     outbound.NuevaComanda
	creaciones int
}

var _ outbound.EnvioStore = (*envioStoreFake)(nil)

func (f *envioStoreFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *envioStoreFake) Estacion(context.Context, int64) (outbound.EstacionRef, bool, error) {
	return f.estacion, f.estacionExiste, nil
}
func (f *envioStoreFake) LineasParaEnvio(_ context.Context, _, _ int64, ids []int64) ([]outbound.LineaPedidoRef, error) {
	f.idsPedidos = ids
	var out []outbound.LineaPedidoRef
	for _, l := range f.lineas {
		if slices.Contains(ids, l.ID) {
			out = append(out, l)
		}
	}
	return out, nil
}
func (f *envioStoreFake) CrearComanda(_ context.Context, n outbound.NuevaComanda) (dtoresult.Comanda, error) {
	f.creaciones++
	f.creada = n
	return dtoresult.Comanda{ID: 9, PedidoID: n.PedidoID}, f.errCrear
}

func fakeEnvio() *envioStoreFake {
	return &envioStoreFake{
		pedido:         outbound.PedidoRef{ID: 10, SucursalID: 1},
		pedidoExiste:   true,
		estacion:       outbound.EstacionRef{ID: 5, SucursalID: 1, EsActiva: true},
		estacionExiste: true,
		lineas: []outbound.LineaPedidoRef{{
			ID: 100, CuentaID: 20, CuentaEstado: "ABIERTA", Estado: "ACTIVO", Cantidad: "3.000", Enviado: "0",
			Descripcion: "Lomo saltado", Instrucciones: texto("sin ají"), EstacionID: idPtr(5),
		}},
	}
}

func envioBase() cmd.EnviarComanda {
	return cmd.EnviarComanda{
		Intencion: cmd.Intencion{Clave: claveEjemplo},
		Actor:     actorEn(1, policy.PedidoCrear),
		PedidoID:  10, EstacionID: 5,
		Lineas: []cmd.LineaEnvio{{DetallePedidoID: 100, Cantidad: "2"}},
	}
}

func TestEnviarComandaCreaLaTandaConLaInstantanea(t *testing.T) {
	f := fakeEnvio()
	res, err := NewEnviarComanda(f).Execute(context.Background(), envioBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía enviarse: %v %v", err, res.Err())
	}
	n := f.creada
	if f.creaciones != 1 || n.PedidoID != 10 || n.SucursalID != 1 || n.EstacionID != 5 ||
		n.ClaveIdempotencia != claveEjemplo || n.EnviadoPor != 7 || len(n.Lineas) != 1 {
		t.Fatalf("comanda mal armada: %+v", n)
	}
	l := n.Lineas[0]
	if l.DetallePedidoID != 100 || l.Cantidad != "2.000" || l.Descripcion != "Lomo saltado" ||
		l.Instrucciones == nil || *l.Instrucciones != "sin ají" {
		t.Fatalf("línea mal armada: %+v", l)
	}
}

func TestEnviarComandaAceptaElRestoPendiente(t *testing.T) {
	f := fakeEnvio()
	f.lineas[0].Enviado = "2.000"
	c := envioBase()
	c.Lineas[0].Cantidad = "1"
	if res, err := NewEnviarComanda(f).Execute(context.Background(), c); err != nil || !res.IsOk() {
		t.Fatalf("el resto pendiente debía enviarse: %v %v", err, res.Err())
	}
}

func TestEnviarComandaPideLasLineasOrdenadasYSinRepetir(t *testing.T) {
	f := fakeEnvio()
	f.lineas = append(f.lineas, outbound.LineaPedidoRef{ID: 50, CuentaID: 20, CuentaEstado: "ABIERTA", Estado: "ACTIVO",
		Cantidad: "1.000", Enviado: "0", Descripcion: "Chicha", EstacionID: idPtr(5)})
	c := envioBase()
	c.Lineas = []cmd.LineaEnvio{{DetallePedidoID: 100, Cantidad: "1"}, {DetallePedidoID: 50, Cantidad: "1"}}
	if res, err := NewEnviarComanda(f).Execute(context.Background(), c); err != nil || !res.IsOk() {
		t.Fatalf("debía enviarse: %v %v", err, res.Err())
	}
	if !slices.Equal(f.idsPedidos, []int64{50, 100}) {
		t.Fatalf("los ids deben pedirse ascendentes (orden de bloqueo): %v", f.idsPedidos)
	}
}

func TestEnviarComandaRechaza(t *testing.T) {
	casos := []struct {
		nombre string
		ajusta func(f *envioStoreFake, c *cmd.EnviarComanda)
		kind   apperror.Kind
		code   string
	}{
		{"sin permiso en ninguna sede", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Actor = actorEn(1, policy.PedidoVer) },
			apperror.KindForbidden, "sin_permiso"},
		{"sin actor", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Actor = appctx.Actor{} },
			apperror.KindUnauthorized, "no_autenticado"},
		{"sin líneas", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas = nil },
			apperror.KindValidation, "lineas_invalidas"},
		{"demasiadas líneas", func(_ *envioStoreFake, c *cmd.EnviarComanda) {
			c.Lineas = make([]cmd.LineaEnvio, 101)
			for i := range c.Lineas {
				c.Lineas[i] = cmd.LineaEnvio{DetallePedidoID: int64(i + 1), Cantidad: "1"}
			}
		}, apperror.KindValidation, "lineas_invalidas"},
		{"línea repetida", func(_ *envioStoreFake, c *cmd.EnviarComanda) {
			c.Lineas = append(c.Lineas, cmd.LineaEnvio{DetallePedidoID: 100, Cantidad: "1"})
		}, apperror.KindValidation, "linea_duplicada"},
		{"id de línea inválido", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].DetallePedidoID = 0 },
			apperror.KindValidation, "linea_invalida"},
		{"cantidad cero", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].Cantidad = "0" },
			apperror.KindValidation, "cantidad_invalida"},
		{"cantidad con demasiados decimales", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].Cantidad = "1.2345" },
			apperror.KindValidation, "cantidad_invalida"},
		{"pedido inexistente", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.pedidoExiste = false },
			apperror.KindNotFound, "no_encontrado"},
		{"sede que el actor no ve", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.pedido.SucursalID = 2 },
			apperror.KindNotFound, "no_encontrado"},
		{"ve la sede pero no puede crear", func(f *envioStoreFake, c *cmd.EnviarComanda) {
			f.pedido.SucursalID = 2
			c.Actor = appctx.NewActor(7, "ana", 1, []appctx.Grant{
				{SucursalID: 1, Permiso: policy.PedidoCrear}, {SucursalID: 2, Permiso: policy.PedidoVer}})
		}, apperror.KindForbidden, "sede_no_autorizada"},
		{"pedido cerrado", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.pedido.Cerrado = true },
			apperror.KindConflict, "pedido_cerrado"},
		{"estación inexistente", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.estacionExiste = false },
			apperror.KindValidation, "estacion_invalida"},
		{"estación de otra sede", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.estacion.SucursalID = 2 },
			apperror.KindValidation, "estacion_invalida"},
		{"estación dada de baja", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.estacion.EsActiva = false },
			apperror.KindValidation, "estacion_invalida"},
		{"línea que no es del pedido", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas = nil },
			apperror.KindValidation, "linea_invalida"},
		{"línea anulada", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].Estado = "ANULADO" },
			apperror.KindConflict, "linea_anulada"},
		{"cuenta cerrada", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].CuentaEstado = "CERRADA" },
			apperror.KindConflict, "cuenta_cerrada"},
		{"producto sin estación", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].EstacionID = nil },
			apperror.KindValidation, "estacion_no_corresponde"},
		{"producto de otra estación", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].EstacionID = idPtr(6) },
			apperror.KindValidation, "estacion_no_corresponde"},
		{"excede el pendiente", func(_ *envioStoreFake, c *cmd.EnviarComanda) { c.Lineas[0].Cantidad = "3.001" },
			apperror.KindConflict, "cantidad_excede_pendiente"},
		{"excede lo que queda tras un envío previo", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.lineas[0].Enviado = "1.500" },
			apperror.KindConflict, "cantidad_excede_pendiente"},
		{"clave de idempotencia usada por otra comanda", func(f *envioStoreFake, _ *cmd.EnviarComanda) { f.errCrear = outbound.ErrClaveEnUso },
			apperror.KindConflict, "clave_en_uso"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeEnvio()
			c := envioBase()
			tc.ajusta(f, &c)
			res, err := NewEnviarComanda(f).Execute(context.Background(), c)
			if err != nil {
				t.Fatalf("un rechazo esperado no es un error técnico: %v", err)
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if tc.code != "clave_en_uso" && f.creaciones != 0 {
				t.Fatalf("un envío rechazado no debe crear la comanda")
			}
		})
	}
}

func TestEnviarComandaPropagaUnErrorTecnico(t *testing.T) {
	f := fakeEnvio()
	f.errCrear = errors.New("se cayó la base")
	res, err := NewEnviarComanda(f).Execute(context.Background(), envioBase())
	if err == nil || res.IsValid() {
		t.Fatalf("un error técnico debe salir por el canal error: %v %+v", err, res)
	}
}
```

- [ ] **Step 3: Ejecutar y comprobar que falla**

Run: `go test ./internal/modules/produccion/application/... -count=1`
Expected: FAIL de compilación: `undefined: NewEnviarComanda`.

- [ ] **Step 4: Implementar el caso de uso**

`application/usecase/command/comandas.go`:

```go
package command

import (
	"context"
	"errors"
	"slices"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	"service-restaurant/internal/modules/produccion/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoEnvio = "no tiene permiso para enviar comandas"
	maxLineasEnvio         = 100

	estadoActivo        = "ACTIVO"
	estadoCuentaAbierta = "ABIERTA"

	CodigoClaveEnUso = "clave_en_uso"
)

func pedidoCerrado() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "pedido_cerrado", "el pedido ya está cerrado o cancelado")
}

func cuentaCerrada() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "cuenta_cerrada", "la cuenta ya no está abierta")
}

// deComanda traduce la clave repetida a un rechazo específico y delega lo demás.
func deComanda(err error, contexto string) (appresult.Result[dtoresult.Comanda], error) {
	if errors.Is(err, outbound.ErrClaveEnUso) {
		return rechazar[dtoresult.Comanda](apperror.New(apperror.KindConflict, CodigoClaveEnUso, "la clave de idempotencia ya fue usada por otra comanda"))
	}
	return deAlmacen[dtoresult.Comanda](contexto, err)
}

// validarLineas comprueba el cuerpo del envío y devuelve, por id de línea, la cantidad pedida y los
// ids ordenados de menor a mayor (el orden de bloqueo).
func validarLineas(lineas []cmd.LineaEnvio) (map[int64]vo.Cantidad, []int64, *apperror.AppError) {
	if len(lineas) == 0 || len(lineas) > maxLineasEnvio {
		return nil, nil, mapper.Validacion("lineas_invalidas", "el envío debe llevar entre 1 y 100 líneas")
	}
	pedidas := make(map[int64]vo.Cantidad, len(lineas))
	ids := make([]int64, 0, len(lineas))
	for _, l := range lineas {
		if l.DetallePedidoID <= 0 {
			return nil, nil, mapper.Validacion("linea_invalida", "el id de una línea no es válido")
		}
		if _, repetida := pedidas[l.DetallePedidoID]; repetida {
			return nil, nil, mapper.Validacion("linea_duplicada", "una línea aparece más de una vez en el envío")
		}
		q, err := vo.NewCantidad(l.Cantidad)
		if err != nil {
			return nil, nil, mapper.Validacion("cantidad_invalida", "la cantidad debe ser mayor que cero y tener hasta 3 decimales")
		}
		pedidas[l.DetallePedidoID] = q
		ids = append(ids, l.DetallePedidoID)
	}
	slices.Sort(ids)
	return pedidas, ids, nil
}

// EnviarComanda envía a una estación las cantidades pendientes de varias líneas de un pedido
// (RES-RF-012). Reenviar con la misma Idempotency-Key devuelve la comanda original: lo resuelve el
// decorador de idempotencia, no este caso de uso.
type EnviarComanda struct{ store outbound.EnvioStore }

func NewEnviarComanda(s outbound.EnvioStore) *EnviarComanda { return &EnviarComanda{s} }

func (uc *EnviarComanda) Execute(ctx context.Context, c cmd.EnviarComanda) (appresult.Result[dtoresult.Comanda], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoEnvio, policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.Comanda](ae)
	}
	pedidas, ids, ae := validarLineas(c.Lineas)
	if ae != nil {
		return rechazar[dtoresult.Comanda](ae)
	}

	// Regla de R1: el pedido se bloquea FOR UPDATE antes que cualquier línea.
	ped, found, err := uc.store.BloquearPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Comanda]("buscar pedido", err)
	}
	if !found {
		return rechazar[dtoresult.Comanda](mapper.NoEncontradoM("pedido"))
	}
	if ae := mapper.Autorizar(c.Actor, ped.SucursalID, mapper.NoEncontradoM("pedido"), policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.Comanda](ae)
	}
	if ped.Cerrado {
		return rechazar[dtoresult.Comanda](pedidoCerrado())
	}

	est, found, err := uc.store.Estacion(ctx, c.EstacionID)
	if err != nil {
		return tecnico[dtoresult.Comanda]("buscar estación", err)
	}
	if !found || est.SucursalID != ped.SucursalID || !est.EsActiva {
		return rechazar[dtoresult.Comanda](mapper.Validacion("estacion_invalida", "la estación no existe, está dada de baja o es de otra sucursal"))
	}

	lineas, err := uc.store.LineasParaEnvio(ctx, c.PedidoID, ped.SucursalID, ids)
	if err != nil {
		return tecnico[dtoresult.Comanda]("buscar líneas", err)
	}
	if len(lineas) != len(ids) {
		return rechazar[dtoresult.Comanda](mapper.Validacion("linea_invalida", "alguna línea no existe o no pertenece al pedido"))
	}

	nuevas := make([]outbound.NuevaComandaLinea, 0, len(lineas))
	for _, l := range lineas {
		if l.Estado != estadoActivo {
			return rechazar[dtoresult.Comanda](apperror.New(apperror.KindConflict, "linea_anulada", "una línea está anulada"))
		}
		if l.CuentaEstado != estadoCuentaAbierta {
			return rechazar[dtoresult.Comanda](cuentaCerrada())
		}
		if l.EstacionID == nil || *l.EstacionID != est.ID {
			return rechazar[dtoresult.Comanda](mapper.Validacion("estacion_no_corresponde", "el producto de una línea no se prepara en esta estación"))
		}
		total, err := vo.CantidadDeAlmacen(l.Cantidad)
		if err != nil {
			return tecnico[dtoresult.Comanda]("leer la cantidad de la línea", err)
		}
		enviado, err := vo.CantidadDeAlmacen(l.Enviado)
		if err != nil {
			return tecnico[dtoresult.Comanda]("leer la cantidad enviada", err)
		}
		cantidad := pedidas[l.ID]
		if cantidad.Excede(policy.Pendiente(total, enviado)) {
			return rechazar[dtoresult.Comanda](apperror.New(apperror.KindConflict, "cantidad_excede_pendiente",
				"la cantidad supera lo que queda pendiente de enviar de una línea"))
		}
		nuevas = append(nuevas, outbound.NuevaComandaLinea{
			DetallePedidoID: l.ID, Cantidad: cantidad.String(), Descripcion: l.Descripcion, Instrucciones: l.Instrucciones,
		})
	}

	com, err := uc.store.CrearComanda(ctx, outbound.NuevaComanda{
		PedidoID: c.PedidoID, SucursalID: ped.SucursalID, EstacionID: est.ID, ClaveIdempotencia: c.Intencion.Clave,
		EnviadoPor: c.Actor.UserID, Lineas: nuevas,
	})
	if err != nil {
		return deComanda(err, "crear comanda")
	}
	return appresult.Ok(com), nil
}
```

- [ ] **Step 5: Formatear, ejecutar y comprobar que pasa**

Run: `gofmt -w internal/modules/produccion && go vet ./internal/modules/produccion/... && go test ./internal/modules/produccion/... -count=1 -v`
Expected: PASS en todos los paquetes (la prueba `TestEnviarComandaRechaza` corre sus 22 subcasos).

- [ ] **Step 6: Commit**

```bash
git add internal/modules/produccion
git commit -m "I3 rebanada 2: caso de uso EnviarComanda con puertos, mapper y pruebas" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Envío de punta a punta — persistencia, HTTP, cableado y pruebas de integración

**Files:**
- Create: `internal/modules/produccion/infrastructure/persistence/internal/cargar/comanda.go`
- Create: `internal/modules/produccion/infrastructure/persistence/write/repository/sql.go`, `comandas_sql.go`
- Create: `internal/modules/produccion/infrastructure/persistence/write/adapter/comandas.go`
- Create: `internal/modules/produccion/infrastructure/config/escritura.go`, `module.go`, `comandas.go`
- Create: `internal/modules/produccion/web/dto/request/comandas.go`, `web/dto/response/comandas.go`
- Create: `internal/modules/produccion/web/mapper/comandas_web_mapper.go`
- Create: `internal/modules/produccion/web/controller/helpers.go`, `comandas_handler.go`, `routes.go`
- Modify: `internal/bootstrap/app.go`, `internal/bootstrap/router.go`
- Create: `tests/integration/cocina_helpers_test.go`, `tests/integration/cocina_comandas_test.go`

**Interfaces:**
- Consumes: Task 3 (`outbound.EnvioStore`, `command.NewEnviarComanda`, `inbound.EnviarComanda`, DTO).
- Produces:
  - `cargar.ScanLinea(fila) (dtoresult.ComandaLinea, error)`, `cargar.ColLinea` (constante SQL), `cargar.Comanda(ctx, db database.DBTX, id int64) (dtoresult.Comanda, bool, error)`, `cargar.Comandas(ctx, db, ids []int64) ([]dtoresult.Comanda, error)`.
  - `repository.ComandaSQLWriter` y `adapter.ComandaWriteAdapter` (`NewComandaWriteAdapter(db database.DBTX)`).
  - `config.Deps{DB *sql.DB; Logger *slog.Logger}`, `config.New(Deps) (*Module, error)`, `(*Module).RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler)`.
  - Ruta `POST /pedidos/{id}/comandas` → 201 con la comanda y `lineas`.
  - Helpers de pruebas (Task 5 y 6 los reutilizan): `estacionSQL`, `productoEnEstacionSQL`, `escenarioCocina`, `nuevoEscenario`, `enviarComanda`, `tokenCocinero`, `comandaResp`, `comandaLineaResp`, `comandasPagina`.

- [ ] **Step 1: Escribir los helpers y las pruebas de integración del envío (fallarán: la ruta no existe)**

`tests/integration/cocina_helpers_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type comandaLineaResp struct {
	ID              string  `json:"id"`
	ComandaID       string  `json:"comanda_id"`
	PedidoID        string  `json:"pedido_id"`
	DetallePedidoID string  `json:"detalle_pedido_id"`
	Cantidad        string  `json:"cantidad"`
	Descripcion     string  `json:"descripcion"`
	Instrucciones   *string `json:"instrucciones"`
	Estado          string  `json:"estado"`
	IniciadoEn      *string `json:"iniciado_en"`
	ListoEn         *string `json:"listo_en"`
	EntregadoEn     *string `json:"entregado_en"`
}

type comandaResp struct {
	ID                string             `json:"id"`
	PedidoID          string             `json:"pedido_id"`
	SucursalID        string             `json:"sucursal_id"`
	EstacionID        string             `json:"estacion_id"`
	MesaID            *string            `json:"mesa_id"`
	ClaveIdempotencia string             `json:"clave_idempotencia"`
	EnviadoPor        string             `json:"enviado_por"`
	Lineas            []comandaLineaResp `json:"lineas"`
}

type comandasPagina struct {
	Items      []comandaResp `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

// estacionSQL crea, como sistema, una estación activa en la sede y devuelve su id.
func estacionSQL(t *testing.T, sede, codigo string) string {
	t.Helper()
	app(t)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.estaciones_preparacion(sucursal_id,codigo,nombre)
		SELECT s.id,'%s','Estación %s' FROM sch_restaurante.sucursales s WHERE s.codigo='%s'`, codigo, codigo, sede))
	return scalarStr(t, `SELECT e.id FROM sch_restaurante.estaciones_preparacion e
		JOIN sch_restaurante.sucursales s ON s.id=e.sucursal_id WHERE s.codigo=$1 AND e.codigo=$2`, sede, codigo)
}

// productoEnEstacionSQL crea un producto disponible en la sede y lo asigna a la estación en esa sede.
func productoEnEstacionSQL(t *testing.T, sede, estacion, precio string) string {
	t.Helper()
	prod := productoVentaSQL(t, sede, "PREPARADO", precio, true)
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos_sucursal SET estacion_id=%s
		WHERE producto_id=%s AND sucursal_id=(SELECT id FROM sch_restaurante.sucursales WHERE codigo='%s')`, estacion, prod, sede))
	return prod
}

// tokenCocinero: opera la cocina y ve pedidos en PRINCIPAL y las sedes dadas.
func tokenCocinero(t *testing.T, sedes ...string) string {
	return tokenPedidos(t, "coc.cocinero", "T_COC_COCINA", []string{"COCINA_OPERAR", "PEDIDO_VER"}, sedes...)
}

// escenarioCocina es un pedido LLEVAR de PRINCIPAL con una línea de la cantidad dada, cuyo producto
// se prepara en una estación propia de la prueba.
type escenarioCocina struct {
	mesero   string // token con PEDIDO_VER, PEDIDO_CREAR y PEDIDO_EDITAR
	cocinero string // token con COCINA_OPERAR y PEDIDO_VER
	pedido   pedidoResp
	cuenta   string
	estacion string
	producto string
	linea    lineaResp
}

func nuevoEscenario(t *testing.T, cantidad string) escenarioCocina {
	t.Helper()
	mesero := meseroPedidos(t)
	est := estacionSQL(t, "PRINCIPAL", "EC"+sufijo())
	prod := productoEnEstacionSQL(t, "PRINCIPAL", est, "10.00")
	p := abrirLlevar(t, mesero, "PRINCIPAL")
	r := agregarConsumo(t, mesero, p.ID, p.Cuentas[0].ID, prod, cantidad, `,"instrucciones":"sin ají"`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	return escenarioCocina{mesero: mesero, cocinero: tokenCocinero(t), pedido: p, cuenta: p.Cuentas[0].ID,
		estacion: est, producto: prod, linea: l}
}

// enviarComanda envía a la estación pares {id de línea, cantidad} y devuelve la respuesta cruda.
func enviarComanda(t *testing.T, tk, pedido, estacion string, lineas ...[2]string) resp {
	t.Helper()
	items := make([]string, len(lineas))
	for i, l := range lineas {
		items[i] = fmt.Sprintf(`{"detalle_pedido_id":%q,"cantidad":%q}`, l[0], l[1])
	}
	cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[%s]}`, estacion, strings.Join(items, ","))
	return post(t, "/api/v1/pedidos/"+pedido+"/comandas", cuerpo, tk)
}

// enviarOK envía y decodifica la comanda creada.
func enviarOK(t *testing.T, tk, pedido, estacion string, lineas ...[2]string) comandaResp {
	t.Helper()
	r := enviarComanda(t, tk, pedido, estacion, lineas...)
	esperar(t, r, http.StatusCreated, "")
	var c comandaResp
	r.json(t, &c)
	return c
}
```

`tests/integration/cocina_comandas_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// RES-CP-012: se envían las cantidades pendientes sin reenviar lo ya despachado.
func TestComandaSeEnviaEnTandasSinReenviar(t *testing.T) {
	e := nuevoEscenario(t, "3")

	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	if c.PedidoID != e.pedido.ID || c.EstacionID != e.estacion || c.SucursalID != sedeID(t, "PRINCIPAL") || c.MesaID != nil ||
		len(c.Lineas) != 1 {
		t.Fatalf("comanda: %+v", c)
	}
	l := c.Lineas[0]
	if l.DetallePedidoID != e.linea.ID || l.Cantidad != "2.000" || l.Estado != "PENDIENTE" ||
		l.Descripcion != e.linea.Descripcion || l.Instrucciones == nil || *l.Instrucciones != "sin ají" {
		t.Fatalf("línea de comanda: %+v", l)
	}

	esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"}),
		http.StatusConflict, "cantidad_excede_pendiente")
	enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "0.001"}),
		http.StatusConflict, "cantidad_excede_pendiente")
}

func TestComandaRepetidaConLaMismaClaveDevuelveLaOriginal(t *testing.T) {
	e := nuevoEscenario(t, "3")
	clave := nuevaClave()
	cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"1"}]}`, e.estacion, e.linea.ID)
	h := map[string]string{"Idempotency-Key": clave}
	ruta := "/api/v1/pedidos/" + e.pedido.ID + "/comandas"

	r1 := callH(t, "POST", ruta, cuerpo, e.mesero, h)
	esperar(t, r1, http.StatusCreated, "")
	var c1, c2 comandaResp
	r1.json(t, &c1)
	r2 := callH(t, "POST", ruta, cuerpo, e.mesero, h)
	esperar(t, r2, http.StatusCreated, "")
	r2.json(t, &c2)
	if c1.ID != c2.ID {
		t.Fatalf("la repetición debe devolver la comanda original: %s y %s", c1.ID, c2.ID)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comandas WHERE pedido_id=$1::text::bigint`, e.pedido.ID); n != 1 {
		t.Fatalf("la repetición no debe crear otra comanda (hay %d)", n)
	}

	otro := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"2"}]}`, e.estacion, e.linea.ID)
	esperar(t, callH(t, "POST", ruta, otro, e.mesero, h), http.StatusConflict, "idempotency_key_reutilizada")
}

func TestComandaRechazaEnviosInvalidos(t *testing.T) {
	e := nuevoEscenario(t, "2")
	otraEstacion := estacionSQL(t, "PRINCIPAL", "EO"+sufijo())
	sinEstacion := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "5.00", true)
	r := agregarConsumo(t, e.mesero, e.pedido.ID, e.cuenta, sinEstacion, "1", "")
	esperar(t, r, http.StatusCreated, "")
	var lSin lineaResp
	r.json(t, &lSin)
	ajeno, ajenaCuenta := pedidoSQL(t, "PRINCIPAL")
	rA := agregarConsumo(t, e.mesero, ajeno, ajenaCuenta, e.producto, "1", "")
	esperar(t, rA, http.StatusCreated, "")
	var lAjena lineaResp
	rA.json(t, &lAjena)

	casos := []struct {
		nombre   string
		estacion string
		lineas   [][2]string
		status   int
		code     string
	}{
		{"estación que no corresponde al producto", otraEstacion, [][2]string{{e.linea.ID, "1"}}, 422, "estacion_no_corresponde"},
		{"producto sin estación", e.estacion, [][2]string{{lSin.ID, "1"}}, 422, "estacion_no_corresponde"},
		{"línea de otro pedido", e.estacion, [][2]string{{lAjena.ID, "1"}}, 422, "linea_invalida"},
		{"línea inexistente", e.estacion, [][2]string{{"999999999", "1"}}, 422, "linea_invalida"},
		{"línea repetida", e.estacion, [][2]string{{e.linea.ID, "1"}, {e.linea.ID, "1"}}, 422, "linea_duplicada"},
		{"cantidad cero", e.estacion, [][2]string{{e.linea.ID, "0"}}, 422, "cantidad_invalida"},
		{"cantidad mayor que la línea", e.estacion, [][2]string{{e.linea.ID, "2.001"}}, 409, "cantidad_excede_pendiente"},
		{"estación inexistente", "999999999", [][2]string{{e.linea.ID, "1"}}, 422, "estacion_invalida"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, tc.estacion, tc.lineas...), tc.status, tc.code)
		})
	}
	esperar(t, post(t, "/api/v1/pedidos/"+e.pedido.ID+"/comandas", fmt.Sprintf(`{"estacion_id":%q,"lineas":[]}`, e.estacion), e.mesero),
		http.StatusUnprocessableEntity, "lineas_invalidas")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comandas WHERE pedido_id=$1::text::bigint`, e.pedido.ID); n != 0 {
		t.Fatalf("ningún envío rechazado debe dejar comandas (hay %d)", n)
	}
}

func TestComandaNoSeEnviaDeUnaLineaAnulada(t *testing.T) {
	e := nuevoEscenario(t, "2")
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.detalles_pedido SET estado='ANULADO', anulado_en=now(),
		anulado_por=%d, motivo_anulacion='prueba' WHERE id=%s`, adminID(t), e.linea.ID))
	esperar(t, enviarComanda(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}),
		http.StatusConflict, "linea_anulada")
}

func TestComandaPermisosYSedes(t *testing.T) {
	e := nuevoEscenario(t, "2")

	t.Run("quien solo ve pedidos no envía", func(t *testing.T) {
		esperar(t, enviarComanda(t, lectorPedidos(t), e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusForbidden, "sin_permiso")
	})
	t.Run("el cocinero no crea envíos", func(t *testing.T) {
		esperar(t, enviarComanda(t, e.cocinero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusForbidden, "sin_permiso")
	})
	t.Run("sin sesión", func(t *testing.T) {
		esperar(t, enviarComanda(t, "", e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"}), http.StatusUnauthorized, "")
	})
	t.Run("pedido inexistente", func(t *testing.T) {
		esperar(t, enviarComanda(t, e.mesero, "999999999", e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusNotFound, "no_encontrado")
	})
	t.Run("pedido de una sede que no ve", func(t *testing.T) {
		crearSedeSQL(t, "COC2")
		ajeno, _ := pedidoSQL(t, "COC2")
		esperar(t, enviarComanda(t, e.mesero, ajeno, e.estacion, [2]string{e.linea.ID, "1"}),
			http.StatusNotFound, "no_encontrado")
	})
}
```

El caso «ve la sede pero no puede crear» (403 `sede_no_autorizada`) no se prueba por HTTP porque `tokenPedidos` asigna un mismo rol a todas las sedes del usuario; lo cubre la prueba unitaria `ve la sede pero no puede crear` de la Task 3.

- [ ] **Step 2: Ejecutar y comprobar que fallan**

Run: `go test ./tests/integration/ -run "TestComanda" -count=1`
Expected: FAIL (404 en `POST /pedidos/{id}/comandas`: la ruta no existe). Si falla por compilación, revisar los helpers.

- [ ] **Step 3: Implementar la carga de comandas (lectura y escritura comparten esta consulta)**

`internal/modules/produccion/infrastructure/persistence/internal/cargar/comanda.go`:

```go
// Package cargar arma comandas con sus líneas con consultas que comparten el lado de lectura y el
// de escritura. Vive bajo persistence/internal: solo la persistencia del módulo puede importarlo.
package cargar

import (
	"context"
	"database/sql"
	"time"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/platform/database"
)

// Las cantidades son numeric: se leen como texto para no perder exactitud.
const (
	colComanda = `c.id, c.pedido_id, c.sucursal_id, c.estacion_id, p.mesa_id, c.clave_idempotencia::text, c.enviado_en,
		c.enviado_por, c.notas, c.creado_en, c.creado_por, c.actualizado_en, c.actualizado_por`

	// ColLinea son las columnas de comanda_detalles que lee ScanLinea.
	ColLinea = `id, comanda_id, pedido_id, detalle_pedido_id, cantidad::text, descripcion, instrucciones, estado,
		iniciado_en, listo_en, entregado_en, motivo_cancelacion, creado_en, creado_por, actualizado_en, actualizado_por`
)

type fila interface{ Scan(...any) error }

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func fechaNula(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

// ScanLinea lee una fila con las columnas ColLinea.
func ScanLinea(s fila) (dtoresult.ComandaLinea, error) {
	var l dtoresult.ComandaLinea
	var instr, motivo sql.NullString
	var ini, listo, entrega sql.NullTime
	err := s.Scan(&l.ID, &l.ComandaID, &l.PedidoID, &l.DetallePedidoID, &l.Cantidad, &l.Descripcion, &instr, &l.Estado,
		&ini, &listo, &entrega, &motivo, &l.CreadoEn, &l.CreadoPor, &l.ActualizadoEn, &l.ActualizadoPor)
	l.Instrucciones, l.MotivoCancelacion = textoNulo(instr), textoNulo(motivo)
	l.IniciadoEn, l.ListoEn, l.EntregadoEn = fechaNula(ini), fechaNula(listo), fechaNula(entrega)
	return l, err
}

// Comandas lee las comandas con esos ids, ordenadas por id, cada una con todas sus líneas. Cada
// consulta se agota antes de lanzar la siguiente: sobre una transacción no puede haber dos
// resultados abiertos a la vez.
func Comandas(ctx context.Context, db database.DBTX, ids []int64) ([]dtoresult.Comanda, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+colComanda+` FROM sch_restaurante.comandas c
		JOIN sch_restaurante.pedidos p ON p.id = c.pedido_id
		WHERE c.id = ANY($1::bigint[]) ORDER BY c.id`, ids)
	if err != nil {
		return nil, err
	}
	out := []dtoresult.Comanda{}
	for rows.Next() {
		var c dtoresult.Comanda
		var mesa sql.NullInt64
		var notas sql.NullString
		if err := rows.Scan(&c.ID, &c.PedidoID, &c.SucursalID, &c.EstacionID, &mesa, &c.ClaveIdempotencia, &c.EnviadoEn,
			&c.EnviadoPor, &notas, &c.CreadoEn, &c.CreadoPor, &c.ActualizadoEn, &c.ActualizadoPor); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if mesa.Valid {
			m := mesa.Int64
			c.MesaID = &m
		}
		c.Notas = textoNulo(notas)
		c.Lineas = []dtoresult.ComandaLinea{}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	if len(out) == 0 {
		return out, nil
	}

	posicion := make(map[int64]int, len(out))
	for i, c := range out {
		posicion[c.ID] = i
	}
	lrows, err := db.QueryContext(ctx, `SELECT `+ColLinea+` FROM sch_restaurante.comanda_detalles
		WHERE comanda_id = ANY($1::bigint[]) ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	defer lrows.Close()
	for lrows.Next() {
		l, err := ScanLinea(lrows)
		if err != nil {
			return nil, err
		}
		i := posicion[l.ComandaID]
		out[i].Lineas = append(out[i].Lineas, l)
	}
	return out, lrows.Err()
}

// Comanda lee una comanda con sus líneas; found=false si no existe.
func Comanda(ctx context.Context, db database.DBTX, id int64) (dtoresult.Comanda, bool, error) {
	cs, err := Comandas(ctx, db, []int64{id})
	if err != nil || len(cs) == 0 {
		return dtoresult.Comanda{}, false, err
	}
	return cs[0], true, nil
}
```
- [ ] **Step 4: Implementar el SQL de escritura y su adaptador**

`internal/modules/produccion/infrastructure/persistence/write/repository/sql.go`:

```go
// Package repository contiene el SQL parametrizado de escritura del módulo produccion.
// Recibe un database.DBTX: en operaciones de escritura es el *sql.Tx de la unidad de trabajo.
package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"service-restaurant/internal/modules/produccion/application/port/outbound"
)

const codigoUniqueViolation = "23505"

// Translate convierte los errores de PostgreSQL conocidos en los errores de puerto que los casos
// de uso entienden como rechazos esperados. Cualquier otro error pasa sin cambios.
func Translate(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case codigoUniqueViolation:
		return fmt.Errorf("%w (%s)", outbound.ErrDuplicado, pg.ConstraintName)
	case "23503": // foreign_key_violation
		return fmt.Errorf("%w (%s)", outbound.ErrReferenciaInvalida, pg.ConstraintName)
	case "23514": // check_violation
		return &outbound.ErrReglaDatos{Mensaje: "el valor no cumple una restricción de datos"}
	case "P0001": // RAISE EXCEPTION de las reglas de la V4: mensajes curados del esquema
		return &outbound.ErrReglaDatos{Mensaje: pg.Message}
	}
	return err
}

func notFound(err error) (bool, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return false, err
}

// restriccionUnica devuelve el nombre de la restricción violada si err es un unique_violation de
// PostgreSQL; si no, una cadena vacía.
func restriccionUnica(err error) string {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == codigoUniqueViolation {
		return pg.ConstraintName
	}
	return ""
}
```

`internal/modules/produccion/infrastructure/persistence/write/repository/comandas_sql.go`:

```go
package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/platform/database"
)

// restriccionClaveComanda es la restricción única de comandas.clave_idempotencia.
const restriccionClaveComanda = "comandas_clave_idempotencia_key"

// traducirComanda distingue la clave repetida por el nombre de la restricción.
func traducirComanda(err error) error {
	if restriccionUnica(err) == restriccionClaveComanda {
		return outbound.ErrClaveEnUso
	}
	return Translate(err)
}

// ComandaSQLWriter opera sobre comandas y comanda_detalles, y lee pedidos, líneas y estaciones.
type ComandaSQLWriter struct{ db database.DBTX }

func NewComandaSQLWriter(db database.DBTX) *ComandaSQLWriter { return &ComandaSQLWriter{db} }

// GetPedidoForUpdate bloquea el pedido hasta el final de la transacción.
func (r *ComandaSQLWriter) GetPedidoForUpdate(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	var p outbound.PedidoRef
	err := r.db.QueryRowContext(ctx, `SELECT id, sucursal_id, cerrado_en IS NOT NULL
		FROM sch_restaurante.pedidos WHERE id = $1 FOR UPDATE`, id).Scan(&p.ID, &p.SucursalID, &p.Cerrado)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.PedidoRef{}, false, err
	}
	return p, true, nil
}

// GetEstacion lee la estación en modo compartido: no puede darse de baja hasta que termine el comando.
func (r *ComandaSQLWriter) GetEstacion(ctx context.Context, id int64) (outbound.EstacionRef, bool, error) {
	var e outbound.EstacionRef
	err := r.db.QueryRowContext(ctx, `SELECT id, sucursal_id, es_activo
		FROM sch_restaurante.estaciones_preparacion WHERE id = $1 FOR SHARE`, id).Scan(&e.ID, &e.SucursalID, &e.EsActiva)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.EstacionRef{}, false, err
	}
	return e, true, nil
}

// GetLineasParaEnvio bloquea las líneas del pedido con esos ids, por id ascendente. Enviado suma lo
// ya enviado de cada línea sin contar las líneas de comanda canceladas. La estación sale de la fila
// activa del producto en la sede del pedido. El pedido ya está bloqueado: dos envíos del mismo
// pedido nunca llegan aquí a la vez.
func (r *ComandaSQLWriter) GetLineasParaEnvio(ctx context.Context, pedidoID, sucursalID int64, ids []int64) ([]outbound.LineaPedidoRef, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.cuenta_id, c.estado, d.estado, d.cantidad::text, d.descripcion, d.notas_preparacion, ps.estacion_id,
		       COALESCE((SELECT sum(cd.cantidad) FROM sch_restaurante.comanda_detalles cd
		                  WHERE cd.detalle_pedido_id = d.id AND cd.estado <> 'CANCELADO'), 0)::text
		FROM sch_restaurante.detalles_pedido d
		JOIN sch_restaurante.cuentas c ON c.id = d.cuenta_id
		LEFT JOIN sch_restaurante.productos_sucursal ps
		       ON ps.producto_id = d.producto_id AND ps.sucursal_id = $3 AND ps.es_activo
		WHERE d.pedido_id = $1 AND d.id = ANY($2::bigint[])
		ORDER BY d.id
		FOR UPDATE OF d`, pedidoID, ids, sucursalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outbound.LineaPedidoRef
	for rows.Next() {
		var l outbound.LineaPedidoRef
		var instr sql.NullString
		var estacion sql.NullInt64
		if err := rows.Scan(&l.ID, &l.CuentaID, &l.CuentaEstado, &l.Estado, &l.Cantidad, &l.Descripcion, &instr, &estacion, &l.Enviado); err != nil {
			return nil, err
		}
		if instr.Valid {
			s := instr.String
			l.Instrucciones = &s
		}
		if estacion.Valid {
			n := estacion.Int64
			l.EstacionID = &n
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// InsertComanda crea la comanda y sus líneas en la transacción en curso y la devuelve armada.
// creado_por, actualizado_por y las marcas de tiempo los fija el trigger de metadatos.
func (r *ComandaSQLWriter) InsertComanda(ctx context.Context, n outbound.NuevaComanda) (dtoresult.Comanda, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.comandas (pedido_id, sucursal_id, estacion_id, clave_idempotencia, enviado_por)
		VALUES ($1, $2, $3, $4::uuid, $5) RETURNING id`,
		n.PedidoID, n.SucursalID, n.EstacionID, n.ClaveIdempotencia, n.EnviadoPor).Scan(&id)
	if err != nil {
		return dtoresult.Comanda{}, traducirComanda(err)
	}
	for _, l := range n.Lineas {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO sch_restaurante.comanda_detalles (comanda_id, pedido_id, detalle_pedido_id, cantidad, descripcion, instrucciones)
			VALUES ($1, $2, $3, $4::text::numeric, $5, $6)`,
			id, n.PedidoID, l.DetallePedidoID, l.Cantidad, l.Descripcion, l.Instrucciones); err != nil {
			return dtoresult.Comanda{}, traducirComanda(err)
		}
	}
	c, found, err := cargar.Comanda(ctx, r.db, id)
	if err != nil {
		return dtoresult.Comanda{}, err
	}
	if !found {
		return dtoresult.Comanda{}, sql.ErrNoRows
	}
	return c, nil
}
```

`internal/modules/produccion/infrastructure/persistence/write/adapter/comandas.go`:

```go
// Package adapter implementa los puertos de salida del módulo produccion sobre SQL.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// ComandaWriteAdapter implementa outbound.EnvioStore.
type ComandaWriteAdapter struct{ repo *repository.ComandaSQLWriter }

func NewComandaWriteAdapter(db database.DBTX) *ComandaWriteAdapter {
	return &ComandaWriteAdapter{repo: repository.NewComandaSQLWriter(db)}
}

var _ outbound.EnvioStore = (*ComandaWriteAdapter)(nil)

func (a *ComandaWriteAdapter) BloquearPedido(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	return a.repo.GetPedidoForUpdate(ctx, id)
}
func (a *ComandaWriteAdapter) Estacion(ctx context.Context, id int64) (outbound.EstacionRef, bool, error) {
	return a.repo.GetEstacion(ctx, id)
}
func (a *ComandaWriteAdapter) LineasParaEnvio(ctx context.Context, pedidoID, sucursalID int64, ids []int64) ([]outbound.LineaPedidoRef, error) {
	return a.repo.GetLineasParaEnvio(ctx, pedidoID, sucursalID, ids)
}
func (a *ComandaWriteAdapter) CrearComanda(ctx context.Context, n outbound.NuevaComanda) (dtoresult.Comanda, error) {
	return a.repo.InsertComanda(ctx, n)
}
```

- [ ] **Step 5: Capa web (cuerpos, mapper, helpers, handler, rutas)**

`web/dto/request/comandas.go`:

```go
// Package request define los cuerpos HTTP del módulo produccion.
package request

// ComandaLinea: id de la línea del pedido y cantidad a enviar (decimal en cadena).
type ComandaLinea struct {
	DetallePedidoID string `json:"detalle_pedido_id"`
	Cantidad        string `json:"cantidad"`
}

// ComandaAlta: los ids viajan como cadenas (contrato). Un campo desconocido responde 400.
type ComandaAlta struct {
	EstacionID string         `json:"estacion_id"`
	Lineas     []ComandaLinea `json:"lineas"`
}
```

`web/dto/response/comandas.go`:

```go
// Package response define los cuerpos de respuesta HTTP del módulo produccion.
package response

import "time"

type ComandaLinea struct {
	ID                string     `json:"id"`
	ComandaID         string     `json:"comanda_id"`
	PedidoID          string     `json:"pedido_id"`
	DetallePedidoID   string     `json:"detalle_pedido_id"`
	Cantidad          string     `json:"cantidad"`
	Descripcion       string     `json:"descripcion"`
	Instrucciones     *string    `json:"instrucciones"`
	Estado            string     `json:"estado"`
	IniciadoEn        *time.Time `json:"iniciado_en"`
	ListoEn           *time.Time `json:"listo_en"`
	EntregadoEn       *time.Time `json:"entregado_en"`
	MotivoCancelacion *string    `json:"motivo_cancelacion"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type Comanda struct {
	ID                string         `json:"id"`
	PedidoID          string         `json:"pedido_id"`
	SucursalID        string         `json:"sucursal_id"`
	EstacionID        string         `json:"estacion_id"`
	MesaID            *string        `json:"mesa_id"`
	ClaveIdempotencia string         `json:"clave_idempotencia"`
	EnviadoEn         time.Time      `json:"enviado_en"`
	EnviadoPor        string         `json:"enviado_por"`
	Notas             *string        `json:"notas"`
	Lineas            []ComandaLinea `json:"lineas"`
	CreadoEn          time.Time      `json:"creado_en"`
	CreadoPor         string         `json:"creado_por"`
	ActualizadoEn     time.Time      `json:"actualizado_en"`
	ActualizadoPor    string         `json:"actualizado_por"`
}

type ComandasPagina struct {
	Items      []Comanda `json:"items"`
	NextCursor *string   `json:"nextCursor"`
}
```

`web/mapper/comandas_web_mapper.go`:

```go
// Package mapper convierte resultados de aplicación en respuestas HTTP.
package mapper

import (
	"strconv"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/web/dto/response"
)

func id(n int64) string { return strconv.FormatInt(n, 10) }

func idOpcional(n *int64) *string {
	if n == nil {
		return nil
	}
	v := id(*n)
	return &v
}

func cursor(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}

func ToLinea(l dtoresult.ComandaLinea) response.ComandaLinea {
	return response.ComandaLinea{
		ID: id(l.ID), ComandaID: id(l.ComandaID), PedidoID: id(l.PedidoID), DetallePedidoID: id(l.DetallePedidoID),
		Cantidad: l.Cantidad, Descripcion: l.Descripcion, Instrucciones: l.Instrucciones, Estado: l.Estado,
		IniciadoEn: l.IniciadoEn, ListoEn: l.ListoEn, EntregadoEn: l.EntregadoEn, MotivoCancelacion: l.MotivoCancelacion,
		CreadoEn: l.CreadoEn, CreadoPor: id(l.CreadoPor), ActualizadoEn: l.ActualizadoEn, ActualizadoPor: id(l.ActualizadoPor),
	}
}

func ToComanda(c dtoresult.Comanda) response.Comanda {
	lineas := make([]response.ComandaLinea, len(c.Lineas))
	for i, l := range c.Lineas {
		lineas[i] = ToLinea(l)
	}
	return response.Comanda{
		ID: id(c.ID), PedidoID: id(c.PedidoID), SucursalID: id(c.SucursalID), EstacionID: id(c.EstacionID),
		MesaID: idOpcional(c.MesaID), ClaveIdempotencia: c.ClaveIdempotencia, EnviadoEn: c.EnviadoEn,
		EnviadoPor: id(c.EnviadoPor), Notas: c.Notas, Lineas: lineas,
		CreadoEn: c.CreadoEn, CreadoPor: id(c.CreadoPor), ActualizadoEn: c.ActualizadoEn, ActualizadoPor: id(c.ActualizadoPor),
	}
}

func ToComandasPagina(p dtoresult.PaginaComandas) response.ComandasPagina {
	items := make([]response.Comanda, len(p.Items))
	for i, c := range p.Items {
		items[i] = ToComanda(c)
	}
	return response.ComandasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
```

`web/controller/helpers.go`:

```go
// Package controller expone el módulo produccion por HTTP.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/platform/idempotency"
	"service-restaurant/internal/platform/web/middleware"
	"service-restaurant/internal/platform/web/response"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

var errResultadoInvalido = errors.New("el caso de uso devolvió un Result inválido")

// responder traduce el Result de un caso de uso a la respuesta HTTP.
func responder[T, R any](w http.ResponseWriter, res appresult.Result[T], err error, status int, f func(T) R) error {
	if err != nil {
		return err
	}
	if !res.IsValid() {
		return errResultadoInvalido
	}
	if res.Err() != nil {
		return res.Err()
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteJSON(w, status, f(res.Value()))
	return nil
}

func actorDe(r *http.Request) (appctx.Actor, error) {
	a, ok := middleware.ActorFrom(r.Context())
	if !ok {
		return appctx.Actor{}, mapper.NoAutenticado()
	}
	return a, nil
}

// pathID lee el id numérico positivo de la ruta ({id}).
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.New(apperror.KindBadRequest, "id_invalido", "el id de la ruta no es válido")
	}
	return id, nil
}

// idCadena convierte un id en cadena (contrato) a int64 positivo. Solo acepta el decimal canónico:
// sin "+" ni ceros a la izquierda.
func idCadena(valor, campo string) (int64, error) {
	n, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != valor {
		return 0, apperror.New(apperror.KindBadRequest, campo+"_invalido", campo+" debe ser un entero positivo en cadena")
	}
	return n, nil
}

// intencion arma la intención idempotente: exige Idempotency-Key (UUID) y calcula la huella sobre
// la operación y el cuerpo canónico (el DTO tipado vuelto a serializar).
func intencion(r *http.Request, operacion string, cuerpo any) (cmd.Intencion, error) {
	clave, err := idempotency.ParseKey(r.Header.Get("Idempotency-Key"))
	if err != nil {
		return cmd.Intencion{}, err
	}
	var raw []byte
	if cuerpo != nil {
		if raw, err = json.Marshal(cuerpo); err != nil {
			return cmd.Intencion{}, err
		}
	}
	return cmd.Intencion{Clave: clave, Operacion: operacion, Huella: idempotency.Fingerprint(operacion, raw)}, nil
}

// consulta analiza la query string de forma estricta: un cursor corrupto no puede tratarse como
// "primera página".
func consulta(r *http.Request) (url.Values, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, apperror.New(apperror.KindBadRequest, "query_invalida", "los parámetros de la consulta están mal codificados")
	}
	return q, nil
}

// enumDe lee un filtro opcional que debe pertenecer a un conjunto cerrado; ausente → "".
func enumDe(q url.Values, campo string, validos ...string) (string, error) {
	v := q.Get(campo)
	if v == "" {
		return "", nil
	}
	for _, ok := range validos {
		if v == ok {
			return v, nil
		}
	}
	return "", apperror.New(apperror.KindBadRequest, campo+"_invalido", campo+" no es un valor válido")
}

// limiteDe lee limit (1..100, por defecto 25).
func limiteDe(q url.Values) (int, error) {
	v := q.Get("limit")
	if v == "" {
		return 25, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 100 {
		return 0, apperror.New(apperror.KindBadRequest, "limit_invalido", "limit debe estar entre 1 y 100")
	}
	return n, nil
}

// idOpcionalDe lee un filtro opcional de id (sucursal_id, estacion_id).
func idOpcionalDe(q url.Values, campo string) (*int64, error) {
	v := q.Get(campo)
	if v == "" {
		return nil, nil
	}
	n, err := idCadena(v, campo)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
```

(`enumDe`, `limiteDe`, `idOpcionalDe` y `consulta` se usan desde la Task 6; `go vet` no se queja de funciones sin usar.)

`web/controller/comandas_handler.go`:

```go
package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/produccion/application/dto/command"
	"service-restaurant/internal/modules/produccion/application/port/inbound"
	"service-restaurant/internal/modules/produccion/web/dto/request"
	webmapper "service-restaurant/internal/modules/produccion/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

// ComandasHandler: envío a cocina, cola de preparación y estados de las líneas (RES-RF-012).
type ComandasHandler struct {
	enviar inbound.EnviarComanda
}

func NewComandasHandler(e inbound.EnviarComanda) *ComandasHandler { return &ComandasHandler{e} }

// Enviar: POST /pedidos/{id}/comandas (Idempotency-Key obligatoria)
func (h *ComandasHandler) Enviar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ComandaAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	estacion, err := idCadena(body.EstacionID, "estacion_id")
	if err != nil {
		return err
	}
	lineas := make([]command.LineaEnvio, len(body.Lineas))
	for i, l := range body.Lineas {
		det, err := idCadena(l.DetallePedidoID, "detalle_pedido_id")
		if err != nil {
			return err
		}
		lineas[i] = command.LineaEnvio{DetallePedidoID: det, Cantidad: l.Cantidad}
	}
	in, err := intencion(r, fmt.Sprintf("POST /pedidos/%d/comandas", pedido), body)
	if err != nil {
		return err
	}
	res, err := h.enviar.Execute(r.Context(), command.EnviarComanda{
		Intencion: in, Actor: actor, PedidoID: pedido, EstacionID: estacion, Lineas: lineas,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToComanda)
}
```

`web/controller/routes.go`:

```go
package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterRoutes monta las rutas de cocina (todas exigen autenticación; el permiso fino lo verifica
// cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, comandas *ComandasHandler, protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("POST /pedidos/{id}/comandas", comandas.Enviar)
}
```

- [ ] **Step 6: Composición del módulo y cableado en el arranque**

`infrastructure/config/escritura.go` (copia exacta del decorador de `atencion`, con las rutas de import de `produccion`):

```go
// Package config compone el módulo produccion: conecta los puertos de aplicación con sus
// adaptadores concretos y decora los casos de uso con transacciones.
package config

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	"service-restaurant/internal/platform/database"
	"service-restaurant/internal/platform/idempotency"
	"service-restaurant/internal/platform/transaction"
	appresult "service-restaurant/internal/shared/kernel/result"
)

var errInvalidResult = errors.New("el caso de uso devolvió un Result inválido")

// txQuerier es lo que los repositorios necesitan del *sql.Tx; la aplicación nunca lo ve.
type txQuerier = database.DBTX

// ejecutor es el caso de uso que se decora.
type ejecutor[C, T any] interface {
	Execute(context.Context, C) (appresult.Result[T], error)
}

// escritura decora un caso de uso de escritura: lo ejecuta en una transacción con el actor
// autenticado fijado (app.usuario_id) y, si la solicitud trae clave de idempotencia, registra la
// clave en esa misma transacción. Solo un Result exitoso se confirma; un rechazo o un error
// revierte todo y no deja rastro, de modo que el reintento se evalúa de nuevo.
type escritura[C, T any] struct {
	db     *sql.DB
	logger *slog.Logger
	status int // estado HTTP que se recuerda para repeticiones idempotentes
	build  func(txQuerier) ejecutor[C, T]
	meta   func(C) (actorID int64, intencion cmd.Intencion)
}

func (e escritura[C, T]) Execute(ctx context.Context, c C) (appresult.Result[T], error) {
	actorID, in := e.meta(c)
	trabajo := func(ctx context.Context, tx *sql.Tx) (appresult.Result[T], error) {
		return e.build(tx).Execute(ctx, c)
	}

	if in.Clave == "" {
		res, err := transaction.Run(ctx, e.db, nil, e.logger, trabajo, appresult.CanCommit[T], transaction.WithActor(actorID))
		return validar(res, err)
	}
	out, err := idempotency.Run(ctx, e.db, e.logger, idempotency.Request{
		Key: in.Clave, UsuarioID: actorID, Operacion: in.Operacion, Huella: in.Huella,
		Status: e.status, Now: time.Now().UTC(),
	}, trabajo, transaction.WithActor(actorID))
	if err != nil {
		return appresult.Result[T]{}, err
	}
	return validar(out.Result, nil)
}

func validar[T any](res appresult.Result[T], err error) (appresult.Result[T], error) {
	if err != nil {
		return appresult.Result[T]{}, err
	}
	if !res.IsValid() {
		return appresult.Result[T]{}, errInvalidResult
	}
	return res, nil
}
```

`infrastructure/config/comandas.go`:

```go
package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/inbound"
	"service-restaurant/internal/modules/produccion/application/usecase/command"
	writeadapter "service-restaurant/internal/modules/produccion/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/produccion/web/controller"
)

// armarComandas compone los casos de uso de cocina, ya decorados con transacción e idempotencia.
func armarComandas(db *sql.DB, logger *slog.Logger) *controller.ComandasHandler {
	enviar := escritura[cmd.EnviarComanda, dtoresult.Comanda]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.EnviarComanda, dtoresult.Comanda] {
			return command.NewEnviarComanda(writeadapter.NewComandaWriteAdapter(q))
		},
		meta: func(c cmd.EnviarComanda) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	return controller.NewComandasHandler(inbound.EnviarComanda(enviar))
}
```

`infrastructure/config/module.go`:

```go
package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	"service-restaurant/internal/modules/produccion/web/controller"
)

// Deps son las dependencias técnicas del módulo.
type Deps struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// Module es el módulo produccion ya ensamblado.
type Module struct {
	deps     Deps
	comandas *controller.ComandasHandler
}

// New ensambla el módulo.
func New(d Deps) (*Module, error) {
	return &Module{deps: d, comandas: armarComandas(d.DB, d.Logger)}, nil
}

// RegisterRoutes monta las rutas del módulo sobre mux (relativas a /api/v1). protect exige un actor
// autenticado; lo aporta el módulo identidad a través del bootstrap.
func (m *Module) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	controller.RegisterRoutes(mux, m.comandas, protect, m.deps.Logger)
}
```

Modificar `internal/bootstrap/app.go`:

1. En el bloque de imports, tras la línea de `identidad`, añadir:
   `produccion "service-restaurant/internal/modules/produccion/infrastructure/config"`
2. En `type App struct`, tras `catalogo  *catalogo.Module`, añadir `produccion *produccion.Module`.
3. En `New`, después del bloque que crea `cat` y antes de `a := &App{...}`, añadir:

```go
	prd, err := produccion.New(produccion.Deps{DB: db, Logger: logger})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
```

4. Cambiar la construcción: `a := &App{cfg: cfg, logger: logger, db: db, identidad: mod, atencion: atn, catalogo: cat, produccion: prd}`.

Modificar `internal/bootstrap/router.go`: tras `a.catalogo.RegisterRoutes(api, protect)` añadir `a.produccion.RegisterRoutes(api, protect)`.

- [ ] **Step 7: Formatear, compilar y ejecutar**

Run: `gofmt -w internal tests && go build ./... && go vet ./... && go test ./internal/... -count=1`
Expected: sin errores; las pruebas unitarias siguen en verde.

Run: `go test ./tests/integration/ -run "TestComanda" -count=1 -v`
Expected: PASS en `TestComandaSeEnviaEnTandasSinReenviar`, `TestComandaRepetidaConLaMismaClaveDevuelveLaOriginal`, `TestComandaRechazaEnviosInvalidos` (8 subcasos), `TestComandaNoSeEnviaDeUnaLineaAnulada` y `TestComandaPermisosYSedes`. Si un código de error difiere del esperado, **no relajar la aserción**: leer la respuesta y corregir el caso de uso o la prueba según el spec §3.1.

- [ ] **Step 8: Commit**

```bash
git add internal tests
git commit -m "I3 rebanada 2: POST /pedidos/{id}/comandas, persistencia, HTTP y pruebas de integración" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Transiciones de preparación

**Files:**
- Modify: `internal/modules/produccion/application/dto/command/comandas.go` (ya contiene `TransicionarLinea`; sin cambios)
- Modify: `internal/modules/produccion/application/port/outbound/comandas.go` (añadir `LineaComandaRef` y `TransicionStore`)
- Modify: `internal/modules/produccion/application/port/inbound/comandas.go` (añadir `TransicionarLinea`)
- Create: `internal/modules/produccion/application/usecase/command/transiciones.go`, `transiciones_test.go`
- Modify: `internal/modules/produccion/infrastructure/persistence/write/repository/comandas_sql.go` (añadir dos métodos)
- Modify: `internal/modules/produccion/infrastructure/persistence/write/adapter/comandas.go` (añadir dos métodos y la aserción)
- Modify: `internal/modules/produccion/infrastructure/config/comandas.go`
- Modify: `internal/modules/produccion/web/dto/request/comandas.go`, `web/controller/comandas_handler.go`, `web/controller/routes.go`
- Create: `tests/integration/cocina_transiciones_test.go`

**Interfaces:**
- Consumes: Tasks 2–4 (`policy.Permite`, `policy.PermisosDe`, `policy.EstadoDestinoValido`, `mapper.Requiere`, `mapper.Autorizar`, `cargar.ScanLinea`, `cargar.ColLinea`, helpers de pruebas).
- Produces:
  - `outbound.LineaComandaRef{ID int64; SucursalID int64; Estado string}`.
  - `outbound.TransicionStore` con `BloquearLineaComanda(ctx, id int64) (LineaComandaRef, bool, error)` y `TransicionarLinea(ctx, id int64, destino string) (dtoresult.ComandaLinea, error)`.
  - `command.NewTransicionarLinea(outbound.TransicionStore) *TransicionarLinea`; `inbound.TransicionarLinea`.
  - Ruta `POST /comanda-detalles/{id}/transiciones` con cuerpo `{"estado": "...", "motivo": null}` → 200 con la línea.

- [ ] **Step 1: Ampliar los puertos**

Añadir al final de `application/port/outbound/comandas.go`:

```go
// LineaComandaRef es una línea de comanda bloqueada con la sede de su comanda.
type LineaComandaRef struct {
	ID         int64
	SucursalID int64
	Estado     string
}

// TransicionStore escribe el avance de una línea de comanda dentro de la transacción del comando.
type TransicionStore interface {
	// BloquearLineaComanda lee la línea FOR UPDATE; found=false si no existe.
	BloquearLineaComanda(ctx context.Context, id int64) (LineaComandaRef, bool, error)
	// TransicionarLinea fija el estado y la marca de tiempo del paso, y devuelve la línea.
	TransicionarLinea(ctx context.Context, id int64, destino string) (dtoresult.ComandaLinea, error)
}
```

Añadir al final de `application/port/inbound/comandas.go`:

```go
type TransicionarLinea interface {
	Execute(context.Context, cmd.TransicionarLinea) (appresult.Result[dtoresult.ComandaLinea], error)
}
```

- [ ] **Step 2: Escribir las pruebas unitarias (fallan: no existe `NewTransicionarLinea`)**

`application/usecase/command/transiciones_test.go`:

```go
package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// transicionStoreFake implementa outbound.TransicionStore.
type transicionStoreFake struct {
	linea       outbound.LineaComandaRef
	lineaExiste bool

	destino       string
	transiciones  int
}

var _ outbound.TransicionStore = (*transicionStoreFake)(nil)

func (f *transicionStoreFake) BloquearLineaComanda(context.Context, int64) (outbound.LineaComandaRef, bool, error) {
	return f.linea, f.lineaExiste, nil
}
func (f *transicionStoreFake) TransicionarLinea(_ context.Context, id int64, destino string) (dtoresult.ComandaLinea, error) {
	f.transiciones++
	f.destino = destino
	return dtoresult.ComandaLinea{ID: id, Estado: destino}, nil
}

func fakeTransicion(estado string) *transicionStoreFake {
	return &transicionStoreFake{linea: outbound.LineaComandaRef{ID: 40, SucursalID: 1, Estado: estado}, lineaExiste: true}
}

func transicionBase(destino string, permisos ...string) cmd.TransicionarLinea {
	return cmd.TransicionarLinea{Actor: actorEn(1, permisos...), LineaID: 40, Estado: destino}
}

func TestTransicionarLineaAvanzaUnPaso(t *testing.T) {
	pasos := []struct{ desde, hacia string }{
		{"PENDIENTE", "PREPARACION"}, {"PREPARACION", "LISTO"}, {"LISTO", "ENTREGADO"},
	}
	for _, p := range pasos {
		f := fakeTransicion(p.desde)
		res, err := NewTransicionarLinea(f).Execute(context.Background(), transicionBase(p.hacia, policy.CocinaOperar))
		if err != nil || !res.IsOk() || f.destino != p.hacia || res.Value().Estado != p.hacia {
			t.Fatalf("%s→%s debía avanzar: %v %v", p.desde, p.hacia, err, res.Err())
		}
	}
}

func TestLaEntregaLaPuedeConfirmarQuienEditaPedidos(t *testing.T) {
	f := fakeTransicion("LISTO")
	res, err := NewTransicionarLinea(f).Execute(context.Background(), transicionBase("ENTREGADO", policy.PedidoEditar))
	if err != nil || !res.IsOk() {
		t.Fatalf("el mesero con PEDIDO_EDITAR debía confirmar la entrega: %v %v", err, res.Err())
	}
}

func TestTransicionarLineaRechaza(t *testing.T) {
	casos := []struct {
		nombre string
		store  *transicionStoreFake
		cmd    cmd.TransicionarLinea
		kind   apperror.Kind
		code   string
	}{
		{"sin actor", fakeTransicion("PENDIENTE"), cmd.TransicionarLinea{LineaID: 40, Estado: "PREPARACION"},
			apperror.KindUnauthorized, "no_autenticado"},
		{"estado desconocido", fakeTransicion("PENDIENTE"), transicionBase("COCINANDO", policy.CocinaOperar),
			apperror.KindValidation, "estado_invalido"},
		{"destino PENDIENTE", fakeTransicion("PENDIENTE"), transicionBase("PENDIENTE", policy.CocinaOperar),
			apperror.KindValidation, "estado_invalido"},
		{"el mesero no prepara", fakeTransicion("PENDIENTE"), transicionBase("PREPARACION", policy.PedidoEditar),
			apperror.KindForbidden, "sin_permiso"},
		{"el lector no entrega", fakeTransicion("LISTO"), transicionBase("ENTREGADO", policy.PedidoVer),
			apperror.KindForbidden, "sin_permiso"},
		{"línea inexistente", &transicionStoreFake{}, transicionBase("PREPARACION", policy.CocinaOperar),
			apperror.KindNotFound, "no_encontrado"},
		{"sede que el actor no ve", func() *transicionStoreFake {
			f := fakeTransicion("PENDIENTE")
			f.linea.SucursalID = 2
			return f
		}(), transicionBase("PREPARACION", policy.CocinaOperar), apperror.KindNotFound, "no_encontrado"},
		{"ve la sede pero no opera allí", func() *transicionStoreFake {
			f := fakeTransicion("PENDIENTE")
			f.linea.SucursalID = 2
			return f
		}(), cmd.TransicionarLinea{LineaID: 40, Estado: "PREPARACION", Actor: appctx.NewActor(7, "ana", 1, []appctx.Grant{
			{SucursalID: 1, Permiso: policy.CocinaOperar}, {SucursalID: 2, Permiso: policy.PedidoVer}})},
			apperror.KindForbidden, "sede_no_autorizada"},
		{"salto PENDIENTE→LISTO", fakeTransicion("PENDIENTE"), transicionBase("LISTO", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"retroceso LISTO→PREPARACION", fakeTransicion("LISTO"), transicionBase("PREPARACION", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"repetir el mismo estado", fakeTransicion("PREPARACION"), transicionBase("PREPARACION", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"CANCELADO es de la rebanada 3", fakeTransicion("PENDIENTE"), transicionBase("CANCELADO", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
		{"nada se avanza desde ENTREGADO", fakeTransicion("ENTREGADO"), transicionBase("ENTREGADO", policy.CocinaOperar),
			apperror.KindConflict, "transicion_invalida"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			res, err := NewTransicionarLinea(tc.store).Execute(context.Background(), tc.cmd)
			if err != nil {
				t.Fatalf("un rechazo esperado no es un error técnico: %v", err)
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if tc.store.transiciones != 0 {
				t.Fatalf("una transición rechazada no debe escribir")
			}
		})
	}
}
```

- [ ] **Step 3: Ejecutar y comprobar que falla**

Run: `go test ./internal/modules/produccion/application/... -count=1`
Expected: FAIL de compilación: `undefined: NewTransicionarLinea`.

- [ ] **Step 4: Implementar el caso de uso**

`application/usecase/command/transiciones.go`:

```go
package command

import (
	"context"
	"fmt"

	cmd "service-restaurant/internal/modules/produccion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoTransicion = "no tiene permiso para cambiar el estado de preparación"

// TransicionarLinea mueve una línea de comanda al estado siguiente (RES-RF-012). Solo bloquea la
// línea: no toma ningún otro bloqueo después, así que no puede cruzarse con las escrituras de
// pedido de R1 o R3 (que bloquean pedido y después líneas).
type TransicionarLinea struct{ store outbound.TransicionStore }

func NewTransicionarLinea(s outbound.TransicionStore) *TransicionarLinea { return &TransicionarLinea{s} }

func (uc *TransicionarLinea) Execute(ctx context.Context, c cmd.TransicionarLinea) (appresult.Result[dtoresult.ComandaLinea], error) {
	if !policy.EstadoDestinoValido(c.Estado) {
		return rechazar[dtoresult.ComandaLinea](mapper.Validacion("estado_invalido", "el estado indicado no es válido"))
	}
	permisos := policy.PermisosDe(c.Estado)
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoTransicion, permisos...); ae != nil {
		return rechazar[dtoresult.ComandaLinea](ae)
	}

	ref, found, err := uc.store.BloquearLineaComanda(ctx, c.LineaID)
	if err != nil {
		return tecnico[dtoresult.ComandaLinea]("buscar línea de comanda", err)
	}
	if !found {
		return rechazar[dtoresult.ComandaLinea](mapper.NoEncontrado("línea de comanda"))
	}
	if ae := mapper.Autorizar(c.Actor, ref.SucursalID, mapper.NoEncontrado("línea de comanda"), permisos...); ae != nil {
		return rechazar[dtoresult.ComandaLinea](ae)
	}
	if !policy.Permite(ref.Estado, c.Estado) {
		return rechazar[dtoresult.ComandaLinea](apperror.New(apperror.KindConflict, "transicion_invalida",
			fmt.Sprintf("no se puede pasar de %s a %s", ref.Estado, c.Estado)))
	}

	l, err := uc.store.TransicionarLinea(ctx, c.LineaID, c.Estado)
	if err != nil {
		return deAlmacen[dtoresult.ComandaLinea]("transicionar línea", err)
	}
	return appresult.Ok(l), nil
}
```

- [ ] **Step 5: Ejecutar las pruebas unitarias**

Run: `gofmt -w internal && go test ./internal/modules/produccion/... -count=1`
Expected: PASS (`TestTransicionarLineaAvanzaUnPaso`, `TestLaEntregaLaPuedeConfirmarQuienEditaPedidos`, `TestTransicionarLineaRechaza` con 13 subcasos).

- [ ] **Step 6: Escribir las pruebas de integración (fallan: la ruta no existe)**

`tests/integration/cocina_transiciones_test.go`:

```go
package integration

import (
	"net/http"
	"testing"
)

func transicionar(t *testing.T, tk, lineaComanda, estado string) resp {
	t.Helper()
	return post(t, "/api/v1/comanda-detalles/"+lineaComanda+"/transiciones", `{"estado":"`+estado+`"}`, tk)
}

func transicionarOK(t *testing.T, tk, lineaComanda, estado string) comandaLineaResp {
	t.Helper()
	r := transicionar(t, tk, lineaComanda, estado)
	esperar(t, r, http.StatusOK, "")
	var l comandaLineaResp
	r.json(t, &l)
	return l
}

// RES-CP-012: la cocina avanza PENDIENTE → PREPARACION → LISTO → ENTREGADO y queda la marca de cada paso.
func TestLineaDeComandaRecorreSusEstados(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	lc := c.Lineas[0].ID

	l := transicionarOK(t, e.cocinero, lc, "PREPARACION")
	if l.Estado != "PREPARACION" || l.IniciadoEn == nil || l.ListoEn != nil || l.EntregadoEn != nil {
		t.Fatalf("PREPARACION: %+v", l)
	}
	l = transicionarOK(t, e.cocinero, lc, "LISTO")
	if l.Estado != "LISTO" || l.IniciadoEn == nil || l.ListoEn == nil || l.EntregadoEn != nil {
		t.Fatalf("LISTO: %+v", l)
	}
	l = transicionarOK(t, e.cocinero, lc, "ENTREGADO")
	if l.Estado != "ENTREGADO" || l.EntregadoEn == nil || l.Cantidad != "2.000" || l.ComandaID != c.ID {
		t.Fatalf("ENTREGADO: %+v", l)
	}
}

func TestLineaDeComandaRechazaTransicionesInvalidas(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	lc := c.Lineas[0].ID

	esperar(t, transicionar(t, e.cocinero, lc, "LISTO"), http.StatusConflict, "transicion_invalida")
	esperar(t, transicionar(t, e.cocinero, lc, "ENTREGADO"), http.StatusConflict, "transicion_invalida")
	esperar(t, transicionar(t, e.cocinero, lc, "CANCELADO"), http.StatusConflict, "transicion_invalida")
	esperar(t, transicionar(t, e.cocinero, lc, "COCINANDO"), http.StatusUnprocessableEntity, "estado_invalido")

	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	esperar(t, transicionar(t, e.cocinero, lc, "PREPARACION"), http.StatusConflict, "transicion_invalida")
	transicionarOK(t, e.cocinero, lc, "LISTO")
	esperar(t, transicionar(t, e.cocinero, lc, "PREPARACION"), http.StatusConflict, "transicion_invalida")

	esperar(t, transicionar(t, e.cocinero, "999999999", "PREPARACION"), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/comanda-detalles/"+lc+"/transiciones", `{"estado":"LISTO","otro":1}`, e.cocinero),
		http.StatusBadRequest, "")
}

func TestLaEntregaLaConfirmaElMeseroPeroNoLaPreparacion(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	lc := c.Lineas[0].ID

	esperar(t, transicionar(t, e.mesero, lc, "PREPARACION"), http.StatusForbidden, "sin_permiso")
	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	transicionarOK(t, e.cocinero, lc, "LISTO")
	esperar(t, transicionar(t, lectorPedidos(t), lc, "ENTREGADO"), http.StatusForbidden, "sin_permiso")
	if l := transicionarOK(t, e.mesero, lc, "ENTREGADO"); l.Estado != "ENTREGADO" {
		t.Fatalf("el mesero debía confirmar la entrega: %+v", l)
	}
}

func TestTransicionNoCruzaSedes(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	// Un cocinero que solo pertenece a COC4: `tokenPedidos` añadiría PRINCIPAL, así que se arma a mano.
	crearSedeSQL(t, "COC4")
	rolConPermisos(t, "T_COC_OTRA", "COCINA_OPERAR", "PEDIDO_VER")
	asegurarUsuario(t, "coc.cocinero.otra", "Pedidos2026Clave", "T_COC_OTRA", "COC4")
	otra := token(t, "coc.cocinero.otra", "Pedidos2026Clave")
	esperar(t, transicionar(t, otra, c.Lineas[0].ID, "PREPARACION"), http.StatusNotFound, "no_encontrado")
}
```

- [ ] **Step 7: Ejecutar y comprobar que las pruebas de integración fallan por la ruta**

Run: `go test ./tests/integration/ -run "TestLineaDeComanda|TestLaEntrega|TestTransicion" -count=1`
Expected: FAIL con 404 en `/comanda-detalles/{id}/transiciones` (la ruta no existe todavía).

- [ ] **Step 8: Implementar persistencia, adaptador, web y composición**

Añadir a `infrastructure/persistence/write/repository/comandas_sql.go`:

```go
// GetLineaComandaForUpdate bloquea la línea de comanda; trae la sede de su comanda.
func (r *ComandaSQLWriter) GetLineaComandaForUpdate(ctx context.Context, id int64) (outbound.LineaComandaRef, bool, error) {
	var l outbound.LineaComandaRef
	err := r.db.QueryRowContext(ctx, `
		SELECT cd.id, c.sucursal_id, cd.estado
		FROM sch_restaurante.comanda_detalles cd
		JOIN sch_restaurante.comandas c ON c.id = cd.comanda_id
		WHERE cd.id = $1
		FOR UPDATE OF cd`, id).Scan(&l.ID, &l.SucursalID, &l.Estado)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.LineaComandaRef{}, false, err
	}
	return l, true, nil
}

// UpdateLineaComanda fija el estado y la marca de tiempo del paso. Las marcas usan la hora de la
// transacción; los CHECK de la tabla (listo_en >= iniciado_en, entregado_en >= listo_en) se cumplen
// porque cada paso es una transacción posterior a la anterior.
func (r *ComandaSQLWriter) UpdateLineaComanda(ctx context.Context, id int64, destino string) (dtoresult.ComandaLinea, error) {
	l, err := cargar.ScanLinea(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.comanda_detalles SET
			estado = $2::text,
			iniciado_en  = CASE WHEN $2::text = 'PREPARACION' THEN CURRENT_TIMESTAMP ELSE iniciado_en END,
			listo_en     = CASE WHEN $2::text = 'LISTO'       THEN CURRENT_TIMESTAMP ELSE listo_en END,
			entregado_en = CASE WHEN $2::text = 'ENTREGADO'   THEN CURRENT_TIMESTAMP ELSE entregado_en END
		WHERE id = $1
		RETURNING `+cargar.ColLinea, id, destino))
	return l, Translate(err)
}
```

Añadir a `infrastructure/persistence/write/adapter/comandas.go` (y cambiar la aserción para cubrir ambos puertos):

```go
var (
	_ outbound.EnvioStore      = (*ComandaWriteAdapter)(nil)
	_ outbound.TransicionStore = (*ComandaWriteAdapter)(nil)
)

func (a *ComandaWriteAdapter) BloquearLineaComanda(ctx context.Context, id int64) (outbound.LineaComandaRef, bool, error) {
	return a.repo.GetLineaComandaForUpdate(ctx, id)
}
func (a *ComandaWriteAdapter) TransicionarLinea(ctx context.Context, id int64, destino string) (dtoresult.ComandaLinea, error) {
	return a.repo.UpdateLineaComanda(ctx, id, destino)
}
```

(Eliminar la línea `var _ outbound.EnvioStore = (*ComandaWriteAdapter)(nil)` que ya existía, para no duplicarla.)

Añadir a `web/dto/request/comandas.go`:

```go
// PreparacionTransicion: estado destino y, por contrato, un motivo que hoy se ignora.
type PreparacionTransicion struct {
	Estado string  `json:"estado"`
	Motivo *string `json:"motivo,omitempty"`
}
```

Modificar `web/controller/comandas_handler.go`: el struct, el constructor y el método nuevo.

```go
type ComandasHandler struct {
	enviar      inbound.EnviarComanda
	transicionar inbound.TransicionarLinea
}

func NewComandasHandler(e inbound.EnviarComanda, t inbound.TransicionarLinea) *ComandasHandler {
	return &ComandasHandler{e, t}
}

// Transicionar: POST /comanda-detalles/{id}/transiciones
func (h *ComandasHandler) Transicionar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.PreparacionTransicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.transicionar.Execute(r.Context(), command.TransicionarLinea{
		Actor: actor, LineaID: id, Estado: body.Estado, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToLinea)
}
```

Modificar `web/controller/routes.go`: añadir `ruta("POST /comanda-detalles/{id}/transiciones", comandas.Transicionar)`.

Modificar `infrastructure/config/comandas.go`: añadir antes de `return`:

```go
	transicionar := escritura[cmd.TransicionarLinea, dtoresult.ComandaLinea]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.TransicionarLinea, dtoresult.ComandaLinea] {
			return command.NewTransicionarLinea(writeadapter.NewComandaWriteAdapter(q))
		},
		meta: func(c cmd.TransicionarLinea) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
```

y cambiar el `return` a `return controller.NewComandasHandler(inbound.EnviarComanda(enviar), inbound.TransicionarLinea(transicionar))`.

- [ ] **Step 9: Formatear, compilar y ejecutar todo lo de cocina**

Run: `gofmt -w internal tests && go build ./... && go vet ./... && go test ./internal/... -count=1 && go test ./tests/integration/ -run "TestComanda|TestLinea|TestLaEntrega|TestTransicion|TestMigracion00012" -count=1 -v`
Expected: PASS en todo, incluida `TestTransicionNoCruzaSedes` ejecutándose (no saltada: en la salida `-v` no debe aparecer `SKIP`).

- [ ] **Step 10: Commit**

```bash
git add internal tests
git commit -m "I3 rebanada 2: transiciones de preparación (POST /comanda-detalles/{id}/transiciones)" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Cola de preparación — GET /comandas

**Files:**
- Create: `internal/modules/produccion/application/dto/query/comandas.go`
- Modify: `internal/modules/produccion/application/port/outbound/comandas.go` (añadir `FiltroComandas`, `ComandaReader`)
- Modify: `internal/modules/produccion/application/port/inbound/comandas.go` (añadir `ListarComandas`)
- Create: `internal/modules/produccion/application/usecase/query/comandas.go`, `comandas_test.go`
- Create: `internal/modules/produccion/infrastructure/persistence/read/repository/comandas_sql_reader.go`
- Create: `internal/modules/produccion/infrastructure/persistence/read/adapter/comandas_read_adapter.go`
- Modify: `internal/modules/produccion/infrastructure/config/comandas.go`
- Modify: `internal/modules/produccion/web/controller/comandas_handler.go`, `routes.go`
- Create: `tests/integration/cocina_cola_test.go`

**Interfaces:**
- Consumes: Tasks 3–5 (`cargar.Comandas`, `mapper.SedesCocina`, `mapper.Requiere`, `mapper.DecodeCursor`, `mapper.EncodeCursor`, `webmapper.ToComandasPagina`, helpers `consulta`, `limiteDe`, `idOpcionalDe`, `enumDe`).
- Produces: `qry.ListarComandas{Actor; SucursalID *int64; EstacionID *int64; Estado string; Limite int; Cursor string}`; `outbound.FiltroComandas{SedeIDs []int64; EstacionID *int64; Estado string; DespuesDeID int64; Limite int}`; `outbound.ComandaReader{ListarComandas(ctx, FiltroComandas) ([]dtoresult.Comanda, error)}`; `query.NewListarComandas(outbound.ComandaReader)`; ruta `GET /comandas`.

- [ ] **Step 1: DTO de consulta y puertos**

`application/dto/query/comandas.go`:

```go
// Package query agrupa los DTO de consultas del módulo produccion.
package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarComandas pagina por cursor (id ascendente) las comandas de las sedes donde el actor opera la
// cocina. Estado vacío significa «con alguna línea pendiente de entregar» (PENDIENTE, PREPARACION o
// LISTO); con un estado, las comandas que tienen alguna línea en él.
type ListarComandas struct {
	Actor      appctx.Actor
	SucursalID *int64
	EstacionID *int64
	Estado     string
	Limite     int
	Cursor     string
}
```

Añadir al final de `application/port/outbound/comandas.go`:

```go
// FiltroComandas acota la cola a las sedes visibles (SedeIDs) y, opcionalmente, por estación y por
// el estado de alguna de sus líneas. Estado vacío = líneas pendientes de entregar.
type FiltroComandas struct {
	SedeIDs     []int64
	EstacionID  *int64
	Estado      string
	DespuesDeID int64
	Limite      int
}

// ComandaReader lee comandas sin bloquear filas.
type ComandaReader interface {
	ListarComandas(ctx context.Context, f FiltroComandas) ([]dtoresult.Comanda, error)
}
```

Añadir al final de `application/port/inbound/comandas.go` (añadiendo el import `qry "service-restaurant/internal/modules/produccion/application/dto/query"`):

```go
type ListarComandas interface {
	Execute(context.Context, qry.ListarComandas) (appresult.Result[dtoresult.PaginaComandas], error)
}
```

- [ ] **Step 2: Pruebas unitarias del caso de uso (fallan: no existe `NewListarComandas`)**

`application/usecase/query/comandas_test.go`:

```go
package query

import (
	"context"
	"testing"

	qry "service-restaurant/internal/modules/produccion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type lectorFake struct {
	comandas []dtoresult.Comanda
	filtro   outbound.FiltroComandas
	llamadas int
}

var _ outbound.ComandaReader = (*lectorFake)(nil)

func (f *lectorFake) ListarComandas(_ context.Context, fl outbound.FiltroComandas) ([]dtoresult.Comanda, error) {
	f.llamadas++
	f.filtro = fl
	return f.comandas, nil
}

func cocinero(sedes ...int64) appctx.Actor {
	g := make([]appctx.Grant, len(sedes))
	for i, s := range sedes {
		g[i] = appctx.Grant{SucursalID: s, Permiso: policy.CocinaOperar}
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

func TestListarComandasUsaSoloLasSedesDeCocina(t *testing.T) {
	f := &lectorFake{}
	res, err := NewListarComandas(f).Execute(context.Background(), qry.ListarComandas{Actor: cocinero(3, 1), Limite: 10})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía listar: %v %v", err, res.Err())
	}
	if got := f.filtro.SedeIDs; len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("sedes del filtro: %v", got)
	}
	if f.filtro.Limite != 11 || res.Value().Items == nil {
		t.Fatalf("pide limite+1 y nunca devuelve items nil: %+v %v", f.filtro, res.Value())
	}
}

func TestListarComandasPaginaConCursor(t *testing.T) {
	f := &lectorFake{comandas: []dtoresult.Comanda{{ID: 4}, {ID: 7}, {ID: 9}}}
	res, _ := NewListarComandas(f).Execute(context.Background(), qry.ListarComandas{Actor: cocinero(1), Limite: 2})
	p := res.Value()
	if len(p.Items) != 2 || p.Items[1].ID != 7 || p.NextCursor != mapper.EncodeCursor(7) {
		t.Fatalf("página: %+v", p)
	}
	res, _ = NewListarComandas(f).Execute(context.Background(), qry.ListarComandas{Actor: cocinero(1), Limite: 2, Cursor: p.NextCursor})
	if f.filtro.DespuesDeID != 7 {
		t.Fatalf("el cursor debe pasar al filtro: %d", f.filtro.DespuesDeID)
	}
}

func TestListarComandasRechaza(t *testing.T) {
	sede2 := int64(2)
	casos := []struct {
		nombre string
		q      qry.ListarComandas
		kind   apperror.Kind
		code   string
	}{
		{"sin actor", qry.ListarComandas{}, apperror.KindUnauthorized, "no_autenticado"},
		{"sin permiso de cocina", qry.ListarComandas{Actor: appctx.NewActor(7, "ana", 1,
			[]appctx.Grant{{SucursalID: 1, Permiso: policy.PedidoVer}})}, apperror.KindForbidden, "sin_permiso"},
		{"sucursal que no opera", qry.ListarComandas{Actor: cocinero(1), SucursalID: &sede2}, apperror.KindNotFound, "no_encontrado"},
		{"cursor inválido", qry.ListarComandas{Actor: cocinero(1), Cursor: "###"}, apperror.KindBadRequest, "cursor_invalido"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := &lectorFake{}
			res, err := NewListarComandas(f).Execute(context.Background(), tc.q)
			if err != nil {
				t.Fatal(err)
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.llamadas != 0 {
				t.Fatal("un rechazo no debe consultar la base")
			}
		})
	}
}
```

- [ ] **Step 3: Ejecutar y comprobar que falla**

Run: `go test ./internal/modules/produccion/application/usecase/query/ -count=1`
Expected: FAIL de compilación: `undefined: NewListarComandas`.

- [ ] **Step 4: Implementar el caso de uso**

`application/usecase/query/comandas.go`:

```go
// Package query implementa las consultas del módulo produccion.
package query

import (
	"context"
	"fmt"
	"slices"

	qry "service-restaurant/internal/modules/produccion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/mapper"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoCola = "no tiene permiso para consultar la cola de preparación"

// tamanio acota el tamaño de página a 1..100 (por defecto 25).
func tamanio(l int) int {
	switch {
	case l < 1:
		return 25
	case l > 100:
		return 100
	}
	return l
}

// ListarComandas devuelve la cola de las sedes donde el actor opera la cocina. Pide limite+1 filas
// para saber si hay más páginas.
type ListarComandas struct{ comandas outbound.ComandaReader }

func NewListarComandas(c outbound.ComandaReader) *ListarComandas { return &ListarComandas{c} }

func (uc *ListarComandas) Execute(ctx context.Context, q qry.ListarComandas) (appresult.Result[dtoresult.PaginaComandas], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoCola, policy.CocinaOperar); ae != nil {
		return appresult.Fail[dtoresult.PaginaComandas](ae), nil
	}
	sedes := mapper.SedesCocina(q.Actor)
	if q.SucursalID != nil {
		if !slices.Contains(sedes, *q.SucursalID) {
			return appresult.Fail[dtoresult.PaginaComandas](mapper.NoEncontrado("sucursal")), nil
		}
		sedes = []int64{*q.SucursalID}
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaComandas](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.comandas.ListarComandas(ctx, outbound.FiltroComandas{
		SedeIDs: sedes, EstacionID: q.EstacionID, Estado: q.Estado, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaComandas]{}, fmt.Errorf("listar comandas: %w", err)
	}
	pagina := dtoresult.PaginaComandas{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Comanda{}
	}
	return appresult.Ok(pagina), nil
}
```

Run: `gofmt -w internal && go test ./internal/modules/produccion/... -count=1`
Expected: PASS.

- [ ] **Step 5: Pruebas de integración de la cola (fallan: la ruta no existe)**

`tests/integration/cocina_cola_test.go`:

```go
package integration

import (
	"net/http"
	"net/url"
	"testing"
)

func colaDe(t *testing.T, tk string, q url.Values) comandasPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/comandas?"+q.Encode(), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p comandasPagina
	r.json(t, &p)
	return p
}

func (p comandasPagina) ids() []string {
	out := make([]string, len(p.Items))
	for i, c := range p.Items {
		out[i] = c.ID
	}
	return out
}

func TestColaMuestraLoPendienteDeEntregarPorEstacion(t *testing.T) {
	e := nuevoEscenario(t, "4")
	c1 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	c2 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "3"})
	porEstacion := url.Values{"estacion_id": {e.estacion}}

	p := colaDe(t, e.cocinero, porEstacion)
	if got := p.ids(); len(got) != 2 || got[0] != c1.ID || got[1] != c2.ID {
		t.Fatalf("cola inicial: %v", got)
	}
	if p.Items[0].Lineas[0].Estado != "PENDIENTE" || p.Items[0].Lineas[0].Cantidad != "1.000" || p.Items[0].MesaID != nil {
		t.Fatalf("contenido de la primera comanda: %+v", p.Items[0])
	}

	// Entregar toda la primera comanda la saca de la cola por defecto, pero sigue visible con estado=ENTREGADO.
	lc := c1.Lineas[0].ID
	transicionarOK(t, e.cocinero, lc, "PREPARACION")
	transicionarOK(t, e.cocinero, lc, "LISTO")
	if got := colaDe(t, e.cocinero, porEstacion).ids(); len(got) != 2 {
		t.Fatalf("una línea LISTO aún está pendiente de entregar: %v", got)
	}
	transicionarOK(t, e.cocinero, lc, "ENTREGADO")
	if got := colaDe(t, e.cocinero, porEstacion).ids(); len(got) != 1 || got[0] != c2.ID {
		t.Fatalf("la cola no debe incluir lo ya entregado: %v", got)
	}
	porEstado := url.Values{"estacion_id": {e.estacion}, "estado": {"ENTREGADO"}}
	if got := colaDe(t, e.cocinero, porEstado).ids(); len(got) != 1 || got[0] != c1.ID {
		t.Fatalf("estado=ENTREGADO: %v", got)
	}
	porListo := url.Values{"estacion_id": {e.estacion}, "estado": {"LISTO"}}
	if got := colaDe(t, e.cocinero, porListo).ids(); len(got) != 0 {
		t.Fatalf("ninguna comanda tiene líneas LISTO: %v", got)
	}
}

func TestColaPaginaPorCursor(t *testing.T) {
	e := nuevoEscenario(t, "3")
	c1 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	c2 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	c3 := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})

	q := url.Values{"estacion_id": {e.estacion}, "limit": {"2"}}
	p1 := colaDe(t, e.cocinero, q)
	if got := p1.ids(); len(got) != 2 || got[0] != c1.ID || got[1] != c2.ID || p1.NextCursor == nil {
		t.Fatalf("primera página: %v %v", got, p1.NextCursor)
	}
	q.Set("cursor", *p1.NextCursor)
	p2 := colaDe(t, e.cocinero, q)
	if got := p2.ids(); len(got) != 1 || got[0] != c3.ID || p2.NextCursor != nil {
		t.Fatalf("segunda página: %v %v", got, p2.NextCursor)
	}
}

func TestColaValidaPermisosFiltrosYSedes(t *testing.T) {
	e := nuevoEscenario(t, "1")
	enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})

	esperar(t, call(t, "GET", "/api/v1/comandas", "", e.mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/comandas", "", ""), http.StatusUnauthorized, "")
	esperar(t, call(t, "GET", "/api/v1/comandas?estado=COCINANDO", "", e.cocinero), http.StatusBadRequest, "estado_invalido")
	esperar(t, call(t, "GET", "/api/v1/comandas?limit=0", "", e.cocinero), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/comandas?estacion_id=abc", "", e.cocinero), http.StatusBadRequest, "estacion_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/comandas?cursor=%23%23%23", "", e.cocinero), http.StatusBadRequest, "cursor_invalido")

	crearSedeSQL(t, "COC5")
	esperar(t, call(t, "GET", "/api/v1/comandas?sucursal_id="+sedeID(t, "COC5"), "", e.cocinero),
		http.StatusNotFound, "no_encontrado")
	if got := colaDe(t, e.cocinero, url.Values{"sucursal_id": {sedeID(t, "PRINCIPAL")}, "estacion_id": {e.estacion}}).ids(); len(got) != 1 {
		t.Fatalf("la sede propia debe listar su comanda: %v", got)
	}
}
```

Run: `go test ./tests/integration/ -run "TestCola" -count=1`
Expected: FAIL (404: la ruta `GET /comandas` no existe).

- [ ] **Step 6: Implementar el lector, el adaptador, el handler y la composición**

`infrastructure/persistence/read/repository/comandas_sql_reader.go`:

```go
// Package repository contiene el SQL de lectura del módulo produccion.
package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/platform/database"
)

// ComandaSQLReader lee la cola de preparación.
type ComandaSQLReader struct{ db database.DBTX }

func NewComandaSQLReader(db database.DBTX) *ComandaSQLReader { return &ComandaSQLReader{db} }

// ListarComandas devuelve las comandas de las sedes de f.SedeIDs ordenadas por id (cursor estable),
// cada una con todas sus líneas. Con Estado vacío, las que tienen alguna línea PENDIENTE, PREPARACION
// o LISTO; con Estado, las que tienen alguna línea en ese estado.
func (r *ComandaSQLReader) ListarComandas(ctx context.Context, f outbound.FiltroComandas) ([]dtoresult.Comanda, error) {
	var estacion sql.NullInt64
	if f.EstacionID != nil {
		estacion = sql.NullInt64{Int64: *f.EstacionID, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id FROM sch_restaurante.comandas c
		WHERE c.sucursal_id = ANY($1::bigint[]) AND c.id > $2
		  AND ($3::bigint IS NULL OR c.estacion_id = $3)
		  AND EXISTS (SELECT 1 FROM sch_restaurante.comanda_detalles cd
		               WHERE cd.comanda_id = c.id
		                 AND (cd.estado = $4::text
		                      OR ($4::text = '' AND cd.estado IN ('PENDIENTE','PREPARACION','LISTO'))))
		ORDER BY c.id LIMIT $5`, f.SedeIDs, f.DespuesDeID, estacion, f.Estado, f.Limite)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	if len(ids) == 0 {
		return nil, nil
	}
	return cargar.Comandas(ctx, r.db, ids)
}
```

`infrastructure/persistence/read/adapter/comandas_read_adapter.go`:

```go
// Package adapter implementa los puertos de lectura del módulo produccion sobre SQL.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/produccion/application/dto/result"
	"service-restaurant/internal/modules/produccion/application/port/outbound"
	"service-restaurant/internal/modules/produccion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ComandaReadAdapter implementa outbound.ComandaReader.
type ComandaReadAdapter struct{ repo *repository.ComandaSQLReader }

func NewComandaReadAdapter(db database.DBTX) *ComandaReadAdapter {
	return &ComandaReadAdapter{repo: repository.NewComandaSQLReader(db)}
}

var _ outbound.ComandaReader = (*ComandaReadAdapter)(nil)

func (a *ComandaReadAdapter) ListarComandas(ctx context.Context, f outbound.FiltroComandas) ([]dtoresult.Comanda, error) {
	return a.repo.ListarComandas(ctx, f)
}
```

En `web/controller/comandas_handler.go`: añadir el import `"service-restaurant/internal/modules/produccion/application/dto/query"`, el campo `listar inbound.ListarComandas`, el estado válido y el método (el constructor pasa a recibir tres puertos):

```go
var estadosLinea = []string{"PENDIENTE", "PREPARACION", "LISTO", "ENTREGADO", "CANCELADO"}

type ComandasHandler struct {
	listar       inbound.ListarComandas
	enviar       inbound.EnviarComanda
	transicionar inbound.TransicionarLinea
}

func NewComandasHandler(l inbound.ListarComandas, e inbound.EnviarComanda, t inbound.TransicionarLinea) *ComandasHandler {
	return &ComandasHandler{l, e, t}
}

// Listar: GET /comandas?sucursal_id=&estacion_id=&estado=&limit=&cursor=
func (h *ComandasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := idOpcionalDe(q, "sucursal_id")
	if err != nil {
		return err
	}
	estacion, err := idOpcionalDe(q, "estacion_id")
	if err != nil {
		return err
	}
	estado, err := enumDe(q, "estado", estadosLinea...)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarComandas{
		Actor: actor, SucursalID: sede, EstacionID: estacion, Estado: estado, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToComandasPagina)
}
```

En `web/controller/routes.go` añadir `ruta("GET /comandas", comandas.Listar)`.

En `infrastructure/config/comandas.go`: añadir los imports `uquery "service-restaurant/internal/modules/produccion/application/usecase/query"` y `readadapter "service-restaurant/internal/modules/produccion/infrastructure/persistence/read/adapter"`, y cambiar el final por:

```go
	lector := readadapter.NewComandaReadAdapter(db)
	return controller.NewComandasHandler(uquery.NewListarComandas(lector),
		inbound.EnviarComanda(enviar), inbound.TransicionarLinea(transicionar))
```

Y en `application/port/inbound/comandas.go` el puerto `ListarComandas` ya quedó añadido en el Step 1.

- [ ] **Step 7: Formatear, compilar y ejecutar**

Run: `gofmt -w internal tests && go build ./... && go vet ./... && go test ./internal/... -count=1 && go test ./tests/integration/ -run "TestCola|TestComanda|TestLinea|TestLaEntrega|TestTransicion" -count=1 -v`
Expected: PASS en todo.

- [ ] **Step 8: Commit**

```bash
git add internal tests
git commit -m "I3 rebanada 2: cola de preparación (GET /comandas) con filtros y cursor" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Concurrencia, arquitectura y suite completa

**Files:**
- Create: `tests/integration/cocina_concurrencia_test.go`

**Interfaces:**
- Consumes: helpers de la Task 4 (`nuevoEscenario`, `enviarComanda`), `callH`, `nuevaClave`.

- [ ] **Step 1: Escribir la prueba de concurrencia**

`tests/integration/cocina_concurrencia_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// Dos envíos simultáneos de la misma línea no pueden superar su cantidad: el pedido se bloquea
// FOR UPDATE, así que el segundo envío ve lo que dejó el primero.
func TestDosEnviosSimultaneosNoSuperanLaCantidad(t *testing.T) {
	e := nuevoEscenario(t, "3")
	cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"2"}]}`, e.estacion, e.linea.ID)
	ruta := "/api/v1/pedidos/" + e.pedido.ID + "/comandas"
	app(t) // inicializa la aplicación antes de lanzar goroutines

	const n = 6
	codigos := make([]int, n)
	var inicio, fin sync.WaitGroup
	inicio.Add(1)
	for i := 0; i < n; i++ {
		fin.Add(1)
		go func(i int) {
			defer fin.Done()
			inicio.Wait()
			r := callH(t, "POST", ruta, cuerpo, e.mesero, map[string]string{"Idempotency-Key": nuevaClave()})
			codigos[i] = r.Code
		}(i)
	}
	inicio.Done()
	fin.Wait()

	creadas, rechazadas := 0, 0
	for _, c := range codigos {
		switch c {
		case http.StatusCreated:
			creadas++
		case http.StatusConflict:
			rechazadas++
		default:
			t.Fatalf("código inesperado %d en %v", c, codigos)
		}
	}
	if creadas != 1 || rechazadas != n-1 {
		t.Fatalf("de %d envíos de 2 sobre una línea de 3 debía crearse exactamente 1: %v", n, codigos)
	}
	if enviado := scalar(t, `SELECT sum(cantidad)::int FROM sch_restaurante.comanda_detalles WHERE detalle_pedido_id=$1::text::bigint`, e.linea.ID); enviado != 2 {
		t.Fatalf("lo enviado debe ser 2 (hay %d)", enviado)
	}
}
```

- [ ] **Step 2: Ejecutar la prueba, la de arquitectura y la suite completa**

Run: `go test ./tests/integration/ -run TestDosEnviosSimultaneos -count=1 -race -v`
Expected: PASS (1 `201` y 5 `409`). Si el detector de carreras reporta algo, no ignorarlo: diagnosticar.

Run: `go test ./tests/architecture/ -count=1 -v`
Expected: PASS. Si falla con «importa internos de otro módulo» o similar, el módulo `produccion` importa algo que no debe; corregir el código, no la regla.

Run: `go vet ./... && go test ./... -count=1`
Expected: PASS en todos los paquetes (la suite de integración tarda varios minutos).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/cocina_concurrencia_test.go
git commit -m "I3 rebanada 2: prueba de concurrencia del envío a cocina" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Contrato y expediente

**Files:**
- Modify: `docs/planificacion/03_api/openapi.yaml`
- Modify: `docs/planificacion/01_documentacion/requisitos_y_trazabilidad.json` y `.html`, `backlog_incrementos.json`
- Modify: `docs/planificacion/CAMBIOS_1_3.txt`, `SHA256SUMS.txt`
- Modify: `service-restaurant/docs/module-map.md`
- Modify: `docs/superpowers/specs/2026-10-28-i3-rebanada2-comandas-design.md`

R2 **no añade rutas ni operaciones** (las tres ya existen en el contrato), así que `MANIFIESTO.json`, `00_LEEME.txt` y `validacion_entrega.json` no cambian sus conteos.

- [ ] **Step 1: Marcar las tres operaciones y ampliar los esquemas**

Desde `docs/planificacion/`:

```bash
awk '/operationId: operacion_(119|120|121)$/ {m=1} m && /x-implementado: false/ {sub(/false/,"true"); m=0} {print}' 03_api/openapi.yaml > "$TEMP/openapi.nuevo" && cp "$TEMP/openapi.nuevo" 03_api/openapi.yaml
git diff --stat 03_api/openapi.yaml
git diff -U0 03_api/openapi.yaml | grep '^[-+] ' 
```

Expected: exactamente 3 líneas `-      x-implementado: false` / `+      x-implementado: true`. Si el diff muestra más o menos, deshacer (`git checkout -- 03_api/openapi.yaml`) y revisar el `awk`. Si el archivo usa fin de línea CRLF, el `awk` los conserva; comprobar que el diff no reescribe todo el archivo.

Con la herramienta Edit sobre `docs/planificacion/03_api/openapi.yaml`:

1. En el esquema `Comandas`, añadir `mesa_id` y `lineas` y la descripción de la cola. Reemplazar este bloque (único: `clave_idempotencia` seguido de `enviado_en`):

```
        clave_idempotencia:
          type: string
          format: uuid
        enviado_en:
          type: string
          format: date-time
```

por

```
        clave_idempotencia:
          type: string
          format: uuid
        mesa_id:
          anyOf:
          - $ref: '#/components/schemas/Id'
          - type: 'null'
          description: Mesa del pedido de salón; null en LLEVAR y DELIVERY.
        lineas:
          type: array
          items:
            $ref: '#/components/schemas/ComandaDetalles'
          description: Todas las líneas de la comanda con su estado de preparación.
        enviado_en:
          type: string
          format: date-time
```

Si Edit informa que el texto no es único, ampliar `old_string` con las dos líneas anteriores del esquema `Comandas` (`pedido_id`/`sucursal_id`/`estacion_id`) hasta que lo sea; no usar `replace_all`.

2. Añadir a la descripción de las tres operaciones el comportamiento implementado. En `operacion_120` (`GET /comandas`), reemplazar su línea `description: Diseño propuesto. El backend valida permiso, sucursal, campos y estado en una transacción cuando corresponda.` (la que precede a `x-implementado: true` y `x-permiso: COCINA_OPERAR` de esa operación) por:

```
      description: Cola de preparación de las sedes donde el usuario opera la cocina. Sin estado, devuelve las comandas con alguna
        línea PENDIENTE, PREPARACION o LISTO; con estado, las que tienen alguna línea en él. Filtros sucursal_id, estacion_id y
        paginación por cursor.
```

Hacer lo mismo con `operacion_119` («Envía a una estación las cantidades pendientes de las líneas indicadas; cada línea debe corresponder a la estación (422 estacion_no_corresponde) y no puede superar lo pendiente (409 cantidad_excede_pendiente). Idempotency-Key obligatorio.») y con `operacion_121` («Avanza la línea al estado siguiente: PENDIENTE→PREPARACION→LISTO→ENTREGADO. COCINA_OPERAR; la entrega también con PEDIDO_EDITAR. CANCELADO se resuelve en la rebanada 3. No lleva Idempotency-Key: repetir una transición da 409 transicion_invalida.»). Cada reemplazo debe dejar una sola clave `description:` por operación y conservar el resto de la operación intacto.

- [ ] **Step 2: Script de cierre del expediente**

Crear `$TEMP/cierre-i3-r2.mjs` con este contenido y ejecutarlo desde `docs/planificacion/`:

```js
import fs from 'node:fs';
import crypto from 'node:crypto';

const leer = (f) => fs.readFileSync(f, 'utf8');
const lista = (j) => (Array.isArray(j) ? j : Object.values(j).find(Array.isArray));
const guardarJson = (f, j) => fs.writeFileSync(f, JSON.stringify(j, null, 2) + '\n');
const falla = (m) => { throw new Error(m); };

// ---- 1. RES-RF-012 en requisitos_y_trazabilidad.json y .html ----------------
const SIT = 'Implementado en I3 (rebanada 2): POST /pedidos/{id}/comandas envía a una estación solo las cantidades pendientes de cada línea (sin reenviar lo despachado, con la estación del producto exigida y Idempotency-Key; repetir la clave devuelve la misma comanda), GET /comandas es la cola de preparación por sede y POST /comanda-detalles/{id}/transiciones avanza cada línea PENDIENTE→PREPARACION→LISTO→ENTREGADO (la entrega también con PEDIDO_EDITAR). Pruebas: TestComandaSeEnviaEnTandasSinReenviar, TestDosEnviosSimultaneosNoSuperanLaCantidad y TestLineaDeComandaRecorreSusEstados. Pendiente: cancelar (R3) y la pantalla de Cocina (R4). RES-CP-012 sigue NO_EJECUTADA.';
const fReq = '01_documentacion/requisitos_y_trazabilidad.json';
const req = JSON.parse(leer(fReq));
const r12 = lista(req).find((x) => x.id === 'RES-RF-012');
if (!r12) falla('RES-RF-012 no existe');
r12.estado_backend = 'IMPLEMENTADO';
r12.situacion_backend = SIT;
guardarJson(fReq, req);

const fHtml = '01_documentacion/requisitos_y_trazabilidad.html';
let html = leer(fHtml);
{
  const a = html.indexOf('id="RES-RF-012"');
  const b = html.indexOf('</tbody>', a);
  if (a < 0 || b < 0) falla('HTML sin sección RES-RF-012');
  let bloque = html.slice(a, b);
  const re = /<td>(PENDIENTE|PARCIAL) · prototipo /;
  if (!re.test(bloque)) falla('RES-RF-012: el HTML no tenía el estado esperado; leer el archivo y ajustar este script');
  bloque = bloque.replace(re, '<td>IMPLEMENTADO · prototipo ');
  bloque = bloque.replace(/<tr><td>Situación backend<\/td><td>.*?<\/td><\/tr>/, '');
  bloque += '<tr><td>Situación backend</td><td>' + SIT + '</td></tr>';
  html = html.slice(0, a) + bloque + html.slice(b);
}
fs.writeFileSync(fHtml, html);

// ---- 2. backlog_incrementos.json -------------------------------------------
const fBack = '01_documentacion/backlog_incrementos.json';
const back = JSON.parse(leer(fBack));
const i3 = lista(back).find((x) => (x.id || x.codigo) === 'I3');
if (!i3) falla('I3 no existe en el backlog');
i3.estado = 'EN_CURSO';
i3.avance = 'Rebanadas 1 y 2 de 4 hechas (backend): apertura de pedido y cuenta (RES-RF-009), consumos como instantánea (RES-RF-010), autorización por sede (RES-RF-002) y, en R2, envío de comandas por estación, cola de preparación y estados de cocina (RES-RF-012). Faltan dividir cuentas y cancelar con motivo (R3, RES-RF-011 y RES-RF-014) y el frontend de Salón y Cocina con el cierre con auditoría (R4). Los casos de aceptación siguen NO_EJECUTADA.';
guardarJson(fBack, back);

// ---- 3. CAMBIOS_1_3.txt ----------------------------------------------------
const fCam = 'CAMBIOS_1_3.txt';
let cam = leer(fCam);
if (!cam.endsWith('\n')) cam += '\n';
cam +=
  '- I3 rebanada 2: comandas y estados de cocina (backend, módulo produccion). RES-RF-012 pasa a IMPLEMENTADO en backend;\n' +
  '  RES-CP-012 sigue NO_EJECUTADA. El contrato marca como implementadas POST /pedidos/{id}/comandas, GET /comandas y\n' +
  '  POST /comanda-detalles/{id}/transiciones (sin nuevas rutas: 120 rutas y 164 operaciones) y el esquema Comandas añade\n' +
  '  mesa_id y lineas. COCINA_OPERAR se asigna al rol COCINA (migración 00012).\n';
fs.writeFileSync(fCam, cam);

// ---- 4. SHA256SUMS.txt -----------------------------------------------------
const fApi = '03_api/openapi.yaml';
const tocados = new Set([fApi, fReq, fHtml, fBack, fCam]);
const sums = leer('SHA256SUMS.txt').split('\n').map((l) => {
  const m = /^([0-9a-f]{64})  (.+)$/.exec(l);
  if (!m || !tocados.has(m[2])) return l;
  return crypto.createHash('sha256').update(fs.readFileSync(m[2])).digest('hex') + '  ' + m[2];
});
fs.writeFileSync('SHA256SUMS.txt', sums.join('\n'));
console.log('ok: RES-RF-012 IMPLEMENTADO, I3 EN_CURSO (R1 y R2)');
```

Run (desde `docs/planificacion/`): `node "$TEMP/cierre-i3-r2.mjs"`
Expected: `ok: RES-RF-012 IMPLEMENTADO, I3 EN_CURSO (R1 y R2)`. Si lanza un error de ancla («no existe», «el HTML no tenía el estado esperado»), abrir el archivo real, entender la diferencia y corregir el script; **no forzar** el cambio. Si falla a mitad, `git checkout -- docs/planificacion` (después de re-aplicar el `awk` y las dos ediciones del Step 1) y reintentar.

- [ ] **Step 3: Validar**

Run (desde `docs/planificacion/`):

```bash
node -e 'JSON.parse(require("fs").readFileSync("01_documentacion/requisitos_y_trazabilidad.json","utf8")); JSON.parse(require("fs").readFileSync("01_documentacion/backlog_incrementos.json","utf8")); console.log("json ok")'
sha256sum -c SHA256SUMS.txt 2>&1 | grep -v ": OK"
git diff --stat .
grep -c "x-implementado: true" 03_api/openapi.yaml
```

Expected: `json ok`; `sha256sum` sin líneas de error; el diff muestra los 5 archivos esperados; el conteo de `x-implementado: true` sube en 3 respecto a antes (era 88 → 91). Para validar el YAML usar una herramienta real si existe (`npx --yes @redocly/cli lint 03_api/openapi.yaml` o `node -e 'require("yaml")'`); si no hay ninguna, decirlo explícitamente en el informe: no marcar el contrato como validado sin una validación real.

- [ ] **Step 4: Documentación del repositorio**

En `service-restaurant/docs/module-map.md`:

- En el párrafo «Módulos hoy», tras la descripción de `catalogo`, añadir: «y **`produccion`** (implementado en I3 rebanada 2: comandas por estación, cola de preparación y estados de las líneas de comanda)». Ajustar la conjunción del texto existente para que lea bien (`…, **`catalogo`** (…) y **`produccion`** (…)`).
- En la tabla, fila «Producción»: cambiar la celda de tablas a «comandas y comanda_detalles (`00003`, I3 rebanada 2) · delivery (`00003`, I6)» y el incremento a «I3 (R2) / I6».

En `docs/superpowers/specs/2026-10-28-i3-rebanada2-comandas-design.md`, añadir al final:

```markdown

## 8. Ajustes durante el plan

Al escribir el plan (`docs/superpowers/plans/2026-10-28-i3-rebanada2-comandas.md`) se reconcilió este diseño con el código y el contrato existentes. Las diferencias con las secciones anteriores son:

| Sección | Diseño | Implementado | Motivo |
|---|---|---|---|
| §1 | `00012` asigna `COCINA_OPERAR` a COCINA «y a ADMIN si no lo tiene» | Solo a COCINA | `00001` asigna todos los permisos existentes a ADMIN |
| §2 | `POST …/transiciones` con `Idempotency-Key` | Sin clave | Mismo criterio que el `PATCH` de consumos de R1; repetirla da 409 `transicion_invalida` |
| §3.2 | Sin permiso: 403 `sin_permiso` | `sin_permiso` si no tiene el permiso en ninguna sede; `sede_no_autorizada` si lo tiene en otra sede pero no en la del recurso | Mismo contrato que `AutorizarPedidos` de R1 |
| §3.2 | `CANCELADO` no se alcanza | `CANCELADO` → 409 `transicion_invalida`; un estado desconocido o `PENDIENTE` → 422 `estado_invalido`; `motivo` se acepta y se ignora | Contrato `PreparacionTransicion` |
| §3.3 | Cola por sede con filtros | Sin `estado`, comandas con alguna línea PENDIENTE, PREPARACION o LISTO; con `estado`, las que tienen alguna línea en él; siempre con todas sus líneas | Una cola de cocina no debe mostrar el historial entregado |
| §3.3 | «mesa si existe» | `mesa_id` (cadena o `null`) | El nombre de la mesa lo resuelve el frontend (R4) |
| §3.1 | Respuesta del envío | Comanda con `lineas[]` y `mesa_id` (esquema `Comandas` ampliado) | La pantalla necesita las líneas creadas |

`pedidos.estado` (`REGISTRADO/PREPARACION/LISTO/ENTREGADO`) no se actualiza desde las comandas en R2: el estado de cocina vive en las líneas de comanda. Derivarlo se decidirá con R3/R4 o I4.
```

- [ ] **Step 5: Última verificación y commit**

Run (desde `service-restaurant/`): `go vet ./... && go test ./... -count=1`
Expected: PASS en todo. Esta es la evidencia que se reporta; no declarar la rebanada terminada sin ella.

```bash
git add docs service-restaurant/docs
git commit -m "I3 rebanada 2: contrato y expediente de comandas (RF-012)" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

## Autorrevisión del plan contra el spec

| Spec | Task |
|---|---|
| §1 migración `00012`, sin cambios de esquema | 1 |
| §2 `POST /pedidos/{id}/comandas` | 3 (caso de uso), 4 (HTTP e integración) |
| §2 `GET /comandas` | 6 |
| §2 `POST /comanda-detalles/{id}/transiciones` | 5 |
| §3.1 reglas 1–8 del envío (bloqueo, estado, pendiente, estación estricta, instantánea, línea repetida, idempotencia, auditoría) | 3 (unitarias), 4 (integración, incluida la idempotencia), 7 (concurrencia) |
| §3.2 transiciones y permisos de la entrega | 2 (política), 5 |
| §3.3 consulta por sede | 6 |
| §4 arquitectura y frontera entre módulos | 3–6; verificada por `tests/architecture` en 7 |
| §5 pruebas (política, integración, concurrencia, arquitectura) | 2, 3, 4, 5, 6, 7 |
| §6 cierre del expediente | 8 |
| §7 pendientes heredados (D4, cancelar líneas con comanda) | No se tocan; quedan para R3 |

Consistencia de nombres revisada: `EnvioStore`/`TransicionStore`/`ComandaReader` (puertos), `ComandaWriteAdapter` (implementa los dos de escritura), `NewComandasHandler(l, e, t)` con el orden listar-enviar-transicionar desde la Task 6 (en las Tasks 4 y 5 el constructor tiene menos parámetros y se amplía en su momento), `cargar.ScanLinea/ColLinea/Comandas/Comanda`, `mapper.Requiere/Autorizar/Visible/SedesCocina`.
