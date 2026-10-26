# Diseño I2 (rebanada 3, backend) — Carta: categorías, productos, estaciones, impuestos y tasas

Requisitos: RES-RF-005 (productos por sucursal), RES-RF-027 (tasas vigentes), con RES-RF-030 (bajas y restauraciones) y RES-RF-036 (errores) en lo que toca a estas entidades. Código: `service-restaurant/` (módulo `catalogo`). El frontend de esta rebanada queda fuera (paso posterior, como en la rebanada 2).

## 1. Alcance

Seis tablas, todas con migración desde `00002`; no se crean tablas nuevas.

| Tabla | Superficie REST | Alcance | Baja |
|---|---|---|---|
| `categorias` | CRUD `/categorias` | global | desactivación simple (sin motivo) |
| `impuestos` | solo `GET /impuestos` y `GET /impuestos/{id}` (lectura) | global | no aplica |
| `tasas_impuesto` | `GET/POST /tasas-impuesto`, `GET/PATCH /tasas-impuesto/{id}` | global | no tiene `es_activo` |
| `estaciones_preparacion` | CRUD `/estaciones` | por sede | desactivación simple |
| `productos` | CRUD `/productos` | global | **baja con motivo** (décimo maestro) |
| `productos_sucursal` | `PUT /productos/{id}/disponibilidad` | por sede | `disponible` |

Fuera: `unidades_medida`, `articulos_inventario` y recetas (I5). Un producto `REVENTA` exige `articulo_id`, y no hay API de artículos hasta I5: el alta con `REVENTA` responde `referencia_invalida` por la FK; los otros tres tipos funcionan.

## 2. Decisiones

**D1. Permisos nuevos (migración 00010).**

| Permiso | ADMIN | Otros roles |
|---|---|---|
| `CARTA_VER` | sí | MESERO, CAJERO, COCINA, ALMACEN |
| `CARTA_ADMINISTRAR` | sí | — |
| `CARTA_DISPONIBILIDAD` | sí | COCINA |
| `ESTACION_VER` | sí | MESERO, COCINA |
| `ESTACION_ADMINISTRAR` | sí | — |
| `IMPUESTO_VER` | sí | — |
| `IMPUESTO_ADMINISTRAR` | sí | — |

`CARTA_VER` cubre categorías y productos. `CARTA_ADMINISTRAR` implica ver (igual que `MESA_ADMINISTRAR` en atención). `IMPUESTO_VER` también lo exigen las lecturas de `impuestos` que usa el formulario de productos: quien administra la carta necesita ver los impuestos, así que `CARTA_ADMINISTRAR` también habilita `GET /impuestos`.

**D2. Alcance por sede.** Categorías, productos, impuestos y tasas son maestros globales: se autoriza con `CanAnywhere` (patrón de clientes). Estaciones y disponibilidad son por sede: se autoriza con `Can(permiso, sede)` y el patrón `AutorizarSede` de `atencion` (sede que no ve → 404; la ve sin administrarla → 403 `sede_no_autorizada`). El listado de estaciones filtra por las sedes visibles.

**D3. Bajas.** Categorías y estaciones: `POST /{id}/baja` sin motivo y `POST /{id}/restauracion` (contrato). Productos: baja con `{motivo}` obligatorio (RF-030, décimo maestro). Todas con `Idempotency-Key`. Una baja repetida o una restauración de algo ya activo responde 409 (`ya_dada_de_baja`, `ya_activa`). Editar algo dado de baja responde 409 (`<entidad>_inactiva`).

**D4. Integridad al dar de baja** (los triggers solo validan padres activos al escribir hijos):
- Categoría con productos activos → 409 `categoria_con_productos_activos`.
- Estación con filas de `productos_sucursal` activas que la usan → 409 `estacion_en_uso`.
- Las tasas no se dan de baja.
Se comprueba dentro de la misma transacción, con la fila del padre bloqueada (`FOR UPDATE`) para que una carrera con un alta de hijo no deje huérfanos.

**D5. Tasas.** `tasa` en 0–100 (`numeric(7,4)`), `vigente_desde` obligatoria, `vigente_hasta` opcional y mayor que la anterior. La exclusión `gist` impide solapes: el error `23P01` se traduce a 409 `vigencia_solapada`. No hay datos por defecto (regla del esquema). `impuesto_id` no se edita (se crea otra tasa). El caso de uso comprueba el impuesto (activo, con `FOR SHARE`) antes de insertar: inexistente o inactivo → 422 `referencia_invalida`. `tasa` admite 1–3 dígitos enteros (con ceros a la izquierda, p. ej. `007.5`) y hasta 4 decimales; mayor que 100 → 422 `tasa_invalida`. `impuesto_id` llega en cadena y se valida en el caso de uso tras el permiso: ausente → 422 `impuesto_id_invalido`, mal formado (`+5`, `007`, `abc`) → 400 `impuesto_id_invalido`; igual con el filtro `impuesto_id` del listado (siempre 403 primero para quien no tiene permiso).

**D6. Productos.**
- Tipos: `PREPARADO`, `REVENTA`, `ADICIONAL`, `SERVICIO`. `REVENTA` ⇔ `articulo_id` (CHECK; se valida antes en el caso de uso con 422 `articulo_requerido`/`articulo_no_permitido`).
- `afectacion_igv` ∈ {`10`,`20`,`30`}; con `10` el `impuesto_id` es obligatorio (422 `impuesto_requerido`).
- Precio: decimal con hasta 2 decimales, ≥ 0, máximo 99 999 999 999.99; `factor_consumo` > 0 con hasta 6 decimales. Los decimales viajan como cadena (contrato); se validan sin `float`.
- `codigo` único (409 `codigo_duplicado`, aunque el producto esté dado de baja); `nombre` ≤ 180; `imagen_url` opcional ≤ 2000 caracteres, esquema `http`/`https`.
- Categoría e impuesto deben estar activos (lo garantiza el trigger `a1_referencias_activas`; se traduce a 422 con mensaje curado). Al restaurar un producto cuya categoría o impuesto sigue inactivo: 409 `referencia_inactiva`.
- El PATCH acepta cualquier campo del alta; `descripcion`, `articulo_id`, `impuesto_id` e `imagen_url` admiten `null` para vaciar.
- Listado: filtros `q` (contiene en el nombre o inicio del código; más de 150 caracteres se truncan), `categoria_id`, `es_activo` (`true` por defecto, `false` o la extensión `todos` sin filtro de estado), `limit`, `cursor`, y `sucursal_id` opcional que añade el estado en esa sede (D7).

**D7. Disponibilidad por sede** (`PUT /productos/{id}/disponibilidad`).
- Sin fila en `productos_sucursal` el producto **no se ofrece** en esa sede (así lo define `v_carta_disponible`). El `PUT` hace *upsert* por `(producto_id, sucursal_id)` y siempre devuelve la fila.
- Cuerpo: `sucursal_id` y `disponible` (contrato) más, **como extensión del contrato**, `precio_venta` y `estacion_id` opcionales, ambos anulables (ausente = sin cambio; `null` = usar precio general / sin estación). La estación debe ser activa y de la misma sede (FK compuesta y trigger `a1_referencias_activas`): una estación de otra sede, inexistente o inactiva responde el MISMO 422 `referencia_invalida` con el mismo mensaje, para no revelar a un admin de una sede que existe una estación en otra (revisión final; antes era `estacion_de_otra_sede`).
- Permisos: cambiar solo `disponible` pide `CARTA_DISPONIBILIDAD` **o** `CARTA_ADMINISTRAR` en esa sede; cambiar `precio_venta` o `estacion_id` pide `CARTA_ADMINISTRAR` en esa sede.
- Producto o sede inactivos → 409 (`producto_inactivo`) / 404 (sede). Pausar no altera pedidos ya registrados (aceptación de RF-005; cuando existan los pedidos, I3).
- `GET /productos?sucursal_id=` añade a cada ítem `en_sucursal`: `{disponible, precio_sucursal, precio_efectivo, estacion_id}` (`precio_sucursal` anulable; `precio_efectivo` es el de la sede o, si no hay, el general del producto) o `null` si no hay fila activa en esa sede. Con `disponible=true|false` el listado filtra por filas activas habilitadas / pausadas (400 `disponible_requiere_sucursal` sin `sucursal_id`); los productos sin fila no salen con `false`. Sin `sucursal_id` el ítem no trae el campo.

**D8. Listados.** Cursor por id ascendente, `limit` 1–100 (por defecto 25, como el resto de la API; el contrato dice 30 y se corrige en el contrato), `es_activo` opcional. Respuesta `{items, nextCursor}` (`nextCursor` es `null` en la última página). Sin `es_activo` (`true`/`false`) los listados no filtran por estado, salvo productos, que por defecto solo trae activos.

**D9. Idempotencia y errores.** Igual que clientes: altas, bajas y restauraciones exigen `Idempotency-Key`; los PATCH y el `PUT` no. Cuerpos con campos desconocidos → 400. Errores en `application/problem+json` con códigos estables.

## 3. Contrato y documentación a actualizar
- `docs/planificacion/03_api/openapi.yaml`: `x-implementado: true` y `x-permiso` reales en las operaciones de esta rebanada; añadir `GET /impuestos` y `GET /impuestos/{id}`; extender `Disponibilidad` (D7) y `Productos` (campo `en_sucursal`); `limit` por defecto 25.
- `requisitos_y_trazabilidad.json`: RF-005 y RF-027 → implementados en backend; y de paso RF-004 (rebanada 2, ya en código).
- `service-restaurant/docs/module-map.md` y `ADD.md`.

## 4. Riesgos
- Pausar un plato no es lo mismo que darlo de baja: la baja del maestro lo retira de todas las sedes. Está en el contrato y se mantiene.
- Una tasa editada después de usada en ventas (I3+) cambiaría el cálculo hacia atrás: hoy no hay ventas; antes de I3 hay que impedir editar tasas con uso (anotado para I3).
- Carrera baja-de-categoría vs alta-de-producto (y baja de estación vs asignación): sin prueba automatizada ni de carga. Queda cubierta por el diseño de bloqueos: el caso de uso toma `FOR UPDATE` del padre y el trigger `a1_referencias_activas` toma `FOR SHARE` de las referencias al insertar el hijo, de modo que ambas operaciones se serializan; se verificó a mano en la revisión final.
