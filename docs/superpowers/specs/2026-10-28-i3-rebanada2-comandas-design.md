# Diseño I3 (rebanada 2, backend) — Comandas y estados de cocina

Requisito: RES-RF-012 (enviar comanda por estación), con RES-RF-002 (autorizar por permiso y sede). Código: `service-restaurant/` (módulo nuevo `produccion`, según ADR-01). El frontend de Cocina queda fuera (R4).

## 1. Alcance

R2 usa las tablas `comandas` y `comanda_detalles` de la migración `00003`, con sus triggers de auditoría y metadatos. No cambia el esquema. Solo añade la migración `00012` de permisos: `COCINA_OPERAR` existe desde `00001` pero no está asignado a ningún rol de cocina, así que `00012` se lo asigna al rol COCINA (y a ADMIN si no lo tiene). `PEDIDO_VER` ya lo recibió COCINA en `00011`. La migración se escribe con `ON CONFLICT DO NOTHING` y su `Down` solo retira lo que insertó.

Fuera de R2: cancelar líneas o comandas (R3), consumo de inventario (I5), pantalla de Cocina (R4), dividir cuentas (R3).

## 2. Superficie REST

| Endpoint | Función | Permiso |
|---|---|---|
| `POST /pedidos/{id}/comandas` | Envía a una estación las cantidades pendientes de una o varias líneas. `Idempotency-Key` obligatorio (se guarda en `clave_idempotencia`) | `PEDIDO_CREAR` |
| `GET /comandas` | Cola de preparación por sede; filtros `sucursal_id`, `estacion_id`, `estado`; paginación por cursor | `COCINA_OPERAR` |
| `POST /comanda-detalles/{id}/transiciones` | Mueve una línea de comanda al estado siguiente | `COCINA_OPERAR`; `LISTO→ENTREGADO` también con `PEDIDO_EDITAR` |

Se marcan `x-implementado: true` en el contrato. Errores según el contrato de I2 (RF-036): `422` validación, `409` conflicto de estado, `403`/`404` por sede.

## 3. Reglas

### 3.1 Envío (`POST /pedidos/{id}/comandas`)

1. Se bloquea el pedido `FOR UPDATE`, después las líneas por `id` ascendente (orden de R1).
2. El pedido debe estar abierto; cada línea debe estar `ACTIVO` y pertenecer a una cuenta abierta. Si no, 409.
3. Pendiente de una línea = `cantidad` menos la suma de `comanda_detalles.cantidad` de esa línea con estado distinto de `CANCELADO`. Si la cantidad solicitada lo supera: 409 `cantidad_excede_pendiente`.
4. Estación estricta: la estación del envío debe coincidir con `productos_sucursal.estacion_id` del producto en la sede del pedido. Si no coincide o el producto no tiene estación: 422 `estacion_no_corresponde`. La estación debe ser activa y de la misma sede.
5. `descripcion` e `instrucciones` (de `notas_preparacion`) se copian de la línea al enviar.
6. Una línea repetida en el mismo envío: 422 `linea_duplicada`.
7. Repetir la misma `Idempotency-Key` con el mismo cuerpo devuelve la comanda original; con otro cuerpo, 409 `idempotency_key_reutilizada`. Reimprimir no crea otro envío.
8. `enviado_por`, `creado_por` y `actualizado_por` son el usuario autenticado.

### 3.2 Transiciones (`POST /comanda-detalles/{id}/transiciones`)

Solo `PENDIENTE→PREPARACION→LISTO→ENTREGADO`, sin saltos ni retrocesos; cualquier otra: 409 `transicion_invalida`. Cada paso fija `iniciado_en`, `listo_en` o `entregado_en`. `CANCELADO` no se alcanza aquí (R3). La comanda no guarda estado de cabecera: se deriva de sus líneas.

Permisos: `PENDIENTE→PREPARACION` y `PREPARACION→LISTO` exigen `COCINA_OPERAR`. `LISTO→ENTREGADO` se permite con `COCINA_OPERAR` o `PEDIDO_EDITAR` (el mesero que lleva el plato), sin permiso nuevo. Sin ninguno de los dos: 403 `sin_permiso`. El acceso a la sede se valida siempre.

### 3.3 Consulta (`GET /comandas`)

Solo sedes visibles para el usuario. Cada elemento incluye comanda, estación, pedido, mesa si existe y sus líneas con estado y marcas de tiempo, ordenadas por `enviado_en`.

## 4. Arquitectura

Módulo `internal/modules/produccion` con las mismas capas que `atencion`:

- `domain/policy`: tabla de transiciones, cálculo del pendiente, regla de permiso de la transición. Reglas puras, sin I/O.
- `application`: casos de uso `EnviarComanda`, `ListarComandas`, `TransicionarComandaDetalle`, con DTO y puertos.
- `infrastructure/persistence`: SQL de `comandas` y `comanda_detalles`.
- `web`: controladores y DTO.

`produccion` no importa `atencion` ni `catalogo`: lee pedido, líneas y estación del producto con SQL en su propia capa de persistencia, igual que `atencion` hace con tablas de `catalogo`. `tests/architecture` debe seguir en verde.

Concurrencia: en la transición se bloquea la línea de comanda; si hace falta el pedido, se toma antes que la línea (orden de R1).

## 5. Pruebas

- Unitarias de política: matriz completa de transiciones, cálculo del pendiente, regla de permiso de la entrega.
- Integración con Postgres real, como en R1:
  - envío feliz y en tandas parciales;
  - exceso sobre el pendiente (409);
  - estación que no corresponde y producto sin estación (422);
  - línea anulada o cuenta cerrada (409);
  - repetición idempotente y reutilización de clave con otro cuerpo (409);
  - 403/404 por sede y cola solo con sedes visibles;
  - transición válida, salto y retroceso (409), entrega por mesero con `PEDIDO_EDITAR`;
  - concurrencia: dos envíos simultáneos de la misma línea no superan la cantidad.
- Arquitectura: `tests/architecture` en verde con el módulo nuevo.

## 6. Cierre del expediente

- OpenAPI: `x-implementado: true` en las tres operaciones y códigos de error nuevos anotados.
- `module-map.md`: Producción pasa a implementado en I3 R2.
- RF-012: `estado_backend` a `IMPLEMENTADO`, casos de aceptación `NO_EJECUTADA` hasta la ejecución formal.
- `avance` de I3 en `backlog_incrementos.json` (el estado sigue `EN_CURSO`).
- Matriz de bajas y formularios: las comandas no tienen baja lógica; se cancelan en R3.

## 7. Pendientes heredados

- D4 de R1 (descuento al bajar la cantidad sin `DESCUENTO_AUTORIZAR`): decidir antes de R3, no afecta a R2.
- Cancelar una línea que ya tiene comanda (R3) deberá pasar sus `comanda_detalles` a `CANCELADO` con motivo.

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
