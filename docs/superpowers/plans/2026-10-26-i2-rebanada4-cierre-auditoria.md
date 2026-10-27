# I2 rebanada 4 — Cierre con auditoría de bajas y formularios — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Verificar RES-RF-030 (bajas y restauraciones) y RES-RF-036 (formularios y errores) sobre las entidades de I2, fijar el resultado con pruebas, corregir los huecos que aparezcan y cerrar el expediente de I2 con RF-030 y RF-036 en `PARCIAL`.

**Architecture:** Una matriz de auditoría (documento) apoyada en pruebas. Backend: una prueba de integración de Go dirigida por tabla que recorre las 11 entidades con baja de I2. Frontend: se audita contra las pruebas existentes y se corrige el único hueco ya detectado (el diálogo compartido no lleva el foco al campo con error). Al final se actualiza el expediente y se abre un PR.

**Tech Stack:** Go 1.x + testcontainers (PostgreSQL 18) en `service-restaurant/`; Angular 21, Vitest y Playwright en `app-restaurant/`; expediente en `docs/planificacion/` (JSON, HTML, YAML, checksums).

Spec: `docs/superpowers/specs/2026-10-26-i2-rebanada4-cierre-auditoria-design.md`.

## Global Constraints

- RF-030 y RF-036 pasan a **`PARCIAL`**, nunca `IMPLEMENTADO`. RES-CP-030 y RES-CP-036 siguen `NO_EJECUTADA`.
- **D2:** una celda de la matriz solo se marca «cumple» si cita una prueba que existe y pasa.
- Los huecos se listan en la matriz **antes** de corregir nada; si un hueco exige una decisión de producto, se detiene y se pregunta al usuario. Aquí solo se corrige sin preguntar el hueco del foco (Task 3), que ya está decidido por el spec (F6) y RF-036.
- El estado del incremento I2 (`EN_CURSO` o cerrado) lo decide el usuario; no se cambia en este plan.
- Los commits terminan con `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.
- No hacer `git push` ni `gh pr create` sin confirmación del usuario (Task 7).
- Las pruebas de integración de Go necesitan Docker (testcontainers). Si Docker no está activo, declararlo; no marcar nada como verificado.
- Los archivos JSON del expediente son exactamente `JSON.stringify(x, null, 2) + "\n"`: se pueden editar con `JSON.parse` → modificar → `JSON.stringify`.
- Rama de trabajo: `i2-rebanada4-cierre-auditoria` (ya creada; contiene el spec en `a624d5e`).

## Estructura de archivos

| Archivo | Acción | Responsabilidad |
|---|---|---|
| `service-restaurant/tests/integration/bajas_contrato_test.go` | Crear | Prueba de integración dirigida por tabla: criterios de backend de RF-030 para cada entidad |
| `app-restaurant/src/app/shared/ui/form-dialog.ts` | Modificar | Llevar el foco al campo con error (cliente y servidor) |
| `app-restaurant/src/app/shared/ui/form-dialog.spec.ts` | Modificar | Pruebas del foco |
| `app-restaurant/e2e/carta.spec.ts` | Modificar | Aserción de foco en un error de campo real |
| `docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md` | Crear | Matriz de auditoría y lista de huecos |
| `docs/planificacion/01_documentacion/backlog_incrementos.json` | Modificar | «avance» de I2 |
| `docs/planificacion/01_documentacion/requisitos_y_trazabilidad.json` y `.html` | Modificar | RF-030 y RF-036 → `PARCIAL` con su situación |
| `docs/planificacion/03_api/openapi.yaml` | Modificar | `x-implementado: true` en clientes y proveedores |
| `docs/planificacion/CAMBIOS_1_3.txt`, `SHA256SUMS.txt` | Modificar | Nota de cambios y checksums |

---

### Task 1: Prueba de integración de bajas (backend)

**Files:**
- Create: `service-restaurant/tests/integration/bajas_contrato_test.go`

**Interfaces:**
- Consumes (helpers del paquete `integration`, ya existentes): `post(t, path, body, tk) resp`, `call(t, method, path, body, tk) resp`, `callH(...)`, `esperar(t, r, status, code)`, `r.json(t, &v)`, `nuevaClave()`, `scalar(t, query, args...) int`, `idDe(t, tabla, campo, valor) string`, `sufijo() string`, `numeroUnico(n int) string`, `token(t, user, pass) string`, `adminToken(t) string`, `asegurarUsuario(t, user, pass, rol, sedes...)`, `gestorToken(t)` y las constantes `gestorUser`, `gestorPass` (sedes), `gestorDirectorio(t)` (clientes y proveedores), `gestorCarta(t)` + `crearCategoria(t, tk, cuerpo) categoriaResp` + `cuerpoProd(cat, codigo, extras...) string` (productos), `gestorMesas(t, sedes...)` + `crearZonaSQL(t, sede, nombre) string` + `altaMesa(sucursalID, zonaID, numero, capacidad) string` + `altaZona(sucursalID, nombre) string` (mesas y zonas), `sedeEstaciones(t) (codigo, id string)` + `gestorEstaciones(t, sedes...)` + `cuerpoEstacion(sede, codigo, nombre) string` (estaciones), `altaSede(codigo) string`.
- Produces: `TestBajasContratoComun`, con subpruebas `TestBajasContratoComun/<entidad>/<criterio>`. Los nombres de criterio son exactamente: `sin_idempotency_key`, `motivo`, `campo_de_auditoria`, `baja`, `auditoria`, `restauracion`, `delete`. La matriz (Task 4) cita estos nombres.

- [ ] **Step 1: Escribir la prueba**

Crear `service-restaurant/tests/integration/bajas_contrato_test.go`:

```go
package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// RES-RF-030 y RES-RF-036 (criterios de backend), comprobados de forma uniforme en cada entidad
// de I2 con baja. Las entidades «maestro» exigen motivo y guardan motivo, actor y fecha; las de
// «desactivación simple» no los piden ni los aceptan. Un fallo de un criterio es un hueco que
// se anota en la matriz de auditoría; un fallo al preparar la entidad es un defecto de esta prueba.

type bajaResp struct {
	ID       string `json:"id"`
	EsActivo bool   `json:"es_activo"`
}

type entidadBaja struct {
	nombre  string // nombre de la subprueba
	ruta    string // colección bajo /api/v1
	tabla   string // tabla en sch_restaurante
	maestro bool   // true: baja con motivo; false: desactivación simple
	// preparar crea una instancia nueva y devuelve el token que puede darla de baja y su id.
	preparar func(t *testing.T) (tk, id string)
}

// crearPorAPI hace el alta con su Idempotency-Key y devuelve el id creado.
func crearPorAPI(t *testing.T, tk, ruta, cuerpo string) string {
	t.Helper()
	r := post(t, "/api/v1/"+ruta, cuerpo, tk)
	esperar(t, r, http.StatusCreated, "")
	var b bajaResp
	r.json(t, &b)
	if b.ID == "" {
		t.Fatalf("el alta de %s no devolvió id: %s", ruta, r.Body.String())
	}
	return b.ID
}

func entidadesBaja() []entidadBaja {
	return []entidadBaja{
		{"sucursales", "sucursales", "sucursales", true, func(t *testing.T) (string, string) {
			tk := gestorToken(t)
			codigo := "A" + strings.ToUpper(sufijo())
			id := crearPorAPI(t, tk, "sucursales", altaSede(codigo))
			// Quien crea la sede no queda asignado a ella: se le asigna para poder darla de baja.
			asegurarUsuario(t, gestorUser, gestorPass, "T_SUC_GESTOR", codigo)
			return token(t, gestorUser, gestorPass), id
		}},
		{"usuarios", "usuarios", "usuarios", true, func(t *testing.T) (string, string) {
			tk := adminToken(t)
			cuerpo := fmt.Sprintf(`{"username":"aud.%s","password":"Valida2026Clave"}`, strings.ToLower(sufijo()))
			return tk, crearPorAPI(t, tk, "usuarios", cuerpo)
		}},
		{"empleados", "empleados", "empleados", true, func(t *testing.T) (string, string) {
			tk := adminToken(t)
			cuerpo := fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombres":"Aud","apellidos":"Prueba %s"}`,
				numeroUnico(8), sufijo())
			return tk, crearPorAPI(t, tk, "empleados", cuerpo)
		}},
		{"clientes", "clientes", "clientes", true, func(t *testing.T) (string, string) {
			tk := gestorDirectorio(t)
			cuerpo := fmt.Sprintf(`{"tipo_documento":"DNI","numero_documento":"%s","nombre_razon_social":"Aud %s"}`,
				numeroUnico(8), sufijo())
			return tk, crearPorAPI(t, tk, "clientes", cuerpo)
		}},
		{"proveedores", "proveedores", "proveedores", true, func(t *testing.T) (string, string) {
			tk := gestorDirectorio(t)
			cuerpo := fmt.Sprintf(`{"tipo_documento":"RUC","numero_documento":"20%s","razon_social":"Aud %s"}`,
				numeroUnico(9), sufijo())
			return tk, crearPorAPI(t, tk, "proveedores", cuerpo)
		}},
		{"productos", "productos", "productos", true, func(t *testing.T) (string, string) {
			tk := gestorCarta(t)
			cat := crearCategoria(t, tk, fmt.Sprintf(`{"nombre":"Aud cat %s"}`, sufijo()))
			return tk, crearPorAPI(t, tk, "productos", cuerpoProd(cat.ID, "AUD"+strings.ToUpper(sufijo())))
		}},
		{"mesas", "mesas", "mesas", true, func(t *testing.T) (string, string) {
			tk := gestorMesas(t)
			principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
			zona := crearZonaSQL(t, "PRINCIPAL", "Zona aud "+sufijo())
			return tk, crearPorAPI(t, tk, "mesas", altaMesa(principal, zona, "A"+sufijo(), 2))
		}},
		{"categorias", "categorias", "categorias", false, func(t *testing.T) (string, string) {
			tk := gestorCarta(t)
			return tk, crearPorAPI(t, tk, "categorias", fmt.Sprintf(`{"nombre":"Aud %s"}`, sufijo()))
		}},
		{"estaciones", "estaciones", "estaciones_preparacion", false, func(t *testing.T) (string, string) {
			codigo, sede := sedeEstaciones(t)
			tk := gestorEstaciones(t, codigo)
			cuerpo := cuerpoEstacion(sede, "AUD"+strings.ToUpper(sufijo()), "Aud "+sufijo())
			return tk, crearPorAPI(t, tk, "estaciones", cuerpo)
		}},
		{"zonas", "zonas", "zonas", false, func(t *testing.T) (string, string) {
			tk := gestorMesas(t)
			principal := idDe(t, "sucursales", "codigo", "PRINCIPAL")
			return tk, crearPorAPI(t, tk, "zonas", altaZona(principal, "Aud "+sufijo()))
		}},
		{"roles", "roles", "roles", false, func(t *testing.T) (string, string) {
			tk := adminToken(t)
			cuerpo := fmt.Sprintf(`{"codigo":"AUD_%s","nombre":"Aud %s"}`, strings.ToUpper(sufijo()), sufijo())
			return tk, crearPorAPI(t, tk, "roles", cuerpo)
		}},
	}
}

func TestBajasContratoComun(t *testing.T) {
	const motivo = "Baja de la prueba de contrato"
	for _, e := range entidadesBaja() {
		t.Run(e.nombre, func(t *testing.T) {
			tk, id := e.preparar(t)
			url := "/api/v1/" + e.ruta + "/" + id
			baja, rest := url+"/baja", url+"/restauracion"
			tabla := "sch_restaurante." + e.tabla
			cuerpoBaja := ""
			if e.maestro {
				cuerpoBaja = `{"motivo":"` + motivo + `"}`
			}
			eventos := func() int {
				return scalar(t, fmt.Sprintf(`SELECT count(*) FROM sch_restaurante.auditoria_eventos
					WHERE tabla='%s' AND accion='UPDATE' AND registro_id=%s`, e.tabla, id))
			}
			antes := eventos()

			t.Run("sin_idempotency_key", func(t *testing.T) {
				esperar(t, call(t, "POST", baja, cuerpoBaja, tk), http.StatusBadRequest, "idempotency_key_requerida")
			})

			t.Run("motivo", func(t *testing.T) {
				if e.maestro {
					esperar(t, post(t, baja, `{}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
					esperar(t, post(t, baja, `{"motivo":"   "}`, tk), http.StatusUnprocessableEntity, "motivo_requerido")
				} else {
					// Una desactivación simple no admite motivo: el cuerpo desconocido se rechaza.
					esperar(t, post(t, baja, `{"motivo":"x"}`, tk), http.StatusBadRequest, "")
				}
			})

			t.Run("campo_de_auditoria", func(t *testing.T) {
				// El actor sale de la sesión: un campo de auditoría en el cuerpo se rechaza y no da de baja.
				cuerpo := `{"eliminado_por":"1"}`
				if e.maestro {
					cuerpo = `{"motivo":"x","eliminado_por":"1"}`
				}
				esperar(t, post(t, baja, cuerpo, tk), http.StatusBadRequest, "")
				if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND es_activo`, tabla, id)); n != 1 {
					t.Fatalf("un cuerpo con campo de auditoría no debe dar de baja (activas: %d)", n)
				}
			})

			t.Run("baja", func(t *testing.T) {
				r := post(t, baja, cuerpoBaja, tk)
				esperar(t, r, http.StatusOK, "")
				var b bajaResp
				r.json(t, &b)
				if b.ID != id || b.EsActivo {
					t.Fatalf("respuesta de la baja: %+v (id %s)", b, id)
				}
				if e.maestro {
					n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND NOT es_activo
						AND motivo_eliminacion=$1 AND eliminado_por IS NOT NULL AND eliminado_en IS NOT NULL`, tabla, id), motivo)
					if n != 1 {
						t.Fatal("un maestro dado de baja debe guardar motivo, usuario y fecha")
					}
				} else if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND NOT es_activo`, tabla, id)); n != 1 {
					t.Fatal("la desactivación simple debe dejar la fila inactiva")
				}
			})

			t.Run("restauracion", func(t *testing.T) {
				r := post(t, rest, `{}`, tk)
				esperar(t, r, http.StatusOK, "")
				var b bajaResp
				r.json(t, &b)
				if b.ID != id || !b.EsActivo {
					t.Fatalf("la restauración debe conservar el id y reactivar: %+v (id %s)", b, id)
				}
				if e.maestro {
					n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s AND es_activo
						AND motivo_eliminacion IS NULL AND eliminado_por IS NULL AND eliminado_en IS NULL`, tabla, id))
					if n != 1 {
						t.Fatal("restaurar un maestro debe limpiar motivo, usuario y fecha de baja")
					}
				}
			})

			t.Run("auditoria", func(t *testing.T) {
				// La baja y la restauración dejan, cada una, un evento de actualización con actor y cambios.
				if despues := eventos(); despues-antes < 2 {
					t.Fatalf("eventos de auditoría nuevos: %d, quería al menos 2", despues-antes)
				}
			})

			t.Run("delete", func(t *testing.T) {
				r := call(t, "DELETE", url, "", tk)
				if r.Code != http.StatusNotFound && r.Code != http.StatusMethodNotAllowed {
					t.Fatalf("DELETE %s respondió %d, debe ser 404 o 405: %s", url, r.Code, r.Body.String())
				}
				if n := scalar(t, fmt.Sprintf(`SELECT count(*) FROM %s WHERE id=%s`, tabla, id)); n != 1 {
					t.Fatal("DELETE no debe borrar la fila")
				}
			})
		})
	}
}
```

- [ ] **Step 2: Compilar**

Run (desde `service-restaurant/`): `go vet ./tests/integration/...`
Expected: sin salida (compila). Si un helper citado en **Interfaces** no existe con esa firma, corregir la llamada en este archivo leyendo el helper real; no inventar otro.

- [ ] **Step 3: Ejecutar la prueba**

Run: `go test -count=1 -run TestBajasContratoComun ./tests/integration/... 2>&1 | tail -60`
Expected: levanta PostgreSQL con testcontainers y ejecuta 11 entidades × 7 criterios. Dos desenlaces válidos:
- **Pasa todo.** Anotar las 77 celdas como «cumple».
- **Falla un criterio de una entidad.** Es un **hueco**: NO se corrige el código de producción en esta task. Copiar a un archivo temporal (`$TEMP/huecos-backend.txt`) la lista `entidad/criterio` + el mensaje, para la matriz (Task 4). Si el fallo es al **preparar** la entidad (el alta no devuelve 201, un helper no existe), es un defecto de esta prueba: corregirla y repetir.

- [ ] **Step 4: Repetibilidad**

Run: `go test -count=2 -run TestBajasContratoComun ./tests/integration/...`
Expected: el mismo resultado que en el Step 3 (la base se comparte y los nombres llevan `sufijo()`).

- [ ] **Step 5: Commit**

```bash
git add service-restaurant/tests/integration/bajas_contrato_test.go
git commit -m "I2 rebanada 4: prueba de contrato de bajas y restauraciones (RF-030)"
```
(Añadir al mensaje la línea `Co-Authored-By` indicada en Global Constraints.) Si hubo criterios fallidos, dejar la prueba con sus fallos y decirlo en el mensaje: «criterios con hueco: …».

---

### Task 2: Prueba del foco en el diálogo compartido (rojo)

**Files:**
- Modify: `app-restaurant/src/app/shared/ui/form-dialog.spec.ts` (dentro de `describe('Formulario de operación', …)`, después del test `'muestra el error de campo bajo el campo y conserva lo escrito'`)

**Interfaces:**
- Consumes: `FormDialog` (`submit(): Promise<void>`, `form`), campos con id `field-<key>`; los mocks `commit` y `close` del `describe`.
- Produces: dos pruebas que fallan hasta la Task 3.

- [ ] **Step 1: Añadir las pruebas**

Insertar tras el test `'muestra el error de campo bajo el campo y conserva lo escrito'`:

```ts
  it('lleva el foco al campo con el error del servidor', async () => {
    commit.mockResolvedValue({ field: 'email', message: 'Ese correo ya existe.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.detectChanges();
    fixture.componentInstance.form.setValue({ name: 'Ana', email: 'ana@example.com' });
    await fixture.componentInstance.submit();
    fixture.detectChanges();
    expect(document.activeElement).toBe(fixture.nativeElement.querySelector('#field-email'));
  });

  it('lleva el foco al primer campo inválido antes de enviar', async () => {
    const fixture = TestBed.createComponent(FormDialog);
    fixture.detectChanges();
    await fixture.componentInstance.submit();
    fixture.detectChanges();
    expect(commit).not.toHaveBeenCalled();
    expect(document.activeElement).toBe(fixture.nativeElement.querySelector('#field-name'));
  });
```

- [ ] **Step 2: Ver que fallan**

Run (desde `app-restaurant/`): `npx ng test --watch=false --include src/app/shared/ui/form-dialog.spec.ts 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | tail -25`
Expected: FAIL en las dos pruebas nuevas (`document.activeElement` es `body`), y el resto del archivo en verde.

- [ ] **Step 3: No hacer commit todavía** (la implementación va en la Task 3 y se commitea junto).

---

### Task 3: Foco al campo con error (verde) y aserción E2E

**Files:**
- Modify: `app-restaurant/src/app/shared/ui/form-dialog.ts` (import de `ElementRef`; clase `FormDialog`)
- Modify: `app-restaurant/e2e/carta.spec.ts` (test `'un código duplicado del servidor aparece en el campo y el diálogo sigue abierto'`)

**Interfaces:**
- Consumes: las dos pruebas de la Task 2.
- Produces: `FormDialog` lleva el foco al primer campo inválido (validación local) y al campo del `FieldProblem` (rechazo del servidor).

- [ ] **Step 1: Importar `ElementRef`**

En `form-dialog.ts`, línea 1, cambiar:

```ts
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
```
por:
```ts
import { ChangeDetectionStrategy, Component, ElementRef, inject, signal } from '@angular/core';
```

- [ ] **Step 2: Inyectar el host y añadir el método de foco**

En la clase `FormDialog`, cambiar:

```ts
  readonly data = inject<FormDialogData>(DIALOG_DATA);
  readonly ref = inject(DialogRef);
  readonly error = signal('');
```
por:
```ts
  readonly data = inject<FormDialogData>(DIALOG_DATA);
  readonly ref = inject(DialogRef);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  readonly error = signal('');
```

y justo antes de `async submit(): Promise<void> {` añadir:

```ts
  /** Lleva el foco al control del campo (RF-036: el error debe poder encontrarse con el teclado). */
  private focusField(key: string): void {
    this.host.nativeElement.querySelector<HTMLElement>(`[id="field-${key}"]`)?.focus();
  }
```

- [ ] **Step 3: Enfocar en la validación local**

Cambiar:

```ts
    if (this.form.invalid) {
      this.error.set('Completa los campos indicados antes de continuar.');
      return;
    }
```
por:
```ts
    if (this.form.invalid) {
      this.error.set('Completa los campos indicados antes de continuar.');
      const primero = this.data.fields.find((f) => this.form.controls[f.key].invalid);
      if (primero) this.focusField(primero.key);
      return;
    }
```

- [ ] **Step 4: Enfocar en el rechazo del servidor**

Cambiar:

```ts
          this.fieldErrors.set({ [outcome.field]: outcome.message });
          this.form.controls[outcome.field].markAsTouched();
```
por:
```ts
          this.fieldErrors.set({ [outcome.field]: outcome.message });
          this.form.controls[outcome.field].markAsTouched();
          this.focusField(outcome.field);
```

- [ ] **Step 5: Ver que pasan**

Run: `npx ng test --watch=false --include src/app/shared/ui/form-dialog.spec.ts 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | tail -12`
Expected: todas las pruebas del archivo pasan.

- [ ] **Step 6: Suite completa, lint y formato**

Run: `npm run test:run 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | grep -E "Test Files|Tests "; npm run lint 2>&1 | tail -3; npm run format:check 2>&1 | tail -1`
Expected: `Test Files  37 passed (37)`, `Tests  422 passed (422)` (420 + 2 nuevas), lint sin errores y «All matched files use Prettier code style!». Si prettier falla en los archivos tocados: `npx prettier --write src/app/shared/ui/form-dialog.ts src/app/shared/ui/form-dialog.spec.ts`.

- [ ] **Step 7: Aserción E2E del foco**

En `app-restaurant/e2e/carta.spec.ts`, en el test `'un código duplicado del servidor aparece en el campo y el diálogo sigue abierto'`, añadir al final del cuerpo (después de la última aserción `toHaveValue`):

```ts
    await expect(dialogo(page).getByLabel('Código', { exact: true })).toBeFocused();
```

Run: `npx playwright test carta.spec -g "código duplicado" --reporter=line 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | tail -8`
Expected: 3 pruebas (desktop, tablet, mobile) aprobadas. Luego `npx prettier --write e2e/carta.spec.ts` si hace falta.

- [ ] **Step 8: Regresión E2E del resto**

Run: `npx playwright test --reporter=dot 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | tail -6`
Expected: `267 passed` y `39 skipped` (los mismos números de la rebanada 3; el foco no debe romper ningún E2E). Si alguno falla porque el foco movido altera una aserción, leer el fallo y ajustar solo esa aserción, explicando el cambio en el commit.

- [ ] **Step 9: Commit**

```bash
git add app-restaurant/src/app/shared/ui/form-dialog.ts app-restaurant/src/app/shared/ui/form-dialog.spec.ts app-restaurant/e2e/carta.spec.ts
git commit -m "I2 rebanada 4: el diálogo lleva el foco al campo con error (RF-036)"
```
(con la línea `Co-Authored-By`).

---

### Task 4: Auditoría del frontend y matriz

**Files:**
- Create: `docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md`

**Interfaces:**
- Consumes: resultados de las Tasks 1 y 3; los greps de abajo.
- Produces: la matriz y la lista de huecos (`H-1`, `H-2`, …) que lee la Task 5.

- [ ] **Step 1: Recoger la evidencia del frontend**

Desde la raíz del repositorio, ejecutar y guardar la salida (se citará en la matriz):

```bash
cd app-restaurant/src/app
echo "== A. Baja con motivo (key: 'motivo' en diálogos)";            grep -rn "key: 'motivo'" features --include=*.ts | grep -v "\.spec\.ts"
echo "== B. Idempotency-Key por intento (newIdempotencyKey en secciones)"; grep -rn "newIdempotencyKey()" features --include=*.ts | grep -v "\.spec\.ts"
echo "== C. Doble envío ignorado (diálogo compartido)";                grep -n "ignora un segundo envío" shared/ui/form-dialog.spec.ts
echo "== D. Error de campo conserva lo escrito + foco";                grep -n "conserva lo escrito\|lleva el foco" shared/ui/form-dialog.spec.ts
echo "== E. Acciones por permiso/estado (puedeAdministrar/puedeEditar en plantillas)"; grep -rln "puedeAdministrar\|puedeEditar" features --include=*.ts | grep -v "\.spec\.ts"
echo "== F. Bajas visibles (Incluir bajas / Dados de baja)";          grep -rln "Incluir bajas\|Dados de baja" features --include=*.ts | grep -v "\.spec\.ts"
echo "== G. 403 en sección";                                           grep -rn "sinAcceso\|No tienes permiso" features --include=*.ts | grep -v "\.spec\.ts" | head -20
echo "== Specs de baja por sección";                                   grep -rln "baja" features --include=*.spec.ts
cd ../..
```

- [ ] **Step 2: Crear la matriz**

Crear `docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md` con esta estructura. Cada celda se reemplaza por una de tres formas exactas: `cumple · <referencia>`, `hueco · H-n` o `n/a · <razón>`. **Ninguna celda puede quedar con «—».**

```markdown
# Matriz de auditoría — bajas y formularios (I2)

Requisitos: RES-RF-030 y RES-RF-036. Spec: `2026-10-26-i2-rebanada4-cierre-auditoria-design.md`.
Fecha de la auditoría: 2026-10-26. Evidencia de backend: `TestBajasContratoComun/<entidad>/<criterio>` (`service-restaurant/tests/integration/bajas_contrato_test.go`).

## Backend

| Entidad | Tipo | Motivo, usuario y fecha (`baja`, `motivo`) | Actor de sesión (`campo_de_auditoria`) | Idempotency-Key (`sin_idempotency_key`) | Restaurar conserva id (`restauracion`) | Sin DELETE (`delete`) | Historial (`auditoria`) |
|---|---|---|---|---|---|---|---|
| sucursales | maestro | — | — | — | — | — | — |
| usuarios | maestro | — | — | — | — | — | — |
| empleados | maestro | — | — | — | — | — | — |
| clientes | maestro | — | — | — | — | — | — |
| proveedores | maestro | — | — | — | — | — | — |
| productos | maestro | — | — | — | — | — | — |
| mesas | maestro | — | — | — | — | — | — |
| categorias | simple | — | — | — | — | — | — |
| estaciones_preparacion | simple | — | — | — | — | — | — |
| zonas | simple | — | — | — | — | — | — |
| roles | simple | — | — | — | — | — | — |
| negocio | sin baja | n/a · singleton: solo se edita | n/a | n/a | n/a | n/a | n/a |
| tasas_impuesto | sin baja | n/a · se editan sus vigencias; no se dan de baja | n/a | n/a | n/a | n/a | n/a |

## Frontend

| Entidad | Motivo solo donde corresponde (A) | Una clave por intento y doble envío (B, C) | Error de campo conserva y enfoca (D) | Acciones por permiso y estado (E) | Bajas visibles (F) | 403 de sección (G) |
|---|---|---|---|---|---|---|
| sucursales | — | — | — | — | — | — |
| usuarios | — | — | — | — | — | — |
| empleados | — | — | — | — | — | — |
| clientes | — | — | — | — | — | — |
| proveedores | — | — | — | — | — | — |
| productos | — | — | — | — | — | — |
| mesas | — | — | — | — | — | — |
| categorias | — | — | — | — | — | — |
| estaciones_preparacion | — | — | — | — | — | — |
| zonas | — | — | — | — | — | — |
| roles | — | — | — | — | — | — |

## Huecos

(una entrada por hueco: `H-n · entidad · criterio · qué falla · arreglo propuesto · ¿requiere decisión de producto?`)
```

Reglas para rellenar:
- **Backend:** una celda es «cumple» solo si la subprueba correspondiente de la Task 1 pasó; escribir `cumple · TestBajasContratoComun/<entidad>/<criterio>`. Si falló, `hueco · H-n`. Las columnas «Motivo…» combinan `baja` y `motivo`: «cumple» solo si pasan las dos.
- **Frontend A:** en las salidas A del Step 1, un maestro debe aparecer con `key: 'motivo'` y una entidad simple no. Si la sección de la entidad no aparece en el repositorio (p. ej. roles sin pantalla de baja), `n/a · sin pantalla de baja`.
- **Frontend B/C:** B se cumple si el archivo de la sección llama `newIdempotencyKey()` al abrir el diálogo de la baja; C se cumple con la prueba del diálogo compartido (línea de la salida C).
- **Frontend D:** cumple con las pruebas de la salida D (incluye las del foco de la Task 3).
- **Frontend E, F, G:** cumple si el archivo de la sección aparece en la salida correspondiente; si no, hueco.
- **Siempre** citar el archivo o la prueba que sirve de evidencia, con su ruta o nombre.

- [ ] **Step 2b: Hueco ya conocido de contrato**

En la sección **Huecos** añadir, como `H-n` con «requiere decisión de producto: sí», esta entrada (verificada: `openapi.yaml` no tiene ninguna ruta `/empleados`, aunque el backend sí la sirve):

```
H-n · empleados · contrato · el backend sirve `GET/POST /empleados`, `GET/PATCH /empleados/{id}` y sus `baja`/`restauracion`, pero `openapi.yaml` no los define (no hay `x-implementado` que marcar) · añadir las 6 operaciones al contrato y actualizar los conteos de rutas y operaciones del expediente · sí: cambia el contrato y los conteos.
```

- [ ] **Step 3: Verificar que no queda ninguna celda vacía**

Run: `grep -c "| — |" docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md`
Expected: `0`.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md
git commit -m "I2 rebanada 4: matriz de auditoría de bajas y formularios"
```
(con la línea `Co-Authored-By`).

---

### Task 5: Puerta de decisión sobre los huecos

**Files:** ninguno.

- [ ] **Step 1: Presentar los huecos al usuario y esperar**

Mostrar al usuario la sección **Huecos** de la matriz: cada `H-n` con su arreglo propuesto y si exige decisión de producto. Preguntar, hueco por hueco, si se corrige en esta rebanada, se difiere a I3–I5 o se acepta. Incluir siempre el hueco del contrato de empleados.

**No continuar con la Task 6 ni corregir nada más hasta tener la respuesta.** Si no hay ningún hueco aparte del contrato de empleados, decirlo explícitamente.

- [ ] **Step 2: Registrar las decisiones**

Para cada hueco aceptado o diferido, escribir en la matriz, bajo su `H-n`, una línea `Decisión: <corregir | diferir a I3|I4|I5 | aceptar> · fecha 2026-10-26`. Para cada hueco que el usuario pida corregir, **agregar a este plan una task nueva** (siguiendo el formato TDD de las Tasks 2 y 3: prueba roja, implementación mínima, prueba verde, commit) y ejecutarla antes de la Task 6. Commit:

```bash
git add docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md
git commit -m "I2 rebanada 4: decisiones sobre los huecos de la matriz"
```
(con la línea `Co-Authored-By`).

---

### Task 6: Cierre del expediente

**Files:**
- Modify: `docs/planificacion/01_documentacion/backlog_incrementos.json`
- Modify: `docs/planificacion/01_documentacion/requisitos_y_trazabilidad.json` y `.html`
- Modify: `docs/planificacion/03_api/openapi.yaml`
- Modify: `docs/planificacion/CAMBIOS_1_3.txt`, `docs/planificacion/SHA256SUMS.txt`

**Interfaces:**
- Consumes: la matriz (Task 4) y las decisiones (Task 5). Si algún hueco corregido o diferido cambia la «situación» que se escribe abajo, ajustar el texto a lo realmente cierto antes de ejecutar el script.

- [ ] **Step 1: Escribir el script de actualización**

Guardar como `$TEMP/cierre-i2.mjs` (fuera del repositorio) y ejecutar desde `docs/planificacion/`:

```js
import fs from 'node:fs';
import crypto from 'node:crypto';

const SIT_030 =
  'Parcial en I2: las bajas y restauraciones de sucursales, usuarios, empleados, clientes, proveedores, productos y mesas (maestros con motivo, usuario y fecha) y de categorías, estaciones, zonas y roles (desactivación simple) cumplen los criterios de backend, fijados por la prueba de integración TestBajasContratoComun, y los de interfaz según la matriz de auditoría (docs/superpowers/specs/2026-10-26-i2-rebanada4-matriz-bajas-formularios.md). Faltan los maestros de inventario y caja (artículos, almacenes y cajas, I4 e I5). RES-CP-030 sigue NO_EJECUTADA.';
const SIT_036 =
  'Parcial en I2: formularios con errores por campo que conservan lo escrito, foco en el campo con error, una Idempotency-Key por intento y doble envío ignorado, en las pantallas de Configuración, Carta, Clientes y proveedores, Empleados y Equipo y accesos (ver la matriz de auditoría). Faltan las pantallas de I3 a I5. RES-CP-036 sigue NO_EJECUTADA.';
const AVANCE =
  'Cuatro rebanadas de I2 hechas, con backend y frontend: 1 (negocio, sucursales, zonas y mesas), 2 (clientes, proveedores y empleados), 3 (carta: categorías, estaciones, impuestos y tasas, productos y disponibilidad por sede) y 4 (auditoría de bajas y formularios: RES-RF-030 y RES-RF-036, con su matriz y las pruebas que la fijan). Los casos de aceptación siguen NO_EJECUTADA; RES-RF-030 y RES-RF-036 quedan PARCIAL hasta que lleguen los maestros y las pantallas de I3 a I5.';

const leer = (f) => fs.readFileSync(f, 'utf8');
const lista = (j) => (Array.isArray(j) ? j : Object.values(j).find(Array.isArray));
const guardarJson = (f, j) => fs.writeFileSync(f, JSON.stringify(j, null, 2) + '\n');

// 1. requisitos_y_trazabilidad.json
const fReq = '01_documentacion/requisitos_y_trazabilidad.json';
const req = JSON.parse(leer(fReq));
for (const [id, sit] of [['RES-RF-030', SIT_030], ['RES-RF-036', SIT_036]]) {
  const r = lista(req).find((x) => x.id === id);
  if (!r) throw new Error(id + ' no existe');
  r.estado_backend = 'PARCIAL';
  r.situacion_backend = sit;
}
guardarJson(fReq, req);

// 2. requisitos_y_trazabilidad.html (una sección por requisito)
const fHtml = '01_documentacion/requisitos_y_trazabilidad.html';
let html = leer(fHtml);
function seccion(id, sit) {
  const a = html.indexOf('id="' + id + '"');
  const b = html.indexOf('</tbody>', a);
  if (a < 0 || b < 0) throw new Error('HTML sin sección ' + id);
  let bloque = html.slice(a, b);
  if (!bloque.includes('<td>PENDIENTE · prototipo ')) throw new Error(id + ': no estaba PENDIENTE');
  bloque = bloque.replace('<td>PENDIENTE · prototipo ', '<td>PARCIAL · prototipo ');
  bloque = bloque.replace(/<tr><td>Situación backend<\/td><td>.*?<\/td><\/tr>/, '');
  bloque += '<tr><td>Situación backend</td><td>' + sit + '</td></tr>';
  html = html.slice(0, a) + bloque + html.slice(b);
}
seccion('RES-RF-030', SIT_030);
seccion('RES-RF-036', SIT_036);
fs.writeFileSync(fHtml, html);

// 3. backlog_incrementos.json
const fBack = '01_documentacion/backlog_incrementos.json';
const back = JSON.parse(leer(fBack));
const i2 = lista(back).find((x) => (x.id || x.codigo) === 'I2');
if (!i2) throw new Error('I2 no existe en el backlog');
i2.avance = AVANCE; // el campo «estado» no se toca: lo decide el usuario
guardarJson(fBack, back);

// 4. openapi.yaml: x-implementado de clientes y proveedores (empleados no está en el contrato)
const fApi = '03_api/openapi.yaml';
let ruta = '';
let cambiadas = 0;
const lineas = leer(fApi).split('\n').map((l) => {
  const p = /^  (\/[^:]*):\s*$/.exec(l);
  if (p) ruta = p[1];
  if (l === '      x-implementado: false' && /^\/(clientes|proveedores)(\/|$)/.test(ruta)) {
    cambiadas++;
    return '      x-implementado: true';
  }
  return l;
});
if (cambiadas !== 12) throw new Error('se esperaban 12 operaciones y fueron ' + cambiadas);
fs.writeFileSync(fApi, lineas.join('\n'));

// 5. CAMBIOS_1_3.txt
const fCam = 'CAMBIOS_1_3.txt';
let cam = leer(fCam);
if (!cam.endsWith('\n')) cam += '\n';
cam +=
  '- I2 rebanada 4: auditoría de bajas y formularios. RES-RF-030 y RES-RF-036 pasan de PENDIENTE a PARCIAL\n' +
  '  (matriz y prueba de contrato en el repositorio); los casos RES-CP-030 y RES-CP-036 siguen NO_EJECUTADA.\n' +
  '  Los clientes y proveedores (12 operaciones) pasan a x-implementado true; los empleados no están en el\n' +
  '  contrato (hueco registrado en la matriz). El avance de I2 recoge las cuatro rebanadas.\n';
fs.writeFileSync(fCam, cam);

// 6. SHA256SUMS.txt
const tocados = new Set([fReq, fHtml, fBack, fApi, fCam]);
const sums = leer('SHA256SUMS.txt')
  .split('\n')
  .map((l) => {
    const m = /^([0-9a-f]{64})  (.+)$/.exec(l);
    if (!m || !tocados.has(m[2])) return l;
    return crypto.createHash('sha256').update(fs.readFileSync(m[2])).digest('hex') + '  ' + m[2];
  });
fs.writeFileSync('SHA256SUMS.txt', sums.join('\n'));
console.log('ok: 12 operaciones marcadas, 2 requisitos en PARCIAL');
```

- [ ] **Step 2: Ejecutarlo**

Run (desde `docs/planificacion/`): `node "$TEMP/cierre-i2.mjs"`
Expected: `ok: 12 operaciones marcadas, 2 requisitos en PARCIAL`. Si lanza «no estaba PENDIENTE» o «se esperaban 12 operaciones», leer el archivo real y corregir el script; no forzar.

- [ ] **Step 3: Verificar checksums y diff**

Run: `sha256sum -c SHA256SUMS.txt 2>&1 | grep -v ": OK"; git diff --stat .`
Expected: `sha256sum` sin líneas de error (todo `OK`); el diff muestra los 6 archivos esperados (los 2 JSON, el HTML, el YAML, `CAMBIOS_1_3.txt` y `SHA256SUMS.txt`). Comprobar además: `node -e 'JSON.parse(require("fs").readFileSync("01_documentacion/requisitos_y_trazabilidad.json","utf8"))'` sin error.

- [ ] **Step 4: Commit**

```bash
git add docs/planificacion
git commit -m "I2 rebanada 4: expediente (RF-030 y RF-036 parciales, avance de I2, contrato)"
```
(con la línea `Co-Authored-By`).

---

### Task 7: Verificación final y PR

**Files:** ninguno (y `app-restaurant/VERIFICACION.md` si cambian los números).

- [ ] **Step 1: Verificación completa**

Frontend (desde `app-restaurant/`):
```bash
npm run lint 2>&1 | tail -2; npm run format:check 2>&1 | tail -1
npm run test:run 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | grep -E "Test Files|Tests "
npm run build 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | grep -E "Initial total"
npx playwright test --reporter=dot 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | tail -4
```
Expected: lint y prettier limpios; `Tests  422 passed` (más las que añada cualquier task de la Task 5); build dentro de 500 kB; Playwright sin fallos (`267 passed`, `39 skipped`).

Backend (desde `service-restaurant/`): `go vet ./... && go test -count=1 ./... 2>&1 | tail -15`
Expected: todo `ok`. Si Docker no está disponible para `tests/integration`, declararlo en el PR como «no ejecutado».

- [ ] **Step 2: Actualizar `VERIFICACION.md` solo con lo realmente medido**

En `app-restaurant/VERIFICACION.md`, actualizar la fecha y los números de la fila de Vitest (`422 pruebas`) si cambiaron; añadir una fila «Auditoría de bajas y formularios» que cite `TestBajasContratoComun` (n.º de entidades y criterios aprobados) y la matriz. No escribir resultados que no se hayan ejecutado.

- [ ] **Step 3: Commit de la verificación**

```bash
git add app-restaurant/VERIFICACION.md
git commit -m "I2 rebanada 4: verificación local actualizada"
```
(con la línea `Co-Authored-By`; omitir si no hubo cambios).

- [ ] **Step 4: Pedir confirmación antes de publicar**

Mostrar al usuario el resumen (commits, resultados, huecos y decisiones) y **preguntar si sube la rama y abre el PR contra `main`**. Solo con un sí explícito:

```bash
git push -u origin i2-rebanada4-cierre-auditoria
gh pr create --base main --head i2-rebanada4-cierre-auditoria --title "I2 rebanada 4: cierre con auditoría de bajas y formularios" --body-file <archivo>
```
El cuerpo (en un archivo del directorio temporal de la sesión, terminado en `🤖 Generated with [Claude Code](https://claude.com/claude-code)`) resume: matriz y prueba de contrato, el foco del diálogo, huecos y decisiones, expediente (RF-030 y RF-036 `PARCIAL`), verificación realizada y lo no ejecutado.

---

## Self-review del plan

**Cobertura del spec.**
- §1 alcance (7 maestros, simples, sin baja): Task 1 (11 entidades) y matriz con `negocio` y `tasas_impuesto` como `n/a` (Task 4).
- §3 matriz y 12 criterios: 6 de backend → Task 1 (columnas ↔ subpruebas), 6 de frontend → Task 4 (A–G; el foco en D por la Task 3).
- §4 pruebas: Go parametrizada (Task 1); frontend unitario (Task 2–3, más la evidencia de los specs existentes); E2E del foco (Task 3); API real: no se amplía, se declara.
- §5 huecos antes de corregir y decisiones de producto: Tasks 4 y 5 (puerta).
- §6 cierre documental: Task 6 (avance, RF-030/036 `PARCIAL`, `x-implementado`, checksums, cambios). `module-map.md`/`ADD.md`: no se tocan porque ningún cambio de backend los afecta.
- §7 entrega: Task 7. D1 (`PARCIAL`) y D2 (sin celda sin prueba): Global Constraints y reglas de la Task 4.

**Decisiones del spec que el plan matiza:** el spec habla de «un spec de frontend por tipo de baja»; la auditoría mostró que el doble envío y el error de campo son del diálogo compartido (`form-dialog.spec.ts` ya los prueba), así que el plan los cita como evidencia y solo añade lo que faltaba (foco) en vez de duplicar specs por sección. Si la matriz revela una sección sin cobertura (E, F o G), la Task 5 abre la task correspondiente.

**Consistencia de nombres.** `TestBajasContratoComun`, los 7 nombres de criterio, `entidadBaja`, `crearPorAPI`, `bajaResp`, `focusField` y `host` se usan igual en todas las tasks; la matriz cita los mismos nombres de subprueba.

**Sin marcadores.** Las partes que dependen de lo que encuentre la auditoría (celdas de la matriz, correcciones de huecos) tienen un procedimiento exacto, un criterio de verificación (`grep -c "| — |"` = 0) y una puerta de decisión en la Task 5, no un «a completar».
