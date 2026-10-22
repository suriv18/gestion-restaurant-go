# I2 · Paso 2 — Módulo `atencion`: zonas y mesas — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Crear el módulo `atencion` con la API de zonas (`/zonas`) y mesas (`/mesas`): listar, leer, crear, editar, dar de baja y restaurar, con permisos `MESA_VER` / `MESA_ADMINISTRAR` por sede.

**Architecture:** Módulo nuevo `internal/modules/atencion` con las mismas capas CQRS que `identidad` (`domain`, `application`, `infrastructure`, `web`). No importa `identidad` (regla verificada por `tests/architecture`): replica los helpers HTTP y el decorador transaccional y define sus propias constantes de permiso. Se compone en `internal/bootstrap` y comparte el `protect` (autenticación) de `identidad`. La migración `00008` (paso 1) ya dejó los permisos `MESA_*` y las columnas de baja con motivo en `mesas`.

**Tech Stack:** Go 1.26, `net/http` (mux 1.22), `database/sql` + pgx, PostgreSQL 18, testcontainers, módulo `service-restaurant`.

**Spec:** `docs/superpowers/specs/2026-10-18-i2-configuracion-salon-design.md` (§1–§5, §8, §9 paso 2). **Requisito previo:** el paso 1 está integrado en `main`.

## Global Constraints

- Todos los comandos se ejecutan desde `service-restaurant/`. Las pruebas de integración necesitan Docker activo.
- Los IDs viajan como **cadenas** en JSON (`"id": "7"`, `"sucursal_id": "1"`, `"zona_id": "3"`); internamente son `int64`.
- Errores en `application/problem+json` con los códigos de I1: 400, 401, 403 (`sin_permiso`), 404 (`no_encontrado`), 409, 422; el código del error nombra el campo (`nombre_invalido`, `capacidad_invalida`, …).
- Los `POST` de creación, baja y restauración exigen `Idempotency-Key` (UUID); sin ella → 400 `idempotency_key_requerida`. Los `PATCH` no la exigen.
- `creado_por`, `actualizado_por` y `eliminado_*` nunca se aceptan del cliente (`DecodeJSON` rechaza campos desconocidos → 400 `json_invalido`); el actor sale de `app.usuario_id`.
- Permisos: lectura `MESA_VER` o `MESA_ADMINISTRAR` en la sede del recurso; escritura `MESA_ADMINISTRAR` en la sede del recurso. Zonas y mesas comparten los dos permisos. **Una sede ajena (sin ningún permiso `MESA_*` en ella) responde 404** para no revelar su existencia; con permiso de lectura pero sin `MESA_ADMINISTRAR` en la sede, responde 403 `sede_no_autorizada`.
- Listados: `limit` (1–100, por defecto 25), `cursor` opaco (id ascendente), filtros `sucursal_id` y `es_activo`. Sin `sucursal_id` se listan las sedes donde el actor tiene `MESA_VER` o `MESA_ADMINISTRAR`; con un `sucursal_id` ajeno → 404.
- **Zona:** nombre único por sede (409 `nombre_duplicado`); baja simple, **sin motivo**; la baja se rechaza con 409 `zona_con_mesas` si tiene mesas activas; `sucursal_id` no se edita.
- **Mesa:** número único por sede (409 `numero_mesa_duplicado`); la zona debe existir, estar activa y pertenecer a la misma sede (422 `zona_invalida`); capacidad > 0; `habilitada` se cambia con `PATCH` sin motivo; baja **con motivo** (máx. 2000, `motivo_requerido`); una mesa dada de baja no se edita ni se habilita hasta restaurarla (409 `mesa_inactiva`); la ocupación no se expone.
- Reglas de la base que se traducen a 422 `regla_de_datos` con su mensaje curado: no dar de baja una mesa con pedido abierto; no restaurar una mesa cuya zona esté dada de baja.
- Dependencias entre capas: `domain` no importa infraestructura; `application` no importa `database/sql` ni `net/http`; ningún módulo importa internos de otro.
- Estilo: comentarios y mensajes en español, igual que el código vecino; `gofmt` limpio; commits en español con el trailer `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.

## Decisiones derivadas (no están en el spec)

| Tema | Decisión |
|---|---|
| Código duplicado de helpers | `atencion` no puede importar `identidad`, así que replica `responder`, `actorDe`, `pathID`, `intencion`, `consulta`, `limiteDe`, `Opcional[T]` y el decorador `escritura`. Es deuda técnica consciente (≈250 líneas): extraerlos a `internal/platform` es una refactorización posterior que toca `identidad` y queda fuera de esta rebanada. |
| Zona o mesa inactiva | No se edita (409 `zona_inactiva` / `mesa_inactiva`); sí se lista y se restaura. |
| Posición de mesa | Decimales exactos como **cadena** (`"12.50"`), `numeric(10,2)`: hasta 8 enteros y 2 decimales, con signo. |
| Capacidad / orden | Capacidad 1–500; `orden_visualizacion` 0–9999. |
| Filtro `zona_id` en `/mesas` | No se implementa (YAGNI); el frontend agrupa en cliente. |
| Prueba de "baja con pedido abierto" | La regla vive en el trigger `a2_baja_operativa` de la V4 (ya probado en I3 cuando existan pedidos). Aquí solo se verifica la traducción del error con una prueba de la restauración con zona inactiva. |

## Estructura de archivos

Todo bajo `internal/modules/atencion/` salvo indicación.

| Archivo | Responsabilidad |
|---|---|
| `domain/policy/permisos.go` | `MesaVer`, `MesaAdministrar` |
| `domain/vo/texto.go`, `mesa.go` (+ `_test.go`) | `TextoObligatorio`, `NumeroMesa`, `Capacidad`, `Orden`, `Posicion` |
| `application/mapper/errores.go`, `autorizacion.go` (+ test) | Códigos, rechazos, cursor, `Requiere`, `SedesVisibles`, `PuedeVer`, `AutorizarSede` |
| `application/port/outbound/errores.go`, `zonas.go`, `mesas.go` | Errores de almacenamiento, `ZonaReader/Store`, `MesaReader/Store` |
| `application/port/inbound/zonas.go`, `mesas.go` | Interfaces de casos de uso |
| `application/dto/{command,query,result}/zonas.go`, `mesas.go`, `command/intencion.go` | DTO de aplicación |
| `application/usecase/{command,query}/…` (+ tests) | Casos de uso |
| `infrastructure/persistence/{read,write}/{repository,adapter}/…` | SQL y adaptadores |
| `infrastructure/config/module.go`, `escritura.go`, `zonas.go`, `mesas.go` | Composición |
| `web/dto/request/…`, `web/dto/response/…`, `web/mapper/…`, `web/controller/…` | HTTP |
| `internal/bootstrap/app.go`, `router.go` (modificar) | Monta el módulo |
| `tests/architecture/import_rules_test.go` (modificar) | `atencion` no importa otros módulos |
| `tests/integration/atencion_*_test.go` | Integración |

---

### Task 1: Cimientos del módulo (dominio, mapper, errores, arquitectura)

**Files:**
- Create: `domain/policy/permisos.go`, `domain/vo/texto.go`, `domain/vo/mesa.go`, `domain/vo/mesa_test.go`
- Create: `application/port/outbound/errores.go`
- Create: `application/mapper/errores.go`, `application/mapper/autorizacion.go`, `application/mapper/autorizacion_test.go`
- Modify: `tests/architecture/import_rules_test.go` (añadir una prueba al final)

**Interfaces:**
- Produces:
  - `policy.MesaVer`, `policy.MesaAdministrar` (string).
  - `vo.TextoObligatorio(raw string, max int) (string, error)`; `vo.NewNumeroMesa(string) (NumeroMesa, error)`; `vo.NewCapacidad(int) (Capacidad, error)` con `Int() int`; `vo.NewOrden(int) (Orden, error)` con `Int() int`; `vo.NewPosicion(string) (Posicion, error)`; cada tipo de texto con `String() string`; errores `vo.ErrTextoInvalido`, `vo.ErrValorInvalido`.
  - `outbound.ErrDuplicado`, `outbound.ErrReferenciaInvalida`, `outbound.ErrReglaDatos{Mensaje string}`.
  - `mapper.Requiere(a, mensaje, permisos...) *apperror.AppError`; `mapper.SedesVisibles(a) []int64`; `mapper.PuedeVer(a, sedeID) bool`; `mapper.AutorizarSede(a, sedeID) *apperror.AppError`; `mapper.NoAutenticado()`, `NoEncontrado(que)`, `Validacion(code, msg)`, `DeAlmacen(err)`, `EncodeCursor(id)`, `DecodeCursor(c)`; constantes `CodigoSinPermiso`, `CodigoNoEncontrado`, `CodigoSedeNoAutorizada`, `CodigoReglaDatos`, `CodigoDuplicado`, `CodigoReferenciaInvalida`, `CodigoCursorInvalido`, `CodigoNoAutenticado`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`domain/vo/mesa_test.go`:

```go
package vo

import (
	"errors"
	"strings"
	"testing"
)

func TestNumeroMesa(t *testing.T) {
	if n, err := NewNumeroMesa("  A-12 "); err != nil || n.String() != "A-12" {
		t.Fatalf("número válido: %v %q", err, n.String())
	}
	for _, malo := range []string{"", "   ", strings.Repeat("9", 21), "A\n1", "A\t1"} {
		if _, err := NewNumeroMesa(malo); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("número %q debía rechazarse con ErrValorInvalido: %v", malo, err)
		}
	}
	if _, err := NewNumeroMesa(strings.Repeat("9", 20)); err != nil {
		t.Errorf("20 caracteres es el máximo permitido: %v", err)
	}
}

func TestCapacidadYOrden(t *testing.T) {
	if c, err := NewCapacidad(4); err != nil || c.Int() != 4 {
		t.Fatalf("capacidad válida: %v", err)
	}
	for _, malo := range []int{0, -1, 501} {
		if _, err := NewCapacidad(malo); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("capacidad %d debía rechazarse", malo)
		}
	}
	if _, err := NewCapacidad(500); err != nil {
		t.Errorf("500 es el máximo: %v", err)
	}
	if o, err := NewOrden(0); err != nil || o.Int() != 0 {
		t.Fatalf("orden 0 es válido: %v", err)
	}
	for _, malo := range []int{-1, 10000} {
		if _, err := NewOrden(malo); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("orden %d debía rechazarse", malo)
		}
	}
}

func TestPosicion(t *testing.T) {
	for _, ok := range []string{"0", "12.5", "-3.25", "99999999.99", " 7 "} {
		if p, err := NewPosicion(ok); err != nil || p.String() != strings.TrimSpace(ok) {
			t.Errorf("posición %q: %v", ok, err)
		}
	}
	for _, mala := range []string{"", "abc", "1.234", "123456789", ".5", "1e3", "--1", "1,5"} {
		if _, err := NewPosicion(mala); !errors.Is(err, ErrValorInvalido) {
			t.Errorf("posición %q debía rechazarse", mala)
		}
	}
}
```

`application/mapper/autorizacion_test.go`:

```go
package mapper

import (
	"errors"
	"slices"
	"testing"

	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

func actorCon(g ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, g) }

func TestRequiere(t *testing.T) {
	if ae := Requiere(appctx.Actor{}, "x", policy.MesaVer); ae == nil || ae.Kind != apperror.KindUnauthorized || ae.Code != CodigoNoAutenticado {
		t.Fatalf("sin actor debe ser 401: %v", ae)
	}
	a := actorCon(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	if ae := Requiere(a, "x", policy.MesaAdministrar); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSinPermiso {
		t.Fatalf("sin permiso debe ser 403: %v", ae)
	}
	if ae := Requiere(a, "x", policy.MesaAdministrar, policy.MesaVer); ae != nil {
		t.Fatalf("alguno de los permisos basta: %v", ae)
	}
}

func TestSedesVisiblesYAutorizarSede(t *testing.T) {
	a := actorCon(
		appctx.Grant{SucursalID: 5, Permiso: policy.MesaVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.MesaAdministrar},
		appctx.Grant{SucursalID: 5, Permiso: policy.MesaAdministrar},
		appctx.Grant{SucursalID: 9, Permiso: "NEGOCIO_VER"},
	)
	if got := SedesVisibles(a); !slices.Equal(got, []int64{2, 5}) {
		t.Fatalf("sedes visibles = %v, quería [2 5]", got)
	}
	if got := SedesVisibles(actorCon()); got == nil || len(got) != 0 {
		t.Fatalf("sin sedes debe devolver un slice vacío no nulo: %#v", got)
	}
	if ae := AutorizarSede(a, 2); ae != nil {
		t.Errorf("administra la sede 2: %v", ae)
	}
	if ae := AutorizarSede(a, 7); ae == nil || ae.Kind != apperror.KindNotFound {
		t.Errorf("una sede ajena responde 404: %v", ae)
	}
	soloVer := actorCon(appctx.Grant{SucursalID: 3, Permiso: policy.MesaVer})
	if ae := AutorizarSede(soloVer, 3); ae == nil || ae.Kind != apperror.KindForbidden || ae.Code != CodigoSedeNoAutorizada {
		t.Errorf("ver sin administrar responde 403 sede_no_autorizada: %v", ae)
	}
}

func TestDeAlmacen(t *testing.T) {
	if DeAlmacen(nil) != nil || DeAlmacen(errors.New("técnico")) != nil {
		t.Fatal("nil y errores técnicos no son rechazos esperados")
	}
	if ae := DeAlmacen(outbound.ErrDuplicado); ae == nil || ae.Kind != apperror.KindConflict || ae.Code != CodigoDuplicado {
		t.Errorf("duplicado: %v", ae)
	}
	if ae := DeAlmacen(outbound.ErrReferenciaInvalida); ae == nil || ae.Kind != apperror.KindValidation {
		t.Errorf("referencia: %v", ae)
	}
	if ae := DeAlmacen(&outbound.ErrReglaDatos{Mensaje: "Cerrar el pedido"}); ae == nil || ae.Kind != apperror.KindBusinessRule ||
		ae.Code != CodigoReglaDatos || ae.Message != "Cerrar el pedido" {
		t.Errorf("regla de datos: %v", ae)
	}
}

func TestCursor(t *testing.T) {
	if id, ae := DecodeCursor(""); id != 0 || ae != nil {
		t.Fatalf("vacío es la primera página: %d %v", id, ae)
	}
	if id, ae := DecodeCursor(EncodeCursor(42)); id != 42 || ae != nil {
		t.Fatalf("ida y vuelta: %d %v", id, ae)
	}
	if _, ae := DecodeCursor("%%%"); ae == nil || ae.Code != CodigoCursorInvalido || ae.Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido: %v", ae)
	}
}
```

Añadir al final de `tests/architecture/import_rules_test.go`:

```go
// El módulo atencion no puede depender de los internos de identidad (la coordinación entre
// módulos ocurre en la composición).
func TestAtencionNoImportaOtrosModulos(t *testing.T) {
	paquetes := []string{
		"internal/modules/atencion/domain/vo",
		"internal/modules/atencion/domain/policy",
		"internal/modules/atencion/application/usecase/command",
		"internal/modules/atencion/infrastructure/persistence/write/repository",
		"internal/modules/atencion/web/controller",
	}
	for _, pkg := range paquetes {
		if violation(pkg, modulePath+"/internal/modules/identidad/domain/policy") == "" {
			t.Errorf("%s no debe poder importar identidad", pkg)
		}
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `go test ./internal/modules/atencion/... 2>&1 | head -20`
Expected: FAIL de compilación (`undefined: NewNumeroMesa`, `undefined: Requiere`, `no required module`/paquete sin archivos no productivos).

- [ ] **Step 3: Implementar**

`domain/policy/permisos.go`:

```go
// Package policy contiene las reglas de permiso del módulo atencion.
package policy

// Permisos de zonas y mesas (migración 00008). Se declaran aquí porque atencion no importa
// identidad; el catálogo de permisos vive en la base.
const (
	MesaVer         = "MESA_VER"
	MesaAdministrar = "MESA_ADMINISTRAR"
)
```

`domain/vo/texto.go`:

```go
// Package vo contiene los objetos de valor del módulo atencion; solo usa la biblioteca estándar.
package vo

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrTextoInvalido = errors.New("texto inválido")
	ErrValorInvalido = errors.New("valor inválido")
)

// TextoObligatorio recorta y exige 1..max caracteres sin caracteres de control.
func TextoObligatorio(raw string, max int) (string, error) {
	v := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(v)
	if n == 0 || n > max {
		return "", ErrTextoInvalido
	}
	for _, r := range v {
		if r < 0x20 && r != '\n' && r != '\t' {
			return "", ErrTextoInvalido
		}
	}
	return v, nil
}
```

`domain/vo/mesa.go`:

```go
package vo

import (
	"regexp"
	"strings"
)

const (
	maxNumeroMesa = 20
	maxCapacidad  = 500
	maxOrden      = 9999
)

var posicionRe = regexp.MustCompile(`^-?[0-9]{1,8}(\.[0-9]{1,2})?$`)

// NumeroMesa identifica la mesa dentro de su sede (1 a 20 caracteres, sin saltos ni tabulaciones).
type NumeroMesa struct{ value string }

func NewNumeroMesa(raw string) (NumeroMesa, error) {
	v, err := TextoObligatorio(raw, maxNumeroMesa)
	if err != nil || strings.ContainsAny(v, "\n\t") {
		return NumeroMesa{}, ErrValorInvalido
	}
	return NumeroMesa{v}, nil
}

func (n NumeroMesa) String() string { return n.value }

// Capacidad es el número de comensales de la mesa (1 a 500).
type Capacidad struct{ value int }

func NewCapacidad(n int) (Capacidad, error) {
	if n < 1 || n > maxCapacidad {
		return Capacidad{}, ErrValorInvalido
	}
	return Capacidad{n}, nil
}

func (c Capacidad) Int() int { return c.value }

// Orden es el orden de visualización de una zona (0 a 9999).
type Orden struct{ value int }

func NewOrden(n int) (Orden, error) {
	if n < 0 || n > maxOrden {
		return Orden{}, ErrValorInvalido
	}
	return Orden{n}, nil
}

func (o Orden) Int() int { return o.value }

// Posicion es una coordenada decimal exacta de numeric(10,2): hasta 8 enteros y 2 decimales.
type Posicion struct{ value string }

func NewPosicion(raw string) (Posicion, error) {
	v := strings.TrimSpace(raw)
	if !posicionRe.MatchString(v) {
		return Posicion{}, ErrValorInvalido
	}
	return Posicion{v}, nil
}

func (p Posicion) String() string { return p.value }
```

`application/port/outbound/errores.go`:

```go
// Package outbound declara los puertos de salida del módulo atencion.
package outbound

import "errors"

// Errores de almacenamiento que los casos de uso traducen a rechazos esperados.
var (
	ErrDuplicado          = errors.New("registro duplicado")
	ErrReferenciaInvalida = errors.New("referencia inexistente")
)

// ErrReglaDatos señala una regla de integridad de la base (triggers de la V4).
type ErrReglaDatos struct{ Mensaje string }

func (e *ErrReglaDatos) Error() string { return "regla de datos: " + e.Mensaje }
```

`application/mapper/errores.go`:

```go
// Package mapper convierte errores en rechazos esperados de aplicación y resuelve la
// autorización por sede del módulo atencion.
package mapper

import (
	"encoding/base64"
	"errors"
	"strconv"

	"service-restaurant/internal/modules/atencion/application/port/outbound"
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

func NoEncontrado(que string) *apperror.AppError {
	return apperror.New(apperror.KindNotFound, CodigoNoEncontrado, que+" no encontrado")
}

func Validacion(code, msg string) *apperror.AppError {
	return apperror.New(apperror.KindValidation, code, msg)
}

// DeAlmacen traduce los errores de almacenamiento conocidos a rechazos esperados.
// Devuelve nil si err no es uno de ellos (error técnico: debe seguir por el canal error).
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
		return apperror.New(apperror.KindBusinessRule, CodigoReglaDatos, regla.Mensaje)
	}
	return nil
}

// EncodeCursor / DecodeCursor: el cursor es el último id entregado, opaco para el cliente.
func EncodeCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func DecodeCursor(c string) (int64, *apperror.AppError) {
	if c == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err == nil {
		if id, perr := strconv.ParseInt(string(raw), 10, 64); perr == nil && id >= 0 {
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
	"slices"

	"service-restaurant/internal/modules/atencion/domain/policy"
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

// SedesVisibles devuelve, ordenadas y sin repetir, las sedes donde el actor tiene MESA_VER o
// MESA_ADMINISTRAR. Nunca es nil, para que un parámetro SQL vacío sea un arreglo vacío.
func SedesVisibles(a appctx.Actor) []int64 {
	ids := append(a.SucursalesCon(policy.MesaVer), a.SucursalesCon(policy.MesaAdministrar)...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if ids == nil {
		return []int64{}
	}
	return ids
}

// PuedeVer indica si el actor tiene algún permiso de mesa en la sede.
func PuedeVer(a appctx.Actor, sedeID int64) bool { return slices.Contains(SedesVisibles(a), sedeID) }

// AutorizarSede decide si el actor puede escribir en la sede: sede que no ve → 404 (no se
// revela su existencia); la ve pero no la administra → 403.
func AutorizarSede(a appctx.Actor, sedeID int64) *apperror.AppError {
	if !PuedeVer(a, sedeID) {
		return NoEncontrado("sucursal")
	}
	if !a.Can(policy.MesaAdministrar, sedeID) {
		return apperror.New(apperror.KindForbidden, CodigoSedeNoAutorizada, "no administra las mesas de esta sucursal")
	}
	return nil
}
```

- [ ] **Step 4: Ejecutar**

Run: `gofmt -l internal/ tests/ ; go vet ./... && go test ./internal/modules/atencion/... ./tests/architecture/ -v 2>&1 | tail -30`
Expected: sin salida de `gofmt`; PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/atencion tests/architecture
git commit -m "I2 paso 2: cimientos del módulo atencion (permisos, objetos de valor, errores y autorización por sede)

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Módulo atencion en marcha + zonas (listar y consultar)

**Files:**
- Create: `application/dto/command/intencion.go`, `application/dto/query/zonas.go`, `application/dto/result/zonas.go`
- Create: `application/port/inbound/zonas.go`, `application/port/outbound/zonas.go`
- Create: `application/usecase/query/zonas.go`, `application/usecase/query/zonas_test.go`
- Create: `infrastructure/persistence/read/repository/zonas_sql_reader.go`, `infrastructure/persistence/read/adapter/zonas_read_adapter.go`
- Create: `infrastructure/config/module.go`, `infrastructure/config/escritura.go`, `infrastructure/config/zonas.go`
- Create: `web/dto/response/zonas.go`, `web/mapper/zonas_web_mapper.go`, `web/controller/helpers.go`, `web/controller/zonas_handler.go`, `web/controller/routes.go`
- Modify: `internal/bootstrap/app.go`, `internal/bootstrap/router.go`
- Create: `tests/integration/atencion_helpers_test.go`, `tests/integration/atencion_zonas_test.go`

**Interfaces:**
- Consumes: Tarea 1 (`mapper.*`, `policy.*`); `platform/transaction`, `platform/idempotency`, `platform/database`, `platform/web/{handler,middleware,request,response}`; helpers de integración del paso 1 (`rolConPermisos`, `asegurarUsuario`, `crearSedeSQL`, `token`, `call`, `callH`, `post`, `esperar`, `idDe`, `scalar`, `asSystem`, `adminToken`).
- Produces:
  - `result.Zona{ID, SucursalID int64; Nombre string; OrdenVisualizacion int; EsActivo bool; CreadoEn time.Time; CreadoPor int64; ActualizadoEn time.Time; ActualizadoPor int64}`; `result.PaginaZonas{Items []Zona; NextCursor string}`.
  - `query.ListarZonas{Actor; SucursalID *int64; Activo *bool; Limite int; Cursor string}`; `query.ObtenerZona{Actor; ID int64}`.
  - `command.Intencion{Clave, Operacion string; Huella [32]byte}`.
  - `inbound.ListarZonas`, `inbound.ObtenerZona`.
  - `outbound.FiltroZonas{SedeIDs []int64; Activo *bool; DespuesDeID int64; Limite int}`; `outbound.ZonaReader{ListarZonas(ctx, FiltroZonas) ([]Zona, error); ObtenerZona(ctx, id) (Zona, bool, error)}`.
  - `controller.NewZonasHandler(listar, obtener)`; `config.Module` con `New(Deps{DB, Logger})` y `RegisterRoutes(mux, protect)`.
  - Helpers del paquete `controller`: `responder`, `actorDe`, `pathID`, `idCadena`, `intencion`, `leerCuerpoOpcionalVacio`, `consulta`, `limiteDe`, `activoDe`, `sucursalDe`.
  - Helpers de integración: `crearZonaSQL(t, sede, nombre)`, `crearMesaSQL(t, sede, zona, numero, capacidad)`, `zonaResp`, `mesaResp`, `zonasPagina`, `mesasPagina`.

- [ ] **Step 1: DTO, puertos y pruebas unitarias (fallan)**

`application/dto/command/intencion.go`:

```go
// Package command agrupa los DTO de comandos del módulo atencion.
package command

// Intencion identifica una operación para la idempotencia (ADR-04). La rellena la capa web con
// la clave del encabezado, la operación y la huella del contenido; los casos de uso la ignoran.
type Intencion struct {
	Clave     string
	Operacion string // p. ej. "POST /zonas/7/baja" (con el id concreto)
	Huella    [32]byte
}
```

`application/dto/query/zonas.go`:

```go
// Package query agrupa los DTO de consultas del módulo atencion.
package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarZonas pagina por cursor (id ascendente) las zonas de las sedes que el actor puede ver.
// SucursalID y Activo filtran cuando no son nil.
type ListarZonas struct {
	Actor      appctx.Actor
	SucursalID *int64
	Activo     *bool
	Limite     int
	Cursor     string
}

type ObtenerZona struct {
	Actor appctx.Actor
	ID    int64
}
```

`application/dto/result/zonas.go`:

```go
// Package result agrupa los resultados de aplicación del módulo atencion.
package result

import "time"

// Zona es un salón, terraza o nivel del local.
type Zona struct {
	ID                 int64
	SucursalID         int64
	Nombre             string
	OrdenVisualizacion int
	EsActivo           bool
	CreadoEn           time.Time
	CreadoPor          int64
	ActualizadoEn      time.Time
	ActualizadoPor     int64
}

// PaginaZonas: NextCursor vacío indica la última página.
type PaginaZonas struct {
	Items      []Zona
	NextCursor string
}
```

`application/port/inbound/zonas.go`:

```go
// Package inbound declara los casos de uso que ofrece el módulo atencion.
package inbound

import (
	"context"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarZonas interface {
	Execute(context.Context, qry.ListarZonas) (appresult.Result[dtoresult.PaginaZonas], error)
}
type ObtenerZona interface {
	Execute(context.Context, qry.ObtenerZona) (appresult.Result[dtoresult.Zona], error)
}
```

`application/port/outbound/zonas.go`:

```go
package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// FiltroZonas acota el listado a las sedes visibles (SedeIDs) y, opcionalmente, a su estado.
type FiltroZonas struct {
	SedeIDs     []int64
	Activo      *bool
	DespuesDeID int64
	Limite      int
}

// ZonaReader lee zonas sin bloquear filas.
type ZonaReader interface {
	ListarZonas(ctx context.Context, f FiltroZonas) ([]dtoresult.Zona, error)
	ObtenerZona(ctx context.Context, id int64) (dtoresult.Zona, bool, error)
}
```

`application/usecase/query/zonas_test.go`:

```go
package query

import (
	"context"
	"slices"
	"testing"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type zonaReaderFake struct {
	filtro   outbound.FiltroZonas
	llamadas int
	filas    []dtoresult.Zona
	una      dtoresult.Zona
}

func (f *zonaReaderFake) ListarZonas(_ context.Context, fl outbound.FiltroZonas) ([]dtoresult.Zona, error) {
	f.llamadas++
	f.filtro = fl
	return f.filas, nil
}
func (f *zonaReaderFake) ObtenerZona(_ context.Context, id int64) (dtoresult.Zona, bool, error) {
	f.llamadas++
	return f.una, f.una.ID == id, nil
}

func actorMesas(g ...appctx.Grant) appctx.Actor { return appctx.NewActor(7, "ana", 1, g) }

func TestListarZonasAcotaPorLasSedesDelActor(t *testing.T) {
	f := &zonaReaderFake{}
	a := actorMesas(
		appctx.Grant{SucursalID: 4, Permiso: policy.MesaVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.MesaAdministrar})
	res, err := NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, Limite: 10})
	if err != nil || !res.IsOk() {
		t.Fatalf("listar: %v %+v", err, res)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{2, 4}) || f.filtro.Limite != 11 {
		t.Fatalf("filtro: %+v (se pide limite+1)", f.filtro)
	}
	sede := int64(4)
	if _, err := NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, SucursalID: &sede}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{4}) {
		t.Fatalf("un sucursal_id visible restringe a esa sede: %+v", f.filtro)
	}
}

func TestListarZonasRechazos(t *testing.T) {
	f := &zonaReaderFake{filas: []dtoresult.Zona{{ID: 1}, {ID: 2}, {ID: 3}}}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	res, _ := NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, Limite: 2})
	if p := res.Value(); len(p.Items) != 2 || p.NextCursor == "" {
		t.Fatalf("debía recortar a 2 y dar cursor: %+v", p)
	}

	f.llamadas = 0
	sinPermiso := actorMesas(appctx.Grant{SucursalID: 1, Permiso: "NEGOCIO_VER"})
	res, _ = NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: sinPermiso})
	if res.Err() == nil || res.Err().Kind != apperror.KindForbidden || f.llamadas != 0 {
		t.Fatalf("sin permiso debe ser 403 sin tocar el lector: %+v", res.Err())
	}
	ajena := int64(9)
	res, _ = NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, SucursalID: &ajena})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound || f.llamadas != 0 {
		t.Fatalf("una sede ajena debe ser 404 sin tocar el lector: %+v", res.Err())
	}
	res, _ = NewListarZonas(f).Execute(context.Background(), qry.ListarZonas{Actor: a, Cursor: "%%%"})
	if res.Err() == nil || res.Err().Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido debe ser 400: %+v", res.Err())
	}
}

func TestObtenerZonaOcultaLasAjenas(t *testing.T) {
	f := &zonaReaderFake{una: dtoresult.Zona{ID: 9, SucursalID: 3}}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	res, _ := NewObtenerZona(f).Execute(context.Background(), qry.ObtenerZona{Actor: a, ID: 9})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una zona de sede ajena debe ser 404: %+v", res.Err())
	}
	f.una = dtoresult.Zona{ID: 5, SucursalID: 1}
	res, _ = NewObtenerZona(f).Execute(context.Background(), qry.ObtenerZona{Actor: a, ID: 5})
	if !res.IsOk() || res.Value().ID != 5 {
		t.Fatalf("una zona propia debe leerse: %+v", res.Err())
	}
	res, _ = NewObtenerZona(f).Execute(context.Background(), qry.ObtenerZona{Actor: a, ID: 77})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una zona inexistente debe ser 404: %+v", res.Err())
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `go test ./internal/modules/atencion/application/usecase/query/ 2>&1 | head`
Expected: FAIL de compilación (`undefined: NewListarZonas`).

- [ ] **Step 3: Casos de uso de consulta**

`application/usecase/query/zonas.go`:

```go
// Package query implementa las consultas del módulo atencion.
package query

import (
	"context"
	"fmt"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	"service-restaurant/internal/shared/application/pagination"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const mensajeSinPermisoLectura = "no tiene permiso para consultar zonas y mesas"

// tamanio normaliza el límite pedido (por defecto 25, máximo 100).
func tamanio(limite int) int {
	return pagination.PageRequest{Size: limite}.Normalize().Size
}

// ListarZonas devuelve las zonas de las sedes visibles. Pide limite+1 filas para saber si hay
// más páginas.
type ListarZonas struct{ zonas outbound.ZonaReader }

func NewListarZonas(z outbound.ZonaReader) *ListarZonas { return &ListarZonas{z} }

func (uc *ListarZonas) Execute(ctx context.Context, q qry.ListarZonas) (appresult.Result[dtoresult.PaginaZonas], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaZonas](ae), nil
	}
	sedes, ok := sedesPedidas(q.Actor, q.SucursalID)
	if !ok {
		return appresult.Fail[dtoresult.PaginaZonas](mapper.NoEncontrado("sucursal")), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaZonas](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.zonas.ListarZonas(ctx, outbound.FiltroZonas{SedeIDs: sedes, Activo: q.Activo, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaZonas]{}, fmt.Errorf("listar zonas: %w", err)
	}
	pagina := dtoresult.PaginaZonas{Items: items}
	if len(items) > n && n > 0 {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Zona{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerZona lee una zona; la de una sede que el actor no ve responde 404.
type ObtenerZona struct{ zonas outbound.ZonaReader }

func NewObtenerZona(z outbound.ZonaReader) *ObtenerZona { return &ObtenerZona{z} }

func (uc *ObtenerZona) Execute(ctx context.Context, q qry.ObtenerZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Zona](ae), nil
	}
	z, found, err := uc.zonas.ObtenerZona(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Zona]{}, fmt.Errorf("obtener zona: %w", err)
	}
	if !found || !mapper.PuedeVer(q.Actor, z.SucursalID) {
		return appresult.Fail[dtoresult.Zona](mapper.NoEncontrado("zona")), nil
	}
	return appresult.Ok(z), nil
}
```

Y el helper compartido por zonas y mesas, en `application/usecase/query/sedes.go`:

```go
package query

import (
	"slices"

	"service-restaurant/internal/modules/atencion/application/mapper"
	appctx "service-restaurant/internal/shared/application/context"
)

// sedesPedidas devuelve las sedes que alcanza un listado: las visibles del actor o, si pidió una
// sucursal concreta, solo esa. ok=false si pidió una sucursal que no ve.
func sedesPedidas(a appctx.Actor, sucursalID *int64) ([]int64, bool) {
	visibles := mapper.SedesVisibles(a)
	if sucursalID == nil {
		return visibles, true
	}
	if !slices.Contains(visibles, *sucursalID) {
		return nil, false
	}
	return []int64{*sucursalID}, true
}
```

- [ ] **Step 4: Ejecutar las pruebas unitarias**

Run: `gofmt -l internal/ ; go test ./internal/modules/atencion/... 2>&1 | tail`
Expected: PASS.

- [ ] **Step 5: Persistencia de lectura**

`infrastructure/persistence/read/repository/zonas_sql_reader.go`:

```go
// Package repository contiene el SQL parametrizado de lectura del módulo atencion.
package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

// ZonaSQLReader lee zonas.
type ZonaSQLReader struct{ db database.DBTX }

func NewZonaSQLReader(db database.DBTX) *ZonaSQLReader { return &ZonaSQLReader{db} }

const zonaCols = `id, sucursal_id, nombre, orden_visualizacion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

type fila interface{ Scan(...any) error }

func leerZona(s fila) (dtoresult.Zona, error) {
	var z dtoresult.Zona
	err := s.Scan(&z.ID, &z.SucursalID, &z.Nombre, &z.OrdenVisualizacion, &z.EsActivo,
		&z.CreadoEn, &z.CreadoPor, &z.ActualizadoEn, &z.ActualizadoPor)
	return z, err
}

// ListZonas devuelve las zonas de f.SedeIDs ordenadas por id (cursor estable).
func (r *ZonaSQLReader) ListZonas(ctx context.Context, f outbound.FiltroZonas) ([]dtoresult.Zona, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+zonaCols+` FROM sch_restaurante.zonas
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2 AND ($3::boolean IS NULL OR es_activo = $3)
		ORDER BY id LIMIT $4`, f.SedeIDs, f.DespuesDeID, activo, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Zona
	for rows.Next() {
		z, err := leerZona(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

func (r *ZonaSQLReader) GetZona(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	z, err := leerZona(r.db.QueryRowContext(ctx, `SELECT `+zonaCols+` FROM sch_restaurante.zonas WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Zona{}, false, nil
	}
	return z, err == nil, err
}
```

`infrastructure/persistence/read/adapter/zonas_read_adapter.go`:

```go
// Package adapter implementa los puertos de salida de lectura del módulo atencion.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// ZonaReadAdapter implementa outbound.ZonaReader.
type ZonaReadAdapter struct{ repo *repository.ZonaSQLReader }

func NewZonaReadAdapter(db database.DBTX) *ZonaReadAdapter {
	return &ZonaReadAdapter{repo: repository.NewZonaSQLReader(db)}
}

var _ outbound.ZonaReader = (*ZonaReadAdapter)(nil)

func (a *ZonaReadAdapter) ListarZonas(ctx context.Context, f outbound.FiltroZonas) ([]dtoresult.Zona, error) {
	return a.repo.ListZonas(ctx, f)
}
func (a *ZonaReadAdapter) ObtenerZona(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	return a.repo.GetZona(ctx, id)
}
```

- [ ] **Step 6: Capa web**

`web/dto/response/zonas.go`:

```go
// Package response define las respuestas HTTP del módulo atencion; los ids viajan como cadenas.
package response

import "time"

type Zona struct {
	ID                 string    `json:"id"`
	SucursalID         string    `json:"sucursal_id"`
	Nombre             string    `json:"nombre"`
	OrdenVisualizacion int       `json:"orden_visualizacion"`
	EsActivo           bool      `json:"es_activo"`
	CreadoEn           time.Time `json:"creado_en"`
	CreadoPor          string    `json:"creado_por"`
	ActualizadoEn      time.Time `json:"actualizado_en"`
	ActualizadoPor     string    `json:"actualizado_por"`
}

type ZonasPagina struct {
	Items      []Zona  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
```

`web/mapper/zonas_web_mapper.go`:

```go
// Package mapper convierte resultados de aplicación en respuestas HTTP.
package mapper

import (
	"strconv"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/web/dto/response"
)

func id(n int64) string { return strconv.FormatInt(n, 10) }

func cursor(c string) *string {
	if c == "" {
		return nil
	}
	return &c
}

func ToZona(z dtoresult.Zona) response.Zona {
	return response.Zona{
		ID: id(z.ID), SucursalID: id(z.SucursalID), Nombre: z.Nombre, OrdenVisualizacion: z.OrdenVisualizacion,
		EsActivo: z.EsActivo, CreadoEn: z.CreadoEn, CreadoPor: id(z.CreadoPor),
		ActualizadoEn: z.ActualizadoEn, ActualizadoPor: id(z.ActualizadoPor),
	}
}

func ToZonasPagina(p dtoresult.PaginaZonas) response.ZonasPagina {
	items := make([]response.Zona, len(p.Items))
	for i, z := range p.Items {
		items[i] = ToZona(z)
	}
	return response.ZonasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
```

`web/controller/helpers.go`:

```go
// Package controller expone el módulo atencion por HTTP.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	"service-restaurant/internal/modules/atencion/application/mapper"
	"service-restaurant/internal/platform/idempotency"
	"service-restaurant/internal/platform/web/middleware"
	webrequest "service-restaurant/internal/platform/web/request"
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

// pathID lee el id numérico positivo de la ruta.
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.New(apperror.KindBadRequest, "id_invalido", "el id de la ruta no es válido")
	}
	return id, nil
}

// idCadena convierte un id en cadena (contrato) a int64 positivo.
func idCadena(valor, campo string) (int64, error) {
	n, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || n <= 0 {
		return 0, apperror.New(apperror.KindBadRequest, campo+"_invalido", campo+" debe ser un entero positivo en cadena")
	}
	return n, nil
}

// intencion arma la intención idempotente: exige Idempotency-Key (UUID) y calcula la huella
// sobre la operación y el cuerpo canónico (el DTO tipado vuelto a serializar).
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

// leerCuerpoOpcionalVacio acepta ausencia de cuerpo o "{}" (restauraciones y bajas simples).
func leerCuerpoOpcionalVacio(r *http.Request, dst any) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 {
		return nil
	}
	return webrequest.DecodeJSON(r, dst)
}

// consulta analiza la query string de forma estricta: un cursor corrupto no puede tratarse
// como "primera página".
func consulta(r *http.Request) (url.Values, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, apperror.New(apperror.KindBadRequest, "query_invalida", "los parámetros de la consulta están mal codificados")
	}
	return q, nil
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

// activoDe lee el filtro opcional es_activo ("true" o "false").
func activoDe(q url.Values) (*bool, error) {
	switch q.Get("es_activo") {
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

// sucursalDe lee el filtro opcional sucursal_id.
func sucursalDe(q url.Values) (*int64, error) {
	v := q.Get("sucursal_id")
	if v == "" {
		return nil, nil
	}
	n, err := idCadena(v, "sucursal_id")
	if err != nil {
		return nil, err
	}
	return &n, nil
}
```

`web/controller/zonas_handler.go`:

```go
package controller

import (
	"net/http"

	"service-restaurant/internal/modules/atencion/application/dto/query"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	webmapper "service-restaurant/internal/modules/atencion/web/mapper"
)

// ZonasHandler: salones, terrazas y niveles del local (RES-RF-008).
type ZonasHandler struct {
	listar  inbound.ListarZonas
	obtener inbound.ObtenerZona
}

func NewZonasHandler(l inbound.ListarZonas, o inbound.ObtenerZona) *ZonasHandler {
	return &ZonasHandler{l, o}
}

// Listar: GET /zonas?sucursal_id=&es_activo=&limit=&cursor=
func (h *ZonasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarZonas{
		Actor: actor, SucursalID: sede, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToZonasPagina)
}

// Obtener: GET /zonas/{id}
func (h *ZonasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerZona{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
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

// RegisterRoutes monta zonas y mesas (todas exigen autenticación; el permiso fino lo verifica
// cada caso de uso).
func RegisterRoutes(mux *http.ServeMux, zonas *ZonasHandler, protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /zonas", zonas.Listar)
	ruta("GET /zonas/{id}", zonas.Obtener)
}
```

- [ ] **Step 7: Composición y bootstrap**

`infrastructure/config/escritura.go`:

```go
// Package config compone el módulo atencion: conecta los puertos de aplicación con sus
// adaptadores concretos y decora los casos de uso con transacciones.
package config

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
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
// autenticado fijado (app.usuario_id) y, si la solicitud trae clave de idempotencia, registra
// la clave en esa misma transacción. Solo un Result exitoso se confirma; un rechazo o un error
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

`infrastructure/config/zonas.go`:

```go
package config

import (
	"database/sql"

	uquery "service-restaurant/internal/modules/atencion/application/usecase/query"
	readadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter"
	"service-restaurant/internal/modules/atencion/web/controller"
)

// armarZonas compone los casos de uso de zonas.
func armarZonas(db *sql.DB) *controller.ZonasHandler {
	lector := readadapter.NewZonaReadAdapter(db)
	return controller.NewZonasHandler(uquery.NewListarZonas(lector), uquery.NewObtenerZona(lector))
}
```

`infrastructure/config/module.go`:

```go
package config

import (
	"database/sql"
	"log/slog"
	"net/http"

	"service-restaurant/internal/modules/atencion/web/controller"
)

// Deps son las dependencias técnicas del módulo.
type Deps struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// Module es el módulo atencion ya ensamblado.
type Module struct {
	deps  Deps
	zonas *controller.ZonasHandler
}

// New ensambla el módulo.
func New(d Deps) (*Module, error) {
	return &Module{deps: d, zonas: armarZonas(d.DB)}, nil
}

// RegisterRoutes monta las rutas del módulo sobre mux (relativas a /api/v1). protect exige un
// actor autenticado; lo aporta el módulo identidad a través del bootstrap.
func (m *Module) RegisterRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	controller.RegisterRoutes(mux, m.zonas, protect, m.deps.Logger)
}
```

En `internal/bootstrap/app.go`: importar `atencion "service-restaurant/internal/modules/atencion/infrastructure/config"`; añadir a `App` el campo `atencion *atencion.Module`; en `New`, después de crear `mod` (identidad) y antes de `a := &App{...}`:

```go
	atn, err := atencion.New(atencion.Deps{DB: db, Logger: logger})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
```

y `a := &App{cfg: cfg, logger: logger, db: db, identidad: mod, atencion: atn}`.

En `internal/bootstrap/router.go`, justo después de `a.identidad.RegisterRoutes(api, protect)`:

```go
	a.atencion.RegisterRoutes(api, protect)
```

- [ ] **Step 8: Compilar y comprobar arquitectura**

Run: `go build ./... && go vet ./... && go test ./tests/architecture/ ./internal/...`
Expected: sin errores.

- [ ] **Step 9: Pruebas de integración de lectura de zonas**

`tests/integration/atencion_helpers_test.go`:

```go
package integration

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"testing"
)

type zonaResp struct {
	ID         string `json:"id"`
	SucursalID string `json:"sucursal_id"`
	Nombre     string `json:"nombre"`
	Orden      int    `json:"orden_visualizacion"`
	EsActivo   bool   `json:"es_activo"`
	CreadoPor  string `json:"creado_por"`
}

type zonasPagina struct {
	Items      []zonaResp `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

func (p zonasPagina) nombres() []string {
	out := make([]string, 0, len(p.Items))
	for _, z := range p.Items {
		out = append(out, z.Nombre)
	}
	sort.Strings(out)
	return out
}

type mesaResp struct {
	ID           string  `json:"id"`
	SucursalID   string  `json:"sucursal_id"`
	ZonaID       string  `json:"zona_id"`
	NumeroMesa   string  `json:"numero_mesa"`
	Capacidad    int     `json:"capacidad"`
	Habilitada   bool    `json:"habilitada"`
	PosicionX    *string `json:"posicion_x"`
	PosicionY    *string `json:"posicion_y"`
	EsActivo     bool    `json:"es_activo"`
	Motivo       *string `json:"motivo_eliminacion"`
	EliminadoPor *string `json:"eliminado_por"`
	CreadoPor    string  `json:"creado_por"`
}

type mesasPagina struct {
	Items      []mesaResp `json:"items"`
	NextCursor *string    `json:"nextCursor"`
}

func (p mesasPagina) numeros() []string {
	out := make([]string, 0, len(p.Items))
	for _, m := range p.Items {
		out = append(out, m.NumeroMesa)
	}
	sort.Strings(out)
	return out
}

// crearZonaSQL inserta una zona activa en la sede (por código) si no existe.
func crearZonaSQL(t *testing.T, sede, nombre string) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.zonas(sucursal_id, nombre)
		SELECT s.id, '`+nombre+`' FROM sch_restaurante.sucursales s WHERE s.codigo='`+sede+`'
		AND NOT EXISTS (SELECT 1 FROM sch_restaurante.zonas z WHERE z.sucursal_id=s.id AND z.nombre='`+nombre+`')`)
}

// crearMesaSQL inserta una mesa activa en la zona (por nombre y sede) si no existe.
func crearMesaSQL(t *testing.T, sede, zona, numero string, capacidad int) {
	t.Helper()
	app(t)
	asSystem(t, `INSERT INTO sch_restaurante.mesas(sucursal_id, zona_id, numero_mesa, capacidad)
		SELECT z.sucursal_id, z.id, '`+numero+`', `+strconv.Itoa(capacidad)+` FROM sch_restaurante.zonas z
		JOIN sch_restaurante.sucursales s ON s.id=z.sucursal_id
		WHERE s.codigo='`+sede+`' AND z.nombre='`+zona+`'
		AND NOT EXISTS (SELECT 1 FROM sch_restaurante.mesas m WHERE m.sucursal_id=z.sucursal_id AND m.numero_mesa='`+numero+`')`)
}

// gestorMesas devuelve el token de un usuario con MESA_VER y MESA_ADMINISTRAR en las sedes dadas
// (por código). Es idempotente: varias pruebas lo reutilizan.
func gestorMesas(t *testing.T, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, "T_MESA_GESTOR", "MESA_VER", "MESA_ADMINISTRAR")
	asegurarUsuario(t, "mesa.gestor", "MesaGestor2026Clave", "T_MESA_GESTOR", append([]string{"PRINCIPAL"}, sedes...)...)
	return token(t, "mesa.gestor", "MesaGestor2026Clave")
}

// lectorMesas devuelve el token de un usuario que solo tiene MESA_VER en las sedes dadas.
func lectorMesas(t *testing.T, sedes ...string) string {
	t.Helper()
	rolConPermisos(t, "T_MESA_LECTOR", "MESA_VER")
	asegurarUsuario(t, "mesa.lector", "MesaLector2026Clave", "T_MESA_LECTOR", append([]string{"PRINCIPAL"}, sedes...)...)
	return token(t, "mesa.lector", "MesaLector2026Clave")
}

func listarZonas(t *testing.T, query, tk string) zonasPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/zonas"+query, "", tk)
	esperar(t, r, http.StatusOK, "")
	var p zonasPagina
	r.json(t, &p)
	return p
}

func listarMesas(t *testing.T, query, tk string) mesasPagina {
	t.Helper()
	r := call(t, "GET", "/api/v1/mesas"+query, "", tk)
	esperar(t, r, http.StatusOK, "")
	var p mesasPagina
	r.json(t, &p)
	return p
}
```

`tests/integration/atencion_zonas_test.go`:

```go
package integration

import (
	"net/http"
	"slices"
	"testing"
)

func TestZonasLecturaRespetaLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Salon L1")
	crearZonaSQL(t, "MESA-AJENA", "Salon ajeno")
	tk := lectorMesas(t) // solo PRINCIPAL

	nombres := listarZonas(t, "", tk).nombres()
	if !slices.Contains(nombres, "Salon L1") || slices.Contains(nombres, "Salon ajeno") {
		t.Fatalf("zonas visibles = %v: debe incluir la de PRINCIPAL y no la de MESA-AJENA", nombres)
	}
	ajena := idDe(t, "sucursales", "codigo", "MESA-AJENA")
	esperar(t, call(t, "GET", "/api/v1/zonas?sucursal_id="+ajena, "", tk), http.StatusNotFound, "no_encontrado")

	propia := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	if got := listarZonas(t, "?sucursal_id="+propia, tk).nombres(); !slices.Contains(got, "Salon L1") {
		t.Fatalf("sucursal_id propio: %v", got)
	}

	r := call(t, "GET", "/api/v1/zonas", "", tk)
	var p zonasPagina
	r.json(t, &p)
	var z zonaResp
	for _, x := range p.Items {
		if x.Nombre == "Salon L1" {
			z = x
		}
	}
	r = call(t, "GET", "/api/v1/zonas/"+z.ID, "", tk)
	esperar(t, r, http.StatusOK, "")
	var leida zonaResp
	r.json(t, &leida)
	if leida.ID != z.ID || leida.SucursalID != propia || !leida.EsActivo || leida.CreadoPor == "" {
		t.Fatalf("zona: %+v", leida)
	}
	ajenaZona := scalarStr(t, `SELECT z.id FROM sch_restaurante.zonas z WHERE z.nombre='Salon ajeno'`)
	esperar(t, call(t, "GET", "/api/v1/zonas/"+ajenaZona, "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/zonas/999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/zonas/abc", "", tk), http.StatusBadRequest, "id_invalido")
}

func TestZonasExigeAlgunPermisoDeMesa(t *testing.T) {
	rolConPermisos(t, "T_SIN_MESA", "NEGOCIO_VER")
	asegurarUsuario(t, "mesa.sin", "MesaSin2026Clave", "T_SIN_MESA", "PRINCIPAL")
	tk := token(t, "mesa.sin", "MesaSin2026Clave")
	esperar(t, call(t, "GET", "/api/v1/zonas", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/zonas/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/zonas", "", ""), http.StatusUnauthorized, "no_autenticado")
	// MESERO y CAJERO reciben MESA_VER con la migración 00008.
	asegurarUsuario(t, "mesa.cajero", "MesaCajero2026Clave", "CAJERO", "PRINCIPAL")
	esperar(t, call(t, "GET", "/api/v1/zonas", "", token(t, "mesa.cajero", "MesaCajero2026Clave")), http.StatusOK, "")
}

func TestZonasFiltroYPaginacion(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Pag A")
	crearZonaSQL(t, "PRINCIPAL", "Pag B")
	crearZonaSQL(t, "PRINCIPAL", "Pag Inactiva")
	asSystem(t, `UPDATE sch_restaurante.zonas SET es_activo=false WHERE nombre='Pag Inactiva'`)
	tk := lectorMesas(t)

	activas := listarZonas(t, "?es_activo=true", tk).nombres()
	if slices.Contains(activas, "Pag Inactiva") || !slices.Contains(activas, "Pag A") {
		t.Fatalf("es_activo=true: %v", activas)
	}
	inactivas := listarZonas(t, "?es_activo=false", tk).nombres()
	if !slices.Contains(inactivas, "Pag Inactiva") || slices.Contains(inactivas, "Pag A") {
		t.Fatalf("es_activo=false: %v", inactivas)
	}
	esperar(t, call(t, "GET", "/api/v1/zonas?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/zonas?limit=0", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/zonas?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")
	esperar(t, call(t, "GET", "/api/v1/zonas?sucursal_id=x", "", tk), http.StatusBadRequest, "sucursal_id_invalido")

	p1 := listarZonas(t, "?limit=1", tk)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarZonas(t, "?limit=1&cursor="+*p1.NextCursor, tk)
	if len(p2.Items) != 1 || p2.Items[0].ID == p1.Items[0].ID {
		t.Fatalf("segunda página: %+v", p2)
	}
}
```

Añadir también a `atencion_helpers_test.go` el helper `scalarStr` (devuelve texto; `scalar` solo devuelve `int`; ya usa los imports `context` y `fmt` del bloque anterior):

```go
// scalarStr devuelve el primer valor de una consulta como texto (para ids).
func scalarStr(t *testing.T, query string, args ...any) string {
	t.Helper()
	var v any
	if err := testDB.QueryRowContext(context.Background(), query, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return fmt.Sprint(v)
}
```

- [ ] **Step 10: Ejecutar integración**

Run: `go test -count=1 ./tests/integration/ -run "TestZonas" -v`
Expected: PASS (3 pruebas). Luego `go test -count=1 -shuffle=on ./tests/integration/ -run "TestZonas|TestSucursal|TestNegocio"`: PASS.

- [ ] **Step 11: Regresión y commit**

Run: `gofmt -l . ; go vet ./... && go test ./... && go test -count=1 ./tests/integration/...`
Expected: todo en verde (pruebas de I1 y del paso 1 incluidas).

```bash
git add internal/modules/atencion internal/bootstrap tests/integration
git commit -m "I2 paso 2: módulo atencion en marcha con GET /zonas y /zonas/{id}

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Zonas — crear, editar, baja y restauración

**Files:**
- Create: `application/dto/command/zonas.go`
- Modify: `application/port/inbound/zonas.go`, `application/port/outbound/zonas.go`
- Create: `application/usecase/command/helpers.go`, `application/usecase/command/zonas.go`, `application/usecase/command/zonas_test.go`
- Create: `infrastructure/persistence/write/repository/sql.go`, `.../write/repository/zonas_sql.go`, `.../write/adapter/zonas.go`
- Create: `web/dto/request/opcional.go`, `web/dto/request/zonas.go`
- Modify: `web/controller/zonas_handler.go` (reemplazo completo), `web/controller/routes.go`, `infrastructure/config/zonas.go`
- Modify: `tests/integration/atencion_zonas_test.go`

**Interfaces:**
- Consumes: Tareas 1–2; `escritura[...]`, `command.Intencion`, `intencion()`, `leerCuerpoOpcionalVacio`, `idCadena`.
- Produces:
  - `command.CrearZona{Intencion; Actor; SucursalID int64; Nombre string; OrdenVisualizacion *int}`
  - `command.EditarZona{Actor; ID int64; Nombre *string; OrdenVisualizacion *int}`
  - `command.CambiarEstadoZona{Intencion; Actor; ID int64; Activo bool}`
  - `inbound.CrearZona`, `inbound.EditarZona`, `inbound.CambiarEstadoZona`
  - `outbound.NuevaZona{SucursalID int64; Nombre string; Orden int}`, `outbound.CambiosZona{Nombre *string; Orden *int}`, `outbound.ZonaStore`, `outbound.ErrNombreZonaDuplicado`
  - `controller.NewZonasHandler(listar, obtener, crear, editar, estado)`

- [ ] **Step 1: DTO, puertos y pruebas unitarias (fallan)**

`application/dto/command/zonas.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// CrearZona da de alta una zona en una sede. OrdenVisualizacion nil = 0.
type CrearZona struct {
	Intencion
	Actor              appctx.Actor
	SucursalID         int64
	Nombre             string
	OrdenVisualizacion *int
}

// EditarZona aplica solo los campos presentes. La sede de la zona no se edita.
type EditarZona struct {
	Actor              appctx.Actor
	ID                 int64
	Nombre             *string
	OrdenVisualizacion *int
}

// CambiarEstadoZona desactiva (Activo=false, sin motivo) o restaura (Activo=true).
type CambiarEstadoZona struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
}
```

Añadir a `application/port/inbound/zonas.go` (importar además `cmd "service-restaurant/internal/modules/atencion/application/dto/command"`):

```go
type CrearZona interface {
	Execute(context.Context, cmd.CrearZona) (appresult.Result[dtoresult.Zona], error)
}
type EditarZona interface {
	Execute(context.Context, cmd.EditarZona) (appresult.Result[dtoresult.Zona], error)
}
type CambiarEstadoZona interface {
	Execute(context.Context, cmd.CambiarEstadoZona) (appresult.Result[dtoresult.Zona], error)
}
```

Añadir a `application/port/outbound/zonas.go` (importar además `"fmt"`):

```go
// ErrNombreZonaDuplicado distingue la colisión del nombre de zona dentro de la sede.
var ErrNombreZonaDuplicado = fmt.Errorf("nombre de zona duplicado: %w", ErrDuplicado)

type NuevaZona struct {
	SucursalID int64
	Nombre     string
	Orden      int
}

type CambiosZona struct {
	Nombre *string
	Orden  *int
}

// ZonaStore escribe zonas dentro de la transacción del comando.
type ZonaStore interface {
	Crear(ctx context.Context, n NuevaZona) (dtoresult.Zona, error)
	// Obtener lee la zona bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Zona, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosZona) (dtoresult.Zona, bool, error)
	ContarMesasActivas(ctx context.Context, zonaID int64) (int, error)
	CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Zona, bool, error)
}
```

`application/usecase/command/zonas_test.go`:

```go
package command

import (
	"context"
	"errors"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type zonaStoreFake struct {
	existente  dtoresult.Zona
	mesas      int
	errCrear   error
	creada     outbound.NuevaZona
	cambios    outbound.CambiosZona
	cambiosHechos int
}

func (f *zonaStoreFake) Crear(_ context.Context, n outbound.NuevaZona) (dtoresult.Zona, error) {
	f.creada = n
	return dtoresult.Zona{ID: 3, SucursalID: n.SucursalID, Nombre: n.Nombre, EsActivo: true}, f.errCrear
}
func (f *zonaStoreFake) Obtener(_ context.Context, id int64) (dtoresult.Zona, bool, error) {
	return f.existente, f.existente.ID == id, nil
}
func (f *zonaStoreFake) Actualizar(_ context.Context, _ int64, c outbound.CambiosZona) (dtoresult.Zona, bool, error) {
	f.cambios = c
	return f.existente, true, nil
}
func (f *zonaStoreFake) ContarMesasActivas(context.Context, int64) (int, error) { return f.mesas, nil }
func (f *zonaStoreFake) CambiarActivo(_ context.Context, _ int64, activo bool) (dtoresult.Zona, bool, error) {
	f.cambiosHechos++
	z := f.existente
	z.EsActivo = activo
	return z, true, nil
}

func gestorDe(sedes ...int64) appctx.Actor {
	var g []appctx.Grant
	for _, s := range sedes {
		g = append(g, appctx.Grant{SucursalID: s, Permiso: policy.MesaAdministrar})
	}
	return appctx.NewActor(7, "ana", 1, g)
}

func soloVer(sede int64) appctx.Actor {
	return appctx.NewActor(7, "ana", 1, []appctx.Grant{{SucursalID: sede, Permiso: policy.MesaVer}})
}

func esperarRechazo(t *testing.T, ae *apperror.AppError, kind apperror.Kind, code string) {
	t.Helper()
	if ae == nil || ae.Kind != kind || ae.Code != code {
		t.Fatalf("rechazo %v, quería %s/%s", ae, kind, code)
	}
}

func entero(n int) *int       { return &n }
func texto(s string) *string { return &s }

func TestCrearZona(t *testing.T) {
	ok := cmd.CrearZona{Actor: gestorDe(1), SucursalID: 1, Nombre: "  Terraza  ", OrdenVisualizacion: entero(2)}
	f := &zonaStoreFake{}
	res, err := NewCrearZona(f).Execute(context.Background(), ok)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía crearse: %v %+v", err, res.Err())
	}
	if f.creada.Nombre != "Terraza" || f.creada.Orden != 2 || f.creada.SucursalID != 1 {
		t.Fatalf("no normalizó: %+v", f.creada)
	}
	_, _ = NewCrearZona(f).Execute(context.Background(), cmd.CrearZona{Actor: gestorDe(1), SucursalID: 1, Nombre: "Sin orden"})
	if f.creada.Orden != 0 {
		t.Errorf("orden por defecto = %d, quería 0", f.creada.Orden)
	}

	casos := []struct {
		nombre string
		mod    func(c *cmd.CrearZona)
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(c *cmd.CrearZona) { c.Actor = appctx.Actor{} }, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(c *cmd.CrearZona) { c.Actor = soloVer(1) }, apperror.KindForbidden, "sin_permiso"},
		{"sede ajena", func(c *cmd.CrearZona) { c.SucursalID = 9 }, apperror.KindNotFound, "no_encontrado"},
		{"nombre en blanco", func(c *cmd.CrearZona) { c.Nombre = "  " }, apperror.KindValidation, "nombre_invalido"},
		{"orden fuera de rango", func(c *cmd.CrearZona) { c.OrdenVisualizacion = entero(10000) }, apperror.KindValidation, "orden_invalido"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			in := ok
			c.mod(&in)
			res, err := NewCrearZona(&zonaStoreFake{}).Execute(context.Background(), in)
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
		})
	}
	t.Run("nombre duplicado", func(t *testing.T) {
		res, _ := NewCrearZona(&zonaStoreFake{errCrear: outbound.ErrNombreZonaDuplicado}).Execute(context.Background(), ok)
		esperarRechazo(t, res.Err(), apperror.KindConflict, "nombre_duplicado")
	})
	t.Run("error técnico", func(t *testing.T) {
		if _, err := NewCrearZona(&zonaStoreFake{errCrear: errors.New("conexión perdida")}).Execute(context.Background(), ok); err == nil {
			t.Fatal("un error técnico debe viajar por el canal error")
		}
	})
}

func TestEditarZona(t *testing.T) {
	activa := dtoresult.Zona{ID: 5, SucursalID: 1, EsActivo: true}
	edit := func(a appctx.Actor) cmd.EditarZona { return cmd.EditarZona{Actor: a, ID: 5, Nombre: texto("  Nueva  ")} }

	res, _ := NewEditarZona(&zonaStoreFake{existente: activa}).Execute(context.Background(), edit(gestorDe(9)))
	esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado") // sede 1 no es del actor
	res, _ = NewEditarZona(&zonaStoreFake{existente: activa}).Execute(context.Background(),
		cmd.EditarZona{Actor: appctx.NewActor(7, "a", 1, []appctx.Grant{
			{SucursalID: 1, Permiso: policy.MesaVer}, {SucursalID: 9, Permiso: policy.MesaAdministrar}}), ID: 5, Nombre: texto("x")})
	esperarRechazo(t, res.Err(), apperror.KindForbidden, "sede_no_autorizada")
	res, _ = NewEditarZona(&zonaStoreFake{existente: dtoresult.Zona{ID: 5, SucursalID: 1}}).Execute(context.Background(), edit(gestorDe(1)))
	esperarRechazo(t, res.Err(), apperror.KindConflict, "zona_inactiva")

	f := &zonaStoreFake{existente: activa}
	res, err := NewEditarZona(f).Execute(context.Background(), cmd.EditarZona{Actor: gestorDe(1), ID: 5, Nombre: texto("  Nueva  "), OrdenVisualizacion: entero(4)})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	if *f.cambios.Nombre != "Nueva" || *f.cambios.Orden != 4 {
		t.Fatalf("cambios: %+v", f.cambios)
	}
	res, _ = NewEditarZona(f).Execute(context.Background(), cmd.EditarZona{Actor: gestorDe(1), ID: 5})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "sin_cambios")
	res, _ = NewEditarZona(f).Execute(context.Background(), cmd.EditarZona{Actor: gestorDe(1), ID: 5, Nombre: texto(" ")})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "nombre_invalido")
}

func TestCambiarEstadoZona(t *testing.T) {
	baja := func(a appctx.Actor) cmd.CambiarEstadoZona { return cmd.CambiarEstadoZona{Actor: a, ID: 5, Activo: false} }
	activa := dtoresult.Zona{ID: 5, SucursalID: 1, EsActivo: true}

	t.Run("con mesas activas no se da de baja", func(t *testing.T) {
		f := &zonaStoreFake{existente: activa, mesas: 2}
		res, _ := NewCambiarEstadoZona(f).Execute(context.Background(), baja(gestorDe(1)))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "zona_con_mesas")
		if f.cambiosHechos != 0 {
			t.Fatal("no debe cambiar el estado")
		}
	})
	t.Run("baja sin mesas", func(t *testing.T) {
		f := &zonaStoreFake{existente: activa}
		res, err := NewCambiarEstadoZona(f).Execute(context.Background(), baja(gestorDe(1)))
		if err != nil || !res.IsOk() || res.Value().EsActivo || f.cambiosHechos != 1 {
			t.Fatalf("baja: %v %+v", err, res.Err())
		}
	})
	t.Run("ya estaba de baja", func(t *testing.T) {
		f := &zonaStoreFake{existente: dtoresult.Zona{ID: 5, SucursalID: 1}}
		res, _ := NewCambiarEstadoZona(f).Execute(context.Background(), baja(gestorDe(1)))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_dada_de_baja")
	})
	t.Run("restaurar", func(t *testing.T) {
		f := &zonaStoreFake{existente: dtoresult.Zona{ID: 5, SucursalID: 1}}
		res, err := NewCambiarEstadoZona(f).Execute(context.Background(), cmd.CambiarEstadoZona{Actor: gestorDe(1), ID: 5, Activo: true})
		if err != nil || !res.IsOk() || !res.Value().EsActivo {
			t.Fatalf("restaurar: %v %+v", err, res.Err())
		}
		f = &zonaStoreFake{existente: activa}
		res, _ = NewCambiarEstadoZona(f).Execute(context.Background(), cmd.CambiarEstadoZona{Actor: gestorDe(1), ID: 5, Activo: true})
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_activa")
	})
	t.Run("sede ajena", func(t *testing.T) {
		res, _ := NewCambiarEstadoZona(&zonaStoreFake{existente: activa}).Execute(context.Background(), baja(gestorDe(9)))
		esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	})
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `go test ./internal/modules/atencion/application/usecase/command/ 2>&1 | head`
Expected: FAIL de compilación (`undefined: NewCrearZona`).

- [ ] **Step 3: Casos de uso**

`application/usecase/command/helpers.go`:

```go
// Package command implementa los comandos del módulo atencion.
package command

import (
	"fmt"

	"service-restaurant/internal/modules/atencion/application/mapper"
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

`application/usecase/command/zonas.go`:

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
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxNombreZona          = 80
	mensajeSinPermisoAdmin = "no tiene permiso para administrar zonas y mesas"

	CodigoNombreDuplicado = "nombre_duplicado"
	CodigoZonaInactiva    = "zona_inactiva"
	CodigoZonaConMesas    = "zona_con_mesas"
	CodigoYaDadaDeBaja    = "ya_dada_de_baja"
	CodigoYaActiva        = "ya_activa"
)

// CrearZona da de alta una zona en una sede donde el actor administra mesas (RES-RF-008).
type CrearZona struct{ zonas outbound.ZonaStore }

func NewCrearZona(z outbound.ZonaStore) *CrearZona { return &CrearZona{z} }

func (uc *CrearZona) Execute(ctx context.Context, c cmd.CrearZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	if ae := mapper.AutorizarSede(c.Actor, c.SucursalID); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	nombre, err := vo.TextoObligatorio(c.Nombre, maxNombreZona)
	if err != nil {
		return rechazar[dtoresult.Zona](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 80 caracteres)"))
	}
	orden := 0
	if c.OrdenVisualizacion != nil {
		o, err := vo.NewOrden(*c.OrdenVisualizacion)
		if err != nil {
			return rechazar[dtoresult.Zona](mapper.Validacion("orden_invalido", "el orden de visualización debe estar entre 0 y 9999"))
		}
		orden = o.Int()
	}
	z, err := uc.zonas.Crear(ctx, outbound.NuevaZona{SucursalID: c.SucursalID, Nombre: nombre, Orden: orden})
	if err != nil {
		return deZona(err, "crear zona")
	}
	return appresult.Ok(z), nil
}

// deZona traduce el nombre duplicado a un rechazo específico y delega lo demás.
func deZona(err error, contexto string) (appresult.Result[dtoresult.Zona], error) {
	if errors.Is(err, outbound.ErrNombreZonaDuplicado) {
		return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoNombreDuplicado, "ya existe una zona con ese nombre en la sucursal"))
	}
	return deAlmacen[dtoresult.Zona](contexto, err)
}

// zonaGestionable carga la zona (bloqueándola) y verifica que el actor ve su sede (si no, 404) y
// que la administra (si no, 403). Devuelve la zona o el rechazo; un error técnico va en err.
func zonaGestionable(ctx context.Context, store outbound.ZonaStore, actor appctx.Actor, id int64) (dtoresult.Zona, *apperror.AppError, error) {
	z, found, err := store.Obtener(ctx, id)
	if err != nil {
		return dtoresult.Zona{}, nil, err
	}
	if !found {
		return dtoresult.Zona{}, mapper.NoEncontrado("zona"), nil
	}
	if ae := mapper.AutorizarSede(actor, z.SucursalID); ae != nil {
		return dtoresult.Zona{}, ae, nil
	}
	return z, nil, nil
}
```

Continuar el archivo con:

```go
// EditarZona modifica solo los campos enviados de una zona activa.
type EditarZona struct{ zonas outbound.ZonaStore }

func NewEditarZona(z outbound.ZonaStore) *EditarZona { return &EditarZona{z} }

func (uc *EditarZona) Execute(ctx context.Context, c cmd.EditarZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	actual, ae, err := zonaGestionable(ctx, uc.zonas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Zona]("buscar zona", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoZonaInactiva, "la zona está dada de baja: restáurela para editarla"))
	}
	if c.Nombre == nil && c.OrdenVisualizacion == nil {
		return rechazar[dtoresult.Zona](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	var cambios outbound.CambiosZona
	if c.Nombre != nil {
		n, err := vo.TextoObligatorio(*c.Nombre, maxNombreZona)
		if err != nil {
			return rechazar[dtoresult.Zona](mapper.Validacion("nombre_invalido", "el nombre es obligatorio (máximo 80 caracteres)"))
		}
		cambios.Nombre = &n
	}
	if c.OrdenVisualizacion != nil {
		o, err := vo.NewOrden(*c.OrdenVisualizacion)
		if err != nil {
			return rechazar[dtoresult.Zona](mapper.Validacion("orden_invalido", "el orden de visualización debe estar entre 0 y 9999"))
		}
		v := o.Int()
		cambios.Orden = &v
	}
	z, found, err := uc.zonas.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deZona(err, "editar zona")
	}
	if !found {
		return rechazar[dtoresult.Zona](mapper.NoEncontrado("zona"))
	}
	return appresult.Ok(z), nil
}

// CambiarEstadoZona desactiva (sin motivo) o restaura una zona. No se desactiva una zona con
// mesas activas: primero hay que dar de baja o mover las mesas.
type CambiarEstadoZona struct{ zonas outbound.ZonaStore }

func NewCambiarEstadoZona(z outbound.ZonaStore) *CambiarEstadoZona { return &CambiarEstadoZona{z} }

func (uc *CambiarEstadoZona) Execute(ctx context.Context, c cmd.CambiarEstadoZona) (appresult.Result[dtoresult.Zona], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	actual, ae, err := zonaGestionable(ctx, uc.zonas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Zona]("buscar zona", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Zona](ae)
	}
	if c.Activo && actual.EsActivo {
		return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoYaActiva, "la zona ya está activa"))
	}
	if !c.Activo {
		if !actual.EsActivo {
			return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la zona ya está dada de baja"))
		}
		n, err := uc.zonas.ContarMesasActivas(ctx, c.ID)
		if err != nil {
			return tecnico[dtoresult.Zona]("contar mesas activas", err)
		}
		if n > 0 {
			return rechazar[dtoresult.Zona](apperror.New(apperror.KindConflict, CodigoZonaConMesas,
				"la zona tiene mesas activas: dé de baja o mueva las mesas antes de desactivarla"))
		}
	}
	z, found, err := uc.zonas.CambiarActivo(ctx, c.ID, c.Activo)
	if err != nil {
		return deZona(err, "cambiar estado de la zona")
	}
	if !found {
		return rechazar[dtoresult.Zona](mapper.NoEncontrado("zona"))
	}
	return appresult.Ok(z), nil
}
```

- [ ] **Step 4: Ejecutar las pruebas unitarias**

Run: `gofmt -l internal/ ; go vet ./internal/modules/atencion/... && go test ./internal/modules/atencion/... 2>&1 | tail`
Expected: PASS.

- [ ] **Step 5: Persistencia de escritura**

`infrastructure/persistence/write/repository/sql.go`:

```go
// Package repository contiene el SQL parametrizado de escritura del módulo atencion.
// Recibe un database.DBTX: en operaciones de escritura es el *sql.Tx de la unidad de trabajo.
package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"service-restaurant/internal/modules/atencion/application/port/outbound"
)

// Translate convierte los errores de PostgreSQL conocidos en los errores de puerto que los
// casos de uso entienden como rechazos esperados. Cualquier otro error pasa sin cambios.
func Translate(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23505": // unique_violation
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

type scanner interface{ Scan(...any) error }

func notFound(err error) (bool, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return false, err
}
```

`infrastructure/persistence/write/repository/zonas_sql.go`:

```go
package repository

import (
	"context"
	"errors"
	"strings"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

const zonaCols = `id, sucursal_id, nombre, orden_visualizacion, es_activo, creado_en, creado_por, actualizado_en, actualizado_por`

func scanZona(s scanner) (dtoresult.Zona, error) {
	var z dtoresult.Zona
	err := s.Scan(&z.ID, &z.SucursalID, &z.Nombre, &z.OrdenVisualizacion, &z.EsActivo,
		&z.CreadoEn, &z.CreadoPor, &z.ActualizadoEn, &z.ActualizadoPor)
	return z, err
}

// traducirZona distingue la colisión del nombre de zona (zonas_sucursal_id_nombre_key).
func traducirZona(err error) error {
	err = Translate(err)
	if errors.Is(err, outbound.ErrDuplicado) && strings.Contains(err.Error(), "zonas_sucursal_id_nombre_key") {
		return outbound.ErrNombreZonaDuplicado
	}
	return err
}

// ZonaSQLWriter opera sobre sch_restaurante.zonas.
type ZonaSQLWriter struct{ db database.DBTX }

func NewZonaSQLWriter(db database.DBTX) *ZonaSQLWriter { return &ZonaSQLWriter{db} }

func (r *ZonaSQLWriter) Insert(ctx context.Context, n outbound.NuevaZona) (dtoresult.Zona, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.zonas (sucursal_id, nombre, orden_visualizacion)
		VALUES ($1, $2, $3) RETURNING `+zonaCols, n.SucursalID, n.Nombre, n.Orden))
	return z, traducirZona(err)
}

// GetForUpdate lee la zona bloqueando su fila hasta el final de la transacción: serializa la
// baja de la zona con el alta de mesas en ella.
func (r *ZonaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx,
		`SELECT `+zonaCols+` FROM sch_restaurante.zonas WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, err
	}
	return z, true, nil
}

func (r *ZonaSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosZona) (dtoresult.Zona, bool, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.zonas SET
			nombre = COALESCE($2, nombre),
			orden_visualizacion = COALESCE($3, orden_visualizacion)
		WHERE id = $1 RETURNING `+zonaCols, id, c.Nombre, c.Orden))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, traducirZona(err)
	}
	return z, true, nil
}

func (r *ZonaSQLWriter) CountMesasActivas(ctx context.Context, zonaID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM sch_restaurante.mesas WHERE zona_id = $1 AND es_activo`, zonaID).Scan(&n)
	return n, err
}

// SetActivo desactiva o reactiva la misma fila (conserva el ID).
func (r *ZonaSQLWriter) SetActivo(ctx context.Context, id int64, activo bool) (dtoresult.Zona, bool, error) {
	z, err := scanZona(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.zonas SET es_activo = $2 WHERE id = $1 RETURNING `+zonaCols, id, activo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Zona{}, false, traducirZona(err)
	}
	return z, true, nil
}
```

`infrastructure/persistence/write/adapter/zonas.go`:

```go
// Package adapter implementa los puertos de salida de escritura del módulo atencion.
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// ZonaWriteAdapter implementa outbound.ZonaStore.
type ZonaWriteAdapter struct{ repo *repository.ZonaSQLWriter }

func NewZonaWriteAdapter(db database.DBTX) *ZonaWriteAdapter {
	return &ZonaWriteAdapter{repo: repository.NewZonaSQLWriter(db)}
}

var _ outbound.ZonaStore = (*ZonaWriteAdapter)(nil)

func (a *ZonaWriteAdapter) Crear(ctx context.Context, n outbound.NuevaZona) (dtoresult.Zona, error) {
	return a.repo.Insert(ctx, n)
}
func (a *ZonaWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Zona, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *ZonaWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosZona) (dtoresult.Zona, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *ZonaWriteAdapter) ContarMesasActivas(ctx context.Context, zonaID int64) (int, error) {
	return a.repo.CountMesasActivas(ctx, zonaID)
}
func (a *ZonaWriteAdapter) CambiarActivo(ctx context.Context, id int64, activo bool) (dtoresult.Zona, bool, error) {
	return a.repo.SetActivo(ctx, id, activo)
}
```

- [ ] **Step 6: Capa web**

`web/dto/request/opcional.go`:

```go
// Package request define los cuerpos HTTP del módulo atencion.
package request

import "encoding/json"

// Opcional distingue un campo ausente de uno enviado como null (PATCH).
type Opcional[T any] struct {
	Presente bool
	Valor    *T
}

func (o *Opcional[T]) UnmarshalJSON(b []byte) error {
	o.Presente = true
	if string(b) == "null" {
		o.Valor = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Valor = &v
	return nil
}

// MarshalJSON permite calcular la huella canónica de la solicitud.
func (o Opcional[T]) MarshalJSON() ([]byte, error) {
	if !o.Presente || o.Valor == nil {
		return []byte("null"), nil
	}
	return json.Marshal(o.Valor)
}

// Motivo es el cuerpo de la baja con motivo (mesas).
type Motivo struct {
	Motivo string `json:"motivo"`
}

// Confirmacion es el cuerpo (vacío) de las restauraciones y de la baja simple de zonas.
type Confirmacion struct{}
```

`web/dto/request/zonas.go`:

```go
package request

// ZonaAlta: sucursal_id viaja como cadena (contrato); un campo desconocido responde 400.
type ZonaAlta struct {
	SucursalID         string `json:"sucursal_id"`
	Nombre             string `json:"nombre"`
	OrdenVisualizacion *int   `json:"orden_visualizacion"`
}

// ZonaEdicion: la sede de una zona no se edita (sucursal_id responde 400).
type ZonaEdicion struct {
	Nombre             *string `json:"nombre,omitempty"`
	OrdenVisualizacion *int    `json:"orden_visualizacion,omitempty"`
}
```

Reemplazar `web/controller/zonas_handler.go` completo:

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

// ZonasHandler: salones, terrazas y niveles del local (RES-RF-008).
type ZonasHandler struct {
	listar  inbound.ListarZonas
	obtener inbound.ObtenerZona
	crear   inbound.CrearZona
	editar  inbound.EditarZona
	estado  inbound.CambiarEstadoZona
}

func NewZonasHandler(l inbound.ListarZonas, o inbound.ObtenerZona, c inbound.CrearZona,
	e inbound.EditarZona, s inbound.CambiarEstadoZona) *ZonasHandler {
	return &ZonasHandler{l, o, c, e, s}
}

// Listar: GET /zonas?sucursal_id=&es_activo=&limit=&cursor=
func (h *ZonasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarZonas{
		Actor: actor, SucursalID: sede, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToZonasPagina)
}

// Obtener: GET /zonas/{id}
func (h *ZonasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerZona{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
}

// Crear: POST /zonas (Idempotency-Key obligatoria)
func (h *ZonasHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.ZonaAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	sede, err := idCadena(body.SucursalID, "sucursal_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, "POST /zonas", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearZona{
		Intencion: in, Actor: actor, SucursalID: sede, Nombre: body.Nombre, OrdenVisualizacion: body.OrdenVisualizacion,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToZona)
}

// Editar: PATCH /zonas/{id}
func (h *ZonasHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.ZonaEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	res, err := h.editar.Execute(r.Context(), command.EditarZona{
		Actor: actor, ID: id, Nombre: body.Nombre, OrdenVisualizacion: body.OrdenVisualizacion,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
}

// Baja: POST /zonas/{id}/baja (sin motivo; Idempotency-Key obligatoria)
func (h *ZonasHandler) Baja(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, false)
}

// Restauracion: POST /zonas/{id}/restauracion (Idempotency-Key obligatoria)
func (h *ZonasHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
	return h.cambiarEstado(w, r, true)
}

func (h *ZonasHandler) cambiarEstado(w http.ResponseWriter, r *http.Request, activo bool) error {
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
	accion := "baja"
	if activo {
		accion = "restauracion"
	}
	in, err := intencion(r, fmt.Sprintf("POST /zonas/%d/%s", id, accion), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoZona{Intencion: in, Actor: actor, ID: id, Activo: activo})
	return responder(w, res, err, http.StatusOK, webmapper.ToZona)
}
```

Reemplazar la función en `web/controller/routes.go`:

```go
func RegisterRoutes(mux *http.ServeMux, zonas *ZonasHandler, protect func(http.Handler) http.Handler, logger *slog.Logger) {
	ruta := func(patron string, e handler.Endpoint) { mux.Handle(patron, protect(handler.Adapt(logger, e))) }

	ruta("GET /zonas", zonas.Listar)
	ruta("POST /zonas", zonas.Crear)
	ruta("GET /zonas/{id}", zonas.Obtener)
	ruta("PATCH /zonas/{id}", zonas.Editar)
	ruta("POST /zonas/{id}/baja", zonas.Baja)
	ruta("POST /zonas/{id}/restauracion", zonas.Restauracion)
}
```

- [ ] **Step 7: Composición**

Reemplazar `infrastructure/config/zonas.go`:

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

// armarZonas compone los casos de uso de zonas, ya decorados con transacción e idempotencia.
func armarZonas(db *sql.DB, logger *slog.Logger) *controller.ZonasHandler {
	crear := escritura[cmd.CrearZona, dtoresult.Zona]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearZona, dtoresult.Zona] {
			return command.NewCrearZona(writeadapter.NewZonaWriteAdapter(q))
		},
		meta: func(c cmd.CrearZona) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarZona, dtoresult.Zona]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarZona, dtoresult.Zona] {
			return command.NewEditarZona(writeadapter.NewZonaWriteAdapter(q))
		},
		meta: func(c cmd.EditarZona) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoZona, dtoresult.Zona]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoZona, dtoresult.Zona] {
			return command.NewCambiarEstadoZona(writeadapter.NewZonaWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoZona) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewZonaReadAdapter(db)
	return controller.NewZonasHandler(uquery.NewListarZonas(lector), uquery.NewObtenerZona(lector),
		inbound.CrearZona(crear), inbound.EditarZona(editar), inbound.CambiarEstadoZona(estado))
}
```

En `infrastructure/config/module.go` cambiar la llamada a `armarZonas(d.DB, d.Logger)`.

- [ ] **Step 8: Compilar**

Run: `go build ./... && go vet ./... && go test ./tests/architecture/ ./internal/...`
Expected: sin errores.

- [ ] **Step 9: Pruebas de integración de escritura de zonas**

Añadir a `tests/integration/atencion_zonas_test.go` (importar además `"fmt"`):

```go
func altaZona(sucursalID, nombre string) string {
	return fmt.Sprintf(`{"sucursal_id":"%s","nombre":"%s","orden_visualizacion":3}`, sucursalID, nombre)
}

func TestZonaCrearConIdempotenciaYValidaciones(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	tk := gestorMesas(t)
	principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	h := map[string]string{"Idempotency-Key": nuevaClave()}

	r1 := callH(t, "POST", "/api/v1/zonas", altaZona(principal, "Zona nueva 1"), tk, h)
	esperar(t, r1, http.StatusCreated, "")
	var z zonaResp
	r1.json(t, &z)
	if z.Nombre != "Zona nueva 1" || !z.EsActivo || z.SucursalID != principal || z.Orden != 3 ||
		z.CreadoPor != idDe(t, "usuarios", "username", "mesa.gestor") {
		t.Fatalf("zona creada: %+v", z)
	}
	// Repetir la misma intención no duplica; otro contenido con la misma clave es conflicto.
	r2 := callH(t, "POST", "/api/v1/zonas", altaZona(principal, "Zona nueva 1"), tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var rep zonaResp
	r2.json(t, &rep)
	if rep.ID != z.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.zonas WHERE nombre='Zona nueva 1'`) != 1 {
		t.Fatalf("la repetición debe devolver la misma zona: %+v vs %+v", rep, z)
	}
	esperar(t, callH(t, "POST", "/api/v1/zonas", altaZona(principal, "Otra"), tk, h), http.StatusConflict, "idempotency_key_reutilizada")

	esperar(t, post(t, "/api/v1/zonas", altaZona(principal, "Zona nueva 1"), tk), http.StatusConflict, "nombre_duplicado")
	esperar(t, call(t, "POST", "/api/v1/zonas", altaZona(principal, "Sin clave"), tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"`+principal+`","nombre":"x","creado_por":"1"}`, tk), http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"abc","nombre":"x"}`, tk), http.StatusBadRequest, "sucursal_id_invalido")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"`+principal+`","nombre":"  "}`, tk), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, post(t, "/api/v1/zonas", `{"sucursal_id":"`+principal+`","nombre":"x","orden_visualizacion":-1}`, tk), http.StatusUnprocessableEntity, "orden_invalido")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.zonas WHERE nombre IN ('Sin clave','x','  ')`) != 0 {
		t.Fatal("los rechazos no deben dejar zonas")
	}

	// Sede ajena: 404. Solo lectura: 403 sin_permiso.
	ajena := idDe(t, "sucursales", "codigo", "MESA-AJENA")
	esperar(t, post(t, "/api/v1/zonas", altaZona(ajena, "Intrusa"), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/zonas", altaZona(principal, "Sin poder"), lectorMesas(t)), http.StatusForbidden, "sin_permiso")
}

func TestZonaEditar(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Editable")
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "MESA-AJENA", "Ajena editable")
	tk := gestorMesas(t)
	id := idDe(t, "zonas", "nombre", "Editable")

	r := callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"Editada","orden_visualizacion":7}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var z zonaResp
	r.json(t, &z)
	if z.Nombre != "Editada" || z.Orden != 7 {
		t.Fatalf("edición: %+v", z)
	}
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"sucursal_id":"1"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":" "}`, tk, nil), http.StatusUnprocessableEntity, "nombre_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/999999", `{"nombre":"x"}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+idDe(t, "zonas", "nombre", "Ajena editable"), `{"nombre":"Hack"}`, tk, nil),
		http.StatusNotFound, "no_encontrado")

	crearZonaSQL(t, "PRINCIPAL", "Otra existente")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"Otra existente"}`, tk, nil), http.StatusConflict, "nombre_duplicado")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"Hack"}`, lectorMesas(t), nil), http.StatusForbidden, "sin_permiso")
}

func TestZonaBajaRestauracionYMesasActivas(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Ciclo zona")
	crearMesaSQL(t, "PRINCIPAL", "Ciclo zona", "CZ-1", 4)
	tk := gestorMesas(t)
	id := idDe(t, "zonas", "nombre", "Ciclo zona")

	// Con una mesa activa la zona no se desactiva.
	esperar(t, post(t, "/api/v1/zonas/"+id+"/baja", "", tk), http.StatusConflict, "zona_con_mesas")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.zonas WHERE nombre='Ciclo zona' AND es_activo`) != 1 {
		t.Fatal("la zona debe seguir activa")
	}
	asSystem(t, `UPDATE sch_restaurante.mesas SET es_activo=false, motivo_eliminacion='Fuera de servicio' WHERE numero_mesa='CZ-1'`)

	h := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/zonas/"+id+"/baja", "", tk, h)
	esperar(t, r, http.StatusOK, "")
	var z zonaResp
	r.json(t, &z)
	if z.EsActivo {
		t.Fatalf("baja: %+v", z)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='zonas' AND accion='UPDATE'
		AND registro_id=`+id+` AND creado_por=`+idDe(t, "usuarios", "username", "mesa.gestor")) == 0 {
		t.Fatal("la baja debe quedar auditada con el actor")
	}
	antes := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='zonas' AND registro_id=`+id)
	esperar(t, callH(t, "POST", "/api/v1/zonas/"+id+"/baja", "", tk, h), http.StatusOK, "")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='zonas' AND registro_id=`+id) != antes {
		t.Fatal("repetir la misma clave no debe volver a escribir")
	}
	esperar(t, post(t, "/api/v1/zonas/"+id+"/baja", "", tk), http.StatusConflict, "ya_dada_de_baja")
	esperar(t, callH(t, "PATCH", "/api/v1/zonas/"+id, `{"nombre":"x"}`, tk, nil), http.StatusConflict, "zona_inactiva")

	r = post(t, "/api/v1/zonas/"+id+"/restauracion", `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rest zonaResp
	r.json(t, &rest)
	if rest.ID != z.ID || !rest.EsActivo {
		t.Fatalf("restauración debe conservar el ID: %+v", rest)
	}
	esperar(t, post(t, "/api/v1/zonas/"+id+"/restauracion", "", tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/zonas/999999/baja", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/zonas/"+id+"/baja", "", lectorMesas(t)), http.StatusForbidden, "sin_permiso")
}
```

- [ ] **Step 10: Ejecutar y verificar**

Run: `go test -count=1 ./tests/integration/ -run "TestZona" -v` y luego `go test -count=1 -shuffle=on ./tests/integration/ -run "TestZona|TestSucursal|TestNegocio"`
Expected: PASS ambas.

- [ ] **Step 11: Regresión y commit**

Run: `gofmt -l . ; go vet ./... && go test ./... && go test -count=1 ./tests/integration/...`
Expected: todo en verde.

```bash
git add internal/modules/atencion tests/integration
git commit -m "I2 paso 2: crear, editar, dar de baja y restaurar zonas

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Mesas — listar y consultar

**Files:**
- Create: `application/dto/query/mesas.go`, `application/dto/result/mesas.go`
- Create: `application/port/inbound/mesas.go`, `application/port/outbound/mesas.go`
- Create: `application/usecase/query/mesas.go`, `application/usecase/query/mesas_test.go`
- Create: `infrastructure/persistence/read/repository/mesas_sql_reader.go`, `infrastructure/persistence/read/adapter/mesas_read_adapter.go`
- Create: `web/dto/response/mesas.go`, `web/mapper/mesas_web_mapper.go`, `web/controller/mesas_handler.go`
- Modify: `web/controller/routes.go`, `infrastructure/config/module.go`; Create: `infrastructure/config/mesas.go`
- Create: `tests/integration/atencion_mesas_test.go`

**Interfaces:**
- Consumes: Tareas 1–3; `sedesPedidas` (paquete `query`).
- Produces:
  - `result.Mesa{ID, SucursalID, ZonaID int64; NumeroMesa string; Capacidad int; Habilitada bool; PosicionX, PosicionY *string; EsActivo bool; MotivoEliminacion *string; EliminadoEn *time.Time; EliminadoPor *int64; CreadoEn time.Time; CreadoPor int64; ActualizadoEn time.Time; ActualizadoPor int64}`, `result.PaginaMesas`.
  - `query.ListarMesas{Actor; SucursalID *int64; Activo *bool; Limite int; Cursor string}`, `query.ObtenerMesa{Actor; ID int64}`.
  - `inbound.ListarMesas`, `inbound.ObtenerMesa`; `outbound.FiltroMesas`, `outbound.MesaReader`.
  - `controller.NewMesasHandler(listar, obtener)`; `response.Mesa`, `response.MesasPagina`; `webmapper.ToMesa`, `ToMesasPagina`.

- [ ] **Step 1: DTO, puertos y pruebas unitarias (fallan)**

`application/dto/query/mesas.go`:

```go
package query

import appctx "service-restaurant/internal/shared/application/context"

// ListarMesas pagina por cursor (id ascendente) las mesas de las sedes que el actor puede ver.
type ListarMesas struct {
	Actor      appctx.Actor
	SucursalID *int64
	Activo     *bool
	Limite     int
	Cursor     string
}

type ObtenerMesa struct {
	Actor appctx.Actor
	ID    int64
}
```

`application/dto/result/mesas.go`:

```go
package result

import "time"

// Mesa es una mesa del salón. La ocupación no se expone: se deriva de los pedidos abiertos (I3).
// Las inactivas llevan motivo y actor de la baja.
type Mesa struct {
	ID                int64
	SucursalID        int64
	ZonaID            int64
	NumeroMesa        string
	Capacidad         int
	Habilitada        bool
	PosicionX         *string
	PosicionY         *string
	EsActivo          bool
	MotivoEliminacion *string
	EliminadoEn       *time.Time
	EliminadoPor      *int64
	CreadoEn          time.Time
	CreadoPor         int64
	ActualizadoEn     time.Time
	ActualizadoPor    int64
}

// PaginaMesas: NextCursor vacío indica la última página.
type PaginaMesas struct {
	Items      []Mesa
	NextCursor string
}
```

`application/port/inbound/mesas.go`:

```go
package inbound

import (
	"context"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	appresult "service-restaurant/internal/shared/kernel/result"
)

type ListarMesas interface {
	Execute(context.Context, qry.ListarMesas) (appresult.Result[dtoresult.PaginaMesas], error)
}
type ObtenerMesa interface {
	Execute(context.Context, qry.ObtenerMesa) (appresult.Result[dtoresult.Mesa], error)
}
```

`application/port/outbound/mesas.go`:

```go
package outbound

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
)

// FiltroMesas acota el listado a las sedes visibles (SedeIDs) y, opcionalmente, a su estado.
type FiltroMesas struct {
	SedeIDs     []int64
	Activo      *bool
	DespuesDeID int64
	Limite      int
}

// MesaReader lee mesas sin bloquear filas.
type MesaReader interface {
	ListarMesas(ctx context.Context, f FiltroMesas) ([]dtoresult.Mesa, error)
	ObtenerMesa(ctx context.Context, id int64) (dtoresult.Mesa, bool, error)
}
```

`application/usecase/query/mesas_test.go`:

```go
package query

import (
	"context"
	"slices"
	"testing"

	qry "service-restaurant/internal/modules/atencion/application/dto/query"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/domain/policy"
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
)

type mesaReaderFake struct {
	filtro   outbound.FiltroMesas
	llamadas int
	filas    []dtoresult.Mesa
	una      dtoresult.Mesa
}

func (f *mesaReaderFake) ListarMesas(_ context.Context, fl outbound.FiltroMesas) ([]dtoresult.Mesa, error) {
	f.llamadas++
	f.filtro = fl
	return f.filas, nil
}
func (f *mesaReaderFake) ObtenerMesa(_ context.Context, id int64) (dtoresult.Mesa, bool, error) {
	f.llamadas++
	return f.una, f.una.ID == id, nil
}

func TestListarMesasAcotaPorLasSedesDelActor(t *testing.T) {
	f := &mesaReaderFake{filas: []dtoresult.Mesa{{ID: 1}, {ID: 2}, {ID: 3}}}
	a := actorMesas(
		appctx.Grant{SucursalID: 4, Permiso: policy.MesaVer},
		appctx.Grant{SucursalID: 2, Permiso: policy.MesaAdministrar})
	res, err := NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a, Limite: 2})
	if err != nil || !res.IsOk() {
		t.Fatalf("listar: %v %+v", err, res)
	}
	if !slices.Equal(f.filtro.SedeIDs, []int64{2, 4}) || f.filtro.Limite != 3 {
		t.Fatalf("filtro: %+v", f.filtro)
	}
	if p := res.Value(); len(p.Items) != 2 || p.NextCursor == "" {
		t.Fatalf("debía recortar a 2 y dar cursor: %+v", p)
	}
}

func TestListarMesasRechazos(t *testing.T) {
	f := &mesaReaderFake{}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	sinPermiso := actorMesas(appctx.Grant{SucursalID: 1, Permiso: "NEGOCIO_VER"})
	res, _ := NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: sinPermiso})
	if res.Err() == nil || res.Err().Kind != apperror.KindForbidden || f.llamadas != 0 {
		t.Fatalf("sin permiso debe ser 403 sin tocar el lector: %+v", res.Err())
	}
	ajena := int64(9)
	res, _ = NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a, SucursalID: &ajena})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound || f.llamadas != 0 {
		t.Fatalf("una sede ajena debe ser 404: %+v", res.Err())
	}
	res, _ = NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a, Cursor: "%%%"})
	if res.Err() == nil || res.Err().Kind != apperror.KindBadRequest {
		t.Fatalf("cursor inválido debe ser 400: %+v", res.Err())
	}
	res, _ = NewListarMesas(f).Execute(context.Background(), qry.ListarMesas{Actor: a})
	if !res.IsOk() || res.Value().Items == nil {
		t.Fatalf("una lista vacía debe ser [] y no nula: %+v", res.Value())
	}
}

func TestObtenerMesaOcultaLasAjenas(t *testing.T) {
	f := &mesaReaderFake{una: dtoresult.Mesa{ID: 9, SucursalID: 3}}
	a := actorMesas(appctx.Grant{SucursalID: 1, Permiso: policy.MesaVer})
	res, _ := NewObtenerMesa(f).Execute(context.Background(), qry.ObtenerMesa{Actor: a, ID: 9})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una mesa de sede ajena debe ser 404: %+v", res.Err())
	}
	f.una = dtoresult.Mesa{ID: 5, SucursalID: 1}
	res, _ = NewObtenerMesa(f).Execute(context.Background(), qry.ObtenerMesa{Actor: a, ID: 5})
	if !res.IsOk() || res.Value().ID != 5 {
		t.Fatalf("una mesa propia debe leerse: %+v", res.Err())
	}
	res, _ = NewObtenerMesa(f).Execute(context.Background(), qry.ObtenerMesa{Actor: a, ID: 77})
	if res.Err() == nil || res.Err().Kind != apperror.KindNotFound {
		t.Fatalf("una mesa inexistente debe ser 404: %+v", res.Err())
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `go test ./internal/modules/atencion/application/usecase/query/ -run "Mesa" 2>&1 | head`
Expected: FAIL de compilación (`undefined: NewListarMesas`).

- [ ] **Step 3: Casos de uso**

`application/usecase/query/mesas.go`:

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

// ListarMesas devuelve las mesas de las sedes visibles. Pide limite+1 filas para saber si hay
// más páginas.
type ListarMesas struct{ mesas outbound.MesaReader }

func NewListarMesas(m outbound.MesaReader) *ListarMesas { return &ListarMesas{m} }

func (uc *ListarMesas) Execute(ctx context.Context, q qry.ListarMesas) (appresult.Result[dtoresult.PaginaMesas], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.PaginaMesas](ae), nil
	}
	sedes, ok := sedesPedidas(q.Actor, q.SucursalID)
	if !ok {
		return appresult.Fail[dtoresult.PaginaMesas](mapper.NoEncontrado("sucursal")), nil
	}
	desde, ae := mapper.DecodeCursor(q.Cursor)
	if ae != nil {
		return appresult.Fail[dtoresult.PaginaMesas](ae), nil
	}
	n := tamanio(q.Limite)
	items, err := uc.mesas.ListarMesas(ctx, outbound.FiltroMesas{SedeIDs: sedes, Activo: q.Activo, DespuesDeID: desde, Limite: n + 1})
	if err != nil {
		return appresult.Result[dtoresult.PaginaMesas]{}, fmt.Errorf("listar mesas: %w", err)
	}
	pagina := dtoresult.PaginaMesas{Items: items}
	if len(items) > n && n > 0 {
		pagina.Items = items[:n]
		pagina.NextCursor = mapper.EncodeCursor(items[n-1].ID)
	}
	if pagina.Items == nil {
		pagina.Items = []dtoresult.Mesa{}
	}
	return appresult.Ok(pagina), nil
}

// ObtenerMesa lee una mesa; la de una sede que el actor no ve responde 404.
type ObtenerMesa struct{ mesas outbound.MesaReader }

func NewObtenerMesa(m outbound.MesaReader) *ObtenerMesa { return &ObtenerMesa{m} }

func (uc *ObtenerMesa) Execute(ctx context.Context, q qry.ObtenerMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(q.Actor, mensajeSinPermisoLectura, policy.MesaVer, policy.MesaAdministrar); ae != nil {
		return appresult.Fail[dtoresult.Mesa](ae), nil
	}
	m, found, err := uc.mesas.ObtenerMesa(ctx, q.ID)
	if err != nil {
		return appresult.Result[dtoresult.Mesa]{}, fmt.Errorf("obtener mesa: %w", err)
	}
	if !found || !mapper.PuedeVer(q.Actor, m.SucursalID) {
		return appresult.Fail[dtoresult.Mesa](mapper.NoEncontrado("mesa")), nil
	}
	return appresult.Ok(m), nil
}
```

- [ ] **Step 4: Ejecutar las pruebas unitarias**

Run: `gofmt -l internal/ ; go test ./internal/modules/atencion/... 2>&1 | tail`
Expected: PASS.

- [ ] **Step 5: Lector SQL y adaptador**

`infrastructure/persistence/read/repository/mesas_sql_reader.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

// MesaSQLReader lee mesas.
type MesaSQLReader struct{ db database.DBTX }

func NewMesaSQLReader(db database.DBTX) *MesaSQLReader { return &MesaSQLReader{db} }

// Las posiciones son numeric(10,2): se leen como texto para no perder exactitud.
const mesaCols = `id, sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x::text, posicion_y::text,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func cadenaNula(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func leerMesa(s fila) (dtoresult.Mesa, error) {
	var m dtoresult.Mesa
	var px, py, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&m.ID, &m.SucursalID, &m.ZonaID, &m.NumeroMesa, &m.Capacidad, &m.Habilitada, &px, &py,
		&m.EsActivo, &motivo, &elimEn, &elimPor, &m.CreadoEn, &m.CreadoPor, &m.ActualizadoEn, &m.ActualizadoPor)
	m.PosicionX, m.PosicionY, m.MotivoEliminacion = cadenaNula(px), cadenaNula(py), cadenaNula(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		m.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		m.EliminadoPor = &v
	}
	return m, err
}

// ListMesas devuelve las mesas de f.SedeIDs ordenadas por id (cursor estable).
func (r *MesaSQLReader) ListMesas(ctx context.Context, f outbound.FiltroMesas) ([]dtoresult.Mesa, error) {
	var activo sql.NullBool
	if f.Activo != nil {
		activo = sql.NullBool{Bool: *f.Activo, Valid: true}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+mesaCols+` FROM sch_restaurante.mesas
		WHERE sucursal_id = ANY($1::bigint[]) AND id > $2 AND ($3::boolean IS NULL OR es_activo = $3)
		ORDER BY id LIMIT $4`, f.SedeIDs, f.DespuesDeID, activo, f.Limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dtoresult.Mesa
	for rows.Next() {
		m, err := leerMesa(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MesaSQLReader) GetMesa(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := leerMesa(r.db.QueryRowContext(ctx, `SELECT `+mesaCols+` FROM sch_restaurante.mesas WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return dtoresult.Mesa{}, false, nil
	}
	return m, err == nil, err
}
```

`infrastructure/persistence/read/adapter/mesas_read_adapter.go`:

```go
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/read/repository"
	"service-restaurant/internal/platform/database"
)

// MesaReadAdapter implementa outbound.MesaReader.
type MesaReadAdapter struct{ repo *repository.MesaSQLReader }

func NewMesaReadAdapter(db database.DBTX) *MesaReadAdapter {
	return &MesaReadAdapter{repo: repository.NewMesaSQLReader(db)}
}

var _ outbound.MesaReader = (*MesaReadAdapter)(nil)

func (a *MesaReadAdapter) ListarMesas(ctx context.Context, f outbound.FiltroMesas) ([]dtoresult.Mesa, error) {
	return a.repo.ListMesas(ctx, f)
}
func (a *MesaReadAdapter) ObtenerMesa(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return a.repo.GetMesa(ctx, id)
}
```

- [ ] **Step 6: Capa web**

`web/dto/response/mesas.go`:

```go
package response

import "time"

// Mesa: las posiciones son decimales exactos serializados como cadena ("12.50").
type Mesa struct {
	ID                string     `json:"id"`
	SucursalID        string     `json:"sucursal_id"`
	ZonaID            string     `json:"zona_id"`
	NumeroMesa        string     `json:"numero_mesa"`
	Capacidad         int        `json:"capacidad"`
	Habilitada        bool       `json:"habilitada"`
	PosicionX         *string    `json:"posicion_x"`
	PosicionY         *string    `json:"posicion_y"`
	EsActivo          bool       `json:"es_activo"`
	MotivoEliminacion *string    `json:"motivo_eliminacion"`
	EliminadoEn       *time.Time `json:"eliminado_en"`
	EliminadoPor      *string    `json:"eliminado_por"`
	CreadoEn          time.Time  `json:"creado_en"`
	CreadoPor         string     `json:"creado_por"`
	ActualizadoEn     time.Time  `json:"actualizado_en"`
	ActualizadoPor    string     `json:"actualizado_por"`
}

type MesasPagina struct {
	Items      []Mesa  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
```

`web/mapper/mesas_web_mapper.go`:

```go
package mapper

import (
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/web/dto/response"
)

func ToMesa(m dtoresult.Mesa) response.Mesa {
	var eliminadoPor *string
	if m.EliminadoPor != nil {
		v := id(*m.EliminadoPor)
		eliminadoPor = &v
	}
	return response.Mesa{
		ID: id(m.ID), SucursalID: id(m.SucursalID), ZonaID: id(m.ZonaID), NumeroMesa: m.NumeroMesa,
		Capacidad: m.Capacidad, Habilitada: m.Habilitada, PosicionX: m.PosicionX, PosicionY: m.PosicionY,
		EsActivo: m.EsActivo, MotivoEliminacion: m.MotivoEliminacion, EliminadoEn: m.EliminadoEn, EliminadoPor: eliminadoPor,
		CreadoEn: m.CreadoEn, CreadoPor: id(m.CreadoPor), ActualizadoEn: m.ActualizadoEn, ActualizadoPor: id(m.ActualizadoPor),
	}
}

func ToMesasPagina(p dtoresult.PaginaMesas) response.MesasPagina {
	items := make([]response.Mesa, len(p.Items))
	for i, m := range p.Items {
		items[i] = ToMesa(m)
	}
	return response.MesasPagina{Items: items, NextCursor: cursor(p.NextCursor)}
}
```

`web/controller/mesas_handler.go`:

```go
package controller

import (
	"net/http"

	"service-restaurant/internal/modules/atencion/application/dto/query"
	"service-restaurant/internal/modules/atencion/application/port/inbound"
	webmapper "service-restaurant/internal/modules/atencion/web/mapper"
)

// MesasHandler: mesas del salón (RES-RF-008).
type MesasHandler struct {
	listar  inbound.ListarMesas
	obtener inbound.ObtenerMesa
}

func NewMesasHandler(l inbound.ListarMesas, o inbound.ObtenerMesa) *MesasHandler {
	return &MesasHandler{l, o}
}

// Listar: GET /mesas?sucursal_id=&es_activo=&limit=&cursor=
func (h *MesasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarMesas{
		Actor: actor, SucursalID: sede, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesasPagina)
}

// Obtener: GET /mesas/{id}
func (h *MesasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerMesa{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}
```

Reemplazar `RegisterRoutes` en `web/controller/routes.go`:

```go
func RegisterRoutes(mux *http.ServeMux, zonas *ZonasHandler, mesas *MesasHandler,
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
}
```

`infrastructure/config/mesas.go`:

```go
package config

import (
	"database/sql"

	uquery "service-restaurant/internal/modules/atencion/application/usecase/query"
	readadapter "service-restaurant/internal/modules/atencion/infrastructure/persistence/read/adapter"
	"service-restaurant/internal/modules/atencion/web/controller"
)

// armarMesas compone los casos de uso de mesas.
func armarMesas(db *sql.DB) *controller.MesasHandler {
	lector := readadapter.NewMesaReadAdapter(db)
	return controller.NewMesasHandler(uquery.NewListarMesas(lector), uquery.NewObtenerMesa(lector))
}
```

En `infrastructure/config/module.go`: añadir el campo `mesas *controller.MesasHandler`; en `New`: `mesas: armarMesas(d.DB)`; en `RegisterRoutes`: `controller.RegisterRoutes(mux, m.zonas, m.mesas, protect, m.deps.Logger)`.

- [ ] **Step 7: Compilar**

Run: `go build ./... && go vet ./... && go test ./tests/architecture/ ./internal/...`
Expected: sin errores.

- [ ] **Step 8: Pruebas de integración de lectura de mesas**

`tests/integration/atencion_mesas_test.go`:

```go
package integration

import (
	"net/http"
	"slices"
	"strconv"
	"testing"
)

func TestMesasLecturaRespetaLasSedesDelActor(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Zona mesas L")
	crearZonaSQL(t, "MESA-AJENA", "Zona mesas ajena")
	crearMesaSQL(t, "PRINCIPAL", "Zona mesas L", "ML-1", 4)
	crearMesaSQL(t, "MESA-AJENA", "Zona mesas ajena", "MAJ-1", 2)
	tk := lectorMesas(t)

	numeros := listarMesas(t, "", tk).numeros()
	if !slices.Contains(numeros, "ML-1") || slices.Contains(numeros, "MAJ-1") {
		t.Fatalf("mesas visibles = %v: debe incluir ML-1 y no MAJ-1", numeros)
	}
	esperar(t, call(t, "GET", "/api/v1/mesas?sucursal_id="+idDe(t, "sucursales", "codigo", "MESA-AJENA"), "", tk),
		http.StatusNotFound, "no_encontrado")

	id := idDe(t, "mesas", "numero_mesa", "ML-1")
	r := call(t, "GET", "/api/v1/mesas/"+id, "", tk)
	esperar(t, r, http.StatusOK, "")
	var m mesaResp
	r.json(t, &m)
	if m.NumeroMesa != "ML-1" || m.Capacidad != 4 || !m.Habilitada || !m.EsActivo || m.PosicionX != nil || m.Motivo != nil ||
		m.SucursalID != idDe(t, "sucursales", "codigo", "PRINCIPAL") || m.ZonaID != idDe(t, "zonas", "nombre", "Zona mesas L") {
		t.Fatalf("mesa: %+v", m)
	}
	esperar(t, call(t, "GET", "/api/v1/mesas/"+idDe(t, "mesas", "numero_mesa", "MAJ-1"), "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/mesas/999999", "", tk), http.StatusNotFound, "no_encontrado")
	esperar(t, call(t, "GET", "/api/v1/mesas/abc", "", tk), http.StatusBadRequest, "id_invalido")
}

func TestMesasExigeAlgunPermisoDeMesa(t *testing.T) {
	rolConPermisos(t, "T_SIN_MESA", "NEGOCIO_VER")
	asegurarUsuario(t, "mesa.sin", "MesaSin2026Clave", "T_SIN_MESA", "PRINCIPAL")
	tk := token(t, "mesa.sin", "MesaSin2026Clave")
	esperar(t, call(t, "GET", "/api/v1/mesas", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/mesas/1", "", tk), http.StatusForbidden, "sin_permiso")
	esperar(t, call(t, "GET", "/api/v1/mesas", "", ""), http.StatusUnauthorized, "no_autenticado")
}

func TestMesasFiltroYPaginacion(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Zona pag mesas")
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-1", 2)
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-2", 2)
	crearMesaSQL(t, "PRINCIPAL", "Zona pag mesas", "PM-3", 2)
	asSystem(t, `UPDATE sch_restaurante.mesas SET es_activo=false, motivo_eliminacion='Retirada' WHERE numero_mesa='PM-3'`)
	tk := lectorMesas(t)

	if got := listarMesas(t, "?es_activo=true", tk).numeros(); slices.Contains(got, "PM-3") || !slices.Contains(got, "PM-1") {
		t.Fatalf("es_activo=true: %v", got)
	}
	inactivas := listarMesas(t, "?es_activo=false", tk)
	if got := inactivas.numeros(); !slices.Contains(got, "PM-3") || slices.Contains(got, "PM-1") {
		t.Fatalf("es_activo=false: %v", got)
	}
	for _, m := range inactivas.Items {
		if m.NumeroMesa == "PM-3" && (m.Motivo == nil || *m.Motivo != "Retirada" || m.EliminadoPor == nil) {
			t.Fatalf("una mesa inactiva debe exponer motivo y actor: %+v", m)
		}
	}
	esperar(t, call(t, "GET", "/api/v1/mesas?es_activo=quizas", "", tk), http.StatusBadRequest, "es_activo_invalido")
	esperar(t, call(t, "GET", "/api/v1/mesas?limit=101", "", tk), http.StatusBadRequest, "limit_invalido")
	esperar(t, call(t, "GET", "/api/v1/mesas?cursor=!!!", "", tk), http.StatusBadRequest, "cursor_invalido")

	p1 := listarMesas(t, "?limit=1", tk)
	if len(p1.Items) != 1 || p1.NextCursor == nil {
		t.Fatalf("primera página: %+v", p1)
	}
	p2 := listarMesas(t, "?limit=1&cursor="+*p1.NextCursor, tk)
	id1, _ := strconv.Atoi(p1.Items[0].ID)
	id2, _ := strconv.Atoi(p2.Items[0].ID)
	if len(p2.Items) != 1 || id1 >= id2 {
		t.Fatalf("la segunda página debe continuar en orden ascendente: %+v / %+v", p1, p2)
	}
}
```

- [ ] **Step 9: Ejecutar**

Run: `go test -count=1 ./tests/integration/ -run "TestMesas|TestZona" -v` y luego con `-shuffle=on`.
Expected: PASS ambas.

- [ ] **Step 10: Regresión y commit**

Run: `gofmt -l . ; go vet ./... && go test ./... && go test -count=1 ./tests/integration/...`
Expected: todo en verde.

```bash
git add internal/modules/atencion tests/integration
git commit -m "I2 paso 2: GET /mesas y /mesas/{id} con visibilidad por sede

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Mesas — crear, editar, baja con motivo y restauración

**Files:**
- Create: `application/dto/command/mesas.go`
- Modify: `application/port/inbound/mesas.go`, `application/port/outbound/mesas.go`
- Create: `application/usecase/command/mesas.go`, `application/usecase/command/mesas_test.go`
- Create: `infrastructure/persistence/write/repository/mesas_sql.go`, `infrastructure/persistence/write/adapter/mesas.go`
- Create: `web/dto/request/mesas.go`
- Modify: `web/controller/mesas_handler.go` (reemplazo completo), `web/controller/routes.go`, `infrastructure/config/mesas.go`
- Modify: `tests/integration/atencion_mesas_test.go`

**Interfaces:**
- Consumes: Tareas 1–4; `zonaStoreFake` y helpers de tests de `command` (`gestorDe`, `soloVer`, `esperarRechazo`, `entero`, `texto`).
- Produces:
  - `command.CrearMesa{Intencion; Actor; SucursalID, ZonaID int64; NumeroMesa string; Capacidad int; Habilitada *bool; PosicionX, PosicionY *string}`
  - `command.Campo[T]{Presente bool; Valor *T}`
  - `command.EditarMesa{Actor; ID int64; ZonaID *int64; NumeroMesa *string; Capacidad *int; Habilitada *bool; PosicionX, PosicionY Campo[string]}`
  - `command.CambiarEstadoMesa{Intencion; Actor; ID int64; Activo bool; Motivo string}`
  - `inbound.CrearMesa`, `inbound.EditarMesa`, `inbound.CambiarEstadoMesa`
  - `outbound.NuevaMesa`, `outbound.CampoOpcional`, `outbound.CambiosMesa`, `outbound.ZonaRef{SucursalID int64; EsActiva bool}`, `outbound.MesaStore`, `outbound.ErrNumeroMesaDuplicado`
  - `controller.NewMesasHandler(listar, obtener, crear, editar, estado)`

- [ ] **Step 1: DTO, puertos y pruebas unitarias (fallan)**

`application/dto/command/mesas.go`:

```go
package command

import appctx "service-restaurant/internal/shared/application/context"

// Campo es un valor opcional de un PATCH: Presente distingue "no enviado" de "enviado como
// null" (vaciar). Valor es nil cuando se envió null.
type Campo[T any] struct {
	Presente bool
	Valor    *T
}

// CrearMesa da de alta una mesa. Habilitada nil = true. Las posiciones son decimales exactos.
type CrearMesa struct {
	Intencion
	Actor      appctx.Actor
	SucursalID int64
	ZonaID     int64
	NumeroMesa string
	Capacidad  int
	Habilitada *bool
	PosicionX  *string
	PosicionY  *string
}

// EditarMesa aplica solo los campos presentes. La sede de la mesa no se edita; la zona puede
// cambiar dentro de la misma sede.
type EditarMesa struct {
	Actor      appctx.Actor
	ID         int64
	ZonaID     *int64
	NumeroMesa *string
	Capacidad  *int
	Habilitada *bool
	PosicionX  Campo[string]
	PosicionY  Campo[string]
}

// CambiarEstadoMesa da de baja (Activo=false, exige Motivo) o restaura (Activo=true).
type CambiarEstadoMesa struct {
	Intencion
	Actor  appctx.Actor
	ID     int64
	Activo bool
	Motivo string
}
```

Añadir a `application/port/inbound/mesas.go` (importar además `cmd "service-restaurant/internal/modules/atencion/application/dto/command"`):

```go
type CrearMesa interface {
	Execute(context.Context, cmd.CrearMesa) (appresult.Result[dtoresult.Mesa], error)
}
type EditarMesa interface {
	Execute(context.Context, cmd.EditarMesa) (appresult.Result[dtoresult.Mesa], error)
}
type CambiarEstadoMesa interface {
	Execute(context.Context, cmd.CambiarEstadoMesa) (appresult.Result[dtoresult.Mesa], error)
}
```

Añadir a `application/port/outbound/mesas.go` (importar además `"fmt"`):

```go
// ErrNumeroMesaDuplicado distingue la colisión del número de mesa dentro de la sede.
var ErrNumeroMesaDuplicado = fmt.Errorf("número de mesa duplicado: %w", ErrDuplicado)

type NuevaMesa struct {
	SucursalID, ZonaID  int64
	NumeroMesa          string
	Capacidad           int
	Habilitada          bool
	PosicionX, PosicionY *string
}

// CampoOpcional es un cambio de un campo anulable: si Presente, se escribe Valor (nil = NULL).
type CampoOpcional struct {
	Presente bool
	Valor    *string
}

// CambiosMesa: solo los campos presentes se modifican.
type CambiosMesa struct {
	ZonaID               *int64
	NumeroMesa           *string
	Capacidad            *int
	Habilitada           *bool
	PosicionX, PosicionY CampoOpcional
}

// ZonaRef resume la zona que una mesa referencia.
type ZonaRef struct {
	SucursalID int64
	EsActiva   bool
}

// MesaStore escribe mesas dentro de la transacción del comando.
type MesaStore interface {
	Crear(ctx context.Context, n NuevaMesa) (dtoresult.Mesa, error)
	// Obtener lee la mesa bloqueando su fila (FOR UPDATE); found=false si no existe.
	Obtener(ctx context.Context, id int64) (dtoresult.Mesa, bool, error)
	// ZonaDeMesa lee la zona bloqueándola en modo compartido, para que no se desactive durante el comando.
	ZonaDeMesa(ctx context.Context, zonaID int64) (ZonaRef, bool, error)
	Actualizar(ctx context.Context, id int64, c CambiosMesa) (dtoresult.Mesa, bool, error)
	DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Mesa, bool, error)
	Restaurar(ctx context.Context, id int64) (dtoresult.Mesa, bool, error)
}
```

`application/usecase/command/mesas_test.go`:

```go
package command

import (
	"context"
	"errors"
	"testing"

	cmd "service-restaurant/internal/modules/atencion/application/dto/command"
	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/shared/kernel/apperror"
	appctx "service-restaurant/internal/shared/application/context"
)

type mesaStoreFake struct {
	existente  dtoresult.Mesa
	zona       outbound.ZonaRef
	zonaExiste bool
	errCrear   error
	errBaja    error
	creada     outbound.NuevaMesa
	cambios    outbound.CambiosMesa
	llamadas   int
	bajas      int
}

func (f *mesaStoreFake) Crear(_ context.Context, n outbound.NuevaMesa) (dtoresult.Mesa, error) {
	f.llamadas++
	f.creada = n
	return dtoresult.Mesa{ID: 3, SucursalID: n.SucursalID, ZonaID: n.ZonaID, NumeroMesa: n.NumeroMesa, EsActivo: true}, f.errCrear
}
func (f *mesaStoreFake) Obtener(_ context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return f.existente, f.existente.ID == id, nil
}
func (f *mesaStoreFake) ZonaDeMesa(context.Context, int64) (outbound.ZonaRef, bool, error) {
	return f.zona, f.zonaExiste, nil
}
func (f *mesaStoreFake) Actualizar(_ context.Context, _ int64, c outbound.CambiosMesa) (dtoresult.Mesa, bool, error) {
	f.llamadas++
	f.cambios = c
	return f.existente, true, nil
}
func (f *mesaStoreFake) DarBaja(_ context.Context, _ int64, _ string) (dtoresult.Mesa, bool, error) {
	f.bajas++
	m := f.existente
	m.EsActivo = false
	return m, true, f.errBaja
}
func (f *mesaStoreFake) Restaurar(_ context.Context, _ int64) (dtoresult.Mesa, bool, error) {
	f.llamadas++
	m := f.existente
	m.EsActivo = true
	return m, true, nil
}

func booleano(b bool) *bool { return &b }

func TestCrearMesa(t *testing.T) {
	zonaOk := outbound.ZonaRef{SucursalID: 1, EsActiva: true}
	ok := cmd.CrearMesa{Actor: gestorDe(1), SucursalID: 1, ZonaID: 8, NumeroMesa: " A-1 ", Capacidad: 4, PosicionX: texto(" 12.50 ")}

	f := &mesaStoreFake{zona: zonaOk, zonaExiste: true}
	res, err := NewCrearMesa(f).Execute(context.Background(), ok)
	if err != nil || !res.IsOk() {
		t.Fatalf("debía crearse: %v %+v", err, res.Err())
	}
	if f.creada.NumeroMesa != "A-1" || f.creada.Capacidad != 4 || !f.creada.Habilitada ||
		f.creada.PosicionX == nil || *f.creada.PosicionX != "12.50" || f.creada.PosicionY != nil {
		t.Fatalf("no normalizó: %+v", f.creada)
	}
	deshabilitada := ok
	deshabilitada.Habilitada = booleano(false)
	_, _ = NewCrearMesa(f).Execute(context.Background(), deshabilitada)
	if f.creada.Habilitada {
		t.Error("habilitada=false debe respetarse")
	}

	casos := []struct {
		nombre string
		mod    func(c *cmd.CrearMesa)
		zona   outbound.ZonaRef
		existe bool
		kind   apperror.Kind
		code   string
	}{
		{"sin sesión", func(c *cmd.CrearMesa) { c.Actor = appctx.Actor{} }, zonaOk, true, apperror.KindUnauthorized, "no_autenticado"},
		{"solo lectura", func(c *cmd.CrearMesa) { c.Actor = soloVer(1) }, zonaOk, true, apperror.KindForbidden, "sin_permiso"},
		{"sede ajena", func(c *cmd.CrearMesa) { c.SucursalID = 9 }, zonaOk, true, apperror.KindNotFound, "no_encontrado"},
		{"número en blanco", func(c *cmd.CrearMesa) { c.NumeroMesa = " " }, zonaOk, true, apperror.KindValidation, "numero_mesa_invalido"},
		{"capacidad cero", func(c *cmd.CrearMesa) { c.Capacidad = 0 }, zonaOk, true, apperror.KindValidation, "capacidad_invalida"},
		{"posición x inválida", func(c *cmd.CrearMesa) { c.PosicionX = texto("abc") }, zonaOk, true, apperror.KindValidation, "posicion_x_invalida"},
		{"posición y inválida", func(c *cmd.CrearMesa) { c.PosicionY = texto("1.234") }, zonaOk, true, apperror.KindValidation, "posicion_y_invalida"},
		{"zona inexistente", func(c *cmd.CrearMesa) {}, zonaOk, false, apperror.KindValidation, "zona_invalida"},
		{"zona de otra sede", func(c *cmd.CrearMesa) {}, outbound.ZonaRef{SucursalID: 2, EsActiva: true}, true, apperror.KindValidation, "zona_invalida"},
		{"zona inactiva", func(c *cmd.CrearMesa) {}, outbound.ZonaRef{SucursalID: 1, EsActiva: false}, true, apperror.KindValidation, "zona_invalida"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			in := ok
			c.mod(&in)
			f := &mesaStoreFake{zona: c.zona, zonaExiste: c.existe}
			res, err := NewCrearMesa(f).Execute(context.Background(), in)
			if err != nil || res.Err() == nil {
				t.Fatalf("debía rechazarse: %v", err)
			}
			esperarRechazo(t, res.Err(), c.kind, c.code)
			if f.llamadas != 0 {
				t.Fatal("un rechazo no debe escribir")
			}
		})
	}
	t.Run("número duplicado", func(t *testing.T) {
		res, _ := NewCrearMesa(&mesaStoreFake{zona: zonaOk, zonaExiste: true, errCrear: outbound.ErrNumeroMesaDuplicado}).Execute(context.Background(), ok)
		esperarRechazo(t, res.Err(), apperror.KindConflict, "numero_mesa_duplicado")
	})
	t.Run("error técnico", func(t *testing.T) {
		if _, err := NewCrearMesa(&mesaStoreFake{zona: zonaOk, zonaExiste: true, errCrear: errors.New("conexión perdida")}).Execute(context.Background(), ok); err == nil {
			t.Fatal("un error técnico debe viajar por el canal error")
		}
	})
}

func TestEditarMesa(t *testing.T) {
	activa := dtoresult.Mesa{ID: 5, SucursalID: 1, ZonaID: 8, EsActivo: true}
	zonaOk := outbound.ZonaRef{SucursalID: 1, EsActiva: true}
	nueva := int64(9)

	res, _ := NewEditarMesa(&mesaStoreFake{existente: activa}).Execute(context.Background(),
		cmd.EditarMesa{Actor: gestorDe(9), ID: 5, Capacidad: entero(2)})
	esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	res, _ = NewEditarMesa(&mesaStoreFake{existente: dtoresult.Mesa{ID: 5, SucursalID: 1}}).Execute(context.Background(),
		cmd.EditarMesa{Actor: gestorDe(1), ID: 5, Habilitada: booleano(true)})
	esperarRechazo(t, res.Err(), apperror.KindConflict, "mesa_inactiva")

	f := &mesaStoreFake{existente: activa, zona: zonaOk, zonaExiste: true}
	res, err := NewEditarMesa(f).Execute(context.Background(), cmd.EditarMesa{
		Actor: gestorDe(1), ID: 5, ZonaID: &nueva, NumeroMesa: texto(" B-2 "), Capacidad: entero(6), Habilitada: booleano(false),
		PosicionX: cmd.Campo[string]{Presente: true}, PosicionY: cmd.Campo[string]{Presente: true, Valor: texto("3.5")}})
	if err != nil || !res.IsOk() {
		t.Fatalf("debía editarse: %v %+v", err, res.Err())
	}
	c := f.cambios
	if *c.ZonaID != 9 || *c.NumeroMesa != "B-2" || *c.Capacidad != 6 || *c.Habilitada ||
		!c.PosicionX.Presente || c.PosicionX.Valor != nil || !c.PosicionY.Presente || *c.PosicionY.Valor != "3.5" {
		t.Fatalf("cambios: %+v", c)
	}

	res, _ = NewEditarMesa(f).Execute(context.Background(), cmd.EditarMesa{Actor: gestorDe(1), ID: 5})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "sin_cambios")
	res, _ = NewEditarMesa(f).Execute(context.Background(), cmd.EditarMesa{Actor: gestorDe(1), ID: 5, Capacidad: entero(0)})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "capacidad_invalida")
	otraSede := &mesaStoreFake{existente: activa, zona: outbound.ZonaRef{SucursalID: 2, EsActiva: true}, zonaExiste: true}
	res, _ = NewEditarMesa(otraSede).Execute(context.Background(), cmd.EditarMesa{Actor: gestorDe(1), ID: 5, ZonaID: &nueva})
	esperarRechazo(t, res.Err(), apperror.KindValidation, "zona_invalida")
	if otraSede.llamadas != 0 {
		t.Fatal("una zona inválida no debe escribir")
	}
}

func TestCambiarEstadoMesa(t *testing.T) {
	activa := dtoresult.Mesa{ID: 5, SucursalID: 1, EsActivo: true}
	baja := func(a appctx.Actor, motivo string) cmd.CambiarEstadoMesa {
		return cmd.CambiarEstadoMesa{Actor: a, ID: 5, Activo: false, Motivo: motivo}
	}

	t.Run("exige motivo", func(t *testing.T) {
		f := &mesaStoreFake{existente: activa}
		res, _ := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "  "))
		esperarRechazo(t, res.Err(), apperror.KindValidation, "motivo_requerido")
		if f.bajas != 0 {
			t.Fatal("no debe dar de baja sin motivo")
		}
	})
	t.Run("la sede ajena se resuelve antes que el motivo", func(t *testing.T) {
		res, _ := NewCambiarEstadoMesa(&mesaStoreFake{existente: activa}).Execute(context.Background(), baja(gestorDe(9), ""))
		esperarRechazo(t, res.Err(), apperror.KindNotFound, "no_encontrado")
	})
	t.Run("baja correcta", func(t *testing.T) {
		f := &mesaStoreFake{existente: activa}
		res, err := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "Mesa dañada"))
		if err != nil || !res.IsOk() || res.Value().EsActivo || f.bajas != 1 {
			t.Fatalf("baja: %v %+v", err, res.Err())
		}
	})
	t.Run("regla de la base (pedido abierto)", func(t *testing.T) {
		f := &mesaStoreFake{existente: activa, errBaja: &outbound.ErrReglaDatos{Mensaje: "Cerrar el pedido de la mesa antes de darla de baja"}}
		res, err := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "Mesa dañada"))
		if err != nil {
			t.Fatalf("una regla de la base es un rechazo, no un error técnico: %v", err)
		}
		esperarRechazo(t, res.Err(), apperror.KindBusinessRule, "regla_de_datos")
	})
	t.Run("ya estaba de baja", func(t *testing.T) {
		f := &mesaStoreFake{existente: dtoresult.Mesa{ID: 5, SucursalID: 1}}
		res, _ := NewCambiarEstadoMesa(f).Execute(context.Background(), baja(gestorDe(1), "Otra vez"))
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_dada_de_baja")
	})
	t.Run("restaurar", func(t *testing.T) {
		f := &mesaStoreFake{existente: dtoresult.Mesa{ID: 5, SucursalID: 1}}
		res, err := NewCambiarEstadoMesa(f).Execute(context.Background(), cmd.CambiarEstadoMesa{Actor: gestorDe(1), ID: 5, Activo: true})
		if err != nil || !res.IsOk() || !res.Value().EsActivo {
			t.Fatalf("restaurar: %v %+v", err, res.Err())
		}
		res, _ = NewCambiarEstadoMesa(&mesaStoreFake{existente: activa}).Execute(context.Background(), cmd.CambiarEstadoMesa{Actor: gestorDe(1), ID: 5, Activo: true})
		esperarRechazo(t, res.Err(), apperror.KindConflict, "ya_activa")
	})
}
```

(`appctx` en este archivo se usa en `appctx.Actor{}` y `appctx.Actor` del helper `baja`; los imports de ambos paquetes `apperror` y `appctx` van en el bloque mostrado.)

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `go test ./internal/modules/atencion/application/usecase/command/ 2>&1 | head`
Expected: FAIL de compilación (`undefined: NewCrearMesa`).

- [ ] **Step 3: Casos de uso**

`application/usecase/command/mesas.go`:

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
	appctx "service-restaurant/internal/shared/application/context"
	"service-restaurant/internal/shared/kernel/apperror"
	appresult "service-restaurant/internal/shared/kernel/result"
)

const (
	maxMotivo = 2000

	CodigoNumeroMesaDuplicado = "numero_mesa_duplicado"
	CodigoZonaInvalida        = "zona_invalida"
	CodigoMesaInactiva        = "mesa_inactiva"
)

// deMesa traduce el número duplicado a un rechazo específico y delega lo demás.
func deMesa(err error, contexto string) (appresult.Result[dtoresult.Mesa], error) {
	if errors.Is(err, outbound.ErrNumeroMesaDuplicado) {
		return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoNumeroMesaDuplicado, "ya existe una mesa con ese número en la sucursal"))
	}
	return deAlmacen[dtoresult.Mesa](contexto, err)
}

func zonaInvalida() *apperror.AppError {
	return mapper.Validacion(CodigoZonaInvalida, "la zona no existe, está dada de baja o pertenece a otra sucursal")
}

// zonaValida comprueba que la zona exista, esté activa y sea de la sede indicada.
func zonaValida(ctx context.Context, store outbound.MesaStore, zonaID, sedeID int64) (bool, error) {
	z, found, err := store.ZonaDeMesa(ctx, zonaID)
	if err != nil {
		return false, err
	}
	return found && z.EsActiva && z.SucursalID == sedeID, nil
}

// normPosicion valida un decimal de numeric(10,2) y lo devuelve normalizado.
func normPosicion(s string) (string, bool) {
	p, err := vo.NewPosicion(s)
	return p.String(), err == nil
}

// posicionOpcional valida una posición anulable de un PATCH: ausente → no se toca; null o en
// blanco → se vacía; con valor → se valida.
func posicionOpcional(c cmd.Campo[string]) (outbound.CampoOpcional, bool) {
	if !c.Presente {
		return outbound.CampoOpcional{}, true
	}
	if c.Valor == nil || *c.Valor == "" {
		return outbound.CampoOpcional{Presente: true}, true
	}
	v, ok := normPosicion(*c.Valor)
	if !ok {
		return outbound.CampoOpcional{}, false
	}
	return outbound.CampoOpcional{Presente: true, Valor: &v}, true
}

// CrearMesa da de alta una mesa en una sede donde el actor administra mesas (RES-RF-008).
type CrearMesa struct{ mesas outbound.MesaStore }

func NewCrearMesa(m outbound.MesaStore) *CrearMesa { return &CrearMesa{m} }

func (uc *CrearMesa) Execute(ctx context.Context, c cmd.CrearMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	if ae := mapper.AutorizarSede(c.Actor, c.SucursalID); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	numero, err := vo.NewNumeroMesa(c.NumeroMesa)
	if err != nil {
		return rechazar[dtoresult.Mesa](mapper.Validacion("numero_mesa_invalido", "el número de mesa es obligatorio (máximo 20 caracteres)"))
	}
	capacidad, err := vo.NewCapacidad(c.Capacidad)
	if err != nil {
		return rechazar[dtoresult.Mesa](mapper.Validacion("capacidad_invalida", "la capacidad debe estar entre 1 y 500"))
	}
	px, ok := posicionOpcional(cmd.Campo[string]{Presente: c.PosicionX != nil, Valor: c.PosicionX})
	if !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_x_invalida", "la posición X debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	py, ok := posicionOpcional(cmd.Campo[string]{Presente: c.PosicionY != nil, Valor: c.PosicionY})
	if !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_y_invalida", "la posición Y debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	valida, err := zonaValida(ctx, uc.mesas, c.ZonaID, c.SucursalID)
	if err != nil {
		return tecnico[dtoresult.Mesa]("buscar zona", err)
	}
	if !valida {
		return rechazar[dtoresult.Mesa](zonaInvalida())
	}
	habilitada := true
	if c.Habilitada != nil {
		habilitada = *c.Habilitada
	}
	m, err := uc.mesas.Crear(ctx, outbound.NuevaMesa{
		SucursalID: c.SucursalID, ZonaID: c.ZonaID, NumeroMesa: numero.String(), Capacidad: capacidad.Int(),
		Habilitada: habilitada, PosicionX: px.Valor, PosicionY: py.Valor,
	})
	if err != nil {
		return deMesa(err, "crear mesa")
	}
	return appresult.Ok(m), nil
}

// mesaGestionable carga la mesa (bloqueándola) y verifica que el actor ve su sede (si no, 404) y
// que la administra (si no, 403). Devuelve la mesa o el rechazo; un error técnico va en err.
func mesaGestionable(ctx context.Context, store outbound.MesaStore, actor appctx.Actor, id int64) (dtoresult.Mesa, *apperror.AppError, error) {
	m, found, err := store.Obtener(ctx, id)
	if err != nil {
		return dtoresult.Mesa{}, nil, err
	}
	if !found {
		return dtoresult.Mesa{}, mapper.NoEncontrado("mesa"), nil
	}
	if ae := mapper.AutorizarSede(actor, m.SucursalID); ae != nil {
		return dtoresult.Mesa{}, ae, nil
	}
	return m, nil, nil
}

// EditarMesa modifica solo los campos enviados de una mesa activa. Habilitar o deshabilitar no
// pide motivo.
type EditarMesa struct{ mesas outbound.MesaStore }

func NewEditarMesa(m outbound.MesaStore) *EditarMesa { return &EditarMesa{m} }

func (uc *EditarMesa) Execute(ctx context.Context, c cmd.EditarMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	actual, ae, err := mesaGestionable(ctx, uc.mesas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Mesa]("buscar mesa", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoMesaInactiva, "la mesa está dada de baja: restáurela para editarla"))
	}
	if c.ZonaID == nil && c.NumeroMesa == nil && c.Capacidad == nil && c.Habilitada == nil &&
		!c.PosicionX.Presente && !c.PosicionY.Presente {
		return rechazar[dtoresult.Mesa](mapper.Validacion("sin_cambios", "no se indicó ningún campo a modificar"))
	}
	cambios := outbound.CambiosMesa{ZonaID: c.ZonaID, Habilitada: c.Habilitada}
	if c.NumeroMesa != nil {
		n, err := vo.NewNumeroMesa(*c.NumeroMesa)
		if err != nil {
			return rechazar[dtoresult.Mesa](mapper.Validacion("numero_mesa_invalido", "el número de mesa es obligatorio (máximo 20 caracteres)"))
		}
		v := n.String()
		cambios.NumeroMesa = &v
	}
	if c.Capacidad != nil {
		capac, err := vo.NewCapacidad(*c.Capacidad)
		if err != nil {
			return rechazar[dtoresult.Mesa](mapper.Validacion("capacidad_invalida", "la capacidad debe estar entre 1 y 500"))
		}
		v := capac.Int()
		cambios.Capacidad = &v
	}
	var ok bool
	if cambios.PosicionX, ok = posicionOpcional(c.PosicionX); !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_x_invalida", "la posición X debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	if cambios.PosicionY, ok = posicionOpcional(c.PosicionY); !ok {
		return rechazar[dtoresult.Mesa](mapper.Validacion("posicion_y_invalida", "la posición Y debe ser un decimal con hasta 8 enteros y 2 decimales"))
	}
	if c.ZonaID != nil {
		valida, err := zonaValida(ctx, uc.mesas, *c.ZonaID, actual.SucursalID)
		if err != nil {
			return tecnico[dtoresult.Mesa]("buscar zona", err)
		}
		if !valida {
			return rechazar[dtoresult.Mesa](zonaInvalida())
		}
	}
	m, found, err := uc.mesas.Actualizar(ctx, c.ID, cambios)
	if err != nil {
		return deMesa(err, "editar mesa")
	}
	if !found {
		return rechazar[dtoresult.Mesa](mapper.NoEncontrado("mesa"))
	}
	return appresult.Ok(m), nil
}

// CambiarEstadoMesa da de baja (con motivo) o restaura una mesa. La base impide dar de baja una
// mesa con pedido abierto y restaurar una mesa cuya zona está dada de baja; ambos casos llegan
// como regla_de_datos con el mensaje de la base.
type CambiarEstadoMesa struct{ mesas outbound.MesaStore }

func NewCambiarEstadoMesa(m outbound.MesaStore) *CambiarEstadoMesa { return &CambiarEstadoMesa{m} }

func (uc *CambiarEstadoMesa) Execute(ctx context.Context, c cmd.CambiarEstadoMesa) (appresult.Result[dtoresult.Mesa], error) {
	if ae := mapper.Requiere(c.Actor, mensajeSinPermisoAdmin, policy.MesaAdministrar); ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	actual, ae, err := mesaGestionable(ctx, uc.mesas, c.Actor, c.ID)
	if err != nil {
		return tecnico[dtoresult.Mesa]("buscar mesa", err)
	}
	if ae != nil {
		return rechazar[dtoresult.Mesa](ae)
	}
	if c.Activo {
		if actual.EsActivo {
			return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoYaActiva, "la mesa ya está activa"))
		}
		m, found, err := uc.mesas.Restaurar(ctx, c.ID)
		if err != nil {
			return deMesa(err, "restaurar mesa")
		}
		if !found {
			return rechazar[dtoresult.Mesa](mapper.NoEncontrado("mesa"))
		}
		return appresult.Ok(m), nil
	}
	motivo, err := vo.TextoObligatorio(c.Motivo, maxMotivo)
	if err != nil {
		return rechazar[dtoresult.Mesa](mapper.Validacion("motivo_requerido", "la baja exige un motivo (máximo 2000 caracteres)"))
	}
	if !actual.EsActivo {
		return rechazar[dtoresult.Mesa](apperror.New(apperror.KindConflict, CodigoYaDadaDeBaja, "la mesa ya está dada de baja"))
	}
	m, found, err := uc.mesas.DarBaja(ctx, c.ID, motivo)
	if err != nil {
		return deMesa(err, "dar de baja la mesa")
	}
	if !found {
		return rechazar[dtoresult.Mesa](mapper.NoEncontrado("mesa"))
	}
	return appresult.Ok(m), nil
}
```

- [ ] **Step 4: Ejecutar las pruebas unitarias**

Run: `gofmt -l internal/ ; go vet ./internal/modules/atencion/... && go test ./internal/modules/atencion/... 2>&1 | tail`
Expected: PASS.

- [ ] **Step 5: Persistencia de escritura**

`infrastructure/persistence/write/repository/mesas_sql.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/platform/database"
)

// Las posiciones son numeric(10,2): se leen como texto y se escriben vía ::text::numeric para
// no perder exactitud ni depender de la codificación del driver.
const mesaCols = `id, sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x::text, posicion_y::text,
	es_activo, motivo_eliminacion, eliminado_en, eliminado_por, creado_en, creado_por, actualizado_en, actualizado_por`

func textoNulo(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func scanMesa(s scanner) (dtoresult.Mesa, error) {
	var m dtoresult.Mesa
	var px, py, motivo sql.NullString
	var elimEn sql.NullTime
	var elimPor sql.NullInt64
	err := s.Scan(&m.ID, &m.SucursalID, &m.ZonaID, &m.NumeroMesa, &m.Capacidad, &m.Habilitada, &px, &py,
		&m.EsActivo, &motivo, &elimEn, &elimPor, &m.CreadoEn, &m.CreadoPor, &m.ActualizadoEn, &m.ActualizadoPor)
	m.PosicionX, m.PosicionY, m.MotivoEliminacion = textoNulo(px), textoNulo(py), textoNulo(motivo)
	if elimEn.Valid {
		t := elimEn.Time
		m.EliminadoEn = &t
	}
	if elimPor.Valid {
		v := elimPor.Int64
		m.EliminadoPor = &v
	}
	return m, err
}

// traducirMesa distingue la colisión del número de mesa (mesas_sucursal_id_numero_mesa_key).
func traducirMesa(err error) error {
	err = Translate(err)
	if errors.Is(err, outbound.ErrDuplicado) && strings.Contains(err.Error(), "mesas_sucursal_id_numero_mesa_key") {
		return outbound.ErrNumeroMesaDuplicado
	}
	return err
}

// MesaSQLWriter opera sobre sch_restaurante.mesas.
type MesaSQLWriter struct{ db database.DBTX }

func NewMesaSQLWriter(db database.DBTX) *MesaSQLWriter { return &MesaSQLWriter{db} }

func (r *MesaSQLWriter) Insert(ctx context.Context, n outbound.NuevaMesa) (dtoresult.Mesa, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		INSERT INTO sch_restaurante.mesas (sucursal_id, zona_id, numero_mesa, capacidad, habilitada, posicion_x, posicion_y)
		VALUES ($1, $2, $3, $4, $5, $6::text::numeric, $7::text::numeric) RETURNING `+mesaCols,
		n.SucursalID, n.ZonaID, n.NumeroMesa, n.Capacidad, n.Habilitada, n.PosicionX, n.PosicionY))
	return m, traducirMesa(err)
}

// GetForUpdate lee la mesa bloqueando su fila hasta el final de la transacción.
func (r *MesaSQLWriter) GetForUpdate(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx,
		`SELECT `+mesaCols+` FROM sch_restaurante.mesas WHERE id = $1 FOR UPDATE`, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, err
	}
	return m, true, nil
}

// GetZona lee la zona en modo compartido: no puede desactivarse hasta que termine el comando.
func (r *MesaSQLWriter) GetZona(ctx context.Context, zonaID int64) (outbound.ZonaRef, bool, error) {
	var z outbound.ZonaRef
	err := r.db.QueryRowContext(ctx,
		`SELECT sucursal_id, es_activo FROM sch_restaurante.zonas WHERE id = $1 FOR SHARE`, zonaID).Scan(&z.SucursalID, &z.EsActiva)
	if missing, err := notFound(err); missing || err != nil {
		return outbound.ZonaRef{}, false, err
	}
	return z, true, nil
}

func (r *MesaSQLWriter) Update(ctx context.Context, id int64, c outbound.CambiosMesa) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET
			zona_id = COALESCE($2, zona_id),
			numero_mesa = COALESCE($3, numero_mesa),
			capacidad = COALESCE($4, capacidad),
			habilitada = COALESCE($5, habilitada),
			posicion_x = CASE WHEN $6 THEN $7::text::numeric ELSE posicion_x END,
			posicion_y = CASE WHEN $8 THEN $9::text::numeric ELSE posicion_y END
		WHERE id = $1 RETURNING `+mesaCols, id, c.ZonaID, c.NumeroMesa, c.Capacidad, c.Habilitada,
		c.PosicionX.Presente, c.PosicionX.Valor, c.PosicionY.Presente, c.PosicionY.Valor))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}

// SetBaja desactiva la mesa; el trigger de metadatos completa actor y fecha de la baja.
func (r *MesaSQLWriter) SetBaja(ctx context.Context, id int64, motivo string) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET es_activo = false, motivo_eliminacion = $2
		WHERE id = $1 RETURNING `+mesaCols, id, motivo))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}

// SetActiva reactiva la misma fila (conserva el ID); el trigger limpia los marcadores de baja.
func (r *MesaSQLWriter) SetActiva(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	m, err := scanMesa(r.db.QueryRowContext(ctx, `
		UPDATE sch_restaurante.mesas SET es_activo = true WHERE id = $1 RETURNING `+mesaCols, id))
	if missing, err := notFound(err); missing || err != nil {
		return dtoresult.Mesa{}, false, traducirMesa(err)
	}
	return m, true, nil
}
```

`infrastructure/persistence/write/adapter/mesas.go`:

```go
package adapter

import (
	"context"

	dtoresult "service-restaurant/internal/modules/atencion/application/dto/result"
	"service-restaurant/internal/modules/atencion/application/port/outbound"
	"service-restaurant/internal/modules/atencion/infrastructure/persistence/write/repository"
	"service-restaurant/internal/platform/database"
)

// MesaWriteAdapter implementa outbound.MesaStore.
type MesaWriteAdapter struct{ repo *repository.MesaSQLWriter }

func NewMesaWriteAdapter(db database.DBTX) *MesaWriteAdapter {
	return &MesaWriteAdapter{repo: repository.NewMesaSQLWriter(db)}
}

var _ outbound.MesaStore = (*MesaWriteAdapter)(nil)

func (a *MesaWriteAdapter) Crear(ctx context.Context, n outbound.NuevaMesa) (dtoresult.Mesa, error) {
	return a.repo.Insert(ctx, n)
}
func (a *MesaWriteAdapter) Obtener(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return a.repo.GetForUpdate(ctx, id)
}
func (a *MesaWriteAdapter) ZonaDeMesa(ctx context.Context, zonaID int64) (outbound.ZonaRef, bool, error) {
	return a.repo.GetZona(ctx, zonaID)
}
func (a *MesaWriteAdapter) Actualizar(ctx context.Context, id int64, c outbound.CambiosMesa) (dtoresult.Mesa, bool, error) {
	return a.repo.Update(ctx, id, c)
}
func (a *MesaWriteAdapter) DarBaja(ctx context.Context, id int64, motivo string) (dtoresult.Mesa, bool, error) {
	return a.repo.SetBaja(ctx, id, motivo)
}
func (a *MesaWriteAdapter) Restaurar(ctx context.Context, id int64) (dtoresult.Mesa, bool, error) {
	return a.repo.SetActiva(ctx, id)
}
```

- [ ] **Step 6: Capa web**

`web/dto/request/mesas.go`:

```go
package request

// MesaAlta: los ids viajan como cadenas (contrato); las posiciones, como decimales exactos en
// cadena. Un campo desconocido responde 400.
type MesaAlta struct {
	SucursalID string  `json:"sucursal_id"`
	ZonaID     string  `json:"zona_id"`
	NumeroMesa string  `json:"numero_mesa"`
	Capacidad  int     `json:"capacidad"`
	Habilitada *bool   `json:"habilitada"`
	PosicionX  *string `json:"posicion_x"`
	PosicionY  *string `json:"posicion_y"`
}

// MesaEdicion: la sede de una mesa no se edita (sucursal_id responde 400).
type MesaEdicion struct {
	ZonaID     *string          `json:"zona_id,omitempty"`
	NumeroMesa *string          `json:"numero_mesa,omitempty"`
	Capacidad  *int             `json:"capacidad,omitempty"`
	Habilitada *bool            `json:"habilitada,omitempty"`
	PosicionX  Opcional[string] `json:"posicion_x"`
	PosicionY  Opcional[string] `json:"posicion_y"`
}
```

Reemplazar `web/controller/mesas_handler.go` completo:

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

// MesasHandler: mesas del salón (RES-RF-008).
type MesasHandler struct {
	listar  inbound.ListarMesas
	obtener inbound.ObtenerMesa
	crear   inbound.CrearMesa
	editar  inbound.EditarMesa
	estado  inbound.CambiarEstadoMesa
}

func NewMesasHandler(l inbound.ListarMesas, o inbound.ObtenerMesa, c inbound.CrearMesa,
	e inbound.EditarMesa, s inbound.CambiarEstadoMesa) *MesasHandler {
	return &MesasHandler{l, o, c, e, s}
}

// Listar: GET /mesas?sucursal_id=&es_activo=&limit=&cursor=
func (h *MesasHandler) Listar(w http.ResponseWriter, r *http.Request) error {
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
	sede, err := sucursalDe(q)
	if err != nil {
		return err
	}
	res, err := h.listar.Execute(r.Context(), query.ListarMesas{
		Actor: actor, SucursalID: sede, Activo: activo, Limite: limite, Cursor: q.Get("cursor")})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesasPagina)
}

// Obtener: GET /mesas/{id}
func (h *MesasHandler) Obtener(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	res, err := h.obtener.Execute(r.Context(), query.ObtenerMesa{Actor: actor, ID: id})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}

// Crear: POST /mesas (Idempotency-Key obligatoria)
func (h *MesasHandler) Crear(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	var body request.MesaAlta
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	sede, err := idCadena(body.SucursalID, "sucursal_id")
	if err != nil {
		return err
	}
	zona, err := idCadena(body.ZonaID, "zona_id")
	if err != nil {
		return err
	}
	in, err := intencion(r, "POST /mesas", body)
	if err != nil {
		return err
	}
	res, err := h.crear.Execute(r.Context(), command.CrearMesa{
		Intencion: in, Actor: actor, SucursalID: sede, ZonaID: zona, NumeroMesa: body.NumeroMesa,
		Capacidad: body.Capacidad, Habilitada: body.Habilitada, PosicionX: body.PosicionX, PosicionY: body.PosicionY,
	})
	return responder(w, res, err, http.StatusCreated, webmapper.ToMesa)
}

// Editar: PATCH /mesas/{id}
func (h *MesasHandler) Editar(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorDe(r)
	if err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var body request.MesaEdicion
	if err := webrequest.DecodeJSON(r, &body); err != nil {
		return err
	}
	var zona *int64
	if body.ZonaID != nil {
		z, err := idCadena(*body.ZonaID, "zona_id")
		if err != nil {
			return err
		}
		zona = &z
	}
	res, err := h.editar.Execute(r.Context(), command.EditarMesa{
		Actor: actor, ID: id, ZonaID: zona, NumeroMesa: body.NumeroMesa, Capacidad: body.Capacidad, Habilitada: body.Habilitada,
		PosicionX: command.Campo[string]{Presente: body.PosicionX.Presente, Valor: body.PosicionX.Valor},
		PosicionY: command.Campo[string]{Presente: body.PosicionY.Presente, Valor: body.PosicionY.Valor},
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}

// Baja: POST /mesas/{id}/baja con {"motivo": "..."} (Idempotency-Key obligatoria)
func (h *MesasHandler) Baja(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /mesas/%d/baja", id), body)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoMesa{
		Intencion: in, Actor: actor, ID: id, Activo: false, Motivo: body.Motivo,
	})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}

// Restauracion: POST /mesas/{id}/restauracion (Idempotency-Key obligatoria)
func (h *MesasHandler) Restauracion(w http.ResponseWriter, r *http.Request) error {
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
	in, err := intencion(r, fmt.Sprintf("POST /mesas/%d/restauracion", id), nil)
	if err != nil {
		return err
	}
	res, err := h.estado.Execute(r.Context(), command.CambiarEstadoMesa{Intencion: in, Actor: actor, ID: id, Activo: true})
	return responder(w, res, err, http.StatusOK, webmapper.ToMesa)
}
```

Añadir a `RegisterRoutes` en `web/controller/routes.go`:

```go
	ruta("POST /mesas", mesas.Crear)
	ruta("PATCH /mesas/{id}", mesas.Editar)
	ruta("POST /mesas/{id}/baja", mesas.Baja)
	ruta("POST /mesas/{id}/restauracion", mesas.Restauracion)
```

- [ ] **Step 7: Composición**

Reemplazar `infrastructure/config/mesas.go`:

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

// armarMesas compone los casos de uso de mesas, ya decorados con transacción e idempotencia.
func armarMesas(db *sql.DB, logger *slog.Logger) *controller.MesasHandler {
	crear := escritura[cmd.CrearMesa, dtoresult.Mesa]{db: db, logger: logger, status: http.StatusCreated,
		build: func(q txQuerier) ejecutor[cmd.CrearMesa, dtoresult.Mesa] {
			return command.NewCrearMesa(writeadapter.NewMesaWriteAdapter(q))
		},
		meta: func(c cmd.CrearMesa) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}
	editar := escritura[cmd.EditarMesa, dtoresult.Mesa]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.EditarMesa, dtoresult.Mesa] {
			return command.NewEditarMesa(writeadapter.NewMesaWriteAdapter(q))
		},
		meta: func(c cmd.EditarMesa) (int64, cmd.Intencion) { return c.Actor.UserID, cmd.Intencion{} }}
	estado := escritura[cmd.CambiarEstadoMesa, dtoresult.Mesa]{db: db, logger: logger, status: http.StatusOK,
		build: func(q txQuerier) ejecutor[cmd.CambiarEstadoMesa, dtoresult.Mesa] {
			return command.NewCambiarEstadoMesa(writeadapter.NewMesaWriteAdapter(q))
		},
		meta: func(c cmd.CambiarEstadoMesa) (int64, cmd.Intencion) { return c.Actor.UserID, c.Intencion }}

	lector := readadapter.NewMesaReadAdapter(db)
	return controller.NewMesasHandler(uquery.NewListarMesas(lector), uquery.NewObtenerMesa(lector),
		inbound.CrearMesa(crear), inbound.EditarMesa(editar), inbound.CambiarEstadoMesa(estado))
}
```

En `infrastructure/config/module.go` cambiar la llamada a `armarMesas(d.DB, d.Logger)`.

- [ ] **Step 8: Compilar**

Run: `go build ./... && go vet ./... && go test ./tests/architecture/ ./internal/...`
Expected: sin errores.

- [ ] **Step 9: Pruebas de integración de escritura de mesas**

Añadir a `tests/integration/atencion_mesas_test.go` (importar además `"fmt"`):

```go
func altaMesa(sucursalID, zonaID, numero string, capacidad int) string {
	return fmt.Sprintf(`{"sucursal_id":"%s","zona_id":"%s","numero_mesa":"%s","capacidad":%d,"posicion_x":"12.50"}`,
		sucursalID, zonaID, numero, capacidad)
}

func TestMesaCrearConIdempotenciaYValidaciones(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Zona crear mesa")
	crearZonaSQL(t, "MESA-AJENA", "Zona crear ajena")
	crearZonaSQL(t, "PRINCIPAL", "Zona crear inactiva")
	asSystem(t, `UPDATE sch_restaurante.zonas SET es_activo=false WHERE nombre='Zona crear inactiva'`)
	tk := gestorMesas(t)
	principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
	zona := idDe(t, "zonas", "nombre", "Zona crear mesa")
	h := map[string]string{"Idempotency-Key": nuevaClave()}

	r1 := callH(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, "NM-1", 4), tk, h)
	esperar(t, r1, http.StatusCreated, "")
	var m mesaResp
	r1.json(t, &m)
	if m.NumeroMesa != "NM-1" || m.Capacidad != 4 || !m.Habilitada || !m.EsActivo || m.ZonaID != zona || m.SucursalID != principal ||
		m.PosicionX == nil || *m.PosicionX != "12.50" || m.PosicionY != nil || m.CreadoPor != idDe(t, "usuarios", "username", "mesa.gestor") {
		t.Fatalf("mesa creada: %+v", m)
	}
	r2 := callH(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, "NM-1", 4), tk, h)
	esperar(t, r2, http.StatusCreated, "")
	var rep mesaResp
	r2.json(t, &rep)
	if rep.ID != m.ID || scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE numero_mesa='NM-1'`) != 1 {
		t.Fatalf("la repetición debe devolver la misma mesa: %+v vs %+v", rep, m)
	}
	esperar(t, callH(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, "NM-9", 2), tk, h), http.StatusConflict, "idempotency_key_reutilizada")

	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, "NM-1", 2), tk), http.StatusConflict, "numero_mesa_duplicado")
	esperar(t, call(t, "POST", "/api/v1/mesas", altaMesa(principal, zona, "NM-2", 2), tk), http.StatusBadRequest, "idempotency_key_requerida")
	esperar(t, post(t, "/api/v1/mesas", `{"sucursal_id":"`+principal+`","zona_id":"`+zona+`","numero_mesa":"NM-3","capacidad":2,"creado_por":"1"}`, tk),
		http.StatusBadRequest, "json_invalido")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, "abc", "NM-4", 2), tk), http.StatusBadRequest, "zona_id_invalido")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, " ", 2), tk), http.StatusUnprocessableEntity, "numero_mesa_invalido")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, "NM-5", 0), tk), http.StatusUnprocessableEntity, "capacidad_invalida")
	esperar(t, post(t, "/api/v1/mesas", `{"sucursal_id":"`+principal+`","zona_id":"`+zona+`","numero_mesa":"NM-6","capacidad":2,"posicion_x":"1.234"}`, tk),
		http.StatusUnprocessableEntity, "posicion_x_invalida")
	// La zona debe existir, estar activa y ser de la misma sede.
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, idDe(t, "zonas", "nombre", "Zona crear ajena"), "NM-7", 2), tk), http.StatusUnprocessableEntity, "zona_invalida")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, idDe(t, "zonas", "nombre", "Zona crear inactiva"), "NM-8", 2), tk), http.StatusUnprocessableEntity, "zona_invalida")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, "999999", "NM-10", 2), tk), http.StatusUnprocessableEntity, "zona_invalida")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE numero_mesa IN ('NM-2','NM-3','NM-4','NM-5','NM-6','NM-7','NM-8','NM-10')`) != 0 {
		t.Fatal("los rechazos no deben dejar mesas")
	}

	ajena := idDe(t, "sucursales", "codigo", "MESA-AJENA")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(ajena, idDe(t, "zonas", "nombre", "Zona crear ajena"), "NM-11", 2), tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/mesas", altaMesa(principal, zona, "NM-12", 2), lectorMesas(t)), http.StatusForbidden, "sin_permiso")
}

func TestMesaEditarYHabilitar(t *testing.T) {
	crearSedeSQL(t, "MESA-AJENA")
	crearZonaSQL(t, "PRINCIPAL", "Zona editar A")
	crearZonaSQL(t, "PRINCIPAL", "Zona editar B")
	crearZonaSQL(t, "MESA-AJENA", "Zona editar ajena")
	crearMesaSQL(t, "PRINCIPAL", "Zona editar A", "ED-1", 4)
	crearMesaSQL(t, "PRINCIPAL", "Zona editar A", "ED-2", 2)
	crearMesaSQL(t, "MESA-AJENA", "Zona editar ajena", "ED-AJ", 2)
	tk := gestorMesas(t)
	id := idDe(t, "mesas", "numero_mesa", "ED-1")
	zonaB := idDe(t, "zonas", "nombre", "Zona editar B")

	// Habilitar o deshabilitar no pide motivo.
	r := callH(t, "PATCH", "/api/v1/mesas/"+id, `{"habilitada":false}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	var m mesaResp
	r.json(t, &m)
	if m.Habilitada || m.NumeroMesa != "ED-1" || m.Capacidad != 4 {
		t.Fatalf("deshabilitar: %+v", m)
	}
	r = callH(t, "PATCH", "/api/v1/mesas/"+id,
		fmt.Sprintf(`{"habilitada":true,"capacidad":6,"zona_id":"%s","posicion_x":"1.5","posicion_y":"-2"}`, zonaB), tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &m)
	if !m.Habilitada || m.Capacidad != 6 || m.ZonaID != zonaB || m.PosicionX == nil || *m.PosicionX != "1.50" || m.PosicionY == nil || *m.PosicionY != "-2.00" {
		t.Fatalf("edición: %+v", m)
	}
	r = callH(t, "PATCH", "/api/v1/mesas/"+id, `{"posicion_x":null}`, tk, nil)
	esperar(t, r, http.StatusOK, "")
	r.json(t, &m)
	if m.PosicionX != nil || m.PosicionY == nil || m.Capacidad != 6 {
		t.Fatalf("null debe vaciar solo la posición X: %+v", m)
	}

	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{}`, tk, nil), http.StatusUnprocessableEntity, "sin_cambios")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"sucursal_id":"1"}`, tk, nil), http.StatusBadRequest, "json_invalido")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"capacidad":0}`, tk, nil), http.StatusUnprocessableEntity, "capacidad_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"numero_mesa":"ED-2"}`, tk, nil), http.StatusConflict, "numero_mesa_duplicado")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, fmt.Sprintf(`{"zona_id":"%s"}`, idDe(t, "zonas", "nombre", "Zona editar ajena")), tk, nil),
		http.StatusUnprocessableEntity, "zona_invalida")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/999999", `{"capacidad":2}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+idDe(t, "mesas", "numero_mesa", "ED-AJ"), `{"capacidad":2}`, tk, nil), http.StatusNotFound, "no_encontrado")
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"capacidad":2}`, lectorMesas(t), nil), http.StatusForbidden, "sin_permiso")
}

func TestMesaBajaConMotivoYRestauracion(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Zona ciclo mesa")
	crearMesaSQL(t, "PRINCIPAL", "Zona ciclo mesa", "CM-1", 4)
	tk := gestorMesas(t)
	id := idDe(t, "mesas", "numero_mesa", "CM-1")
	gestor := idDe(t, "usuarios", "username", "mesa.gestor")

	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", "", tk), http.StatusBadRequest, "cuerpo_requerido")

	h := map[string]string{"Idempotency-Key": nuevaClave()}
	r := callH(t, "POST", "/api/v1/mesas/"+id+"/baja", `{"motivo":"Mesa dañada"}`, tk, h)
	esperar(t, r, http.StatusOK, "")
	var m mesaResp
	r.json(t, &m)
	if m.EsActivo || m.Motivo == nil || *m.Motivo != "Mesa dañada" || m.EliminadoPor == nil || *m.EliminadoPor != gestor {
		t.Fatalf("baja: %+v", m)
	}
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='mesas' AND accion='UPDATE'
		AND registro_id=`+id+` AND creado_por=`+gestor) == 0 {
		t.Fatal("la baja debe quedar auditada con el actor")
	}
	antes := scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='mesas' AND registro_id=`+id)
	esperar(t, callH(t, "POST", "/api/v1/mesas/"+id+"/baja", `{"motivo":"Mesa dañada"}`, tk, h), http.StatusOK, "")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.auditoria_eventos WHERE tabla='mesas' AND registro_id=`+id) != antes {
		t.Fatal("repetir la misma clave no debe volver a escribir")
	}
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"Otra vez"}`, tk), http.StatusConflict, "ya_dada_de_baja")
	// Una mesa dada de baja no se edita ni se habilita hasta restaurarla.
	esperar(t, callH(t, "PATCH", "/api/v1/mesas/"+id, `{"habilitada":true}`, tk, nil), http.StatusConflict, "mesa_inactiva")

	r = post(t, "/api/v1/mesas/"+id+"/restauracion", `{}`, tk)
	esperar(t, r, http.StatusOK, "")
	var rest mesaResp
	r.json(t, &rest)
	if rest.ID != m.ID || !rest.EsActivo || rest.Motivo != nil || rest.EliminadoPor != nil {
		t.Fatalf("restauración debe conservar el ID y limpiar los marcadores: %+v", rest)
	}
	esperar(t, post(t, "/api/v1/mesas/"+id+"/restauracion", "", tk), http.StatusConflict, "ya_activa")
	esperar(t, post(t, "/api/v1/mesas/999999/baja", `{"motivo":"x"}`, tk), http.StatusNotFound, "no_encontrado")
	esperar(t, post(t, "/api/v1/mesas/"+id+"/baja", `{"motivo":"x"}`, lectorMesas(t)), http.StatusForbidden, "sin_permiso")
}

// La base impide restaurar una mesa cuya zona está dada de baja: llega como regla_de_datos con
// el mensaje curado del esquema.
func TestMesaNoSeRestauraConLaZonaDadaDeBaja(t *testing.T) {
	crearZonaSQL(t, "PRINCIPAL", "Zona restaurar mesa")
	crearMesaSQL(t, "PRINCIPAL", "Zona restaurar mesa", "RM-1", 2)
	tk := gestorMesas(t)
	mesa := idDe(t, "mesas", "numero_mesa", "RM-1")
	zona := idDe(t, "zonas", "nombre", "Zona restaurar mesa")

	esperar(t, post(t, "/api/v1/mesas/"+mesa+"/baja", `{"motivo":"Cierre de zona"}`, tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/zonas/"+zona+"/baja", "", tk), http.StatusOK, "")

	r := post(t, "/api/v1/mesas/"+mesa+"/restauracion", "", tk)
	esperar(t, r, http.StatusUnprocessableEntity, "regla_de_datos")
	if scalar(t, `SELECT count(*) FROM sch_restaurante.mesas WHERE numero_mesa='RM-1' AND NOT es_activo`) != 1 {
		t.Fatal("la mesa debe seguir dada de baja")
	}

	// Restaurada la zona, la mesa ya puede restaurarse.
	esperar(t, post(t, "/api/v1/zonas/"+zona+"/restauracion", "", tk), http.StatusOK, "")
	esperar(t, post(t, "/api/v1/mesas/"+mesa+"/restauracion", "", tk), http.StatusOK, "")
}
```

- [ ] **Step 10: Ejecutar integración**

Run: `go test -count=1 ./tests/integration/ -run "TestMesa|TestZona" -v` y luego `go test -count=1 -shuffle=on ./tests/integration/...`
Expected: PASS ambas.

- [ ] **Step 11: Verificación completa del paso 2**

Run: `gofmt -l . ; go vet ./... && go test ./... && go test -count=1 ./tests/integration/...`
Expected: sin salida de `gofmt`; todo PASS (incluidas las pruebas de I1, del paso 1 y de este paso).

Comprobación manual de consistencia con el spec §8: baja de zona con mesas activas → 409 ✔; baja de mesa sin motivo → 422 ✔; número de mesa duplicado → 409 ✔; sede ajena → 404 ✔; `*_VER` no escribe → 403 ✔; motivo y actor de la baja de mesa en la fila y en `auditoria_eventos` ✔; idempotencia ✔; `tests/architecture` cubre `atencion` ✔.

- [ ] **Step 12: Commit**

```bash
git add internal/modules/atencion tests/integration
git commit -m "I2 paso 2: crear, editar, dar de baja y restaurar mesas

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

## Autorrevisión del plan

**Cobertura del spec (paso 2):**
- §1 permisos de mesa compartidos por zonas y mesas, baja de mesa con motivo, baja de zona simple, idempotencia en POST → Tareas 1, 3 y 5.
- §3 autorización de zonas y mesas (lectura `MESA_VER`/`MESA_ADMINISTRAR`, escritura `MESA_ADMINISTRAR`, sede ajena → 404) → `mapper.AutorizarSede` (Tarea 1) usado en todas las escrituras; `PuedeVer`/`SedesVisibles` en lecturas (Tareas 2 y 4).
- §4 rutas `GET/POST /zonas`, `GET/PATCH /zonas/{id}`, `POST …/baja`, `POST …/restauracion` y las equivalentes de mesas; `limit`/`cursor`; filtros `sucursal_id` y `es_activo`; reglas (nombre único por sede, baja de zona con mesas activas → 409, número único por sede, zona de la misma sede, capacidad > 0, `habilitada` por `PATCH`, mesa de baja no se edita, ocupación no expuesta, actor no aceptado del cliente, traducción de errores) → Tareas 2–5.
- §5 módulo `atencion` que no importa `identidad`, coordinación en la composición, `tests/architecture` cubre `atencion` → Tareas 1 (prueba específica) y 2 (bootstrap).
- §8 pruebas: unitarias de dominio y políticas (Tareas 1, 3, 5), integración (todas las filas aplicables de la lista), arquitectura. **No cubierto, con motivo:** «baja de mesa con pedido abierto» queda a cargo del trigger de la V4 y se prueba su traducción con la regla gemela (restaurar con zona de baja); el pedido abierto requiere el módulo de pedidos (I3).
- Fuera de este paso, según el spec: contrato OpenAPI, ADR 005, `module-map.md`, README y expediente (paso 3); frontend (paso 4).

**Placeholders:** no quedan TBD ni pasos sin código.

**Consistencia de tipos:** `mapper.AutorizarSede(a, sedeID)` y `mapper.PuedeVer(a, sedeID)` se usan con la misma firma en consultas y comandos; `NewZonasHandler` pasa de 2 a 5 argumentos en la Tarea 3 y `NewMesasHandler` de 2 a 5 en la Tarea 5 (sus únicas llamadas, en `config/zonas.go` y `config/mesas.go`, se reemplazan en esos pasos); `RegisterRoutes` cambia de firma en las Tareas 2 (zonas), 4 (zonas y mesas) y su única llamada está en `module.go`, que se actualiza en la Tarea 4; `outbound.CampoOpcional` (mesas) y `command.Campo[T]` / `request.Opcional[T]` comparten la forma `{Presente, Valor *T}`.
