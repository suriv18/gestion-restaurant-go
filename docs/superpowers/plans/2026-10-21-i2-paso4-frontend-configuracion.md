# I2 · Paso 4 — Frontend de Configuración — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Que la pantalla Configuración de `app-restaurant` trabaje contra la API real: «Tu restaurante» (negocio), «Sedes» y «Zonas y mesas», con permisos, diálogos con idempotencia y errores de campo en el campo; en modo demo conserva la pantalla de ejemplo.

**Architecture:** Mismo patrón que «Equipo y accesos»: un contenedor (`configuracion-page.ts`) elige entre la pantalla real (`app-configuracion-real`) y la demo; un store propio de la página (`ConfiguracionStore`, `@Injectable()`, signals) trae todo del servidor y expone acciones que devuelven el texto de error al diálogo; un servicio `ConfiguracionApi` habla con `/negocio`, `/sucursales`, `/zonas`, `/mesas`. Tres secciones (`NegocioSection`, `SedesSection`, `SalonSection`) comparten el store que provee la pantalla real. El diálogo compartido `Dialogs.form` se extiende para pintar errores de campo.

**Tech Stack:** Angular 21.2 (zoneless, standalone, OnPush, signals), TypeScript 5.9 estricto, Vitest 4 (jsdom, vía `ng test`), Playwright 1.64, ESLint/angular-eslint, Prettier (printWidth 100, comillas simples).

**Spec:** `docs/superpowers/specs/2026-10-18-i2-configuracion-salon-design.md` (§7 Frontend, §8 Pruebas de frontend). **Requisito previo:** pasos 1–3 integrados en `main` (lo están: PR #1 y #2).

## Global Constraints

- Todos los comandos se ejecutan desde `app-restaurant/`. Antes de cada commit: `npm run format`, luego `npm run lint`, `npm run format:check` y `npm run test:run` deben pasar; `npm run build` al cerrar las tareas 8 y 10.
- Commits en español con el trailer `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`. Textos de interfaz en español, tuteo, mayúscula solo inicial en botones («Dar de baja», «Confirmar baja», «Restaurar», «Reintentar», «Entendido», «Guardar», «Cancelar»).
- Los IDs viajan como **cadenas** (`"id": "7"`, `"sucursal_id": "1"`, `"zona_id": "3"`); las posiciones de mesa son **cadenas decimales** (`"12.50"`). Respuestas de lista: `{ items: T[], nextCursor: string | null }`; `limit` 1–100.
- Los `POST` de creación, baja y restauración llevan `Idempotency-Key` (UUID) generada **una vez por formulario abierto** con `newIdempotencyKey()` y reutilizada en los reintentos; los `PATCH` no la llevan. El navegador nunca envía `creado_por`, `actualizado_por` ni `eliminado_*`.
- Bajas con motivo (1–2000): **sedes** y **mesas**. La baja de **zona** no pide motivo (diálogo de confirmación sin campos). Restaurar no abre diálogo.
- Permisos de la interfaz (no son seguridad; el servidor vuelve a validar): negocio ver `NEGOCIO_VER`|`NEGOCIO_ADMINISTRAR`, editar `NEGOCIO_ADMINISTRAR`; sedes ver `SUCURSAL_VER`|`SUCURSAL_ADMINISTRAR`, escribir `SUCURSAL_ADMINISTRAR`; zonas y mesas ver `MESA_VER`|`MESA_ADMINISTRAR`, escribir `MESA_ADMINISTRAR`. `auth.hasPermission(p)` es «en alguna sede» (la sesión no trae permisos por sede): el 403 `sede_no_autorizada` y el 404 del servidor mandan.
- Sin permiso de ninguna sección → un aviso `note-box orange`. Una sección sin permiso de lectura **no aparece**; sin permiso de escritura queda en solo lectura (sin botones de acción).
- Un 403/4xx al cargar una sección muestra el mensaje en esa sección (con «Reintentar»), no una pantalla rota. Las demás secciones siguen funcionando.
- Las acciones de los diálogos devuelven `Promise<string | FieldProblem | undefined>`: `undefined` = éxito (cierra); texto = aviso general en el diálogo; `FieldProblem` = error bajo el campo. El diálogo no se cierra hasta que salga bien y conserva lo escrito.
- Sede recién creada: aviso “Asígnala a tu equipo en Equipo y accesos para que alguien pueda operarla” (texto del spec §7).
- Lint: selectores con prefijo `app-`, `@for` con `track`, sin `any`, plantillas accesibles (botones reales, etiquetas asociadas). Sin Python; las ediciones masivas se hacen con las herramientas del agente.

## Decisiones derivadas (no están en el spec)

| Tema | Decisión |
|---|---|
| Errores de campo | `Dialogs.form` se extiende: `commit` puede devolver `{ field, message }`; el mensaje sale bajo ese campo, el campo se marca `aria-invalid` y el error se borra al editarlo. Si `field` no existe en el formulario, cae al aviso general. |
| Ubicación | El contenedor se mueve de `features/catalogos/` a `features/configuracion/` (ruta actualizada). La demo sigue con `RestaurantStore`. |
| Nombres de sede | Con `SUCURSAL_VER|ADMINISTRAR` se usan los de `GET /sucursales`; sin ese permiso (p. ej. mesero) se muestra «Sede {id}». |
| Paginación | Se extrae `collect()` de `equipo-store.ts` a `core/http/collect.ts` (con `Pagina<T>`) y ambos stores la comparten. |
| Cambios parciales | Los `PATCH` envían solo los campos que cambiaron (`cambiosDe`); sin cambios no hay petición y el diálogo se cierra. Campos opcionales vaciados viajan como `null`. |
| Habilitar mesa | Interruptor en la fila (`role="switch"`), sin diálogo ni motivo. |
| Fuera de alcance | Reordenar zonas por arrastre, filtro por zona, plano del salón, ocupación de mesas (I3). |

## Estructura de archivos

Todo bajo `app-restaurant/src/app/` salvo indicación.

| Archivo | Responsabilidad |
|---|---|
| `shared/ui/form-dialog.ts` (+spec) | `FieldProblem` y errores de campo |
| `core/http/collect.ts` (+spec), `core/http/problem.ts` (+`problem-code.spec.ts`) | `Pagina<T>`, `collect()`, `problemCode()` |
| `features/identidad/data-access/equipo-store.ts`, `accesos.models.ts` | Usan `collect`/`Pagina` compartidos |
| `features/configuracion/configuracion.models.ts` | Tipos del API |
| `features/configuracion/configuracion.forms.ts` (+spec) | Normalización de formularios y `cambiosDe` |
| `features/configuracion/data-access/configuracion-api.ts` | HTTP |
| `features/configuracion/data-access/configuracion-store.ts` (+spec) | Estado y acciones |
| `features/configuracion/negocio-section.ts`, `sedes-section.ts`, `salon-section.ts` | Secciones |
| `features/configuracion/configuracion-real.ts` | Pantalla real |
| `features/configuracion/configuracion-page.ts`/`.html` (+spec) | Contenedor real/demo (movido) |
| `app.routes.ts` | Ruta al nuevo contenedor |
| `e2e/configuracion-support.ts`, `e2e/configuracion.spec.ts`, `e2e/configuracion.real-api.spec.ts` | E2E |
| `README.md`, `VERIFICACION.md` (de `app-restaurant/`) | Documentación |

---

### Task 1: Errores de campo en `Dialogs.form`

**Files:**
- Modify: `src/app/shared/ui/form-dialog.ts`
- Modify: `src/app/shared/ui/form-dialog.spec.ts`

**Interfaces:**
- Consumes: `FormDialog`, `FormDialogData`, `Field` existentes.
- Produces: `export interface FieldProblem { field: string; message: string }`; `commit` puede devolver `Promise<string | FieldProblem | undefined>`.

- [ ] **Step 1: Pruebas que fallan**

Añadir al final del `describe('Formulario de operación', …)` de `form-dialog.spec.ts` (el `beforeEach` ya define `commit`, `close` y el campo `name`):

```ts
  it('muestra el error de campo bajo el campo y conserva lo escrito', async () => {
    commit.mockResolvedValue({ field: 'name', message: 'Ese nombre ya existe.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    fixture.detectChanges();
    const el = fixture.nativeElement as HTMLElement;
    expect(el.textContent).toContain('Ese nombre ya existe.');
    expect(el.querySelector('#field-name')?.getAttribute('aria-invalid')).toBe('true');
    expect(fixture.componentInstance.form.controls['name'].value).toBe('Terraza');
    expect(close).not.toHaveBeenCalled();
  });

  it('borra el error del campo al editarlo', async () => {
    commit.mockResolvedValue({ field: 'name', message: 'Ese nombre ya existe.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    fixture.componentInstance.form.controls['name'].setValue('Terraza 2');
    fixture.detectChanges();
    expect((fixture.nativeElement as HTMLElement).textContent).not.toContain('Ese nombre ya existe.');
  });

  it('un error de un campo que no existe cae al aviso general', async () => {
    commit.mockResolvedValue({ field: 'otro', message: 'Dato inválido.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    fixture.detectChanges();
    expect(fixture.componentInstance.error()).toBe('Dato inválido.');
    expect(close).not.toHaveBeenCalled();
  });
```

- [ ] **Step 2: Verificar que fallan**

Run: `npx ng test --watch=false --include src/app/shared/ui/form-dialog.spec.ts`
Expected: FAIL (el diálogo hoy ignora objetos y cierra: `close` fue llamado / el texto no aparece).

- [ ] **Step 3: Implementar**

En `form-dialog.ts`:

(a) Imports: añadir `import { takeUntilDestroyed } from '@angular/core/rxjs-interop';`.

(b) Tras `export interface Field`, añadir:

```ts
/** Rechazo del servidor asociado a un campo: el diálogo lo pinta bajo ese campo. */
export interface FieldProblem {
  field: string;
  message: string;
}

function isFieldProblem(value: unknown): value is FieldProblem {
  return typeof value === 'object' && value !== null && 'field' in value && 'message' in value;
}
```

(c) En el comentario de `commit`, tras «un texto resuelto…», añadir: «Un `FieldProblem` resuelto se muestra bajo su campo (o en el aviso general si el campo no existe).»

(d) En la plantilla, reemplazar las tres apariciones de `[attr.aria-invalid]="invalid(field.key)"` por `[attr.aria-invalid]="flagged(field.key)"` y reemplazar el bloque

```html
            @if (invalid(field.key)) {
              <small class="text-red-700">Revisa este campo.</small>
            }
```

por

```html
            @if (fieldErrors()[field.key]; as message) {
              <small class="text-red-700" role="alert">{{ message }}</small>
            } @else if (invalid(field.key)) {
              <small class="text-red-700">Revisa este campo.</small>
            }
```

(e) En la clase `FormDialog`: añadir `readonly fieldErrors = signal<Record<string, string>>({});`, un constructor y el método `flagged`:

```ts
  constructor() {
    for (const [key, control] of Object.entries(this.form.controls)) {
      control.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => {
        if (!this.fieldErrors()[key]) return;
        this.fieldErrors.update((current) => {
          const next = { ...current };
          delete next[key];
          return next;
        });
      });
    }
  }

  flagged(key: string): boolean {
    return this.invalid(key) || !!this.fieldErrors()[key];
  }
```

(f) En `submit()`: tras `this.error.set('');` añadir `this.fieldErrors.set({});` y reemplazar el bloque `const message = … this.ref.close(true);` por:

```ts
      const outcome = result instanceof Promise ? await result : undefined;
      if (typeof outcome === 'string' && outcome) {
        this.error.set(outcome);
        return;
      }
      if (isFieldProblem(outcome)) {
        if (outcome.field in this.form.controls) {
          this.fieldErrors.set({ [outcome.field]: outcome.message });
          this.form.controls[outcome.field].markAsTouched();
        } else {
          this.error.set(outcome.message);
        }
        return;
      }
      this.ref.close(true);
```

- [ ] **Step 4: Verificar que pasan**

Run: `npx ng test --watch=false --include src/app/shared/ui/form-dialog.spec.ts` → PASS (los 3 nuevos y los existentes).

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add src/app/shared/ui
git commit -m "I2 paso 4: errores de campo en el diálogo de formularios

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 2: `collect()` compartido y `problemCode()`

**Files:**
- Create: `src/app/core/http/collect.ts`, `src/app/core/http/collect.spec.ts`, `src/app/core/http/problem-code.spec.ts`
- Modify: `src/app/core/http/problem.ts`, `src/app/features/identidad/data-access/accesos.models.ts`, `src/app/features/identidad/data-access/equipo-store.ts`

**Interfaces:**
- Produces: `Pagina<T>`, `collect<T>(fetchPage: (cursor?: string) => Promise<Pagina<T>>): Promise<T[]>` (máx. 20 páginas), `problemCode(error: unknown): string | undefined`.

- [ ] **Step 1: Pruebas que fallan**

`src/app/core/http/collect.spec.ts`:

```ts
import { collect, Pagina } from './collect';

describe('Carga de páginas', () => {
  it('sigue el cursor hasta que no hay más páginas', async () => {
    const fetchPage = vi
      .fn<(cursor?: string) => Promise<Pagina<number>>>()
      .mockResolvedValueOnce({ items: [1, 2], nextCursor: 'c1' })
      .mockResolvedValueOnce({ items: [3], nextCursor: null });
    expect(await collect(fetchPage)).toEqual([1, 2, 3]);
    expect(fetchPage).toHaveBeenNthCalledWith(1, undefined);
    expect(fetchPage).toHaveBeenNthCalledWith(2, 'c1');
  });

  it('se detiene a las 20 páginas aunque el servidor siga devolviendo cursor', async () => {
    const fetchPage = vi.fn(async () => ({ items: [1], nextCursor: 'siempre' }));
    expect(await collect(fetchPage)).toHaveLength(20);
    expect(fetchPage).toHaveBeenCalledTimes(20);
  });
});
```

`src/app/core/http/problem-code.spec.ts`:

```ts
import { HttpErrorResponse } from '@angular/common/http';
import { problemCode } from './problem';

describe('Código del problema', () => {
  it('devuelve el código del cuerpo problem+json', () => {
    const error = new HttpErrorResponse({ status: 409, error: { code: 'nombre_duplicado' } });
    expect(problemCode(error)).toBe('nombre_duplicado');
  });

  it('devuelve undefined si no hay código o el error no es HTTP', () => {
    expect(problemCode(new HttpErrorResponse({ status: 500 }))).toBeUndefined();
    expect(problemCode(new Error('x'))).toBeUndefined();
  });
});
```

- [ ] **Step 2: Verificar que fallan**

Run: `npx ng test --watch=false --include "src/app/core/http/*.spec.ts"` → FAIL (módulos/exports inexistentes).

- [ ] **Step 3: Implementar**

`src/app/core/http/collect.ts`:

```ts
/** Envoltorio de listas paginadas por cursor (`GET` de la API). */
export interface Pagina<T> {
  items: T[];
  nextCursor: string | null;
}

const MAX_PAGES = 20;

/** Recorre las páginas por cursor (con tope) y devuelve todos los elementos. */
export async function collect<T>(
  fetchPage: (cursor?: string) => Promise<Pagina<T>>,
): Promise<T[]> {
  const all: T[] = [];
  let cursor: string | undefined;
  for (let i = 0; i < MAX_PAGES; i++) {
    const page = await fetchPage(cursor);
    all.push(...page.items);
    if (!page.nextCursor) return all;
    cursor = page.nextCursor;
  }
  return all;
}
```

En `problem.ts`, al final:

```ts
/** Código estable del rechazo (`code` del problem+json), si el servidor lo envió. */
export function problemCode(error: unknown): string | undefined {
  if (!(error instanceof HttpErrorResponse)) return undefined;
  return ((error.error ?? {}) as Problem).code;
}
```

En `accesos.models.ts` reemplazar la declaración `export interface Pagina<T> { … }` por `import type { Pagina } from '../../../core/http/collect';` (arriba, con los demás imports) y `export type { Pagina };`. En `equipo-store.ts`: eliminar `const MAX_PAGES = 20;` y la función `collect` del final del archivo, y añadir `import { collect } from '../../../core/http/collect';` (la importación de `Pagina` si se usa pasa a `from './accesos.models'` como ya está).

- [ ] **Step 4: Verificar que pasa todo lo existente**

Run: `npm run test:run` → PASS completo (incluye `equipo-store.spec.ts`, que ejercita `collect`).

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add src/app/core src/app/features/identidad
git commit -m "I2 paso 4: collect() y problemCode() compartidos

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Modelos, normalización de formularios y API

**Files:**
- Create: `src/app/features/configuracion/configuracion.models.ts`, `configuracion.forms.ts`, `configuracion.forms.spec.ts`, `data-access/configuracion-api.ts`

**Interfaces:**
- Consumes: `Pagina` (`core/http/collect`), `API_BASE` (`core/auth/auth.models`), `HttpClient`.
- Produces: tipos `Negocio`, `Sucursal`, `Zona`, `Mesa` y de cambios/altas; helpers `texto`, `opcional`, `entero`, `cambiosDe`; `ConfiguracionApi`.

- [ ] **Step 1: Pruebas que fallan** — `configuracion.forms.spec.ts`:

```ts
import { cambiosDe, entero, opcional, texto } from './configuracion.forms';

describe('Normalización de formularios de Configuración', () => {
  it('texto recorta espacios', () => {
    expect(texto('  Terraza ')).toBe('Terraza');
  });

  it('opcional devuelve null si está vacío y el texto recortado si no', () => {
    expect(opcional('   ')).toBeNull();
    expect(opcional(' 12.50 ')).toBe('12.50');
  });

  it('entero convierte a número', () => {
    expect(entero('8')).toBe(8);
    expect(entero(' 12 ')).toBe(12);
  });

  it('cambiosDe devuelve solo los campos que cambiaron, incluido el vaciado a null', () => {
    const actual = { nombre: 'Salón', orden: 1, telefono: '999', ubigeo: null };
    const nuevo = { nombre: 'Salón', orden: 2, telefono: null, ubigeo: null };
    expect(cambiosDe(actual, nuevo)).toEqual({ orden: 2, telefono: null });
  });

  it('cambiosDe devuelve un objeto vacío si nada cambió', () => {
    expect(cambiosDe({ a: 'x' }, { a: 'x' })).toEqual({});
  });
});
```

- [ ] **Step 2: Verificar que fallan** — `npx ng test --watch=false --include src/app/features/configuracion/configuracion.forms.spec.ts` → FAIL (módulo inexistente).

- [ ] **Step 3: Implementar**

`configuracion.models.ts`:

```ts
/** Contratos de la API de Configuración. Todos los IDs viajan como cadenas. */
export interface Auditoria {
  creado_en: string;
  creado_por: string;
  actualizado_en: string;
  actualizado_por: string;
}

export interface Negocio extends Auditoria {
  id: string;
  singleton: boolean;
  ruc: string;
  razon_social: string;
  nombre_comercial: string;
  direccion_fiscal: string;
  ubigeo: string | null;
  telefono: string | null;
  email: string | null;
  moneda: string;
  zona_horaria: string;
  es_activo: boolean;
}

export interface NegocioCambios {
  razon_social?: string;
  nombre_comercial?: string;
  direccion_fiscal?: string;
  ubigeo?: string | null;
  telefono?: string | null;
  email?: string | null;
  moneda?: string;
  zona_horaria?: string;
}

export interface Sucursal extends Auditoria {
  id: string;
  negocio_id: string;
  codigo: string;
  nombre: string;
  direccion: string;
  ubigeo: string | null;
  codigo_establecimiento_sunat: string | null;
  telefono: string | null;
  es_activo: boolean;
  motivo_eliminacion: string | null;
  eliminado_en: string | null;
  eliminado_por: string | null;
}

export interface SucursalNueva {
  codigo: string;
  nombre: string;
  direccion: string;
  ubigeo?: string;
  codigo_establecimiento_sunat?: string;
  telefono?: string;
}

export interface SucursalCambios {
  codigo?: string;
  nombre?: string;
  direccion?: string;
  ubigeo?: string | null;
  codigo_establecimiento_sunat?: string | null;
  telefono?: string | null;
}

export interface Zona extends Auditoria {
  id: string;
  sucursal_id: string;
  nombre: string;
  orden_visualizacion: number;
  es_activo: boolean;
}

export interface ZonaNueva {
  sucursal_id: string;
  nombre: string;
  orden_visualizacion?: number;
}

export interface ZonaCambios {
  nombre?: string;
  orden_visualizacion?: number;
}

export interface Mesa extends Auditoria {
  id: string;
  sucursal_id: string;
  zona_id: string;
  numero_mesa: string;
  capacidad: number;
  habilitada: boolean;
  posicion_x: string | null;
  posicion_y: string | null;
  es_activo: boolean;
  motivo_eliminacion: string | null;
  eliminado_en: string | null;
  eliminado_por: string | null;
}

export interface MesaNueva {
  sucursal_id: string;
  zona_id: string;
  numero_mesa: string;
  capacidad: number;
  habilitada?: boolean;
  posicion_x?: string;
  posicion_y?: string;
}

export interface MesaCambios {
  zona_id?: string;
  numero_mesa?: string;
  capacidad?: number;
  habilitada?: boolean;
  posicion_x?: string | null;
  posicion_y?: string | null;
}

/** Sede elegible en «Zonas y mesas». */
export interface SedeRef {
  id: string;
  nombre: string;
}
```

`configuracion.forms.ts`:

```ts
/** Texto de un campo obligatorio, sin espacios sobrantes. */
export const texto = (valor: string): string => valor.trim();

/** Texto de un campo opcional: vacío = null (el servidor lo limpia). */
export const opcional = (valor: string): string | null => {
  const limpio = valor.trim();
  return limpio === '' ? null : limpio;
};

export const entero = (valor: string): number => Number(valor.trim());

type Valor = string | number | boolean | null;

/** Campos de `nuevo` cuyo valor difiere del `actual`: base de los PATCH parciales. */
export function cambiosDe(
  actual: Record<string, Valor>,
  nuevo: Record<string, Valor>,
): Record<string, Valor> {
  const cambios: Record<string, Valor> = {};
  for (const [clave, valor] of Object.entries(nuevo)) {
    if (actual[clave] !== valor) cambios[clave] = valor;
  }
  return cambios;
}
```

`data-access/configuracion-api.ts`:

```ts
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import type { Pagina } from '../../../core/http/collect';
import type {
  Mesa,
  MesaCambios,
  MesaNueva,
  Negocio,
  NegocioCambios,
  Sucursal,
  SucursalCambios,
  SucursalNueva,
  Zona,
  ZonaCambios,
  ZonaNueva,
} from '../configuracion.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

function page(cursor?: string): HttpParams {
  let params = new HttpParams().set('limit', '100');
  if (cursor) params = params.set('cursor', cursor);
  return params;
}

@Injectable({ providedIn: 'root' })
export class ConfiguracionApi {
  private readonly http = inject(HttpClient);

  negocio(): Observable<Negocio> {
    return this.http.get<Negocio>(`${API_BASE}/negocio`);
  }
  editarNegocio(cambios: NegocioCambios): Observable<Negocio> {
    return this.http.patch<Negocio>(`${API_BASE}/negocio`, cambios);
  }

  sucursales(cursor?: string): Observable<Pagina<Sucursal>> {
    return this.http.get<Pagina<Sucursal>>(`${API_BASE}/sucursales`, { params: page(cursor) });
  }
  crearSucursal(body: SucursalNueva, key: string): Observable<Sucursal> {
    return this.http.post<Sucursal>(`${API_BASE}/sucursales`, body, idem(key));
  }
  editarSucursal(id: string, cambios: SucursalCambios): Observable<Sucursal> {
    return this.http.patch<Sucursal>(`${API_BASE}/sucursales/${id}`, cambios);
  }
  bajaSucursal(id: string, motivo: string, key: string): Observable<Sucursal> {
    return this.http.post<Sucursal>(`${API_BASE}/sucursales/${id}/baja`, { motivo }, idem(key));
  }
  restaurarSucursal(id: string, key: string): Observable<Sucursal> {
    return this.http.post<Sucursal>(`${API_BASE}/sucursales/${id}/restauracion`, {}, idem(key));
  }

  zonas(cursor?: string): Observable<Pagina<Zona>> {
    return this.http.get<Pagina<Zona>>(`${API_BASE}/zonas`, { params: page(cursor) });
  }
  crearZona(body: ZonaNueva, key: string): Observable<Zona> {
    return this.http.post<Zona>(`${API_BASE}/zonas`, body, idem(key));
  }
  editarZona(id: string, cambios: ZonaCambios): Observable<Zona> {
    return this.http.patch<Zona>(`${API_BASE}/zonas/${id}`, cambios);
  }
  bajaZona(id: string, key: string): Observable<Zona> {
    return this.http.post<Zona>(`${API_BASE}/zonas/${id}/baja`, {}, idem(key));
  }
  restaurarZona(id: string, key: string): Observable<Zona> {
    return this.http.post<Zona>(`${API_BASE}/zonas/${id}/restauracion`, {}, idem(key));
  }

  mesas(cursor?: string): Observable<Pagina<Mesa>> {
    return this.http.get<Pagina<Mesa>>(`${API_BASE}/mesas`, { params: page(cursor) });
  }
  crearMesa(body: MesaNueva, key: string): Observable<Mesa> {
    return this.http.post<Mesa>(`${API_BASE}/mesas`, body, idem(key));
  }
  editarMesa(id: string, cambios: MesaCambios): Observable<Mesa> {
    return this.http.patch<Mesa>(`${API_BASE}/mesas/${id}`, cambios);
  }
  bajaMesa(id: string, motivo: string, key: string): Observable<Mesa> {
    return this.http.post<Mesa>(`${API_BASE}/mesas/${id}/baja`, { motivo }, idem(key));
  }
  restaurarMesa(id: string, key: string): Observable<Mesa> {
    return this.http.post<Mesa>(`${API_BASE}/mesas/${id}/restauracion`, {}, idem(key));
  }
}
```

- [ ] **Step 4: Verificar** — `npx ng test --watch=false --include src/app/features/configuracion/configuracion.forms.spec.ts` → PASS; `npx tsc -p tsconfig.app.json --noEmit` → sin errores.

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add src/app/features/configuracion
git commit -m "I2 paso 4: modelos, normalización de formularios y API de Configuración

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Store — carga, permisos, negocio y sedes

**Files:**
- Create: `src/app/features/configuracion/data-access/configuracion-store.ts`, `configuracion-store.spec.ts`

**Interfaces:**
- Consumes: `ConfiguracionApi`, `AuthSession` (`hasPermission`), `collect`, `describeProblem`, `problemCode`, tipos del Task 3, `FieldProblem` (Task 1).
- Produces (usados por las tareas 5–8): `ConfiguracionStore` con signals `status`, `errors`, `notice`, `busyId`, `negocio`, `sedes`, `zonas`, `mesas`; computed `puedeVerNegocio/EditarNegocio/VerSedes/EditarSedes/VerSalon/EditarSalon`, `sinAcceso`, `sedesSalon`, `sedeActual`, `zonasSede`, `zonasActivasSede`; métodos `load()`, `dismissNotice()`, `elegirSede(id)`, `mesasDeZona(zonaId)`, `editarNegocio(cambios)`, `crearSede(body,key)`, `editarSede(sede,nuevo)`, `bajaSede(sede,motivo,key)`, `restaurarSede(sede,key)`; tipo `ActionError = string | FieldProblem`.

- [ ] **Step 1: Pruebas que fallan** — `configuracion-store.spec.ts`:

```ts
import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import type { Mesa, Negocio, Sucursal, Zona } from '../configuracion.models';
import { ConfiguracionApi } from './configuracion-api';
import { ConfiguracionStore } from './configuracion-store';

const TODOS = [
  'NEGOCIO_VER',
  'NEGOCIO_ADMINISTRAR',
  'SUCURSAL_VER',
  'SUCURSAL_ADMINISTRAR',
  'MESA_VER',
  'MESA_ADMINISTRAR',
];

const api = {
  negocio: vi.fn(),
  editarNegocio: vi.fn(),
  sucursales: vi.fn(),
  crearSucursal: vi.fn(),
  editarSucursal: vi.fn(),
  bajaSucursal: vi.fn(),
  restaurarSucursal: vi.fn(),
  zonas: vi.fn(),
  crearZona: vi.fn(),
  editarZona: vi.fn(),
  bajaZona: vi.fn(),
  restaurarZona: vi.fn(),
  mesas: vi.fn(),
  crearMesa: vi.fn(),
  editarMesa: vi.fn(),
  bajaMesa: vi.fn(),
  restaurarMesa: vi.fn(),
};

let permisos: string[];

const audit = {
  creado_en: '2026-10-21T10:00:00Z',
  creado_por: '1',
  actualizado_en: '2026-10-21T10:00:00Z',
  actualizado_por: '1',
};
const negocio = (over: Partial<Negocio> = {}): Negocio => ({
  id: '1',
  singleton: true,
  ruc: '20123456789',
  razon_social: 'Mi Restaurante SAC',
  nombre_comercial: 'Mi Restaurante',
  direccion_fiscal: 'Av. Principal 123',
  ubigeo: null,
  telefono: null,
  email: null,
  moneda: 'PEN',
  zona_horaria: 'America/Lima',
  es_activo: true,
  ...audit,
  ...over,
});
const sede = (id: string, over: Partial<Sucursal> = {}): Sucursal => ({
  id,
  negocio_id: '1',
  codigo: 'S' + id,
  nombre: 'Sede ' + id,
  direccion: 'Calle ' + id,
  ubigeo: null,
  codigo_establecimiento_sunat: null,
  telefono: null,
  es_activo: true,
  motivo_eliminacion: null,
  eliminado_en: null,
  eliminado_por: null,
  ...audit,
  ...over,
});
const zona = (id: string, sucursal: string, over: Partial<Zona> = {}): Zona => ({
  id,
  sucursal_id: sucursal,
  nombre: 'Zona ' + id,
  orden_visualizacion: 0,
  es_activo: true,
  ...audit,
  ...over,
});
const mesa = (id: string, zonaId: string, sucursal: string, over: Partial<Mesa> = {}): Mesa => ({
  id,
  sucursal_id: sucursal,
  zona_id: zonaId,
  numero_mesa: id,
  capacidad: 4,
  habilitada: true,
  posicion_x: null,
  posicion_y: null,
  es_activo: true,
  motivo_eliminacion: null,
  eliminado_en: null,
  eliminado_por: null,
  ...audit,
  ...over,
});
const lista = <T>(items: T[]) => of({ items, nextCursor: null });
const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

function setup(): ConfiguracionStore {
  TestBed.configureTestingModule({
    providers: [
      ConfiguracionStore,
      { provide: ConfiguracionApi, useValue: api },
      {
        provide: AuthSession,
        useValue: { hasPermission: (p: string) => permisos.includes(p), user: signal({ id: '2' }) },
      },
    ],
  });
  return TestBed.inject(ConfiguracionStore);
}

describe('Estado de Configuración', () => {
  beforeEach(() => {
    permisos = TODOS;
    Object.values(api).forEach((f) => f.mockReset());
    api.negocio.mockReturnValue(of(negocio()));
    api.sucursales.mockReturnValue(lista([sede('1'), sede('2', { es_activo: false })]));
    api.zonas.mockReturnValue(lista([zona('10', '1')]));
    api.mesas.mockReturnValue(lista([mesa('100', '10', '1')]));
  });

  it('carga negocio, sedes, zonas y mesas con todos los permisos', async () => {
    const store = setup();
    await store.load();
    expect(store.status()).toBe('ready');
    expect(store.negocio()?.nombre_comercial).toBe('Mi Restaurante');
    expect(store.sedes()).toHaveLength(2);
    expect(store.zonas()).toHaveLength(1);
    expect(store.mesas()).toHaveLength(1);
    expect(store.puedeEditarNegocio()).toBe(true);
  });

  it('un mesero (solo MESA_VER) solo pide zonas y mesas y muestra «Sede {id}»', async () => {
    permisos = ['MESA_VER'];
    const store = setup();
    await store.load();
    expect(api.negocio).not.toHaveBeenCalled();
    expect(api.sucursales).not.toHaveBeenCalled();
    expect(store.sedesSalon()).toEqual([{ id: '1', nombre: 'Sede 1' }]);
    expect(store.puedeEditarSalon()).toBe(false);
  });

  it('sin ningún permiso no llama a la API', async () => {
    permisos = [];
    const store = setup();
    await store.load();
    expect(store.sinAcceso()).toBe(true);
    expect(Object.values(api).every((f) => f.mock.calls.length === 0)).toBe(true);
    expect(store.status()).toBe('ready');
  });

  it('un fallo en una sección no tumba las demás', async () => {
    api.negocio.mockReturnValue(problem(403, 'sin_permiso', 'No tienes permiso.'));
    const store = setup();
    await store.load();
    expect(store.errors().negocio).toBe('No tienes permiso para realizar esta acción.');
    expect(store.errors().salon).toBeUndefined();
    expect(store.zonas()).toHaveLength(1);
  });

  it('las sedes del salón son solo las activas con nombre real cuando se pueden ver', async () => {
    const store = setup();
    await store.load();
    expect(store.sedesSalon()).toEqual([{ id: '1', nombre: 'Sede 1' }]);
    expect(store.sedeActual()).toBe('1');
  });

  it('editarNegocio guarda los cambios y avisa', async () => {
    api.editarNegocio.mockReturnValue(of(negocio({ nombre_comercial: 'Nuevo' })));
    const store = setup();
    await store.load();
    const result = await store.editarNegocio({ nombre_comercial: 'Nuevo' });
    expect(result).toBeUndefined();
    expect(store.negocio()?.nombre_comercial).toBe('Nuevo');
    expect(store.notice()?.kind).toBe('ok');
  });

  it('editarNegocio sin cambios no llama a la API', async () => {
    const store = setup();
    await store.load();
    expect(await store.editarNegocio({})).toBeUndefined();
    expect(api.editarNegocio).not.toHaveBeenCalled();
  });

  it('un rechazo de campo vuelve como {field, message}', async () => {
    api.editarNegocio.mockReturnValue(problem(422, 'moneda_invalida', 'La moneda no es válida.'));
    const store = setup();
    await store.load();
    expect(await store.editarNegocio({ moneda: 'xx' })).toEqual({
      field: 'moneda',
      message: 'La moneda no es válida.',
    });
  });

  it('crearSede refresca la lista y recuerda asignarla en Equipo y accesos', async () => {
    api.crearSucursal.mockReturnValue(of(sede('3')));
    api.sucursales.mockReturnValueOnce(lista([sede('1')])).mockReturnValue(lista([sede('1'), sede('3')]));
    const store = setup();
    await store.load();
    const result = await store.crearSede(
      { codigo: 'S3', nombre: 'Sede 3', direccion: 'Calle 3' },
      'k1',
    );
    expect(result).toBeUndefined();
    expect(api.crearSucursal).toHaveBeenCalledWith(
      { codigo: 'S3', nombre: 'Sede 3', direccion: 'Calle 3' },
      'k1',
    );
    expect(store.sedes()).toHaveLength(2);
    expect(store.notice()?.text).toContain('Asígnala a tu equipo en Equipo y accesos');
  });

  it('código duplicado de sede vuelve como error del campo codigo', async () => {
    api.crearSucursal.mockReturnValue(problem(409, 'codigo_duplicado', 'Ya existe ese código.'));
    const store = setup();
    await store.load();
    expect(
      await store.crearSede({ codigo: 'S1', nombre: 'x', direccion: 'y' }, 'k1'),
    ).toEqual({ field: 'codigo', message: 'Ya existe ese código.' });
  });

  it('editarSede envía solo lo cambiado y no llama si nada cambió', async () => {
    api.editarSucursal.mockReturnValue(of(sede('1', { nombre: 'Centro' })));
    const store = setup();
    await store.load();
    const original = store.sedes()[0];
    expect(await store.editarSede(original, {})).toBeUndefined();
    expect(api.editarSucursal).not.toHaveBeenCalled();
    await store.editarSede(original, { nombre: 'Centro' });
    expect(api.editarSucursal).toHaveBeenCalledWith('1', { nombre: 'Centro' });
  });

  it('bajaSede con la última sede activa devuelve el mensaje del servidor', async () => {
    api.bajaSucursal.mockReturnValue(
      problem(422, 'ultima_sede_activa', 'No se puede dar de baja la última sucursal activa.'),
    );
    const store = setup();
    await store.load();
    expect(await store.bajaSede(store.sedes()[0], 'Cierre', 'k1')).toBe(
      'No se puede dar de baja la última sucursal activa.',
    );
  });

  it('restaurarSede avisa y no permite dos acciones a la vez', async () => {
    let liberar: (v: Sucursal) => void = () => undefined;
    api.restaurarSucursal.mockReturnValue({
      subscribe: (o: { next: (v: Sucursal) => void; complete: () => void }) => {
        liberar = (v) => {
          o.next(v);
          o.complete();
        };
        return { unsubscribe() {} };
      },
    });
    const store = setup();
    await store.load();
    const inactiva = store.sedes()[1];
    const primera = store.restaurarSede(inactiva, 'k1');
    expect(store.busyId()).toBe('2');
    await store.restaurarSede(inactiva, 'k2');
    expect(api.restaurarSucursal).toHaveBeenCalledTimes(1);
    liberar(sede('2'));
    await primera;
    expect(store.busyId()).toBeNull();
  });
});
```

- [ ] **Step 2: Verificar que fallan** — `npx ng test --watch=false --include src/app/features/configuracion/data-access/configuracion-store.spec.ts` → FAIL (store inexistente).

- [ ] **Step 3: Implementar** — `configuracion-store.ts` (las acciones de zonas y mesas se añaden en el Task 5):

```ts
import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem, problemCode } from '../../../core/http/problem';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import { cambiosDe } from '../configuracion.forms';
import type {
  Mesa,
  Negocio,
  NegocioCambios,
  SedeRef,
  Sucursal,
  SucursalCambios,
  SucursalNueva,
  Zona,
} from '../configuracion.models';
import { ConfiguracionApi } from './configuracion-api';

export const NEGOCIO_VER = 'NEGOCIO_VER';
export const NEGOCIO_ADMINISTRAR = 'NEGOCIO_ADMINISTRAR';
export const SUCURSAL_VER = 'SUCURSAL_VER';
export const SUCURSAL_ADMINISTRAR = 'SUCURSAL_ADMINISTRAR';
export const MESA_VER = 'MESA_VER';
export const MESA_ADMINISTRAR = 'MESA_ADMINISTRAR';

export interface ConfiguracionNotice {
  kind: 'ok' | 'error';
  text: string;
}

export interface SectionErrors {
  negocio?: string;
  sedes?: string;
  salon?: string;
}

/** Resultado de una acción de diálogo: undefined = éxito; texto = aviso; FieldProblem = campo. */
export type ActionError = string | FieldProblem;

/** Código de rechazo del servidor -> campo del formulario que lo causó. */
const FIELD_BY_CODE: Record<string, string> = {
  razon_social_invalida: 'razon_social',
  nombre_comercial_invalido: 'nombre_comercial',
  direccion_fiscal_invalida: 'direccion_fiscal',
  moneda_invalida: 'moneda',
  zona_horaria_invalida: 'zona_horaria',
  email_invalido: 'email',
  codigo_duplicado: 'codigo',
  codigo_invalido: 'codigo',
  direccion_invalida: 'direccion',
  ubigeo_invalido: 'ubigeo',
  telefono_invalido: 'telefono',
  codigo_establecimiento_sunat_invalido: 'codigo_establecimiento_sunat',
  nombre_duplicado: 'nombre',
  nombre_invalido: 'nombre',
  orden_invalido: 'orden_visualizacion',
  numero_mesa_duplicado: 'numero_mesa',
  numero_mesa_invalido: 'numero_mesa',
  capacidad_invalida: 'capacidad',
  posicion_x_invalida: 'posicion_x',
  posicion_y_invalida: 'posicion_y',
  zona_invalida: 'zona_id',
  motivo_requerido: 'motivo',
};

@Injectable()
export class ConfiguracionStore {
  protected readonly api = inject(ConfiguracionApi);
  private readonly auth = inject(AuthSession);

  readonly status = signal<'loading' | 'ready'>('loading');
  readonly errors = signal<SectionErrors>({});
  readonly notice = signal<ConfiguracionNotice | null>(null);
  readonly busyId = signal<string | null>(null);
  readonly negocio = signal<Negocio | null>(null);
  readonly sedes = signal<Sucursal[]>([]);
  readonly zonas = signal<Zona[]>([]);
  readonly mesas = signal<Mesa[]>([]);
  private readonly sedeElegida = signal<string | null>(null);

  readonly puedeVerNegocio = computed(() => this.tiene(NEGOCIO_VER) || this.tiene(NEGOCIO_ADMINISTRAR));
  readonly puedeEditarNegocio = computed(() => this.tiene(NEGOCIO_ADMINISTRAR));
  readonly puedeVerSedes = computed(() => this.tiene(SUCURSAL_VER) || this.tiene(SUCURSAL_ADMINISTRAR));
  readonly puedeEditarSedes = computed(() => this.tiene(SUCURSAL_ADMINISTRAR));
  readonly puedeVerSalon = computed(() => this.tiene(MESA_VER) || this.tiene(MESA_ADMINISTRAR));
  readonly puedeEditarSalon = computed(() => this.tiene(MESA_ADMINISTRAR));
  readonly sinAcceso = computed(
    () => !this.puedeVerNegocio() && !this.puedeVerSedes() && !this.puedeVerSalon(),
  );

  /** Sedes elegibles: con permiso de sedes, las activas con su nombre; si no, las que aparecen en los datos. */
  readonly sedesSalon = computed<SedeRef[]>(() => {
    if (this.puedeVerSedes()) {
      return this.sedes()
        .filter((s) => s.es_activo)
        .map((s) => ({ id: s.id, nombre: s.nombre }));
    }
    const ids = new Set([
      ...this.zonas().map((z) => z.sucursal_id),
      ...this.mesas().map((m) => m.sucursal_id),
    ]);
    return [...ids]
      .sort((a, b) => Number(a) - Number(b))
      .map((id) => ({ id, nombre: `Sede ${id}` }));
  });

  readonly sedeActual = computed<string | null>(() => {
    const lista = this.sedesSalon();
    return lista.find((s) => s.id === this.sedeElegida())?.id ?? lista[0]?.id ?? null;
  });

  readonly zonasSede = computed(() =>
    this.zonas()
      .filter((z) => z.sucursal_id === this.sedeActual())
      .sort(
        (a, b) =>
          a.orden_visualizacion - b.orden_visualizacion || a.nombre.localeCompare(b.nombre, 'es'),
      ),
  );
  readonly zonasActivasSede = computed(() => this.zonasSede().filter((z) => z.es_activo));

  elegirSede(id: string): void {
    this.sedeElegida.set(id);
  }

  mesasDeZona(zonaId: string): Mesa[] {
    return this.mesas()
      .filter((m) => m.zona_id === zonaId)
      .sort((a, b) => a.numero_mesa.localeCompare(b.numero_mesa, 'es', { numeric: true }));
  }

  async load(): Promise<void> {
    this.status.set('loading');
    this.errors.set({});
    if (this.sinAcceso()) {
      this.status.set('ready');
      return;
    }
    await Promise.all([this.cargarNegocio(), this.cargarSedes(), this.cargarSalon()]);
    this.status.set('ready');
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  // --- Negocio ---------------------------------------------------------------------------

  async editarNegocio(cambios: NegocioCambios): Promise<ActionError | undefined> {
    if (Object.keys(cambios).length === 0) return undefined;
    try {
      this.negocio.set(await firstValueFrom(this.api.editarNegocio(cambios)));
    } catch (error) {
      return this.comoError(error);
    }
    this.say('ok', 'Guardamos los datos de tu restaurante.');
    return undefined;
  }

  // --- Sedes -----------------------------------------------------------------------------

  async crearSede(body: SucursalNueva, key: string): Promise<ActionError | undefined> {
    return this.mutarSedes(
      () => firstValueFrom(this.api.crearSucursal(body, key)),
      `Se creó la sede ${body.nombre}. Asígnala a tu equipo en Equipo y accesos para que alguien pueda operarla.`,
    );
  }

  async editarSede(
    sede: Sucursal,
    nuevo: Record<string, string | null>,
  ): Promise<ActionError | undefined> {
    const cambios = cambiosDe({ ...sede }, nuevo) as SucursalCambios;
    if (Object.keys(cambios).length === 0) return undefined;
    return this.mutarSedes(
      () => firstValueFrom(this.api.editarSucursal(sede.id, cambios)),
      `Guardamos los cambios de la sede ${sede.nombre}.`,
    );
  }

  async bajaSede(sede: Sucursal, motivo: string, key: string): Promise<ActionError | undefined> {
    return this.mutarSedes(
      () => firstValueFrom(this.api.bajaSucursal(sede.id, motivo, key)),
      `La sede ${sede.nombre} quedó de baja.`,
    );
  }

  async restaurarSede(sede: Sucursal, key: string): Promise<void> {
    await this.conBloqueo(sede.id, async () => {
      await firstValueFrom(this.api.restaurarSucursal(sede.id, key));
      await this.refrescarSedes();
      this.say('ok', `La sede ${sede.nombre} volvió a estar activa.`);
    });
  }

  // --- Internos --------------------------------------------------------------------------

  protected tiene(permiso: string): boolean {
    return this.auth.hasPermission(permiso);
  }

  protected say(kind: 'ok' | 'error', text: string): void {
    this.notice.set({ kind, text });
  }

  protected fallo(seccion: keyof SectionErrors, error: unknown): void {
    this.errors.update((actual) => ({ ...actual, [seccion]: describeProblem(error) }));
  }

  /** Error del servidor -> texto general o error de un campo, según su código. */
  protected comoError(error: unknown): ActionError {
    const message = describeProblem(error);
    const code = problemCode(error);
    const field = code ? FIELD_BY_CODE[code] : undefined;
    return field ? { field, message } : message;
  }

  /** Ejecuta una acción sin diálogo (interruptor, restaurar): una a la vez y con aviso de error. */
  protected async conBloqueo(id: string, accion: () => Promise<void>): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(id);
    try {
      await accion();
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  private async mutarSedes(
    accion: () => Promise<unknown>,
    ok: string,
  ): Promise<ActionError | undefined> {
    try {
      await accion();
    } catch (error) {
      return this.comoError(error);
    }
    await this.refrescarSedes();
    this.say('ok', ok);
    return undefined;
  }

  private async cargarNegocio(): Promise<void> {
    if (!this.puedeVerNegocio()) return;
    try {
      this.negocio.set(await firstValueFrom(this.api.negocio()));
    } catch (error) {
      this.fallo('negocio', error);
    }
  }

  private async cargarSedes(): Promise<void> {
    if (!this.puedeVerSedes()) return;
    try {
      this.sedes.set(await collect((c) => firstValueFrom(this.api.sucursales(c))));
    } catch (error) {
      this.fallo('sedes', error);
    }
  }

  protected async cargarSalon(): Promise<void> {
    if (!this.puedeVerSalon()) return;
    try {
      const [zonas, mesas] = await Promise.all([
        collect((c) => firstValueFrom(this.api.zonas(c))),
        collect((c) => firstValueFrom(this.api.mesas(c))),
      ]);
      this.zonas.set(zonas);
      this.mesas.set(mesas);
    } catch (error) {
      this.fallo('salon', error);
    }
  }

  private async refrescarSedes(): Promise<void> {
    try {
      this.sedes.set(await collect((c) => firstValueFrom(this.api.sucursales(c))));
    } catch (error) {
      this.say('error', `No se pudo actualizar la lista de sedes: ${describeProblem(error)}`);
    }
  }
}
```

- [ ] **Step 4: Verificar** — `npx ng test --watch=false --include src/app/features/configuracion/data-access/configuracion-store.spec.ts` → PASS (12 pruebas).

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add src/app/features/configuracion
git commit -m "I2 paso 4: store de Configuración con carga por permisos, negocio y sedes

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Store — zonas y mesas

**Files:**
- Modify: `src/app/features/configuracion/data-access/configuracion-store.ts`, `configuracion-store.spec.ts`

**Interfaces:**
- Consumes: Task 4 (`mutar…`, `conBloqueo`, `comoError`, `cargarSalon`, `say`, `tiene`).
- Produces: `crearZona(body, key)`, `editarZona(zona, nuevo)`, `bajaZona(zona, key)`, `restaurarZona(zona, key)`, `crearMesa(body, key)`, `editarMesa(mesa, nuevo)`, `habilitarMesa(mesa)`, `bajaMesa(mesa, motivo, key)`, `restaurarMesa(mesa, key)`.

- [ ] **Step 1: Pruebas que fallan** — añadir dentro del `describe` de `configuracion-store.spec.ts`:

```ts
  it('crearZona refresca zonas y mesas y avisa', async () => {
    api.crearZona.mockReturnValue(of(zona('11', '1', { nombre: 'Terraza' })));
    const store = setup();
    await store.load();
    api.zonas.mockReturnValue(lista([zona('10', '1'), zona('11', '1', { nombre: 'Terraza' })]));
    const result = await store.crearZona({ sucursal_id: '1', nombre: 'Terraza' }, 'k1');
    expect(result).toBeUndefined();
    expect(store.zonas()).toHaveLength(2);
    expect(store.notice()?.text).toContain('Terraza');
  });

  it('nombre duplicado de zona vuelve como error del campo nombre', async () => {
    api.crearZona.mockReturnValue(problem(409, 'nombre_duplicado', 'Ya existe una zona con ese nombre.'));
    const store = setup();
    await store.load();
    expect(await store.crearZona({ sucursal_id: '1', nombre: 'Zona 10' }, 'k1')).toEqual({
      field: 'nombre',
      message: 'Ya existe una zona con ese nombre.',
    });
  });

  it('bajaZona con mesas activas devuelve el mensaje del servidor como aviso general', async () => {
    api.bajaZona.mockReturnValue(problem(409, 'zona_con_mesas', 'La zona tiene mesas activas.'));
    const store = setup();
    await store.load();
    expect(await store.bajaZona(store.zonas()[0], 'k1')).toBe('La zona tiene mesas activas.');
  });

  it('editarZona sin cambios no llama a la API', async () => {
    const store = setup();
    await store.load();
    const z = store.zonas()[0];
    expect(await store.editarZona(z, { nombre: z.nombre, orden_visualizacion: 0 })).toBeUndefined();
    expect(api.editarZona).not.toHaveBeenCalled();
  });

  it('crearMesa con número repetido vuelve como error del campo numero_mesa', async () => {
    api.crearMesa.mockReturnValue(problem(409, 'numero_mesa_duplicado', 'Ya existe esa mesa.'));
    const store = setup();
    await store.load();
    expect(
      await store.crearMesa(
        { sucursal_id: '1', zona_id: '10', numero_mesa: '100', capacidad: 4 },
        'k1',
      ),
    ).toEqual({ field: 'numero_mesa', message: 'Ya existe esa mesa.' });
  });

  it('editarMesa envía solo lo cambiado, incluido vaciar la posición con null', async () => {
    api.editarMesa.mockReturnValue(of(mesa('100', '10', '1', { capacidad: 6 })));
    const store = setup();
    await store.load();
    const m = store.mesas()[0];
    await store.editarMesa(m, {
      zona_id: '10',
      numero_mesa: '100',
      capacidad: 6,
      posicion_x: null,
      posicion_y: null,
    });
    expect(api.editarMesa).toHaveBeenCalledWith('100', { capacidad: 6 });
  });

  it('habilitarMesa alterna el estado en la fila y no deja avisos de error', async () => {
    api.editarMesa.mockReturnValue(of(mesa('100', '10', '1', { habilitada: false })));
    const store = setup();
    await store.load();
    await store.habilitarMesa(store.mesas()[0]);
    expect(api.editarMesa).toHaveBeenCalledWith('100', { habilitada: false });
    expect(store.mesas()[0].habilitada).toBe(false);
    expect(store.busyId()).toBeNull();
  });

  it('habilitarMesa con error muestra el aviso y libera el bloqueo', async () => {
    api.editarMesa.mockReturnValue(problem(409, 'mesa_inactiva', 'La mesa está dada de baja.'));
    const store = setup();
    await store.load();
    await store.habilitarMesa(store.mesas()[0]);
    expect(store.notice()).toEqual({ kind: 'error', text: 'La mesa está dada de baja.' });
    expect(store.busyId()).toBeNull();
  });

  it('bajaMesa sin motivo del servidor vuelve como error del campo motivo', async () => {
    api.bajaMesa.mockReturnValue(problem(422, 'motivo_requerido', 'Indica el motivo.'));
    const store = setup();
    await store.load();
    expect(await store.bajaMesa(store.mesas()[0], 'x', 'k1')).toEqual({
      field: 'motivo',
      message: 'Indica el motivo.',
    });
  });

  it('restaurarMesa con la zona dada de baja avisa con el mensaje del servidor', async () => {
    api.restaurarMesa.mockReturnValue(
      problem(422, 'regla_de_datos', 'No se puede restaurar la mesa porque su zona está dada de baja.'),
    );
    const store = setup();
    await store.load();
    await store.restaurarMesa(store.mesas()[0], 'k1');
    expect(store.notice()?.kind).toBe('error');
    expect(store.notice()?.text).toContain('su zona está dada de baja');
  });
```

- [ ] **Step 2: Verificar que fallan** — misma orden del Task 4 → FAIL (métodos inexistentes).

- [ ] **Step 3: Implementar** — en `configuracion-store.ts` añadir los imports de tipos `MesaCambios`, `MesaNueva`, `ZonaCambios`, `ZonaNueva` y, antes de `// --- Internos`, el bloque:

```ts
  // --- Zonas -----------------------------------------------------------------------------

  async crearZona(body: ZonaNueva, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.crearZona(body, key)),
      `Se creó la zona ${body.nombre}.`,
    );
  }

  async editarZona(
    zona: Zona,
    nuevo: Record<string, string | number>,
  ): Promise<ActionError | undefined> {
    const cambios = cambiosDe({ ...zona }, nuevo) as ZonaCambios;
    if (Object.keys(cambios).length === 0) return undefined;
    return this.mutarSalon(
      () => firstValueFrom(this.api.editarZona(zona.id, cambios)),
      `Guardamos los cambios de la zona ${zona.nombre}.`,
    );
  }

  async bajaZona(zona: Zona, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.bajaZona(zona.id, key)),
      `La zona ${zona.nombre} quedó de baja.`,
    );
  }

  async restaurarZona(zona: Zona, key: string): Promise<void> {
    await this.conBloqueo(zona.id, async () => {
      await firstValueFrom(this.api.restaurarZona(zona.id, key));
      await this.cargarSalon();
      this.say('ok', `La zona ${zona.nombre} volvió a estar activa.`);
    });
  }

  // --- Mesas -----------------------------------------------------------------------------

  async crearMesa(body: MesaNueva, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.crearMesa(body, key)),
      `Se creó la mesa ${body.numero_mesa}.`,
    );
  }

  async editarMesa(
    mesa: Mesa,
    nuevo: Record<string, string | number | null>,
  ): Promise<ActionError | undefined> {
    const cambios = cambiosDe({ ...mesa }, nuevo) as MesaCambios;
    if (Object.keys(cambios).length === 0) return undefined;
    return this.mutarSalon(
      () => firstValueFrom(this.api.editarMesa(mesa.id, cambios)),
      `Guardamos los cambios de la mesa ${mesa.numero_mesa}.`,
    );
  }

  /** Interruptor «habilitada»: sin diálogo ni motivo; actualiza solo esa fila. */
  async habilitarMesa(mesa: Mesa): Promise<void> {
    await this.conBloqueo(mesa.id, async () => {
      const actualizada = await firstValueFrom(
        this.api.editarMesa(mesa.id, { habilitada: !mesa.habilitada }),
      );
      this.mesas.update((lista) => lista.map((m) => (m.id === actualizada.id ? actualizada : m)));
    });
  }

  async bajaMesa(mesa: Mesa, motivo: string, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.bajaMesa(mesa.id, motivo, key)),
      `La mesa ${mesa.numero_mesa} quedó de baja.`,
    );
  }

  async restaurarMesa(mesa: Mesa, key: string): Promise<void> {
    await this.conBloqueo(mesa.id, async () => {
      await firstValueFrom(this.api.restaurarMesa(mesa.id, key));
      await this.cargarSalon();
      this.say('ok', `La mesa ${mesa.numero_mesa} volvió a estar activa.`);
    });
  }
```

y, en los internos, el método:

```ts
  private async mutarSalon(
    accion: () => Promise<unknown>,
    ok: string,
  ): Promise<ActionError | undefined> {
    try {
      await accion();
    } catch (error) {
      return this.comoError(error);
    }
    await this.cargarSalon();
    this.say('ok', ok);
    return undefined;
  }
```

Nota: `cargarSalon()` ya registra su propio fallo en `errors().salon`; tras una mutación exitosa eso deja el aviso de éxito y el error visible en la sección, que ofrece «Reintentar».

- [ ] **Step 4: Verificar** — `npx ng test --watch=false --include src/app/features/configuracion/data-access/configuracion-store.spec.ts` → PASS (22 pruebas).

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add src/app/features/configuracion
git commit -m "I2 paso 4: acciones de zonas y mesas en el store de Configuración

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Secciones «Tu restaurante» y «Sedes»

**Files:**
- Create: `src/app/features/configuracion/negocio-section.ts`, `sedes-section.ts`, `negocio-section.spec.ts`, `sedes-section.spec.ts`

**Interfaces:**
- Consumes: `ConfiguracionStore` (inyectado; lo provee la pantalla real en el Task 8), `Dialogs`, `newIdempotencyKey`, helpers `texto`/`opcional`, `app-page-header` no se usa aquí.
- Produces: `NegocioSection` (selector `app-negocio-section`) y `SedesSection` (selector `app-sedes-section`).

- [ ] **Step 1: Pruebas que fallan**

`negocio-section.spec.ts`:

```ts
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { NegocioSection } from './negocio-section';
import type { Negocio } from './configuracion.models';

const negocio: Negocio = {
  id: '1',
  singleton: true,
  ruc: '20123456789',
  razon_social: 'Mi Restaurante SAC',
  nombre_comercial: 'Mi Restaurante',
  direccion_fiscal: 'Av. Principal 123',
  ubigeo: null,
  telefono: null,
  email: null,
  moneda: 'PEN',
  zona_horaria: 'America/Lima',
  es_activo: true,
  creado_en: '',
  creado_por: '1',
  actualizado_en: '',
  actualizado_por: '1',
};

function setup(editar: boolean, editarNegocio = vi.fn().mockResolvedValue(undefined)) {
  const store = {
    negocio: signal<Negocio | null>(negocio),
    errors: signal({}),
    puedeEditarNegocio: signal(editar),
    editarNegocio,
    load: vi.fn(),
  };
  TestBed.configureTestingModule({
    imports: [NegocioSection],
    providers: [{ provide: ConfiguracionStore, useValue: store }],
  });
  const fixture = TestBed.createComponent(NegocioSection);
  fixture.detectChanges();
  return { fixture, store, el: fixture.nativeElement as HTMLElement };
}

describe('Sección Tu restaurante', () => {
  it('muestra el RUC sin poder editarlo', () => {
    const { el } = setup(true);
    const ruc = el.querySelector('#negocio-ruc') as HTMLInputElement;
    expect(ruc.value).toBe('20123456789');
    expect(ruc.readOnly).toBe(true);
  });

  it('sin permiso de edición queda en solo lectura y sin botón de guardar', () => {
    const { el } = setup(false);
    expect((el.querySelector('#negocio-nombre_comercial') as HTMLInputElement).readOnly).toBe(true);
    expect(el.querySelector('button[type="submit"]')).toBeNull();
  });

  it('guarda solo lo que cambió', async () => {
    const { fixture, store, el } = setup(true);
    const input = el.querySelector('#negocio-nombre_comercial') as HTMLInputElement;
    input.value = 'Nuevo nombre';
    input.dispatchEvent(new Event('input'));
    (el.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    expect(store.editarNegocio).toHaveBeenCalledWith({ nombre_comercial: 'Nuevo nombre' });
  });

  it('pinta el error de campo que devuelve el store y conserva lo escrito', async () => {
    const editar = vi.fn().mockResolvedValue({ field: 'moneda', message: 'La moneda no es válida.' });
    const { fixture, el } = setup(true, editar);
    const moneda = el.querySelector('#negocio-moneda') as HTMLInputElement;
    moneda.value = 'xx';
    moneda.dispatchEvent(new Event('input'));
    (el.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(el.textContent).toContain('La moneda no es válida.');
    expect((el.querySelector('#negocio-moneda') as HTMLInputElement).value).toBe('xx');
  });
});
```

`sedes-section.spec.ts`:

```ts
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs } from '../../shared/ui/form-dialog';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { SedesSection } from './sedes-section';
import type { Sucursal } from './configuracion.models';

const sede = (id: string, activa: boolean): Sucursal => ({
  id,
  negocio_id: '1',
  codigo: 'S' + id,
  nombre: 'Sede ' + id,
  direccion: 'Calle ' + id,
  ubigeo: null,
  codigo_establecimiento_sunat: null,
  telefono: null,
  es_activo: activa,
  motivo_eliminacion: activa ? null : 'Cierre',
  eliminado_en: null,
  eliminado_por: null,
  creado_en: '',
  creado_por: '1',
  actualizado_en: '',
  actualizado_por: '1',
});

function setup(editar: boolean) {
  const store = {
    sedes: signal([sede('1', true), sede('2', false)]),
    errors: signal({}),
    busyId: signal<string | null>(null),
    puedeEditarSedes: signal(editar),
    restaurarSede: vi.fn(),
    load: vi.fn(),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [SedesSection],
    providers: [
      { provide: ConfiguracionStore, useValue: store },
      { provide: Dialogs, useValue: dialogs },
    ],
  });
  const fixture = TestBed.createComponent(SedesSection);
  fixture.detectChanges();
  return { store, dialogs, el: fixture.nativeElement as HTMLElement };
}

describe('Sección Sedes', () => {
  it('lista las sedes y su estado', () => {
    const { el } = setup(true);
    expect(el.textContent).toContain('Sede 1');
    expect(el.textContent).toContain('De baja');
  });

  it('con permiso ofrece alta, edición, baja y restauración', () => {
    const { el } = setup(true);
    const textos = [...el.querySelectorAll('button')].map((b) => b.textContent?.trim());
    expect(textos).toEqual(
      expect.arrayContaining(['Nueva sede', 'Editar', 'Dar de baja', 'Restaurar']),
    );
  });

  it('sin permiso de escritura no hay botones de acción', () => {
    const { el } = setup(false);
    expect(el.querySelectorAll('button')).toHaveLength(0);
  });

  it('dar de baja abre un diálogo que exige motivo', () => {
    const { el, dialogs } = setup(true);
    const baja = [...el.querySelectorAll('button')].find((b) => b.textContent?.trim() === 'Dar de baja');
    baja?.click();
    const data = dialogs.form.mock.calls[0][0];
    expect(data.danger).toBe(true);
    expect(data.fields.map((f: { key: string }) => f.key)).toEqual(['motivo']);
  });

  it('restaurar no abre diálogo y usa una clave de idempotencia', () => {
    const { el, dialogs, store } = setup(true);
    const restaurar = [...el.querySelectorAll('button')].find((b) => b.textContent?.trim() === 'Restaurar');
    restaurar?.click();
    expect(dialogs.form).not.toHaveBeenCalled();
    expect(store.restaurarSede).toHaveBeenCalledWith(expect.objectContaining({ id: '2' }), expect.any(String));
  });
});
```

- [ ] **Step 2: Verificar que fallan** — `npx ng test --watch=false --include "src/app/features/configuracion/*-section.spec.ts"` → FAIL (componentes inexistentes).

- [ ] **Step 3: Implementar**

`negocio-section.ts`:

```ts
import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { cambiosDe, opcional, texto } from './configuracion.forms';
import { ConfiguracionStore } from './data-access/configuracion-store';

const CAMPOS = [
  { key: 'razon_social', label: 'Razón social', opcional: false },
  { key: 'nombre_comercial', label: 'Nombre comercial', opcional: false },
  { key: 'direccion_fiscal', label: 'Dirección fiscal', opcional: false },
  { key: 'ubigeo', label: 'Ubigeo (6 dígitos)', opcional: true },
  { key: 'telefono', label: 'Teléfono', opcional: true },
  { key: 'email', label: 'Correo', opcional: true },
  { key: 'moneda', label: 'Moneda (código de 3 letras)', opcional: false },
  { key: 'zona_horaria', label: 'Zona horaria', opcional: false },
] as const;

@Component({
  selector: 'app-negocio-section',
  imports: [ReactiveFormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="negocio-title">
      <div class="panel-header">
        <h2 id="negocio-title" class="font-bold">Tu restaurante</h2>
      </div>
      <div class="panel-body">
        @if (store.errors().negocio; as error) {
          <div class="empty" role="alert">
            <h3>No pudimos cargar los datos de tu restaurante</h3>
            <p>{{ error }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        } @else if (store.negocio()) {
          <form class="settings-form" [formGroup]="form" (ngSubmit)="guardar()">
            <div class="form-grid">
              <label class="field" for="negocio-ruc">
                <span>RUC</span>
                <input id="negocio-ruc" [value]="store.negocio()?.ruc" readonly />
                <small>El RUC no se puede cambiar.</small>
              </label>
              @for (campo of campos; track campo.key) {
                <label class="field" [for]="'negocio-' + campo.key">
                  <span>{{ campo.label }}</span>
                  <input
                    [id]="'negocio-' + campo.key"
                    [formControlName]="campo.key"
                    [readonly]="!store.puedeEditarNegocio()"
                    [attr.aria-invalid]="!!errores()[campo.key]"
                  />
                  @if (errores()[campo.key]; as message) {
                    <small class="text-red-700" role="alert">{{ message }}</small>
                  }
                </label>
              }
            </div>
            @if (aviso()) {
              <p class="form-error" role="alert">{{ aviso() }}</p>
            }
            @if (store.puedeEditarNegocio()) {
              <button type="submit" class="btn primary" [disabled]="guardando()">
                Guardar cambios
              </button>
            } @else {
              <p class="note-box" role="status">Solo puedes consultar estos datos.</p>
            }
          </form>
        }
      </div>
    </section>
  `,
})
export class NegocioSection {
  readonly store = inject(ConfiguracionStore);
  readonly campos = CAMPOS;
  readonly errores = signal<Record<string, string>>({});
  readonly aviso = signal('');
  readonly guardando = signal(false);

  readonly form = new FormGroup(
    Object.fromEntries(CAMPOS.map((c) => [c.key, new FormControl('', { nonNullable: true })])),
  );

  private readonly actual = computed(() => this.store.negocio());

  constructor() {
    effect(() => {
      const negocio = this.actual();
      if (!negocio) return;
      this.form.reset(
        Object.fromEntries(
          CAMPOS.map((c) => [c.key, (negocio as unknown as Record<string, string | null>)[c.key] ?? '']),
        ),
      );
    });
    this.form.valueChanges.subscribe(() => {
      if (Object.keys(this.errores()).length) this.errores.set({});
    });
  }

  async guardar(): Promise<void> {
    const negocio = this.actual();
    if (!negocio || this.guardando()) return;
    const valores = this.form.getRawValue();
    const nuevo = Object.fromEntries(
      CAMPOS.map((c) => [c.key, c.opcional ? opcional(valores[c.key]) : texto(valores[c.key])]),
    );
    const cambios = cambiosDe({ ...negocio }, nuevo);
    this.aviso.set('');
    this.errores.set({});
    this.guardando.set(true);
    try {
      const resultado = await this.store.editarNegocio(cambios);
      if (typeof resultado === 'string') this.aviso.set(resultado);
      else if (resultado) this.errores.set({ [resultado.field]: resultado.message });
    } finally {
      this.guardando.set(false);
    }
  }
}
```

`sedes-section.ts`:

```ts
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs } from '../../shared/ui/form-dialog';
import { opcional, texto } from './configuracion.forms';
import type { Sucursal } from './configuracion.models';
import { ConfiguracionStore } from './data-access/configuracion-store';

@Component({
  selector: 'app-sedes-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel mt-6" aria-labelledby="sedes-title">
      <div class="panel-header">
        <h2 id="sedes-title" class="font-bold">Sedes</h2>
        @if (store.puedeEditarSedes()) {
          <button type="button" class="btn primary" (click)="crear()">Nueva sede</button>
        }
      </div>
      <div class="panel-body">
        @if (store.errors().sedes; as error) {
          <div class="empty" role="alert">
            <h3>No pudimos cargar las sedes</h3>
            <p>{{ error }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        } @else {
          @for (sede of store.sedes(); track sede.id) {
            <article class="setting-block" [class.inactive]="!sede.es_activo">
              <div>
                <h3>{{ sede.nombre }} <span class="subtle">({{ sede.codigo }})</span></h3>
                <p>{{ sede.direccion }}</p>
                @if (!sede.es_activo && sede.motivo_eliminacion) {
                  <p class="subtle">Motivo de la baja: {{ sede.motivo_eliminacion }}</p>
                }
              </div>
              <div class="inline-actions">
                <span class="pill" [class.green]="sede.es_activo">
                  {{ sede.es_activo ? 'Activa' : 'De baja' }}
                </span>
                @if (store.puedeEditarSedes()) {
                  @if (sede.es_activo) {
                    <button type="button" class="btn" (click)="editar(sede)">Editar</button>
                    <button type="button" class="btn danger" (click)="baja(sede)">Dar de baja</button>
                  } @else {
                    <button
                      type="button"
                      class="btn"
                      [disabled]="store.busyId() !== null"
                      (click)="restaurar(sede)"
                    >
                      Restaurar
                    </button>
                  }
                }
              </div>
            </article>
          } @empty {
            <p class="empty" role="status">Todavía no hay sedes registradas.</p>
          }
        }
      </div>
    </section>
  `,
})
export class SedesSection {
  readonly store = inject(ConfiguracionStore);
  private readonly dialogs = inject(Dialogs);

  crear(): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva sede',
      description: 'Después de crearla, asígnala a tu equipo en Equipo y accesos.',
      submit: 'Crear sede',
      fields: [
        { key: 'codigo', label: 'Código', hint: 'De 2 a 20 caracteres: mayúsculas, números, _ o -.' },
        { key: 'nombre', label: 'Nombre' },
        { key: 'direccion', label: 'Dirección' },
        { key: 'ubigeo', label: 'Ubigeo (6 dígitos)', required: false },
        { key: 'codigo_establecimiento_sunat', label: 'Código de establecimiento SUNAT', required: false },
        { key: 'telefono', label: 'Teléfono', type: 'tel', required: false },
      ],
      commit: (v) =>
        this.store.crearSede(
          {
            codigo: texto(v['codigo']).toUpperCase(),
            nombre: texto(v['nombre']),
            direccion: texto(v['direccion']),
            ...this.opcionales(v),
          },
          key,
        ),
    });
  }

  editar(sede: Sucursal): void {
    this.dialogs.form({
      title: 'Editar sede: ' + sede.nombre,
      description: 'Cambia solo lo necesario; lo que dejes igual no se envía.',
      submit: 'Guardar cambios',
      fields: [
        { key: 'codigo', label: 'Código', value: sede.codigo },
        { key: 'nombre', label: 'Nombre', value: sede.nombre },
        { key: 'direccion', label: 'Dirección', value: sede.direccion },
        { key: 'ubigeo', label: 'Ubigeo (6 dígitos)', required: false, value: sede.ubigeo ?? '' },
        {
          key: 'codigo_establecimiento_sunat',
          label: 'Código de establecimiento SUNAT',
          required: false,
          value: sede.codigo_establecimiento_sunat ?? '',
        },
        { key: 'telefono', label: 'Teléfono', type: 'tel', required: false, value: sede.telefono ?? '' },
      ],
      commit: (v) =>
        this.store.editarSede(sede, {
          codigo: texto(v['codigo']).toUpperCase(),
          nombre: texto(v['nombre']),
          direccion: texto(v['direccion']),
          ubigeo: opcional(v['ubigeo']),
          codigo_establecimiento_sunat: opcional(v['codigo_establecimiento_sunat']),
          telefono: opcional(v['telefono']),
        }),
    });
  }

  baja(sede: Sucursal): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja: ' + sede.nombre,
      description:
        'La sede deja de operar; sus datos y su historial se conservan y podrás restaurarla.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'motivo', label: 'Motivo', type: 'textarea' }],
      commit: (v) => this.store.bajaSede(sede, texto(v['motivo']), key),
    });
  }

  restaurar(sede: Sucursal): void {
    void this.store.restaurarSede(sede, newIdempotencyKey());
  }

  private opcionales(v: Record<string, string>) {
    const out: { ubigeo?: string; codigo_establecimiento_sunat?: string; telefono?: string } = {};
    const ubigeo = opcional(v['ubigeo']);
    const sunat = opcional(v['codigo_establecimiento_sunat']);
    const telefono = opcional(v['telefono']);
    if (ubigeo) out.ubigeo = ubigeo;
    if (sunat) out.codigo_establecimiento_sunat = sunat;
    if (telefono) out.telefono = telefono;
    return out;
  }
}
```

- [ ] **Step 4: Verificar** — `npx ng test --watch=false --include "src/app/features/configuracion/*-section.spec.ts"` → PASS. Ajustar los selectores de las pruebas solo si la plantilla real los cambia, sin debilitar lo que afirman.

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add src/app/features/configuracion
git commit -m "I2 paso 4: secciones Tu restaurante y Sedes

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Sección «Zonas y mesas»

**Files:**
- Create: `src/app/features/configuracion/salon-section.ts`, `salon-section.spec.ts`

**Interfaces:**
- Consumes: `ConfiguracionStore` (Tasks 4–5), `Dialogs`, `newIdempotencyKey`, `texto`, `opcional`, `entero`.
- Produces: `SalonSection` (selector `app-salon-section`).

- [ ] **Step 1: Pruebas que fallan** — `salon-section.spec.ts`:

```ts
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs } from '../../shared/ui/form-dialog';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { SalonSection } from './salon-section';
import type { Mesa, Zona } from './configuracion.models';

const audit = { creado_en: '', creado_por: '1', actualizado_en: '', actualizado_por: '1' };
const zona = (id: string, activa = true): Zona => ({
  id,
  sucursal_id: '1',
  nombre: 'Zona ' + id,
  orden_visualizacion: 0,
  es_activo: activa,
  ...audit,
});
const mesa = (id: string, over: Partial<Mesa> = {}): Mesa => ({
  id,
  sucursal_id: '1',
  zona_id: '10',
  numero_mesa: id,
  capacidad: 4,
  habilitada: true,
  posicion_x: null,
  posicion_y: null,
  es_activo: true,
  motivo_eliminacion: null,
  eliminado_en: null,
  eliminado_por: null,
  ...audit,
  ...over,
});

function setup(editar: boolean, sedes = [{ id: '1', nombre: 'Principal' }]) {
  const zonas = [zona('10'), zona('11', false)];
  const store = {
    errors: signal({}),
    busyId: signal<string | null>(null),
    puedeEditarSalon: signal(editar),
    sedesSalon: signal(sedes),
    sedeActual: signal<string | null>('1'),
    zonasSede: signal(zonas),
    zonasActivasSede: signal([zonas[0]]),
    mesasDeZona: (id: string) =>
      id === '10' ? [mesa('1'), mesa('2', { es_activo: false, motivo_eliminacion: 'Rota' })] : [],
    elegirSede: vi.fn(),
    habilitarMesa: vi.fn(),
    restaurarZona: vi.fn(),
    restaurarMesa: vi.fn(),
    load: vi.fn(),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [SalonSection],
    providers: [
      { provide: ConfiguracionStore, useValue: store },
      { provide: Dialogs, useValue: dialogs },
    ],
  });
  const fixture = TestBed.createComponent(SalonSection);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const boton = (texto: string) =>
    [...el.querySelectorAll('button')].find((b) => b.textContent?.trim() === texto);
  return { store, dialogs, el, boton };
}

describe('Sección Zonas y mesas', () => {
  it('agrupa las mesas bajo su zona', () => {
    const { el } = setup(true);
    expect(el.textContent).toContain('Zona 10');
    expect(el.textContent).toContain('Mesa 1');
    expect(el.textContent).toContain('Motivo de la baja: Rota');
  });

  it('con una sola sede no muestra el selector; con varias sí', () => {
    expect(setup(true).el.querySelector('select[aria-label="Sede"]')).toBeNull();
    const varias = setup(true, [
      { id: '1', nombre: 'Principal' },
      { id: '2', nombre: 'Norte' },
    ]);
    expect(varias.el.querySelector('select[aria-label="Sede"]')).not.toBeNull();
  });

  it('sin permiso de escritura no hay botones de acción ni interruptor habilitado', () => {
    const { el } = setup(false);
    expect([...el.querySelectorAll('button')].filter((b) => !b.classList.contains('toggle'))).toHaveLength(0);
    expect((el.querySelector('button.toggle') as HTMLButtonElement).disabled).toBe(true);
  });

  it('el interruptor de habilitación llama al store sin diálogo', () => {
    const { el, store, dialogs } = setup(true);
    (el.querySelector('button.toggle') as HTMLButtonElement).click();
    expect(store.habilitarMesa).toHaveBeenCalledWith(expect.objectContaining({ id: '1' }));
    expect(dialogs.form).not.toHaveBeenCalled();
  });

  it('dar de baja una mesa exige motivo; dar de baja una zona no tiene campos', () => {
    const { boton, dialogs } = setup(true);
    boton('Dar de baja mesa 1')?.click();
    expect(dialogs.form.mock.calls[0][0].fields.map((f: { key: string }) => f.key)).toEqual(['motivo']);
    boton('Dar de baja zona Zona 10')?.click();
    expect(dialogs.form.mock.calls[1][0].fields).toEqual([]);
  });

  it('nueva mesa ofrece solo zonas activas y los campos del spec', () => {
    const { boton, dialogs } = setup(true);
    boton('Nueva mesa')?.click();
    const campos = dialogs.form.mock.calls[0][0].fields;
    expect(campos.map((f: { key: string }) => f.key)).toEqual([
      'zona_id',
      'numero_mesa',
      'capacidad',
      'posicion_x',
      'posicion_y',
    ]);
    expect(campos[0].options).toEqual([{ value: '10', label: 'Zona 10' }]);
  });

  it('restaurar zona o mesa no abre diálogo', () => {
    const { boton, dialogs, store } = setup(true);
    boton('Restaurar zona Zona 11')?.click();
    boton('Restaurar mesa 2')?.click();
    expect(dialogs.form).not.toHaveBeenCalled();
    expect(store.restaurarZona).toHaveBeenCalled();
    expect(store.restaurarMesa).toHaveBeenCalled();
  });
});
```

(Los textos de los botones son únicos: cada uno lleva `aria-label` y el texto visible coincide con el que busca `boton()`; si se prefiere texto visible corto, buscar por `getAttribute('aria-label')` en el helper.)

- [ ] **Step 2: Verificar que fallan** — `npx ng test --watch=false --include src/app/features/configuracion/salon-section.spec.ts` → FAIL.

- [ ] **Step 3: Implementar** — `salon-section.ts`:

```ts
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs } from '../../shared/ui/form-dialog';
import { entero, opcional, texto } from './configuracion.forms';
import type { Mesa, MesaNueva, Zona } from './configuracion.models';
import { ConfiguracionStore } from './data-access/configuracion-store';

@Component({
  selector: 'app-salon-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel mt-6" aria-labelledby="salon-title">
      <div class="panel-header">
        <h2 id="salon-title" class="font-bold">Zonas y mesas</h2>
        @if (store.sedesSalon().length > 1) {
          <select aria-label="Sede" (change)="elegirSede($event)">
            @for (sede of store.sedesSalon(); track sede.id) {
              <option [value]="sede.id" [selected]="sede.id === store.sedeActual()">
                {{ sede.nombre }}
              </option>
            }
          </select>
        }
      </div>
      <div class="panel-body">
        @if (store.errors().salon; as error) {
          <div class="empty" role="alert">
            <h3>No pudimos cargar las zonas y mesas</h3>
            <p>{{ error }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        } @else {
          @if (store.puedeEditarSalon() && store.sedeActual()) {
            <div class="inline-actions mb-4">
              <button type="button" class="btn" (click)="nuevaZona()">Nueva zona</button>
              <button
                type="button"
                class="btn primary"
                [disabled]="store.zonasActivasSede().length === 0"
                (click)="nuevaMesa()"
              >
                Nueva mesa
              </button>
            </div>
          }
          @for (zona of store.zonasSede(); track zona.id) {
            <article class="setting-block" [class.inactive]="!zona.es_activo">
              <div>
                <h3>{{ zona.nombre }}</h3>
                <span class="pill" [class.green]="zona.es_activo">
                  {{ zona.es_activo ? 'Activa' : 'De baja' }}
                </span>
              </div>
              @if (store.puedeEditarSalon()) {
                <div class="inline-actions">
                  @if (zona.es_activo) {
                    <button type="button" class="btn" (click)="editarZona(zona)">Editar</button>
                    <button
                      type="button"
                      class="btn danger"
                      [attr.aria-label]="'Dar de baja zona ' + zona.nombre"
                      (click)="bajaZona(zona)"
                    >
                      Dar de baja
                    </button>
                  } @else {
                    <button
                      type="button"
                      class="btn"
                      [attr.aria-label]="'Restaurar zona ' + zona.nombre"
                      [disabled]="store.busyId() !== null"
                      (click)="restaurarZona(zona)"
                    >
                      Restaurar
                    </button>
                  }
                </div>
              }
            </article>
            @for (mesa of store.mesasDeZona(zona.id); track mesa.id) {
              <div class="setting-block ml-6" [class.inactive]="!mesa.es_activo">
                <div>
                  <h4>Mesa {{ mesa.numero_mesa }} · {{ mesa.capacidad }} personas</h4>
                  @if (mesa.posicion_x !== null && mesa.posicion_y !== null) {
                    <p class="subtle">Posición ({{ mesa.posicion_x }}, {{ mesa.posicion_y }})</p>
                  }
                  @if (!mesa.es_activo && mesa.motivo_eliminacion) {
                    <p class="subtle">Motivo de la baja: {{ mesa.motivo_eliminacion }}</p>
                  }
                </div>
                <div class="inline-actions">
                  <button
                    type="button"
                    class="toggle"
                    role="switch"
                    [class.on]="mesa.habilitada"
                    [attr.aria-checked]="mesa.habilitada"
                    [attr.aria-label]="'Habilitada: mesa ' + mesa.numero_mesa"
                    [disabled]="!store.puedeEditarSalon() || !mesa.es_activo || store.busyId() !== null"
                    (click)="store.habilitarMesa(mesa)"
                  ></button>
                  @if (store.puedeEditarSalon()) {
                    @if (mesa.es_activo) {
                      <button type="button" class="btn" (click)="editarMesa(mesa)">Editar</button>
                      <button
                        type="button"
                        class="btn danger"
                        [attr.aria-label]="'Dar de baja mesa ' + mesa.numero_mesa"
                        (click)="bajaMesa(mesa)"
                      >
                        Dar de baja
                      </button>
                    } @else {
                      <button
                        type="button"
                        class="btn"
                        [attr.aria-label]="'Restaurar mesa ' + mesa.numero_mesa"
                        [disabled]="store.busyId() !== null"
                        (click)="restaurarMesa(mesa)"
                      >
                        Restaurar
                      </button>
                    }
                  }
                </div>
              </div>
            }
          } @empty {
            <p class="empty" role="status">Esta sede todavía no tiene zonas.</p>
          }
        }
      </div>
    </section>
  `,
})
export class SalonSection {
  readonly store = inject(ConfiguracionStore);
  private readonly dialogs = inject(Dialogs);

  elegirSede(evento: Event): void {
    this.store.elegirSede((evento.target as HTMLSelectElement).value);
  }

  nuevaZona(): void {
    const sede = this.store.sedeActual();
    if (!sede) return;
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva zona',
      description: 'Una zona agrupa mesas del salón (por ejemplo, Terraza).',
      submit: 'Crear zona',
      fields: [
        { key: 'nombre', label: 'Nombre' },
        { key: 'orden_visualizacion', label: 'Orden', type: 'number', min: 0, max: 9999, required: false, value: 0 },
      ],
      commit: (v) =>
        this.store.crearZona(
          {
            sucursal_id: sede,
            nombre: texto(v['nombre']),
            orden_visualizacion: v['orden_visualizacion'] === '' ? 0 : entero(v['orden_visualizacion']),
          },
          key,
        ),
    });
  }

  editarZona(zona: Zona): void {
    this.dialogs.form({
      title: 'Editar zona: ' + zona.nombre,
      description: 'Lo que dejes igual no se envía.',
      submit: 'Guardar cambios',
      fields: [
        { key: 'nombre', label: 'Nombre', value: zona.nombre },
        {
          key: 'orden_visualizacion',
          label: 'Orden',
          type: 'number',
          min: 0,
          max: 9999,
          required: false,
          value: zona.orden_visualizacion,
        },
      ],
      commit: (v) =>
        this.store.editarZona(zona, {
          nombre: texto(v['nombre']),
          orden_visualizacion: v['orden_visualizacion'] === '' ? 0 : entero(v['orden_visualizacion']),
        }),
    });
  }

  bajaZona(zona: Zona): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja la zona ' + zona.nombre,
      description: 'No puede tener mesas activas. Podrás restaurarla después.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [],
      commit: () => this.store.bajaZona(zona, key),
    });
  }

  restaurarZona(zona: Zona): void {
    void this.store.restaurarZona(zona, newIdempotencyKey());
  }

  nuevaMesa(): void {
    const sede = this.store.sedeActual();
    const zonas = this.store.zonasActivasSede();
    if (!sede || zonas.length === 0) return;
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva mesa',
      description: 'El número debe ser único en la sede.',
      submit: 'Crear mesa',
      fields: [
        {
          key: 'zona_id',
          label: 'Zona',
          type: 'select',
          options: zonas.map((z) => ({ value: z.id, label: z.nombre })),
        },
        { key: 'numero_mesa', label: 'Número de mesa', hint: 'Hasta 20 caracteres.' },
        { key: 'capacidad', label: 'Capacidad', type: 'number', min: 1, max: 500, value: 4 },
        { key: 'posicion_x', label: 'Posición X', required: false, hint: 'Opcional. Ej.: 12.50' },
        { key: 'posicion_y', label: 'Posición Y', required: false, hint: 'Opcional. Ej.: 8.00' },
      ],
      commit: (v) => {
        const cuerpo: MesaNueva = {
          sucursal_id: sede,
          zona_id: v['zona_id'],
          numero_mesa: texto(v['numero_mesa']),
          capacidad: entero(v['capacidad']),
        };
        const x = opcional(v['posicion_x']);
        const y = opcional(v['posicion_y']);
        if (x) cuerpo.posicion_x = x;
        if (y) cuerpo.posicion_y = y;
        return this.store.crearMesa(cuerpo, key);
      },
    });
  }

  editarMesa(mesa: Mesa): void {
    const zonas = this.store.zonasActivasSede();
    this.dialogs.form({
      title: 'Editar mesa ' + mesa.numero_mesa,
      description: 'Lo que dejes igual no se envía. Deja la posición en blanco para quitarla.',
      submit: 'Guardar cambios',
      fields: [
        {
          key: 'zona_id',
          label: 'Zona',
          type: 'select',
          value: mesa.zona_id,
          options: zonas.map((z) => ({ value: z.id, label: z.nombre })),
        },
        { key: 'numero_mesa', label: 'Número de mesa', value: mesa.numero_mesa },
        { key: 'capacidad', label: 'Capacidad', type: 'number', min: 1, max: 500, value: mesa.capacidad },
        { key: 'posicion_x', label: 'Posición X', required: false, value: mesa.posicion_x ?? '' },
        { key: 'posicion_y', label: 'Posición Y', required: false, value: mesa.posicion_y ?? '' },
      ],
      commit: (v) =>
        this.store.editarMesa(mesa, {
          zona_id: v['zona_id'],
          numero_mesa: texto(v['numero_mesa']),
          capacidad: entero(v['capacidad']),
          posicion_x: opcional(v['posicion_x']),
          posicion_y: opcional(v['posicion_y']),
        }),
    });
  }

  bajaMesa(mesa: Mesa): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja la mesa ' + mesa.numero_mesa,
      description: 'No puede tener un pedido abierto. Podrás restaurarla después.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'motivo', label: 'Motivo', type: 'textarea' }],
      commit: (v) => this.store.bajaMesa(mesa, texto(v['motivo']), key),
    });
  }

  restaurarMesa(mesa: Mesa): void {
    void this.store.restaurarMesa(mesa, newIdempotencyKey());
  }
}
```

Nota: en la prueba, `boton('Dar de baja mesa 1')` busca por `textContent`; como el texto visible es «Dar de baja», el helper del spec debe buscar por `aria-label` (`b.getAttribute('aria-label') === texto || b.textContent?.trim() === texto`). Ajustar el helper del spec en este paso (no la plantilla).

- [ ] **Step 4: Verificar** — `npx ng test --watch=false --include src/app/features/configuracion/salon-section.spec.ts` → PASS.

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add src/app/features/configuracion
git commit -m "I2 paso 4: sección Zonas y mesas

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Pantalla real, contenedor movido y ruta

**Files:**
- Create: `src/app/features/configuracion/configuracion-real.ts`, `configuracion-real.spec.ts`, `configuracion-page.spec.ts`
- Move: `src/app/features/catalogos/configuracion-page.ts` → `src/app/features/configuracion/configuracion-page.ts`; `configuracion-page.html` ídem (`git mv`)
- Modify: el contenedor movido, `src/app/app.routes.ts`

**Interfaces:**
- Consumes: `ConfiguracionStore`, las tres secciones (Tasks 6–7), `PageHeader`, `AuthSession`.
- Produces: `ConfiguracionReal` (selector `app-configuracion-real`, `providers: [ConfiguracionStore]`), contenedor `ConfiguracionPage` que muestra la real con sesión y la demo sin ella.

- [ ] **Step 1: Pruebas que fallan**

`configuracion-real.spec.ts`:

```ts
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ConfiguracionReal } from './configuracion-real';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { NegocioSection } from './negocio-section';
import { SalonSection } from './salon-section';
import { SedesSection } from './sedes-section';

function setup(over: Record<string, unknown> = {}) {
  const store = {
    status: signal<'loading' | 'ready'>('ready'),
    notice: signal<{ kind: 'ok' | 'error'; text: string } | null>(null),
    sinAcceso: signal(false),
    puedeVerNegocio: signal(true),
    puedeVerSedes: signal(true),
    puedeVerSalon: signal(true),
    load: vi.fn(),
    dismissNotice: vi.fn(),
    ...over,
  };
  TestBed.configureTestingModule({ imports: [ConfiguracionReal] });
  // Las secciones se sustituyen por plantillas vacías: aquí solo se prueba qué secciones aparecen.
  for (const seccion of [NegocioSection, SedesSection, SalonSection]) {
    TestBed.overrideComponent(seccion, { set: { template: '', imports: [], providers: [] } });
  }
  TestBed.overrideComponent(ConfiguracionReal, {
    set: { providers: [{ provide: ConfiguracionStore, useValue: store }] },
  });
  const fixture = TestBed.createComponent(ConfiguracionReal);
  fixture.detectChanges();
  return { store, el: fixture.nativeElement as HTMLElement };
}

describe('Pantalla real de Configuración', () => {
  it('carga al iniciar y muestra las tres secciones con todos los permisos', () => {
    const { store, el } = setup();
    expect(store.load).toHaveBeenCalled();
    expect(el.querySelector('app-negocio-section')).not.toBeNull();
    expect(el.querySelector('app-sedes-section')).not.toBeNull();
    expect(el.querySelector('app-salon-section')).not.toBeNull();
  });

  it('oculta las secciones sin permiso de lectura', () => {
    const { el } = setup({ puedeVerNegocio: signal(false), puedeVerSedes: signal(false) });
    expect(el.querySelector('app-negocio-section')).toBeNull();
    expect(el.querySelector('app-sedes-section')).toBeNull();
    expect(el.querySelector('app-salon-section')).not.toBeNull();
  });

  it('sin ningún permiso muestra el aviso y ninguna sección', () => {
    const { el } = setup({ sinAcceso: signal(true) });
    expect(el.textContent).toContain('No tienes permiso');
    expect(el.querySelector('app-salon-section')).toBeNull();
  });

  it('muestra el aviso del store y permite cerrarlo', () => {
    const { el, store } = setup({ notice: signal({ kind: 'ok', text: 'Se creó la sede X.' }) });
    expect(el.textContent).toContain('Se creó la sede X.');
    (el.querySelector('.note-box button') as HTMLButtonElement).click();
    expect(store.dismissNotice).toHaveBeenCalled();
  });
});
```

`configuracion-page.spec.ts`:

```ts
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { AuthSession } from '../../core/auth/auth-session';
import { ConfiguracionPage } from './configuracion-page';
import { ConfiguracionReal } from './configuracion-real';

function setup(autenticado: boolean) {
  TestBed.configureTestingModule({
    imports: [ConfiguracionPage],
    providers: [{ provide: AuthSession, useValue: { isAuthenticated: signal(autenticado) } }],
  });
  TestBed.overrideComponent(ConfiguracionReal, {
    set: { template: '<p id="real"></p>', imports: [], providers: [] },
  });
  const fixture = TestBed.createComponent(ConfiguracionPage);
  fixture.detectChanges();
  return fixture.nativeElement as HTMLElement;
}

describe('Contenedor de Configuración', () => {
  it('con sesión real muestra la pantalla real y no la demostración', () => {
    const el = setup(true);
    expect(el.querySelector('#real')).not.toBeNull();
    expect(el.textContent).not.toContain('Restablecer demostración');
  });

  it('sin sesión conserva la pantalla de demostración', () => {
    const el = setup(false);
    expect(el.querySelector('#real')).toBeNull();
    expect(el.textContent).toContain('Restablecer demostración');
  });
});
```

- [ ] **Step 2: Verificar que fallan** — `npx ng test --watch=false --include "src/app/features/configuracion/configuracion-*.spec.ts"` → FAIL.

- [ ] **Step 3: Implementar**

(a) Mover:

```bash
mkdir -p src/app/features/configuracion
git mv src/app/features/catalogos/configuracion-page.ts src/app/features/configuracion/configuracion-page.ts
git mv src/app/features/catalogos/configuracion-page.html src/app/features/configuracion/configuracion-page.html
```

Corregir en el `.ts` movido las rutas relativas de sus imports (de `../../core/…`/`../../shared/…` según la ubicación vieja; `features/catalogos` y `features/configuracion` están al mismo nivel, así que los imports relativos a `core` y `shared` no cambian). Buscar con `npx tsc -p tsconfig.app.json --noEmit` cualquier import roto.

(b) `configuracion-real.ts`:

```ts
import { ChangeDetectionStrategy, Component, inject, OnInit } from '@angular/core';
import { PageHeader } from '../../shared/ui/page-header';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { NegocioSection } from './negocio-section';
import { SalonSection } from './salon-section';
import { SedesSection } from './sedes-section';

@Component({
  selector: 'app-configuracion-real',
  imports: [PageHeader, NegocioSection, SedesSection, SalonSection],
  providers: [ConfiguracionStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-page-header
      eyebrow="CONFIGURACIÓN"
      title="A tu manera"
      description="Los detalles que hacen que este restaurante sea tuyo."
    />
    @if (store.notice(); as notice) {
      <p
        class="note-box mb-4"
        [class.orange]="notice.kind === 'error'"
        [attr.role]="notice.kind === 'error' ? 'alert' : 'status'"
      >
        <span>{{ notice.text }}</span>
        <button type="button" class="btn" (click)="store.dismissNotice()">Entendido</button>
      </p>
    }
    @if (store.status() === 'loading') {
      <p class="empty" role="status">Cargando la configuración…</p>
    } @else if (store.sinAcceso()) {
      <p class="note-box orange" role="status">
        No tienes permiso para ver la configuración del restaurante. Pídele acceso a quien
        administra tu equipo.
      </p>
    } @else {
      @if (store.puedeVerNegocio()) {
        <app-negocio-section />
      }
      @if (store.puedeVerSedes()) {
        <app-sedes-section />
      }
      @if (store.puedeVerSalon()) {
        <app-salon-section />
      }
    }
  `,
})
export class ConfiguracionReal implements OnInit {
  readonly store = inject(ConfiguracionStore);

  ngOnInit(): void {
    void this.store.load();
  }
}
```

(c) Contenedor: en `configuracion-page.ts` añadir `import { AuthSession } from '../../core/auth/auth-session';`, `import { ConfiguracionReal } from './configuracion-real';`, `ConfiguracionReal` a `imports`, y el campo `readonly auth = inject(AuthSession);`. En `configuracion-page.html`: envolver **todo** el contenido actual así (primera línea y última línea del archivo):

```html
@if (auth.isAuthenticated()) {
  <app-configuracion-real />
} @else {
  <!-- … contenido demo existente, sin cambios … -->
}
```

(d) `app.routes.ts` (≈ línea 94): `import('./features/catalogos/configuracion-page')` → `import('./features/configuracion/configuracion-page')`. Mantener `title`.

- [ ] **Step 4: Verificar todo**

Run: `npm run test:run` → PASS completo (incluye `shell.spec.ts` con sus 14 enlaces).
Run: `npm run build` → éxito sin errores de tipos ni de presupuesto.

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add -A src/app
git commit -m "I2 paso 4: pantalla real de Configuración y contenedor real/demo

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 9: E2E de Playwright con API simulada

**Files:**
- Create: `e2e/configuracion-support.ts`, `e2e/configuracion.spec.ts`

**Interfaces:**
- Consumes: `e2e/support.ts` (`mockAuthApi`, `enterDemo`, `SESSION`, `TOKEN`) y el patrón de `e2e/team-support.ts` / `e2e/equipo.spec.ts` (login por formulario, ayudantes de diálogo, `calls`).
- Produces: `mockConfiguracionApi(page)` y las pruebas `configuracion.spec.ts`.

- [ ] **Step 1: Leer los apoyos existentes**

Leer `e2e/support.ts`, `e2e/team-support.ts` y `e2e/equipo.spec.ts` completos. Los nombres usados abajo (`mockAuthApi`, `ApiMock.session`, `SESSION`, forma de `Call`) son los que describe el relevamiento; si alguno difiere, ajustar **este código** a los nombres reales (no cambiar los apoyos existentes).

- [ ] **Step 2: Escribir el apoyo** — `e2e/configuracion-support.ts` (estado en memoria, registro de llamadas y fallos programables, igual que `team-support.ts`):

```ts
import type { Page, Route } from '@playwright/test';

export interface Llamada {
  method: string;
  path: string;
  body: unknown;
  idempotencyKey: string | undefined;
}

const AUDIT = {
  creado_en: '2026-10-21T10:00:00Z',
  creado_por: '1',
  actualizado_en: '2026-10-21T10:00:00Z',
  actualizado_por: '1',
};

export interface ConfiguracionMock {
  llamadas: Llamada[];
  /** Próximo rechazo programado para `METHOD /ruta` (una sola vez). */
  rechazos: Map<string, { status: number; code: string; detail: string }>;
}

const problem = (status: number, code: string, detail: string) => ({
  status,
  contentType: 'application/problem+json',
  body: JSON.stringify({ type: 'about:blank', title: 'Error', status, code, detail }),
});

export async function mockConfiguracionApi(page: Page): Promise<ConfiguracionMock> {
  const mock: ConfiguracionMock = { llamadas: [], rechazos: new Map() };
  let seq = 100;
  const negocio = {
    id: '1',
    singleton: true,
    ruc: '20123456789',
    razon_social: 'Mi Restaurante SAC',
    nombre_comercial: 'Mi Restaurante',
    direccion_fiscal: 'Av. Principal 123',
    ubigeo: null as string | null,
    telefono: null as string | null,
    email: null as string | null,
    moneda: 'PEN',
    zona_horaria: 'America/Lima',
    es_activo: true,
    ...AUDIT,
  };
  const sedes = [
    {
      id: '1',
      negocio_id: '1',
      codigo: 'PRINCIPAL',
      nombre: 'Sede Principal',
      direccion: 'Av. Principal 123',
      ubigeo: null,
      codigo_establecimiento_sunat: null,
      telefono: null,
      es_activo: true,
      motivo_eliminacion: null as string | null,
      eliminado_en: null,
      eliminado_por: null,
      ...AUDIT,
    },
  ];
  const zonas = [
    { id: '10', sucursal_id: '1', nombre: 'Salón', orden_visualizacion: 0, es_activo: true, ...AUDIT },
  ];
  const mesas = [
    {
      id: '100',
      sucursal_id: '1',
      zona_id: '10',
      numero_mesa: '1',
      capacidad: 4,
      habilitada: true,
      posicion_x: null,
      posicion_y: null,
      es_activo: true,
      motivo_eliminacion: null as string | null,
      eliminado_en: null,
      eliminado_por: null,
      ...AUDIT,
    },
  ];

  await page.route('**/api/v1/**', async (route: Route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace('/api/v1', '');
    if (path === '/sesion' || path.startsWith('/auth/')) return route.fallback();
    const method = request.method();
    const body = request.postData() ? JSON.parse(request.postData() as string) : undefined;
    mock.llamadas.push({
      method,
      path,
      body,
      idempotencyKey: request.headers()['idempotency-key'],
    });
    const clave = `${method} ${path}`;
    const rechazo = mock.rechazos.get(clave);
    if (rechazo) {
      mock.rechazos.delete(clave);
      return route.fulfill(problem(rechazo.status, rechazo.code, rechazo.detail));
    }
    const json = (data: unknown, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    const lista = (items: unknown[]) => json({ items, nextCursor: null });
    const id = path.split('/')[2];

    if (path === '/negocio' && method === 'GET') return json(negocio);
    if (path === '/negocio' && method === 'PATCH') return json(Object.assign(negocio, body));
    if (path === '/sucursales' && method === 'GET') return lista(sedes);
    if (path === '/sucursales' && method === 'POST') {
      const nueva = { ...sedes[0], id: String(++seq), es_activo: true, ...body };
      sedes.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/sucursales/') && path.endsWith('/baja')) {
      const s = sedes.find((x) => x.id === id)!;
      Object.assign(s, { es_activo: false, motivo_eliminacion: body.motivo });
      return json(s);
    }
    if (path.startsWith('/sucursales/') && path.endsWith('/restauracion')) {
      const s = sedes.find((x) => x.id === id)!;
      Object.assign(s, { es_activo: true, motivo_eliminacion: null });
      return json(s);
    }
    if (path === '/zonas' && method === 'GET') return lista(zonas);
    if (path === '/zonas' && method === 'POST') {
      const nueva = { id: String(++seq), es_activo: true, orden_visualizacion: 0, ...AUDIT, ...body };
      zonas.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/zonas/') && path.endsWith('/baja')) {
      const z = zonas.find((x) => x.id === id)!;
      z.es_activo = false;
      return json(z);
    }
    if (path.startsWith('/zonas/') && path.endsWith('/restauracion')) {
      const z = zonas.find((x) => x.id === id)!;
      z.es_activo = true;
      return json(z);
    }
    if (path === '/mesas' && method === 'GET') return lista(mesas);
    if (path === '/mesas' && method === 'POST') {
      const nueva = {
        id: String(++seq),
        habilitada: true,
        es_activo: true,
        posicion_x: null,
        posicion_y: null,
        motivo_eliminacion: null,
        eliminado_en: null,
        eliminado_por: null,
        ...AUDIT,
        ...body,
      };
      mesas.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/mesas/') && method === 'PATCH') {
      const m = mesas.find((x) => x.id === id)!;
      return json(Object.assign(m, body));
    }
    if (path.startsWith('/mesas/') && path.endsWith('/baja')) {
      const m = mesas.find((x) => x.id === id)!;
      Object.assign(m, { es_activo: false, motivo_eliminacion: body.motivo });
      return json(m);
    }
    if (path.startsWith('/mesas/') && path.endsWith('/restauracion')) {
      const m = mesas.find((x) => x.id === id)!;
      Object.assign(m, { es_activo: true, motivo_eliminacion: null });
      return json(m);
    }
    return route.fulfill(problem(404, 'no_encontrado', 'Ruta no simulada: ' + clave));
  });
  return mock;
}
```

- [ ] **Step 3: Escribir las pruebas** — `e2e/configuracion.spec.ts`:

```ts
import { expect, test, type Page } from '@playwright/test';
import { mockConfiguracionApi } from './configuracion-support';
import { enterDemo, mockAuthApi } from './support';

const TODOS = [
  'NEGOCIO_VER',
  'NEGOCIO_ADMINISTRAR',
  'SUCURSAL_VER',
  'SUCURSAL_ADMINISTRAR',
  'MESA_VER',
  'MESA_ADMINISTRAR',
];

async function abrir(page: Page, permisos: string[]) {
  const auth = await mockAuthApi(page);
  auth.session = { ...auth.session, permisos };
  const mock = await mockConfiguracionApi(page);
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill('admin');
  await page.getByLabel('Contraseña', { exact: true }).fill('UnaClaveSegura2026');
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/dashboard$/);
  await page.goto('/configuracion');
  return mock;
}

const dialogo = (page: Page) => page.getByRole('dialog');

test.describe('Configuración con la API simulada', () => {
  test('muestra negocio, sedes, zonas y mesas', async ({ page }) => {
    await abrir(page, TODOS);
    await expect(page.getByRole('heading', { name: 'Tu restaurante' })).toBeVisible();
    await expect(page.locator('#negocio-ruc')).toHaveValue('20123456789');
    await expect(page.getByText('Sede Principal')).toBeVisible();
    await expect(page.getByText('Mesa 1 · 4 personas')).toBeVisible();
  });

  test('edita el negocio enviando solo lo que cambió', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await page.locator('#negocio-nombre_comercial').fill('La Casa');
    await page.getByRole('button', { name: 'Guardar cambios' }).click();
    await expect(page.getByText('Guardamos los datos de tu restaurante.')).toBeVisible();
    const patch = mock.llamadas.find((c) => c.method === 'PATCH' && c.path === '/negocio');
    expect(patch?.body).toEqual({ nombre_comercial: 'La Casa' });
  });

  test('un rechazo de campo del negocio aparece bajo el campo y conserva lo escrito', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('PATCH /negocio', {
      status: 422,
      code: 'moneda_invalida',
      detail: 'La moneda debe tener 3 letras mayúsculas.',
    });
    await page.locator('#negocio-moneda').fill('xx');
    await page.getByRole('button', { name: 'Guardar cambios' }).click();
    await expect(page.getByText('La moneda debe tener 3 letras mayúsculas.')).toBeVisible();
    await expect(page.locator('#negocio-moneda')).toHaveValue('xx');
  });

  test('crea una sede, avisa que hay que asignarla y no duplica con el mismo formulario', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    await page.getByRole('button', { name: 'Nueva sede' }).click();
    await dialogo(page).getByLabel('Código').fill('norte');
    await dialogo(page).getByLabel('Nombre').fill('Sede Norte');
    await dialogo(page).getByLabel('Dirección').fill('Calle Norte 1');
    await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
    await expect(page.getByText(/Asígnala a tu equipo en Equipo y accesos/)).toBeVisible();
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/sucursales');
    expect(alta?.idempotencyKey).toMatch(/^[0-9a-f-]{36}$/);
    expect(alta?.body).toMatchObject({ codigo: 'NORTE', nombre: 'Sede Norte' });
  });

  test('código de sede duplicado se muestra en el campo y el diálogo sigue abierto', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('POST /sucursales', {
      status: 409,
      code: 'codigo_duplicado',
      detail: 'Ya existe una sede con ese código.',
    });
    await page.getByRole('button', { name: 'Nueva sede' }).click();
    await dialogo(page).getByLabel('Código').fill('PRINCIPAL');
    await dialogo(page).getByLabel('Nombre').fill('Otra');
    await dialogo(page).getByLabel('Dirección').fill('Calle 1');
    await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
    await expect(dialogo(page).getByText('Ya existe una sede con ese código.')).toBeVisible();
    await expect(dialogo(page).getByLabel('Nombre')).toHaveValue('Otra');
  });

  test('baja de sede pide motivo y la sede puede restaurarse', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await page.getByRole('button', { name: 'Dar de baja' }).first().click();
    await dialogo(page).getByLabel('Motivo').fill('Cierre temporal');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText('Motivo de la baja: Cierre temporal')).toBeVisible();
    expect(
      mock.llamadas.find((c) => c.path.endsWith('/baja') && c.path.startsWith('/sucursales'))?.body,
    ).toEqual({ motivo: 'Cierre temporal' });
    await page.getByRole('button', { name: 'Restaurar' }).first().click();
    await expect(page.getByText(/volvió a estar activa/)).toBeVisible();
  });

  test('crea una zona y una mesa en esa zona', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await page.getByRole('button', { name: 'Nueva zona' }).click();
    await dialogo(page).getByLabel('Nombre').fill('Terraza');
    await dialogo(page).getByRole('button', { name: 'Crear zona' }).click();
    await expect(page.getByRole('heading', { name: 'Terraza' })).toBeVisible();

    await page.getByRole('button', { name: 'Nueva mesa' }).click();
    await dialogo(page).getByLabel('Zona').selectOption({ label: 'Terraza' });
    await dialogo(page).getByLabel('Número de mesa').fill('T1');
    await dialogo(page).getByLabel('Capacidad').fill('6');
    await dialogo(page).getByRole('button', { name: 'Crear mesa' }).click();
    await expect(page.getByText('Mesa T1 · 6 personas')).toBeVisible();
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/mesas');
    expect(alta?.body).toMatchObject({ numero_mesa: 'T1', capacidad: 6, sucursal_id: '1' });
  });

  test('número de mesa duplicado aparece en el campo', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('POST /mesas', {
      status: 409,
      code: 'numero_mesa_duplicado',
      detail: 'Ya existe una mesa con ese número en la sucursal.',
    });
    await page.getByRole('button', { name: 'Nueva mesa' }).click();
    await dialogo(page).getByLabel('Número de mesa').fill('1');
    await dialogo(page).getByRole('button', { name: 'Crear mesa' }).click();
    await expect(
      dialogo(page).getByText('Ya existe una mesa con ese número en la sucursal.'),
    ).toBeVisible();
  });

  test('el interruptor habilita o deshabilita la mesa sin motivo', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await page.getByRole('switch', { name: 'Habilitada: mesa 1' }).click();
    await expect
      .poll(() => mock.llamadas.find((c) => c.method === 'PATCH' && c.path === '/mesas/100')?.body)
      .toEqual({ habilitada: false });
  });

  test('baja de mesa pide motivo; baja de zona con mesas muestra el 409 del servidor', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('POST /zonas/10/baja', {
      status: 409,
      code: 'zona_con_mesas',
      detail: 'La zona tiene mesas activas.',
    });
    await page.getByRole('button', { name: 'Dar de baja zona Salón' }).click();
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(dialogo(page).getByText('La zona tiene mesas activas.')).toBeVisible();
    await dialogo(page).getByRole('button', { name: 'Cancelar' }).click();

    await page.getByRole('button', { name: 'Dar de baja mesa 1' }).click();
    await dialogo(page).getByLabel('Motivo').fill('Mesa rota');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText('Motivo de la baja: Mesa rota')).toBeVisible();
  });

  test('un mesero (solo MESA_VER) ve el salón en solo lectura y no pide negocio ni sedes', async ({
    page,
  }) => {
    const mock = await abrir(page, ['MESA_VER']);
    await expect(page.getByText('Mesa 1 · 4 personas')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Tu restaurante' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Nueva mesa' })).toHaveCount(0);
    expect(mock.llamadas.some((c) => c.path === '/negocio' || c.path === '/sucursales')).toBe(false);
  });

  test('sin ningún permiso de configuración muestra el aviso', async ({ page }) => {
    const mock = await abrir(page, []);
    await expect(page.getByText(/No tienes permiso para ver la configuración/)).toBeVisible();
    expect(mock.llamadas).toHaveLength(0);
  });

  test('en modo demostración conserva la pantalla de ejemplo', async ({ page }) => {
    await enterDemo(page);
    await page.goto('/configuracion');
    await expect(page.getByRole('button', { name: 'Restablecer demostración' })).toBeVisible();
  });
});
```

- [ ] **Step 4: Ejecutar**

Run: `npx playwright test configuracion.spec --project=desktop` → PASS. Si un selector no coincide (por ejemplo el nombre accesible exacto de un botón), corregir **la prueba** salvo que revele un defecto real de la pantalla (en ese caso arreglar la pantalla y su prueba unitaria). Luego `npx playwright test configuracion.spec` (los tres proyectos: desktop, tablet, mobile) → PASS.

- [ ] **Step 5: Calidad y commit**

Run: `npm run format && npm run lint && npm run format:check`

```bash
git add e2e
git commit -m "I2 paso 4: E2E de Configuración con la API simulada

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 10: E2E contra la API real y documentación

**Files:**
- Create: `e2e/configuracion.real-api.spec.ts`
- Modify: `README.md`, `VERIFICACION.md` (ambos en `app-restaurant/`)

**Interfaces:**
- Consumes: el patrón de `e2e/equipo.real-api.spec.ts` (variables `E2E_API_USER` / `E2E_API_PASSWORD`, `test.skip` sin ellas, `mode: 'serial'`, datos únicos por corrida).

- [ ] **Step 1: Leer** `e2e/equipo.real-api.spec.ts` y copiar de ahí el modo de iniciar sesión por formulario y el uso del `request` de Playwright.

- [ ] **Step 2: Escribir la prueba** — `e2e/configuracion.real-api.spec.ts`:

```ts
import { expect, test, type Page } from '@playwright/test';

const admin = process.env['E2E_API_USER'];
const adminPass = process.env['E2E_API_PASSWORD'];

test.skip(!admin || !adminPass, 'Defina E2E_API_USER y E2E_API_PASSWORD para correr contra el API real.');
test.describe.configure({ mode: 'serial' });

const unico = String(Date.now()).slice(-8);
const codigo = `E2E${unico}`;
const sede = `Sede E2E ${unico}`;
const zona = `Zona E2E ${unico}`;
const numero = `E${unico.slice(-4)}`;

async function entrar(page: Page) {
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill(admin!);
  await page.getByLabel('Contraseña', { exact: true }).fill(adminPass!);
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/dashboard$/);
  await page.goto('/configuracion');
}

const dialogo = (page: Page) => page.getByRole('dialog');

test('crea una sede, una zona y una mesa, da de baja la mesa con motivo y la restaura', async ({
  page,
}) => {
  await entrar(page);
  await expect(page.getByRole('heading', { name: 'Tu restaurante' })).toBeVisible();

  // Sede: el administrador crea la sede pero no la ve en «Zonas y mesas» hasta que se la asignen.
  await page.getByRole('button', { name: 'Nueva sede' }).click();
  await dialogo(page).getByLabel('Código').fill(codigo);
  await dialogo(page).getByLabel('Nombre').fill(sede);
  await dialogo(page).getByLabel('Dirección').fill('Av. de la Prueba 1');
  await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
  await expect(page.getByText(/Asígnala a tu equipo en Equipo y accesos/)).toBeVisible();
  await expect(page.getByText(sede)).toBeVisible();

  // El mismo código otra vez: error en el campo, el diálogo sigue abierto.
  await page.getByRole('button', { name: 'Nueva sede' }).click();
  await dialogo(page).getByLabel('Código').fill(codigo);
  await dialogo(page).getByLabel('Nombre').fill('Repetida');
  await dialogo(page).getByLabel('Dirección').fill('Calle 1');
  await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
  await expect(dialogo(page)).toBeVisible();
  await expect(dialogo(page).getByLabel('Nombre')).toHaveValue('Repetida');
  await dialogo(page).getByRole('button', { name: 'Cancelar' }).click();

  // Zona y mesa en una sede donde el administrador sí opera (la primera del selector).
  await page.getByRole('button', { name: 'Nueva zona' }).click();
  await dialogo(page).getByLabel('Nombre').fill(zona);
  await dialogo(page).getByRole('button', { name: 'Crear zona' }).click();
  await expect(page.getByRole('heading', { name: zona })).toBeVisible();

  await page.getByRole('button', { name: 'Nueva mesa' }).click();
  await dialogo(page).getByLabel('Zona').selectOption({ label: zona });
  await dialogo(page).getByLabel('Número de mesa').fill(numero);
  await dialogo(page).getByLabel('Capacidad').fill('4');
  await dialogo(page).getByRole('button', { name: 'Crear mesa' }).click();
  await expect(page.getByText(`Mesa ${numero} · 4 personas`)).toBeVisible();

  // Baja con motivo y restauración (conserva el ID: la mesa vuelve a aparecer activa).
  await page.getByRole('button', { name: `Dar de baja mesa ${numero}` }).click();
  await dialogo(page).getByLabel('Motivo').fill('Prueba E2E');
  await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(page.getByText('Motivo de la baja: Prueba E2E')).toBeVisible();
  await page.getByRole('button', { name: `Restaurar mesa ${numero}` }).click();
  await expect(page.getByText(`La mesa ${numero} volvió a estar activa.`)).toBeVisible();
});
```

Nota de entorno: si el administrador de la prueba solo opera en sedes preexistentes, la zona y la mesa se crean en la sede que muestre el selector por defecto; la sede nueva se queda sin asignar (como avisa el aviso). Los datos de la corrida son únicos (`Date.now()`), no se limpian (solo existen bajas lógicas) y no chocan entre corridas.

- [ ] **Step 3: Ejecutar contra el backend real (opt-in)**

Prerrequisitos: backend en marcha (`cd service-restaurant && docker compose up -d --build`, instalación inicial con `cmd/admin` según su README) y el front con proxy (`API_TARGET=http://localhost:8080 npm start` o `API_PORT=18080`).

Run: `E2E_API_USER=admin E2E_API_PASSWORD=<clave> npx playwright test configuracion.real-api --project=desktop` → PASS. Sin las variables: `npx playwright test configuracion.real-api` → la prueba queda **omitida**. Si no hay backend disponible, dejar constancia en el reporte (no marcar como ejecutada).

- [ ] **Step 4: Documentación**

(a) `README.md` (de `app-restaurant/`): donde el «Alcance actual» dice qué pantallas hablan con la API (Equipo y accesos), añadir Configuración (negocio, sedes, zonas y mesas) con sus permisos `NEGOCIO_*`, `SUCURSAL_*`, `MESA_*`; indicar que el Salón de pedidos, el plano y el resumen siguen en demostración hasta I3; mencionar `features/configuracion/` y el patrón (contenedor real/demo + store por página). (b) `VERIFICACION.md`: añadir a «Recorridos verificados» los de Configuración (negocio editable y RUC fijo, alta/baja/restauración de sede, zonas, mesas, interruptor, solo lectura con `MESA_VER`, sin permisos, demo) y actualizar los conteos de pruebas unitarias y E2E con los números **reales** de `npm run test:run` y `npx playwright test` (leer la salida; no estimar).

- [ ] **Step 5: Verificación final y commit**

Run (todo desde `app-restaurant/`): `npm run lint && npm run format:check && npm run test:run && npm run build && npx playwright test` → todo en verde (las pruebas `real-api` se omiten sin variables).

```bash
git add e2e README.md VERIFICACION.md
git commit -m "I2 paso 4: E2E contra el API real y documentación del frontend

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

## Autorrevisión del plan frente al spec

- §7 `features/configuracion/` con `data-access/configuracion-api.ts`, `configuracion.models.ts`, `data-access/configuracion-store.ts`, `configuracion-real.ts` → Tasks 3, 4–5, 8 (más secciones en 6–7).
- Store `@Injectable()` provisto en la página, con `status`, `notice`, `busyId` y carga con paginación acotada (`collect`, 20 páginas) → Tasks 2, 4. `loadError` del spec se materializa como `errors()` **por sección** (el spec pide que un 403 en una sección no rompa la pantalla).
- `configuracion-page.ts` como contenedor real/demo → Task 8; «Restablecer demostración» solo en demo → se conserva el HTML demo intacto.
- Permisos por sección (sección ausente sin permiso; solo lectura sin escritura; RUC no editable) → Tasks 4, 6, 8.
- Sedes: alta, edición, baja con motivo, restauración y el aviso de asignación → Tasks 5–6 y E2E.
- Zonas y mesas agrupadas por sede con selector si hay varias; 409 de zona con mesas visible; mesa: número, zona, capacidad, posición opcional, interruptor sin motivo, baja con motivo y restauración → Tasks 5, 7, 9.
- Comportamiento común RF-036: `Dialogs.form`, `newIdempotencyKey()` por formulario abierto, errores de campo en el campo, `describeProblem()`, 403 sin pantalla rota, diálogo que no cierra hasta salir bien → Tasks 1, 4–7.
- §8 pruebas: `configuracion-store.spec.ts` (Tasks 4–5), E2E en modo demo y mocked (`configuracion.spec.ts`, Task 9) y `configuracion.real-api.spec.ts` (Task 10); CI del front en verde (`lint`, `format:check`, `test:run`, `build`, `test:e2e` en cada tarea).
- Huecos del relevamiento resueltos: errores de campo (Task 1), sesión sin permisos por sede (permiso «en alguna sede» + servidor manda), nombres de sede sin `SUCURSAL_VER` (fallback «Sede {id}»), códigos de duplicado por campo (`FIELD_BY_CODE`).
- Consistencia de nombres entre tareas: `ActionError`, `FieldProblem {field, message}`, `ConfiguracionStore` y sus métodos (`crearSede`/`editarSede`/`bajaSede`/`restaurarSede`, `crearZona`/`editarZona`/`bajaZona`/`restaurarZona`, `crearMesa`/`editarMesa`/`habilitarMesa`/`bajaMesa`/`restaurarMesa`), `sedesSalon`/`sedeActual`/`zonasSede`/`zonasActivasSede`/`mesasDeZona`, selectores `app-negocio-section`/`app-sedes-section`/`app-salon-section`/`app-configuracion-real`.
