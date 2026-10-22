# I2 · Paso 3 — Contrato, ADR 005 y expediente — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dejar documentado lo que el backend de I2 ya implementa (pasos 1 y 2): contrato OpenAPI alineado con la API real, ADR 005 «Baja de mesa con motivo», documentos del servicio actualizados y expediente (`docs/planificacion`) coherente, con `SHA256SUMS.txt` regenerado.

**Architecture:** Solo documentación y un archivo vacío que se elimina; no hay código Go ni migraciones. El contrato vive en `docs/planificacion/03_api/openapi.yaml` (generado por volcado: usa anclas `&idNNN`/`*idNNN`, así que cada edición se valida parseando el YAML). El expediente es un paquete con checksums que se mantiene a mano.

**Tech Stack:** YAML (OpenAPI 3.1.1), Markdown, JSON, HTML de una sola línea, `sha256sum`, Node (`npx`) para validar YAML/JSON (no hay Python en el entorno).

**Spec:** `docs/superpowers/specs/2026-10-18-i2-configuracion-salon-design.md` (§2 «Consecuencia documental», §4, §6, §9 paso 3, §10). **Requisito previo:** pasos 1 y 2 integrados en `main` (lo están: PR #1).

## Global Constraints

- Todas las rutas son relativas a la raíz del repo. `PLAN` = `docs/planificacion`. `SVC` = `service-restaurant`.
- Idioma: español, igual que los documentos vecinos. Commits en español con el trailer `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.
- Decisiones del usuario (no reabrir): (1) alinear **todo el slice** del YAML con lo implementado; (2) cambiar «nueve maestros» por «diez» de forma **uniforme** en las 17 restauraciones del YAML; (3) `diccionario_datos.html` gana las 3 columnas de `mesas` con la nota «migración 00008» **sin tocar** los totales (49 tablas / 664 columnas describen el SQL V4); (4) se crea `PLAN/CAMBIOS_1_3.txt` y `MANIFIESTO.json` pasa a versión documental 1.3; `CAMBIOS_1_2.txt` no se edita.
- Valores de contrato que el servidor implementa (fuente de verdad, ya verificados): `limit` 1–100 con **defecto 25**; `cursor` opaco; filtros `sucursal_id` y `es_activo` (sin `q`); IDs como cadenas; errores `application/problem+json` con `type`, `title`, `status`, `detail`, `code`, `requestId`; los `PATCH` rechazan campos desconocidos con 400 `json_invalido` (en particular `sucursal_id` en zonas/mesas, `negocio_id` en sucursales y `ruc` en negocio); `POST` de creación, baja y restauración exigen `Idempotency-Key`; restauración y baja de zona aceptan cuerpo ausente o `{}`; baja de mesa y de sucursal exigen `{"motivo": "..."}` (1–2000); baja de zona sin motivo.
- Permisos por operación: negocio `GET` → `NEGOCIO_VER` o `NEGOCIO_ADMINISTRAR`, `PATCH` → `NEGOCIO_ADMINISTRAR`; sucursales lectura → `SUCURSAL_VER` o `SUCURSAL_ADMINISTRAR`, escritura → `SUCURSAL_ADMINISTRAR`; zonas y mesas lectura → `MESA_VER` o `MESA_ADMINISTRAR`, escritura → `MESA_ADMINISTRAR`.
- Conteos a mantener coherentes: `mesas` pasa a `MAESTRO_PRINCIPAL` → **10** maestros con baja y motivo, **16** catálogos de desactivación simple (antes 9 y 17).
- No se edita `PLAN/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql` ni `catalogo_estructural.json` (describen V4).
- Ningún workflow de CI verifica `PLAN/`; sí corre `service-restaurant-ci.yml` por tocar `SVC/**`. Antes de cada commit del Task 1 no hace falta `go test`; basta con que `git diff --stat` solo muestre documentos.
- Validadores (ejecutar desde la raíz): YAML → `npx --yes js-yaml PLAN/03_api/openapi.yaml > /dev/null`; JSON → `node -e "JSON.parse(require('fs').readFileSync(process.argv[1],'utf8'))" <archivo>`; checksums → `cd docs/planificacion && sha256sum -c SHA256SUMS.txt`. Si `npx` no puede descargar `js-yaml`, usar `npx --yes @redocly/cli lint PLAN/03_api/openapi.yaml`.

## Estructura de archivos

| Archivo | Responsabilidad |
|---|---|
| `SVC/docs/adr/005-baja-de-mesa-con-motivo.md` (nuevo) | Decisión, alcance, consecuencias y verificación |
| `SVC/docs/module-map.md`, `SVC/README.md`, `SVC/docs/ADD.md` | Estado de módulos, endpoints y referencias |
| `SVC/api/openapi.yaml` (se elimina) | Archivo vacío desde el primer commit |
| `PLAN/03_api/openapi.yaml`, `PLAN/03_api/LEEME.txt` | Contrato |
| `PLAN/01_documentacion/{requisitos_y_trazabilidad,backlog_incrementos,registro_adr}.json`, `…{requisitos_y_trazabilidad,backlog_incrementos}.html`, `decisiones_arquitectonicas.html` | Trazabilidad, avance de I2, ADR-03 |
| `PLAN/02_base_datos/{politica_campos_por_tabla.json,matriz_auditoria.html,diccionario_datos.html}` | `mesas` como décimo maestro |
| `PLAN/06_calidad/{plan_pruebas.json,plan_pruebas.html,validacion_entrega.json}` | RES-CP-030 y situación del backend |
| `PLAN/{00_LEEME.txt,MANIFIESTO.json,CAMBIOS_1_3.txt,SHA256SUMS.txt}` | Metadatos y checksums |

---

### Task 1: ADR 005 y documentos del servicio

**Files:**
- Create: `service-restaurant/docs/adr/005-baja-de-mesa-con-motivo.md`
- Modify: `service-restaurant/docs/module-map.md`, `service-restaurant/README.md`, `service-restaurant/docs/ADD.md`
- Delete: `service-restaurant/api/openapi.yaml`

**Interfaces:**
- Consumes: formato de `adr/001…004` (título `# ADR-00N — …`, línea `Estado: …`, secciones `## Contexto`, `## Decisión`, `## Consecuencias`, `## Verificación`).
- Produces: el ADR-005 del repo, que Task 3 cita por nombre de archivo.

- [ ] **Step 1: Leer el modelo de ADR y los tres documentos**

Leer `service-restaurant/docs/adr/004-acceso-jwt-sesion-y-tablas-tecnicas.md` (formato), `module-map.md` (líneas 3 y la tabla de módulos), `README.md` (línea 5 y la sección API, línea ~52) y `docs/ADD.md` (línea 3, §8 «Desviaciones y huecos del contrato OpenAPI 0.2.0», §9 «Pendiente», línea ~81 «contrato HTTP contra `api/openapi.yaml`»).

- [ ] **Step 2: Crear el ADR 005**

Crear `service-restaurant/docs/adr/005-baja-de-mesa-con-motivo.md` con este contenido:

````markdown
# ADR-005 — Baja de mesa con motivo

Estado: aceptado para este repositorio (2026-10-21). Aporta al ADR-03 del expediente (auditoría contextual y baja lógica selectiva). No debe confundirse con el ADR-05 del expediente («Integraciones externas recuperables»): la numeración del repositorio es independiente.

## Contexto

El expediente define `mesas` como catálogo de desactivación simple (`ACTIVIDAD_SIMPLE`): la ruta `POST /mesas/{id}/baja` «no recibe motivo» y RF-030 distingue 9 maestros con baja y motivo de 17 catálogos de desactivación simple. En la rebanada 1 de I2 (configuración y salón) el equipo decidió que dar de baja una mesa debe dejar constancia de quién, cuándo y por qué, igual que una sucursal. Una mesa retirada del salón es una decisión operativa que conviene explicar, y la restauración debe conservar el mismo ID.

## Decisión

- `mesas` pasa a ser el décimo maestro con baja y motivo. La migración `00008_configuracion_salon` agrega `eliminado_en`, `eliminado_por` y `motivo_eliminacion` con el mismo `CHECK` que `sucursales` (activa: los tres nulos; inactiva: los tres presentes y el motivo no vacío) y recrea el trigger `a0_metadatos` con `fn_metadatos_fila('true','true')`, que ya exige el motivo, fija actor y fecha de baja y los limpia al restaurar. `zz_auditoria` registra el evento completo.
- `POST /api/v1/mesas/{id}/baja` exige `{"motivo": "<1–2000 caracteres>"}`; sin motivo responde 422 `motivo_requerido`. La restauración conserva el ID y limpia los marcadores.
- La baja de zona sigue siendo simple y sin motivo; se rechaza con 409 `zona_con_mesas` si la zona tiene mesas activas.
- Una mesa dada de baja no se edita ni se habilita hasta restaurarla (409 `mesa_inactiva`). El trigger `a2_baja_operativa` (V4) impide dar de baja una mesa con pedido abierto.
- Las mesas existentes que estuvieran inactivas al migrar reciben el motivo `Baja anterior a la migración 00008`.

## Consecuencias

- Se corrigen los documentos del expediente que decían «No recibe motivo» o «9 maestros / 17 catálogos»: contrato (`openapi.yaml`), RF-030, ADR-03, `politica_campos_por_tabla.json`, `matriz_auditoria.html`, `diccionario_datos.html` y RES-CP-030. El SQL V4 y sus totales (49 tablas, 664 columnas) no se modifican: describen la base inicial y la diferencia vive en la migración 00008.
- Si el equipo revierte la decisión, basta omitir las columnas y el trigger de la migración y devolver la baja de mesa al contrato sin cuerpo.
- La interfaz (paso 4) pide el motivo al dar de baja una mesa, con el mismo diálogo que sucursales.

## Verificación

- Pruebas de integración de `tests/integration/atencion_mesas_test.go`: baja sin motivo → 422, motivo y actor en la fila y en `auditoria_eventos`, restauración con el mismo ID y reintento idempotente sin nuevo evento.
- `sha256sum -c SHA256SUMS.txt` en `docs/planificacion` tras actualizar el expediente.
````

- [ ] **Step 3: Actualizar `module-map.md`**

(a) En la línea 3, donde dice «Módulos hoy: **`identidad`** (implementado en I1 …) y `catalogo`, esqueleto de ejemplo…», añadir **`atencion`** (zonas y mesas, implementado en I2). (b) En la tabla de módulos: la fila Identidad pasa a indicar «…negocio y sucursales (administración en I2)…» y la fila Atención a «zonas y mesas (`00003`, I2) · reservas, pedidos, cuentas, detalles (I3; reservas: I6)». (c) En «Dirección de imports» añadir una frase: `tests/architecture` cubre también `atencion`, que no importa `identidad`.

- [ ] **Step 4: Actualizar `README.md`**

Línea 5: añadir `atencion` y «negocio y sucursales» al estado. En la sección API (línea ~52) añadir, tras la administración de accesos, las rutas `GET/PATCH /negocio`, `GET/POST /sucursales`, `GET/PATCH /sucursales/{id}`, `/zonas…` y `/mesas…` (con sus `…/baja` y `…/restauracion`), los seis permisos nuevos (`NEGOCIO_*`, `SUCURSAL_*`, `MESA_*`) y un enlace a `docs/adr/005-baja-de-mesa-con-motivo.md`.

- [ ] **Step 5: Actualizar `docs/ADD.md`**

(a) Línea 3: dejar de decir «sin módulos de negocio todavía» (hay `identidad` y `atencion`). (b) En §8, la frase que dice que `GET /usuarios` y `GET /sucursales-asignables` «deben incorporarse al OpenAPI» pasa a decir que se incorporaron en el paso 3 de I2. (c) En §9 «Pendiente», reemplazar «contrato HTTP contra `api/openapi.yaml`» por «contrato HTTP en `../docs/planificacion/03_api/openapi.yaml`»; quitar de la lista lo ya hecho de I2 rebanada 1 y dejar «rebanada 2 de I2 (carta, clientes, proveedores, tasas) y frontend». (d) Añadir una línea de índice de ADR (001–005) en §2 si no existe.

- [ ] **Step 6: Eliminar el archivo vacío y comprobar referencias**

Run: `git rm service-restaurant/api/openapi.yaml`

Run: `grep -rn "api/openapi.yaml" --include=*.md --include=*.yml --include=*.go . | grep -v node_modules | grep -v docs/superpowers`
Expected: solo `docs/estructura/PLANTILLA_BACKEND_GO.md` (plantilla genérica, se deja) y, en `SVC/docs`, ninguna que apunte a `service-restaurant/api/openapi.yaml`. Si el directorio `service-restaurant/api/` queda vacío, `git` no lo versiona; no hace falta nada más.

- [ ] **Step 7: Verificar y commitear**

Run: `git diff --stat` → solo `.md` y el archivo eliminado.

```bash
git add service-restaurant
git commit -m "I2 paso 3: ADR 005 (baja de mesa con motivo) y documentos del servicio

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Contrato OpenAPI — permisos, baja con motivo y alineación del slice

**Files:**
- Modify: `docs/planificacion/03_api/openapi.yaml`, `docs/planificacion/03_api/LEEME.txt`

**Interfaces:**
- Consumes: Global Constraints (valores de contrato y permisos); esquemas existentes `Motivo` (línea ~5610), `Confirmacion`, `Sucursales` (modelo de `eliminado_*`, líneas ~5768–5785), `IdempotencyKey`.
- Produces: contrato del slice alineado; Task 3 añade rutas nuevas sobre este archivo.

El YAML es un volcado con anclas (`&id048`, `*id001`…): al editar un esquema que usa anclas, no borrar un ancla que otro esquema referencie (buscar `*idNNN` antes de eliminar un `&idNNN`).

- [ ] **Step 1: Línea base de validación**

Run: `npx --yes js-yaml docs/planificacion/03_api/openapi.yaml > /dev/null && echo OK`
Expected: `OK`. Run: `grep -c "x-permiso: Permiso fino pendiente" docs/planificacion/03_api/openapi.yaml` y anotar el número (N0).

- [ ] **Step 2: `x-permiso` de las 20 operaciones del slice**

Para cada operación de la tabla, reemplazar su línea `x-permiso: Permiso fino pendiente de migración en el catálogo` (la que está dentro de ese bloque) por el valor indicado. Valores: formato `x-permiso: NEGOCIO_VER` para lectura; si la lectura admite dos permisos, seguir cómo el archivo expresa los de I1 en `/usuarios*` (`USUARIO_ADMINISTRAR`, línea ~3702); si el archivo no tiene convención para «uno u otro», escribir `x-permiso: NEGOCIO_VER | NEGOCIO_ADMINISTRAR` y documentar en la descripción «basta cualquiera de los dos».

| operationId | Ruta y método | x-permiso |
|---|---|---|
| operacion_003 | GET /negocio | NEGOCIO_VER \| NEGOCIO_ADMINISTRAR |
| operacion_004 | PATCH /negocio | NEGOCIO_ADMINISTRAR |
| operacion_087 / 089 | GET /sucursales, GET /sucursales/{id} | SUCURSAL_VER \| SUCURSAL_ADMINISTRAR |
| operacion_088 / 090 / 091 / 092 | POST /sucursales, PATCH, baja, restauración | SUCURSAL_ADMINISTRAR |
| operacion_041 / 043 | GET /zonas, GET /zonas/{id} | MESA_VER \| MESA_ADMINISTRAR |
| operacion_042 / 044 / 045 / 046 | POST /zonas, PATCH, baja, restauración | MESA_ADMINISTRAR |
| operacion_035 / 037 | GET /mesas, GET /mesas/{id} | MESA_VER \| MESA_ADMINISTRAR |
| operacion_036 / 038 / 039 / 040 | POST /mesas, PATCH, baja, restauración | MESA_ADMINISTRAR |

Run: `grep -c "x-permiso: Permiso fino pendiente" docs/planificacion/03_api/openapi.yaml` → debe ser `N0 − 20`.

- [ ] **Step 3: Baja de mesa con motivo (operacion_039)**

En `/mesas/{id}/baja` (líneas ~1400–1432): (a) `summary: Desactivar mesas` → `summary: Dar de baja mesas`; (b) en la descripción, reemplazar «Cambia es_activo a false. No recibe motivo ni guarda marcadores de eliminación; registra actor de actualización y evento.» por «Exige motivo (1–2000 caracteres); cambia es_activo a false y guarda eliminado_en, eliminado_por y motivo_eliminacion; registra el evento de auditoría. Se rechaza con 422 si la mesa tiene un pedido abierto.»; (c) añadir al final del bloque, igual que `/sucursales/{id}/baja` (líneas 3201–3238):

```yaml
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Motivo'
```

- [ ] **Step 4: «Nueve» → «diez» uniforme**

En las 17 descripciones de restauración, reemplazar «solo los nueve maestros principales limpian marcadores de baja» por «solo los diez maestros principales limpian marcadores de baja» (el texto está partido en dos líneas en algunos bloques: buscar `nueve maestros` y `marcadores de baja` por separado).

Run: `grep -c "nueve maestros" docs/planificacion/03_api/openapi.yaml` → `0`. Run: `grep -c "diez maestros" …` → `17`.

- [ ] **Step 5: Esquema `Mesas` y `Error`**

(a) En `components.schemas.Mesas` (líneas ~7261–7328) añadir tras `es_activo` las tres propiedades copiadas de `Sucursales` (`eliminado_en`, `eliminado_por`, `motivo_eliminacion`, cada una `anyOf` con `null` y `readOnly: true`), sin añadirlas a `required`.

(b) Reemplazar el esquema `Error` (línea ~5570) por la forma real `application/problem+json`:

```yaml
    Error:
      type: object
      description: Problema HTTP (application/problem+json). El campo con error se codifica en code (por ejemplo posicion_x_invalida, motivo_requerido).
      required:
      - type
      - title
      - status
      - code
      properties:
        type:
          type: string
        title:
          type: string
        status:
          type: integer
        detail:
          type: string
        code:
          type: string
        requestId:
          type: string
```

Buscar con `grep -n "fieldErrors\|traceId" docs/planificacion/03_api/openapi.yaml` cualquier otra mención y ajustar sus descripciones (no hay que retirar rutas).

- [ ] **Step 6: Esquemas de edición y altas**

Sin duplicar anclas ajenas: (a) `NegocioEdicion` (línea ~9473): quitar `ruc`. (b) `SucursalesAlta` (~9405): quitar `negocio_id` de `properties` y de `required`; `SucursalesEdicion`: quitar `negocio_id`. (c) `ZonasEdicion` y `MesasEdicion`: quitar `sucursal_id`. En los cuatro mantener `additionalProperties: false` y `minProperties: 1` donde ya existan. Si un esquema de edición se construye con `<<`/anclas compartidas con el alta, definir las propiedades de la edición explícitamente en lugar de modificar el ancla compartida.

- [ ] **Step 7: Parámetros de listado y restauración**

(a) `GET /mesas`, `/zonas`, `/sucursales`: `limit` con `default: 25` (máx. 100); declarar `es_activo` (boolean, sin `default`: sin parámetro se devuelven activos e inactivos) en los tres; quitar `q` de `/mesas`; quitar `sucursal_id` de `GET /sucursales` y de `GET /negocio` (la implementación no los lee). (b) Las tres restauraciones (`/sucursales`, `/zonas`, `/mesas`): `requestBody.required: false`. (c) `POST /zonas/{id}/baja`: sin `requestBody` (ya es así).

- [ ] **Step 8: `LEEME.txt` del contrato**

En `PLAN/03_api/LEEME.txt`: la línea «V4: los tres marcadores de baja solo existen en nueve maestros principales.» pasa a «…en diez maestros principales (mesas desde la migración 00008).» y «Los catálogos simples usan es_activo y sus rutas /baja no requieren body.» pasa a «Los catálogos simples usan es_activo y sus rutas /baja no requieren body; sucursales y mesas exigen {motivo}.».

- [ ] **Step 9: Validar y commitear**

Run: `npx --yes js-yaml docs/planificacion/03_api/openapi.yaml > /dev/null && echo OK` → `OK`.
Run: `grep -n "No recibe motivo" docs/planificacion/03_api/openapi.yaml | head -20` → ninguna línea del bloque `/mesas/{id}/baja`; las de otros catálogos simples permanecen.

```bash
git add docs/planificacion/03_api
git commit -m "I2 paso 3: contrato del slice alineado (permisos, baja de mesa con motivo, esquemas y errores)

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Contrato OpenAPI — `GET /usuarios` y `GET /sucursales-asignables`

**Files:**
- Modify: `docs/planificacion/03_api/openapi.yaml`

**Interfaces:**
- Consumes: estilo de `/usuarios` (`post`, operacion_105, ~línea 3692), parámetros `limit`/`cursor` de `/roles` (~3307–3317), bloque de errores `*id001…*id007`; implementación en `service-restaurant/internal/modules/identidad/web/controller/administracion.go` y sus DTO de respuesta (`web/dto/response/`), que son la fuente de los campos.
- Produces: rutas `GET /usuarios` (`operacion_154`) y `GET /sucursales-asignables` (`operacion_155`) y los esquemas `UsuarioEquipo`, `UsuariosLista`, `SucursalesAsignablesLista`.

- [ ] **Step 1: Leer la implementación**

Leer el handler y los DTO de respuesta de `GET /usuarios` y `GET /sucursales-asignables` en `identidad` (campos, nulabilidad, filtros y permisos exigidos, `limit` por defecto) y anotar los nombres JSON exactos. Los esquemas siguientes deben coincidir campo a campo; si el código difiere de esta guía (`UsuarioEquipo` = esquema `Usuarios` + `motivo_baja`, `roles[]`, `sucursales[]`), manda el código.

- [ ] **Step 2: Añadir `get` bajo `/usuarios`**

Dentro de `paths./usuarios`, antes del `post`, añadir un `get` con: `operationId: operacion_154`, `summary: Listar equipo`, `tags: [Accesos]`, la `description` estándar («Diseño propuesto. El backend valida permiso, sucursal, campos y estado en una transacción cuando corresponda.» sustituida por una descripción real: lista paginada del equipo visible para el actor, con roles y sedes), `x-implementado: true`, `x-permiso: USUARIO_ADMINISTRAR`, parámetros `limit` (default 25, máx. 100), `cursor`, `es_activo` y los filtros que la implementación lea, y respuestas `'200'` → `$ref: '#/components/schemas/UsuariosLista'` más `'400': *id001` … `'503': *id007`.

- [ ] **Step 3: Añadir `/sucursales-asignables`**

Nueva ruta con `get`: `operationId: operacion_155`, `summary: Listar sucursales asignables`, `tags: [Accesos]`, `x-implementado: true`, `x-permiso` según el código, respuesta `'200'` → `$ref: '#/components/schemas/SucursalesAsignablesLista'` (solo `items`, sin `nextCursor`; cada ítem con `id`, `codigo`, `nombre`) y el bloque de errores estándar.

- [ ] **Step 4: Esquemas**

Añadir en `components.schemas`, junto a `Usuarios`/`Sucursales`: `UsuarioEquipo` (los campos de `Usuarios` por `allOf` más `motivo_baja` nullable, `roles` array y `sucursales` array con los ítems reales), `UsuariosLista` (`items: [UsuarioEquipo]`, `nextCursor` nullable) y `SucursalesAsignablesLista` (`items` con `id` como cadena, `codigo`, `nombre`).

- [ ] **Step 5: `x-implementado: true` en el slice**

Marcar `x-implementado: true` en las operaciones realmente implementadas de I1 (`/auth/login`, `/sesion`, `/roles`, `/permisos`, `/usuarios*`) y del slice de I2 (las 20 del Task 2 más las 2 nuevas). Confirmar cada una contra `internal/modules/*/web/controller/routes.go`; no marcar ninguna que no esté registrada allí. Run: `grep -c "x-implementado: true" docs/planificacion/03_api/openapi.yaml` y comparar con el número de rutas registradas.

- [ ] **Step 6: Validar y commitear**

Run: `npx --yes js-yaml docs/planificacion/03_api/openapi.yaml > /dev/null && echo OK` → `OK`.
Run: `grep -c "operationId:" docs/planificacion/03_api/openapi.yaml` → `155`. Rutas: `grep -c "^  /" docs/planificacion/03_api/openapi.yaml` → `113` (solo `/sucursales-asignables` es ruta nueva; `/usuarios` ya existía).

```bash
git add docs/planificacion/03_api
git commit -m "I2 paso 3: contrato con GET /usuarios y GET /sucursales-asignables y operaciones implementadas marcadas

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Expediente (`docs/planificacion`) y checksums

**Files:**
- Modify: `PLAN/02_base_datos/politica_campos_por_tabla.json`, `PLAN/02_base_datos/matriz_auditoria.html`, `PLAN/02_base_datos/diccionario_datos.html`
- Modify: `PLAN/01_documentacion/requisitos_y_trazabilidad.json`, `…/requisitos_y_trazabilidad.html`, `…/backlog_incrementos.json`, `…/backlog_incrementos.html`, `…/registro_adr.json`, `…/decisiones_arquitectonicas.html`
- Modify: `PLAN/06_calidad/plan_pruebas.json`, `PLAN/06_calidad/plan_pruebas.html`, `PLAN/06_calidad/validacion_entrega.json`
- Modify: `PLAN/00_LEEME.txt`, `PLAN/MANIFIESTO.json`, `PLAN/SHA256SUMS.txt`
- Create: `PLAN/CAMBIOS_1_3.txt`

**Interfaces:**
- Consumes: Global Constraints (10 y 16; decisiones 3 y 4); ADR-005 del repo (Task 1); contrato (Tasks 2–3: 113 rutas y 155 operaciones).
- Produces: expediente coherente y `sha256sum -c` en verde.

Los `.html` son una sola línea: editar con `sed -i 's/…/…/'` o con `Edit` sobre cadenas literales únicas; después de cada edición comprobar con `grep -c`.

- [ ] **Step 1: `politica_campos_por_tabla.json`**

En la entrada `"tabla": "mesas"` (líneas ~266–275): `"tipo": "ACTIVIDAD_SIMPLE"` → `"tipo": "MAESTRO_PRINCIPAL"` y `"marcadores_eliminacion": false` → `true` (copiar la forma exacta de la entrada de `sucursales`, líneas 12–21). Validar: `node -e "const p=JSON.parse(require('fs').readFileSync('docs/planificacion/02_base_datos/politica_campos_por_tabla.json','utf8')); const a=Array.isArray(p)?p:(p.tablas||Object.values(p)[0]); const c={}; a.forEach(t=>c[t.tipo]=(c[t.tipo]||0)+1); console.log(c)"` → `MAESTRO_PRINCIPAL: 10`, `ACTIVIDAD_SIMPLE: 16`. (Ajustar la ruta al arreglo según la estructura real del archivo.)

- [ ] **Step 2: `matriz_auditoria.html`**

(a) Encabezado: «9 maestros principales» → «10 maestros principales» y «17 catálogos o relaciones» → «16 catálogos o relaciones». (b) Fila de `mesas`: sustituir `<td>Ninguno</td><td>No aplica / derivado</td><td>Desactivar/reactivar es_activo; sin motivo obligatorio ni marcadores</td>` por el contenido equivalente de la fila de `sucursales` (marcadores «Los 3» y «Baja con motivo, actor y fecha; restauración del mismo ID»). Comprobar: `grep -c "sin motivo obligatorio ni marcadores" docs/planificacion/02_base_datos/matriz_auditoria.html` baja en uno (queda para los demás simples).

- [ ] **Step 3: `diccionario_datos.html`**

En la sección `<h2>mesas</h2>` (línea 28) añadir al final de la tabla de columnas las filas `eliminado_en` (TIMESTAMPTZ, nulo), `eliminado_por` (BIGINT, nulo) y `motivo_eliminacion` (TEXT, nulo), copiando el marcado de las mismas filas en la sección `sucursales`, con la nota «Agregada en la migración 00008 (I2)» en la columna de descripción. No modificar los totales «49 tablas / 664 columnas».

- [ ] **Step 4: Trazabilidad (JSON y HTML)**

En `requisitos_y_trazabilidad.json`: (a) RES-RF-030: «Distinguir 9 maestros con baja y motivo de 17 catálogos o relaciones con desactivación simple.» → «…10 maestros… 16 catálogos…» y «Los 9 maestros registran motivo…Las otras 17 tablas…» → «Los 10 maestros…Las otras 16 tablas…». (b) RES-RF-003 y RES-RF-008: `"estado_backend": "IMPLEMENTADO"` y `situacion_backend` con el patrón de RF-001 («Implementado en I2 (rebanada 1: negocio y sucursales / zonas y mesas); la prueba de aceptación RES-CP-00N sigue NO_EJECUTADA hasta ejecutarse contra la API real.») — `resultado_prueba_backend` permanece `NO_EJECUTADA`. (c) RES-RF-036: sin cambio de estado (el frontend es el paso 4); añadir `situacion_backend` indicando «Parcial en I2: errores por campo en negocio, sucursales, zonas y mesas; pendiente la interfaz». En el `.html` replicar los mismos cambios: textos de RF-030 (2 ocurrencias), y para RF-003/008 la fila `<td>Situación</td>` y la fila extra «Situación backend» con el patrón de RF-001. Validar JSON con el comando de Global Constraints.

- [ ] **Step 5: Backlog, registro ADR y decisiones**

(a) `backlog_incrementos.json`: I2 `"estado": "PROPUESTO"` → `"EN_CURSO"` y añadir `avance` (patrón de I1): «Rebanada 1 hecha en backend (negocio, sucursales, zonas y mesas; contrato, ADR 005 y expediente); falta el frontend (paso 4) y las rebanadas con RF-004, RF-005 y RF-027». Replicar en `backlog_incrementos.html`. (b) `registro_adr.json`, ADR-03, campo `decision`: «nueve maestros principales con tres marcadores de eliminación y 17 catálogos o relaciones sin ellos» → «diez … y 16 …»; en `verificacion` o `revision` añadir «Baja de mesa con motivo: ver ADR-005 del repositorio (service-restaurant/docs/adr)». No cambiar `MANIFIESTO.registros_adr` (7). (c) `decisiones_arquitectonicas.html`: el mismo cambio de «nueve → diez / 17 → 16» en el texto del ADR-03.

- [ ] **Step 6: Plan de pruebas y validación de entrega**

(a) `plan_pruebas.json` RES-CP-030 (línea ~563–568): `accion` y `esperado` con 10 y 16; replicar en `plan_pruebas.html`. El resultado sigue `NO_EJECUTADA`. (b) `validacion_entrega.json` líneas ~36–50: «Backend: solo I1 está implementado (I2 a I9 pendientes)» → «Backend: I1 e I2 (rebanada 1: negocio, sucursales, zonas y mesas) están implementados (el resto pendiente)»; las frases sobre RES-CP aprobados no cambian.

- [ ] **Step 7: `00_LEEME.txt`, `MANIFIESTO.json` y `CAMBIOS_1_3.txt`**

(a) `00_LEEME.txt` línea ~40: «9 maestros con datos de eliminación, 17 catálogos/relaciones sin ellos.» → «10 … 16 …»; línea ~38: «112 rutas y 153 operaciones» → «113 rutas y 155 operaciones». (b) `MANIFIESTO.json`: `"tablas_con_marcadores_eliminacion": 9` → `10`; `rutas_api_propuestas: 112` → `113`; `operaciones_api_propuestas: 153` → `155`; `"version_documental": "1.2"` → `"1.3"`; dejar columnas/tablas totales como están. (c) Crear `PLAN/CAMBIOS_1_3.txt` con: título «Cambios de la versión 1.3 (I2, rebanada 1)», fecha 2026-10-21, y viñetas: mesas pasa a décimo maestro con baja y motivo (RF-030: 10 y 16; política de campos; matriz; diccionario con nota de la migración 00008); contrato alineado con la implementación (permisos finos, baja de mesa con `{motivo}`, esquemas de edición, errores problem+json, `GET /usuarios` y `GET /sucursales-asignables`, 113 rutas y 155 operaciones); RF-003 y RF-008 pasan a IMPLEMENTADO en backend; I2 EN_CURSO; ADR-003 actualizado y ADR-005 del repositorio; nota «El SQL V4 y sus totales no cambian».

- [ ] **Step 8: Regenerar `SHA256SUMS.txt` y verificar**

Regenerar solo las líneas de los archivos tocados, manteniendo el formato `<hash>  <ruta relativa>` y el orden; añadir la línea de `CAMBIOS_1_3.txt`. Desde `docs/planificacion`:

```bash
cd docs/planificacion
awk '{print $2}' SHA256SUMS.txt > ../../.superpowers/sdd/orden-sha.txt   # orden actual (ruta ignorada por git)
echo CAMBIOS_1_3.txt >> ../../.superpowers/sdd/orden-sha.txt
while read f; do sha256sum "$f"; done < ../../.superpowers/sdd/orden-sha.txt > SHA256SUMS.txt
```

Después de regenerar, comprobar con: `sha256sum -c SHA256SUMS.txt | grep -v ': OK$'` → sin salida; `wc -l SHA256SUMS.txt` → 63 líneas (62 + CAMBIOS_1_3.txt). Verificar con `git diff --stat -- SHA256SUMS.txt` que solo cambian las líneas de los archivos editados.

- [ ] **Step 9: Validación final y commit**

Run: validar con `node` cada `.json` editado y `npx --yes js-yaml` el YAML; `grep -rn "9 maestros\|nueve maestros\|17 catálogos" docs/planificacion --include=*.json --include=*.html --include=*.yaml --include=LEEME.txt --include=00_LEEME.txt` → sin coincidencias (`CAMBIOS_1_2.txt` queda como histórico).

```bash
git add docs/planificacion
git commit -m "I2 paso 3: expediente actualizado (mesas como décimo maestro, RF-003/008/030, I2 en curso) y checksums

Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>"
```

---

## Autorrevisión del plan frente al spec

- §6 `openapi.yaml`: `x-permiso` (Task 2 paso 2), baja de mesa con `{motivo}` (paso 3), `GET /usuarios` y `GET /sucursales-asignables` (Task 3).
- §6 archivo vacío `service-restaurant/api/openapi.yaml` (Task 1 paso 6).
- §6 ADR 005, `module-map.md`, README (Task 1).
- §6 expediente: RF-003/008 a IMPLEMENTADO (Task 4 paso 4), RF-030 y matriz de auditoría con el décimo maestro (pasos 2 y 4), política de campos y diccionario (pasos 1 y 3).
- §2 «Consecuencia documental»: RF-030, `politica_campos_por_tabla.json`, diccionario, `matriz_auditoria.html` y contrato — cubiertos.
- §10: la desviación del expediente queda registrada en el ADR-005.
- Decisiones del usuario: alinear todo el slice (Task 2 pasos 5–7), «diez» uniforme (paso 4), diccionario con nota 00008 (Task 4 paso 3), `CAMBIOS_1_3.txt` y versión 1.3 (paso 7).
- Conteos coherentes: 10/16 en YAML, JSON, HTML y LEEME; 113 rutas y 155 operaciones en contrato, `00_LEEME.txt` y `MANIFIESTO.json`.
