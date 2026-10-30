# I3 rebanada 3 — Dividir cuentas y cancelar con motivo: plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implementar RES-RF-011 y RES-RF-014 en el módulo `atencion`: dividir las líneas de un pedido en una cuenta nueva, anular una línea con motivo y cancelar un pedido entero con motivo, cancelando en cascada lo que cocina aún no entregó.

**Architecture:** Tres casos de uso nuevos en `internal/modules/atencion` (`DividirCuenta`, `AnularDetalle`, `CancelarPedido`), con las mismas capas que R1 y decorados con transacción e idempotencia (`escritura[C,T]`). La cascada a `comanda_detalles` es SQL de la persistencia de `atencion` dentro de la misma transacción (R1 ya consulta esa tabla). Las reglas puras (qué líneas viajan con un padre) viven en `domain/policy`. Una migración (`00013`) asigna `PEDIDO_ANULAR` al rol CAJERO; el esquema no cambia.

**Tech Stack:** Go 1.26, `net/http` (ServeMux), `database/sql` + pgx, PostgreSQL 18 (testcontainers en las pruebas de integración), goose.

Spec: `docs/superpowers/specs/2026-10-29-i3-rebanada3-cancelar-dividir-design.md`. Rama: `i3-rebanada3-cancelar-dividir` (la actual).

## Global Constraints

- Toda ruta y comando se ejecuta desde `service-restaurant/` salvo que el paso diga otra cosa. Las pruebas de integración necesitan Docker (testcontainers). `-race` no se puede usar en este equipo (no hay `gcc`).
- `atencion` **no importa** `produccion`, `catalogo` ni `identidad` (lo verifica `tests/architecture`). Dominio y aplicación no importan `database/sql`, `net/http` ni drivers. `web` no importa dominio ni infraestructura.
- Tablas siempre con el esquema: `sch_restaurante.<tabla>`.
- Cantidades y ids viajan como cadenas en JSON (contrato).
- Orden de bloqueo (regla de R1 y R2): todo escritor de `atencion` bloquea primero el pedido `FOR UPDATE`. Los escritores de líneas de cocina del módulo `produccion` (transiciones) bloquean solo la fila de `comanda_detalles` y no el pedido; por eso `atencion` debe bloquear las filas de `comanda_detalles` (`FOR UPDATE`) antes de decidir cualquier cosa que dependa de su estado. Dentro de una transacción de `atencion` el orden es: pedido → líneas/cuentas → líneas de comanda.
- Errores (`apperror`): `422` validación, `409` conflicto de estado, `403`/`404` por sede. Sin ningún permiso del conjunto exigido: `403 sin_permiso`. Con visibilidad de la sede pero sin el permiso: `403 sede_no_autorizada`. Pedido de una sede que el actor no ve: `404 no_encontrado`.
- Mensajes de usuario y comentarios en español, con tildes, como el resto del repositorio.
- Mensaje de cada commit termina con la línea `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>` (se pasa con un segundo `-m`).
- Después de editar Go: `gofmt -w internal tests` antes de compilar o probar.

## Ajustes respecto al spec (decididos al planificar)

| Spec | Plan | Motivo |
|---|---|---|
| §3.4: «el total ignora las líneas anuladas; afecta a la lectura de pedidos de R1» | No hay cambio de código en los totales; solo se amplía `Linea` con `anulado_en`, `anulado_por` y `motivo_anulacion` | `leerTotales` ya filtra `estado = 'ACTIVO'`; lo que faltaba era devolver el motivo de la anulación |
| §3.5: D4 responde 403 `sin_permiso_descuento` | Reutiliza el rechazo existente `sinPermisoDescuento()` (403 `sin_permiso`) | Es el que ya emiten `AgregarDetalle` y `EditarDetalle` para descuentos; un código nuevo rompería la coherencia del contrato |
| §3.1: `nombre_cuenta` según el contrato admite 100 caracteres | Máximo 80 y se corrige el contrato | `cuentas.etiqueta` es `VARCHAR(80)` |
| §3.1: la respuesta de dividir es el esquema `Cuentas` | Es el esquema `CuentaPedido` (cuenta con sus líneas y totales) | `Cuentas` exige `pedido_id` y metadatos de auditoría que la API de pedidos no devuelve; `CuentaPedido` ya describe la cuenta anidada de `GET /pedidos/{id}` |
| §4: «reglas de qué estados de cocina admiten anular» en `domain` | Constante en la persistencia (`estadosCocinaCancelables`) | Es un filtro de la sentencia `UPDATE`; duplicarlo en dominio no lo haría más seguro y sí más fácil de desalinear |
| §3.1: pago confirmado en la cuenta de origen | Comprobación en la aplicación con el puerto `CuentasConPago` (409 `cuenta_con_pago`); el trigger `a_guardar_detalle` queda como segunda defensa | La regla se rechaza antes con un código propio, no con el mensaje genérico del trigger |

Fuera de alcance, y así queda dicho: pagos, devoluciones y cierre del pedido con saldo (I4); el frontend de Salón y Cocina (R4); derivar `pedidos.estado` de las comandas.

## Mapa de archivos

Todo bajo `service-restaurant/` salvo que se indique.

Crear:

| Archivo | Responsabilidad |
|---|---|
| `migrations/00013_anulacion_permisos.sql` | Asigna `PEDIDO_ANULAR` al rol CAJERO |
| `internal/modules/atencion/domain/policy/division.go` | `SeleccionDeDivision`: qué líneas viajan al dividir (los adicionales con su padre) |
| `internal/modules/atencion/application/dto/command/cuentas.go`, `anulaciones.go` | Comandos `DividirCuenta`, `AnularDetalle`, `CancelarPedido` |
| `internal/modules/atencion/application/port/inbound/cuentas.go` | Puertos de entrada de los tres casos de uso |
| `internal/modules/atencion/application/port/outbound/cuentas.go`, `anulaciones.go` | `PedidoBloqueable`, `DivisionStore`, `AnulacionStore` y sus tipos |
| `internal/modules/atencion/application/usecase/command/cuentas.go`, `anulaciones.go` | Casos de uso de escritura |
| `internal/modules/atencion/infrastructure/persistence/write/repository/cuentas_sql.go`, `anulaciones_sql.go` | SQL de división, anulación y cancelación |
| `internal/modules/atencion/web/dto/request/cuentas.go` | Cuerpo de `dividir-cuenta` |
| `tests/integration/anulacion_permisos_migracion_test.go`, `pedidos_r3_helpers_test.go`, `pedidos_descuento_cantidad_test.go`, `cuentas_division_test.go`, `pedidos_anulacion_test.go`, `pedidos_cancelacion_concurrencia_test.go` | Pruebas de integración |

Modificar: `internal/modules/atencion/domain/policy/permisos.go`, `domain/vo/pedido.go`, `application/usecase/command/consumos.go`, `application/dto/result/pedidos.go`, `infrastructure/persistence/internal/columnas/pedidos.go`, `infrastructure/persistence/write/adapter/pedidos.go`, `infrastructure/config/pedidos.go`, `web/controller/pedidos_handler.go`, `web/controller/routes.go`, `web/dto/response/pedidos.go`, `web/mapper/pedidos_web_mapper.go`, `tests/integration/pedidos_helpers_test.go`; y, en la última tarea, `docs/planificacion/…` (contrato y expediente), `service-restaurant/docs/module-map.md` y el spec de R3 (sección de ajustes).

---

### Task 1: Migración 00013 — PEDIDO_ANULAR para el cajero

**Files:**
- Create: `migrations/00013_anulacion_permisos.sql`
- Test: `tests/integration/anulacion_permisos_migracion_test.go`

**Interfaces:**
- Consumes: `cuentaRolPermiso(t, roles, permisos string) int` (en `carta_permisos_migracion_test.go`) y `sinComentarios(s string) string` (en `configuracion_migracion_test.go`).
- Produces: el rol `CAJERO` tiene `PEDIDO_ANULAR` (`es_activo`) en `roles_permisos`. Las Tasks 5 y 7 prueban con roles de prueba propios; esta migración es la que lo deja así en producción.

- [ ] **Step 1: Escribir las pruebas que fallan**

`tests/integration/anulacion_permisos_migracion_test.go`:

```go
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
```

- [ ] **Step 2: Ejecutar y comprobar que falla**

Run: `go test ./tests/integration/ -run "TestMigracion00013" -count=1`
Expected: FAIL. `TestMigracion00013Permisos` dice «CAJERO debe recibir PEDIDO_ANULAR (0)» y `TestMigracion00013BajaLimpia` falla con «file does not exist».

- [ ] **Step 3: Escribir la migración**

`migrations/00013_anulacion_permisos.sql`:

```sql
-- Migracion 00013: permiso de anulacion para el cajero (I3, rebanada 3).
-- PEDIDO_ANULAR existe desde 00001 y solo lo tiene ADMIN (00001 asigna todos los permisos a ADMIN):
-- aqui se asigna al rol CAJERO. Anular es una accion de supervision: MESERO y COCINA no la reciben.
-- Requiere PostgreSQL 18 y las migraciones 00001-00012.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='CAJERO' AND p.codigo='PEDIDO_ANULAR'
  AND NOT EXISTS (SELECT 1 FROM roles_permisos x WHERE x.rol_id=r.id AND x.permiso_id=p.id);

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo='CAJERO')
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='PEDIDO_ANULAR');
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
```

- [ ] **Step 4: Ejecutar y comprobar que pasa**

Run: `go test ./tests/integration/ -run "TestMigracion0001[0-3]" -count=1`
Expected: PASS (incluye las migraciones anteriores, que siguen intactas).

- [ ] **Step 5: Commit**

```bash
git add migrations/00013_anulacion_permisos.sql tests/integration/anulacion_permisos_migracion_test.go
git commit -m "I3 rebanada 3: migración 00013, PEDIDO_ANULAR para el rol CAJERO" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Dominio — permiso de anular y selección de líneas al dividir

**Files:**
- Modify: `internal/modules/atencion/domain/policy/permisos.go`
- Create: `internal/modules/atencion/domain/policy/division.go`, `division_test.go`
- Modify: `internal/modules/atencion/domain/vo/pedido.go`
- Create: `internal/modules/atencion/domain/vo/pedido_comparar_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `policy.PedidoAnular = "PEDIDO_ANULAR"`.
  - `policy.LineaPedido{ID int64; PadreID *int64; CuentaID int64; Activa bool}`.
  - `policy.SeleccionDeDivision(pedidas []int64, lineas []LineaPedido) ([]int64, error)`: devuelve los ids a mover, ordenados y sin repetir. Errores: `policy.ErrSeleccionVacia`, `ErrLineaRepetida`, `ErrLineaInexistente`, `ErrLineaAnulada`, `ErrAdicionalSinPadre`.
  - `(vo.Cantidad).Menor(otra vo.Cantidad) bool`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`internal/modules/atencion/domain/policy/division_test.go`:

```go
package policy

import (
	"errors"
	"slices"
	"testing"
)

func idp(n int64) *int64 { return &n }

func lineasDePrueba() []LineaPedido {
	return []LineaPedido{
		{ID: 1, CuentaID: 20, Activa: true},
		{ID: 2, CuentaID: 20, Activa: true},
		{ID: 3, CuentaID: 20, PadreID: idp(1), Activa: true},  // adicional activo de la 1
		{ID: 4, CuentaID: 20, PadreID: idp(1), Activa: false}, // adicional anulado de la 1
		{ID: 5, CuentaID: 21, Activa: false},                  // línea anulada
	}
}

func TestSeleccionDeDivision(t *testing.T) {
	casos := []struct {
		nombre  string
		pedidas []int64
		quiere  []int64
	}{
		{"una línea simple", []int64{2}, []int64{2}},
		{"el padre arrastra a sus adicionales activos", []int64{1}, []int64{1, 3}},
		{"padre e hijo explícitos no se duplican", []int64{3, 1}, []int64{1, 3}},
		{"varias líneas salen ordenadas", []int64{2, 1}, []int64{1, 2, 3}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, err := SeleccionDeDivision(c.pedidas, lineasDePrueba())
			if err != nil || !slices.Equal(got, c.quiere) {
				t.Fatalf("selección %v (%v), quería %v", got, err, c.quiere)
			}
		})
	}
}

func TestSeleccionDeDivisionRechaza(t *testing.T) {
	casos := []struct {
		nombre  string
		pedidas []int64
		quiere  error
	}{
		{"vacía", nil, ErrSeleccionVacia},
		{"línea repetida", []int64{2, 2}, ErrLineaRepetida},
		{"línea que no es del pedido", []int64{99}, ErrLineaInexistente},
		{"línea anulada", []int64{5}, ErrLineaAnulada},
		{"adicional sin su padre", []int64{3}, ErrAdicionalSinPadre},
		{"adicional anulado", []int64{4, 1}, ErrLineaAnulada},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if _, err := SeleccionDeDivision(c.pedidas, lineasDePrueba()); !errors.Is(err, c.quiere) {
				t.Fatalf("error %v, quería %v", err, c.quiere)
			}
		})
	}
}
```

`internal/modules/atencion/domain/vo/pedido_comparar_test.go`:

```go
package vo

import "testing"

func TestCantidadMenor(t *testing.T) {
	casos := []struct {
		a, b   string
		quiere bool
	}{
		{"1", "2", true},
		{"2", "1", false},
		{"2", "2.000", false},
		{"1.999", "2", true},
	}
	for _, c := range casos {
		a, err := NewCantidad(c.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := NewCantidad(c.b)
		if err != nil {
			t.Fatal(err)
		}
		if got := a.Menor(b); got != c.quiere {
			t.Errorf("%s.Menor(%s) = %v, quería %v", c.a, c.b, got, c.quiere)
		}
	}
}
```

- [ ] **Step 2: Ejecutar y comprobar que falla**

Run: `go test ./internal/modules/atencion/domain/... -count=1`
Expected: FAIL de compilación: `undefined: SeleccionDeDivision` y `a.Menor undefined`.

- [ ] **Step 3: Implementar**

En `internal/modules/atencion/domain/policy/permisos.go`, después de la línea `PedidoEditar       = "PEDIDO_EDITAR"` (Edit; la línea es única):

```go
	PedidoAnular       = "PEDIDO_ANULAR" // existe desde 00001; la migración 00013 lo asigna a CAJERO
```

`internal/modules/atencion/domain/policy/division.go`:

```go
package policy

import (
	"errors"
	"slices"
)

// Rechazos de la selección de líneas al dividir una cuenta.
var (
	ErrSeleccionVacia    = errors.New("la selección de líneas está vacía")
	ErrLineaRepetida     = errors.New("una línea se repite en la selección")
	ErrLineaInexistente  = errors.New("la línea no pertenece al pedido")
	ErrLineaAnulada      = errors.New("la línea está anulada")
	ErrAdicionalSinPadre = errors.New("un adicional no viaja sin la línea de la que cuelga")
)

// LineaPedido es lo que la división de cuentas necesita saber de una línea del pedido.
type LineaPedido struct {
	ID       int64
	PadreID  *int64
	CuentaID int64
	Activa   bool
}

// SeleccionDeDivision resuelve qué líneas se mueven a la cuenta nueva: las pedidas más los
// adicionales activos que cuelgan de ellas. Un adicional pedido sin su padre en la selección se
// rechaza. El resultado va ordenado y sin repetidos.
func SeleccionDeDivision(pedidas []int64, lineas []LineaPedido) ([]int64, error) {
	if len(pedidas) == 0 {
		return nil, ErrSeleccionVacia
	}
	porID := make(map[int64]LineaPedido, len(lineas))
	for _, l := range lineas {
		porID[l.ID] = l
	}
	pedida := make(map[int64]bool, len(pedidas))
	for _, id := range pedidas {
		if pedida[id] {
			return nil, ErrLineaRepetida
		}
		pedida[id] = true
	}
	for _, id := range pedidas {
		l, ok := porID[id]
		switch {
		case !ok:
			return nil, ErrLineaInexistente
		case !l.Activa:
			return nil, ErrLineaAnulada
		case l.PadreID != nil && !pedida[*l.PadreID]:
			return nil, ErrAdicionalSinPadre
		}
	}
	movidas := slices.Clone(pedidas)
	for _, l := range lineas {
		if l.PadreID != nil && pedida[*l.PadreID] && !pedida[l.ID] && l.Activa {
			movidas = append(movidas, l.ID)
		}
	}
	slices.Sort(movidas)
	return movidas, nil
}
```

En `internal/modules/atencion/domain/vo/pedido.go`, después de `func (c Cantidad) String() string { return c.text }`:

```go

// Menor indica si c es estrictamente menor que otra.
func (c Cantidad) Menor(otra Cantidad) bool { return c.r.Cmp(otra.r) < 0 }
```

- [ ] **Step 4: Formatear, ejecutar y comprobar que pasa**

Run: `gofmt -w internal && go test ./internal/modules/atencion/domain/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/atencion/domain
git commit -m "I3 rebanada 3: dominio de la división de cuentas y permiso de anular" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Decisión D4 — bajar la cantidad de una línea con descuento exige DESCUENTO_AUTORIZAR

**Files:**
- Modify: `internal/modules/atencion/application/usecase/command/consumos.go`
- Modify: `internal/modules/atencion/application/usecase/command/consumos_test.go`
- Create: `tests/integration/pedidos_descuento_cantidad_test.go`

**Interfaces:**
- Consumes: `(vo.Cantidad).Menor` de la Task 2; `sinPermisoDescuento()` y `conDescuento()` ya existentes.
- Produces: `EditarDetalle` rechaza con 403 `sin_permiso` si la cantidad baja y la línea resultante conserva `descuento_total > 0` sin `DESCUENTO_AUTORIZAR`.

- [ ] **Step 1: Escribir las pruebas que fallan**

Añadir al final de `internal/modules/atencion/application/usecase/command/consumos_test.go`:

```go
// D4: bajar la cantidad de una línea que conserva descuento hace que el descuento pese más sobre
// ella, así que exige el mismo permiso que fijarlo.
func TestEditarDetalleBajarLaCantidadConDescuentoExigePermiso(t *testing.T) {
	conDescuentoPrevio := func() *pedidoStoreFake {
		f := fakeEdicion()
		d := f.detalles[55]
		d.DescuentoTotal = "1.00"
		f.detalles[55] = d
		return f
	}

	f := conDescuentoPrevio()
	c := editarBase()
	c.Cantidad = texto("1") // baja de 2 a 1
	res, err := NewEditarDetalle(f).Execute(context.Background(), c)
	sinFallo(t, err)
	esperarRechazo(t, res.Err(), apperror.KindForbidden, "sin_permiso")
	if f.actualizaciones != 0 {
		t.Fatal("un rechazo no debe escribir")
	}

	for nombre, mod := range map[string]func(*cmd.EditarDetalle){
		"subir la cantidad no exige el permiso":      func(c *cmd.EditarDetalle) { c.Cantidad = texto("3") },
		"dejarla igual no exige el permiso":          func(c *cmd.EditarDetalle) { c.Cantidad = texto("2") },
		"quien autoriza descuentos puede bajarla":    func(c *cmd.EditarDetalle) { c.Actor, c.Cantidad = conDescuento(), texto("1") },
		"bajarla y quitar el descuento no lo exige":  func(c *cmd.EditarDetalle) { c.Cantidad, c.DescuentoMonto = texto("1"), texto("0") },
	} {
		t.Run(nombre, func(t *testing.T) {
			f := conDescuentoPrevio()
			c := editarBase()
			mod(&c)
			res, err := NewEditarDetalle(f).Execute(context.Background(), c)
			if err != nil || !res.IsOk() || f.actualizaciones != 1 {
				t.Fatalf("debía editarse: %v %+v", err, res.Err())
			}
		})
	}

	// Sin descuento previo, bajar la cantidad tampoco exige el permiso.
	f = fakeEdicion()
	if res, err := NewEditarDetalle(f).Execute(context.Background(), func() cmd.EditarDetalle {
		c := editarBase()
		c.Cantidad = texto("1")
		return c
	}()); err != nil || !res.IsOk() {
		t.Fatalf("sin descuento no se exige el permiso: %v %+v", err, res.Err())
	}
}
```

`tests/integration/pedidos_descuento_cantidad_test.go`:

```go
package integration

import (
	"net/http"
	"testing"
)

// D4: sin DESCUENTO_AUTORIZAR no se puede bajar la cantidad de una línea que conserva descuento.
func TestBajarLaCantidadConDescuentoExigeElPermiso(t *testing.T) {
	cajero := cajeroPedidos(t)
	mesero := meseroPedidos(t)
	p := abrirLlevar(t, cajero, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	r := agregarConsumo(t, cajero, p.ID, p.Cuentas[0].ID, prod, "3", `,"descuento_monto":"1.00"`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles/" + l.ID

	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"2"}`, mesero), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"4"}`, mesero), http.StatusOK, "")
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"2"}`, cajero), http.StatusOK, "")
}
```

- [ ] **Step 2: Ejecutar y comprobar que fallan**

Run: `go test ./internal/modules/atencion/application/usecase/command/ -run TestEditarDetalleBajar -count=1`
Expected: FAIL: «rechazo <nil>, quería Forbidden/sin_permiso» (hoy se permite).

- [ ] **Step 3: Implementar**

En `internal/modules/atencion/application/usecase/command/consumos.go` (Edit), dentro de `EditarDetalle.Execute`, entre el bloque `if !vo.DescuentoCabe(*nuevaCantidad, det.PrecioUnitario, *nuevoDescuento) {…}` y la línea `l, err := uc.pedidos.ActualizarDetalle(ctx, c.DetalleID, cambios)`.

`old_string`:

```go
		return rechazar[dtoresult.Linea](descuentoExcede())
	}

	l, err := uc.pedidos.ActualizarDetalle(ctx, c.DetalleID, cambios)
```

`new_string`:

```go
		return rechazar[dtoresult.Linea](descuentoExcede())
	}
	// D4: bajar la cantidad de una línea que conserva descuento hace que el descuento pese más sobre
	// ella; sin DESCUENTO_AUTORIZAR no se permite.
	if cambios.Cantidad != nil && !nuevoDescuento.EsCero() && !c.Actor.Can(policy.DescuentoAutorizar, ped.SucursalID) {
		anterior, err := vo.NewCantidad(det.Cantidad)
		if err != nil {
			return tecnico[dtoresult.Linea]("leer la cantidad de la línea", err)
		}
		if nuevaCantidad.Menor(anterior) {
			return rechazar[dtoresult.Linea](sinPermisoDescuento())
		}
	}

	l, err := uc.pedidos.ActualizarDetalle(ctx, c.DetalleID, cambios)
```

- [ ] **Step 4: Formatear, ejecutar y comprobar que pasa**

Run: `gofmt -w internal tests && go test ./internal/modules/atencion/... -count=1 && go test ./tests/integration/ -run "TestBajarLaCantidad|TestEditar|TestConsumo" -count=1`
Expected: PASS (las pruebas de consumos existentes siguen en verde).

- [ ] **Step 5: Commit**

```bash
git add internal tests
git commit -m "I3 rebanada 3: bajar la cantidad de una línea con descuento exige DESCUENTO_AUTORIZAR (D4)" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Aplicación — caso de uso DividirCuenta

**Files:**
- Create: `internal/modules/atencion/application/dto/command/cuentas.go`
- Create: `internal/modules/atencion/application/port/inbound/cuentas.go`
- Create: `internal/modules/atencion/application/port/outbound/cuentas.go`
- Modify: `internal/modules/atencion/application/usecase/command/consumos.go` (generaliza `pedidoEditable`)
- Create: `internal/modules/atencion/application/usecase/command/cuentas.go`, `cuentas_fake_test.go`, `cuentas_test.go`

**Interfaces:**
- Consumes: `policy.SeleccionDeDivision`, `policy.PedidoCrear` (Task 2); `textoOpcional`, `rechazar`, `tecnico`, `deAlmacen`, `cuentaCerrada`, `estadoActivo`, `estadoCuentaAbta` (ya existen).
- Produces:
  - `cmd.DividirCuenta{Intencion; Actor appctx.Actor; PedidoID int64; DetalleIDs []int64; NombreCuenta *string}`.
  - `outbound.PedidoBloqueable` (`BloquearPedido`), `outbound.LineaCuenta{ID, CuentaID int64; PadreID *int64; Estado string}`, `outbound.NuevaCuenta{PedidoID int64; Etiqueta *string}`, `outbound.DivisionStore` (ver abajo).
  - `inbound.DividirCuenta` (`Execute(ctx, cmd.DividirCuenta) (Result[dtoresult.Cuenta], error)`).
  - `command.NewDividirCuenta(outbound.DivisionStore) *DividirCuenta`.
  - Helpers internos que reutiliza la Task 6: `pedidoAutorizado(ctx, store outbound.PedidoBloqueable, actor, pedidoID, permiso string) (outbound.PedidoRef, *apperror.AppError, error)`, `cuentaConPago()`, `lineaAnulada()`.

- [ ] **Step 1: DTO, puertos y generalización de `pedidoEditable` (compila solo)**

`internal/modules/atencion/application/dto/command/cuentas.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// DividirCuenta mueve líneas activas de las cuentas abiertas de un pedido a una cuenta nueva del
// mismo pedido. Los adicionales viajan con su línea padre. NombreCuenta es la etiqueta opcional.
type DividirCuenta struct {
	Intencion
	Actor        appctx.Actor
	PedidoID     int64
	DetalleIDs   []int64
	NombreCuenta *string
}
```

`internal/modules/atencion/application/port/inbound/cuentas.go`:

```go
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type DividirCuenta interface {
	Execute(context.Context, cmd.DividirCuenta) (appresult.Result[dtoresult.Cuenta], error)
}
```

`internal/modules/atencion/application/port/outbound/cuentas.go`:

```go
package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// PedidoBloqueable bloquea un pedido FOR UPDATE. Lo comparten los puertos de escritura de pedidos,
// de división de cuentas y de anulación: todos empiezan por este bloqueo.
type PedidoBloqueable interface {
	// BloquearPedido lee el pedido FOR UPDATE; found=false si no existe.
	BloquearPedido(ctx context.Context, id int64) (PedidoRef, bool, error)
}

// LineaCuenta es una línea bloqueada con lo que la división de cuentas necesita.
type LineaCuenta struct {
	ID       int64
	CuentaID int64
	PadreID  *int64
	Estado   string
}

// NuevaCuenta es la cuenta que nace al dividir. Etiqueta nil la deja sin nombre.
type NuevaCuenta struct {
	PedidoID int64
	Etiqueta *string
}

// DivisionStore escribe la división de un pedido dentro de la transacción del comando. Los métodos
// que devuelven filas las bloquean hasta el final de la transacción.
type DivisionStore interface {
	PedidoBloqueable
	// LineasDelPedido bloquea FOR UPDATE todas las líneas del pedido, ordenadas por id.
	LineasDelPedido(ctx context.Context, pedidoID int64) ([]LineaCuenta, error)
	// CuentasDelPedido bloquea FOR UPDATE las cuentas del pedido, ordenadas por id.
	CuentasDelPedido(ctx context.Context, pedidoID int64) ([]CuentaRef, error)
	// CuentasConPago devuelve, de las cuentas dadas, las que tienen algún pago CONFIRMADO.
	CuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error)
	// CrearCuenta inserta la cuenta con el siguiente número del pedido y devuelve su id.
	CrearCuenta(ctx context.Context, n NuevaCuenta) (int64, error)
	// MoverLineas reasigna las líneas a la cuenta.
	MoverLineas(ctx context.Context, pedidoID, cuentaID int64, lineaIDs []int64) error
	// CuentaConLineas lee la cuenta con sus líneas y totales.
	CuentaConLineas(ctx context.Context, pedidoID, cuentaID int64) (dtoresult.Cuenta, error)
}
```

En `internal/modules/atencion/application/usecase/command/consumos.go`, tres Edit para generalizar la comprobación del pedido (la lógica no cambia: solo el permiso pasa a ser un parámetro y `pedidoEditable` queda como atajo):

1. `old_string`:

```go
// pedidoEditable bloquea el pedido y comprueba que el actor pueda editarlo en su sede y que siga
// abierto. El rechazo esperado va en el segundo retorno; el fallo técnico, en el tercero.
// Regla: bloquea el pedido FOR UPDATE antes que cualquier cuenta o línea (spec §7).
func pedidoEditable(ctx context.Context, store outbound.PedidoStore, actor appctx.Actor, pedidoID int64) (outbound.PedidoRef, *apperror.AppError, error) {
```

`new_string`:

```go
// pedidoAutorizado bloquea el pedido y comprueba que el actor tenga el permiso en su sede y que
// siga abierto. El rechazo esperado va en el segundo retorno; el fallo técnico, en el tercero.
// Regla: bloquea el pedido FOR UPDATE antes que cualquier cuenta o línea (spec §7).
func pedidoAutorizado(ctx context.Context, store outbound.PedidoBloqueable, actor appctx.Actor, pedidoID int64, permiso string) (outbound.PedidoRef, *apperror.AppError, error) {
```

2. `old_string`: `mapper.AutorizarPedidos(actor, ped.SucursalID, policy.PedidoEditar, mapper.NoEncontradoM("pedido"))`
   `new_string`: `mapper.AutorizarPedidos(actor, ped.SucursalID, permiso, mapper.NoEncontradoM("pedido"))`

3. `old_string`:

```go
	return ped, nil, nil
}

// AgregarDetalle agrega una línea a una cuenta abierta fijando
```

`new_string`:

```go
	return ped, nil, nil
}

// pedidoEditable es pedidoAutorizado con PEDIDO_EDITAR: agregar y editar consumos.
func pedidoEditable(ctx context.Context, store outbound.PedidoStore, actor appctx.Actor, pedidoID int64) (outbound.PedidoRef, *apperror.AppError, error) {
	return pedidoAutorizado(ctx, store, actor, pedidoID, policy.PedidoEditar)
}

// AgregarDetalle agrega una línea a una cuenta abierta fijando
```

Run: `gofmt -w internal && go build ./... && go test ./internal/modules/atencion/... -count=1`
Expected: compila y las pruebas existentes siguen en verde.

- [ ] **Step 2: Escribir el doble y las pruebas del caso de uso (fallan: no existe `NewDividirCuenta`)**

`internal/modules/atencion/application/usecase/command/cuentas_fake_test.go`:

```go
package command

import (
	"context"
	"slices"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
)

// divisionFake implementa outbound.DivisionStore con respuestas fijadas por la prueba.
type divisionFake struct {
	pedido       outbound.PedidoRef
	pedidoExiste bool
	lineas       []outbound.LineaCuenta
	cuentas      []outbound.CuentaRef
	conPago      []int64
	nuevaID      int64

	creada      outbound.NuevaCuenta
	creaciones  int
	movidas     []int64
	destino     int64
	movimientos int
}

var _ outbound.DivisionStore = (*divisionFake)(nil)

func (f *divisionFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *divisionFake) LineasDelPedido(context.Context, int64) ([]outbound.LineaCuenta, error) {
	return f.lineas, nil
}
func (f *divisionFake) CuentasDelPedido(context.Context, int64) ([]outbound.CuentaRef, error) {
	return f.cuentas, nil
}
func (f *divisionFake) CuentasConPago(_ context.Context, ids []int64) ([]int64, error) {
	var out []int64
	for _, id := range ids {
		if slices.Contains(f.conPago, id) {
			out = append(out, id)
		}
	}
	return out, nil
}
func (f *divisionFake) CrearCuenta(_ context.Context, n outbound.NuevaCuenta) (int64, error) {
	f.creaciones++
	f.creada = n
	return f.nuevaID, nil
}
func (f *divisionFake) MoverLineas(_ context.Context, _, cuentaID int64, ids []int64) error {
	f.movimientos++
	f.destino = cuentaID
	f.movidas = ids
	return nil
}
func (f *divisionFake) CuentaConLineas(_ context.Context, _, cuentaID int64) (dtoresult.Cuenta, error) {
	return dtoresult.Cuenta{ID: cuentaID, Numero: 2}, nil
}

// fakeDivision es un pedido abierto con una cuenta y tres líneas: la 3 es un adicional de la 1.
func fakeDivision() *divisionFake {
	return &divisionFake{
		pedido:       outbound.PedidoRef{ID: 10, SucursalID: 1, Estado: "REGISTRADO"},
		pedidoExiste: true,
		lineas: []outbound.LineaCuenta{
			{ID: 1, CuentaID: 20, Estado: "ACTIVO"},
			{ID: 2, CuentaID: 20, Estado: "ACTIVO"},
			{ID: 3, CuentaID: 20, PadreID: idPtr(1), Estado: "ACTIVO"},
		},
		cuentas: []outbound.CuentaRef{{ID: 20, PedidoID: 10, Estado: "ABIERTA"}},
		nuevaID: 21,
	}
}
```

`internal/modules/atencion/application/usecase/command/cuentas_test.go`:

```go
package command

import (
	"context"
	"slices"
	"strings"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func dividirBase() cmd.DividirCuenta {
	return cmd.DividirCuenta{
		Intencion: cmd.Intencion{Clave: claveEjemplo},
		Actor:     actorPedidos(1, policy.PedidoCrear),
		PedidoID:  10, DetalleIDs: []int64{1},
	}
}

func TestDividirCuentaMueveLasLineasYSusAdicionales(t *testing.T) {
	f := fakeDivision()
	c := dividirBase()
	c.NombreCuenta = texto("  Juan  ")
	res, err := NewDividirCuenta(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía dividirse: %v %+v", err, res.Err())
	}
	if f.creaciones != 1 || f.creada.PedidoID != 10 || f.creada.Etiqueta == nil || *f.creada.Etiqueta != "Juan" {
		t.Fatalf("cuenta mal creada: %+v", f.creada)
	}
	if f.movimientos != 1 || f.destino != 21 || !slices.Equal(f.movidas, []int64{1, 3}) {
		t.Fatalf("el padre arrastra a su adicional: destino %d, movidas %v", f.destino, f.movidas)
	}
	if res.Value().ID != 21 {
		t.Fatalf("debe devolver la cuenta nueva: %+v", res.Value())
	}
}

func TestDividirCuentaSinNombreYDeVariasCuentas(t *testing.T) {
	f := fakeDivision()
	f.lineas[1].CuentaID = 22
	f.cuentas = append(f.cuentas, outbound.CuentaRef{ID: 22, PedidoID: 10, Estado: "ABIERTA"})
	c := dividirBase()
	c.DetalleIDs = []int64{2, 1}
	res, err := NewDividirCuenta(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() || f.creada.Etiqueta != nil || !slices.Equal(f.movidas, []int64{1, 2, 3}) {
		t.Fatalf("líneas de dos cuentas a una nueva: %v %+v %+v %v", err, res.Err(), f.creada, f.movidas)
	}
}

func TestDividirCuentaRechazos(t *testing.T) {
	otraSede := appctx.NewActor(7, "ana", 1, []appctx.Grant{
		{SucursalID: 1, Permiso: policy.PedidoVer}, {SucursalID: 2, Permiso: policy.PedidoCrear}})
	casos := []struct {
		nombre string
		mod    func(f *divisionFake, c *cmd.DividirCuenta)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *divisionFake, c *cmd.DividirCuenta) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *divisionFake, c *cmd.DividirCuenta) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"permiso solo en otra sede", func(_ *divisionFake, c *cmd.DividirCuenta) { c.Actor = otraSede }, apperror.KindForbidden, "sede_no_autorizada"},
		{"nombre demasiado largo", func(_ *divisionFake, c *cmd.DividirCuenta) { c.NombreCuenta = texto(strings.Repeat("x", 81)) }, apperror.KindValidation, "nombre_cuenta_invalido"},
		{"selección vacía", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = nil }, apperror.KindValidation, "seleccion_vacia"},
		{"pedido inexistente", func(f *divisionFake, _ *cmd.DividirCuenta) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *divisionFake, _ *cmd.DividirCuenta) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado", func(f *divisionFake, _ *cmd.DividirCuenta) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"línea repetida", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = []int64{2, 2} }, apperror.KindValidation, "detalle_duplicado"},
		{"línea que no es del pedido", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = []int64{99} }, apperror.KindNotFound, "no_encontrado"},
		{"línea anulada", func(f *divisionFake, c *cmd.DividirCuenta) {
			f.lineas[1].Estado = "ANULADO"
			c.DetalleIDs = []int64{2}
		}, apperror.KindConflict, "linea_anulada"},
		{"adicional sin su padre", func(_ *divisionFake, c *cmd.DividirCuenta) { c.DetalleIDs = []int64{3} }, apperror.KindValidation, "adicional_sin_padre"},
		{"cuenta de origen cerrada", func(f *divisionFake, _ *cmd.DividirCuenta) { f.cuentas[0].Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"cuenta de origen con pago confirmado", func(f *divisionFake, _ *cmd.DividirCuenta) { f.conPago = []int64{20} }, apperror.KindConflict, "cuenta_con_pago"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeDivision()
			in := dividirBase()
			tc.mod(f, &in)
			res, err := NewDividirCuenta(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.creaciones != 0 || f.movimientos != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}
```

- [ ] **Step 3: Ejecutar y comprobar que falla**

Run: `go test ./internal/modules/atencion/application/usecase/command/ -run TestDividirCuenta -count=1`
Expected: FAIL de compilación: `undefined: NewDividirCuenta`.

- [ ] **Step 4: Implementar el caso de uso**

`internal/modules/atencion/application/usecase/command/cuentas.go`:

```go
package command

import (
	"context"
	"errors"
	"slices"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoDividir = "no tiene permiso para dividir cuentas"
	maxEtiquetaCuenta        = 80
)

func cuentaConPago() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "cuenta_con_pago",
		"la cuenta tiene pagos confirmados: sus consumos ya no se pueden cambiar")
}

func lineaAnulada() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "linea_anulada", "la línea está anulada")
}

// rechazoSeleccion traduce los rechazos de la política de selección; nil si el error no es uno.
func rechazoSeleccion(err error) *apperror.AppError {
	switch {
	case errors.Is(err, policy.ErrSeleccionVacia):
		return mapper.Validacion("seleccion_vacia", "indique al menos una línea para mover")
	case errors.Is(err, policy.ErrLineaRepetida):
		return mapper.Validacion("detalle_duplicado", "una línea se repite en la selección")
	case errors.Is(err, policy.ErrLineaInexistente):
		return mapper.NoEncontrado("línea")
	case errors.Is(err, policy.ErrLineaAnulada):
		return lineaAnulada()
	case errors.Is(err, policy.ErrAdicionalSinPadre):
		return mapper.Validacion("adicional_sin_padre", "un adicional viaja con la línea de la que cuelga: seleccione también esa línea")
	}
	return nil
}

// DividirCuenta mueve líneas activas a una cuenta nueva del mismo pedido (RES-RF-011). No toca lo
// enviado a cocina: las comandas cuelgan de la línea, no de la cuenta.
type DividirCuenta struct{ cuentas outbound.DivisionStore }

func NewDividirCuenta(c outbound.DivisionStore) *DividirCuenta { return &DividirCuenta{c} }

func (uc *DividirCuenta) Execute(ctx context.Context, c cmd.DividirCuenta) (appresult.Result[dtoresult.Cuenta], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoDividir, policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.Cuenta](ae)
	}
	etiqueta, ae := textoOpcional(c.NombreCuenta, maxEtiquetaCuenta, false, "nombre_cuenta_invalido",
		"el nombre de la cuenta admite hasta 80 caracteres en una línea")
	if ae != nil {
		return rechazar[dtoresult.Cuenta](ae)
	}
	if len(c.DetalleIDs) == 0 {
		return rechazar[dtoresult.Cuenta](rechazoSeleccion(policy.ErrSeleccionVacia))
	}

	_, ae, err := pedidoAutorizado(ctx, uc.cuentas, c.Actor, c.PedidoID, policy.PedidoCrear)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Cuenta](ae)
	}

	lineas, err := uc.cuentas.LineasDelPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("leer las líneas del pedido", err)
	}
	vista := make([]policy.LineaPedido, len(lineas))
	cuentaDe := make(map[int64]int64, len(lineas))
	for i, l := range lineas {
		vista[i] = policy.LineaPedido{ID: l.ID, PadreID: l.PadreID, CuentaID: l.CuentaID, Activa: l.Estado == estadoActivo}
		cuentaDe[l.ID] = l.CuentaID
	}
	movidas, err := policy.SeleccionDeDivision(c.DetalleIDs, vista)
	if err != nil {
		if ae := rechazoSeleccion(err); ae != nil {
			return rechazar[dtoresult.Cuenta](ae)
		}
		return tecnico[dtoresult.Cuenta]("resolver la selección", err)
	}

	cuentas, err := uc.cuentas.CuentasDelPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("leer las cuentas del pedido", err)
	}
	estado := make(map[int64]string, len(cuentas))
	for _, cu := range cuentas {
		estado[cu.ID] = cu.Estado
	}
	var origen []int64
	for _, id := range movidas {
		if cu := cuentaDe[id]; !slices.Contains(origen, cu) {
			origen = append(origen, cu)
		}
	}
	slices.Sort(origen)
	for _, cu := range origen {
		if estado[cu] != estadoCuentaAbta {
			return rechazar[dtoresult.Cuenta](cuentaCerrada())
		}
	}
	conPago, err := uc.cuentas.CuentasConPago(ctx, origen)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("buscar pagos de las cuentas", err)
	}
	if len(conPago) > 0 {
		return rechazar[dtoresult.Cuenta](cuentaConPago())
	}

	id, err := uc.cuentas.CrearCuenta(ctx, outbound.NuevaCuenta{PedidoID: c.PedidoID, Etiqueta: etiqueta})
	if err != nil {
		return deAlmacen[dtoresult.Cuenta]("crear la cuenta", err)
	}
	if err := uc.cuentas.MoverLineas(ctx, c.PedidoID, id, movidas); err != nil {
		return deAlmacen[dtoresult.Cuenta]("mover las líneas", err)
	}
	cuenta, err := uc.cuentas.CuentaConLineas(ctx, c.PedidoID, id)
	if err != nil {
		return tecnico[dtoresult.Cuenta]("leer la cuenta nueva", err)
	}
	return appresult.Ok(cuenta), nil
}
```

- [ ] **Step 5: Formatear, ejecutar y comprobar que pasa**

Run: `gofmt -w internal && go vet ./internal/modules/atencion/... && go test ./internal/modules/atencion/... -count=1`
Expected: PASS en todo el módulo.

- [ ] **Step 6: Commit**

```bash
git add internal
git commit -m "I3 rebanada 3: caso de uso DividirCuenta con puertos y pruebas" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Dividir cuenta de punta a punta — persistencia, HTTP, cableado y pruebas de integración

**Files:**
- Create: `internal/modules/atencion/infrastructure/persistence/write/repository/cuentas_sql.go`
- Modify: `internal/modules/atencion/infrastructure/persistence/write/adapter/pedidos.go`
- Create: `internal/modules/atencion/web/dto/request/cuentas.go`
- Modify: `internal/modules/atencion/web/mapper/pedidos_web_mapper.go`
- Modify: `internal/modules/atencion/web/controller/pedidos_handler.go`, `routes.go`
- Modify: `internal/modules/atencion/infrastructure/config/pedidos.go`
- Modify: `tests/integration/pedidos_helpers_test.go`
- Create: `tests/integration/pedidos_r3_helpers_test.go`, `tests/integration/cuentas_division_test.go`

**Interfaces:**
- Consumes: Task 4 (`outbound.DivisionStore`, `command.NewDividirCuenta`, `inbound.DividirCuenta`).
- Produces:
  - Métodos de `*PedidoSQLWriter`: `GetLineasDelPedido`, `GetCuentasForUpdate`, `GetCuentasConPago`, `InsertCuenta`, `MoveLineas`, `GetCuentaConLineas`; y en `*PedidoWriteAdapter`: `LineasDelPedido`, `CuentasDelPedido`, `CuentasConPago`, `CrearCuenta`, `MoverLineas`, `CuentaConLineas` (con `var _ outbound.DivisionStore = (*PedidoWriteAdapter)(nil)`).
  - `webmapper.ToCuenta(dtoresult.Cuenta) response.Cuenta`.
  - Ruta `POST /pedidos/{id}/dividir-cuenta` → 200 con la cuenta nueva y sus líneas.
  - `NewPedidosHandler(l, o, a, ag, e, d)`: se añade el puerto `inbound.DividirCuenta` como último parámetro (la Task 7 añade dos más).
  - Helpers de pruebas (las Tasks 7 y 8 los reutilizan): `supervisorPedidos`, `agregarOK`, `pagoConfirmadoSQL`, `dividirCuenta`, `sumaImportes`; y el campo `Etiqueta` en `cuentaResp`.

- [ ] **Step 1: Escribir los helpers y las pruebas de integración (fallarán: la ruta no existe)**

En `tests/integration/pedidos_helpers_test.go` (Edit), añadir `Etiqueta` a `cuentaResp`. `old_string`:

```go
type cuentaResp struct {
	ID       string      `json:"id"`
	Numero   int         `json:"numero"`
	Estado   string      `json:"estado"`
```

`new_string`:

```go
type cuentaResp struct {
	ID       string      `json:"id"`
	Numero   int         `json:"numero"`
	Etiqueta *string     `json:"etiqueta"`
	Estado   string      `json:"estado"`
```

`tests/integration/pedidos_r3_helpers_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// supervisorPedidos: ver, crear, editar y anular pedidos y autorizar descuentos en PRINCIPAL y las
// sedes dadas. Es lo que el cajero real recibe tras las migraciones 00011 y 00013.
func supervisorPedidos(t *testing.T, sedes ...string) string {
	usuario := "ped.supervisor"
	if len(sedes) > 0 {
		usuario += "." + strings.Join(sedes, "+")
	}
	return tokenPedidos(t, usuario, "T_PED_SUPERVISOR",
		[]string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR", "PEDIDO_ANULAR", "DESCUENTO_AUTORIZAR"}, sedes...)
}

// agregarOK agrega una línea y la devuelve ya decodificada.
func agregarOK(t *testing.T, tk, pedido, cuenta, producto, cantidad, extras string) lineaResp {
	t.Helper()
	r := agregarConsumo(t, tk, pedido, cuenta, producto, cantidad, extras)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	return l
}

// pagoConfirmadoSQL registra, como sistema, un pago EFECTIVO confirmado sobre la cuenta. Abre antes
// una caja y una sesión propias; el monto no puede superar el saldo de la cuenta.
func pagoConfirmadoSQL(t *testing.T, pedido, cuenta, monto string) {
	t.Helper()
	cod := "CJ" + sufijo()
	asSystem(t,
		fmt.Sprintf(`INSERT INTO sch_restaurante.cajas(sucursal_id,codigo,nombre)
			SELECT p.sucursal_id,'%s','Caja %s' FROM sch_restaurante.pedidos p WHERE p.id=%s`, cod, cod, pedido),
		fmt.Sprintf(`INSERT INTO sch_restaurante.sesiones_caja(caja_id,sucursal_id,moneda,abierto_por)
			SELECT c.id,c.sucursal_id,(SELECT moneda FROM sch_restaurante.pedidos WHERE id=%s),
			       (SELECT id FROM sch_restaurante.usuarios WHERE es_sistema)
			FROM sch_restaurante.cajas c WHERE c.codigo='%s'`, pedido, cod),
		fmt.Sprintf(`INSERT INTO sch_restaurante.pagos(cuenta_id,sesion_caja_id,metodo_pago_id,monto,efectivo_recibido,estado,clave_idempotencia)
			SELECT %s, s.id, m.id, %s, %s, 'CONFIRMADO', gen_random_uuid()
			FROM sch_restaurante.sesiones_caja s JOIN sch_restaurante.cajas c ON c.id=s.caja_id,
			     sch_restaurante.metodos_pago m
			WHERE c.codigo='%s' AND m.codigo='EFECTIVO'`, cuenta, monto, monto, cod))
}

// dividirCuenta envía la división de las líneas dadas; extras se añade al cuerpo (p. ej.
// `,"nombre_cuenta":"Juan"`). Devuelve la respuesta cruda.
func dividirCuenta(t *testing.T, tk, pedido string, ids []string, extras string) resp {
	t.Helper()
	citados := make([]string, len(ids))
	for i, id := range ids {
		citados[i] = fmt.Sprintf("%q", id)
	}
	cuerpo := `{"detalle_ids":[` + strings.Join(citados, ",") + `]` + extras + `}`
	return post(t, "/api/v1/pedidos/"+pedido+"/dividir-cuenta", cuerpo, tk)
}

// sumaImportes devuelve la suma de importe_total de las líneas activas del pedido.
func sumaImportes(t *testing.T, pedido string) string {
	t.Helper()
	return scalarStr(t, `SELECT coalesce(sum(importe_total),0)::text FROM sch_restaurante.detalles_pedido
		WHERE pedido_id=$1::text::bigint AND estado='ACTIVO'`, pedido)
}
```

`tests/integration/cuentas_division_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// RES-CP-011: la suma de las cuentas antes y después de dividir coincide.
func TestDividirCuentaConservaLaSuma(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	agregarOK(t, tk, p.ID, cuenta, prod, "1", "")
	l2 := agregarOK(t, tk, p.ID, cuenta, prod, "2", "")
	agregarOK(t, tk, p.ID, cuenta, prod, "3", "")
	antes := sumaImportes(t, p.ID)
	if antes != "60.00" {
		t.Fatalf("suma inicial %s", antes)
	}

	r := dividirCuenta(t, tk, p.ID, []string{l2.ID}, `,"nombre_cuenta":" Juan "`)
	esperar(t, r, http.StatusOK, "")
	var nueva cuentaResp
	r.json(t, &nueva)
	if nueva.Numero != 2 || nueva.Etiqueta == nil || *nueva.Etiqueta != "Juan" || nueva.Estado != "ABIERTA" ||
		len(nueva.Lineas) != 1 || nueva.Lineas[0].ID != l2.ID || nueva.Total != "20.00" {
		t.Fatalf("cuenta nueva: %+v", nueva)
	}

	g := obtenerPedido(t, tk, p.ID)
	if len(g.Cuentas) != 2 || g.Cuentas[0].Total != "40.00" || g.Cuentas[1].Total != "20.00" || len(g.Cuentas[0].Lineas) != 2 {
		t.Fatalf("cuentas tras dividir: %+v", g.Cuentas)
	}
	if despues := sumaImportes(t, p.ID); despues != antes {
		t.Fatalf("la suma cambió: %s → %s", antes, despues)
	}
}

func TestDividirCuentaArrastraLosAdicionales(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	plato := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	extra := productoVentaSQL(t, "PRINCIPAL", "ADICIONAL", "2.00", true)
	padre := agregarOK(t, tk, p.ID, cuenta, plato, "1", "")
	hijo := agregarOK(t, tk, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID))
	otro := agregarOK(t, tk, p.ID, cuenta, plato, "1", "")

	esperar(t, dividirCuenta(t, tk, p.ID, []string{hijo.ID}, ""), http.StatusUnprocessableEntity, "adicional_sin_padre")

	r := dividirCuenta(t, tk, p.ID, []string{padre.ID}, "")
	esperar(t, r, http.StatusOK, "")
	var nueva cuentaResp
	r.json(t, &nueva)
	if len(nueva.Lineas) != 2 || nueva.Total != "12.00" {
		t.Fatalf("el adicional debe viajar con su padre: %+v", nueva)
	}
	g := obtenerPedido(t, tk, p.ID)
	if len(g.Cuentas[0].Lineas) != 1 || g.Cuentas[0].Lineas[0].ID != otro.ID {
		t.Fatalf("la cuenta de origen conserva solo la otra línea: %+v", g.Cuentas[0])
	}
}

// La cocina no se entera de la división: la comanda cuelga de la línea, no de la cuenta.
func TestDividirCuentaConservaLaTrazabilidadDeCocina(t *testing.T) {
	e := nuevoEscenario(t, "2")
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})

	r := dividirCuenta(t, e.mesero, e.pedido.ID, []string{e.linea.ID}, "")
	esperar(t, r, http.StatusOK, "")

	if cuenta := scalarStr(t, `SELECT cuenta_id::text FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, e.linea.ID); cuenta == e.cuenta {
		t.Fatal("la línea debió cambiar de cuenta")
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint AND estado='PENDIENTE'`, c.Lineas[0].ID); n != 1 {
		t.Fatalf("la línea de comanda debe seguir intacta (%d)", n)
	}
	if got := colaDe(t, e.cocinero, url.Values{"estacion_id": {e.estacion}}).ids(); len(got) != 1 || got[0] != c.ID {
		t.Fatalf("la cola de cocina no cambia: %v", got)
	}
}

func TestDividirCuentaRechazos(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	l1 := agregarOK(t, tk, p.ID, cuenta, prod, "2", "")
	otro := abrirLlevar(t, tk, "PRINCIPAL")
	ajena := agregarOK(t, tk, otro.ID, otro.Cuentas[0].ID, prod, "1", "")

	esperar(t, dividirCuenta(t, lectorPedidos(t), p.ID, []string{l1.ID}, ""), http.StatusForbidden, "sin_permiso")
	esperar(t, dividirCuenta(t, tk, p.ID, nil, ""), http.StatusUnprocessableEntity, "seleccion_vacia")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{l1.ID, l1.ID}, ""), http.StatusUnprocessableEntity, "detalle_duplicado")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{ajena.ID}, ""), http.StatusNotFound, "no_encontrado")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{"abc"}, ""), http.StatusBadRequest, "detalle_ids_invalido")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{l1.ID}, `,"nombre_cuenta":"`+strings.Repeat("x", 81)+`"`),
		http.StatusUnprocessableEntity, "nombre_cuenta_invalido")

	pagoConfirmadoSQL(t, p.ID, cuenta, "1.00")
	esperar(t, dividirCuenta(t, tk, p.ID, []string{l1.ID}, ""), http.StatusConflict, "cuenta_con_pago")
	if n := len(obtenerPedido(t, tk, p.ID).Cuentas); n != 1 {
		t.Fatalf("un rechazo no debe crear cuentas (%d)", n)
	}
}

func TestDividirCuentaEsIdempotente(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	l := agregarOK(t, tk, p.ID, p.Cuentas[0].ID, prod, "1", "")
	ruta := "/api/v1/pedidos/" + p.ID + "/dividir-cuenta"
	cuerpo := fmt.Sprintf(`{"detalle_ids":[%q]}`, l.ID)
	h := map[string]string{"Idempotency-Key": nuevaClave()}

	r1 := callH(t, "POST", ruta, cuerpo, tk, h)
	esperar(t, r1, http.StatusOK, "")
	r2 := callH(t, "POST", ruta, cuerpo, tk, h)
	esperar(t, r2, http.StatusOK, "")
	var c1, c2 cuentaResp
	r1.json(t, &c1)
	r2.json(t, &c2)
	if c1.ID != c2.ID {
		t.Fatalf("repetir la clave devuelve la misma cuenta: %s y %s", c1.ID, c2.ID)
	}
	if n := len(obtenerPedido(t, tk, p.ID).Cuentas); n != 2 {
		t.Fatalf("debe existir una sola cuenta nueva (%d cuentas)", n)
	}
}
```

Run: `go test ./tests/integration/ -run "TestDividirCuenta" -count=1`
Expected: FAIL (404: la ruta `POST /pedidos/{id}/dividir-cuenta` no existe).

- [ ] **Step 2: SQL de escritura y adaptador**

`internal/modules/atencion/infrastructure/persistence/write/repository/cuentas_sql.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
)

// GetLineasDelPedido bloquea FOR UPDATE todas las líneas del pedido, por id ascendente.
func (r *PedidoSQLWriter) GetLineasDelPedido(ctx context.Context, pedidoID int64) ([]outbound.LineaCuenta, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, cuenta_id, detalle_padre_id, estado
		FROM sch_restaurante.detalles_pedido WHERE pedido_id = $1 ORDER BY id FOR UPDATE`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outbound.LineaCuenta
	for rows.Next() {
		var l outbound.LineaCuenta
		var padre sql.NullInt64
		if err := rows.Scan(&l.ID, &l.CuentaID, &padre, &l.Estado); err != nil {
			return nil, err
		}
		if padre.Valid {
			v := padre.Int64
			l.PadreID = &v
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetCuentasForUpdate bloquea las cuentas del pedido, por id ascendente.
func (r *PedidoSQLWriter) GetCuentasForUpdate(ctx context.Context, pedidoID int64) ([]outbound.CuentaRef, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, pedido_id, estado
		FROM sch_restaurante.cuentas WHERE pedido_id = $1 ORDER BY id FOR UPDATE`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outbound.CuentaRef
	for rows.Next() {
		var c outbound.CuentaRef
		if err := rows.Scan(&c.ID, &c.PedidoID, &c.Estado); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCuentasConPago devuelve, de las cuentas dadas, las que tienen algún pago CONFIRMADO.
func (r *PedidoSQLWriter) GetCuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT cuenta_id FROM sch_restaurante.pagos
		WHERE cuenta_id = ANY($1::bigint[]) AND estado = 'CONFIRMADO' ORDER BY cuenta_id`, cuentaIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// InsertCuenta crea la cuenta con el siguiente número del pedido. Es seguro sin carreras porque el
// pedido ya está bloqueado FOR UPDATE por el comando.
func (r *PedidoSQLWriter) InsertCuenta(ctx context.Context, n outbound.NuevaCuenta) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.cuentas (pedido_id, numero, etiqueta)
		SELECT $1::bigint, COALESCE(max(numero), 0) + 1, $2::text
		FROM sch_restaurante.cuentas WHERE pedido_id = $1
		RETURNING id`, n.PedidoID, n.Etiqueta).Scan(&id)
	return id, Translate(err)
}

// MoveLineas reasigna las líneas a la cuenta. Si no se mueven todas es un error técnico: revierte.
func (r *PedidoSQLWriter) MoveLineas(ctx context.Context, pedidoID, cuentaID int64, lineaIDs []int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE sch_restaurante.detalles_pedido SET cuenta_id = $2
		WHERE pedido_id = $1 AND id = ANY($3::bigint[])`, pedidoID, cuentaID, lineaIDs)
	if err != nil {
		return Translate(err)
	}
	if n, err := res.RowsAffected(); err != nil || n != int64(len(lineaIDs)) {
		return fmt.Errorf("se movieron %d líneas de %d (%v)", n, len(lineaIDs), err)
	}
	return nil
}

// GetCuentaConLineas lee la cuenta con sus líneas y totales a partir del pedido completo.
func (r *PedidoSQLWriter) GetCuentaConLineas(ctx context.Context, pedidoID, cuentaID int64) (dtoresult.Cuenta, error) {
	d, found, err := cargar.Pedido(ctx, r.db, pedidoID)
	if err != nil {
		return dtoresult.Cuenta{}, err
	}
	if found {
		for _, c := range d.Cuentas {
			if c.ID == cuentaID {
				return c, nil
			}
		}
	}
	return dtoresult.Cuenta{}, sql.ErrNoRows
}
```

En `internal/modules/atencion/infrastructure/persistence/write/adapter/pedidos.go`, añadir al final del archivo:

```go

var _ outbound.DivisionStore = (*PedidoWriteAdapter)(nil)

func (a *PedidoWriteAdapter) LineasDelPedido(ctx context.Context, pedidoID int64) ([]outbound.LineaCuenta, error) {
	return a.repo.GetLineasDelPedido(ctx, pedidoID)
}
func (a *PedidoWriteAdapter) CuentasDelPedido(ctx context.Context, pedidoID int64) ([]outbound.CuentaRef, error) {
	return a.repo.GetCuentasForUpdate(ctx, pedidoID)
}
func (a *PedidoWriteAdapter) CuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error) {
	return a.repo.GetCuentasConPago(ctx, cuentaIDs)
}
func (a *PedidoWriteAdapter) CrearCuenta(ctx context.Context, n outbound.NuevaCuenta) (int64, error) {
	return a.repo.InsertCuenta(ctx, n)
}
func (a *PedidoWriteAdapter) MoverLineas(ctx context.Context, pedidoID, cuentaID int64, lineaIDs []int64) error {
	return a.repo.MoveLineas(ctx, pedidoID, cuentaID, lineaIDs)
}
func (a *PedidoWriteAdapter) CuentaConLineas(ctx context.Context, pedidoID, cuentaID int64) (dtoresult.Cuenta, error) {
	return a.repo.GetCuentaConLineas(ctx, pedidoID, cuentaID)
}
```

- [ ] **Step 3: Capa web, cableado y ruta**

`internal/modules/atencion/web/dto/request/cuentas.go`:

```go
package request

// CuentaDivision es el cuerpo de dividir-cuenta (esquema DividirCuenta del contrato). Los ids viajan
// como cadenas.
type CuentaDivision struct {
	DetalleIDs   []string `json:"detalle_ids"`
	NombreCuenta *string  `json:"nombre_cuenta,omitempty"`
}
```

En `internal/modules/atencion/web/mapper/pedidos_web_mapper.go` (Edit): extraer `ToCuenta`. `old_string`:

```go
	for i, c := range d.Cuentas {
		lineas := make([]response.Linea, len(c.Lineas))
		for j, l := range c.Lineas {
			lineas[j] = ToLinea(l)
		}
		out.Cuentas[i] = response.Cuenta{
			ID: id(c.ID), Numero: c.Numero, ClienteID: idOpcional(c.ClienteID), Etiqueta: c.Etiqueta, Estado: c.Estado,
			Subtotal: c.Subtotal, Impuesto: c.Impuesto, Total: c.Total, Lineas: lineas,
		}
	}
	return out
}
```

`new_string`:

```go
	for i, c := range d.Cuentas {
		out.Cuentas[i] = ToCuenta(c)
	}
	return out
}

// ToCuenta arma la cuenta con sus líneas y totales (esquema CuentaPedido del contrato).
func ToCuenta(c dtoresult.Cuenta) response.Cuenta {
	lineas := make([]response.Linea, len(c.Lineas))
	for j, l := range c.Lineas {
		lineas[j] = ToLinea(l)
	}
	return response.Cuenta{
		ID: id(c.ID), Numero: c.Numero, ClienteID: idOpcional(c.ClienteID), Etiqueta: c.Etiqueta, Estado: c.Estado,
		Subtotal: c.Subtotal, Impuesto: c.Impuesto, Total: c.Total, Lineas: lineas,
	}
}
```

En `internal/modules/atencion/web/controller/pedidos_handler.go`: (a) Edit del manejador y su constructor. `old_string`:

```go
	editar  inbound.EditarDetalle
}

func NewPedidosHandler(l inbound.ListarPedidos, o inbound.ObtenerPedido, a inbound.AbrirPedido,
	ag inbound.AgregarDetalle, e inbound.EditarDetalle) *PedidosHandler {
	return &PedidosHandler{l, o, a, ag, e}
}
```

`new_string`:

```go
	editar  inbound.EditarDetalle
	dividir inbound.DividirCuenta
}

func NewPedidosHandler(l inbound.ListarPedidos, o inbound.ObtenerPedido, a inbound.AbrirPedido,
	ag inbound.AgregarDetalle, e inbound.EditarDetalle, d inbound.DividirCuenta) *PedidosHandler {
	return &PedidosHandler{l, o, a, ag, e, d}
}
```

(b) Añadir al final del archivo:

```go

// DividirCuenta: POST /pedidos/{id}/dividir-cuenta (Idempotency-Key obligatoria)
func (h *PedidosHandler) DividirCuenta(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.CuentaDivision
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	ids := make([]int64, len(body.DetalleIDs))
	for i, s := range body.DetalleIDs {
		if ids[i], err = idCadena(s, "detalle_ids"); err != nil {
			return err
		}
	}
	in, err := intencion(r, fmt.Sprintf("POST /pedidos/%d/dividir-cuenta", pedido), body)
	if err != nil {
		return err
	}
	res, err := h.dividir.Execute(r.Context(), command.DividirCuenta{
		Intencion: in, Actor: actor, PedidoID: pedido, DetalleIDs: ids, NombreCuenta: body.NombreCuenta,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToCuenta)
}
```

En `internal/modules/atencion/web/controller/routes.go`, después de la línea `ruta("PATCH /pedidos/{id}/detalles/{detalle_id}", pedidos.EditarDetalle)` (Edit):

```go
	ruta("POST /pedidos/{id}/dividir-cuenta", pedidos.DividirCuenta)
```

En `internal/modules/atencion/infrastructure/config/pedidos.go` (Edit). `old_string`:

```go
	lector := readadapter.NewPedidoReadAdapter(db)
	return controller.NewPedidosHandler(uquery.NewListarPedidos(lector), uquery.NewObtenerPedido(lector),
		inbound.AbrirPedido(abrir), inbound.AgregarDetalle(agregar), inbound.EditarDetalle(editar))
```

`new_string`:

```go
	dividir := escritura[cmd.DividirCuenta, dtoresult.Cuenta]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.DividirCuenta, dtoresult.Cuenta] {
			return command.NewDividirCuenta(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.DividirCuenta) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewPedidoReadAdapter(db)
	return controller.NewPedidosHandler(uquery.NewListarPedidos(lector), uquery.NewObtenerPedido(lector),
		inbound.AbrirPedido(abrir), inbound.AgregarDetalle(agregar), inbound.EditarDetalle(editar),
		inbound.DividirCuenta(dividir))
```

- [ ] **Step 4: Formatear, compilar y ejecutar**

Run: `gofmt -w internal tests && go build ./... && go vet ./... && go test ./internal/... -count=1 && go test ./tests/integration/ -run "TestDividirCuenta|TestPedido|TestConsumo|TestBajarLaCantidad" -count=1 -v`
Expected: PASS en todo (las pruebas de pedidos y consumos de R1 siguen en verde).

Si `pagoConfirmadoSQL` falla con «supera el saldo pendiente», la cuenta no tiene líneas por al menos el monto: el helper se usa solo con cuentas que ya tienen consumos.

- [ ] **Step 5: Commit**

```bash
git add internal tests
git commit -m "I3 rebanada 3: POST /pedidos/{id}/dividir-cuenta, persistencia, HTTP y cableado" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Aplicación — casos de uso AnularDetalle y CancelarPedido

**Files:**
- Create: `internal/modules/atencion/application/dto/command/anulaciones.go`
- Create: `internal/modules/atencion/application/port/inbound/anulaciones.go`
- Create: `internal/modules/atencion/application/port/outbound/anulaciones.go`
- Create: `internal/modules/atencion/application/usecase/command/anulaciones.go`, `anulaciones_fake_test.go`, `anulaciones_test.go`

**Interfaces:**
- Consumes: Task 4 (`pedidoAutorizado`, `cuentaConPago`, `lineaAnulada`, `outbound.PedidoBloqueable`, `outbound.CuentaRef`, `outbound.DetalleRef`); `policy.PedidoAnular` (Task 2); `vo.TextoObligatorio`.
- Produces:
  - `cmd.AnularDetalle{Intencion; Actor; PedidoID, DetalleID int64; Motivo string}` y `cmd.CancelarPedido{Intencion; Actor; PedidoID int64; Motivo string}`.
  - `outbound.Anulacion{Motivo string; UsuarioID int64}` y `outbound.AnulacionStore` (ver abajo).
  - `inbound.AnularDetalle` (→ `Result[dtoresult.Linea]`) e `inbound.CancelarPedido` (→ `Result[dtoresult.PedidoDetalle]`).
  - `command.NewAnularDetalle(outbound.AnulacionStore)` y `command.NewCancelarPedido(outbound.AnulacionStore)`.

- [ ] **Step 1: DTO y puertos (compilan solos)**

`internal/modules/atencion/application/dto/command/anulaciones.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// AnularDetalle anula una línea activa de una cuenta abierta junto con sus adicionales. El motivo
// es obligatorio (1 a 2000 caracteres).
type AnularDetalle struct {
	Intencion
	Actor     appctx.Actor
	PedidoID  int64
	DetalleID int64
	Motivo    string
}

// CancelarPedido cancela el pedido entero: sus cuentas abiertas, sus líneas activas y lo que cocina
// aún no entregó. El motivo es obligatorio (1 a 2000 caracteres).
type CancelarPedido struct {
	Intencion
	Actor    appctx.Actor
	PedidoID int64
	Motivo   string
}
```

`internal/modules/atencion/application/port/inbound/anulaciones.go`:

```go
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type AnularDetalle interface {
	Execute(context.Context, cmd.AnularDetalle) (appresult.Result[dtoresult.Linea], error)
}
type CancelarPedido interface {
	Execute(context.Context, cmd.CancelarPedido) (appresult.Result[dtoresult.PedidoDetalle], error)
}
```

`internal/modules/atencion/application/port/outbound/anulaciones.go`:

```go
package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// Anulacion lleva el motivo ya validado y el usuario que anula.
type Anulacion struct {
	Motivo    string
	UsuarioID int64
}

// AnulacionStore escribe anulaciones y cancelaciones dentro de la transacción del comando. Los
// métodos que devuelven una referencia bloquean las filas hasta el final de la transacción.
type AnulacionStore interface {
	PedidoBloqueable
	// DetalleDelPedido lee la línea FOR UPDATE; found=false si no existe o es de otro pedido.
	DetalleDelPedido(ctx context.Context, pedidoID, detalleID int64) (DetalleRef, bool, error)
	// CuentaDelPedido lee la cuenta FOR UPDATE; found=false si no existe o es de otro pedido.
	CuentaDelPedido(ctx context.Context, pedidoID, cuentaID int64) (CuentaRef, bool, error)
	// CuentasDelPedido bloquea FOR UPDATE las cuentas del pedido, ordenadas por id.
	CuentasDelPedido(ctx context.Context, pedidoID int64) ([]CuentaRef, error)
	// CuentasConPago devuelve, de las cuentas dadas, las que tienen algún pago CONFIRMADO.
	CuentasConPago(ctx context.Context, cuentaIDs []int64) ([]int64, error)
	// CocinaEntregadaEnLinea indica si la línea o alguno de sus adicionales tiene una línea de
	// comanda ya ENTREGADO.
	CocinaEntregadaEnLinea(ctx context.Context, pedidoID, detalleID int64) (bool, error)
	// CocinaEntregadaEnPedido indica si alguna línea de comanda del pedido está ENTREGADO.
	CocinaEntregadaEnPedido(ctx context.Context, pedidoID int64) (bool, error)
	// AnularLinea anula la línea y sus adicionales activos, cancela sus líneas de comanda que aún no
	// se entregaron y devuelve la línea anulada.
	AnularLinea(ctx context.Context, pedidoID, detalleID int64, a Anulacion) (dtoresult.Linea, error)
	// CancelarPedido cancela el pedido, sus cuentas abiertas, sus líneas activas y sus líneas de
	// comanda no entregadas, y devuelve el pedido completo.
	CancelarPedido(ctx context.Context, pedidoID int64, a Anulacion) (dtoresult.PedidoDetalle, error)
}
```

Run: `go build ./...`
Expected: compila.

- [ ] **Step 2: Escribir el doble y las pruebas (fallan: no existen `NewAnularDetalle` ni `NewCancelarPedido`)**

`internal/modules/atencion/application/usecase/command/anulaciones_fake_test.go`:

```go
package command

import (
	"context"
	"slices"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
)

// anulacionFake implementa outbound.AnulacionStore con respuestas fijadas por la prueba.
type anulacionFake struct {
	pedido          outbound.PedidoRef
	pedidoExiste    bool
	detalle         outbound.DetalleRef
	detalleExiste   bool
	cuenta          outbound.CuentaRef
	cuentaExiste    bool
	cuentas         []outbound.CuentaRef
	conPago         []int64
	entregadaLinea  bool
	entregadaPedido bool

	anulaciones   int
	anulada       int64
	cancelaciones int
	motivo        outbound.Anulacion
}

var _ outbound.AnulacionStore = (*anulacionFake)(nil)

func (f *anulacionFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *anulacionFake) DetalleDelPedido(context.Context, int64, int64) (outbound.DetalleRef, bool, error) {
	return f.detalle, f.detalleExiste, nil
}
func (f *anulacionFake) CuentaDelPedido(context.Context, int64, int64) (outbound.CuentaRef, bool, error) {
	return f.cuenta, f.cuentaExiste, nil
}
func (f *anulacionFake) CuentasDelPedido(context.Context, int64) ([]outbound.CuentaRef, error) {
	return f.cuentas, nil
}
func (f *anulacionFake) CuentasConPago(_ context.Context, ids []int64) ([]int64, error) {
	var out []int64
	for _, id := range ids {
		if slices.Contains(f.conPago, id) {
			out = append(out, id)
		}
	}
	return out, nil
}
func (f *anulacionFake) CocinaEntregadaEnLinea(context.Context, int64, int64) (bool, error) {
	return f.entregadaLinea, nil
}
func (f *anulacionFake) CocinaEntregadaEnPedido(context.Context, int64) (bool, error) {
	return f.entregadaPedido, nil
}
func (f *anulacionFake) AnularLinea(_ context.Context, _, detalleID int64, a outbound.Anulacion) (dtoresult.Linea, error) {
	f.anulaciones++
	f.anulada = detalleID
	f.motivo = a
	return dtoresult.Linea{ID: detalleID, Estado: "ANULADO"}, nil
}
func (f *anulacionFake) CancelarPedido(_ context.Context, pedidoID int64, a outbound.Anulacion) (dtoresult.PedidoDetalle, error) {
	f.cancelaciones++
	f.motivo = a
	return dtoresult.PedidoDetalle{Pedido: dtoresult.Pedido{ID: pedidoID, Estado: "CANCELADO"}}, nil
}

// fakeAnulacion es un pedido abierto con dos cuentas abiertas y una línea activa (55) en la 20.
func fakeAnulacion() *anulacionFake {
	return &anulacionFake{
		pedido:       outbound.PedidoRef{ID: 10, SucursalID: 1, Estado: "REGISTRADO"},
		pedidoExiste: true,
		detalle: outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, ProductoID: 30,
			Cantidad: "2.000", PrecioUnitario: "10.000000", DescuentoTotal: "0.00", Estado: "ACTIVO"},
		detalleExiste: true,
		cuenta:        outbound.CuentaRef{ID: 20, PedidoID: 10, Estado: "ABIERTA"},
		cuentaExiste:  true,
		cuentas: []outbound.CuentaRef{
			{ID: 20, PedidoID: 10, Estado: "ABIERTA"}, {ID: 21, PedidoID: 10, Estado: "ABIERTA"}},
	}
}
```

`internal/modules/atencion/application/usecase/command/anulaciones_test.go`:

```go
package command

import (
	"context"
	"strings"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func supervisor() appctx.Actor { return actorPedidos(1, policy.PedidoVer, policy.PedidoAnular) }

// anularEnOtraSede: ve pedidos en la sede 1 pero anula solo en la 2.
func anularEnOtraSede() appctx.Actor {
	return appctx.NewActor(7, "ana", 1, []appctx.Grant{
		{SucursalID: 1, Permiso: policy.PedidoVer}, {SucursalID: 2, Permiso: policy.PedidoAnular}})
}

func anularBase() cmd.AnularDetalle {
	return cmd.AnularDetalle{
		Intencion: cmd.Intencion{Clave: claveEjemplo}, Actor: supervisor(),
		PedidoID: 10, DetalleID: 55, Motivo: "  el cliente se fue  ",
	}
}

func cancelarBase() cmd.CancelarPedido {
	return cmd.CancelarPedido{
		Intencion: cmd.Intencion{Clave: claveEjemplo}, Actor: supervisor(), PedidoID: 10, Motivo: "  cliente se retiró  ",
	}
}

func TestAnularDetalle(t *testing.T) {
	f := fakeAnulacion()
	res, err := NewAnularDetalle(f).Execute(context.Background(), anularBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía anularse: %v %+v", err, res.Err())
	}
	if f.anulaciones != 1 || f.anulada != 55 || f.motivo.Motivo != "el cliente se fue" || f.motivo.UsuarioID != 7 {
		t.Fatalf("anulación mal armada: %+v", f)
	}
	if res.Value().ID != 55 || res.Value().Estado != "ANULADO" {
		t.Fatalf("debe devolver la línea anulada: %+v", res.Value())
	}
}

func TestAnularDetalleRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *anulacionFake, c *cmd.AnularDetalle)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"sin permiso de anular", func(_ *anulacionFake, c *cmd.AnularDetalle) {
			c.Actor = actorPedidos(1, policy.PedidoVer, policy.PedidoEditar)
		}, apperror.KindForbidden, "sin_permiso"},
		{"permiso solo en otra sede", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Actor = anularEnOtraSede() }, apperror.KindForbidden, "sede_no_autorizada"},
		{"motivo vacío", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Motivo = "   " }, apperror.KindValidation, "motivo_invalido"},
		{"motivo demasiado largo", func(_ *anulacionFake, c *cmd.AnularDetalle) { c.Motivo = strings.Repeat("x", 2001) }, apperror.KindValidation, "motivo_invalido"},
		{"pedido inexistente", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"línea inexistente", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.detalleExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"línea ya anulada", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.detalle.Estado = "ANULADO" }, apperror.KindConflict, "linea_anulada"},
		{"cuenta cerrada", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.cuenta.Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"cuenta con pago confirmado", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.conPago = []int64{20} }, apperror.KindConflict, "cuenta_con_pago"},
		{"línea ya entregada", func(f *anulacionFake, _ *cmd.AnularDetalle) { f.entregadaLinea = true }, apperror.KindConflict, "linea_entregada"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeAnulacion()
			in := anularBase()
			tc.mod(f, &in)
			res, err := NewAnularDetalle(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.anulaciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}

func TestCancelarPedido(t *testing.T) {
	f := fakeAnulacion()
	res, err := NewCancelarPedido(f).Execute(context.Background(), cancelarBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía cancelarse: %v %+v", err, res.Err())
	}
	if f.cancelaciones != 1 || f.motivo.Motivo != "cliente se retiró" || f.motivo.UsuarioID != 7 {
		t.Fatalf("cancelación mal armada: %+v", f)
	}
	if res.Value().ID != 10 || res.Value().Estado != "CANCELADO" {
		t.Fatalf("debe devolver el pedido cancelado: %+v", res.Value())
	}
}

func TestCancelarPedidoRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *anulacionFake, c *cmd.CancelarPedido)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *anulacionFake, c *cmd.CancelarPedido) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"sin permiso de anular", func(_ *anulacionFake, c *cmd.CancelarPedido) {
			c.Actor = actorPedidos(1, policy.PedidoVer, policy.PedidoEditar)
		}, apperror.KindForbidden, "sin_permiso"},
		{"permiso solo en otra sede", func(_ *anulacionFake, c *cmd.CancelarPedido) { c.Actor = anularEnOtraSede() }, apperror.KindForbidden, "sede_no_autorizada"},
		{"motivo vacío", func(_ *anulacionFake, c *cmd.CancelarPedido) { c.Motivo = "" }, apperror.KindValidation, "motivo_invalido"},
		{"pedido inexistente", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido ya cerrado o cancelado", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"alguna cuenta con pago confirmado", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.conPago = []int64{21} }, apperror.KindConflict, "pedido_con_pago"},
		{"alguna línea ya entregada", func(f *anulacionFake, _ *cmd.CancelarPedido) { f.entregadaPedido = true }, apperror.KindConflict, "linea_entregada"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			f := fakeAnulacion()
			in := cancelarBase()
			tc.mod(f, &in)
			res, err := NewCancelarPedido(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), tc.kind, tc.code)
			if f.cancelaciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}
```

- [ ] **Step 3: Ejecutar y comprobar que falla**

Run: `go test ./internal/modules/atencion/application/usecase/command/ -run "TestAnularDetalle|TestCancelarPedido" -count=1`
Expected: FAIL de compilación: `undefined: NewAnularDetalle` y `undefined: NewCancelarPedido`.

- [ ] **Step 4: Implementar los casos de uso**

`internal/modules/atencion/application/usecase/command/anulaciones.go`:

```go
package command

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	"service-restaurant/internal/modules/atencion/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	mensajeSinPermisoAnular = "no tiene permiso para anular pedidos"
	maxMotivo               = 2000
)

// validarMotivo recorta el motivo y exige 1..2000 caracteres.
func validarMotivo(raw string) (string, *apperror.AppError) {
	m, err := vo.TextoObligatorio(raw, maxMotivo)
	if err != nil {
		return "", mapper.Validacion("motivo_invalido", "el motivo es obligatorio (máximo 2000 caracteres)")
	}
	return m, nil
}

func pedidoConPago() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "pedido_con_pago",
		"el pedido tiene pagos confirmados: el consumo cobrado no se anula, se registra una devolución")
}

func lineaEntregada(mensaje string) *apperror.AppError {
	return apperror.New(apperror.KindConflict, "linea_entregada", mensaje)
}

// AnularDetalle anula una línea activa y sus adicionales, y cancela en cocina lo que aún no se
// entregó (RES-RF-014). La línea se conserva con su motivo y su auditoría.
type AnularDetalle struct{ pedidos outbound.AnulacionStore }

func NewAnularDetalle(p outbound.AnulacionStore) *AnularDetalle { return &AnularDetalle{p} }

func (uc *AnularDetalle) Execute(ctx context.Context, c cmd.AnularDetalle) (appresult.Result[dtoresult.Linea], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAnular, policy.PedidoAnular); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	motivo, ae := validarMotivo(c.Motivo)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	_, ae, err := pedidoAutorizado(ctx, uc.pedidos, c.Actor, c.PedidoID, policy.PedidoAnular)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}

	det, found, err := uc.pedidos.DetalleDelPedido(ctx, c.PedidoID, c.DetalleID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar línea", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.NoEncontrado("línea"))
	}
	if det.Estado != estadoActivo {
		return rechazar[dtoresult.Linea](lineaAnulada())
	}
	cuenta, found, err := uc.pedidos.CuentaDelPedido(ctx, c.PedidoID, det.CuentaID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar cuenta", err)
	}
	if !found || cuenta.Estado != estadoCuentaAbta {
		return rechazar[dtoresult.Linea](cuentaCerrada())
	}
	conPago, err := uc.pedidos.CuentasConPago(ctx, []int64{det.CuentaID})
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pagos de la cuenta", err)
	}
	if len(conPago) > 0 {
		return rechazar[dtoresult.Linea](cuentaConPago())
	}
	entregada, err := uc.pedidos.CocinaEntregadaEnLinea(ctx, c.PedidoID, c.DetalleID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar entregas de la línea", err)
	}
	if entregada {
		return rechazar[dtoresult.Linea](lineaEntregada("la línea ya se entregó a la mesa: lo servido no se anula"))
	}

	l, err := uc.pedidos.AnularLinea(ctx, c.PedidoID, c.DetalleID, outbound.Anulacion{Motivo: motivo, UsuarioID: c.Actor.UserID})
	if err != nil {
		return deAlmacen[dtoresult.Linea]("anular línea", err)
	}
	return appresult.Ok(l), nil
}

// CancelarPedido cancela el pedido entero: cuentas abiertas, líneas activas y lo que cocina aún no
// entregó, con un solo motivo. El consumo ya cobrado no se devuelve aquí (RES-RF-014).
type CancelarPedido struct{ pedidos outbound.AnulacionStore }

func NewCancelarPedido(p outbound.AnulacionStore) *CancelarPedido { return &CancelarPedido{p} }

func (uc *CancelarPedido) Execute(ctx context.Context, c cmd.CancelarPedido) (appresult.Result[dtoresult.PedidoDetalle], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAnular, policy.PedidoAnular); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	motivo, ae := validarMotivo(c.Motivo)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	_, ae, err := pedidoAutorizado(ctx, uc.pedidos, c.Actor, c.PedidoID, policy.PedidoAnular)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar pedido", err)
	}
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}

	cuentas, err := uc.pedidos.CuentasDelPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("leer las cuentas del pedido", err)
	}
	ids := make([]int64, len(cuentas))
	for i, cu := range cuentas {
		ids[i] = cu.ID
	}
	conPago, err := uc.pedidos.CuentasConPago(ctx, ids)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar pagos del pedido", err)
	}
	if len(conPago) > 0 {
		return rechazar[dtoresult.PedidoDetalle](pedidoConPago())
	}
	entregada, err := uc.pedidos.CocinaEntregadaEnPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar entregas del pedido", err)
	}
	if entregada {
		return rechazar[dtoresult.PedidoDetalle](lineaEntregada("el pedido tiene líneas ya entregadas a la mesa: lo servido no se anula"))
	}

	d, err := uc.pedidos.CancelarPedido(ctx, c.PedidoID, outbound.Anulacion{Motivo: motivo, UsuarioID: c.Actor.UserID})
	if err != nil {
		return deAlmacen[dtoresult.PedidoDetalle]("cancelar pedido", err)
	}
	return appresult.Ok(d), nil
}
```

- [ ] **Step 5: Formatear, ejecutar y comprobar que pasa**

Run: `gofmt -w internal && go vet ./internal/modules/atencion/... && go test ./internal/modules/atencion/... -count=1`
Expected: PASS en todo el módulo.

- [ ] **Step 6: Commit**

```bash
git add internal
git commit -m "I3 rebanada 3: casos de uso AnularDetalle y CancelarPedido con puertos y pruebas" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Anular y cancelar de punta a punta — persistencia, HTTP, cableado y pruebas de integración

**Files:**
- Modify: `internal/modules/atencion/application/dto/result/pedidos.go`
- Modify: `internal/modules/atencion/infrastructure/persistence/internal/columnas/pedidos.go`
- Create: `internal/modules/atencion/infrastructure/persistence/write/repository/anulaciones_sql.go`
- Modify: `internal/modules/atencion/infrastructure/persistence/write/adapter/pedidos.go`
- Modify: `internal/modules/atencion/web/dto/response/pedidos.go`, `web/mapper/pedidos_web_mapper.go`
- Modify: `internal/modules/atencion/web/controller/pedidos_handler.go`, `routes.go`
- Modify: `internal/modules/atencion/infrastructure/config/pedidos.go`
- Modify: `tests/integration/pedidos_helpers_test.go`
- Create: `tests/integration/pedidos_anulacion_test.go`

**Interfaces:**
- Consumes: Task 6 (`outbound.AnulacionStore`, `command.NewAnularDetalle`, `command.NewCancelarPedido`, `inbound.AnularDetalle`, `inbound.CancelarPedido`); helpers de la Task 5 (`supervisorPedidos`, `agregarOK`, `pagoConfirmadoSQL`, `dividirCuenta`) y de R2 (`nuevoEscenario`, `enviarOK`, `transicionarOK`, `colaDe`).
- Produces:
  - `dtoresult.Linea` con `AnuladoEn *time.Time`, `AnuladoPor *int64`, `MotivoAnulacion *string`; `response.Linea` con `anulado_en`, `anulado_por`, `motivo_anulacion`.
  - Rutas `POST /pedidos/{id}/detalles/{detalle_id}/anulacion` y `POST /pedidos/{id}/cancelacion` → 200.
  - `NewPedidosHandler(l, o, a, ag, e, d, an, ca)`.

- [ ] **Step 1: Escribir las pruebas de integración (fallarán: las rutas no existen)**

En `tests/integration/pedidos_helpers_test.go` (Edit), dos cambios de campos:

1. En `lineaResp`. `old_string`:

```go
	Estado         string  `json:"estado"`
	Notas          *string `json:"notas_preparacion"`
}
```

`new_string`:

```go
	Estado         string  `json:"estado"`
	Notas          *string `json:"notas_preparacion"`
	MotivoAnulacion *string `json:"motivo_anulacion"`
}
```

2. En `pedidoResp`. `old_string`:

```go
	Notas             *string      `json:"notas"`
	Reparto           *repartoResp `json:"reparto"`
```

`new_string`:

```go
	Notas             *string      `json:"notas"`
	MotivoCancelacion *string      `json:"motivo_cancelacion"`
	CerradoEn         *string      `json:"cerrado_en"`
	Reparto           *repartoResp `json:"reparto"`
```

(Si algún `old_string` no coincide por espacios, leer la estructura en el archivo y añadir los mismos campos; `gofmt` alinea.)

`tests/integration/pedidos_anulacion_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// anularLinea anula una línea con motivo y devuelve la respuesta cruda.
func anularLinea(t *testing.T, tk, pedido, detalle, motivo string) resp {
	t.Helper()
	return post(t, "/api/v1/pedidos/"+pedido+"/detalles/"+detalle+"/anulacion", fmt.Sprintf(`{"motivo":%q}`, motivo), tk)
}

// cancelarPedido cancela el pedido con motivo y devuelve la respuesta cruda.
func cancelarPedido(t *testing.T, tk, pedido, motivo string) resp {
	t.Helper()
	return post(t, "/api/v1/pedidos/"+pedido+"/cancelacion", fmt.Sprintf(`{"motivo":%q}`, motivo), tk)
}

// entregarTodo lleva la línea de comanda por PREPARACION, LISTO y ENTREGADO.
func entregarTodo(t *testing.T, cocinero, lineaComanda string) {
	t.Helper()
	for _, estado := range []string{"PREPARACION", "LISTO", "ENTREGADO"} {
		transicionarOK(t, cocinero, lineaComanda, estado)
	}
}

func TestAnularLineaEnviadaCancelaSuComanda(t *testing.T) {
	e := nuevoEscenario(t, "3")
	sup := supervisorPedidos(t)
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	transicionarOK(t, e.cocinero, c.Lineas[0].ID, "PREPARACION")

	r := anularLinea(t, sup, e.pedido.ID, e.linea.ID, "  el cliente se fue  ")
	esperar(t, r, http.StatusOK, "")
	var l lineaResp
	r.json(t, &l)
	if l.Estado != "ANULADO" || l.MotivoAnulacion == nil || *l.MotivoAnulacion != "el cliente se fue" {
		t.Fatalf("línea anulada: %+v", l)
	}

	g := obtenerPedido(t, sup, e.pedido.ID)
	if g.Cuentas[0].Total != "0.00" || len(g.Cuentas[0].Lineas) != 1 || g.Cuentas[0].Lineas[0].Estado != "ANULADO" {
		t.Fatalf("la línea se conserva anulada y no suma al total: %+v", g.Cuentas[0])
	}
	if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, c.Lineas[0].ID); estado != "CANCELADO" {
		t.Fatalf("la línea de comanda debe quedar CANCELADO (%s)", estado)
	}
	if got := colaDe(t, e.cocinero, url.Values{"estacion_id": {e.estacion}}).ids(); len(got) != 0 {
		t.Fatalf("lo cancelado sale de la cola de cocina: %v", got)
	}
}

func TestAnularLineaArrastraSusAdicionales(t *testing.T) {
	sup := supervisorPedidos(t)
	p := abrirLlevar(t, sup, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	plato := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	extra := productoVentaSQL(t, "PRINCIPAL", "ADICIONAL", "2.00", true)
	padre := agregarOK(t, sup, p.ID, cuenta, plato, "1", "")
	hijo := agregarOK(t, sup, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID))
	otro := agregarOK(t, sup, p.ID, cuenta, plato, "1", "")

	esperar(t, anularLinea(t, sup, p.ID, padre.ID, "se equivocó"), http.StatusOK, "")

	g := obtenerPedido(t, sup, p.ID)
	estados := map[string]string{}
	for _, l := range g.Cuentas[0].Lineas {
		estados[l.ID] = l.Estado
	}
	if estados[padre.ID] != "ANULADO" || estados[hijo.ID] != "ANULADO" || estados[otro.ID] != "ACTIVO" || g.Cuentas[0].Total != "10.00" {
		t.Fatalf("el adicional se anula con su padre: %v total %s", estados, g.Cuentas[0].Total)
	}
}

func TestAnularLineaRechazos(t *testing.T) {
	e := nuevoEscenario(t, "2")
	sup := supervisorPedidos(t)
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	esperar(t, anularLinea(t, e.mesero, e.pedido.ID, e.linea.ID, "x"), http.StatusForbidden, "sin_permiso")
	esperar(t, anularLinea(t, sup, e.pedido.ID, e.linea.ID, "   "), http.StatusUnprocessableEntity, "motivo_invalido")
	otro := abrirLlevar(t, sup, "PRINCIPAL")
	esperar(t, anularLinea(t, sup, otro.ID, e.linea.ID, "x"), http.StatusNotFound, "no_encontrado")

	// Una línea ya anulada no se anula dos veces.
	suelta := agregarOK(t, sup, e.pedido.ID, e.cuenta, prod, "1", "")
	esperar(t, anularLinea(t, sup, e.pedido.ID, suelta.ID, "primera"), http.StatusOK, "")
	esperar(t, anularLinea(t, sup, e.pedido.ID, suelta.ID, "segunda"), http.StatusConflict, "linea_anulada")

	// Lo servido no se anula, y el rechazo no cambia nada.
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	entregarTodo(t, e.cocinero, c.Lineas[0].ID)
	esperar(t, anularLinea(t, sup, e.pedido.ID, e.linea.ID, "x"), http.StatusConflict, "linea_entregada")
	if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.detalles_pedido WHERE id=$1::text::bigint`, e.linea.ID); estado != "ACTIVO" {
		t.Fatalf("la línea entregada debe seguir ACTIVO (%s)", estado)
	}

	// Con un pago confirmado en la cuenta, la anulación se rechaza antes que por la entrega.
	pagoConfirmadoSQL(t, e.pedido.ID, e.cuenta, "1.00")
	esperar(t, anularLinea(t, sup, e.pedido.ID, e.linea.ID, "x"), http.StatusConflict, "cuenta_con_pago")
}

func TestAnulacionRespetaLasSedes(t *testing.T) {
	crearSedeSQL(t, "ANU7")
	pedido, _ := pedidoSQL(t, "ANU7")
	sup := supervisorPedidos(t) // solo PRINCIPAL
	esperar(t, cancelarPedido(t, sup, pedido, "x"), http.StatusNotFound, "no_encontrado")
	esperar(t, anularLinea(t, sup, pedido, "1", "x"), http.StatusNotFound, "no_encontrado")
}

func TestCancelarPedidoCancelaTodo(t *testing.T) {
	e := nuevoEscenario(t, "3")
	sup := supervisorPedidos(t)
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "2"})
	transicionarOK(t, e.cocinero, c.Lineas[0].ID, "PREPARACION")

	r := cancelarPedido(t, sup, e.pedido.ID, " cliente se retiró ")
	esperar(t, r, http.StatusOK, "")
	var p pedidoResp
	r.json(t, &p)
	if p.Estado != "CANCELADO" || p.MotivoCancelacion == nil || *p.MotivoCancelacion != "cliente se retiró" || p.CerradoEn == nil ||
		len(p.Cuentas) != 1 || p.Cuentas[0].Estado != "CANCELADA" || p.Cuentas[0].Total != "0.00" ||
		len(p.Cuentas[0].Lineas) != 1 || p.Cuentas[0].Lineas[0].Estado != "ANULADO" {
		t.Fatalf("pedido cancelado: %+v", p)
	}
	if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.comanda_detalles WHERE id=$1::text::bigint`, c.Lineas[0].ID); estado != "CANCELADO" {
		t.Fatalf("la línea de comanda debe quedar CANCELADO (%s)", estado)
	}
	if got := colaDe(t, e.cocinero, url.Values{"estacion_id": {e.estacion}}).ids(); len(got) != 0 {
		t.Fatalf("lo cancelado sale de la cola de cocina: %v", got)
	}
	if g := obtenerPedido(t, sup, e.pedido.ID); g.Estado != "CANCELADO" {
		t.Fatalf("el pedido sigue consultable y cancelado: %+v", g)
	}

	esperar(t, cancelarPedido(t, sup, e.pedido.ID, "otra vez"), http.StatusConflict, "pedido_cerrado")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	esperar(t, agregarConsumo(t, sup, e.pedido.ID, e.cuenta, prod, "1", ""), http.StatusConflict, "pedido_cerrado")
}

func TestCancelarPedidoLiberaLaMesa(t *testing.T) {
	sup := supervisorPedidos(t)
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	abrir := func() resp {
		return post(t, "/api/v1/pedidos",
			fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2}`, sedeID(t, "PRINCIPAL"), mesa), sup)
	}
	r := abrir()
	esperar(t, r, http.StatusCreated, "")
	var p pedidoResp
	r.json(t, &p)
	esperar(t, abrir(), http.StatusConflict, "")

	esperar(t, cancelarPedido(t, sup, p.ID, "el cliente no llegó"), http.StatusOK, "")
	esperar(t, abrir(), http.StatusCreated, "")
}

func TestCancelarPedidoRechazos(t *testing.T) {
	sup := supervisorPedidos(t)
	e := nuevoEscenario(t, "2")
	esperar(t, cancelarPedido(t, e.mesero, e.pedido.ID, "x"), http.StatusForbidden, "sin_permiso")
	esperar(t, cancelarPedido(t, sup, e.pedido.ID, ""), http.StatusUnprocessableEntity, "motivo_invalido")

	// Algo ya servido: no se cancela y no cambia nada.
	c := enviarOK(t, e.mesero, e.pedido.ID, e.estacion, [2]string{e.linea.ID, "1"})
	entregarTodo(t, e.cocinero, c.Lineas[0].ID)
	esperar(t, cancelarPedido(t, sup, e.pedido.ID, "x"), http.StatusConflict, "linea_entregada")
	if p := obtenerPedido(t, sup, e.pedido.ID); p.Estado == "CANCELADO" || p.Cuentas[0].Estado != "ABIERTA" || p.Cuentas[0].Lineas[0].Estado != "ACTIVO" {
		t.Fatalf("un rechazo no cambia nada: %+v", p)
	}

	// El consumo cobrado no se devuelve automáticamente.
	f := nuevoEscenario(t, "2")
	pagoConfirmadoSQL(t, f.pedido.ID, f.cuenta, "1.00")
	esperar(t, cancelarPedido(t, sup, f.pedido.ID, "x"), http.StatusConflict, "pedido_con_pago")
}

func TestDividirCuentaRechazaLoAnuladoYLoCancelado(t *testing.T) {
	sup := supervisorPedidos(t)
	p := abrirLlevar(t, sup, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	l := agregarOK(t, sup, p.ID, p.Cuentas[0].ID, prod, "1", "")

	esperar(t, anularLinea(t, sup, p.ID, l.ID, "x"), http.StatusOK, "")
	esperar(t, dividirCuenta(t, sup, p.ID, []string{l.ID}, ""), http.StatusConflict, "linea_anulada")

	esperar(t, cancelarPedido(t, sup, p.ID, "x"), http.StatusOK, "")
	esperar(t, dividirCuenta(t, sup, p.ID, []string{l.ID}, ""), http.StatusConflict, "pedido_cerrado")
}

func TestAnularYCancelarSonIdempotentes(t *testing.T) {
	sup := supervisorPedidos(t)
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	p := abrirLlevar(t, sup, "PRINCIPAL")
	l := agregarOK(t, sup, p.ID, p.Cuentas[0].ID, prod, "1", "")
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles/" + l.ID + "/anulacion"
	h := map[string]string{"Idempotency-Key": nuevaClave()}
	esperar(t, callH(t, "POST", ruta, `{"motivo":"x"}`, sup, h), http.StatusOK, "")
	esperar(t, callH(t, "POST", ruta, `{"motivo":"x"}`, sup, h), http.StatusOK, "")
	esperar(t, callH(t, "POST", ruta, `{"motivo":"otro"}`, sup, h), http.StatusConflict, "idempotency_key_reutilizada")

	q := abrirLlevar(t, sup, "PRINCIPAL")
	rutaC := "/api/v1/pedidos/" + q.ID + "/cancelacion"
	hc := map[string]string{"Idempotency-Key": nuevaClave()}
	esperar(t, callH(t, "POST", rutaC, `{"motivo":"x"}`, sup, hc), http.StatusOK, "")
	esperar(t, callH(t, "POST", rutaC, `{"motivo":"x"}`, sup, hc), http.StatusOK, "")
}
```

Run: `go test ./tests/integration/ -run "TestAnular|TestCancelar|TestAnulacion|TestDividirCuentaRechazaLo" -count=1`
Expected: FAIL (404: las rutas no existen).

- [ ] **Step 2: Ampliar `Linea` con los datos de la anulación**

En `internal/modules/atencion/application/dto/result/pedidos.go` (Edit). `old_string`:

```go
	ActualizadoEn    time.Time
	ActualizadoPor   int64
}

// Cuenta agrupa
```

`new_string`:

```go
	ActualizadoEn    time.Time
	ActualizadoPor   int64
	AnuladoEn        *time.Time
	AnuladoPor       *int64
	MotivoAnulacion  *string
}

// Cuenta agrupa
```

En `internal/modules/atencion/infrastructure/persistence/internal/columnas/pedidos.go` (dos Edit).

1. `old_string`: `		notas_preparacion, creado_en, creado_por, actualizado_en, actualizado_por`` seguida de salto de línea y `)`. Es decir:

```go
		notas_preparacion, creado_en, creado_por, actualizado_en, actualizado_por`
)
```

`new_string`:

```go
		notas_preparacion, creado_en, creado_por, actualizado_en, actualizado_por, anulado_en, anulado_por,
		motivo_anulacion`
)
```

2. En `ScanLinea`. `old_string`:

```go
	var padre sql.NullInt64
	var notas sql.NullString
	err := s.Scan(&l.ID, &l.PedidoID, &l.CuentaID, &l.ProductoID, &padre, &l.Descripcion, &l.Cantidad, &l.PrecioUnitario,
		&l.DescuentoTotal, &l.AfectacionIGV, &l.TasaImpuesto, &l.ImporteTotal, &l.BaseImponible, &l.Estado,
		&notas, &l.CreadoEn, &l.CreadoPor, &l.ActualizadoEn, &l.ActualizadoPor)
	l.DetallePadreID, l.NotasPreparacion = enteroNulo(padre), textoNulo(notas)
	return l, err
```

`new_string`:

```go
	var padre, anuladoPor sql.NullInt64
	var notas, motivo sql.NullString
	var anuladoEn sql.NullTime
	err := s.Scan(&l.ID, &l.PedidoID, &l.CuentaID, &l.ProductoID, &padre, &l.Descripcion, &l.Cantidad, &l.PrecioUnitario,
		&l.DescuentoTotal, &l.AfectacionIGV, &l.TasaImpuesto, &l.ImporteTotal, &l.BaseImponible, &l.Estado,
		&notas, &l.CreadoEn, &l.CreadoPor, &l.ActualizadoEn, &l.ActualizadoPor, &anuladoEn, &anuladoPor, &motivo)
	l.DetallePadreID, l.NotasPreparacion = enteroNulo(padre), textoNulo(notas)
	l.AnuladoPor, l.MotivoAnulacion = enteroNulo(anuladoPor), textoNulo(motivo)
	if anuladoEn.Valid {
		t := anuladoEn.Time
		l.AnuladoEn = &t
	}
	return l, err
```

En `internal/modules/atencion/web/dto/response/pedidos.go` (Edit). `old_string`:

```go
	ActualizadoEn    time.Time `json:"actualizado_en"`
	ActualizadoPor   string    `json:"actualizado_por"`
}

type Cuenta struct {
```

`new_string`:

```go
	ActualizadoEn    time.Time `json:"actualizado_en"`
	ActualizadoPor   string    `json:"actualizado_por"`
	AnuladoEn        *time.Time `json:"anulado_en"`
	AnuladoPor       *string    `json:"anulado_por"`
	MotivoAnulacion  *string    `json:"motivo_anulacion"`
}

type Cuenta struct {
```

En `internal/modules/atencion/web/mapper/pedidos_web_mapper.go` (Edit). `old_string`:

```go
		ActualizadoEn: l.ActualizadoEn, ActualizadoPor: id(l.ActualizadoPor),
	}
}
```

`new_string`:

```go
		ActualizadoEn: l.ActualizadoEn, ActualizadoPor: id(l.ActualizadoPor),
		AnuladoEn: l.AnuladoEn, AnuladoPor: idOpcional(l.AnuladoPor), MotivoAnulacion: l.MotivoAnulacion,
	}
}
```

- [ ] **Step 3: SQL de anulación y cancelación, y adaptador**

`internal/modules/atencion/infrastructure/persistence/write/repository/anulaciones_sql.go`:

```go
package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
)

// estadosCocinaCancelables son los estados de una línea de comanda que aún se pueden cancelar: el
// plato todavía no llegó a la mesa. ENTREGADO y CANCELADO quedan como están.
const estadosCocinaCancelables = `'PENDIENTE','PREPARACION','LISTO'`

// ExistsCocinaEntregadaEnLinea indica si la línea o alguno de sus adicionales tiene una línea de
// comanda ya ENTREGADO.
func (r *PedidoSQLWriter) ExistsCocinaEntregadaEnLinea(ctx context.Context, pedidoID, detalleID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM sch_restaurante.comanda_detalles cd
		JOIN sch_restaurante.detalles_pedido d ON d.id = cd.detalle_pedido_id
		WHERE d.pedido_id = $1 AND (d.id = $2 OR d.detalle_padre_id = $2) AND cd.estado = 'ENTREGADO')`,
		pedidoID, detalleID).Scan(&ok)
	return ok, err
}

// ExistsCocinaEntregadaEnPedido indica si alguna línea de comanda del pedido está ENTREGADO.
func (r *PedidoSQLWriter) ExistsCocinaEntregadaEnPedido(ctx context.Context, pedidoID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM sch_restaurante.comanda_detalles WHERE pedido_id = $1 AND estado = 'ENTREGADO')`,
		pedidoID).Scan(&ok)
	return ok, err
}

// UpdateAnularLinea cancela las líneas de comanda de la línea y sus adicionales que aún no se
// entregaron, anula la línea y sus adicionales activos y devuelve la línea. La línea se conserva.
func (r *PedidoSQLWriter) UpdateAnularLinea(ctx context.Context, pedidoID, detalleID int64, a outbound.Anulacion) (dtoresult.Linea, error) {
	if _, err := r.db.ExecContext(ctx, `UPDATE sch_restaurante.comanda_detalles cd
		SET estado = 'CANCELADO', motivo_cancelacion = $3
		FROM sch_restaurante.detalles_pedido d
		WHERE d.id = cd.detalle_pedido_id AND d.pedido_id = $1 AND (d.id = $2 OR d.detalle_padre_id = $2)
		  AND cd.estado IN (`+estadosCocinaCancelables+`)`, pedidoID, detalleID, a.Motivo); err != nil {
		return dtoresult.Linea{}, Translate(err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE sch_restaurante.detalles_pedido
		SET estado = 'ANULADO', anulado_en = clock_timestamp(), anulado_por = $3, motivo_anulacion = $4
		WHERE pedido_id = $1 AND (id = $2 OR detalle_padre_id = $2) AND estado = 'ACTIVO'`,
		pedidoID, detalleID, a.UsuarioID, a.Motivo); err != nil {
		return dtoresult.Linea{}, Translate(err)
	}
	l, err := columnas.ScanLinea(r.db.QueryRowContext(ctx,
		`SELECT `+columnas.Linea+` FROM sch_restaurante.detalles_pedido WHERE id = $1`, detalleID))
	return l, Translate(err)
}

// UpdateCancelarPedido cancela, en este orden (los triggers del esquema lo exigen): las líneas de
// comanda no entregadas, las líneas activas, las cuentas abiertas y por último el pedido, que queda
// cerrado y libera la mesa. Devuelve el pedido completo.
func (r *PedidoSQLWriter) UpdateCancelarPedido(ctx context.Context, pedidoID int64, a outbound.Anulacion) (dtoresult.PedidoDetalle, error) {
	pasos := []struct {
		sentencia string
		args      []any
	}{
		{`UPDATE sch_restaurante.comanda_detalles SET estado = 'CANCELADO', motivo_cancelacion = $2
			WHERE pedido_id = $1 AND estado IN (` + estadosCocinaCancelables + `)`, []any{pedidoID, a.Motivo}},
		{`UPDATE sch_restaurante.detalles_pedido
			SET estado = 'ANULADO', anulado_en = clock_timestamp(), anulado_por = $3, motivo_anulacion = $2
			WHERE pedido_id = $1 AND estado = 'ACTIVO'`, []any{pedidoID, a.Motivo, a.UsuarioID}},
		{`UPDATE sch_restaurante.cuentas SET estado = 'CANCELADA', motivo_cancelacion = $2
			WHERE pedido_id = $1 AND estado = 'ABIERTA'`, []any{pedidoID, a.Motivo}},
		{`UPDATE sch_restaurante.pedidos
			SET estado = 'CANCELADO', cerrado_en = clock_timestamp(), motivo_cancelacion = $2
			WHERE id = $1`, []any{pedidoID, a.Motivo}},
	}
	for _, p := range pasos {
		if _, err := r.db.ExecContext(ctx, p.sentencia, p.args...); err != nil {
			return dtoresult.PedidoDetalle{}, Translate(err)
		}
	}
	d, found, err := cargar.Pedido(ctx, r.db, pedidoID)
	if err != nil {
		return dtoresult.PedidoDetalle{}, err
	}
	if !found {
		return dtoresult.PedidoDetalle{}, sql.ErrNoRows
	}
	return d, nil
}
```

En `internal/modules/atencion/infrastructure/persistence/write/adapter/pedidos.go`, añadir al final:

```go

var _ outbound.AnulacionStore = (*PedidoWriteAdapter)(nil)

func (a *PedidoWriteAdapter) CocinaEntregadaEnLinea(ctx context.Context, pedidoID, detalleID int64) (bool, error) {
	return a.repo.ExistsCocinaEntregadaEnLinea(ctx, pedidoID, detalleID)
}
func (a *PedidoWriteAdapter) CocinaEntregadaEnPedido(ctx context.Context, pedidoID int64) (bool, error) {
	return a.repo.ExistsCocinaEntregadaEnPedido(ctx, pedidoID)
}
func (a *PedidoWriteAdapter) AnularLinea(ctx context.Context, pedidoID, detalleID int64, an outbound.Anulacion) (dtoresult.Linea, error) {
	return a.repo.UpdateAnularLinea(ctx, pedidoID, detalleID, an)
}
func (a *PedidoWriteAdapter) CancelarPedido(ctx context.Context, pedidoID int64, an outbound.Anulacion) (dtoresult.PedidoDetalle, error) {
	return a.repo.UpdateCancelarPedido(ctx, pedidoID, an)
}
```

- [ ] **Step 4: Capa web, cableado y rutas**

En `internal/modules/atencion/web/controller/pedidos_handler.go` (Edit del manejador y su constructor, que la Task 5 dejó con `dividir`). `old_string`:

```go
	dividir inbound.DividirCuenta
}

func NewPedidosHandler(l inbound.ListarPedidos, o inbound.ObtenerPedido, a inbound.AbrirPedido,
	ag inbound.AgregarDetalle, e inbound.EditarDetalle, d inbound.DividirCuenta) *PedidosHandler {
	return &PedidosHandler{l, o, a, ag, e, d}
}
```

`new_string`:

```go
	dividir  inbound.DividirCuenta
	anular   inbound.AnularDetalle
	cancelar inbound.CancelarPedido
}

func NewPedidosHandler(l inbound.ListarPedidos, o inbound.ObtenerPedido, a inbound.AbrirPedido,
	ag inbound.AgregarDetalle, e inbound.EditarDetalle, d inbound.DividirCuenta,
	an inbound.AnularDetalle, ca inbound.CancelarPedido) *PedidosHandler {
	return &PedidosHandler{l, o, a, ag, e, d, an, ca}
}
```

Añadir al final del mismo archivo:

```go

// AnularDetalle: POST /pedidos/{id}/detalles/{detalle_id}/anulacion (Idempotency-Key obligatoria)
func (h *PedidosHandler) AnularDetalle(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	detalle, err := pathIDDe(r, "detalle_id")
	if err != nil {
		return err
	}
	var body request.Motivo
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /pedidos/%d/detalles/%d/anulacion", pedido, detalle), body)
	if err != nil {
		return err
	}
	res, err := h.anular.Execute(r.Context(), command.AnularDetalle{
		Intencion: in, Actor: actor, PedidoID: pedido, DetalleID: detalle, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToLinea)
}

// Cancelar: POST /pedidos/{id}/cancelacion (Idempotency-Key obligatoria)
func (h *PedidosHandler) Cancelar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.Motivo
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /pedidos/%d/cancelacion", pedido), body)
	if err != nil {
		return err
	}
	res, err := h.cancelar.Execute(r.Context(), command.CancelarPedido{
		Intencion: in, Actor: actor, PedidoID: pedido, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToPedidoDetalle)
}
```

En `internal/modules/atencion/web/controller/routes.go`, después de la línea `ruta("POST /pedidos/{id}/dividir-cuenta", pedidos.DividirCuenta)` (Edit):

```go
	ruta("POST /pedidos/{id}/cancelacion", pedidos.Cancelar)
	ruta("POST /pedidos/{id}/detalles/{detalle_id}/anulacion", pedidos.AnularDetalle)
```

En `internal/modules/atencion/infrastructure/config/pedidos.go` (dos Edit).

1. Antes del lector. `old_string`: `	lector := readadapter.NewPedidoReadAdapter(db)` (única). `new_string`:

```go
	anular := escritura[cmd.AnularDetalle, dtoresult.Linea]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.AnularDetalle, dtoresult.Linea] {
			return command.NewAnularDetalle(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.AnularDetalle) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	cancelar := escritura[cmd.CancelarPedido, dtoresult.PedidoDetalle]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CancelarPedido, dtoresult.PedidoDetalle] {
			return command.NewCancelarPedido(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.CancelarPedido) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewPedidoReadAdapter(db)
```

2. `old_string`:

```go
		inbound.DividirCuenta(dividir))
```

`new_string`:

```go
		inbound.DividirCuenta(dividir), inbound.AnularDetalle(anular), inbound.CancelarPedido(cancelar))
```

- [ ] **Step 5: Formatear, compilar y ejecutar**

Run: `gofmt -w internal tests && go build ./... && go vet ./... && go test ./internal/... -count=1 && go test ./tests/integration/ -run "TestAnular|TestCancelar|TestAnulacion|TestDividirCuenta|TestPedido|TestConsumo|TestComanda|TestCola" -count=1 -v`
Expected: PASS en todo. Si alguna prueba de R1 o R2 falla por una columna nueva de `Linea`, revisar que `columnas.Linea` y `ScanLinea` quedaron con las mismas 22 columnas, en el mismo orden.

- [ ] **Step 6: Commit**

```bash
git add internal tests
git commit -m "I3 rebanada 3: anular línea y cancelar pedido con motivo, persistencia, HTTP y cableado" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Concurrencia, arquitectura y suite completa

**Files:**
- Create: `tests/integration/pedidos_cancelacion_concurrencia_test.go`

**Interfaces:**
- Consumes: helpers de la Task 5 (`supervisorPedidos`), de R2 (`nuevoEscenario`) y de R1 (`callH`, `nuevaClave`, `scalar`, `scalarStr`, `app`).

- [ ] **Step 1: Escribir la prueba de concurrencia**

`tests/integration/pedidos_cancelacion_concurrencia_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// Una cancelación y un envío a cocina simultáneos sobre el mismo pedido: ambos bloquean el pedido
// FOR UPDATE primero, así que el resultado es siempre coherente. O el envío ocurrió antes y la
// cancelación canceló su comanda, o la cancelación ocurrió antes y el envío se rechazó con 409.
func TestCancelarYEnviarSimultaneosDejanUnEstadoCoherente(t *testing.T) {
	sup := supervisorPedidos(t)
	app(t) // inicializa la aplicación antes de lanzar goroutines

	for i := 0; i < 5; i++ {
		e := nuevoEscenario(t, "3")
		cuerpo := fmt.Sprintf(`{"estacion_id":%q,"lineas":[{"detalle_pedido_id":%q,"cantidad":"2"}]}`, e.estacion, e.linea.ID)
		rutaEnvio := "/api/v1/pedidos/" + e.pedido.ID + "/comandas"
		rutaCancel := "/api/v1/pedidos/" + e.pedido.ID + "/cancelacion"

		var codEnvio, codCancel int
		var inicio, fin sync.WaitGroup
		inicio.Add(1)
		fin.Add(2)
		go func() {
			defer fin.Done()
			inicio.Wait()
			codEnvio = callH(t, "POST", rutaEnvio, cuerpo, e.mesero, map[string]string{"Idempotency-Key": nuevaClave()}).Code
		}()
		go func() {
			defer fin.Done()
			inicio.Wait()
			codCancel = callH(t, "POST", rutaCancel, `{"motivo":"carrera"}`, sup, map[string]string{"Idempotency-Key": nuevaClave()}).Code
		}()
		inicio.Done()
		fin.Wait()

		if codCancel != http.StatusOK {
			t.Fatalf("vuelta %d: la cancelación debe tener éxito (%d)", i, codCancel)
		}
		if codEnvio != http.StatusCreated && codEnvio != http.StatusConflict {
			t.Fatalf("vuelta %d: el envío da 201 o 409, no %d", i, codEnvio)
		}
		if estado := scalarStr(t, `SELECT estado FROM sch_restaurante.pedidos WHERE id=$1::text::bigint`, e.pedido.ID); estado != "CANCELADO" {
			t.Fatalf("vuelta %d: el pedido debe quedar CANCELADO (%s)", i, estado)
		}
		if n := scalar(t, `SELECT count(*) FROM sch_restaurante.comanda_detalles
			WHERE pedido_id=$1::text::bigint AND estado <> 'CANCELADO'`, e.pedido.ID); n != 0 {
			t.Fatalf("vuelta %d: no puede quedar cocina viva en un pedido cancelado (%d líneas)", i, n)
		}
		enviadas := scalar(t, `SELECT count(*) FROM sch_restaurante.comanda_detalles WHERE pedido_id=$1::text::bigint`, e.pedido.ID)
		if (codEnvio == http.StatusCreated) != (enviadas == 1) {
			t.Fatalf("vuelta %d: envío %d pero %d líneas de comanda", i, codEnvio, enviadas)
		}
	}
}
```

- [ ] **Step 2: Ejecutar la prueba, la de arquitectura y la suite completa**

Run: `go test ./tests/integration/ -run TestCancelarYEnviarSimultaneos -count=1 -v`
Expected: PASS. Si falla con un código distinto de 201/409 o deja cocina viva, hay un orden de bloqueo mal puesto: diagnosticar antes de seguir.

Run: `go test ./tests/architecture/ -count=1 -v`
Expected: PASS. Si falla con «importa internos de otro módulo» o similar, `atencion` importa algo que no debe; corregir el código, no la regla.

Run: `go vet ./... && go test ./... -count=1`
Expected: PASS en todos los paquetes (la suite de integración tarda un par de minutos).

- [ ] **Step 3: Commit**

```bash
git add tests/integration/pedidos_cancelacion_concurrencia_test.go
git commit -m "I3 rebanada 3: prueba de concurrencia de cancelar contra enviar a cocina" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Contrato y expediente

**Files:**
- Modify: `docs/planificacion/03_api/openapi.yaml`
- Modify: `docs/planificacion/01_documentacion/requisitos_y_trazabilidad.json` y `.html`, `backlog_incrementos.json`
- Modify: `docs/planificacion/CAMBIOS_1_3.txt`, `MANIFIESTO.json`, `00_LEEME.txt`, `06_calidad/validacion_entrega.json`, `SHA256SUMS.txt`
- Modify: `service-restaurant/docs/module-map.md`
- Modify: `docs/superpowers/specs/2026-10-29-i3-rebanada3-cancelar-dividir-design.md`

R3 **añade una ruta y una operación** (`POST /pedidos/{id}/detalles/{detalle_id}/anulacion`, `operacion_165`): el contrato pasa de 120 rutas y 164 operaciones a 121 y 165. Por eso, a diferencia de R2, también cambian `MANIFIESTO.json`, `00_LEEME.txt` y `06_calidad/validacion_entrega.json`.

- [ ] **Step 1: Marcar las dos operaciones existentes**

Desde `docs/planificacion/`:

```bash
awk '/operationId: operacion_(116|117)$/ {m=1} m && /x-implementado: false/ {sub(/false/,"true"); m=0} {print}' 03_api/openapi.yaml > "$TEMP/openapi.nuevo" && cp "$TEMP/openapi.nuevo" 03_api/openapi.yaml
git diff --stat 03_api/openapi.yaml
git diff -U0 03_api/openapi.yaml | grep '^[-+] '
```

Expected: exactamente 2 pares `-      x-implementado: false` / `+      x-implementado: true`. Si el diff muestra más o menos, deshacer (`git checkout -- 03_api/openapi.yaml`) y revisar el `awk`.

- [ ] **Step 2: Editar el contrato**

Con la herramienta Edit sobre `docs/planificacion/03_api/openapi.yaml` (cada `old_string` es único; no usar `replace_all`):

1. `operacion_116` (dividir cuenta): descripción **y** esquema de respuesta. `old_string`:

```
      summary: Mover líneas a nueva cuenta
      tags:
      - Atención
      description: Diseño propuesto. El backend valida permiso, sucursal, campos y estado en una transacción cuando corresponda.
        Idempotency-Key identifica la intención estable. La API debe persistir el resultado; para comandos sin columna propia
        en V4 se requiere diseño y migración adicional.
      x-implementado: true
```

`new_string`:

```
      summary: Mover líneas a nueva cuenta
      tags:
      - Atención
      description: Mueve líneas activas de las cuentas abiertas de un pedido a una cuenta nueva. Los adicionales viajan con su
        línea padre (422 adicional_sin_padre si se pide uno solo). No cambia lo enviado a cocina. Se rechaza con 409
        cuenta_con_pago si una cuenta de origen tiene pagos confirmados. Idempotency-Key obligatorio.
      x-implementado: true
```

2. La respuesta de esa misma operación pasa de `Cuentas` a `CuentaPedido` (la cuenta con sus líneas y totales). `old_string`:

```
                $ref: '#/components/schemas/Cuentas'
        '400': *id001
        '401': *id002
        '403': *id003
        '404': *id004
        '409': *id005
        '422': *id006
        '503': *id007
      parameters:
      - name: id
        in: path
        required: true
        schema:
          $ref: '#/components/schemas/Id'
      - $ref: '#/components/parameters/IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DividirCuenta'
```

`new_string`: el mismo bloque con la primera línea cambiada a `                $ref: '#/components/schemas/CuentaPedido'`.

3. `operacion_117` (cancelar pedido). `old_string`:

```
      summary: Cancelar pedido con motivo
      tags:
      - Atención
      description: Diseño propuesto. El backend valida permiso, sucursal, campos y estado en una transacción cuando corresponda.
        Idempotency-Key identifica la intención estable. La API debe persistir el resultado; para comandos sin columna propia
        en V4 se requiere diseño y migración adicional.
      x-implementado: true
```

`new_string`:

```
      summary: Cancelar pedido con motivo
      tags:
      - Atención
      description: Cancela el pedido completo con un motivo. Las cuentas abiertas pasan a CANCELADA, las líneas activas a ANULADO
        y las líneas de cocina aún no entregadas a CANCELADO; las líneas se conservan con su auditoría y la mesa queda libre.
        Se rechaza con 409 pedido_con_pago si hay pagos confirmados (la devolución se registra aparte) y con 409 linea_entregada
        si ya se sirvió algo. Idempotency-Key obligatorio.
      x-implementado: true
```

4. Ruta nueva. Insertarla justo antes de `/pedidos/{id}/dividir-cuenta:`. `old_string`:

```
  /pedidos/{id}/dividir-cuenta:
    post:
```

`new_string`:

```
  /pedidos/{id}/detalles/{detalle_id}/anulacion:
    post:
      operationId: operacion_165
      summary: Anular consumo con motivo
      tags:
      - Atención
      description: Anula una línea activa de una cuenta abierta junto con sus adicionales. Las líneas de cocina PENDIENTE,
        PREPARACION o LISTO pasan a CANCELADO con el mismo motivo; la línea se conserva como ANULADO con su auditoría. Se
        rechaza con 409 linea_entregada si ya se sirvió y con 409 cuenta_con_pago si la cuenta tiene pagos confirmados.
        Idempotency-Key obligatorio.
      x-implementado: true
      x-permiso: PEDIDO_ANULAR
      responses:
        '200':
          description: Resultado confirmado por el servicio; no implica efecto fiscal o bancario externo.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DetallesPedido'
        '400': *id001
        '401': *id002
        '403': *id003
        '404': *id004
        '409': *id005
        '422': *id006
        '503': *id007
      parameters:
      - name: id
        in: path
        required: true
        schema:
          $ref: '#/components/schemas/Id'
      - name: detalle_id
        in: path
        required: true
        schema:
          $ref: '#/components/schemas/Id'
      - $ref: '#/components/parameters/IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Motivo'
  /pedidos/{id}/dividir-cuenta:
    post:
```

5. El nombre de la cuenta admite 80 caracteres, como la columna `cuentas.etiqueta`. `old_string`:

```
        nombre_cuenta:
          type: string
          maxLength: 100
```

`new_string`:

```
        nombre_cuenta:
          type: string
          maxLength: 80
```

Run: `git diff --stat 03_api/openapi.yaml && grep -c "x-implementado: true" 03_api/openapi.yaml`
Expected: 94 (eran 91 antes de R3: 2 operaciones marcadas y 1 nueva).

- [ ] **Step 3: Script de cierre del expediente**

Crear `$TEMP/cierre-i3-r3.mjs` con este contenido y ejecutarlo desde `docs/planificacion/`:

```js
import fs from 'node:fs';
import crypto from 'node:crypto';

const leer = (f) => fs.readFileSync(f, 'utf8');
const lista = (j) => (Array.isArray(j) ? j : Object.values(j).find(Array.isArray));
const guardarJson = (f, j) => fs.writeFileSync(f, JSON.stringify(j, null, 2) + '\n');
const falla = (m) => { throw new Error(m); };
const reemplazar = (f, de, a) => {
  const t = leer(f);
  if (!t.includes(de)) falla(`${f}: no contiene «${de}»; leer el archivo y ajustar este script`);
  fs.writeFileSync(f, t.replace(de, a));
};

// ---- 1. RES-RF-011 y RES-RF-014 en requisitos_y_trazabilidad.json y .html ----
const SITUACIONES = {
  'RES-RF-011': 'Implementado en I3 (rebanada 3): POST /pedidos/{id}/dividir-cuenta mueve líneas activas a una cuenta nueva del mismo pedido (los adicionales viajan con su línea padre, la suma de importes se conserva, lo enviado a cocina no cambia y con un pago confirmado en la cuenta de origen se rechaza con 409 cuenta_con_pago). Pruebas: TestDividirCuentaConservaLaSuma, TestDividirCuentaConservaLaTrazabilidadDeCocina y TestDividirCuentaRechazos. RES-CP-011 sigue NO_EJECUTADA.',
  'RES-RF-014': 'Implementado en I3 (rebanada 3): POST /pedidos/{id}/cancelacion cancela el pedido entero y POST /pedidos/{id}/detalles/{detalle_id}/anulacion anula una línea, ambos con motivo y PEDIDO_ANULAR (asignado al CAJERO en la migración 00013). Las líneas se conservan como ANULADO con su auditoría, las cuentas abiertas quedan CANCELADA y las líneas de cocina no entregadas pasan a CANCELADO; lo ya servido (409 linea_entregada) y lo cobrado (409 pedido_con_pago o cuenta_con_pago) no se anulan: la devolución es de I4. Pruebas: TestAnularLineaEnviadaCancelaSuComanda, TestCancelarPedidoCancelaTodo, TestCancelarPedidoRechazos y TestCancelarYEnviarSimultaneosDejanUnEstadoCoherente. RES-CP-014 sigue NO_EJECUTADA.',
};
const fReq = '01_documentacion/requisitos_y_trazabilidad.json';
const req = JSON.parse(leer(fReq));
for (const [id, sit] of Object.entries(SITUACIONES)) {
  const r = lista(req).find((x) => x.id === id);
  if (!r) falla(`${id} no existe`);
  r.estado_backend = 'IMPLEMENTADO';
  r.situacion_backend = sit;
}
guardarJson(fReq, req);

const fHtml = '01_documentacion/requisitos_y_trazabilidad.html';
let html = leer(fHtml);
for (const [id, sit] of Object.entries(SITUACIONES)) {
  const a = html.indexOf(`id="${id}"`);
  const b = html.indexOf('</tbody>', a);
  if (a < 0 || b < 0) falla(`HTML sin sección ${id}`);
  let bloque = html.slice(a, b);
  const re = /<td>(PENDIENTE|PARCIAL) · prototipo /;
  if (!re.test(bloque)) falla(`${id}: el HTML no tenía el estado esperado; leer el archivo y ajustar este script`);
  bloque = bloque.replace(re, '<td>IMPLEMENTADO · prototipo ');
  bloque = bloque.replace(/<tr><td>Situación backend<\/td><td>.*?<\/td><\/tr>/, '');
  bloque += '<tr><td>Situación backend</td><td>' + sit + '</td></tr>';
  html = html.slice(0, a) + bloque + html.slice(b);
}
fs.writeFileSync(fHtml, html);

// ---- 2. backlog_incrementos.json -------------------------------------------
const fBack = '01_documentacion/backlog_incrementos.json';
const back = JSON.parse(leer(fBack));
const i3 = lista(back).find((x) => (x.id || x.codigo) === 'I3');
if (!i3) falla('I3 no existe en el backlog');
i3.estado = 'EN_CURSO';
i3.avance = 'Rebanadas 1, 2 y 3 de 4 hechas (backend): apertura de pedido y cuenta (RES-RF-009), consumos como instantánea (RES-RF-010), autorización por sede (RES-RF-002), envío de comandas por estación, cola de preparación y estados de cocina (RES-RF-012) y, en R3, dividir cuentas (RES-RF-011) y anular líneas o cancelar el pedido con motivo (RES-RF-014). Falta el frontend de Salón y Cocina con el cierre con auditoría (R4). Los casos de aceptación siguen NO_EJECUTADA.';
guardarJson(fBack, back);

// ---- 3. CAMBIOS_1_3.txt ----------------------------------------------------
const fCam = 'CAMBIOS_1_3.txt';
let cam = leer(fCam);
if (!cam.endsWith('\n')) cam += '\n';
cam +=
  '- I3 rebanada 3: dividir cuentas y cancelar con motivo (backend, módulo atencion). RES-RF-011 y RES-RF-014 pasan a\n' +
  '  IMPLEMENTADO en backend; RES-CP-011 y RES-CP-014 siguen NO_EJECUTADA. El contrato marca como implementadas\n' +
  '  POST /pedidos/{id}/dividir-cuenta y POST /pedidos/{id}/cancelacion, añade POST /pedidos/{id}/detalles/{detalle_id}/anulacion\n' +
  '  (121 rutas y 165 operaciones), cambia la respuesta de dividir-cuenta a CuentaPedido y limita nombre_cuenta a 80\n' +
  '  caracteres, como la columna. PEDIDO_ANULAR se asigna al rol CAJERO (migración 00013). Bajar la cantidad de una línea\n' +
  '  con descuento exige DESCUENTO_AUTORIZAR.\n';
fs.writeFileSync(fCam, cam);

// ---- 4. Conteos del contrato: 120/164 → 121/165 ----------------------------
reemplazar('MANIFIESTO.json', '"rutas_api_propuestas": 120,', '"rutas_api_propuestas": 121,');
reemplazar('MANIFIESTO.json', '"operaciones_api_propuestas": 164,', '"operaciones_api_propuestas": 165,');
reemplazar('00_LEEME.txt', '120 rutas y 164 operaciones', '121 rutas y 165 operaciones');
reemplazar('06_calidad/validacion_entrega.json',
  'y 120 rutas y 164 operaciones tras añadir la edición de consumos en I3; no se revalidó',
  'y 120 rutas y 164 operaciones tras añadir la edición de consumos en I3, y 121 rutas y 165 operaciones tras añadir la anulación de consumos en la rebanada 3 de I3; no se revalidó');

// ---- 5. SHA256SUMS.txt -----------------------------------------------------
const fApi = '03_api/openapi.yaml';
const tocados = new Set([fApi, fReq, fHtml, fBack, fCam, 'MANIFIESTO.json', '00_LEEME.txt', '06_calidad/validacion_entrega.json']);
const sums = leer('SHA256SUMS.txt').split('\n').map((l) => {
  const m = /^([0-9a-f]{64})  (.+)$/.exec(l);
  if (!m || !tocados.has(m[2])) return l;
  return crypto.createHash('sha256').update(fs.readFileSync(m[2])).digest('hex') + '  ' + m[2];
});
fs.writeFileSync('SHA256SUMS.txt', sums.join('\n'));
console.log('ok: RES-RF-011 y RES-RF-014 IMPLEMENTADO, contrato en 121 rutas y 165 operaciones');
```

Run (desde `docs/planificacion/`): `node "$TEMP/cierre-i3-r3.mjs"`
Expected: `ok: RES-RF-011 y RES-RF-014 IMPLEMENTADO, contrato en 121 rutas y 165 operaciones`. Si lanza un error de ancla («no existe», «no contiene», «el HTML no tenía el estado esperado»), abrir el archivo real, entender la diferencia y corregir el script; **no forzar** el cambio. Si falla a mitad, `git checkout -- docs/planificacion` y repetir los Steps 1 y 2 antes de reintentar.

- [ ] **Step 4: Validar**

Run (desde `docs/planificacion/`):

```bash
node -e 'const fs=require("fs"); for (const f of ["01_documentacion/requisitos_y_trazabilidad.json","01_documentacion/backlog_incrementos.json","MANIFIESTO.json","06_calidad/validacion_entrega.json"]) JSON.parse(fs.readFileSync(f,"utf8")); console.log("json ok")'
sha256sum -c SHA256SUMS.txt 2>&1 | grep -v ": OK"
git diff --stat .
grep -c "x-implementado: true" 03_api/openapi.yaml
git show HEAD:docs/planificacion/03_api/openapi.yaml > "$TEMP/base.yaml"
npx --yes @redocly/cli lint "$TEMP/base.yaml" 2>&1 | tail -3
npx --yes @redocly/cli lint 03_api/openapi.yaml 2>&1 | tail -3
```

Expected: `json ok`; `sha256sum` sin líneas de error; el diff muestra los archivos esperados (openapi, requisitos json y html, backlog, CAMBIOS_1_3, MANIFIESTO, 00_LEEME, validacion_entrega y SHA256SUMS); el conteo de `x-implementado: true` es 94; y el lint del contrato nuevo da **los mismos** errores y advertencias que el de `HEAD` (hoy 1 error —un `enum` con enteros que ya existía— y 15 advertencias). Si el contrato nuevo añade alguno, corregirlo. El lint es una validación real de la sintaxis y las referencias del contrato.

- [ ] **Step 5: Documentación del repositorio**

En `service-restaurant/docs/module-map.md` (dos Edit):

1. En el párrafo «Módulos hoy», la descripción de `atencion`. `old_string`: `pedidos, cuentas y consumos, implementado en I3 rebanada 1)`. `new_string`: `pedidos, cuentas y consumos, implementado en I3 rebanada 1; dividir cuentas y anular o cancelar con motivo, implementado en I3 rebanada 3)`.
2. En la tabla, fila «Atención». `old_string`: `permisos en `00011`) · reservas (I6) | I2 / I3 (reservas: I6) |`. `new_string`: `permisos en `00011` y `00013`) · reservas (I6) | I2 / I3 (reservas: I6) |`.

En `docs/superpowers/specs/2026-10-29-i3-rebanada3-cancelar-dividir-design.md`, añadir al final:

```markdown

## 8. Ajustes durante el plan

Al escribir el plan (`docs/superpowers/plans/2026-10-29-i3-rebanada3-cancelar-dividir.md`) se reconcilió este diseño con el código y el contrato existentes. Las diferencias con las secciones anteriores son:

| Sección | Diseño | Implementado | Motivo |
|---|---|---|---|
| §3.4 | El total ignora las líneas anuladas; afecta a la lectura de pedidos de R1 | Sin cambio en los totales; `Linea` añade `anulado_en`, `anulado_por` y `motivo_anulacion` | `leerTotales` ya filtra `estado = 'ACTIVO'`; faltaba devolver el motivo |
| §3.5 | D4 responde 403 `sin_permiso_descuento` | 403 `sin_permiso`, el rechazo que ya emiten `AgregarDetalle` y `EditarDetalle` | Coherencia del contrato de errores |
| §3.1 | `nombre_cuenta` según el contrato admite 100 caracteres | Máximo 80 y se corrige el contrato | `cuentas.etiqueta` es `VARCHAR(80)` |
| §3.1 | La respuesta de dividir es `Cuentas` | Es `CuentaPedido` (cuenta con líneas y totales) | `Cuentas` exige `pedido_id` y metadatos que la API de pedidos no devuelve |
| §4 | Regla de estados de cocina cancelables en `domain` | Constante `estadosCocinaCancelables` en la persistencia | Es el filtro de la sentencia `UPDATE`; duplicarla en dominio no la haría más segura |
| §3.1 | Rechazo por pago confirmado | Comprobación en la aplicación (409 `cuenta_con_pago`); el trigger `a_guardar_detalle` queda como segunda defensa | Código propio en lugar del mensaje genérico del trigger |

Orden de las sentencias de cancelación: líneas de comanda, líneas de pedido, cuentas y, al final, el pedido. Los triggers del esquema lo exigen (una cuenta con líneas activas no se cancela y un pedido con cuentas abiertas no se cierra).
```

- [ ] **Step 6: Última verificación y commit**

Run (desde `service-restaurant/`): `go vet ./... && go test ./... -count=1`
Expected: PASS en todo. Esta es la evidencia que se reporta; no declarar la rebanada terminada sin ella.

```bash
git add docs service-restaurant/docs
git commit -m "I3 rebanada 3: contrato y expediente de dividir cuentas y cancelar con motivo (RF-011, RF-014)" -m "Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

## Autorrevisión del plan contra el spec

| Spec | Task |
|---|---|
| §1 migración `00013` (`PEDIDO_ANULAR` a CAJERO), sin cambios de esquema | 1 |
| §2 `POST /pedidos/{id}/dividir-cuenta` | 4 (caso de uso), 5 (HTTP e integración) |
| §2 `POST /pedidos/{id}/detalles/{detalle_id}/anulacion` y `POST /pedidos/{id}/cancelacion` | 6 (casos de uso), 7 (HTTP e integración) |
| §3.1 reglas 1–8 de dividir (bloqueo, estados, selección, adicionales, pago confirmado, cuenta nueva, cocina intacta, suma) | 2 (política), 4 (unitarias), 5 (integración) |
| §3.2 anular línea (adicionales, cocina hasta LISTO, `linea_entregada`, pago) | 6, 7 |
| §3.3 cancelar pedido (cascada, mesa libre, `pedido_con_pago`) | 6, 7 |
| §3.4 lectura: totales sin anuladas, motivo visible | 7 (ya filtraba; se expone el motivo) |
| §3.5 D4 | 2 (`Menor`), 3 |
| §4 arquitectura y orden de bloqueo | 4–7; verificada por `tests/architecture` en 8 |
| §5 pruebas (unitarias, integración, concurrencia, arquitectura) | 2, 3, 4, 5, 6, 7, 8 |
| §6 cierre del expediente (121 rutas, 165 operaciones) | 9 |
| §7 pendientes (R4, I4) | No se tocan |

Consistencia de nombres revisada: `PedidoBloqueable` (puerto común), `DivisionStore` y `AnulacionStore` (los dos implementados por `PedidoWriteAdapter`), `pedidoAutorizado(ctx, store, actor, pedidoID, permiso)` con el atajo `pedidoEditable`, `cuentaConPago()`/`lineaAnulada()` definidos en la Task 4 y reutilizados en la 6, `NewPedidosHandler(l, o, a, ag, e, d)` en la Task 5 y `(l, o, a, ag, e, d, an, ca)` desde la Task 7, y los helpers de pruebas `supervisorPedidos`, `agregarOK`, `pagoConfirmadoSQL`, `dividirCuenta` y `sumaImportes` (Task 5) usados por las Tasks 7 y 8.
