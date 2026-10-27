# Diseño I2 (rebanada 3, frontend) — Carta, estaciones y tasas

Requisitos: RES-RF-005 (productos por sucursal), RES-RF-027 (tasas vigentes), con RES-RF-030 (bajas) y RES-RF-036 (errores y formularios). Consume el backend de la rebanada 3 (`docs/superpowers/specs/2026-10-25-i2-rebanada3-carta-design.md`, PR #5). Código: `app-restaurant/` (Angular 21, zoneless, signals, Vitest, Playwright). Patrón a imitar: Configuración (paso 4) y directorio (rebanada 2): contenedor `*-page` que elige entre vista real (sesión) y demo, `*-real` con store propio, `data-access/{api,store}`, secciones con diálogos `Dialogs.form`.

## 1. Alcance y ubicación (guía de pantallas)
- **`/carta` («Carta y recetas»)**, con sesión real: pestañas **Productos** y **Categorías**. Sin sesión se conserva la demo tal cual (`RestaurantStore`). Las **recetas** no tienen backend (I5): la parte de receta queda solo en la demo, y la vista real no la muestra.
- **`/configuracion`**: dos secciones nuevas, **Estaciones de preparación** (por sede) e **Impuestos y tasas** (la guía ubica tasas y estaciones en Configuración).
- Fuera: recetas, artículos de inventario (`REVENTA`), unidades de medida, imágenes subidas (solo `imagen_url`), cocina real (I3).

## 2. Decisiones
**F1. Permisos en la UI** (mismos del backend; es comodidad, el servidor revalida):

| Capacidad | Permisos |
|---|---|
| Ver carta (productos y categorías) | `CARTA_VER` \| `CARTA_ADMINISTRAR` \| `CARTA_DISPONIBILIDAD` |
| Administrar carta (alta, edición, baja, restauración, precio y estación por sede) | `CARTA_ADMINISTRAR` |
| Pausar/habilitar en una sede | `CARTA_DISPONIBILIDAD` \| `CARTA_ADMINISTRAR` |
| Ver estaciones | `ESTACION_VER` \| `ESTACION_ADMINISTRAR` |
| Administrar estaciones | `ESTACION_ADMINISTRAR` |
| Ver impuestos y tasas | `IMPUESTO_VER` \| `IMPUESTO_ADMINISTRAR` \| `CARTA_ADMINISTRAR` |
| Administrar tasas | `IMPUESTO_ADMINISTRAR` |

Cada sección se oculta o muestra «sin acceso» según lo que el actor puede ver (patrón `puedeVer*`/`sinAcceso` del store de Configuración). Los botones de escritura no se muestran sin permiso.

**F2. Productos.**
- Listado en tabla (no solo tarjetas): código, nombre, categoría, tipo, precio, afectación, estado y, con sede elegida, **Disponibilidad** (Disponible / Pausado / No ofrecido) y precio efectivo.
- Filtros enviados al servidor: búsqueda `q`, `categoria_id`, estado (Activos por defecto / Dados de baja / Todos → `es_activo=true|false|todos`) y sede (`sucursal_id`, más filtro `disponible` solo con sede). Paginación por cursor con «Cargar más» (`nextCursor`), sin recorrer todo el catálogo.
- Selector de sede: sedes de la sesión (`AuthSession`), sede inicial = la primera; «Todas las sedes» = sin `sucursal_id`.
- Alta/edición en diálogo: categoría (solo activas), código, nombre, descripción, tipo, precio, factor de consumo (solo si el tipo lo usa; por defecto 1), afectación IGV, impuesto (obligatorio si afectación = 10; solo impuestos activos), URL de imagen. **`REVENTA` no se ofrece** en el selector (necesita artículo de inventario, I5); si un producto existente es `REVENTA` se muestra y su tipo no se edita.
- Edición: PATCH solo con campos cambiados (`cambiosDe`); null para vaciar descripción, impuesto e imagen.
- Baja con motivo obligatorio (explica el efecto: sale de todas las sedes) y restauración con el mismo id; las bajas no se ocultan (filtro de estado).
- **Disponibilidad por sede:** acción «Pausar»/«Habilitar» sobre la sede seleccionada (`PUT`, sin Idempotency-Key). Con `CARTA_ADMINISTRAR`, diálogo «Ajustes de sede» para precio de sede (vacío = precio general) y estación (de esa sede). Sin sede elegida las acciones por sede se deshabilitan con explicación.
- Decimales siempre como cadena (sin `Number`/float para dinero); presentación `S/ 25.00` derivada por manipulación de texto. Máscaras: precio ≥ 0 con hasta 2 decimales; el servidor es la fuente de verdad.

**F3. Categorías.** Tabla: nombre, descripción, orden, estado. Alta/edición en diálogo (nombre, descripción, orden), baja sin motivo (con confirmación; el 409 `categoria_con_productos_activos` se muestra con su `detail`), restauración. Filtro «Incluir bajas». Se ordena en cliente por `orden_visualizacion`, `nombre`.

**F4. Estaciones** (sección de Configuración). Por sede (selector de la sede de salón ya existente en Configuración o el de sesión; reutiliza el mecanismo de sede actual de `salon-section`). Alta/edición (código y nombre; el código no se edita), baja sin motivo, restauración; 409 `estacion_en_uso` visible.

**F5. Impuestos y tasas** (sección de Configuración). Lista de impuestos (solo lectura) y, por impuesto, sus tasas con vigencia (`desde`, `hasta` o «vigente»); alta de tasa (impuesto, tasa 0–100, `vigente_desde`, `vigente_hasta` opcional, fundamento) y edición de tasa/vigencias/fundamento (no del impuesto). Resaltar la tasa vigente hoy. `vigencia_solapada` y demás rechazos junto al campo. Texto de ayuda: no hay tasa por defecto; sin tasa vigente no se puede vender con IGV (regla de RF-027).

**F6. Errores y formularios** (RF-036). Mismo mecanismo que Configuración: `FieldProblem` por código a campo (`CAMPO_POR_CODIGO`), conservar lo escrito, foco al error, `aria-label` por fila, estados carga/vacío/error con «Reintentar», aviso `note-box` con `role`. Idempotency-Key generada al abrir el diálogo (una por intento de usuario) en altas, bajas y restauraciones; no en PATCH ni PUT.

**F7. Demo intacta.** `CartaPage` conserva la demo cuando no hay sesión; no se tocan `RestaurantStore` ni los datos del prototipo.

## 3. Pruebas y entrega
- Unitarias (Vitest): stores (permisos, carga, escritura, errores, cursores), secciones/vistas con store simulado, utilidades de decimales/formato.
- E2E simulado (Playwright): `e2e/carta-support.ts` (API en memoria con `llamadas[]` y `rechazos`), `e2e/carta.spec.ts` (desktop/tablet/mobile), cobertura de configuración de estaciones/tasas en `configuracion.spec.ts` o spec nuevo.
- E2E contra API real: `e2e/carta.real-api.spec.ts` (opt-in con `E2E_API_USER/PASSWORD`). Se intenta ejecutar contra el backend local; si no es posible, se declara no ejecutado.
- `README.md` y `VERIFICACION.md` del frontend actualizados; trazabilidad (`cobertura_prototipo`/estado frontend de RF-005 y RF-027) solo con lo realmente verificado.

## 4. Riesgos
- `GET /productos` por defecto lista solo activos; la UI debe pedir `es_activo` explícito para no confundir «no hay» con «están de baja».
- El selector de sede depende de `AuthSession.user().branches`; un usuario con permisos solo en algunas sedes verá 404 al pedir otra (se muestra «sede no disponible»).
- `disponible=false` solo lista pausados con fila; los «No ofrecidos» se ven con `en_sucursal: null` (aviso en el filtro).
