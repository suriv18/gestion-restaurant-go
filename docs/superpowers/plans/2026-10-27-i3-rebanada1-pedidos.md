# I3 rebanada 1 — Pedidos: apertura, cuenta y consumos (backend) — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implementar en el módulo `atencion` la apertura de pedidos con su cuenta (RES-RF-009), el alta y edición de consumos como instantánea (RES-RF-010) y las consultas de pedidos, con autorización por permiso y sede (RES-RF-002), y reflejarlo en el contrato y el expediente.

**Architecture:** Mismo corte vertical que mesas: DTO y puertos (`application`), casos de uso con fakes, persistencia SQL parametrizada (`infrastructure/persistence`), controller HTTP y composición con `escritura[...]` (transacción + idempotencia de la plataforma). Una migración `00011` solo siembra permisos; las tablas, restricciones y triggers ya existen en `00003`. La lectura de productos, tasas y mesas se hace con SQL propio dentro de `atencion` (no se importa `catalogo`).

**Tech Stack:** Go (`net/http`, `database/sql` + pgx, goose), PostgreSQL 18, testcontainers para `tests/integration`, expediente en `docs/planificacion/` (JSON, HTML, YAML, checksums).

Spec: `docs/superpowers/specs/2026-10-27-i3-rebanada1-pedidos-design.md`.

## Global Constraints

- Las rutas, nombres de campo y esquemas se alinean con el contrato existente (`docs/planificacion/03_api/openapi.yaml`): ids como cadenas, decimales como cadenas, `Idempotency-Key` UUID obligatorio en `POST`.
- Sin migración de esquema: solo `00011_pedidos_permisos.sql` (permisos). No se toca `00003`.
- `atencion` **no** importa `catalogo` ni `identidad`; la dirección de imports la verifica `tests/architecture` (se ejecuta en la Task 9).
- Los decimales viajan como `string` de extremo a extremo (VO con `math/big`); nunca `float64`.
- Los rechazos esperados son `*apperror.AppError` (nunca error técnico); los errores de base se traducen en la capa de persistencia a `outbound.Err*`.
- Los commits terminan con `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.
- No hacer `git push` ni `gh pr create` sin confirmación del usuario (Task 11).
- Las pruebas de integración necesitan Docker (testcontainers). Si Docker no está activo, declararlo; no marcar nada como verificado.
- Los archivos JSON del expediente son exactamente `JSON.stringify(x, null, 2) + "\n"`.
- Rama de trabajo: `i3-rebanada1-pedidos` (ya creada; contiene el spec en `24a8850`).
- Ejecutar los comandos `go` desde `service-restaurant/`.

### Ajustes al spec que impone el código existente (se documentan en la Task 11)

| Spec | Plan | Motivo |
|---|---|---|
| Permiso nuevo `PEDIDO_DESCONTAR` | Se usa `DESCUENTO_AUTORIZAR` (ya existe desde `00001`) y se asigna a CAJERO | Evita un permiso duplicado del mismo concepto |
| `PEDIDO_CREAR` es un permiso nuevo | Ya existe (solo ADMIN); `00011` lo asigna a MESERO y CAJERO y crea solo `PEDIDO_VER` y `PEDIDO_EDITAR` | Evita insertarlo dos veces |
| `notas_preparacion`, `descuento_total`, `cuenta_id` opcional en el alta de línea | `instrucciones`, `descuento_monto`, `cuenta_id` obligatorio | Nombres del contrato `ConsumoAlta` |
| `reparto` con `ubigeo`, coordenadas y `programado_en` | `reparto` = `DeliveryDatos` del contrato (`destinatario`, `telefono`, `direccion`, `referencia`, `canal`) | Los campos extra pertenecen a `PUT /pedidos/{id}/delivery` (I6) |
| Código `permiso_insuficiente` | `sin_permiso` | Código que ya usa todo el módulo |
| Repetición idempotente responde 200 | Responde con el estado original (201) | Así lo hace `idempotency.Run` |
| `version_bloqueo` no se expone | Se expone de solo lectura | Lo exige el esquema `Pedidos` del contrato; no se acepta como entrada |
| `POST detalles` requiere `PEDIDO_CREAR` (contrato) | Requiere `PEDIDO_EDITAR` | Lo fija el spec (D3/D5); el contrato se corrige |

## Estructura de archivos

| Archivo | Acción | Responsabilidad |
|---|---|---|
| `service-restaurant/migrations/00011_pedidos_permisos.sql` | Crear | Permisos `PEDIDO_VER`, `PEDIDO_EDITAR` y asignaciones |
| `service-restaurant/tests/integration/pedidos_permisos_migracion_test.go` | Crear | Prueba de la migración (Up y Down) |
| `internal/modules/atencion/domain/policy/permisos.go` | Modificar | Constantes de permisos de pedidos |
| `internal/modules/atencion/domain/vo/pedido.go` (+ `_test.go`) | Crear | Tipo de pedido, comensales, cantidad, dinero y regla del descuento |
| `internal/modules/atencion/application/mapper/autorizacion_pedidos.go` (+ test) | Crear | Autorización por sede para pedidos |
| `internal/modules/atencion/application/mapper/errores.go` | Modificar | `NoEncontradoM` (género masculino) |
| `internal/modules/atencion/application/dto/{command,query,result}/pedidos.go` | Crear | DTO de comandos, consultas y resultados |
| `internal/modules/atencion/application/port/outbound/pedidos.go` | Crear | Puertos `PedidoStore` y `PedidoReader` |
| `internal/modules/atencion/application/port/inbound/pedidos.go` | Crear | Puertos de entrada de los casos de uso |
| `internal/modules/atencion/application/usecase/command/{pedidos,validaciones_pedidos,consumos}.go` (+ tests) | Crear | `AbrirPedido`, `AgregarDetalle`, `EditarDetalle` |
| `internal/modules/atencion/application/usecase/query/{pedidos,sedes}.go` (+ test) | Crear/Modificar | `ListarPedidos`, `ObtenerPedido` |
| `internal/modules/atencion/infrastructure/persistence/internal/columnas/pedidos.go` | Crear | Columnas y escaneo de pedidos, cuentas y líneas |
| `internal/modules/atencion/infrastructure/persistence/internal/cargar/pedido.go` | Crear | Ensamblado de un pedido con cuentas, líneas y totales |
| `internal/modules/atencion/infrastructure/persistence/write/{repository,adapter}/pedidos*.go` | Crear | SQL de escritura y adaptador de `PedidoStore` |
| `internal/modules/atencion/infrastructure/persistence/read/{repository,adapter}/pedidos*.go` | Crear | SQL de lectura y adaptador de `PedidoReader` |
| `internal/modules/atencion/web/dto/{request,response}/pedidos.go` | Crear | Cuerpos HTTP |
| `internal/modules/atencion/web/mapper/pedidos_web_mapper.go` | Crear | Resultado → respuesta |
| `internal/modules/atencion/web/controller/{pedidos_handler,helpers,routes}.go` | Crear/Modificar | Handler y rutas |
| `internal/modules/atencion/infrastructure/config/{pedidos,module}.go` | Crear/Modificar | Composición del módulo |
| `service-restaurant/tests/integration/pedidos_*_test.go` | Crear | Pruebas HTTP contra PostgreSQL |
| `docs/planificacion/…`, `service-restaurant/docs/module-map.md`, spec y matriz | Modificar | Contrato y expediente |

---

### Task 1: Migración 00011 de permisos

**Files:**
- Create: `service-restaurant/migrations/00011_pedidos_permisos.sql`
- Create: `service-restaurant/tests/integration/pedidos_permisos_migracion_test.go`

**Interfaces:**
- Consumes: helpers del paquete `integration` (`app`, `scalar`, `sinComentarios`, `cuentaRolPermiso(t, roles, permisos string) int` de `carta_permisos_migracion_test.go`) y `migrations.FS`.
- Produces: permisos `PEDIDO_VER` y `PEDIDO_EDITAR`; asignaciones a roles. Constantes de permiso que usan las Tasks 2 y 10.

- [ ] **Step 1: Escribir la prueba**

Crear `service-restaurant/tests/integration/pedidos_permisos_migracion_test.go`:

```go
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
```

- [ ] **Step 2: Ejecutar la prueba y verificar que falla**

Run: `cd service-restaurant && go test ./tests/integration/ -run 'TestMigracion00011' -count=1 -v`
Expected: FAIL (`permisos nuevos = 0` y `open 00011_pedidos_permisos.sql: file does not exist`). Requiere Docker.

- [ ] **Step 3: Escribir la migración**

Crear `service-restaurant/migrations/00011_pedidos_permisos.sql`:

```sql
-- Migracion 00011: permisos de pedidos (I3, rebanada 1).
-- PEDIDO_VER y PEDIDO_EDITAR son nuevos. PEDIDO_CREAR y DESCUENTO_AUTORIZAR existen desde 00001 solo para ADMIN:
-- aqui se asignan a MESERO y CAJERO (PEDIDO_CREAR) y a CAJERO (DESCUENTO_AUTORIZAR).
-- Requiere PostgreSQL 18 y las migraciones 00001-00010.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO permisos(codigo,descripcion) VALUES
 ('PEDIDO_VER','Consultar pedidos, cuentas y consumos de la sede'),
 ('PEDIDO_EDITAR','Agregar y editar consumos de un pedido abierto');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='ADMIN' AND p.codigo IN ('PEDIDO_VER','PEDIDO_EDITAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO','COCINA') AND p.codigo='PEDIDO_VER';

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO') AND p.codigo IN ('PEDIDO_CREAR','PEDIDO_EDITAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='CAJERO' AND p.codigo='DESCUENTO_AUTORIZAR';

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
ALTER TABLE permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos WHERE permiso_id IN (SELECT id FROM permisos WHERE codigo IN ('PEDIDO_VER','PEDIDO_EDITAR'));
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo IN ('MESERO','CAJERO'))
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='PEDIDO_CREAR');
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo='CAJERO')
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='DESCUENTO_AUTORIZAR');
DELETE FROM permisos WHERE codigo IN ('PEDIDO_VER','PEDIDO_EDITAR');
ALTER TABLE permisos ENABLE TRIGGER a_no_borrar;
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
```

- [ ] **Step 4: Ejecutar la prueba y la suite de migraciones**

Run: `cd service-restaurant && go test ./tests/integration/ -run 'TestMigracion|TestMigrations' -count=1`
Expected: PASS. Si otra prueba de migraciones falla por contar permisos o versiones, leerla y ajustar su expectativa a «11 migraciones / 2 permisos más»; no tocar `00011` para acomodarla.

- [ ] **Step 5: Commit**

```bash
git add service-restaurant/migrations/00011_pedidos_permisos.sql service-restaurant/tests/integration/pedidos_permisos_migracion_test.go
git commit -m "$(cat <<'EOF'
I3 rebanada 1: migración 00011 con los permisos de pedidos

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Dominio y autorización (VO, política y mapper)

**Files:**
- Modify: `service-restaurant/internal/modules/atencion/domain/policy/permisos.go`
- Create: `service-restaurant/internal/modules/atencion/domain/vo/pedido.go`
- Create: `service-restaurant/internal/modules/atencion/domain/vo/pedido_test.go`
- Modify: `service-restaurant/internal/modules/atencion/application/mapper/errores.go`
- Create: `service-restaurant/internal/modules/atencion/application/mapper/autorizacion_pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/mapper/autorizacion_pedidos_test.go`

**Interfaces:**
- Produces:
  - `policy.PedidoVer`, `policy.PedidoCrear`, `policy.PedidoEditar`, `policy.DescuentoAutorizar` (strings).
  - `vo.TipoPedido` (`vo.TipoSalon`, `vo.TipoLlevar`, `vo.TipoDelivery`), `vo.NewTipoPedido(string) (TipoPedido, error)`, `(TipoPedido).String()`.
  - `vo.NewComensales(int) (Comensales, error)`, `(Comensales).Int() int`.
  - `vo.NewCantidad(string) (Cantidad, error)`, `(Cantidad).String() string`; `vo.NewDinero(string) (Dinero, error)`, `(Dinero).String() string`, `(Dinero).EsCero() bool`.
  - `vo.DescuentoCabe(c Cantidad, precioUnitario string, d Dinero) bool`.
  - `mapper.NoEncontradoM(que string) *apperror.AppError` (mensaje «… no encontrado», código `no_encontrado`).
  - `mapper.SedesConPedidos(a appctx.Actor) []int64`, `mapper.PuedeVerPedidos(a appctx.Actor, sedeID int64) bool`, `mapper.AutorizarPedidos(a appctx.Actor, sedeID int64, permiso string, ante404 *apperror.AppError) *apperror.AppError`.

- [ ] **Step 1: Escribir las pruebas del VO**

Crear `service-restaurant/internal/modules/atencion/domain/vo/pedido_test.go`:

```go
package vo

import "testing"

func TestTipoPedido(t *testing.T) {
	for _, ok := range []string{"SALON", " LLEVAR ", "DELIVERY"} {
		if _, err := NewTipoPedido(ok); err != nil {
			t.Errorf("%q debía ser válido: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "salon", "MOSTRADOR"} {
		if _, err := NewTipoPedido(mal); err == nil {
			t.Errorf("%q debía rechazarse", mal)
		}
	}
	if tp, _ := NewTipoPedido(" LLEVAR "); tp != TipoLlevar || tp.String() != "LLEVAR" {
		t.Errorf("no normalizó: %q", tp)
	}
}

func TestComensales(t *testing.T) {
	for _, n := range []int{1, 2, 500} {
		if c, err := NewComensales(n); err != nil || c.Int() != n {
			t.Errorf("%d debía ser válido: %v", n, err)
		}
	}
	for _, n := range []int{0, -1, 501} {
		if _, err := NewComensales(n); err == nil {
			t.Errorf("%d debía rechazarse", n)
		}
	}
}

func TestCantidad(t *testing.T) {
	for _, ok := range []string{"1", "2.5", " 0.001 ", "99999999999.999"} {
		if _, err := NewCantidad(ok); err != nil {
			t.Errorf("%q debía ser válida: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "0", "0.000", "-1", "+1", "1.2345", "1,5", "abc", "123456789012", "1e3"} {
		if _, err := NewCantidad(mal); err == nil {
			t.Errorf("%q debía rechazarse", mal)
		}
	}
	if c, _ := NewCantidad(" 2.50 "); c.String() != "2.50" {
		t.Errorf("no recortó: %q", c.String())
	}
}

func TestDinero(t *testing.T) {
	for _, ok := range []string{"0", "0.00", "5", "12.34", "999999999999.99"} {
		if _, err := NewDinero(ok); err != nil {
			t.Errorf("%q debía ser válido: %v", ok, err)
		}
	}
	for _, mal := range []string{"", "-1", "1.234", "x", "1000000000000"} {
		if _, err := NewDinero(mal); err == nil {
			t.Errorf("%q debía rechazarse", mal)
		}
	}
	if d, _ := NewDinero("0.00"); !d.EsCero() {
		t.Error("0.00 es cero")
	}
	if d, _ := NewDinero("0.01"); d.EsCero() {
		t.Error("0.01 no es cero")
	}
}

func TestDescuentoCabe(t *testing.T) {
	casos := []struct {
		nombre, cantidad, precio, descuento string
		quiere                              bool
	}{
		{"igual al importe", "2", "10.000000", "20.00", true},
		{"un centavo más", "2", "10.000000", "20.01", false},
		{"cero siempre cabe", "1", "0.000000", "0", true},
		{"redondea el importe a 2 decimales", "3", "3.333333", "10.00", true},
		{"el redondeo no regala un centavo", "3", "3.333333", "10.01", false},
		{"cantidad fraccionaria", "0.5", "9.990000", "5.00", true},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			cant, err := NewCantidad(c.cantidad)
			if err != nil {
				t.Fatal(err)
			}
			d, err := NewDinero(c.descuento)
			if err != nil {
				t.Fatal(err)
			}
			if got := DescuentoCabe(cant, c.precio, d); got != c.quiere {
				t.Errorf("DescuentoCabe(%s × %s, %s) = %v, quería %v", c.cantidad, c.precio, c.descuento, got, c.quiere)
			}
		})
	}
	cant, _ := NewCantidad("1")
	d, _ := NewDinero("0")
	if DescuentoCabe(cant, "no-es-un-numero", d) {
		t.Error("un precio ilegible no debe aceptar el descuento")
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `cd service-restaurant && go test ./internal/modules/atencion/domain/vo/ -run 'TestTipoPedido|TestComensales|TestCantidad|TestDinero|TestDescuentoCabe' -count=1`
Expected: FAIL de compilación (`undefined: NewTipoPedido`, …).

- [ ] **Step 3: Escribir los VO**

Crear `service-restaurant/internal/modules/atencion/domain/vo/pedido.go`:

```go
package vo

import (
	"math/big"
	"regexp"
	"strings"
)

const maxComensales = 500

var (
	cantidadRe = regexp.MustCompile(`^[0-9]{1,11}(\.[0-9]{1,3})?$`)
	dineroRe   = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,2})?$`)
)

// TipoPedido es el canal de atención de un pedido.
type TipoPedido string

const (
	TipoSalon    TipoPedido = "SALON"
	TipoLlevar   TipoPedido = "LLEVAR"
	TipoDelivery TipoPedido = "DELIVERY"
)

func NewTipoPedido(raw string) (TipoPedido, error) {
	switch t := TipoPedido(strings.TrimSpace(raw)); t {
	case TipoSalon, TipoLlevar, TipoDelivery:
		return t, nil
	}
	return "", ErrValorInvalido
}

func (t TipoPedido) String() string { return string(t) }

// Comensales es el número de personas de un pedido (1 a 500).
type Comensales struct{ value int }

func NewComensales(n int) (Comensales, error) {
	if n < 1 || n > maxComensales {
		return Comensales{}, ErrValorInvalido
	}
	return Comensales{n}, nil
}

func (c Comensales) Int() int { return c.value }

// Cantidad es una cantidad decimal exacta de numeric(14,3), estrictamente positiva.
type Cantidad struct {
	text string
	r    *big.Rat
}

func NewCantidad(raw string) (Cantidad, error) {
	v := strings.TrimSpace(raw)
	if !cantidadRe.MatchString(v) {
		return Cantidad{}, ErrValorInvalido
	}
	r, ok := new(big.Rat).SetString(v)
	if !ok || r.Sign() <= 0 {
		return Cantidad{}, ErrValorInvalido
	}
	return Cantidad{v, r}, nil
}

func (c Cantidad) String() string { return c.text }

// Dinero es un importe decimal exacto de numeric(14,2), mayor o igual que cero.
type Dinero struct {
	text string
	r    *big.Rat
}

func NewDinero(raw string) (Dinero, error) {
	v := strings.TrimSpace(raw)
	if !dineroRe.MatchString(v) {
		return Dinero{}, ErrValorInvalido
	}
	r, ok := new(big.Rat).SetString(v)
	if !ok {
		return Dinero{}, ErrValorInvalido
	}
	return Dinero{v, r}, nil
}

func (d Dinero) String() string { return d.text }
func (d Dinero) EsCero() bool   { return d.r.Sign() == 0 }

// DescuentoCabe indica si el descuento no supera el importe bruto de la línea:
// round(cantidad × precio_unitario, 2), con el redondeo de PostgreSQL (mitad hacia arriba).
// Es la misma regla que el CHECK de detalles_pedido, para rechazar antes con un código propio.
func DescuentoCabe(c Cantidad, precioUnitario string, d Dinero) bool {
	p, ok := new(big.Rat).SetString(strings.TrimSpace(precioUnitario))
	if !ok {
		return false
	}
	bruto := redondear2(new(big.Rat).Mul(c.r, p))
	return d.r.Cmp(bruto) <= 0
}

// redondear2 redondea un racional no negativo a 2 decimales, la mitad hacia arriba.
func redondear2(r *big.Rat) *big.Rat {
	x := new(big.Rat).Mul(r, big.NewRat(100, 1))
	n := new(big.Int).Quo(x.Num(), x.Denom())
	resto := new(big.Rat).Sub(x, new(big.Rat).SetInt(n))
	if resto.Cmp(big.NewRat(1, 2)) >= 0 {
		n.Add(n, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(n, big.NewInt(100))
}
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `cd service-restaurant && go test ./internal/modules/atencion/domain/vo/ -count=1 -v 2>&1 | tail -20`
Expected: PASS de todo el paquete (incluidos los tests de mesa y texto, que no cambian).

- [ ] **Step 5: Política y mapper — escribir las pruebas**

Añadir al final de `service-restaurant/internal/modules/atencion/domain/policy/permisos.go`:

```go

// Permisos de pedidos (migración 00011; PEDIDO_CREAR y DESCUENTO_AUTORIZAR existen desde 00001).
const (
	PedidoVer          = "PEDIDO_VER"
	PedidoCrear        = "PEDIDO_CREAR"
	PedidoEditar       = "PEDIDO_EDITAR"
	DescuentoAutorizar = "DESCUENTO_AUTORIZAR"
)
```

Crear `service-restaurant/internal/modules/atencion/application/mapper/autorizacion_pedidos_test.go`:

```go
package mapper

import (
	"slices"
	"testing"

	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func TestSedesConPedidos(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 5, Permiso: policy.PedidoVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoCrear},
		appctx.Grant{SucursalID: 5, Permiso: policy.PedidoEditar},
		appctx.Grant{SucursalID: 9, Permiso: policy.MesaVer},
	)
	if got := SedesConPedidos(a); !slices.Equal(got, []int64{2, 5}) {
		t.Fatalf("sedes con pedidos = %v, quería [2 5]", got)
	}
	if got := SedesConPedidos(actorCon()); got == nil || len(got) != 0 {
		t.Fatalf("sin sedes debe devolver un slice vacío no nulo: %#v", got)
	}
}

func TestAutorizarPedidos(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 1, Permiso: policy.PedidoVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoCrear},
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoEditar},
	)
	ante404 := NoEncontradoM("pedido")

	// Sede que no ve: 404 con el rechazo que se le pasó.
	if ae := AutorizarPedidos(a, 9, policy.PedidoEditar, ante404); ae != ante404 {
		t.Fatalf("sede ajena debe devolver el 404 recibido: %v", ae)
	}
	// La ve pero sin el permiso del comando: 403.
	if ae := AutorizarPedidos(a, 1, policy.PedidoEditar, ante404); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSedeNoAutorizada {
		t.Fatalf("solo lectura debe ser 403 sede_no_autorizada: %v", ae)
	}
	// Con el permiso: autorizado. CREAR y EDITAR implican ver.
	if ae := AutorizarPedidos(a, 2, policy.PedidoEditar, ante404); ae != nil {
		t.Fatalf("debía autorizar: %v", ae)
	}
	if !PuedeVerPedidos(a, 2) || !PuedeVerPedidos(a, 1) || PuedeVerPedidos(a, 9) {
		t.Fatal("PuedeVerPedidos: debe ver la 1 y la 2, no la 9")
	}
}

func TestNoEncontradoM(t *testing.T) {
	ae := NoEncontradoM("pedido")
	if ae.Kind != apperror.KindNotFound || ae.Code != CodigoNoEncontrado || ae.Message != "pedido no encontrado" {
		t.Fatalf("NoEncontradoM = %+v", ae)
	}
}
```

- [ ] **Step 6: Ejecutar y verificar que falla**

Run: `cd service-restaurant && go test ./internal/modules/atencion/application/mapper/ -run 'TestSedesConPedidos|TestAutorizarPedidos|TestNoEncontradoM' -count=1`
Expected: FAIL de compilación (`undefined: SedesConPedidos`, `NoEncontradoM`).

- [ ] **Step 7: Implementar el mapper**

En `service-restaurant/internal/modules/atencion/application/mapper/errores.go`, justo después de la función `NoEncontrado`, añadir:

```go

// NoEncontradoM es como NoEncontrado para sustantivos masculinos ("pedido", "producto").
func NoEncontradoM(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrado")
}
```

Crear `service-restaurant/internal/modules/atencion/application/mapper/autorizacion_pedidos.go`:

```go
package mapper

import (
	"slices"

	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

// SedesConPedidos devuelve, ordenadas y sin repetir, las sedes donde el actor tiene algún permiso
// de pedidos (ver, crear o editar). Nunca es nil.
func SedesConPedidos(a appctx.Actor) []int64 {
	ids := append(a.SucursalesCon(policy.PedidoVer), a.SucursalesCon(policy.PedidoCrear)...)
	ids = append(ids, a.SucursalesCon(policy.PedidoEditar)...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if ids == nil {
		return []int64{}
	}
	return ids
}

// PuedeVerPedidos indica si el actor tiene algún permiso de pedidos en la sede.
func PuedeVerPedidos(a appctx.Actor, sedeID int64) bool {
	return a.Can(policy.PedidoVer, sedeID) || a.Can(policy.PedidoCrear, sedeID) || a.Can(policy.PedidoEditar, sedeID)
}

// AutorizarPedidos decide si el actor puede ejecutar una operación de pedidos con el permiso en la
// sede: si no ve pedidos de esa sede responde ante404 (no se revela su existencia); si los ve pero
// no tiene el permiso, 403. ante404 lo elige quien llama para nombrar el recurso ("sucursal" en
// la apertura, "pedido" en las operaciones por id).
func AutorizarPedidos(a appctx.Actor, sedeID int64, permiso string, ante404 *apperror.AppError) *apperror.AppError {
	if !PuedeVerPedidos(a, sedeID) {
		return ante404
	}
	if !a.Can(permiso, sedeID) {
		return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada, "no tiene permiso para esta operación de pedidos en esta sucursal")
	}
	return nil
}
```

- [ ] **Step 8: Ejecutar y verificar que pasa**

Run: `cd service-restaurant && go test ./internal/modules/atencion/... -count=1 2>&1 | tail -15`
Expected: PASS en todos los paquetes de `atencion`.

- [ ] **Step 9: Commit**

```bash
git add service-restaurant/internal/modules/atencion/domain service-restaurant/internal/modules/atencion/application/mapper
git commit -m "$(cat <<'EOF'
I3 rebanada 1: VO de pedidos y autorización por sede

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: DTO y puertos

**Files:**
- Create: `service-restaurant/internal/modules/atencion/application/dto/result/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/dto/command/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/dto/query/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/port/outbound/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/port/inbound/pedidos.go`

**Interfaces:**
- Produces: los tipos exactos que usan las Tasks 4 a 9 (ver el código). Sin lógica: se verifica compilando.

- [ ] **Step 1: DTO de resultado**

Crear `service-restaurant/internal/modules/atencion/application/dto/result/pedidos.go`:

```go
package result

import "time"

// Pedido es la cabecera de un pedido. Los importes y cantidades de sus líneas son decimales
// exactos serializados como cadena.
type Pedido struct {
	ID                int64
	SucursalID        int64
	CodigoPublico     string
	ClaveIdempotencia string
	MesaID            *int64
	ReservaID         *int64
	ClienteID         *int64
	MeseroID          *int64
	Tipo              string
	Estado            string
	Moneda            string
	Comensales        int
	AbiertoEn         time.Time
	CerradoEn         *time.Time
	Notas             *string
	MotivoCancelacion *string
	VersionBloqueo    int
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
}

// Reparto son los datos de entrega de un pedido DELIVERY.
type Reparto struct {
	Destinatario string
	Telefono     string
	Direccion    string
	Referencia   *string
	Canal        string
	Estado       string
}

// Linea es un consumo del pedido como instantánea: descripción, precio y tasa quedan fijos aunque
// la carta cambie. ImporteTotal y BaseImponible los calcula la base (columnas generadas).
type Linea struct {
	ID               int64
	PedidoID         int64
	CuentaID         int64
	ProductoID       int64
	DetallePadreID   *int64
	Descripcion      string
	Cantidad         string
	PrecioUnitario   string
	DescuentoTotal   string
	AfectacionIGV    string
	TasaImpuesto     string
	ImporteTotal     string
	BaseImponible    string
	Estado           string
	NotasPreparacion *string
	CreadoEn         time.Time
	CreadoPor        int64
	ActualizadoEn    time.Time
	ActualizadoPor   int64
}

// Cuenta agrupa líneas. Subtotal, Impuesto y Total suman solo las líneas activas.
type Cuenta struct {
	ID        int64
	Numero    int
	ClienteID *int64
	Etiqueta  *string
	Estado    string
	Subtotal  string
	Impuesto  string
	Total     string
	Lineas    []Linea
}

// PedidoDetalle es un pedido con su reparto (solo DELIVERY) y sus cuentas.
type PedidoDetalle struct {
	Pedido
	Reparto *Reparto
	Cuentas []Cuenta
}

// PaginaPedidos: NextCursor vacío indica la última página.
type PaginaPedidos struct {
	Items      []Pedido
	NextCursor string
}
```

- [ ] **Step 2: DTO de comandos y consultas**

Crear `service-restaurant/internal/modules/atencion/application/dto/command/pedidos.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// Reparto son los datos de entrega que acompañan a un pedido DELIVERY.
type Reparto struct {
	Destinatario string
	Telefono     string
	Direccion    string
	Referencia   *string
	Canal        *string
}

// AbrirPedido crea el pedido y su cuenta nº 1. Tipo es SALON, LLEVAR o DELIVERY; MesaID solo en
// SALON y Reparto solo en DELIVERY.
type AbrirPedido struct {
	Intencion
	Actor      appctx.Actor
	SucursalID int64
	Tipo       string
	MesaID     *int64
	Comensales int
	ClienteID  *int64
	Notas      *string
	Reparto    *Reparto
}

// AgregarDetalle agrega una línea a una cuenta abierta. Cantidad y DescuentoMonto son decimales
// exactos en cadena; el servidor fija descripción, precio y tasa desde la carta.
type AgregarDetalle struct {
	Intencion
	Actor          appctx.Actor
	PedidoID       int64
	CuentaID       int64
	ProductoID     int64
	Cantidad       string
	DetallePadreID *int64
	Instrucciones  *string
	DescuentoMonto *string
}

// EditarDetalle modifica solo lo enviado de una línea activa. Instrucciones distingue "no
// enviado" de "enviado como null" (vaciar).
type EditarDetalle struct {
	Actor          appctx.Actor
	PedidoID       int64
	DetalleID      int64
	Cantidad       *string
	Instrucciones  Campo[string]
	DescuentoMonto *string
}
```

Crear `service-restaurant/internal/modules/atencion/application/dto/query/pedidos.go`:

```go
package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarPedidos pagina por cursor (id ascendente) los pedidos de las sedes que el actor puede ver.
// Estado y Tipo vacíos no filtran.
type ListarPedidos struct {
	Actor      appctx.Actor
	SucursalID *int64
	Estado     string
	Tipo       string
	MesaID     *int64
	Limite     int
	Cursor     string
}

type ObtenerPedido struct {
	Actor appctx.Actor
	ID    int64
}
```

- [ ] **Step 3: Puertos de salida**

Crear `service-restaurant/internal/modules/atencion/application/port/outbound/pedidos.go`:

```go
package outbound

import (
	"context"
	"fmt"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// Rechazos de almacenamiento propios de pedidos; ambos envuelven ErrDuplicado.
var (
	ErrMesaOcupada = fmt.Errorf("la mesa ya tiene un pedido abierto: %w", ErrDuplicado)
	ErrClaveEnUso  = fmt.Errorf("la clave de idempotencia ya fue usada por otro pedido: %w", ErrDuplicado)
)

// FiltroPedidos acota el listado a las sedes visibles (SedeIDs) y, opcionalmente, por estado,
// tipo y mesa.
type FiltroPedidos struct {
	SedeIDs     []int64
	Estado      string
	Tipo        string
	MesaID      *int64
	DespuesDeID int64
	Limite      int
}

// PedidoReader lee pedidos sin bloquear filas.
type PedidoReader interface {
	ListarPedidos(ctx context.Context, f FiltroPedidos) ([]dtoresult.Pedido, error)
	ObtenerPedido(ctx context.Context, id int64) (dtoresult.PedidoDetalle, bool, error)
}

// NuevoReparto son los datos de entrega a persistir. Canal nil usa el valor por defecto (PROPIO).
type NuevoReparto struct {
	Destinatario string
	Telefono     string
	Direccion    string
	Referencia   *string
	Canal        *string
}

// NuevoPedido: la cuenta nº 1 y, si Reparto no es nil, los datos de entrega se crean con él.
type NuevoPedido struct {
	SucursalID        int64
	ClaveIdempotencia string
	Tipo              string
	MesaID            *int64
	MeseroID          *int64
	ClienteID         *int64
	Comensales        int
	Notas             *string
	Reparto           *NuevoReparto
}

// MesaRef resume la mesa que un pedido de salón ocupa.
type MesaRef struct {
	SucursalID int64
	Capacidad  int
	Habilitada bool
	EsActiva   bool
}

// PedidoRef resume un pedido bloqueado. Cerrado incluye los cancelados.
type PedidoRef struct {
	ID         int64
	SucursalID int64
	Estado     string
	Cerrado    bool
}

// CuentaRef resume una cuenta bloqueada.
type CuentaRef struct {
	ID       int64
	PedidoID int64
	Estado   string
}

// ProductoVenta es lo que hace falta de la carta para fijar una línea. Precio ya incluye el precio
// propio de la sede si existe. Tasa es la vigente hoy del impuesto del producto ("" si no hay).
// Disponible es false si el producto no tiene fila activa en la sede.
type ProductoVenta struct {
	ID            int64
	Nombre        string
	Tipo          string
	AfectacionIGV string
	Precio        string
	Tasa          string
	EsActivo      bool
	Disponible    bool
}

// DetalleRef es una línea bloqueada. EnComanda indica que ya se envió a cocina.
type DetalleRef struct {
	ID             int64
	PedidoID       int64
	CuentaID       int64
	ProductoID     int64
	Cantidad       string
	PrecioUnitario string
	DescuentoTotal string
	Estado         string
	EnComanda      bool
}

// NuevoDetalle es la instantánea de una línea: todo salvo importe_total y base_imponible.
type NuevoDetalle struct {
	PedidoID         int64
	CuentaID         int64
	ProductoID       int64
	DetallePadreID   *int64
	Descripcion      string
	Cantidad         string
	PrecioUnitario   string
	DescuentoTotal   string
	AfectacionIGV    string
	TasaImpuesto     string
	NotasPreparacion *string
}

// CambiosDetalle: solo los campos presentes se modifican.
type CambiosDetalle struct {
	Cantidad       *string
	DescuentoTotal *string
	Instrucciones  CampoOpcional
}

// PedidoStore escribe pedidos dentro de la transacción del comando. Los métodos que devuelven una
// referencia bloquean la fila hasta el final de la transacción.
type PedidoStore interface {
	// MesaParaPedido lee la mesa en modo compartido, para que no se deshabilite durante el comando.
	MesaParaPedido(ctx context.Context, mesaID int64) (MesaRef, bool, error)
	// EmpleadoDeUsuario devuelve el empleado activo vinculado al usuario, o nil.
	EmpleadoDeUsuario(ctx context.Context, usuarioID int64) (*int64, error)
	// CrearPedido inserta pedido, cuenta nº 1 y reparto, y devuelve el pedido ya armado.
	CrearPedido(ctx context.Context, n NuevoPedido) (dtoresult.PedidoDetalle, error)
	// BloquearPedido lee el pedido FOR UPDATE.
	BloquearPedido(ctx context.Context, id int64) (PedidoRef, bool, error)
	// CuentaDelPedido lee la cuenta FOR UPDATE; found=false si no existe o es de otro pedido.
	CuentaDelPedido(ctx context.Context, pedidoID, cuentaID int64) (CuentaRef, bool, error)
	// ProductoParaVenta lee el producto con su precio y tasa vigentes en la sede (FOR SHARE).
	ProductoParaVenta(ctx context.Context, productoID, sedeID int64) (ProductoVenta, bool, error)
	// DetalleDelPedido lee la línea FOR UPDATE; found=false si no existe o es de otro pedido.
	DetalleDelPedido(ctx context.Context, pedidoID, detalleID int64) (DetalleRef, bool, error)
	InsertarDetalle(ctx context.Context, n NuevoDetalle) (dtoresult.Linea, error)
	ActualizarDetalle(ctx context.Context, id int64, c CambiosDetalle) (dtoresult.Linea, error)
}
```

- [ ] **Step 4: Puertos de entrada**

Crear `service-restaurant/internal/modules/atencion/application/port/inbound/pedidos.go`:

```go
package inbound

import (
	"context"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarPedidos interface {
	Execute(context.Context, qry.ListarPedidos) (appresult.Result[dtoresult.PaginaPedidos], error)
}
type ObtenerPedido interface {
	Execute(context.Context, qry.ObtenerPedido) (appresult.Result[dtoresult.PedidoDetalle], error)
}
type AbrirPedido interface {
	Execute(context.Context, cmd.AbrirPedido) (appresult.Result[dtoresult.PedidoDetalle], error)
}
type AgregarDetalle interface {
	Execute(context.Context, cmd.AgregarDetalle) (appresult.Result[dtoresult.Linea], error)
}
type EditarDetalle interface {
	Execute(context.Context, cmd.EditarDetalle) (appresult.Result[dtoresult.Linea], error)
}
```

- [ ] **Step 5: Verificar que compila**

Run: `cd service-restaurant && go build ./internal/modules/atencion/... && go vet ./internal/modules/atencion/...`
Expected: sin salida (compila y no hay advertencias).

- [ ] **Step 6: Commit**

```bash
git add service-restaurant/internal/modules/atencion/application
git commit -m "$(cat <<'EOF'
I3 rebanada 1: DTO y puertos de pedidos

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Caso de uso AbrirPedido

**Files:**
- Create: `service-restaurant/internal/modules/atencion/application/usecase/command/validaciones_pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/usecase/command/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/usecase/command/pedidos_fake_test.go`
- Create: `service-restaurant/internal/modules/atencion/application/usecase/command/pedidos_test.go`

**Interfaces:**
- Consumes: todo de las Tasks 2 y 3, más los helpers de prueba ya existentes en el paquete: `texto(s string) *string`, `esperarRechazo(t, ae, kind, code)`, `sinFallo(t, err)`.
- Produces: `command.NewAbrirPedido(outbound.PedidoStore) *AbrirPedido` con `Execute(ctx, cmd.AbrirPedido)`; constantes `CodigoMesaOcupada`, `CodigoClaveEnUso`; helpers de validación `textoOpcional`, `validarCantidad`, `validarDescuento` (Task 5 los reutiliza); el fake `pedidoStoreFake` (Task 5 lo reutiliza).

- [ ] **Step 1: Escribir el fake y las pruebas**

Crear `service-restaurant/internal/modules/atencion/application/usecase/command/pedidos_fake_test.go`:

```go
package command

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	appctx "service-restaurant/internal/shared/application/context"
)

// pedidoStoreFake implementa outbound.PedidoStore con respuestas fijadas por la prueba.
type pedidoStoreFake struct {
	mesa       outbound.MesaRef
	mesaExiste bool
	empleado   *int64
	errCrear   error
	creado     outbound.NuevoPedido
	creaciones int

	pedido         outbound.PedidoRef
	pedidoExiste   bool
	cuenta         outbound.CuentaRef
	cuentaExiste   bool
	producto       outbound.ProductoVenta
	productoExiste bool
	detalles       map[int64]outbound.DetalleRef

	insertado       outbound.NuevoDetalle
	inserciones     int
	cambios         outbound.CambiosDetalle
	actualizaciones int
}

var _ outbound.PedidoStore = (*pedidoStoreFake)(nil)

func (f *pedidoStoreFake) MesaParaPedido(context.Context, int64) (outbound.MesaRef, bool, error) {
	return f.mesa, f.mesaExiste, nil
}
func (f *pedidoStoreFake) EmpleadoDeUsuario(context.Context, int64) (*int64, error) {
	return f.empleado, nil
}
func (f *pedidoStoreFake) CrearPedido(_ context.Context, n outbound.NuevoPedido) (dtoresult.PedidoDetalle, error) {
	f.creaciones++
	f.creado = n
	return dtoresult.PedidoDetalle{Pedido: dtoresult.Pedido{ID: 5, SucursalID: n.SucursalID, Tipo: n.Tipo}}, f.errCrear
}
func (f *pedidoStoreFake) BloquearPedido(context.Context, int64) (outbound.PedidoRef, bool, error) {
	return f.pedido, f.pedidoExiste, nil
}
func (f *pedidoStoreFake) CuentaDelPedido(context.Context, int64, int64) (outbound.CuentaRef, bool, error) {
	return f.cuenta, f.cuentaExiste, nil
}
func (f *pedidoStoreFake) ProductoParaVenta(context.Context, int64, int64) (outbound.ProductoVenta, bool, error) {
	return f.producto, f.productoExiste, nil
}
func (f *pedidoStoreFake) DetalleDelPedido(_ context.Context, _ int64, id int64) (outbound.DetalleRef, bool, error) {
	d, ok := f.detalles[id]
	return d, ok, nil
}
func (f *pedidoStoreFake) InsertarDetalle(_ context.Context, n outbound.NuevoDetalle) (dtoresult.Linea, error) {
	f.inserciones++
	f.insertado = n
	return dtoresult.Linea{ID: 99, Cantidad: n.Cantidad}, nil
}
func (f *pedidoStoreFake) ActualizarDetalle(_ context.Context, id int64, c outbound.CambiosDetalle) (dtoresult.Linea, error) {
	f.actualizaciones++
	f.cambios = c
	return dtoresult.Linea{ID: id}, nil
}

// actorPedidos arma un actor con los permisos dados en una sola sede.
func actorPedidos(sede int64, permisos ...string) appctx.Actor {
	g := make([]appctx.Grant, 0, len(permisos))
	for _, p := range permisos {
		g = append(g, appctx.Grant{SucursalID: sede, Permiso: p})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func idPtr(n int64) *int64 { return &n }
```

Crear `service-restaurant/internal/modules/atencion/application/usecase/command/pedidos_test.go`:

```go
package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

const claveEjemplo = "6f1c0a52-3c1e-4a53-9a39-0b9a3a2f1a11"

func fakeApertura() *pedidoStoreFake {
	return &pedidoStoreFake{
		mesa:       outbound.MesaRef{SucursalID: 1, Capacidad: 4, Habilitada: true, EsActiva: true},
		mesaExiste: true,
	}
}

func abrirBase() cmd.AbrirPedido {
	return cmd.AbrirPedido{
		Intencion: cmd.Intencion{Clave: claveEjemplo},
		Actor:     actorPedidos(1, policy.PedidoCrear),
		SucursalID: 1, Tipo: "SALON", MesaID: idPtr(7), Comensales: 2,
	}
}

func TestAbrirPedidoSalon(t *testing.T) {
	f := fakeApertura()
	f.empleado = idPtr(41)
	c := abrirBase()
	c.Notas = texto("  sin cebolla  ")
	c.ClienteID = idPtr(12)

	res, err := NewAbrirPedido(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía abrirse: %v %+v", err, res.Err())
	}
	n := f.creado
	if n.SucursalID != 1 || n.ClaveIdempotencia != claveEjemplo || n.Tipo != "SALON" || n.MesaID == nil || *n.MesaID != 7 ||
		n.Comensales != 2 || n.ClienteID == nil || *n.ClienteID != 12 || n.MeseroID == nil || *n.MeseroID != 41 ||
		n.Notas == nil || *n.Notas != "sin cebolla" || n.Reparto != nil {
		t.Fatalf("pedido mal armado: %+v", n)
	}
	if res.Value().ID != 5 {
		t.Fatalf("debe devolver el pedido creado: %+v", res.Value())
	}
}

func TestAbrirPedidoLlevarYDelivery(t *testing.T) {
	f := fakeApertura()
	llevar := abrirBase()
	llevar.Tipo, llevar.MesaID = "LLEVAR", nil
	res, err := NewAbrirPedido(f).Execute(context.Background(), llevar)
	if err != nil || !res.IsOk() || f.creado.MesaID != nil || f.creado.Reparto != nil || f.creado.MeseroID != nil {
		t.Fatalf("LLEVAR: %v %+v %+v", err, res.Err(), f.creado)
	}

	delivery := abrirBase()
	delivery.Tipo, delivery.MesaID = "DELIVERY", nil
	delivery.Reparto = &cmd.Reparto{Destinatario: " Ana Pérez ", Telefono: "999111222", Direccion: " Av. Lima 123 ", Canal: texto("PEDIDOSYA")}
	res, err = NewAbrirPedido(f).Execute(context.Background(), delivery)
	if err != nil || !res.IsOk() {
		t.Fatalf("DELIVERY: %v %+v", err, res.Err())
	}
	r := f.creado.Reparto
	if r == nil || r.Destinatario != "Ana Pérez" || r.Telefono != "999111222" || r.Direccion != "Av. Lima 123" ||
		r.Referencia != nil || r.Canal == nil || *r.Canal != "PEDIDOSYA" {
		t.Fatalf("reparto mal armado: %+v", r)
	}
}

func TestAbrirPedidoRechazos(t *testing.T) {
	deliveryOk := &cmd.Reparto{Destinatario: "Ana", Telefono: "999", Direccion: "Av. Lima 1"}
	casos := []struct {
		nombre string
		mod    func(f *pedidoStoreFake, c *cmd.AbrirPedido)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"sede ajena", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.SucursalID = 9 }, apperror.KindNotFound, "no_encontrado"},
		{"la sede se ve pero no se crea en ella", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Actor = appctx.NewActor(7, "ana", 1, []appctx.Grant{
				{SucursalID: 1, Permiso: policy.PedidoVer}, {SucursalID: 2, Permiso: policy.PedidoCrear}})
		}, apperror.KindForbidden, "sede_no_autorizada"},
		{"tipo desconocido", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Tipo = "MOSTRADOR" }, apperror.KindValidation, "tipo_invalido"},
		{"comensales cero", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Comensales = 0 }, apperror.KindValidation, "comensales_invalidos"},
		{"notas con caracteres de control", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Notas = texto("a\x00b") }, apperror.KindValidation, "notas_invalidas"},
		{"salón sin mesa", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.MesaID = nil }, apperror.KindValidation, "mesa_requerida"},
		{"llevar con mesa", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Tipo = "LLEVAR" }, apperror.KindValidation, "mesa_no_aplica"},
		{"reparto en un pedido de salón", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Reparto = deliveryOk }, apperror.KindValidation, "reparto_no_aplica"},
		{"delivery sin reparto", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Tipo, c.MesaID = "DELIVERY", nil }, apperror.KindValidation, "reparto_requerido"},
		{"delivery sin destinatario", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Tipo, c.MesaID = "DELIVERY", nil
			c.Reparto = &cmd.Reparto{Destinatario: " ", Telefono: "999", Direccion: "Av. Lima 1"}
		}, apperror.KindValidation, "destinatario_invalido"},
		{"delivery sin teléfono", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Tipo, c.MesaID = "DELIVERY", nil
			c.Reparto = &cmd.Reparto{Destinatario: "Ana", Telefono: "", Direccion: "Av. Lima 1"}
		}, apperror.KindValidation, "telefono_invalido"},
		{"delivery sin dirección", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) {
			c.Tipo, c.MesaID = "DELIVERY", nil
			c.Reparto = &cmd.Reparto{Destinatario: "Ana", Telefono: "999", Direccion: ""}
		}, apperror.KindValidation, "direccion_invalida"},
		{"mesa inexistente", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesaExiste = false }, apperror.KindValidation, "mesa_invalida"},
		{"mesa de otra sede", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesa.SucursalID = 2 }, apperror.KindValidation, "mesa_invalida"},
		{"mesa no habilitada", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesa.Habilitada = false }, apperror.KindValidation, "mesa_no_disponible"},
		{"mesa dada de baja", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.mesa.EsActiva = false }, apperror.KindValidation, "mesa_no_disponible"},
		{"capacidad excedida", func(_ *pedidoStoreFake, c *cmd.AbrirPedido) { c.Comensales = 5 }, apperror.KindValidation, "capacidad_excedida"},
		{"mesa con pedido abierto", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.errCrear = outbound.ErrMesaOcupada }, apperror.KindConflict, "mesa_ocupada"},
		{"clave usada por otro pedido", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.errCrear = outbound.ErrClaveEnUso }, apperror.KindConflict, "clave_en_uso"},
		{"referencia inexistente", func(f *pedidoStoreFake, _ *cmd.AbrirPedido) { f.errCrear = outbound.ErrReferenciaInvalida }, apperror.KindValidation, "referencia_invalida"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := fakeApertura()
			in := abrirBase()
			c.mod(f, &in)
			res, err := NewAbrirPedido(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}
}

func TestAbrirPedidoRechazadoNoToca(t *testing.T) {
	f := fakeApertura()
	in := abrirBase()
	in.Comensales = 9
	_, err := NewAbrirPedido(f).Execute(context.Background(), in)
	sinFallo(t, err)
	if f.creaciones != 0 {
		t.Fatal("un rechazo no debe escribir")
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `cd service-restaurant && go test ./internal/modules/atencion/application/usecase/command/ -run 'TestAbrirPedido' -count=1`
Expected: FAIL de compilación (`undefined: NewAbrirPedido`).

- [ ] **Step 3: Escribir las validaciones**

Crear `service-restaurant/internal/modules/atencion/application/usecase/command/validaciones_pedidos.go`:

```go
package command

import (
	"strings"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/vo"
	"service-restaurant/internal/shared/kernel/apperror"
)

const (
	maxNotasPedido   = 2000
	maxDestinatario  = 150
	maxTelefono      = 30
	maxDireccion     = 500
	maxReferencia    = 250
	maxCanal         = 40
	maxInstrucciones = 2000
)

// textoOpcional valida un texto anulable: nil o solo espacios → ausente; con contenido, se
// recorta y se valida (multilinea admite saltos de línea y tabulaciones).
func textoOpcional(raw *string, max int, multilinea bool, code, msg string) (*string, *apperror.AppError) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var v string
	var err error
	if multilinea {
		v, err = vo.TextoObligatorio(*raw, max)
	} else {
		v, err = vo.TextoUnaLinea(*raw, max)
	}
	if err != nil {
		return nil, mapper.Validacion(code, msg)
	}
	return &v, nil
}

func validarTipoPedido(raw string) (vo.TipoPedido, *apperror.AppError) {
	t, err := vo.NewTipoPedido(raw)
	if err != nil {
		return "", mapper.Validacion("tipo_invalido", "el tipo debe ser SALON, LLEVAR o DELIVERY")
	}
	return t, nil
}

func validarComensales(raw int) (vo.Comensales, *apperror.AppError) {
	c, err := vo.NewComensales(raw)
	if err != nil {
		return vo.Comensales{}, mapper.Validacion("comensales_invalidos", "los comensales deben estar entre 1 y 500")
	}
	return c, nil
}

func validarNotas(raw *string) (*string, *apperror.AppError) {
	return textoOpcional(raw, maxNotasPedido, true, "notas_invalidas", "las notas admiten hasta 2000 caracteres sin caracteres de control")
}

// validarMesaSegunTipo: SALON exige mesa y los demás tipos no la admiten.
func validarMesaSegunTipo(tipo vo.TipoPedido, mesaID *int64) *apperror.AppError {
	if tipo == vo.TipoSalon && mesaID == nil {
		return mapper.Validacion("mesa_requerida", "el pedido de SALON exige una mesa")
	}
	if tipo != vo.TipoSalon && mesaID != nil {
		return mapper.Validacion("mesa_no_aplica", "solo el pedido de SALON lleva mesa")
	}
	return nil
}

// validarReparto: DELIVERY exige los datos de entrega y los demás tipos no los admiten.
func validarReparto(tipo vo.TipoPedido, r *cmd.Reparto) (*outbound.NuevoReparto, *apperror.AppError) {
	if tipo != vo.TipoDelivery {
		if r != nil {
			return nil, mapper.Validacion("reparto_no_aplica", "solo el pedido DELIVERY lleva datos de reparto")
		}
		return nil, nil
	}
	if r == nil {
		return nil, mapper.Validacion("reparto_requerido", "el pedido DELIVERY exige los datos de reparto")
	}
	dest, err := vo.TextoUnaLinea(r.Destinatario, maxDestinatario)
	if err != nil {
		return nil, mapper.Validacion("destinatario_invalido", "el destinatario es obligatorio (máximo 150 caracteres)")
	}
	tel, err := vo.TextoUnaLinea(r.Telefono, maxTelefono)
	if err != nil {
		return nil, mapper.Validacion("telefono_invalido", "el teléfono es obligatorio (máximo 30 caracteres)")
	}
	dir, err := vo.TextoObligatorio(r.Direccion, maxDireccion)
	if err != nil {
		return nil, mapper.Validacion("direccion_invalida", "la dirección es obligatoria (máximo 500 caracteres)")
	}
	ref, ae := textoOpcional(r.Referencia, maxReferencia, false, "referencia_entrega_invalida", "la referencia admite hasta 250 caracteres en una línea")
	if ae != nil {
		return nil, ae
	}
	canal, ae := textoOpcional(r.Canal, maxCanal, false, "canal_invalido", "el canal admite hasta 40 caracteres en una línea")
	if ae != nil {
		return nil, ae
	}
	return &outbound.NuevoReparto{Destinatario: dest, Telefono: tel, Direccion: dir, Referencia: ref, Canal: canal}, nil
}

func validarCantidad(raw string) (vo.Cantidad, *apperror.AppError) {
	c, err := vo.NewCantidad(raw)
	if err != nil {
		return vo.Cantidad{}, mapper.Validacion("cantidad_invalida", "la cantidad debe ser mayor que cero, con hasta 11 enteros y 3 decimales")
	}
	return c, nil
}

// validarDescuento: nil → 0.
func validarDescuento(raw *string) (vo.Dinero, *apperror.AppError) {
	v := "0"
	if raw != nil {
		v = *raw
	}
	d, err := vo.NewDinero(v)
	if err != nil {
		return vo.Dinero{}, mapper.Validacion("descuento_invalido", "el descuento debe ser un importe con hasta 12 enteros y 2 decimales")
	}
	return d, nil
}

func validarInstrucciones(raw *string) (*string, *apperror.AppError) {
	return textoOpcional(raw, maxInstrucciones, true, "instrucciones_invalidas", "las instrucciones admiten hasta 2000 caracteres sin caracteres de control")
}
```

- [ ] **Step 4: Escribir el caso de uso**

Crear `service-restaurant/internal/modules/atencion/application/usecase/command/pedidos.go`:

```go
package command

import (
	"context"
	"errors"

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
	mensajeSinPermisoPedidos = "no tiene permiso para gestionar pedidos"

	CodigoMesaOcupada = "mesa_ocupada"
	CodigoClaveEnUso  = "clave_en_uso"
)

// dePedido traduce la mesa ocupada y la clave repetida a rechazos específicos y delega lo demás.
func dePedido(err error, contexto string) (appresult.Result[dtoresult.PedidoDetalle], error) {
	switch {
	case errors.Is(err, outbound.ErrMesaOcupada):
		return rechazar[dtoresult.PedidoDetalle](apperror.New(apperror.KindConflict, CodigoMesaOcupada, "la mesa ya tiene un pedido abierto"))
	case errors.Is(err, outbound.ErrClaveEnUso):
		return rechazar[dtoresult.PedidoDetalle](apperror.New(apperror.KindConflict, CodigoClaveEnUso, "la clave de idempotencia ya fue usada por otro pedido"))
	}
	return deAlmacen[dtoresult.PedidoDetalle](contexto, err)
}

// AbrirPedido crea el pedido y su cuenta nº 1 en una sola transacción (RES-RF-009).
type AbrirPedido struct{ pedidos outbound.PedidoStore }

func NewAbrirPedido(p outbound.PedidoStore) *AbrirPedido { return &AbrirPedido{p} }

func (uc *AbrirPedido) Execute(ctx context.Context, c cmd.AbrirPedido) (appresult.Result[dtoresult.PedidoDetalle], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoPedidos, policy.PedidoCrear); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	if ae := mapper.AutorizarPedidos(c.Actor, c.SucursalID, policy.PedidoCrear, mapper.NoEncontrado("sucursal")); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	tipo, ae := validarTipoPedido(c.Tipo)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	comensales, ae := validarComensales(c.Comensales)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	notas, ae := validarNotas(c.Notas)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	if ae := validarMesaSegunTipo(tipo, c.MesaID); ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}
	reparto, ae := validarReparto(tipo, c.Reparto)
	if ae != nil {
		return rechazar[dtoresult.PedidoDetalle](ae)
	}

	if tipo == vo.TipoSalon {
		mesa, found, err := uc.pedidos.MesaParaPedido(ctx, *c.MesaID)
		if err != nil {
			return tecnico[dtoresult.PedidoDetalle]("buscar mesa", err)
		}
		if !found || mesa.SucursalID != c.SucursalID {
			return rechazar[dtoresult.PedidoDetalle](mapper.Validacion("mesa_invalida", "la mesa no existe o pertenece a otra sucursal"))
		}
		if !mesa.EsActiva || !mesa.Habilitada {
			return rechazar[dtoresult.PedidoDetalle](mapper.Validacion("mesa_no_disponible", "la mesa está dada de baja o no está habilitada"))
		}
		if comensales.Int() > mesa.Capacidad {
			return rechazar[dtoresult.PedidoDetalle](mapper.Validacion("capacidad_excedida", "los comensales superan la capacidad de la mesa"))
		}
	}

	mesero, err := uc.pedidos.EmpleadoDeUsuario(ctx, c.Actor.UserID)
	if err != nil {
		return tecnico[dtoresult.PedidoDetalle]("buscar empleado del usuario", err)
	}
	p, err := uc.pedidos.CrearPedido(ctx, outbound.NuevoPedido{
		SucursalID: c.SucursalID, ClaveIdempotencia: c.Intencion.Clave, Tipo: tipo.String(), MesaID: c.MesaID,
		MeseroID: mesero, ClienteID: c.ClienteID, Comensales: comensales.Int(), Notas: notas, Reparto: reparto,
	})
	if err != nil {
		return dePedido(err, "abrir pedido")
	}
	return appresult.Ok(p), nil
}
```

- [ ] **Step 5: Ejecutar y verificar que pasa**

Run: `cd service-restaurant && go test ./internal/modules/atencion/application/usecase/command/ -count=1 2>&1 | tail -15`
Expected: PASS (los tests de mesas y zonas siguen pasando).

- [ ] **Step 6: Commit**

```bash
git add service-restaurant/internal/modules/atencion/application/usecase/command
git commit -m "$(cat <<'EOF'
I3 rebanada 1: caso de uso AbrirPedido (RF-009)

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Casos de uso AgregarDetalle y EditarDetalle

**Files:**
- Create: `service-restaurant/internal/modules/atencion/application/usecase/command/consumos.go`
- Create: `service-restaurant/internal/modules/atencion/application/usecase/command/consumos_test.go`

**Interfaces:**
- Consumes: `pedidoStoreFake`, `actorPedidos`, `idPtr` (Task 4); `validarCantidad`, `validarDescuento`, `validarInstrucciones`, `textoOpcional` (Task 4); `vo.DescuentoCabe` (Task 2).
- Produces: `command.NewAgregarDetalle(outbound.PedidoStore) *AgregarDetalle` y `command.NewEditarDetalle(outbound.PedidoStore) *EditarDetalle`, ambos con `Execute(ctx, cmd.X) (appresult.Result[dtoresult.Linea], error)`.

- [ ] **Step 1: Escribir las pruebas**

Crear `service-restaurant/internal/modules/atencion/application/usecase/command/consumos_test.go`:

```go
package command

import (
	"context"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func fakeConsumo() *pedidoStoreFake {
	return &pedidoStoreFake{
		pedido:       outbound.PedidoRef{ID: 10, SucursalID: 1, Estado: "REGISTRADO"},
		pedidoExiste: true,
		cuenta:       outbound.CuentaRef{ID: 20, PedidoID: 10, Estado: "ABIERTA"},
		cuentaExiste: true,
		producto: outbound.ProductoVenta{ID: 30, Nombre: "Lomo saltado", Tipo: "PREPARADO", AfectacionIGV: "10",
			Precio: "10.00", Tasa: "18.0000", EsActivo: true, Disponible: true},
		productoExiste: true,
		detalles:       map[int64]outbound.DetalleRef{},
	}
}

func agregarBase() cmd.AgregarDetalle {
	return cmd.AgregarDetalle{
		Intencion: cmd.Intencion{Clave: claveEjemplo},
		Actor:     actorPedidos(1, policy.PedidoEditar),
		PedidoID:  10, CuentaID: 20, ProductoID: 30, Cantidad: "2",
	}
}

func conDescuento() appctx.Actor {
	return actorPedidos(1, policy.PedidoEditar, policy.DescuentoAutorizar)
}

func TestAgregarDetalleFijaLaInstantanea(t *testing.T) {
	f := fakeConsumo()
	c := agregarBase()
	c.Cantidad = " 2.5 "
	c.Instrucciones = texto("  sin ají ")
	res, err := NewAgregarDetalle(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía agregarse: %v %+v", err, res.Err())
	}
	n := f.insertado
	if n.PedidoID != 10 || n.CuentaID != 20 || n.ProductoID != 30 || n.Descripcion != "Lomo saltado" ||
		n.Cantidad != "2.5" || n.PrecioUnitario != "10.00" || n.DescuentoTotal != "0" ||
		n.AfectacionIGV != "10" || n.TasaImpuesto != "18.0000" || n.DetallePadreID != nil ||
		n.NotasPreparacion == nil || *n.NotasPreparacion != "sin ají" {
		t.Fatalf("instantánea mal armada: %+v", n)
	}
}

func TestAgregarDetalleExoneradoVaConTasaCero(t *testing.T) {
	f := fakeConsumo()
	f.producto.AfectacionIGV, f.producto.Tasa = "20", ""
	res, err := NewAgregarDetalle(f).Execute(context.Background(), agregarBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("un producto exonerado no necesita tasa: %v %+v", err, res.Err())
	}
	if f.insertado.AfectacionIGV != "20" || f.insertado.TasaImpuesto != "0" {
		t.Fatalf("tasa: %+v", f.insertado)
	}
}

func TestAgregarDetalleConDescuentoAutorizado(t *testing.T) {
	f := fakeConsumo()
	c := agregarBase()
	c.Actor = conDescuento()
	c.DescuentoMonto = texto("5.00")
	res, err := NewAgregarDetalle(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() || f.insertado.DescuentoTotal != "5.00" {
		t.Fatalf("descuento autorizado: %v %+v %+v", err, res.Err(), f.insertado)
	}
}

func TestAgregarAdicionalConPadre(t *testing.T) {
	f := fakeConsumo()
	f.producto.Tipo = "ADICIONAL"
	f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, Estado: "ACTIVO"}
	c := agregarBase()
	c.DetallePadreID = idPtr(55)
	res, err := NewAgregarDetalle(f).Execute(context.Background(), c)
	if err != nil || !res.IsOk() || f.insertado.DetallePadreID == nil || *f.insertado.DetallePadreID != 55 {
		t.Fatalf("adicional con padre: %v %+v %+v", err, res.Err(), f.insertado)
	}
}

func TestAgregarDetalleRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *pedidoStoreFake, c *cmd.AgregarDetalle)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"cantidad cero", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Cantidad = "0" }, apperror.KindValidation, "cantidad_invalida"},
		{"cantidad negativa", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Cantidad = "-1" }, apperror.KindValidation, "cantidad_invalida"},
		{"descuento ilegible", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.DescuentoMonto = texto("abc") }, apperror.KindValidation, "descuento_invalido"},
		{"instrucciones con caracteres de control", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.Instrucciones = texto("a\x00") }, apperror.KindValidation, "instrucciones_invalidas"},
		{"pedido inexistente", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado o cancelado", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"descuento sin permiso", func(_ *pedidoStoreFake, c *cmd.AgregarDetalle) { c.DescuentoMonto = texto("1.00") }, apperror.KindForbidden, "sin_permiso"},
		{"cuenta inexistente o ajena", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.cuentaExiste = false }, apperror.KindValidation, "cuenta_invalida"},
		{"cuenta cerrada", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.cuenta.Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"producto inexistente", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.productoExiste = false }, apperror.KindValidation, "producto_invalido"},
		{"producto dado de baja", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.EsActivo = false }, apperror.KindValidation, "producto_inactivo"},
		{"producto no disponible en la sede", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.Disponible = false }, apperror.KindValidation, "producto_no_disponible"},
		{"gravado sin tasa vigente", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.Tasa = "" }, apperror.KindValidation, "tasa_no_vigente"},
		{"descuento mayor que el importe", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), "1", texto("10.01")
		}, apperror.KindValidation, "descuento_excede_importe"},
		{"adicional sin padre", func(f *pedidoStoreFake, _ *cmd.AgregarDetalle) { f.producto.Tipo = "ADICIONAL" }, apperror.KindValidation, "adicional_sin_padre"},
		{"padre en un producto que no es adicional", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, Estado: "ACTIVO"}
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_no_aplica"},
		{"padre inexistente", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.producto.Tipo = "ADICIONAL"
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_invalido"},
		{"padre de otra cuenta", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.producto.Tipo = "ADICIONAL"
			f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 21, Estado: "ACTIVO"}
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_invalido"},
		{"padre anulado", func(f *pedidoStoreFake, c *cmd.AgregarDetalle) {
			f.producto.Tipo = "ADICIONAL"
			f.detalles[55] = outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, Estado: "ANULADO"}
			c.DetallePadreID = idPtr(55)
		}, apperror.KindValidation, "padre_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := fakeConsumo()
			in := agregarBase()
			c.mod(f, &in)
			res, err := NewAgregarDetalle(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
			if f.inserciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}

func detalleActivo() outbound.DetalleRef {
	return outbound.DetalleRef{ID: 55, PedidoID: 10, CuentaID: 20, ProductoID: 30,
		Cantidad: "2.000", PrecioUnitario: "10.000000", DescuentoTotal: "0.00", Estado: "ACTIVO"}
}

func fakeEdicion() *pedidoStoreFake {
	f := fakeConsumo()
	f.detalles[55] = detalleActivo()
	return f
}

func editarBase() cmd.EditarDetalle {
	return cmd.EditarDetalle{Actor: actorPedidos(1, policy.PedidoEditar), PedidoID: 10, DetalleID: 55, Cantidad: texto("3")}
}

func TestEditarDetalle(t *testing.T) {
	f := fakeEdicion()
	res, err := NewEditarDetalle(f).Execute(context.Background(), editarBase())
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	if f.actualizaciones != 1 || f.cambios.Cantidad == nil || *f.cambios.Cantidad != "3" ||
		f.cambios.DescuentoTotal != nil || f.cambios.Instrucciones.Presente {
		t.Fatalf("cambios mal armados: %+v", f.cambios)
	}

	// Vaciar las instrucciones (null) se distingue de no enviarlas.
	f = fakeEdicion()
	c := editarBase()
	c.Cantidad = nil
	c.Instrucciones = cmd.Campo[string]{Presente: true}
	if res, err := NewEditarDetalle(f).Execute(context.Background(), c); err != nil || !res.IsOk() ||
		!f.cambios.Instrucciones.Presente || f.cambios.Instrucciones.Valor != nil {
		t.Fatalf("vaciar instrucciones: %v %+v %+v", err, res.Err(), f.cambios)
	}

	// Con permiso de descuento se puede fijar uno.
	f = fakeEdicion()
	c = editarBase()
	c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), nil, texto("5.00")
	if res, err := NewEditarDetalle(f).Execute(context.Background(), c); err != nil || !res.IsOk() ||
		f.cambios.DescuentoTotal == nil || *f.cambios.DescuentoTotal != "5.00" {
		t.Fatalf("descuento: %v %+v %+v", err, res.Err(), f.cambios)
	}
}

func TestEditarDetalleRechazos(t *testing.T) {
	casos := []struct {
		nombre string
		mod    func(f *pedidoStoreFake, c *cmd.EditarDetalle)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Actor = actorPedidos(1, policy.PedidoVer) }, apperror.KindForbidden, "sin_permiso"},
		{"sin cambios", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Cantidad = nil }, apperror.KindValidation, "sin_cambios"},
		{"cantidad cero", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Cantidad = texto("0") }, apperror.KindValidation, "cantidad_invalida"},
		{"pedido inexistente", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.pedidoExiste = false }, apperror.KindNotFound, "no_encontrado"},
		{"pedido de otra sede", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.pedido.SucursalID = 2 }, apperror.KindNotFound, "no_encontrado"},
		{"pedido cerrado", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.pedido.Cerrado = true }, apperror.KindConflict, "pedido_cerrado"},
		{"línea inexistente", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { delete(f.detalles, 55) }, apperror.KindNotFound, "no_encontrado"},
		{"línea anulada", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) {
			d := f.detalles[55]
			d.Estado = "ANULADO"
			f.detalles[55] = d
		}, apperror.KindConflict, "linea_anulada"},
		{"línea enviada a cocina", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) {
			d := f.detalles[55]
			d.EnComanda = true
			f.detalles[55] = d
		}, apperror.KindConflict, "linea_en_cocina"},
		{"cuenta cerrada", func(f *pedidoStoreFake, _ *cmd.EditarDetalle) { f.cuenta.Estado = "CERRADA" }, apperror.KindConflict, "cuenta_cerrada"},
		{"descuento sin permiso", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) { c.Cantidad, c.DescuentoMonto = nil, texto("1.00") }, apperror.KindForbidden, "sin_permiso"},
		{"descuento ilegible", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) {
			c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), nil, texto("x")
		}, apperror.KindValidation, "descuento_invalido"},
		{"descuento mayor que el importe", func(_ *pedidoStoreFake, c *cmd.EditarDetalle) {
			c.Actor, c.Cantidad, c.DescuentoMonto = conDescuento(), nil, texto("20.01")
		}, apperror.KindValidation, "descuento_excede_importe"},
		{"bajar la cantidad deja el descuento por encima del importe", func(f *pedidoStoreFake, c *cmd.EditarDetalle) {
			d := f.detalles[55]
			d.DescuentoTotal = "15.00"
			f.detalles[55] = d
			c.Cantidad = texto("1")
		}, apperror.KindValidation, "descuento_excede_importe"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f := fakeEdicion()
			in := editarBase()
			c.mod(f, &in)
			res, err := NewEditarDetalle(f).Execute(context.Background(), in)
			sinFallo(t, err)
			if res.Err() == nil {
				t.Fatal("debía rechazarse")
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
			if f.actualizaciones != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `cd service-restaurant && go test ./internal/modules/atencion/application/usecase/command/ -run 'TestAgregar|TestEditarDetalle' -count=1`
Expected: FAIL de compilación (`undefined: NewAgregarDetalle`, `NewEditarDetalle`).

- [ ] **Step 3: Implementar los casos de uso**

Crear `service-restaurant/internal/modules/atencion/application/usecase/command/consumos.go`:

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
	estadoActivo     = "ACTIVO"
	estadoCuentaAbta = "ABIERTA"
	tipoAdicional    = "ADICIONAL"
	afectacionIGV    = "10"

	mensajeSinPermisoDescuento = "no tiene permiso para aplicar descuentos"
)

func sinPermisoDescuento() *apperror.AppError {
	return apperror.New(apperror.KindForbidden, mapper.CodigoSinPermiso, mensajeSinPermisoDescuento)
}

func descuentoExcede() *apperror.AppError {
	return mapper.Validacion("descuento_excede_importe", "el descuento no puede superar el importe de la línea")
}

func pedidoCerrado() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "pedido_cerrado", "el pedido ya está cerrado o cancelado")
}

func cuentaCerrada() *apperror.AppError {
	return apperror.New(apperror.KindConflict, "cuenta_cerrada", "la cuenta ya no está abierta")
}

// AgregarDetalle agrega una línea a una cuenta abierta fijando descripción, precio y tasa desde
// la carta en este instante (RES-RF-010).
type AgregarDetalle struct{ pedidos outbound.PedidoStore }

func NewAgregarDetalle(p outbound.PedidoStore) *AgregarDetalle { return &AgregarDetalle{p} }

func (uc *AgregarDetalle) Execute(ctx context.Context, c cmd.AgregarDetalle) (appresult.Result[dtoresult.Linea], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoPedidos, policy.PedidoEditar); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	cantidad, ae := validarCantidad(c.Cantidad)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	descuento, ae := validarDescuento(c.DescuentoMonto)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	instrucciones, ae := validarInstrucciones(c.Instrucciones)
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}

	ped, found, err := uc.pedidos.BloquearPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pedido", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.NoEncontradoM("pedido"))
	}
	if ae := mapper.AutorizarPedidos(c.Actor, ped.SucursalID, policy.PedidoEditar, mapper.NoEncontradoM("pedido")); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if ped.Cerrado {
		return rechazar[dtoresult.Linea](pedidoCerrado())
	}
	if !descuento.EsCero() && !c.Actor.Can(policy.DescuentoAutorizar, ped.SucursalID) {
		return rechazar[dtoresult.Linea](sinPermisoDescuento())
	}

	cuenta, found, err := uc.pedidos.CuentaDelPedido(ctx, c.PedidoID, c.CuentaID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar cuenta", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.Validacion("cuenta_invalida", "la cuenta no existe o no pertenece al pedido"))
	}
	if cuenta.Estado != estadoCuentaAbta {
		return rechazar[dtoresult.Linea](cuentaCerrada())
	}

	prod, found, err := uc.pedidos.ProductoParaVenta(ctx, c.ProductoID, ped.SucursalID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar producto", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.Validacion("producto_invalido", "el producto no existe"))
	}
	if !prod.EsActivo {
		return rechazar[dtoresult.Linea](mapper.Validacion("producto_inactivo", "el producto está dado de baja"))
	}
	if !prod.Disponible {
		return rechazar[dtoresult.Linea](mapper.Validacion("producto_no_disponible", "el producto no está disponible en esta sucursal"))
	}
	tasa := "0"
	if prod.AfectacionIGV == afectacionIGV {
		if prod.Tasa == "" {
			return rechazar[dtoresult.Linea](mapper.Validacion("tasa_no_vigente", "el impuesto del producto no tiene una tasa vigente"))
		}
		tasa = prod.Tasa
	}

	ae, err = uc.validarPadre(ctx, c, prod.Tipo)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar línea padre", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if !vo.DescuentoCabe(cantidad, prod.Precio, descuento) {
		return rechazar[dtoresult.Linea](descuentoExcede())
	}

	l, err := uc.pedidos.InsertarDetalle(ctx, outbound.NuevoDetalle{
		PedidoID: c.PedidoID, CuentaID: c.CuentaID, ProductoID: prod.ID, DetallePadreID: c.DetallePadreID,
		Descripcion: prod.Nombre, Cantidad: cantidad.String(), PrecioUnitario: prod.Precio,
		DescuentoTotal: descuento.String(), AfectacionIGV: prod.AfectacionIGV, TasaImpuesto: tasa,
		NotasPreparacion: instrucciones,
	})
	if err != nil {
		return deAlmacen[dtoresult.Linea]("agregar detalle", err)
	}
	return appresult.Ok(l), nil
}
```

Continúa en el mismo archivo `consumos.go` (añadir al final):

```go
// validarPadre aplica la regla de los adicionales: un ADICIONAL cuelga de una línea activa de la
// misma cuenta, y solo un ADICIONAL puede colgar de otra línea.
func (uc *AgregarDetalle) validarPadre(ctx context.Context, c cmd.AgregarDetalle, tipoProducto string) (*apperror.AppError, error) {
	esAdicional := tipoProducto == tipoAdicional
	switch {
	case c.DetallePadreID == nil && esAdicional:
		return mapper.Validacion("adicional_sin_padre", "un adicional debe indicar la línea de la que cuelga"), nil
	case c.DetallePadreID != nil && !esAdicional:
		return mapper.Validacion("padre_no_aplica", "solo un adicional puede colgar de otra línea"), nil
	case c.DetallePadreID == nil:
		return nil, nil
	}
	padre, found, err := uc.pedidos.DetalleDelPedido(ctx, c.PedidoID, *c.DetallePadreID)
	if err != nil {
		return nil, err
	}
	if !found || padre.CuentaID != c.CuentaID || padre.Estado != estadoActivo {
		return mapper.Validacion("padre_invalido", "la línea padre no existe, está anulada o es de otra cuenta"), nil
	}
	return nil, nil
}

// EditarDetalle modifica cantidad, instrucciones o descuento de una línea activa de una cuenta
// abierta, mientras no se haya enviado a cocina.
type EditarDetalle struct{ pedidos outbound.PedidoStore }

func NewEditarDetalle(p outbound.PedidoStore) *EditarDetalle { return &EditarDetalle{p} }

func (uc *EditarDetalle) Execute(ctx context.Context, c cmd.EditarDetalle) (appresult.Result[dtoresult.Linea], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoPedidos, policy.PedidoEditar); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if c.Cantidad == nil && c.DescuentoMonto == nil && !c.Instrucciones.Presente {
		return rechazar[dtoresult.Linea](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosDetalle
	var nuevaCantidad *vo.Cantidad
	var nuevoDescuento *vo.Dinero
	if c.Cantidad != nil {
		q, ae := validarCantidad(*c.Cantidad)
		if ae != nil {
			return rechazar[dtoresult.Linea](ae)
		}
		s := q.String()
		cambios.Cantidad, nuevaCantidad = &s, &q
	}
	if c.DescuentoMonto != nil {
		d, ae := validarDescuento(c.DescuentoMonto)
		if ae != nil {
			return rechazar[dtoresult.Linea](ae)
		}
		s := d.String()
		cambios.DescuentoTotal, nuevoDescuento = &s, &d
	}
	if c.Instrucciones.Presente {
		v, ae := validarInstrucciones(c.Instrucciones.Valor)
		if ae != nil {
			return rechazar[dtoresult.Linea](ae)
		}
		cambios.Instrucciones = outbound.CampoOpcional{Presente: true, Valor: v}
	}

	ped, found, err := uc.pedidos.BloquearPedido(ctx, c.PedidoID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar pedido", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.NoEncontradoM("pedido"))
	}
	if ae := mapper.AutorizarPedidos(c.Actor, ped.SucursalID, policy.PedidoEditar, mapper.NoEncontradoM("pedido")); ae != nil {
		return rechazar[dtoresult.Linea](ae)
	}
	if ped.Cerrado {
		return rechazar[dtoresult.Linea](pedidoCerrado())
	}
	if nuevoDescuento != nil && !c.Actor.Can(policy.DescuentoAutorizar, ped.SucursalID) {
		return rechazar[dtoresult.Linea](sinPermisoDescuento())
	}

	det, found, err := uc.pedidos.DetalleDelPedido(ctx, c.PedidoID, c.DetalleID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar línea", err)
	}
	if !found {
		return rechazar[dtoresult.Linea](mapper.NoEncontrado("línea"))
	}
	if det.Estado != estadoActivo {
		return rechazar[dtoresult.Linea](apperror.New(apperror.KindConflict, "linea_anulada", "la línea está anulada"))
	}
	if det.EnComanda {
		return rechazar[dtoresult.Linea](apperror.New(apperror.KindConflict, "linea_en_cocina", "la línea ya se envió a cocina"))
	}
	cuenta, found, err := uc.pedidos.CuentaDelPedido(ctx, c.PedidoID, det.CuentaID)
	if err != nil {
		return tecnico[dtoresult.Linea]("buscar cuenta", err)
	}
	if !found || cuenta.Estado != estadoCuentaAbta {
		return rechazar[dtoresult.Linea](cuentaCerrada())
	}

	// El descuento resultante (nuevo o el que ya tenía) debe caber en el importe resultante.
	if nuevaCantidad == nil {
		q, err := vo.NewCantidad(det.Cantidad)
		if err != nil {
			return tecnico[dtoresult.Linea]("leer la cantidad de la línea", err)
		}
		nuevaCantidad = &q
	}
	if nuevoDescuento == nil {
		d, err := vo.NewDinero(det.DescuentoTotal)
		if err != nil {
			return tecnico[dtoresult.Linea]("leer el descuento de la línea", err)
		}
		nuevoDescuento = &d
	}
	if !vo.DescuentoCabe(*nuevaCantidad, det.PrecioUnitario, *nuevoDescuento) {
		return rechazar[dtoresult.Linea](descuentoExcede())
	}

	l, err := uc.pedidos.ActualizarDetalle(ctx, c.DetalleID, cambios)
	if err != nil {
		return deAlmacen[dtoresult.Linea]("editar detalle", err)
	}
	return appresult.Ok(l), nil
}
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `cd service-restaurant && go vet ./internal/modules/atencion/... && go test ./internal/modules/atencion/application/usecase/command/ -count=1 2>&1 | tail -20`
Expected: PASS. Si `go vet` marca `ae` o `err` sin usar o redeclarados, ajustar las declaraciones (`:=` frente a `=`) sin cambiar la lógica.

- [ ] **Step 5: Commit**

```bash
git add service-restaurant/internal/modules/atencion/application/usecase/command
git commit -m "$(cat <<'EOF'
I3 rebanada 1: casos de uso AgregarDetalle y EditarDetalle (RF-010)

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Consultas ListarPedidos y ObtenerPedido

**Files:**
- Modify: `service-restaurant/internal/modules/atencion/application/usecase/query/sedes.go`
- Create: `service-restaurant/internal/modules/atencion/application/usecase/query/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/application/usecase/query/pedidos_test.go`

**Interfaces:**
- Consumes: `outbound.PedidoReader`, `outbound.FiltroPedidos`, `mapper.SedesConPedidos`, `mapper.PuedeVerPedidos`, helper existente `tamanio(limite int) int` (en el paquete `query`; lo usa `ListarMesas`).
- Produces: `query.NewListarPedidos(outbound.PedidoReader)` y `query.NewObtenerPedido(outbound.PedidoReader)`.

- [ ] **Step 1: Escribir las pruebas**

Crear `service-restaurant/internal/modules/atencion/application/usecase/query/pedidos_test.go`:

```go
package query

import (
	"context"
	"slices"
	"testing"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type pedidoReaderFake struct {
	items    []dtoresult.Pedido
	detalle  dtoresult.PedidoDetalle
	existe   bool
	filtro   outbound.FiltroPedidos
	llamadas int
}

func (f *pedidoReaderFake) ListarPedidos(_ context.Context, fl outbound.FiltroPedidos) ([]dtoresult.Pedido, error) {
	f.llamadas++
	f.filtro = fl
	return f.items, nil
}
func (f *pedidoReaderFake) ObtenerPedido(context.Context, int64) (dtoresult.PedidoDetalle, bool, error) {
	return f.detalle, f.existe, nil
}

func lectorDePedidos(grants ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, grants) }

func TestListarPedidos(t *testing.T) {
	actor := lectorDePedidos(
		appctx.Grant{SucursalID: 2, Permiso: policy.PedidoVer},
		appctx.Grant{SucursalID: 1, Permiso: policy.PedidoCrear},
	)

	t.Run("sin permiso", func(t *testing.T) {
		res, err := NewListarPedidos(&pedidoReaderFake{}).Execute(context.Background(),
			qry.ListarPedidos{Actor: lectorDePedidos(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer}), Limite: 10})
		if err != nil || res.Err() == nil || res.Err().Kind != apperror.KindForbidden || res.Err().Code != mapper.CodigoSinPermiso {
			t.Fatalf("debía ser 403: %v %+v", err, res.Err())
		}
	})

	t.Run("sede que no ve", func(t *testing.T) {
		f := &pedidoReaderFake{}
		res, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, SucursalID: idPtr(9), Limite: 10})
		if err != nil || res.Err() == nil || res.Err().Kind != apperror.KindNotFound || f.llamadas != 0 {
			t.Fatalf("debía ser 404 sin consultar: %v %+v", err, res.Err())
		}
	})

	t.Run("filtra por las sedes visibles y pasa los filtros", func(t *testing.T) {
		f := &pedidoReaderFake{}
		_, err := NewListarPedidos(f).Execute(context.Background(),
			qry.ListarPedidos{Actor: actor, Estado: "REGISTRADO", Tipo: "SALON", MesaID: idPtr(4), Limite: 10})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(f.filtro.SedeIDs, []int64{1, 2}) || f.filtro.Estado != "REGISTRADO" || f.filtro.Tipo != "SALON" ||
			f.filtro.MesaID == nil || *f.filtro.MesaID != 4 || f.filtro.Limite != 11 {
			t.Fatalf("filtro: %+v", f.filtro)
		}
	})

	t.Run("una sede concreta", func(t *testing.T) {
		f := &pedidoReaderFake{}
		_, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, SucursalID: idPtr(2), Limite: 10})
		if err != nil || !slices.Equal(f.filtro.SedeIDs, []int64{2}) {
			t.Fatalf("debía acotar a la sede 2: %v %+v", err, f.filtro)
		}
	})

	t.Run("pagina con cursor", func(t *testing.T) {
		f := &pedidoReaderFake{items: []dtoresult.Pedido{{ID: 1}, {ID: 2}, {ID: 3}}}
		res, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Limite: 2})
		if err != nil || !res.IsOk() {
			t.Fatalf("%v %+v", err, res.Err())
		}
		p := res.Value()
		if len(p.Items) != 2 || p.NextCursor != mapper.EncodeCursor(2) {
			t.Fatalf("página: %+v", p)
		}
		desde, _ := mapper.DecodeCursor(p.NextCursor)
		if _, err := NewListarPedidos(f).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Limite: 2, Cursor: p.NextCursor}); err != nil || f.filtro.DespuesDeID != desde {
			t.Fatalf("el cursor debe pasar al filtro: %v %+v", err, f.filtro)
		}
	})

	t.Run("cursor inválido", func(t *testing.T) {
		res, err := NewListarPedidos(&pedidoReaderFake{}).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Cursor: "!!!", Limite: 10})
		if err != nil || res.Err() == nil || res.Err().Code != mapper.CodigoCursorInvalido {
			t.Fatalf("debía rechazar el cursor: %v %+v", err, res.Err())
		}
	})

	t.Run("sin resultados devuelve lista vacía", func(t *testing.T) {
		res, _ := NewListarPedidos(&pedidoReaderFake{}).Execute(context.Background(), qry.ListarPedidos{Actor: actor, Limite: 10})
		if res.Value().Items == nil || len(res.Value().Items) != 0 || res.Value().NextCursor != "" {
			t.Fatalf("página vacía: %+v", res.Value())
		}
	})
}

func TestObtenerPedido(t *testing.T) {
	actor := lectorDePedidos(appctx.Grant{SucursalID: 1, Permiso: policy.PedidoVer})
	f := &pedidoReaderFake{existe: true, detalle: dtoresult.PedidoDetalle{Pedido: dtoresult.Pedido{ID: 5, SucursalID: 1}}}

	if res, err := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: actor, ID: 5}); err != nil || !res.IsOk() || res.Value().ID != 5 {
		t.Fatalf("debía devolver el pedido: %v %+v", err, res.Err())
	}
	// Pedido de una sede que el actor no ve: 404, igual que uno inexistente.
	f.detalle.SucursalID = 2
	if res, err := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: actor, ID: 5}); err != nil || res.Err() == nil ||
		res.Err().Kind != apperror.KindNotFound || res.Err().Message != "pedido no encontrado" {
		t.Fatalf("sede ajena debía ser 404: %v %+v", err, res.Err())
	}
	f.existe = false
	if res, _ := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: actor, ID: 5}); res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("inexistente debía ser 404: %+v", res.Err())
	}
	// Sin permiso de pedidos: 403.
	otro := lectorDePedidos(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	if res, _ := NewObtenerPedido(f).Execute(context.Background(), qry.ObtenerPedido{Actor: otro, ID: 5}); res.Err() == nil || res.Err().Kind != apperror.KindForbidden {
		t.Fatalf("sin permiso debía ser 403: %+v", res.Err())
	}
}

func idPtr(n int64) *int64 { return &n }
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `cd service-restaurant && go test ./internal/modules/atencion/application/usecase/query/ -run 'TestListarPedidos|TestObtenerPedido' -count=1`
Expected: FAIL de compilación (`undefined: NewListarPedidos`). Si `idPtr` ya existe en otro archivo de prueba del paquete `query`, el compilador lo dirá: borrar la copia de este archivo.

- [ ] **Step 3: Implementar**

Reemplazar el contenido de `service-restaurant/internal/modules/atencion/application/usecase/query/sedes.go` por:

```go
package query

import (
	"slices"

	"service-restaurant/internal/modules/atencion/application/mapper"
	appctx "service-restaurant/internal/shared/application/context"
)

// sedesPedidas devuelve las sedes que alcanza un listado de mesas: las visibles del actor o, si
// pidió una sucursal concreta, solo esa. ok=false si pidió una sucursal que no ve.
func sedesPedidas(a appctx.Actor, sucursalID *int64) ([]int64, bool) {
	return acotarSedes(mapper.SedesVisibles(a), sucursalID)
}

// sedesDePedidos es como sedesPedidas para los permisos de pedidos.
func sedesDePedidos(a appctx.Actor, sucursalID *int64) ([]int64, bool) {
	return acotarSedes(mapper.SedesConPedidos(a), sucursalID)
}

func acotarSedes(visibles []int64, sucursalID *int64) ([]int64, bool) {
	if sucursalID == nil {
		return visibles, true
	}
	if !slices.Contains(visibles, *sucursalID) {
		return nil, false
	}
	return []int64{*sucursalID}, true
}
```

Crear `service-restaurant/internal/modules/atencion/application/usecase/query/pedidos.go`:

```go
package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoPedidos = "no tiene permiso para consultar pedidos"

// ListarPedidos devuelve los pedidos de las sedes visibles. Pide limite+1 filas para saber si hay
// más páginas.
type ListarPedidos struct{ pedidos outbound.PedidoReader }

func NewListarPedidos(p outbound.PedidoReader) *ListarPedidos { return &ListarPedidos{p} }

func (uc *ListarPedidos) Execute(ctx context.Context, q qry.ListarPedidos) (appresult.Result[dtoresult.PaginaPedidos], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoPedidos, policy.PedidoVer, policy.PedidoCrear, policy.PedidoEditar); ae != nil {
		return appresult.Fail[dtoresult.PaginaPedidos](ae), nil
	}
	sedes, ok := sedesDePedidos(q.Actor, q.SucursalID)
	if !ok {
		return appresult.Fail[dtoresult.PaginaPedidos](mapper.NoEncontrado("sucursal")), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaPedidos](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.pedidos.ListarPedidos(ctx, outbound.FiltroPedidos{
		SedeIDs: sedes, Estado: q.Estado, Tipo: q.Tipo, MesaID: q.MesaID, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaPedidos]{}, fmt.Errorf("listar pedidos: %w", err)
	}
	pagina := dtoresult.PaginaPedidos{Items: items}
	if len(items) > n {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Pedido{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerPedido lee un pedido con sus cuentas y líneas; el de una sede que el actor no ve
// responde 404, igual que uno inexistente.
type ObtenerPedido struct{ pedidos outbound.PedidoReader }

func NewObtenerPedido(p outbound.PedidoReader) *ObtenerPedido { return &ObtenerPedido{p} }

func (uc *ObtenerPedido) Execute(ctx context.Context, q qry.ObtenerPedido) (appresult.Result[dtoresult.PedidoDetalle], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoPedidos, policy.PedidoVer, policy.PedidoCrear, policy.PedidoEditar); ae != nil {
		return appresult.Fail[dtoresult.PedidoDetalle](ae), nil
	}
	p, found, err := uc.pedidos.ObtenerPedido(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.PedidoDetalle]{}, fmt.Errorf("obtener pedido: %w", err)
	}
	if !found || !mapper.PuedeVerPedidos(q.Actor, p.SucursalID) {
		return appresult.Fail[dtoresult.PedidoDetalle](mapper.NoEncontradoM("pedido")), nil
	}
	return appresult.Ok(p), nil
}
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `cd service-restaurant && go test ./internal/modules/atencion/application/usecase/query/ -count=1 2>&1 | tail -15`
Expected: PASS (los tests de mesas y zonas siguen pasando con `sedesPedidas` refactorizada).

- [ ] **Step 5: Commit**

```bash
git add service-restaurant/internal/modules/atencion/application/usecase/query
git commit -m "$(cat <<'EOF'
I3 rebanada 1: consultas ListarPedidos y ObtenerPedido

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Persistencia (columnas, ensamblado, escritura y lectura)

**Files:**
- Create: `service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar/pedido.go`
- Create: `service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository/pedidos_sql.go`
- Create: `service-restaurant/internal/modules/atencion/infrastructure/persistence/write/adapter/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository/pedidos_sql_reader.go`
- Create: `service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter/pedidos_read_adapter.go`

**Interfaces:**
- Consumes: `outbound.PedidoStore`, `outbound.PedidoReader` y los tipos de la Task 3; `database.DBTX`; `columnas.Fila`, `columnas.textoNulo` (mismo paquete `columnas`).
- Produces: `adapter.NewPedidoWriteAdapter(db database.DBTX) *PedidoWriteAdapter` (implementa `outbound.PedidoStore`) y `readadapter.NewPedidoReadAdapter(db database.DBTX) *PedidoReadAdapter` (implementa `outbound.PedidoReader`).

Esta capa no tiene pruebas unitarias propias: igual que mesas, se verifica con compilación aquí y con las pruebas de integración de la Task 10, que ejercen cada consulta contra PostgreSQL.

- [ ] **Step 1: Columnas y escaneo**

Crear `service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas/pedidos.go`:

```go
package columnas

import (
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// Las cantidades y los importes son numeric: se leen como texto para no perder exactitud.
const (
	Pedido = `id, sucursal_id, codigo_publico::text, clave_idempotencia::text, mesa_id, reserva_id, cliente_id, mesero_id,
		tipo, estado, moneda, comensales, abierto_en, cerrado_en, notas, motivo_cancelacion, version_bloqueo,
		creado_en, creado_por, actualizado_en, actualizado_por`

	Cuenta = `id, numero, cliente_id, etiqueta, estado`

	Linea = `id, pedido_id, cuenta_id, producto_id, detalle_padre_id, descripcion, cantidad::text, precio_unitario::text,
		descuento_total::text, afectacion_igv, tasa_impuesto::text, importe_total::text, base_imponible::text, estado,
		notas_preparacion, creado_en, creado_por, actualizado_en, actualizado_por`
)

func enteroNulo(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

// ScanPedido lee una fila con las columnas de Pedido.
func ScanPedido(s Fila) (dtoresult.Pedido, error) {
	var p dtoresult.Pedido
	var mesa, reserva, cliente, mesero sql.NullInt64
	var cerrado sql.NullTime
	var notas, motivo sql.NullString
	err := s.Scan(&p.ID, &p.SucursalID, &p.CodigoPublico, &p.ClaveIdempotencia, &mesa, &reserva, &cliente, &mesero,
		&p.Tipo, &p.Estado, &p.Moneda, &p.Comensales, &p.AbiertoEn, &cerrado, &notas, &motivo, &p.VersionBloqueo,
		&p.CreadoEn, &p.CreadoPor, &p.ActualizadoEn, &p.ActualizadoPor)
	p.MesaID, p.ReservaID, p.ClienteID, p.MeseroID = enteroNulo(mesa), enteroNulo(reserva), enteroNulo(cliente), enteroNulo(mesero)
	if cerrado.Valid {
		t := cerrado.Time
		p.CerradoEn = &t
	}
	p.Notas, p.MotivoCancelacion = textoNulo(notas), textoNulo(motivo)
	return p, err
}

// ScanCuenta lee una fila con las columnas de Cuenta (sin líneas ni totales).
func ScanCuenta(s Fila) (dtoresult.Cuenta, error) {
	var c dtoresult.Cuenta
	var cliente sql.NullInt64
	var etiqueta sql.NullString
	err := s.Scan(&c.ID, &c.Numero, &cliente, &etiqueta, &c.Estado)
	c.ClienteID, c.Etiqueta = enteroNulo(cliente), textoNulo(etiqueta)
	return c, err
}

// ScanLinea lee una fila con las columnas de Linea.
func ScanLinea(s Fila) (dtoresult.Linea, error) {
	var l dtoresult.Linea
	var padre sql.NullInt64
	var notas sql.NullString
	err := s.Scan(&l.ID, &l.PedidoID, &l.CuentaID, &l.ProductoID, &padre, &l.Descripcion, &l.Cantidad, &l.PrecioUnitario,
		&l.DescuentoTotal, &l.AfectacionIGV, &l.TasaImpuesto, &l.ImporteTotal, &l.BaseImponible, &l.Estado,
		&notas, &l.CreadoEn, &l.CreadoPor, &l.ActualizadoEn, &l.ActualizadoPor)
	l.DetallePadreID, l.NotasPreparacion = enteroNulo(padre), textoNulo(notas)
	return l, err
}
```

- [ ] **Step 2: Ensamblado de un pedido**

Crear `service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar/pedido.go`:

```go
// Package cargar arma un pedido completo (reparto, cuentas, líneas y totales) con consultas que
// comparten el lado de lectura y el de escritura. Vive bajo persistence/internal: solo la
// persistencia del módulo puede importarlo.
package cargar

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// Pedido lee el pedido con su reparto, sus cuentas, las líneas de cada cuenta y los totales de las
// líneas activas. found=false si no existe. Cada consulta se agota antes de lanzar la siguiente:
// sobre una transacción no puede haber dos resultados abiertos a la vez.
func Pedido(ctx context.Context, db database.DBTX, id int64) (dtoresult.PedidoDetalle, bool, error) {
	p, err := columnas.ScanPedido(db.QueryRowContext(ctx,
		`SELECT `+columnas.Pedido+` FROM sch_restaurante.pedidos WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.PedidoDetalle{}, false, nil
	}
	if err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	d := dtoresult.PedidoDetalle{Pedido: p, Cuentas: []dtoresult.Cuenta{}}
	if d.Reparto, err = leerReparto(ctx, db, id); err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	if d.Cuentas, err = leerCuentas(ctx, db, id); err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	lineas, err := leerLineas(ctx, db, id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	totales, err := leerTotales(ctx, db, id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, false, err
	}
	for i := range d.Cuentas {
		c := &d.Cuentas[i]
		c.Lineas = []dtoresult.Linea{}
		c.Subtotal, c.Impuesto, c.Total = "0.00", "0.00", "0.00"
		if t, ok := totales[c.ID]; ok {
			c.Subtotal, c.Impuesto, c.Total = t[0], t[1], t[2]
		}
		for _, l := range lineas {
			if l.CuentaID == c.ID {
				c.Lineas = append(c.Lineas, l)
			}
		}
	}
	return d, true, nil
}

func leerReparto(ctx context.Context, db database.DBTX, pedidoID int64) (*dtoresult.Reparto, error) {
	var r dtoresult.Reparto
	var ref sql.NullString
	err := db.QueryRowContext(ctx, `SELECT destinatario, telefono, direccion, referencia, canal, estado
		FROM sch_restaurante.pedidos_delivery WHERE pedido_id = $1`, pedidoID).
		Scan(&r.Destinatario, &r.Telefono, &r.Direccion, &ref, &r.Canal, &r.Estado)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if ref.Valid {
		v := ref.String
		r.Referencia = &v
	}
	return &r, nil
}

func leerCuentas(ctx context.Context, db database.DBTX, pedidoID int64) ([]dtoresult.Cuenta, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+columnas.Cuenta+` FROM sch_restaurante.cuentas WHERE pedido_id = $1 ORDER BY numero`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dtoresult.Cuenta{}
	for rows.Next() {
		c, err := columnas.ScanCuenta(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func leerLineas(ctx context.Context, db database.DBTX, pedidoID int64) ([]dtoresult.Linea, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+columnas.Linea+` FROM sch_restaurante.detalles_pedido WHERE pedido_id = $1 ORDER BY id`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Linea
	for rows.Next() {
		l, err := columnas.ScanLinea(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// leerTotales devuelve, por cuenta, [subtotal, impuesto, total] de las líneas activas.
func leerTotales(ctx context.Context, db database.DBTX, pedidoID int64) (map[int64][3]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT cuenta_id, sum(base_imponible)::text, (sum(importe_total) - sum(base_imponible))::text, sum(importe_total)::text
		FROM sch_restaurante.detalles_pedido WHERE pedido_id = $1 AND estado = 'ACTIVO' GROUP BY cuenta_id`, pedidoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][3]string{}
	for rows.Next() {
		var cuenta int64
		var t [3]string
		if err := rows.Scan(&cuenta, &t[0], &t[1], &t[2]); err != nil {
			return nil, err
		}
		out[cuenta] = t
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: SQL de escritura**

Crear `service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository/pedidos_sql.go`:

```go
package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// Restricciones únicas que tienen un rechazo propio.
const (
	restriccionMesaOcupada = "uq_pedido_mesa_abierta"
	restriccionClavePedido = "pedidos_clave_idempotencia_key"
)

// traducirPedido distingue la mesa ocupada y la clave repetida por el nombre de la restricción.
func traducirPedido(err error) error {
	switch restriccionUnica(err) {
	case restriccionMesaOcupada:
		return outbound.ErrMesaOcupada
	case restriccionClavePedido:
		return outbound.ErrClaveEnUso
	}
	return Translate(err)
}

// PedidoSQLWriter opera sobre pedidos, cuentas, pedidos_delivery y detalles_pedido.
type PedidoSQLWriter struct{ db database.DBTX }

func NewPedidoSQLWriter(db database.DBTX) *PedidoSQLWriter { return &PedidoSQLWriter{db} }

// GetMesa lee la mesa en modo compartido: no puede deshabilitarse hasta que termine el comando.
func (r *PedidoSQLWriter) GetMesa(ctx context.Context, id int64) (outbound.MesaRef, bool, error) {
	var m outbound.MesaRef
	err := r.db.QueryRowContext(ctx, `SELECT sucursal_id, capacidad, habilitada, es_activo
		FROM sch_restaurante.mesas WHERE id = $1 FOR SHARE`, id).Scan(&m.SucursalID, &m.Capacidad, &m.Habilitada, &m.EsActiva)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.MesaRef{}, false, err
	}
	return m, true, nil
}

// GetEmpleadoDeUsuario devuelve el empleado activo vinculado al usuario, o nil.
func (r *PedidoSQLWriter) GetEmpleadoDeUsuario(ctx context.Context, usuarioID int64) (*int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM sch_restaurante.empleados WHERE usuario_id = $1 AND es_activo`, usuarioID).Scan(&id)
	if missing, err := notFound(err); missing || err != nil {
		return nil, err
	}
	return &id, nil
}

// InsertPedido crea pedido, cuenta nº 1 y reparto en la transacción en curso y devuelve el pedido
// armado. La moneda es la del negocio (PEN si no hay negocio activo).
func (r *PedidoSQLWriter) InsertPedido(ctx context.Context, n outbound.NuevoPedido) (dtoresult.PedidoDetalle, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.pedidos (sucursal_id, clave_idempotencia, mesa_id, cliente_id, mesero_id, tipo, comensales, notas, moneda)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8,
			COALESCE((SELECT moneda FROM sch_restaurante.negocio WHERE es_activo ORDER BY id LIMIT 1), 'PEN'))
		RETURNING id`,
		n.SucursalID, n.ClaveIdempotencia, n.MesaID, n.ClienteID, n.MeseroID, n.Tipo, n.Comensales, n.Notas).Scan(&id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, traducirPedido(err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO sch_restaurante.cuentas (pedido_id, numero, cliente_id) VALUES ($1, 1, $2)`, id, n.ClienteID); err != nil {
		return dtoresult.PedidoDetalle{}, traducirPedido(err)
	}
	if n.Reparto != nil {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO sch_restaurante.pedidos_delivery (pedido_id, destinatario, telefono, direccion, referencia, canal)
			VALUES ($1, $2, $3, $4, $5, COALESCE($6::text, 'PROPIO'))`,
			id, n.Reparto.Destinatario, n.Reparto.Telefono, n.Reparto.Direccion, n.Reparto.Referencia, n.Reparto.Canal); err != nil {
			return dtoresult.PedidoDetalle{}, traducirPedido(err)
		}
	}
	d, found, err := cargar.Pedido(ctx, r.db, id)
	if err != nil {
		return dtoresult.PedidoDetalle{}, err
	}
	if !found {
		return dtoresult.PedidoDetalle{}, sql.ErrNoRows
	}
	return d, nil
}

// GetPedidoForUpdate bloquea el pedido hasta el final de la transacción.
func (r *PedidoSQLWriter) GetPedidoForUpdate(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	var p outbound.PedidoRef
	err := r.db.QueryRowContext(ctx, `SELECT id, sucursal_id, estado, cerrado_en IS NOT NULL
		FROM sch_restaurante.pedidos WHERE id = $1 FOR UPDATE`, id).Scan(&p.ID, &p.SucursalID, &p.Estado, &p.Cerrado)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.PedidoRef{}, false, err
	}
	return p, true, nil
}

// GetCuentaForUpdate bloquea la cuenta; no la encuentra si es de otro pedido.
func (r *PedidoSQLWriter) GetCuentaForUpdate(ctx context.Context, pedidoID, cuentaID int64) (outbound.CuentaRef, bool, error) {
	var c outbound.CuentaRef
	err := r.db.QueryRowContext(ctx, `SELECT id, pedido_id, estado
		FROM sch_restaurante.cuentas WHERE id = $1 AND pedido_id = $2 FOR UPDATE`, cuentaID, pedidoID).Scan(&c.ID, &c.PedidoID, &c.Estado)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.CuentaRef{}, false, err
	}
	return c, true, nil
}

// GetProductoVenta lee el producto con el precio de la sede (si lo tiene propio) y la tasa vigente
// hoy (hora de Lima) de su impuesto. El producto se bloquea en modo compartido para que su baja
// se serialice con la venta. Disponible es false si no hay fila activa del producto en la sede.
func (r *PedidoSQLWriter) GetProductoVenta(ctx context.Context, productoID, sedeID int64) (outbound.ProductoVenta, bool, error) {
	var p outbound.ProductoVenta
	var tasa sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id, p.nombre, p.tipo, p.afectacion_igv, COALESCE(ps.precio_venta, p.precio_venta)::text,
		       t.tasa::text, p.es_activo, COALESCE(ps.disponible, false)
		FROM sch_restaurante.productos p
		LEFT JOIN sch_restaurante.productos_sucursal ps
		       ON ps.producto_id = p.id AND ps.sucursal_id = $2 AND ps.es_activo
		LEFT JOIN sch_restaurante.tasas_impuesto t
		       ON t.impuesto_id = p.impuesto_id
		      AND t.vigente_desde <= (now() AT TIME ZONE 'America/Lima')::date
		      AND (t.vigente_hasta IS NULL OR t.vigente_hasta > (now() AT TIME ZONE 'America/Lima')::date)
		WHERE p.id = $1
		FOR SHARE OF p`, productoID, sedeID).Scan(&p.ID, &p.Nombre, &p.Tipo, &p.AfectacionIGV, &p.Precio, &tasa, &p.EsActivo, &p.Disponible)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.ProductoVenta{}, false, err
	}
	if tasa.Valid {
		p.Tasa = tasa.String
	}
	return p, true, nil
}

// GetDetalleForUpdate bloquea la línea; no la encuentra si es de otro pedido. EnComanda indica que
// ya tiene un envío a cocina.
func (r *PedidoSQLWriter) GetDetalleForUpdate(ctx context.Context, pedidoID, detalleID int64) (outbound.DetalleRef, bool, error) {
	var d outbound.DetalleRef
	err := r.db.QueryRowContext(ctx, `
		SELECT id, pedido_id, cuenta_id, producto_id, cantidad::text, precio_unitario::text, descuento_total::text, estado,
		       EXISTS (SELECT 1 FROM sch_restaurante.comanda_detalles cd WHERE cd.detalle_pedido_id = detalles_pedido.id)
		FROM sch_restaurante.detalles_pedido WHERE id = $1 AND pedido_id = $2 FOR UPDATE`, detalleID, pedidoID).
		Scan(&d.ID, &d.PedidoID, &d.CuentaID, &d.ProductoID, &d.Cantidad, &d.PrecioUnitario, &d.DescuentoTotal, &d.Estado, &d.EnComanda)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.DetalleRef{}, false, err
	}
	return d, true, nil
}

func (r *PedidoSQLWriter) InsertDetalle(ctx context.Context, n outbound.NuevoDetalle) (dtoresult.Linea, error) {
	l, err := columnas.ScanLinea(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.detalles_pedido
			(pedido_id, cuenta_id, producto_id, detalle_padre_id, descripcion, cantidad, precio_unitario,
			 descuento_total, afectacion_igv, tasa_impuesto, notas_preparacion)
		VALUES ($1, $2, $3, $4, $5, $6::text::numeric, $7::text::numeric, $8::text::numeric, $9, $10::text::numeric, $11)
		RETURNING `+columnas.Linea,
		n.PedidoID, n.CuentaID, n.ProductoID, n.DetallePadreID, n.Descripcion, n.Cantidad, n.PrecioUnitario,
		n.DescuentoTotal, n.AfectacionIGV, n.TasaImpuesto, n.NotasPreparacion))
	return l, Translate(err)
}

func (r *PedidoSQLWriter) UpdateDetalle(ctx context.Context, id int64, c outbound.CambiosDetalle) (dtoresult.Linea, error) {
	l, err := columnas.ScanLinea(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.detalles_pedido SET
			cantidad = COALESCE($2::text::numeric, cantidad),
			descuento_total = COALESCE($3::text::numeric, descuento_total),
			notas_preparacion = CASE WHEN $4 THEN $5::text ELSE notas_preparacion END
		WHERE id = $1 RETURNING `+columnas.Linea, id, c.Cantidad, c.DescuentoTotal, c.Instrucciones.Presente, c.Instrucciones.Valor))
	return l, Translate(err)
}
```

- [ ] **Step 4: Adaptador de escritura**

Crear `service-restaurant/internal/modules/atencion/infrastructure/persistence/write/adapter/pedidos.go`:

```go
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// PedidoWriteAdapter implementa outbound.PedidoStore.
type PedidoWriteAdapter struct{ repo *repository.PedidoSQLWriter }

func NewPedidoWriteAdapter(db database.DBTX) *PedidoWriteAdapter {
	return &PedidoWriteAdapter{repo: repository.NewPedidoSQLWriter(db)}
}

var _ outbound.PedidoStore = (*PedidoWriteAdapter)(nil)

func (a *PedidoWriteAdapter) MesaParaPedido(ctx context.Context, mesaID int64) (outbound.MesaRef, bool, error) {
	return a.repo.GetMesa(ctx, mesaID)
}
func (a *PedidoWriteAdapter) EmpleadoDeUsuario(ctx context.Context, usuarioID int64) (*int64, error) {
	return a.repo.GetEmpleadoDeUsuario(ctx, usuarioID)
}
func (a *PedidoWriteAdapter) CrearPedido(ctx context.Context, n outbound.NuevoPedido) (dtoresult.PedidoDetalle, error) {
	return a.repo.InsertPedido(ctx, n)
}
func (a *PedidoWriteAdapter) BloquearPedido(ctx context.Context, id int64) (outbound.PedidoRef, bool, error) {
	return a.repo.GetPedidoForUpdate(ctx, id)
}
func (a *PedidoWriteAdapter) CuentaDelPedido(ctx context.Context, pedidoID, cuentaID int64) (outbound.CuentaRef, bool, error) {
	return a.repo.GetCuentaForUpdate(ctx, pedidoID, cuentaID)
}
func (a *PedidoWriteAdapter) ProductoParaVenta(ctx context.Context, productoID, sedeID int64) (outbound.ProductoVenta, bool, error) {
	return a.repo.GetProductoVenta(ctx, productoID, sedeID)
}
func (a *PedidoWriteAdapter) DetalleDelPedido(ctx context.Context, pedidoID, detalleID int64) (outbound.DetalleRef, bool, error) {
	return a.repo.GetDetalleForUpdate(ctx, pedidoID, detalleID)
}
func (a *PedidoWriteAdapter) InsertarDetalle(ctx context.Context, n outbound.NuevoDetalle) (dtoresult.Linea, error) {
	return a.repo.InsertDetalle(ctx, n)
}
func (a *PedidoWriteAdapter) ActualizarDetalle(ctx context.Context, id int64, c outbound.CambiosDetalle) (dtoresult.Linea, error) {
	return a.repo.UpdateDetalle(ctx, id, c)
}
```

- [ ] **Step 5: Lectura**

Crear `service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository/pedidos_sql_reader.go`:

```go
package repository

import (
	"context"
	"database/sql"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/cargar"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/internal/columnas"
	"service-restaurant/internal/platform/database"
)

// PedidoSQLReader lee pedidos.
type PedidoSQLReader struct{ db database.DBTX }

func NewPedidoSQLReader(db database.DBTX) *PedidoSQLReader { return &PedidoSQLReader{db} }

// ListarPedidos devuelve las cabeceras de las sedes de f.SedeIDs ordenadas por id (cursor estable).
// Estado y Tipo vacíos no filtran.
func (r *PedidoSQLReader) ListarPedidos(ctx context.Context, f outbound.FiltroPedidos) ([]dtoresult.Pedido, error) {
	var mesa sql.NullInt64
	if f.MesaID != nil {
		mesa = sql.NullInt64{Int64: *f.MesaID, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+columnas.Pedido+` FROM sch_restaurante.pedidos
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2
		  AND ($3::text = '' OR estado = $3) AND ($4::text = '' OR tipo = $4)
		  AND ($5::bigint IS NULL OR mesa_id = $5)
		ORDER BY id LIMIT $6`, f.SedeIDs, f.DespuesDeID, f.Estado, f.Tipo, mesa, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Pedido
	for rows.Next() {
		p, err := columnas.ScanPedido(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PedidoSQLReader) ObtenerPedido(ctx context.Context, id int64) (dtoresult.PedidoDetalle, bool, error) {
	return cargar.Pedido(ctx, r.db, id)
}
```

Crear `service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter/pedidos_read_adapter.go`:

```go
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// PedidoReadAdapter implementa outbound.PedidoReader.
type PedidoReadAdapter struct{ repo *repository.PedidoSQLReader }

func NewPedidoReadAdapter(db database.DBTX) *PedidoReadAdapter {
	return &PedidoReadAdapter{repo: repository.NewPedidoSQLReader(db)}
}

var _ outbound.PedidoReader = (*PedidoReadAdapter)(nil)

func (a *PedidoReadAdapter) ListarPedidos(ctx context.Context, f outbound.FiltroPedidos) ([]dtoresult.Pedido, error) {
	return a.repo.ListarPedidos(ctx, f)
}
func (a *PedidoReadAdapter) ObtenerPedido(ctx context.Context, id int64) (dtoresult.PedidoDetalle, bool, error) {
	return a.repo.ObtenerPedido(ctx, id)
}
```

- [ ] **Step 6: Verificar que compila y que la arquitectura se respeta**

Run: `cd service-restaurant && go build ./... && go vet ./internal/modules/atencion/... && go test ./tests/architecture/... -count=1`
Expected: sin errores de compilación, `vet` limpio y la prueba de arquitectura en PASS. Si la prueba de arquitectura rechaza `persistence/internal/cargar` (por ejemplo, si solo admite `columnas` bajo `internal`), leer su regla y mover `cargar` a `persistence/internal/columnas/` como `cargar.go` dentro del mismo paquete `columnas`, ajustando los imports; no relajar la regla.

- [ ] **Step 7: Commit**

```bash
git add service-restaurant/internal/modules/atencion/infrastructure/persistence
git commit -m "$(cat <<'EOF'
I3 rebanada 1: persistencia de pedidos, cuentas y consumos

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Capa web (cuerpos, mapper, handler y rutas)

**Files:**
- Create: `service-restaurant/internal/modules/atencion/web/dto/request/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/web/dto/response/pedidos.go`
- Create: `service-restaurant/internal/modules/atencion/web/mapper/pedidos_web_mapper.go`
- Modify: `service-restaurant/internal/modules/atencion/web/controller/helpers.go`
- Modify: `service-restaurant/internal/modules/atencion/web/controller/helpers_test.go`
- Create: `service-restaurant/internal/modules/atencion/web/controller/pedidos_handler.go`
- Modify: `service-restaurant/internal/modules/atencion/web/controller/routes.go`

**Interfaces:**
- Consumes: puertos inbound (Task 3); `request.Opcional[T]` y `idCadena`, `pathID`, `consulta`, `limiteDe`, `sucursalDe`, `actorDe`, `intencion`, `responder` existentes.
- Produces: `controller.NewPedidosHandler(l inbound.ListarPedidos, o inbound.ObtenerPedido, a inbound.AbrirPedido, ag inbound.AgregarDetalle, e inbound.EditarDetalle) *PedidosHandler`; `controller.RegisterRoutes(mux, zonas, mesas, pedidos, protect, logger)`; helpers `pathIDDe(r, nombre)` y `enumDe(q, campo, validos...)`.

- [ ] **Step 1: Prueba de los helpers nuevos**

Añadir al final de `service-restaurant/internal/modules/atencion/web/controller/helpers_test.go` (el archivo ya importa `net/url` y `testing`; comprobar los imports al pegar y añadir los que falten):

```go
func TestEnumDe(t *testing.T) {
	q := url.Values{"estado": {"REGISTRADO"}, "tipo": {"MOSTRADOR"}}
	if v, err := enumDe(q, "estado", "REGISTRADO", "LISTO"); err != nil || v != "REGISTRADO" {
		t.Fatalf("valor válido: %q %v", v, err)
	}
	if v, err := enumDe(url.Values{}, "estado", "REGISTRADO"); err != nil || v != "" {
		t.Fatalf("ausente no filtra: %q %v", v, err)
	}
	if _, err := enumDe(q, "tipo", "SALON", "LLEVAR"); err == nil {
		t.Fatal("un valor fuera del conjunto debe rechazarse")
	}
}

func TestPathIDDe(t *testing.T) {
	r := httptest.NewRequest("GET", "/x", nil)
	r.SetPathValue("detalle_id", "12")
	if id, err := pathIDDe(r, "detalle_id"); err != nil || id != 12 {
		t.Fatalf("pathIDDe = %d, %v", id, err)
	}
	r.SetPathValue("detalle_id", "0")
	if _, err := pathIDDe(r, "detalle_id"); err == nil {
		t.Fatal("el id debe ser positivo")
	}
}
```

Si `helpers_test.go` no importa `net/http/httptest`, añadir `"net/http/httptest"` a su bloque de imports.

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `cd service-restaurant && go test ./internal/modules/atencion/web/controller/ -run 'TestEnumDe|TestPathIDDe' -count=1`
Expected: FAIL de compilación (`undefined: enumDe`, `pathIDDe`).

- [ ] **Step 3: Helpers**

En `service-restaurant/internal/modules/atencion/web/controller/helpers.go`, reemplazar la función `pathID` por estas dos:

```go
// pathID lee el id numérico positivo de la ruta ({id}).
func pathID(r *http.Request) (int64, error) { return pathIDDe(r, "id") }

// pathIDDe lee un id numérico positivo de un segmento nombrado de la ruta.
func pathIDDe(r *http.Request, nombre string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(nombre), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.New(apperror.KindBadRequest, nombre+"_invalido", "el id de la ruta no es válido")
	}
	return id, nil
}
```

**Atención:** el código de error de `pathID` debe seguir siendo `id_invalido` (lo prueban los tests de mesas: `esperar(..., "id_invalido")`). Como `nombre+"_invalido"` con `nombre = "id"` da `id_invalido`, se conserva. Para `detalle_id` da `detalle_id_invalido`.

Y añadir al final del archivo:

```go
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
```

- [ ] **Step 4: Cuerpos de petición y respuesta**

Crear `service-restaurant/internal/modules/atencion/web/dto/request/pedidos.go`:

```go
package request

// Reparto: datos de entrega de un pedido DELIVERY (esquema DeliveryDatos del contrato).
type Reparto struct {
	Destinatario string  `json:"destinatario"`
	Telefono     string  `json:"telefono"`
	Direccion    string  `json:"direccion"`
	Referencia   *string `json:"referencia,omitempty"`
	Canal        *string `json:"canal,omitempty"`
}

// PedidoAlta: los ids viajan como cadenas (contrato). Un campo desconocido responde 400.
type PedidoAlta struct {
	SucursalID string   `json:"sucursal_id"`
	Tipo       string   `json:"tipo"`
	MesaID     *string  `json:"mesa_id,omitempty"`
	Comensales int      `json:"comensales"`
	ClienteID  *string  `json:"cliente_id,omitempty"`
	Notas      *string  `json:"notas,omitempty"`
	Reparto    *Reparto `json:"reparto,omitempty"`
}

// ConsumoAlta: cantidad y descuento_monto son decimales exactos en cadena.
type ConsumoAlta struct {
	ProductoID     string  `json:"producto_id"`
	CuentaID       string  `json:"cuenta_id"`
	Cantidad       string  `json:"cantidad"`
	DetallePadreID *string `json:"detalle_padre_id,omitempty"`
	Instrucciones  *string `json:"instrucciones,omitempty"`
	DescuentoMonto *string `json:"descuento_monto,omitempty"`
}

// ConsumoEdicion: instrucciones distingue ausente de null (vaciar).
type ConsumoEdicion struct {
	Cantidad       *string          `json:"cantidad,omitempty"`
	Instrucciones  Opcional[string] `json:"instrucciones"`
	DescuentoMonto *string          `json:"descuento_monto,omitempty"`
}
```

Crear `service-restaurant/internal/modules/atencion/web/dto/response/pedidos.go`:

```go
package response

import "time"

// Pedido es la cabecera (esquema Pedidos del contrato). version_bloqueo es de solo lectura.
type Pedido struct {
	ID                string     `json:"id"`
	SucursalID        string     `json:"sucursal_id"`
	CodigoPublico     string     `json:"codigo_publico"`
	ClaveIdempotencia string     `json:"clave_idempotencia"`
	MesaID            *string    `json:"mesa_id"`
	ReservaID         *string    `json:"reserva_id"`
	ClienteID         *string    `json:"cliente_id"`
	MeseroID          *string    `json:"mesero_id"`
	Tipo              string     `json:"tipo"`
	Estado            string     `json:"estado"`
	Moneda            string     `json:"moneda"`
	Comensales        int        `json:"comensales"`
	AbiertoEn         time.Time  `json:"abierto_en"`
	CerradoEn         *time.Time `json:"cerrado_en"`
	Notas             *string    `json:"notas"`
	MotivoCancelacion *string    `json:"motivo_cancelacion"`
	VersionBloqueo    int        `json:"version_bloqueo"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

// Linea es un consumo (esquema DetallesPedido del contrato).
type Linea struct {
	ID               string    `json:"id"`
	PedidoID         string    `json:"pedido_id"`
	CuentaID         string    `json:"cuenta_id"`
	ProductoID       string    `json:"producto_id"`
	DetallePadreID   *string   `json:"detalle_padre_id"`
	Descripcion      string    `json:"descripcion"`
	Cantidad         string    `json:"cantidad"`
	PrecioUnitario   string    `json:"precio_unitario"`
	DescuentoTotal   string    `json:"descuento_total"`
	AfectacionIGV    string    `json:"afectacion_igv"`
	TasaImpuesto     string    `json:"tasa_impuesto"`
	ImporteTotal     string    `json:"importe_total"`
	BaseImponible    string    `json:"base_imponible"`
	Estado           string    `json:"estado"`
	NotasPreparacion *string   `json:"notas_preparacion"`
	CreadoEn         time.Time `json:"creado_en"`
	CreadoPor        string    `json:"creado_por"`
	ActualizadoEn    time.Time `json:"actualizado_en"`
	ActualizadoPor   string    `json:"actualizado_por"`
}

type Cuenta struct {
	ID        string  `json:"id"`
	Numero    int     `json:"numero"`
	ClienteID *string `json:"cliente_id"`
	Etiqueta  *string `json:"etiqueta"`
	Estado    string  `json:"estado"`
	Subtotal  string  `json:"subtotal"`
	Impuesto  string  `json:"impuesto"`
	Total     string  `json:"total"`
	Lineas    []Linea `json:"lineas"`
}

type Reparto struct {
	Destinatario string  `json:"destinatario"`
	Telefono     string  `json:"telefono"`
	Direccion    string  `json:"direccion"`
	Referencia   *string `json:"referencia"`
	Canal        string  `json:"canal"`
	Estado       string  `json:"estado"`
}

// PedidoDetalle: la cabecera más el reparto (null salvo DELIVERY) y las cuentas con sus líneas.
type PedidoDetalle struct {
	Pedido
	Reparto *Reparto `json:"reparto"`
	Cuentas []Cuenta `json:"cuentas"`
}

type PedidosPagina struct {
	Items      []Pedido `json:"items"`
	NextCursor *string  `json:"nextCursor"`
}
```

- [ ] **Step 5: Mapper web**

Crear `service-restaurant/internal/modules/atencion/web/mapper/pedidos_web_mapper.go`:

```go
package mapper

import (
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/web/dto/response"
)

func idOpcional(n *int64) *string {
	if n == nil {
		return nil
	}
	v := id(*n)
	return &v
}

func ToPedido(p dtoresult.Pedido) response.Pedido {
	return response.Pedido{
		ID: id(p.ID), SucursalID: id(p.SucursalID), CodigoPublico: p.CodigoPublico, ClaveIdempotencia: p.ClaveIdempotencia,
		MesaID: idOpcional(p.MesaID), ReservaID: idOpcional(p.ReservaID), ClienteID: idOpcional(p.ClienteID), MeseroID: idOpcional(p.MeseroID),
		Tipo: p.Tipo, Estado: p.Estado, Moneda: p.Moneda, Comensales: p.Comensales, AbiertoEn: p.AbiertoEn, CerradoEn: p.CerradoEn,
		Notas: p.Notas, MotivoCancelacion: p.MotivoCancelacion, VersionBloqueo: p.VersionBloqueo,
		CreadoEn: p.CreadoEn, CreadoPor: id(p.CreadoPor), ActualizadoEn: p.ActualizadoEn, ActualizadoPor: id(p.ActualizadoPor),
	}
}

func ToLinea(l dtoresult.Linea) response.Linea {
	return response.Linea{
		ID: id(l.ID), PedidoID: id(l.PedidoID), CuentaID: id(l.CuentaID), ProductoID: id(l.ProductoID),
		DetallePadreID: idOpcional(l.DetallePadreID), Descripcion: l.Descripcion, Cantidad: l.Cantidad,
		PrecioUnitario: l.PrecioUnitario, DescuentoTotal: l.DescuentoTotal, AfectacionIGV: l.AfectacionIGV,
		TasaImpuesto: l.TasaImpuesto, ImporteTotal: l.ImporteTotal, BaseImponible: l.BaseImponible, Estado: l.Estado,
		NotasPreparacion: l.NotasPreparacion, CreadoEn: l.CreadoEn, CreadoPor: id(l.CreadoPor),
		ActualizadoEn: l.ActualizadoEn, ActualizadoPor: id(l.ActualizadoPor),
	}
}

func ToPedidoDetalle(d dtoresult.PedidoDetalle) response.PedidoDetalle {
	out := response.PedidoDetalle{Pedido: ToPedido(d.Pedido), Cuentas: make([]response.Cuenta, len(d.Cuentas))}
	if d.Reparto != nil {
		out.Reparto = &response.Reparto{
			Destinatario: d.Reparto.Destinatario, Telefono: d.Reparto.Telefono, Direccion: d.Reparto.Direccion,
			Referencia: d.Reparto.Referencia, Canal: d.Reparto.Canal, Estado: d.Reparto.Estado,
		}
	}
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

func ToPedidosPagina(p dtoresult.PaginaPedidos) response.PedidosPagina {
	items := make([]response.Pedido, len(p.Items))
	for i, it := range p.Items {
		items[i] = ToPedido(it)
	}
	return response.PedidosPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
```

- [ ] **Step 6: Handler**

Crear `service-restaurant/internal/modules/atencion/web/controller/pedidos_handler.go`:

```go
package controller

import (
	"fmt"
	"net/http"

	"service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/dto/query"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	"service-restaurant/internal/modules/atencion/web/dto/request"
	webmapper "service-restaurant/internal/modules/atencion/web/mapper"
	webrequest "service-restaurant/internal/platform/web/request"
)

var (
	estadosPedido = []string{"REGISTRADO", "PREPARACION", "LISTO", "ENTREGADO", "CANCELADO"}
	tiposPedido   = []string{"SALON", "LLEVAR", "DELIVERY"}
)

// PedidosHandler: pedidos, cuentas y consumos del salón (RES-RF-009, RES-RF-010).
type PedidosHandler struct {
	listar  inbound.ListarPedidos
	obtener inbound.ObtenerPedido
	abrir   inbound.AbrirPedido
	agregar inbound.AgregarDetalle
	editar  inbound.EditarDetalle
}

func NewPedidosHandler(l inbound.ListarPedidos, o inbound.ObtenerPedido, a inbound.AbrirPedido,
	ag inbound.AgregarDetalle, e inbound.EditarDetalle) *PedidosHandler {
	return &PedidosHandler{l, o, a, ag, e}
}

// opcionalID convierte un id en cadena anulable (contrato) a *int64.
func opcionalID(valor *string, campo string) (*int64, error) {
	if valor == nil {
		return nil, nil
	}
	n, err := idCadena(*valor, campo)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// Listar: GET /pedidos?sucursal_id=&estado=&tipo=&mesa_id=&limit=&cursor=
func (h *PedidosHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	estado, err := enumDe(q, "estado", estadosPedido...)
	if err != nil {
		return err
	}
	tipo, err := enumDe(q, "tipo", tiposPedido...)
	if err != nil {
		return err
	}
	var mesa *int64
	if v := q.Get("mesa_id"); v != "" {
		if mesa, err = opcionalID(&v, "mesa_id"); err != nil {
			return err
		}
	}
	res, err := h.listar.Execute(r.Context(), query.ListarPedidos{
		Actor: actor, SucursalID: sede, Estado: estado, Tipo: tipo, MesaID: mesa, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToPedidosPagina)
}

// Obtener: GET /pedidos/{id}
func (h *PedidosHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerPedido{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToPedidoDetalle)
}

// Abrir: POST /pedidos (Idempotency-Key obligatoria)
func (h *PedidosHandler) Abrir(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.PedidoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	sede, err := idCadena(body.SucursalID, "sucursal_id")
	if err != nil {
		return err
	}
	mesa, err := opcionalID(body.MesaID, "mesa_id")
	if err != nil {
		return err
	}
	cliente, err := opcionalID(body.ClienteID, "cliente_id")
	if err != nil {
		return err
	}
	var reparto *command.Reparto
	if body.Reparto != nil {
		reparto = &command.Reparto{Destinatario: body.Reparto.Destinatario, Telefono: body.Reparto.Telefono,
			Direccion: body.Reparto.Direccion, Referencia: body.Reparto.Referencia, Canal: body.Reparto.Canal}
	}
	in, err := intencion(r, "POST /pedidos", body)
	if err != nil {
		return err
	}
	res, err := h.abrir.Execute(r.Context(), command.AbrirPedido{
		Intencion: in, Actor: actor, SucursalID: sede, Tipo: body.Tipo, MesaID: mesa, Comensales: body.Comensales,
		ClienteID: cliente, Notas: body.Notas, Reparto: reparto,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToPedidoDetalle)
}

// AgregarDetalle: POST /pedidos/{id}/detalles (Idempotency-Key obligatoria)
func (h *PedidosHandler) AgregarDetalle(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	pedido, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ConsumoAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	producto, err := idCadena(body.ProductoID, "producto_id")
	if err != nil {
		return err
	}
	cuenta, err := idCadena(body.CuentaID, "cuenta_id")
	if err != nil {
		return err
	}
	padre, err := opcionalID(body.DetallePadreID, "detalle_padre_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, fmt.Sprintf("POST /pedidos/%d/detalles", pedido), body)
	if err != nil {
		return err
	}
	res, err := h.agregar.Execute(r.Context(), command.AgregarDetalle{
		Intencion: in, Actor: actor, PedidoID: pedido, CuentaID: cuenta, ProductoID: producto, Cantidad: body.Cantidad,
		DetallePadreID: padre, Instrucciones: body.Instrucciones, DescuentoMonto: body.DescuentoMonto,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToLinea)
}

// EditarDetalle: PATCH /pedidos/{id}/detalles/{detalle_id}
func (h *PedidosHandler) EditarDetalle(w http.ResponseWriter, r *http.Request) error {
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
	var body request.ConsumoEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarDetalle{
		Actor: actor, PedidoID: pedido, DetalleID: detalle, Cantidad: body.Cantidad, DescuentoMonto: body.DescuentoMonto,
		Instrucciones: command.Campo[string]{Presente: body.Instrucciones.Presente, Valor: body.Instrucciones.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToLinea)
}
```

- [ ] **Step 7: Rutas**

Reemplazar el contenido de `service-restaurant/internal/modules/atencion/web/controller/routes.go` por:

```go
package controller

import (
	"log/slog"
	"net/http"

	"service-restaurant/internal/platform/web/handler"
)

// RegisterRoutes monta zonas, mesas y pedidos (todas exigen autenticación; el permiso fino lo
// verifica cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, zonas *ZonasHandler, mesas *MesasHandler, pedidos *PedidosHandler,
	protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /zonas", zonas.Listar)
	ruta("POST /zonas", zonas.Crear)
	ruta("GET /zonas/{id}", zonas.Obtener)
	ruta("PATCH /zonas/{id}", zonas.Editar)
	ruta("POST /zonas/{id}/baja", zonas.Baja)
	ruta("POST /zonas/{id}/restauracion", zonas.Restauracion)

	ruta("GET /mesas", mesas.Listar)
	ruta("GET /mesas/{id}", mesas.Obtener)
	ruta("POST /mesas", mesas.Crear)
	ruta("PATCH /mesas/{id}", mesas.Editar)
	ruta("POST /mesas/{id}/baja", mesas.Baja)
	ruta("POST /mesas/{id}/restauracion", mesas.Restauracion)

	ruta("GET /pedidos", pedidos.Listar)
	ruta("POST /pedidos", pedidos.Abrir)
	ruta("GET /pedidos/{id}", pedidos.Obtener)
	ruta("POST /pedidos/{id}/detalles", pedidos.AgregarDetalle)
	ruta("PATCH /pedidos/{id}/detalles/{detalle_id}", pedidos.EditarDetalle)
}
```

- [ ] **Step 8: Ejecutar las pruebas del controller**

Run: `cd service-restaurant && go test ./internal/modules/atencion/web/... -count=1 2>&1 | tail -15`
Expected: PASS (`TestEnumDe`, `TestPathIDDe` y los tests existentes). El módulo completo todavía no compila porque `module.go` llama a `RegisterRoutes` con la firma anterior; se arregla en la Task 9.

- [ ] **Step 9: Commit**

```bash
git add service-restaurant/internal/modules/atencion/web
git commit -m "$(cat <<'EOF'
I3 rebanada 1: cuerpos, handler y rutas de pedidos

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Composición del módulo

**Files:**
- Create: `service-restaurant/internal/modules/atencion/infrastructure/config/pedidos.go`
- Modify: `service-restaurant/internal/modules/atencion/infrastructure/config/module.go`

**Interfaces:**
- Consumes: `escritura[C, T]`, `ejecutor`, `txQuerier` (en `escritura.go`), `writeadapter.NewPedidoWriteAdapter`, `readadapter.NewPedidoReadAdapter`, casos de uso y `controller.NewPedidosHandler`.
- Produces: el módulo `atencion` sirve `/pedidos` con transacción e idempotencia.

- [ ] **Step 1: Composición de pedidos**

Crear `service-restaurant/internal/modules/atencion/infrastructure/config/pedidos.go`:

```go
package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	"service-restaurant/internal/modules/atencion/application/usecase/command"
	uquery "service-restaurant/internal/modules/atencion/application/usecase/query"
	readadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter"
	writeadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/write/adapter"
	"service-restaurant/internal/modules/atencion/web/controller"
)

// armarPedidos compone los casos de uso de pedidos y consumos, ya decorados con transacción e
// idempotencia. La edición de una línea no lleva clave: es una operación sobre un recurso ya
// identificado y repetirla produce el mismo estado.
func armarPedidos(db *sql.DB, logger *slog.Logger) *controller.PedidosHandler {
	abrir := escritura[cmd.AbrirPedido, dtoresult.PedidoDetalle]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.AbrirPedido, dtoresult.PedidoDetalle] {
			return command.NewAbrirPedido(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.AbrirPedido) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	agregar := escritura[cmd.AgregarDetalle, dtoresult.Linea]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.AgregarDetalle, dtoresult.Linea] {
			return command.NewAgregarDetalle(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.AgregarDetalle) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarDetalle, dtoresult.Linea]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarDetalle, dtoresult.Linea] {
			return command.NewEditarDetalle(writeadapter.NewPedidoWriteAdapter(q))
		},
		meta: func(c cmd.EditarDetalle) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}

	lector := readadapter.NewPedidoReadAdapter(db)
	return controller.NewPedidosHandler(uquery.NewListarPedidos(lector), uquery.NewObtenerPedido(lector),
		inbound.AbrirPedido(abrir), inbound.AgregarDetalle(agregar), inbound.EditarDetalle(editar))
}
```

- [ ] **Step 2: Conectar el módulo**

En `service-restaurant/internal/modules/atencion/infrastructure/config/module.go`, reemplazar el struct `Module`, `New` y `RegisterRoutes` por:

```go
// Module es el módulo atencion ya ensamblado.
type Module struct {
	deps    Deps
	zonas   *controller.ZonasHandler
	mesas   *controller.MesasHandler
	pedidos *controller.PedidosHandler
}

// New ensambla el módulo.
func New(d Deps) (*Module, error) {
	return &Module{deps: d, zonas: armarZonas(d.DB, d.Logger), mesas: armarMesas(d.DB, d.Logger),
		pedidos: armarPedidos(d.DB, d.Logger)}, nil
}

// RegisterRoutes monta las rutas del módulo sobre mux (relativas a /api/v1). protect exige un
// actor autenticado; lo aporta el módulo identidad a través del bootstrap.
func (m *Module) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	controller.RegisterRoutes(mux, m.zonas, m.mesas, m.pedidos, protect, m.deps.Logger)
}
```

- [ ] **Step 3: Compilar, vet, arquitectura y pruebas unitarias**

Run: `cd service-restaurant && go build ./... && go vet ./... && go test ./internal/... ./tests/architecture/... -count=1 2>&1 | tail -25`
Expected: compila, `vet` limpio, todo en PASS (las pruebas de `tests/integration` no se ejecutan aquí).

- [ ] **Step 4: Humo de arranque de rutas**

Run: `cd service-restaurant && go test ./tests/integration/ -run 'TestIdentidad|TestMesasExigeAlgunPermisoDeMesa' -count=1 2>&1 | tail -8`
Expected: PASS. Confirma que la aplicación sigue arrancando con las rutas nuevas (un patrón de ruta duplicado o inválido haría `panic` al construir el `mux`).

- [ ] **Step 5: Commit**

```bash
git add service-restaurant/internal/modules/atencion/infrastructure/config
git commit -m "$(cat <<'EOF'
I3 rebanada 1: el módulo atencion sirve pedidos y consumos

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Pruebas de integración HTTP

**Files:**
- Create: `service-restaurant/tests/integration/pedidos_helpers_test.go`
- Create: `service-restaurant/tests/integration/pedidos_apertura_test.go`
- Create: `service-restaurant/tests/integration/pedidos_consumos_test.go`
- Create: `service-restaurant/tests/integration/pedidos_sedes_test.go`

**Interfaces:**
- Consumes: helpers existentes `app`, `call`, `post`, `callH`, `nuevaClave`, `esperar`, `asSystem`, `scalar`, `scalarStr`, `idDe`, `sufijo`, `crearSedeSQL`, `crearZonaSQL`, `crearMesaSQL`, `rolConPermisos`, `asegurarUsuario`, `token`, `conLimite` y el tipo `resp` (con `.Code`, `.Body`, `.json(t, &v)`).
- Produces: la evidencia de RES-RF-009, RES-RF-010 y RES-RF-002 para pedidos.

- [ ] **Step 1: Helpers de datos y de respuestas**

Crear `service-restaurant/tests/integration/pedidos_helpers_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"testing"
)

type lineaResp struct {
	ID             string  `json:"id"`
	PedidoID       string  `json:"pedido_id"`
	CuentaID       string  `json:"cuenta_id"`
	ProductoID     string  `json:"producto_id"`
	DetallePadreID *string `json:"detalle_padre_id"`
	Descripcion    string  `json:"descripcion"`
	Cantidad       string  `json:"cantidad"`
	PrecioUnitario string  `json:"precio_unitario"`
	DescuentoTotal string  `json:"descuento_total"`
	AfectacionIGV  string  `json:"afectacion_igv"`
	TasaImpuesto   string  `json:"tasa_impuesto"`
	ImporteTotal   string  `json:"importe_total"`
	BaseImponible  string  `json:"base_imponible"`
	Estado         string  `json:"estado"`
	Notas          *string `json:"notas_preparacion"`
}

type cuentaResp struct {
	ID       string      `json:"id"`
	Numero   int         `json:"numero"`
	Estado   string      `json:"estado"`
	Subtotal string      `json:"subtotal"`
	Impuesto string      `json:"impuesto"`
	Total    string      `json:"total"`
	Lineas   []lineaResp `json:"lineas"`
}

type repartoResp struct {
	Destinatario string  `json:"destinatario"`
	Telefono     string  `json:"telefono"`
	Direccion    string  `json:"direccion"`
	Referencia   *string `json:"referencia"`
	Canal        string  `json:"canal"`
	Estado       string  `json:"estado"`
}

type pedidoResp struct {
	ID                string       `json:"id"`
	SucursalID        string       `json:"sucursal_id"`
	ClaveIdempotencia string       `json:"clave_idempotencia"`
	MesaID            *string      `json:"mesa_id"`
	MeseroID          *string      `json:"mesero_id"`
	Tipo              string       `json:"tipo"`
	Estado            string       `json:"estado"`
	Moneda            string       `json:"moneda"`
	Comensales        int          `json:"comensales"`
	Notas             *string      `json:"notas"`
	Reparto           *repartoResp `json:"reparto"`
	Cuentas           []cuentaResp `json:"cuentas"`
}

type pedidosPagina struct {
	Items      []pedidoResp `json:"items"`
	NextCursor *string      `json:"nextCursor"`
}

// impuestoPedidosSQL garantiza un impuesto propio de estas pruebas con tasa del 18 % vigente desde
// el año 2000: no choca con las tasas que crean las pruebas de tasas_impuesto.
func impuestoPedidosSQL(t *testing.T) string {
	t.Helper()
	app(t)
	asSystem(t,
		`INSERT INTO sch_restaurante.impuestos(codigo,nombre) SELECT 'IGV_PED','IGV de las pruebas de pedidos'
		 WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.impuestos WHERE codigo='IGV_PED')`,
		`INSERT INTO sch_restaurante.tasas_impuesto(impuesto_id,tasa,vigente_desde)
		 SELECT i.id,18,DATE '2000-01-01' FROM sch_restaurante.impuestos i
		 WHERE i.codigo='IGV_PED' AND NOT EXISTS (SELECT 1 FROM sch_restaurante.tasas_impuesto x WHERE x.impuesto_id=i.id)`)
	return scalarStr(t, `SELECT id FROM sch_restaurante.impuestos WHERE codigo='IGV_PED'`)
}

func categoriaPedidosSQL(t *testing.T) string {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.categorias(nombre) SELECT 'Pedidos IT'
		WHERE NOT EXISTS (SELECT 1 FROM sch_restaurante.categorias WHERE nombre='Pedidos IT')`)
	return scalarStr(t, `SELECT id FROM sch_restaurante.categorias WHERE nombre='Pedidos IT'`)
}

// productoVentaSQL crea un producto gravado con IGV_PED (18 %) del tipo y precio dados y, si sede
// no está vacía, lo deja en esa sede con la disponibilidad indicada. Devuelve su id.
func productoVentaSQL(t *testing.T, sede, tipo, precio string, disponible bool) string {
	t.Helper()
	cat, imp := categoriaPedidosSQL(t), impuestoPedidosSQL(t)
	codigo := "PED-" + sufijo()
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.productos(categoria_id,codigo,nombre,tipo,precio_venta,afectacion_igv,impuesto_id)
		VALUES (%s,'%s','Producto %s','%s',%s,'10',%s)`, cat, codigo, codigo, tipo, precio, imp))
	if sede != "" {
		asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.productos_sucursal(producto_id,sucursal_id,disponible)
			SELECT p.id,s.id,%t FROM sch_restaurante.productos p, sch_restaurante.sucursales s
			WHERE p.codigo='%s' AND s.codigo='%s'`, disponible, codigo, sede))
	}
	return scalarStr(t, `SELECT id FROM sch_restaurante.productos WHERE codigo=$1`, codigo)
}

func tokenPedidos(t *testing.T, usuario, rol string, permisos []string, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, rol, permisos...)
	asegurarUsuario(t, usuario, "Pedidos2026Clave", rol, append([]string{"PRINCIPAL"}, sedes...)...)
	return token(t, usuario, "Pedidos2026Clave")
}

// cajeroPedidos: ver, crear y editar pedidos y autorizar descuentos en PRINCIPAL y las sedes dadas.
func cajeroPedidos(t *testing.T, sedes ...string) string {
	return tokenPedidos(t, "ped.cajero", "T_PED_CAJERO",
		[]string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR", "DESCUENTO_AUTORIZAR"}, sedes...)
}

// meseroPedidos: ver, crear y editar pedidos, sin autorizar descuentos.
func meseroPedidos(t *testing.T, sedes ...string) string {
	return tokenPedidos(t, "ped.mesero", "T_PED_MESERO", []string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR"}, sedes...)
}

// lectorPedidos: solo ver pedidos.
func lectorPedidos(t *testing.T, sedes ...string) string {
	return tokenPedidos(t, "ped.lector", "T_PED_LECTOR", []string{"PEDIDO_VER"}, sedes...)
}

// mesaPedidosSQL crea una mesa nueva de la capacidad dada en la sede y devuelve su id.
func mesaPedidosSQL(t *testing.T, sede string, capacidad int) string {
	t.Helper()
	zona := "Zona pedidos " + sede
	crearZonaSQL(t, sede, zona)
	return crearMesaSQL(t, sede, zona, "PD-"+sufijo(), capacidad)
}

func sedeID(t *testing.T, codigo string) string { return idDe(t, "sucursales", "codigo", codigo) }

// abrirLlevar abre un pedido LLEVAR en la sede y devuelve la respuesta ya decodificada.
func abrirLlevar(t *testing.T, tk, sede string) pedidoResp {
	t.Helper()
	r := post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, sede)), tk)
	esperar(t, r, http.StatusCreated, "")
	var p pedidoResp
	r.json(t, &p)
	return p
}

func obtenerPedido(t *testing.T, tk, id string) pedidoResp {
	t.Helper()
	r := call(t, "GET", "/api/v1/pedidos/"+id, "", tk)
	esperar(t, r, http.StatusOK, "")
	var p pedidoResp
	r.json(t, &p)
	return p
}

// agregarConsumo agrega una línea y devuelve la respuesta cruda.
func agregarConsumo(t *testing.T, tk, pedido, cuenta, producto, cantidad string, extras string) resp {
	t.Helper()
	cuerpo := fmt.Sprintf(`{"producto_id":%q,"cuenta_id":%q,"cantidad":%q%s}`, producto, cuenta, cantidad, extras)
	return post(t, "/api/v1/pedidos/"+pedido+"/detalles", cuerpo, tk)
}

// pedidoSQL inserta, como sistema, un pedido LLEVAR con su cuenta nº 1 en la sede; devuelve ids.
func pedidoSQL(t *testing.T, sede string) (pedidoID, cuentaID string) {
	t.Helper()
	app(t)
	asSystem(t,
		fmt.Sprintf(`INSERT INTO sch_restaurante.pedidos(sucursal_id,clave_idempotencia,tipo,comensales)
			SELECT s.id,gen_random_uuid(),'LLEVAR',1 FROM sch_restaurante.sucursales s WHERE s.codigo='%s'`, sede))
	pedidoID = scalarStr(t, `SELECT p.id FROM sch_restaurante.pedidos p JOIN sch_restaurante.sucursales s ON s.id=p.sucursal_id
		WHERE s.codigo=$1 ORDER BY p.id DESC LIMIT 1`, sede)
	asSystem(t, fmt.Sprintf(`INSERT INTO sch_restaurante.cuentas(pedido_id,numero) VALUES (%s,1)`, pedidoID))
	cuentaID = scalarStr(t, `SELECT id FROM sch_restaurante.cuentas WHERE pedido_id=$1::text::bigint`, pedidoID)
	return pedidoID, cuentaID
}
```

- [ ] **Step 2: Pruebas de apertura (RF-009)**

Crear `service-restaurant/tests/integration/pedidos_apertura_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"testing"
)

func TestPedidosAbrirSalonCreaPedidoYCuenta(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	tk := cajeroPedidos(t)
	sede := sedeID(t, "PRINCIPAL")

	r := post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2,"notas":" sin cebolla "}`, sede, mesa), tk)
	esperar(t, r, http.StatusCreated, "")
	var p pedidoResp
	r.json(t, &p)
	if p.Tipo != "SALON" || p.Estado != "REGISTRADO" || p.Moneda != "PEN" || p.Comensales != 2 || p.SucursalID != sede ||
		p.MesaID == nil || *p.MesaID != mesa || p.Notas == nil || *p.Notas != "sin cebolla" || p.Reparto != nil {
		t.Fatalf("pedido: %+v", p)
	}
	if len(p.Cuentas) != 1 || p.Cuentas[0].Numero != 1 || p.Cuentas[0].Estado != "ABIERTA" ||
		p.Cuentas[0].Total != "0.00" || len(p.Cuentas[0].Lineas) != 0 {
		t.Fatalf("debe traer la cuenta nº 1 abierta y vacía: %+v", p.Cuentas)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.cuentas WHERE pedido_id=$1::text::bigint`, p.ID); n != 1 {
		t.Fatalf("cuentas del pedido = %d, quería 1", n)
	}
	// El pedido abierto ocupa la mesa.
	if got := scalarStr(t, `SELECT estado_actual FROM sch_restaurante.v_mesas_estado WHERE mesa_id=$1::text::bigint`, mesa); got != "OCUPADA" {
		t.Fatalf("la mesa debe quedar OCUPADA, está %s", got)
	}
	// GET devuelve lo mismo.
	if g := obtenerPedido(t, tk, p.ID); g.ID != p.ID || len(g.Cuentas) != 1 {
		t.Fatalf("GET: %+v", g)
	}
}

func TestPedidosAbrirLlevarYDelivery(t *testing.T) {
	tk := cajeroPedidos(t)
	sede := sedeID(t, "PRINCIPAL")

	p := abrirLlevar(t, tk, "PRINCIPAL")
	if p.Tipo != "LLEVAR" || p.MesaID != nil || p.Reparto != nil {
		t.Fatalf("LLEVAR: %+v", p)
	}

	r := post(t, "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"DELIVERY","comensales":1,
		"reparto":{"destinatario":" Ana Pérez ","telefono":"999111222","direccion":"Av. Lima 123","referencia":"Frente al parque"}}`, sede), tk)
	esperar(t, r, http.StatusCreated, "")
	var d pedidoResp
	r.json(t, &d)
	if d.Tipo != "DELIVERY" || d.Reparto == nil || d.Reparto.Destinatario != "Ana Pérez" || d.Reparto.Canal != "PROPIO" ||
		d.Reparto.Estado != "PENDIENTE" || d.Reparto.Referencia == nil || *d.Reparto.Referencia != "Frente al parque" {
		t.Fatalf("DELIVERY: %+v", d)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos_delivery WHERE pedido_id=$1::text::bigint`, d.ID); n != 1 {
		t.Fatalf("pedidos_delivery = %d, quería 1", n)
	}
}

func TestPedidosAbrirRechazos(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 2)
	tk := cajeroPedidos(t)
	sede := sedeID(t, "PRINCIPAL")
	casos := []struct {
		nombre, cuerpo string
		status         int
		code           string
	}{
		{"capacidad excedida", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":3}`, sede, mesa), 422, "capacidad_excedida"},
		{"salón sin mesa", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","comensales":1}`, sede), 422, "mesa_requerida"},
		{"llevar con mesa", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","mesa_id":%q,"comensales":1}`, sede, mesa), 422, "mesa_no_aplica"},
		{"mesa inexistente", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":"999999","comensales":1}`, sede), 422, "mesa_invalida"},
		{"delivery sin reparto", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"DELIVERY","comensales":1}`, sede), 422, "reparto_requerido"},
		{"tipo desconocido", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"MOSTRADOR","comensales":1}`, sede), 422, "tipo_invalido"},
		{"comensales cero", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":0}`, sede), 422, "comensales_invalidos"},
		{"campo desconocido", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1,"total":"10"}`, sede), 400, ""},
		{"id mal formado", `{"sucursal_id":"abc","tipo":"LLEVAR","comensales":1}`, 400, "sucursal_id_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) { esperar(t, post(t, "/api/v1/pedidos", c.cuerpo, tk), c.status, c.code) })
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE mesa_id=$1::text::bigint`, mesa); n != 0 {
		t.Fatalf("los rechazos no deben dejar pedidos (%d)", n)
	}
	// Sin Idempotency-Key.
	esperar(t, call(t, "POST", "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sede), tk),
		http.StatusBadRequest, "idempotency_key_requerida")
}

func TestPedidosMesaOcupada(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	tk := cajeroPedidos(t)
	cuerpo := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2}`, sedeID(t, "PRINCIPAL"), mesa)
	esperar(t, post(t, "/api/v1/pedidos", cuerpo, tk), http.StatusCreated, "")
	esperar(t, post(t, "/api/v1/pedidos", cuerpo, tk), http.StatusConflict, "mesa_ocupada")
}

// Dos aperturas simultáneas de la misma mesa: solo una gana.
func TestPedidosDosAperturasSimultaneasDeUnaMesa(t *testing.T) {
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	tk := cajeroPedidos(t)
	cuerpo := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":2}`, sedeID(t, "PRINCIPAL"), mesa)

	codigos := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codigos {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codigos[i] = post(t, "/api/v1/pedidos", cuerpo, tk).Code
		}(i)
	}
	wg.Wait()
	sort.Ints(codigos)
	if codigos[0] != http.StatusCreated || codigos[1] != http.StatusConflict {
		t.Fatalf("estados %v, quería [201 409]", codigos)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE mesa_id=$1::text::bigint AND cerrado_en IS NULL`, mesa); n != 1 {
		t.Fatalf("pedidos abiertos en la mesa = %d, quería 1", n)
	}
}

// RES-CP-009: si falla la creación de la cuenta no queda una cabecera de pedido huérfana.
func TestPedidosAperturaSinCabeceraHuerfana(t *testing.T) {
	app(t)
	asSystem(t,
		`CREATE OR REPLACE FUNCTION sch_restaurante.fn_test_falla_cuenta() RETURNS trigger LANGUAGE plpgsql AS $$
		 BEGIN
		   IF EXISTS (SELECT 1 FROM sch_restaurante.pedidos WHERE id = NEW.pedido_id AND notas = 'FALLA-CUENTA') THEN
		     RAISE EXCEPTION 'falla simulada de la cuenta';
		   END IF;
		   RETURN NEW;
		 END $$`,
		`CREATE TRIGGER zz_test_falla_cuenta BEFORE INSERT ON sch_restaurante.cuentas
		 FOR EACH ROW EXECUTE FUNCTION sch_restaurante.fn_test_falla_cuenta()`)
	t.Cleanup(func() {
		asSystem(t, `DROP TRIGGER IF EXISTS zz_test_falla_cuenta ON sch_restaurante.cuentas`,
			`DROP FUNCTION IF EXISTS sch_restaurante.fn_test_falla_cuenta()`)
	})

	tk := cajeroPedidos(t)
	r := post(t, "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1,"notas":"FALLA-CUENTA"}`, sedeID(t, "PRINCIPAL")), tk)
	esperar(t, r, http.StatusUnprocessableEntity, "regla_de_datos")
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE notas='FALLA-CUENTA'`); n != 0 {
		t.Fatalf("quedaron %d cabeceras huérfanas", n)
	}
}

func TestPedidosAperturaIdempotente(t *testing.T) {
	tk := cajeroPedidos(t)
	cuerpo := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, "PRINCIPAL"))
	clave := nuevaClave()
	h := map[string]string{"Idempotency-Key": clave}

	r1 := callH(t, http.MethodPost, "/api/v1/pedidos", cuerpo, tk, h)
	esperar(t, r1, http.StatusCreated, "")
	r2 := callH(t, http.MethodPost, "/api/v1/pedidos", cuerpo, tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var p1, p2 pedidoResp
	r1.json(t, &p1)
	r2.json(t, &p2)
	if p1.ID != p2.ID || p1.ClaveIdempotencia != clave {
		t.Fatalf("el reintento debe devolver el mismo pedido: %s / %s (clave %s)", p1.ID, p2.ID, p1.ClaveIdempotencia)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.pedidos WHERE clave_idempotencia=$1::uuid`, clave); n != 1 {
		t.Fatalf("pedidos con esa clave = %d, quería 1", n)
	}
	// La misma clave con otro contenido es un conflicto.
	otro := fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":3}`, sedeID(t, "PRINCIPAL"))
	esperar(t, callH(t, http.MethodPost, "/api/v1/pedidos", otro, tk, h), http.StatusConflict, "idempotency_key_reutilizada")
}
```

- [ ] **Step 3: Pruebas de consumos (RF-010)**

Crear `service-restaurant/tests/integration/pedidos_consumos_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// RES-CP-010: cambiar el precio de la carta no modifica un detalle existente.
func TestConsumoConservaElPrecioHistorico(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	r := agregarConsumo(t, tk, p.ID, cuenta, prod, "2", `,"instrucciones":" sin ají "`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	if l.Cantidad != "2.000" || l.PrecioUnitario != "10.000000" || l.DescuentoTotal != "0.00" || l.AfectacionIGV != "10" ||
		l.TasaImpuesto != "18.0000" || l.ImporteTotal != "20.00" || l.BaseImponible != "16.95" || l.Estado != "ACTIVO" ||
		l.Notas == nil || *l.Notas != "sin ají" || l.CuentaID != cuenta || l.PedidoID != p.ID {
		t.Fatalf("línea: %+v", l)
	}

	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos SET precio_venta = 15 WHERE id = %s`, prod))
	r2 := agregarConsumo(t, tk, p.ID, cuenta, prod, "2", "")
	esperar(t, r2, http.StatusCreated, "")
	var l2 lineaResp
	r2.json(t, &l2)
	if l2.PrecioUnitario != "15.000000" {
		t.Fatalf("la línea nueva usa el precio vigente: %+v", l2)
	}

	g := obtenerPedido(t, tk, p.ID)
	c := g.Cuentas[0]
	if len(c.Lineas) != 2 || c.Lineas[0].PrecioUnitario != "10.000000" || c.Lineas[0].ImporteTotal != "20.00" {
		t.Fatalf("la línea existente no debe cambiar: %+v", c.Lineas)
	}
	if c.Subtotal != "42.37" || c.Impuesto != "7.63" || c.Total != "50.00" {
		t.Fatalf("totales de la cuenta: %+v", c)
	}
}

func TestConsumoUsaElPrecioPropioDeLaSede(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos_sucursal SET precio_venta = 12.50 WHERE producto_id = %s`, prod))
	r := agregarConsumo(t, tk, p.ID, p.Cuentas[0].ID, prod, "1", "")
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	if l.PrecioUnitario != "12.500000" {
		t.Fatalf("debe usar el precio de la sede: %+v", l)
	}
}

func TestConsumoRechazos(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	ok := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	t.Run("cantidad no positiva", func(t *testing.T) {
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, ok, "0", ""), 422, "cantidad_invalida")
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, ok, "-1", ""), 422, "cantidad_invalida")
	})
	t.Run("producto inexistente", func(t *testing.T) {
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, "999999", "1", ""), 422, "producto_invalido")
	})
	t.Run("producto dado de baja", func(t *testing.T) {
		baja := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
		asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.productos SET es_activo=false, motivo_eliminacion='prueba' WHERE id=%s`, baja))
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, baja, "1", ""), 422, "producto_inactivo")
	})
	t.Run("producto pausado en la sede", func(t *testing.T) {
		pausado := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", false)
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, pausado, "1", ""), 422, "producto_no_disponible")
	})
	t.Run("producto sin fila en la sede", func(t *testing.T) {
		sinSede := productoVentaSQL(t, "", "PREPARADO", "10.00", true)
		esperar(t, agregarConsumo(t, tk, p.ID, cuenta, sinSede, "1", ""), 422, "producto_no_disponible")
	})
	t.Run("cuenta de otro pedido", func(t *testing.T) {
		otro := abrirLlevar(t, tk, "PRINCIPAL")
		esperar(t, agregarConsumo(t, tk, p.ID, otro.Cuentas[0].ID, ok, "1", ""), 422, "cuenta_invalida")
	})
	t.Run("pedido inexistente", func(t *testing.T) {
		esperar(t, agregarConsumo(t, tk, "999999", cuenta, ok, "1", ""), 404, "no_encontrado")
	})
	t.Run("campo desconocido", func(t *testing.T) {
		esperar(t, post(t, "/api/v1/pedidos/"+p.ID+"/detalles",
			fmt.Sprintf(`{"producto_id":%q,"cuenta_id":%q,"cantidad":"1","precio_unitario":"1.00"}`, ok, cuenta), tk), 400, "")
	})
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, p.ID); n != 0 {
		t.Fatalf("los rechazos no deben dejar líneas (%d)", n)
	}
}

func TestConsumoDescuentoExigePermiso(t *testing.T) {
	cajero, mesero := cajeroPedidos(t), meseroPedidos(t)
	p := abrirLlevar(t, cajero, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	extra := `,"descuento_monto":"5.00"`

	esperar(t, agregarConsumo(t, mesero, p.ID, cuenta, prod, "2", extra), http.StatusForbidden, "sin_permiso")
	r := agregarConsumo(t, cajero, p.ID, cuenta, prod, "2", extra)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	if l.DescuentoTotal != "5.00" || l.ImporteTotal != "15.00" {
		t.Fatalf("línea con descuento: %+v", l)
	}
	// El mesero sí agrega sin descuento.
	esperar(t, agregarConsumo(t, mesero, p.ID, cuenta, prod, "1", ""), http.StatusCreated, "")
	// El descuento no puede superar el importe.
	esperar(t, agregarConsumo(t, cajero, p.ID, cuenta, prod, "1", `,"descuento_monto":"10.01"`), 422, "descuento_excede_importe")
}

func TestConsumoAdicionalCuelgaDeUnaLinea(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	cuenta := p.Cuentas[0].ID
	plato := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "20.00", true)
	extra := productoVentaSQL(t, "PRINCIPAL", "ADICIONAL", "3.00", true)

	esperar(t, agregarConsumo(t, tk, p.ID, cuenta, extra, "1", ""), 422, "adicional_sin_padre")

	r := agregarConsumo(t, tk, p.ID, cuenta, plato, "1", "")
	esperar(t, r, http.StatusCreated, "")
	var padre lineaResp
	r.json(t, &padre)

	esperar(t, agregarConsumo(t, tk, p.ID, cuenta, plato, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID)), 422, "padre_no_aplica")
	esperar(t, agregarConsumo(t, tk, p.ID, cuenta, extra, "1", `,"detalle_padre_id":"999999"`), 422, "padre_invalido")

	r = agregarConsumo(t, tk, p.ID, cuenta, extra, "1", fmt.Sprintf(`,"detalle_padre_id":%q`, padre.ID))
	esperar(t, r, http.StatusCreated, "")
	var hija lineaResp
	r.json(t, &hija)
	if hija.DetallePadreID == nil || *hija.DetallePadreID != padre.ID {
		t.Fatalf("la línea hija debe apuntar a su padre: %+v", hija)
	}
}

func TestConsumoAgregarEsIdempotente(t *testing.T) {
	tk := cajeroPedidos(t)
	p := abrirLlevar(t, tk, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	cuerpo := fmt.Sprintf(`{"producto_id":%q,"cuenta_id":%q,"cantidad":"1"}`, prod, p.Cuentas[0].ID)
	h := map[string]string{"Idempotency-Key": nuevaClave()}
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles"

	r1 := callH(t, http.MethodPost, ruta, cuerpo, tk, h)
	r2 := callH(t, http.MethodPost, ruta, cuerpo, tk, h)
	esperar(t, r1, http.StatusCreated, "")
	esperar(t, r2, http.StatusCreated, "")
	var l1, l2 lineaResp
	r1.json(t, &l1)
	r2.json(t, &l2)
	if l1.ID != l2.ID {
		t.Fatalf("el reintento debe devolver la misma línea: %s / %s", l1.ID, l2.ID)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, p.ID); n != 1 {
		t.Fatalf("líneas = %d, quería 1", n)
	}
}

func TestConsumoEditar(t *testing.T) {
	cajero, mesero := cajeroPedidos(t), meseroPedidos(t)
	p := abrirLlevar(t, cajero, "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)
	r := agregarConsumo(t, cajero, p.ID, p.Cuentas[0].ID, prod, "2", `,"instrucciones":"sin sal"`)
	esperar(t, r, http.StatusCreated, "")
	var l lineaResp
	r.json(t, &l)
	ruta := "/api/v1/pedidos/" + p.ID + "/detalles/" + l.ID

	// Cantidad.
	r = call(t, "PATCH", ruta, `{"cantidad":"3"}`, mesero)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &l)
	if l.Cantidad != "3.000" || l.ImporteTotal != "30.00" || l.Notas == nil {
		t.Fatalf("tras editar la cantidad: %+v", l)
	}
	// Vaciar las instrucciones (null).
	r = call(t, "PATCH", ruta, `{"instrucciones":null}`, mesero)
	esperar(t, r, http.StatusOK, "")
	var vacia lineaResp
	r.json(t, &vacia)
	if vacia.Notas != nil {
		t.Fatalf("las instrucciones debían vaciarse: %+v", vacia)
	}
	// Sin cambios, cantidad inválida y desconocidos.
	esperar(t, call(t, "PATCH", ruta, `{}`, mesero), 422, "sin_cambios")
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"0"}`, mesero), 422, "cantidad_invalida")
	esperar(t, call(t, "PATCH", ruta, `{"precio_unitario":"1.00"}`, mesero), 400, "")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+p.ID+"/detalles/999999", `{"cantidad":"1"}`, mesero), 404, "no_encontrado")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+p.ID+"/detalles/abc", `{"cantidad":"1"}`, mesero), 400, "detalle_id_invalido")
	// Descuento: sin permiso 403; con permiso, 200; y no puede superar el importe.
	esperar(t, call(t, "PATCH", ruta, `{"descuento_monto":"1.00"}`, mesero), http.StatusForbidden, "sin_permiso")
	r = call(t, "PATCH", ruta, `{"descuento_monto":"10.00"}`, cajero)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &l)
	if l.DescuentoTotal != "10.00" || l.ImporteTotal != "20.00" {
		t.Fatalf("tras el descuento: %+v", l)
	}
	esperar(t, call(t, "PATCH", ruta, `{"descuento_monto":"30.01"}`, cajero), 422, "descuento_excede_importe")
	// Una línea anulada ya no se edita.
	asSystem(t, fmt.Sprintf(`UPDATE sch_restaurante.detalles_pedido SET estado='ANULADO', anulado_en=now(),
		anulado_por=(SELECT id FROM sch_restaurante.usuarios WHERE es_sistema), motivo_anulacion='prueba' WHERE id=%s`, l.ID))
	esperar(t, call(t, "PATCH", ruta, `{"cantidad":"1"}`, mesero), http.StatusConflict, "linea_anulada")
}
```

- [ ] **Step 4: Pruebas de sedes, permisos y lectura (RF-002)**

Crear `service-restaurant/tests/integration/pedidos_sedes_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// RES-CP-002: un usuario de la sede A no puede leer ni modificar un pedido de la sede B,
// aunque conozca su id.
func TestPedidosAislamientoPorSede(t *testing.T) {
	crearSedeSQL(t, "PED-AJENA")
	ajenoID, ajenaCuenta := pedidoSQL(t, "PED-AJENA")
	propio := abrirLlevar(t, cajeroPedidos(t), "PRINCIPAL")
	prod := productoVentaSQL(t, "PED-AJENA", "PREPARADO", "10.00", true)
	tk := cajeroPedidos(t) // solo PRINCIPAL

	esperar(t, call(t, "GET", "/api/v1/pedidos/"+ajenoID, "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, agregarConsumo(t, tk, ajenoID, ajenaCuenta, prod, "1", ""), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+ajenoID+"/detalles/1", `{"cantidad":"1"}`, tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, "PED-AJENA")), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/pedidos?sucursal_id="+sedeID(t, "PED-AJENA"), "", tk), http.StatusNotFound, "no_encontrado")

	ids := listarTodosPedidos(t, "", tk)
	if !slices.Contains(ids, propio.ID) || slices.Contains(ids, ajenoID) {
		t.Fatalf("el listado debe incluir el propio y no el ajeno: %v", ids)
	}
	if n := scalar(t, `SELECT count(*) FROM sch_restaurante.detalles_pedido WHERE pedido_id=$1::text::bigint`, ajenoID); n != 0 {
		t.Fatalf("el pedido ajeno no debe recibir líneas (%d)", n)
	}

	// Con la sede asignada el mismo usuario sí lo ve.
	tkAmbas := tokenPedidos(t, "ped.multi", "T_PED_CAJERO",
		[]string{"PEDIDO_VER", "PEDIDO_CREAR", "PEDIDO_EDITAR", "DESCUENTO_AUTORIZAR"}, "PED-AJENA")
	if g := obtenerPedido(t, tkAmbas, ajenoID); g.ID != ajenoID {
		t.Fatalf("con la sede asignada debe verlo: %+v", g)
	}
}

func TestPedidosVerSinPermisoDeEscritura(t *testing.T) {
	lector := lectorPedidos(t)
	p := abrirLlevar(t, cajeroPedidos(t), "PRINCIPAL")
	prod := productoVentaSQL(t, "PRINCIPAL", "PREPARADO", "10.00", true)

	// El lector ve el pedido, pero no puede abrir ni modificar: tiene PEDIDO_VER sin PEDIDO_CREAR ni PEDIDO_EDITAR.
	if g := obtenerPedido(t, lector, p.ID); g.ID != p.ID {
		t.Fatalf("el lector debe ver el pedido: %+v", g)
	}
	esperar(t, post(t, "/api/v1/pedidos",
		fmt.Sprintf(`{"sucursal_id":%q,"tipo":"LLEVAR","comensales":1}`, sedeID(t, "PRINCIPAL")), lector), http.StatusForbidden, "sin_permiso")
	esperar(t, agregarConsumo(t, lector, p.ID, p.Cuentas[0].ID, prod, "1", ""), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "PATCH", "/api/v1/pedidos/"+p.ID+"/detalles/1", `{"cantidad":"1"}`, lector), http.StatusForbidden, "sin_permiso")
}

func TestPedidosExigeAlgunPermisoDePedidos(t *testing.T) {
	rolConPermisos(t, "T_SIN_PEDIDOS", "NEGOCIO_VER")
	asegurarUsuario(t, "ped.sin", "Pedidos2026Clave", "T_SIN_PEDIDOS", "PRINCIPAL")
	tk := token(t, "ped.sin", "Pedidos2026Clave")
	esperar(t, call(t, "GET", "/api/v1/pedidos", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/pedidos/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/pedidos", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func listarPedidos(t *testing.T, query, tk string) pedidosPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/pedidos"+conLimite(query), "", tk)
	esperar(t, r, http.StatusOK, "")
	var p pedidosPagina
	r.json(t, &p)
	return p
}

// listarTodosPedidos sigue los cursores hasta agotar el listado: la base se comparte entre pruebas
// y una sola página de 100 podría dejar fuera los pedidos de esta.
func listarTodosPedidos(t *testing.T, query, tk string) []string {
	t.Helper()
	var ids []string
	cursor := ""
	for {
		q := query
		if cursor != "" {
			sep := "?"
			if strings.Contains(q, "?") {
				sep = "&"
			}
			q += sep + "cursor=" + cursor
		}
		p := listarPedidos(t, q, tk)
		ids = append(ids, p.ids()...)
		if p.NextCursor == nil {
			return ids
		}
		cursor = *p.NextCursor
	}
}

func (p pedidosPagina) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, it := range p.Items {
		out = append(out, it.ID)
	}
	return out
}

func TestPedidosFiltrosYPaginacion(t *testing.T) {
	tk := cajeroPedidos(t)
	a := abrirLlevar(t, tk, "PRINCIPAL")
	b := abrirLlevar(t, tk, "PRINCIPAL")
	mesa := mesaPedidosSQL(t, "PRINCIPAL", 4)
	r := post(t, "/api/v1/pedidos", fmt.Sprintf(`{"sucursal_id":%q,"tipo":"SALON","mesa_id":%q,"comensales":1}`, sedeID(t, "PRINCIPAL"), mesa), tk)
	esperar(t, r, http.StatusCreated, "")
	var s pedidoResp
	r.json(t, &s)

	if ids := listarTodosPedidos(t, "?tipo=SALON&mesa_id="+mesa, tk); !slices.Equal(ids, []string{s.ID}) {
		t.Fatalf("filtro por tipo y mesa: %v, quería [%s]", ids, s.ID)
	}
	if ids := listarTodosPedidos(t, "?tipo=LLEVAR", tk); !slices.Contains(ids, a.ID) || !slices.Contains(ids, b.ID) || slices.Contains(ids, s.ID) {
		t.Fatalf("filtro por tipo LLEVAR: %v", ids)
	}
	if ids := listarTodosPedidos(t, "?estado=CANCELADO&tipo=LLEVAR", tk); slices.Contains(ids, a.ID) {
		t.Fatalf("filtro por estado: %v", ids)
	}
	esperar(t, call(t, "GET", "/api/v1/pedidos?estado=QUIZAS", "", tk), http.StatusBadRequest, "estado_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?tipo=QUIZAS", "", tk), http.StatusBadRequest, "tipo_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?mesa_id=abc", "", tk), http.StatusBadRequest, "mesa_id_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/pedidos?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")

	p1 := listarPedidos(t, "?limit=1", tk)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarPedidos(t, "?limit=1&cursor="+*p1.NextCursor, tk)
	id1, _ := strconv.Atoi(p1.Items[0].ID)
	id2, _ := strconv.Atoi(p2.Items[0].ID)
	if len(p2.Items) != 1 || id1 >= id2 {
		t.Fatalf("la segunda página debe continuar en orden ascendente: %+v / %+v", p1, p2)
	}
}
```

- [ ] **Step 5: Ejecutar la batería de pedidos**

Run: `cd service-restaurant && go test ./tests/integration/ -run 'TestPedidos|TestConsumo|TestMigracion00011' -count=1 -v 2>&1 | tail -60`
Expected: todos PASS. Requiere Docker. Si una prueba falla, leer el mensaje: las causas más probables son (a) una columna con otro nombre en el SQL de fixtures, (b) un código de error distinto al esperado (leer el cuerpo `problem+json` que imprime `esperar`) o (c) un trigger de la base con una regla que el caso de uso no anticipó. Corregir el caso de uso o el SQL, no relajar la prueba.

- [ ] **Step 6: Ejecutar la suite completa de integración (regresión)**

Run: `cd service-restaurant && go test ./tests/integration/... -count=1 2>&1 | tail -15`
Expected: PASS. Confirma que `00011` y las rutas nuevas no rompen mesas, carta, identidad ni la prueba de bajas.

- [ ] **Step 7: Ejecutar con -race las pruebas concurrentes**

Run: `cd service-restaurant && go test ./tests/integration/ -run 'TestPedidosDosAperturasSimultaneasDeUnaMesa' -race -count=3`
Expected: PASS las tres veces, sin avisos de carrera.

- [ ] **Step 8: Commit**

```bash
git add service-restaurant/tests/integration/pedidos_helpers_test.go service-restaurant/tests/integration/pedidos_apertura_test.go service-restaurant/tests/integration/pedidos_consumos_test.go service-restaurant/tests/integration/pedidos_sedes_test.go
git commit -m "$(cat <<'EOF'
I3 rebanada 1: pruebas de integración de pedidos, consumos y sedes

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Contrato, expediente y cierre

**Files:**
- Modify: `docs/planificacion/03_api/openapi.yaml`
- Modify: `docs/planificacion/01_documentacion/requisitos_y_trazabilidad.json` y `.html`
- Modify: `docs/planificacion/01_documentacion/backlog_incrementos.json`
- Modify: `docs/planificacion/CAMBIOS_1_3.txt`, `00_LEEME.txt`, `MANIFIESTO.json`, `06_calidad/validacion_entrega.json`, `SHA256SUMS.txt`
- Modify: `service-restaurant/docs/module-map.md`
- Modify: `docs/superpowers/specs/2026-10-27-i3-rebanada1-pedidos-design.md` (§7 «Ajustes durante el plan»)
- Modify: `docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md` (nota sobre pedidos)

**Interfaces:**
- Consumes: el código de las Tasks 1 a 10 (rutas, códigos y campos reales).
- Produces: contrato y expediente coherentes con lo implementado.

- [ ] **Step 1: Script de actualización del contrato y del expediente**

Guardar como `$TEMP/cierre-i3-r1.mjs` (fuera del repositorio) y ejecutar desde `docs/planificacion/`:

```js
import fs from 'node:fs';
import crypto from 'node:crypto';

const leer = (f) => fs.readFileSync(f, 'utf8');
const lista = (j) => (Array.isArray(j) ? j : Object.values(j).find(Array.isArray));
const guardarJson = (f, j) => fs.writeFileSync(f, JSON.stringify(j, null, 2) + '\n');
const falla = (m) => { throw new Error(m); };

// ---- 1. openapi.yaml -------------------------------------------------------
const fApi = '03_api/openapi.yaml';
let api = leer(fApi);

// 1a. x-implementado y x-permiso de las cuatro operaciones existentes.
const permisos = {
  'get /pedidos': 'PEDIDO_VER',
  'post /pedidos': 'PEDIDO_CREAR',
  'get /pedidos/{id}': 'PEDIDO_VER',
  'post /pedidos/{id}/detalles': 'PEDIDO_EDITAR',
};
let ruta = '', metodo = '', marcadas = 0;
let lineas = api.split('\n').map((l) => {
  const p = /^  (\/[^:]*):\s*$/.exec(l);
  if (p) { ruta = p[1]; metodo = ''; }
  const m = /^    (get|post|put|patch|delete):\s*$/.exec(l);
  if (m) metodo = m[1];
  const clave = metodo + ' ' + ruta;
  if (permisos[clave]) {
    if (l === '      x-implementado: false') { marcadas++; return '      x-implementado: true'; }
    if (l.startsWith('      x-permiso:')) return '      x-permiso: ' + permisos[clave];
  }
  return l;
});
if (marcadas !== 4) falla('se esperaban 4 operaciones y fueron ' + marcadas);
api = lineas.join('\n');

// 1b. Las respuestas de POST /pedidos (operacion_113) y GET /pedidos/{id} (operacion_114) devuelven PedidoDetalle.
for (const op of ['operacion_113', 'operacion_114']) {
  const i = api.indexOf('operationId: ' + op);
  if (i < 0) falla(op + ' no existe');
  const ref = "$ref: '#/components/schemas/Pedidos'";
  const j = api.indexOf(ref, i);
  if (j < 0) falla('sin $ref Pedidos tras ' + op);
  api = api.slice(0, j) + "$ref: '#/components/schemas/PedidoDetalle'" + api.slice(j + ref.length);
}

// 1c. Filtros estado, tipo y mesa_id en GET /pedidos: se insertan antes de "    post:" de operacion_113.
{
  const i = api.indexOf('operationId: operacion_113');
  const k = api.lastIndexOf('\n    post:\n', i);
  if (k < 0) falla('no encontré el post de /pedidos');
  const params =
`
      - name: estado
        in: query
        schema:
          type: string
          enum:
          - REGISTRADO
          - PREPARACION
          - LISTO
          - ENTREGADO
          - CANCELADO
      - name: tipo
        in: query
        schema:
          type: string
          enum:
          - SALON
          - LLEVAR
          - DELIVERY
      - name: mesa_id
        in: query
        schema:
          $ref: '#/components/schemas/Id'`;
  api = api.slice(0, k) + params + api.slice(k);
}

// 1d. Nueva operación PATCH /pedidos/{id}/detalles/{detalle_id}.
{
  const ids = [...api.matchAll(/operationId: operacion_(\d+)/g)].map((m) => Number(m[1]));
  const sig = Math.max(...ids) + 1;
  const ancla = '\n  /pedidos/{id}/dividir-cuenta:\n';
  if (api.split(ancla).length !== 2) falla('ancla de dividir-cuenta no única');
  const op =
`
  /pedidos/{id}/detalles/{detalle_id}:
    patch:
      operationId: operacion_${sig}
      summary: Editar consumo
      tags:
      - Atención
      description: Cambia la cantidad, las instrucciones o el descuento de una línea activa de una cuenta abierta mientras no
        se haya enviado a cocina. El descuento exige DESCUENTO_AUTORIZAR y no puede superar el importe de la línea.
      x-implementado: true
      x-permiso: PEDIDO_EDITAR
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
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ConsumoEdicion'`;
  api = api.replace(ancla, op + ancla);
}

// 1e. Esquemas: reparto en PedidoAlta, detalle_padre_id en ConsumoAlta, y los nuevos.
{
  const a1 = '        notas:\n          type: string\n          maxLength: 2000\n    ConsumoAlta:\n';
  if (api.split(a1).length !== 2) falla('ancla de PedidoAlta.notas no única');
  api = api.replace(a1,
`        notas:
          type: string
          maxLength: 2000
        reparto:
          $ref: '#/components/schemas/DeliveryDatos'
          description: Obligatorio si tipo es DELIVERY y no se admite en los demás tipos.
    ConsumoAlta:
`);
  const a2 = '        instrucciones:\n          type: string\n          maxLength: 2000\n        descuento_monto:\n';
  if (api.split(a2).length !== 2) falla('ancla de ConsumoAlta.instrucciones no única');
  api = api.replace(a2,
`        detalle_padre_id:
          $ref: '#/components/schemas/Id'
          description: Solo para productos ADICIONAL; indica la línea activa de la misma cuenta de la que cuelga.
        instrucciones:
          type: string
          maxLength: 2000
        descuento_monto:
`);
  const a3 = '\n    DividirCuenta:\n';
  if (api.split(a3).length !== 2) falla('ancla de DividirCuenta no única');
  api = api.replace(a3,
`
    ConsumoEdicion:
      type: object
      additionalProperties: false
      properties:
        cantidad:
          $ref: '#/components/schemas/Cantidad'
        instrucciones:
          anyOf:
          - type: string
            maxLength: 2000
          - type: 'null'
          description: null vacía las instrucciones.
        descuento_monto:
          $ref: '#/components/schemas/Dinero'
      description: Debe enviarse al menos un campo. Un descuento exige permiso.
    RepartoPedido:
      type: object
      required:
      - destinatario
      - telefono
      - direccion
      - canal
      - estado
      properties:
        destinatario:
          type: string
        telefono:
          type: string
        direccion:
          type: string
        referencia:
          anyOf:
          - type: string
          - type: 'null'
        canal:
          type: string
        estado:
          type: string
    CuentaPedido:
      type: object
      required:
      - id
      - numero
      - estado
      - subtotal
      - impuesto
      - total
      - lineas
      properties:
        id:
          $ref: '#/components/schemas/Id'
        numero:
          type: integer
        cliente_id:
          anyOf:
          - $ref: '#/components/schemas/Id'
          - type: 'null'
        etiqueta:
          anyOf:
          - type: string
          - type: 'null'
        estado:
          type: string
        subtotal:
          $ref: '#/components/schemas/Dinero'
        impuesto:
          $ref: '#/components/schemas/Dinero'
        total:
          $ref: '#/components/schemas/Dinero'
        lineas:
          type: array
          items:
            $ref: '#/components/schemas/DetallesPedido'
      description: Los totales suman solo las líneas activas y los calcula la API.
    PedidoDetalle:
      description: Pedido con sus datos de reparto (solo DELIVERY) y sus cuentas con las líneas de cada una.
      allOf:
      - $ref: '#/components/schemas/Pedidos'
      - type: object
        required:
        - cuentas
        properties:
          reparto:
            anyOf:
            - $ref: '#/components/schemas/RepartoPedido'
            - type: 'null'
          cuentas:
            type: array
            items:
              $ref: '#/components/schemas/CuentaPedido'
    DividirCuenta:
`);
}
fs.writeFileSync(fApi, api);

// ---- 2. requisitos_y_trazabilidad.json y .html ------------------------------
const SIT = {
  'RES-RF-009': 'Implementado en I3 (rebanada 1): POST /pedidos abre el pedido y su cuenta nº 1 en una sola transacción (SALON con mesa, capacidad y habilitación validadas; LLEVAR; DELIVERY con sus datos de reparto), con Idempotency-Key y una sola apertura por mesa. Una falla al crear la cuenta no deja cabecera huérfana (prueba TestPedidosAperturaSinCabeceraHuerfana). Pendiente el frontend (R4). RES-CP-009 sigue NO_EJECUTADA.',
  'RES-RF-010': 'Implementado en I3 (rebanada 1): POST /pedidos/{id}/detalles fija descripción, precio y tasa vigente como instantánea; la API calcula importes; cantidades no positivas se rechazan y cambiar el precio de la carta no modifica una línea existente (prueba TestConsumoConservaElPrecioHistorico). PATCH edita cantidad, instrucciones y descuento (DESCUENTO_AUTORIZAR). Pendiente el frontend (R4). RES-CP-010 sigue NO_EJECUTADA.',
  'RES-RF-002': 'Implementado: el mecanismo (Actor.Can por permiso y sede) rige la administración de accesos y, desde I3 rebanada 1, los pedidos: un usuario de la sede A no lee ni modifica un pedido de la sede B aunque conozca su id (404; prueba TestPedidosAislamientoPorSede). Las operaciones de cocina, división y cancelación (R2 y R3) usarán el mismo mecanismo. RES-CP-002 sigue NO_EJECUTADA.',
};
const fReq = '01_documentacion/requisitos_y_trazabilidad.json';
const req = JSON.parse(leer(fReq));
for (const id of Object.keys(SIT)) {
  const r = lista(req).find((x) => x.id === id);
  if (!r) falla(id + ' no existe');
  r.estado_backend = 'IMPLEMENTADO';
  r.situacion_backend = SIT[id];
}
guardarJson(fReq, req);

const fHtml = '01_documentacion/requisitos_y_trazabilidad.html';
let html = leer(fHtml);
for (const id of Object.keys(SIT)) {
  const a = html.indexOf('id="' + id + '"');
  const b = html.indexOf('</tbody>', a);
  if (a < 0 || b < 0) falla('HTML sin sección ' + id);
  let bloque = html.slice(a, b);
  const re = /<td>(PENDIENTE|PARCIAL) · prototipo /;
  if (!re.test(bloque)) falla(id + ': el HTML no tenía el estado esperado; leer el archivo y ajustar este script');
  bloque = bloque.replace(re, '<td>IMPLEMENTADO · prototipo ');
  bloque = bloque.replace(/<tr><td>Situación backend<\/td><td>.*?<\/td><\/tr>/, '');
  bloque += '<tr><td>Situación backend</td><td>' + SIT[id] + '</td></tr>';
  html = html.slice(0, a) + bloque + html.slice(b);
}
fs.writeFileSync(fHtml, html);

// ---- 3. backlog_incrementos.json -------------------------------------------
const fBack = '01_documentacion/backlog_incrementos.json';
const back = JSON.parse(leer(fBack));
const i3 = lista(back).find((x) => (x.id || x.codigo) === 'I3');
if (!i3) falla('I3 no existe en el backlog');
i3.estado = 'EN_CURSO';
i3.avance = 'Rebanada 1 de 4 hecha (backend): apertura de pedido y cuenta (RES-RF-009), consumos como instantánea (RES-RF-010) y consultas, con autorización por sede (RES-RF-002). Faltan las comandas por estación y los estados de cocina (R2, RES-RF-012), dividir cuentas y cancelar con motivo (R3, RES-RF-011 y RES-RF-014) y el frontend de Salón y Cocina con el cierre con auditoría (R4). Los casos de aceptación siguen NO_EJECUTADA.';
guardarJson(fBack, back);

// ---- 4. Conteos del contrato (una ruta y una operación más) ----------------
const fMan = 'MANIFIESTO.json';
let man = leer(fMan);
if (!man.includes('"rutas_api_propuestas": 119') || !man.includes('"operaciones_api_propuestas": 163')) falla('MANIFIESTO: conteos distintos de 119/163');
man = man.replace('"rutas_api_propuestas": 119', '"rutas_api_propuestas": 120').replace('"operaciones_api_propuestas": 163', '"operaciones_api_propuestas": 164');
fs.writeFileSync(fMan, man);

const fLee = '00_LEEME.txt';
let lee = leer(fLee);
if (!lee.includes('119 rutas y 163 operaciones')) falla('00_LEEME: no menciona 119 rutas y 163 operaciones');
fs.writeFileSync(fLee, lee.replace('119 rutas y 163 operaciones', '120 rutas y 164 operaciones'));

const fVal = '06_calidad/validacion_entrega.json';
const val = JSON.parse(leer(fVal));
const sVal = JSON.stringify(val);
if (!sVal.includes('119 rutas y 163 operaciones')) falla('validacion_entrega: no menciona 119 rutas y 163 operaciones');
guardarJson(fVal, JSON.parse(sVal.replace('119 rutas y 163 operaciones', '119 rutas y 163 operaciones, y 120 rutas y 164 operaciones tras añadir la edición de consumos en I3')));

// ---- 5. CAMBIOS_1_3.txt ----------------------------------------------------
const fCam = 'CAMBIOS_1_3.txt';
let cam = leer(fCam);
if (!cam.endsWith('\n')) cam += '\n';
cam +=
  '- I3 rebanada 1: pedidos, cuentas y consumos (backend). RES-RF-009 y RES-RF-010 pasan a IMPLEMENTADO en backend y\n' +
  '  RES-RF-002 pasa de PARCIAL a IMPLEMENTADO (aislamiento por sede de los pedidos); los casos de prueba siguen NO_EJECUTADA.\n' +
  '  El contrato marca 4 operaciones de /pedidos como implementadas, añade PATCH /pedidos/{id}/detalles/{detalle_id}\n' +
  '  (120 rutas y 164 operaciones), los filtros estado, tipo y mesa_id del listado, el campo reparto de PedidoAlta,\n' +
  '  detalle_padre_id de ConsumoAlta y los esquemas PedidoDetalle, CuentaPedido, RepartoPedido y ConsumoEdicion.\n' +
  '  El descuento usa el permiso existente DESCUENTO_AUTORIZAR. I3 pasa a EN_CURSO.\n';
fs.writeFileSync(fCam, cam);

// ---- 6. SHA256SUMS.txt -----------------------------------------------------
const tocados = new Set([fApi, fReq, fHtml, fBack, fMan, fLee, fVal, fCam]);
const sums = leer('SHA256SUMS.txt').split('\n').map((l) => {
  const m = /^([0-9a-f]{64})  (.+)$/.exec(l);
  if (!m || !tocados.has(m[2])) return l;
  return crypto.createHash('sha256').update(fs.readFileSync(m[2])).digest('hex') + '  ' + m[2];
});
fs.writeFileSync('SHA256SUMS.txt', sums.join('\n'));
console.log('ok: 4 operaciones marcadas, 1 nueva, 3 requisitos IMPLEMENTADO, I3 EN_CURSO');
```

- [ ] **Step 2: Ejecutarlo**

Run (desde `docs/planificacion/`): `node "$TEMP/cierre-i3-r1.mjs"`
Expected: `ok: 4 operaciones marcadas, 1 nueva, 3 requisitos IMPLEMENTADO, I3 EN_CURSO`. Si lanza un error de ancla («no única», «no existe», «conteos distintos»), abrir el archivo real, entender la diferencia y corregir el script; **no forzar** un cambio. Si falla a mitad, `git checkout -- docs/planificacion` y reintentar con el script corregido (el script escribe varios archivos).

- [ ] **Step 3: Validar el contrato y los checksums**

Run (desde `docs/planificacion/`):

```bash
node -e 'JSON.parse(require("fs").readFileSync("01_documentacion/requisitos_y_trazabilidad.json","utf8")); JSON.parse(require("fs").readFileSync("01_documentacion/backlog_incrementos.json","utf8")); console.log("json ok")'
sha256sum -c SHA256SUMS.txt 2>&1 | grep -v ": OK"
git diff --stat .
grep -c "x-implementado: true" 03_api/openapi.yaml
```

Expected: `json ok`; `sha256sum` sin líneas de error; el diff muestra los 8 archivos esperados; el conteo de `x-implementado: true` sube en 5 respecto a antes (4 marcadas + 1 nueva; era 83 → 88). Para validar el YAML, si hay una herramienta disponible (`npx --yes @redocly/cli lint 03_api/openapi.yaml` o `python -c "import yaml"`), usarla; si no, comprobar al menos que el archivo carga con `node -e 'require("yaml")'` o decirlo explícitamente en el informe: no marcar el contrato como validado sin una validación real.

- [ ] **Step 4: Documentación del repositorio**

En `service-restaurant/docs/module-map.md`:

- En el párrafo «Módulos hoy», sustituir «**`atencion`** (zonas y mesas, implementado en I2)» por «**`atencion`** (zonas y mesas, implementado en I2; pedidos, cuentas y consumos, implementado en I3 rebanada 1)».
- En la tabla, fila «Atención»: cambiar la celda de tablas a «zonas y mesas (`00003`, I2) · pedidos, cuentas, detalles y datos de reparto (`00003`, I3 rebanada 1) · reservas (I6)» y el incremento a «I2 / I3 (reservas: I6)».

En `docs/superpowers/specs/2026-10-27-i3-rebanada1-pedidos-design.md`, añadir al final:

```markdown

## 7. Ajustes durante el plan

Al escribir el plan (`docs/superpowers/plans/2026-10-27-i3-rebanada1-pedidos.md`) se reconcilió este diseño con el código y el contrato existentes. Las diferencias con las secciones anteriores son:

| Sección | Diseño | Implementado | Motivo |
|---|---|---|---|
| D5, D4 | Permiso nuevo `PEDIDO_DESCONTAR` | `DESCUENTO_AUTORIZAR` (ya existe desde `00001`), asignado a CAJERO | Evitar un permiso duplicado |
| D5 | `PEDIDO_CREAR` nuevo | Ya existe; `00011` lo asigna a MESERO y CAJERO y crea solo `PEDIDO_VER` y `PEDIDO_EDITAR` | Evitar insertarlo dos veces |
| D2 | `notas_preparacion`, `descuento_total`, `cuenta_id` opcional | `instrucciones`, `descuento_monto`, `cuenta_id` obligatorio | Nombres del contrato `ConsumoAlta` |
| D1 | `reparto` con `ubigeo`, coordenadas y `programado_en` | `reparto` = `DeliveryDatos` (destinatario, teléfono, dirección, referencia, canal) | Los campos extra son de `PUT /pedidos/{id}/delivery` (I6) |
| D4 | Código `permiso_insuficiente` | `sin_permiso` | Código ya usado por todo el módulo |
| D1 | Repetición idempotente: 200 | 201 (el estado original) | Así lo hace `idempotency.Run` |
| D7 | `version_bloqueo` no se expone | Se expone de solo lectura | Lo exige el esquema `Pedidos` |
| §2 | `POST detalles` con `PEDIDO_CREAR` (contrato) | `PEDIDO_EDITAR` | Lo fija D3/D5; el contrato se corrigió |
```

En `docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md`, añadir al final:

```markdown

## Nota (I3 rebanada 1)

Los pedidos, las cuentas y sus líneas **no tienen baja lógica**: se cancelan o se anulan con motivo (RES-RF-014, rebanada 3 de I3), y esa operación se audita aparte. Por eso no entran en esta matriz. Sus formularios (abrir pedido, agregar y editar consumos) llegan con el frontend de la rebanada 4 de I3.
```

- [ ] **Step 5: Verificación final**

Run (desde `service-restaurant/`): `go build ./... && go vet ./... && go test ./internal/... ./tests/architecture/... -count=1 && go test ./tests/integration/... -count=1 2>&1 | tail -10`
Expected: todo PASS. Si Docker no está activo, decirlo y no declarar verificada la integración.

- [ ] **Step 6: Commit**

```bash
git add docs/planificacion docs/superpowers/specs service-restaurant/docs/module-map.md
git commit -m "$(cat <<'EOF'
I3 rebanada 1: contrato y expediente de pedidos (RF-009, RF-010, RF-002)

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7: Entrega**

Mostrar al usuario `git log --oneline origin/main..HEAD` (o `main..HEAD`) y el resumen de pruebas. **No** hacer `git push` ni abrir el PR sin su confirmación expresa; ofrecerlo en una línea.

---

## Auto-revisión del plan contra el spec

| Spec | Tarea |
|---|---|
| §2 `POST /pedidos` | 4 (caso de uso), 7 (SQL), 8 (HTTP), 10 (pruebas) |
| §2 `GET /pedidos`, `GET /pedidos/{id}` | 6, 7, 8, 10 |
| §2 `POST /pedidos/{id}/detalles` | 5, 7, 8, 10 |
| §2 `PATCH /pedidos/{id}/detalles/{did}` | 5, 7, 8, 10 |
| D1 apertura por tipo, mesa, mesero, moneda, transacción, idempotencia | 4, 7, 10 (`TestPedidosAbrir*`, `…SinCabeceraHuerfana`, `…Idempotente`) |
| D2 instantánea, validaciones, adicionales, idempotencia de líneas | 5, 7, 10 (`TestConsumo*`) |
| D3 edición y reglas de cuenta/pedido/comanda | 5 (`linea_en_cocina` con fake), 10 (`TestConsumoEditar`) |
| D4 descuento con permiso | 5, 10 (`TestConsumoDescuentoExigePermiso`) |
| D5 permisos | 1, 2 |
| D6 alcance por sede | 2, 4–6, 10 (`TestPedidosAislamientoPorSede`) |
| D7 concurrencia | 7 (`FOR UPDATE`), 10 (`TestPedidosDosAperturasSimultaneasDeUnaMesa`) |
| D8 totales | 7 (`cargar.totales`), 10 (`TestConsumoConservaElPrecioHistorico`) |
| §4 estructura de capas | 2–9 |
| §5 pruebas, incl. arquitectura | 2–6 (unitarias), 9 (arquitectura), 10 (integración) |
| §6 trazabilidad y matriz | 11 |

Huecos conocidos y aceptados: la regla «línea enviada a cocina no se edita» solo se prueba con el fake (no hay comandas hasta R2); se verificará con integración en R2. El rechazo 403 `sede_no_autorizada` se prueba en unitarias (un mismo usuario hereda sus permisos en todas sus sedes, por lo que el escenario no se puede armar con el modelo de roles actual en integración).
