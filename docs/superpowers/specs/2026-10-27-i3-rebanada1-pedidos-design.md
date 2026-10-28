# Diseño I3 (rebanada 1, backend) — Pedidos: apertura, cuenta y consumos

Requisitos: RES-RF-009 (abrir pedido y cuenta), RES-RF-010 (agregar consumos históricos) y, en lo que toca a pedidos, RES-RF-002 (autorizar por permiso y sede), con RES-RF-027 (tasas vigentes) como fuente de la tasa de cada línea. Código: `service-restaurant/` (módulo `atencion`). El frontend de esta rebanada queda fuera (R4).

## 1. Alcance de I3 y de esta rebanada

I3 ("Pedidos y cocina": RF-009, 010, 011, 012, 014) se reparte en cuatro rebanadas, cada una con su spec, plan e implementación:

| Rebanada | Contenido | Módulo |
|---|---|---|
| **R1** (este documento) | RF-009 abrir pedido y cuenta; RF-010 consumos; consultas. Hace verificable el 403/404 por sede de RF-002 | `atencion` |
| R2 | RF-012 comandas por estación y estados de cocina | `produccion` (nuevo, según ADR-01) |
| R3 | RF-011 dividir cuentas; RF-014 cancelar con motivo (pedido y línea) | `atencion` y `produccion` |
| R4 | Frontend de Salón y Cocina; cierre con auditoría | `app-restaurant` |

R1 no crea tablas: usa `pedidos`, `cuentas`, `detalles_pedido` y `pedidos_delivery` de la migración `00003`, con sus triggers (`fn_validar_pedido`, `fn_validar_cuenta`, `fn_guardar_detalle_pedido`, `fn_validar_delivery`). Solo añade la migración `00011` de permisos.

Fuera de R1: anular líneas y cancelar pedidos o cuentas (R3); enviar a cocina (R2); dividir cuentas (R3); pagos y cierre de pedido (I4); asignación de repartidor y estados de entrega (I6); reservas como origen del pedido (I6, `reserva_id` queda `NULL`); descuentos por promoción o cupón.

## 2. Superficie REST

| Endpoint | Función | Permiso |
|---|---|---|
| `POST /pedidos` | Crea el pedido y su cuenta nº 1 en una sola transacción. `Idempotency-Key` obligatorio | `PEDIDO_CREAR` |
| `GET /pedidos` | Lista con filtros `sucursal_id`, `estado`, `tipo`, `mesa_id` y paginación; solo sedes visibles | `PEDIDO_VER` |
| `GET /pedidos/{id}` | Pedido con cuentas, líneas activas y anuladas, datos de reparto y totales | `PEDIDO_VER` |
| `POST /pedidos/{id}/detalles` | Agrega una línea a una cuenta abierta. `Idempotency-Key` obligatorio | `PEDIDO_EDITAR` |
| `PATCH /pedidos/{id}/detalles/{did}` | Edita cantidad, notas de preparación o descuento | `PEDIDO_EDITAR` (`PEDIDO_DESCONTAR` si toca el descuento) |

Estados y códigos siguen el contrato de errores de I2 (RF-036): `422` validación con `errores[]` por campo, `409` conflicto de estado, `403`/`404` por sede. La API no expone bloqueo optimista (ningún módulo lo hace hoy): las escrituras concurrentes se serializan con `FOR UPDATE` (D7).

## 3. Decisiones

**D1. Apertura (`POST /pedidos`).**
Cuerpo: `sucursal_id`, `tipo` (`SALON`, `LLEVAR`, `DELIVERY`), `mesa_id` (solo `SALON`), `comensales` (≥ 1, por defecto 1), `cliente_id` (opcional), `notas` (opcional) y `reparto` (solo `DELIVERY`).

- `SALON`: exige `mesa_id` de esa sede, activa y habilitada, y `comensales ≤ capacidad`. Con mesa ya ocupada responde 409 `mesa_ocupada` (índice `uq_pedido_mesa_abierta`; se traduce el error de restricción, no se confía en una lectura previa).
- `LLEVAR`: sin mesa (`mesa_id` presente → 422).
- `DELIVERY`: sin mesa; `reparto` obligatorio con `destinatario`, `telefono` y `direccion`, y opcionales `referencia`, `ubigeo`, `latitud`, `longitud`, `programado_en`. Se inserta en `pedidos_delivery` en la misma transacción con `canal = 'PROPIO'` y `estado = 'PENDIENTE'`. El costo de envío **no** se calcula aquí: se cobra como una línea de un producto de tipo `SERVICIO`, agregada con `POST detalles`. La asignación de repartidor y las transiciones de entrega son de I6.
- `mesero_id`: el empleado vinculado al usuario que abre; `NULL` si el usuario no tiene empleado vinculado.
- `moneda`: la del negocio (por defecto `PEN`); inmutable después (trigger).
- Se crean en una sola transacción pedido (`REGISTRADO`), cuenta nº 1 (`ABIERTA`) y, si aplica, datos de reparto. Si cualquier parte falla no queda cabecera huérfana (aceptación de RF-009).
- `clave_idempotencia` del pedido sale de `Idempotency-Key`. Mismo header y mismo cuerpo → misma respuesta (200 con el pedido original); mismo header con cuerpo distinto → 409 `idempotencia_conflicto`.

**D2. Líneas como instantánea (RF-010).**
Cuerpo de `POST detalles`: `cuenta_id` (opcional; por defecto la cuenta nº 1), `producto_id`, `cantidad`, `notas_preparacion`, `descuento_total` (opcional, por defecto 0).

- La API copia del producto y de su tasa vigente a esa fecha: `descripcion`, `precio_unitario`, `afectacion_igv` y `tasa_impuesto`. Cambiar la carta después no modifica líneas existentes. `importe_total` y `base_imponible` son columnas generadas; la API solo los lee.
- Valida: `cantidad > 0` (hasta 3 decimales); producto activo (si no, 422 `producto_inactivo`); disponible en esa sede (`productos_sucursal.disponible`, si no, 422 `producto_no_disponible`); cuenta del pedido y en estado `ABIERTA`; pedido sin cerrar; `descuento_total ≤ round(cantidad × precio, 2)`.
- Tipos `ADICIONAL` y `detalle_padre_id`: se admite `detalle_padre_id` (línea del mismo pedido y cuenta) solo si el producto es `ADICIONAL`; un `ADICIONAL` sin padre → 422. Recetas (`receta_id`) quedan `NULL` hasta I5.
- Idempotencia: un reintento con el mismo `Idempotency-Key` no duplica la línea (se guarda la clave en la tabla técnica de idempotencia de `00007`, patrón de I2).

**D3. Edición de líneas.**
`PATCH` admite `cantidad`, `notas_preparacion` y `descuento_total`. Se rechaza con 409 si la cuenta ya no está abierta, el pedido está cerrado o la línea ya tiene detalle de comanda (`comanda_detalles`; hasta R2 no existe ninguna, pero la comprobación se implementa ya para no tener que reabrir esta regla). Anular una línea enviada es de R3.

**D4. Descuento.**
Aceptar `descuento_total` distinto de 0, en alta o edición, exige el permiso `PEDIDO_DESCONTAR`; sin él, 403 `permiso_insuficiente`.

**D5. Permisos (migración `00011`).**

| Permiso | ADMIN | Otros roles |
|---|---|---|
| `PEDIDO_VER` | sí | MESERO, CAJERO, COCINA |
| `PEDIDO_CREAR` | sí | MESERO, CAJERO |
| `PEDIDO_EDITAR` | sí | MESERO, CAJERO |
| `PEDIDO_DESCONTAR` | sí | CAJERO |

`PEDIDO_CREAR` y `PEDIDO_EDITAR` implican ver. Las lecturas de mesas y productos que necesita el formulario de apertura siguen cubiertas por `MESA_VER` y `CARTA_VER`.

**D6. Alcance por sede.**
Todo pedido pertenece a una sede. Se autoriza con `Can(permiso, sede)` y el patrón `AutorizarSede`: sede que el usuario no ve → 404; la ve pero sin el permiso del comando → 403 `sede_no_autorizada`. Para operaciones sobre `{id}` se carga primero el pedido por id y se autoriza contra su `sucursal_id`, de modo que conocer el ID de un pedido de otra sede nunca da acceso (aceptación de RF-002). `GET /pedidos` filtra por las sedes visibles.

**D7. Concurrencia y consistencia.**
Las escrituras sobre líneas bloquean primero el pedido (`FOR UPDATE`) y luego la cuenta, en ese orden, como ya hacen los triggers de `00003`. Dos `PATCH` simultáneos sobre la misma línea se ejecutan uno tras otro; el segundo valida contra el estado ya modificado por el primero. `version_bloqueo` lo incrementan los triggers y no se expone.

**D8. Lectura y totales.**
`GET /pedidos/{id}` devuelve por cuenta: líneas, `subtotal` (suma de `base_imponible` de líneas activas), `impuesto` (total menos subtotal) y `total` (suma de `importe_total` de líneas activas). Se calculan en la consulta; no se persisten. El estado de cobro queda fuera hasta I4.

## 4. Estructura del código

Sigue las capas y la dirección de imports del mapa de módulos:

- `domain/vo`: cantidad (3 decimales, > 0), importe (2 decimales), tipo de pedido y estado. `domain/policy`: permisos de pedidos y reglas de transición de edición.
- `application`: comandos `AbrirPedido`, `AgregarDetalle`, `EditarDetalle`; consultas `ListarPedidos`, `ObtenerPedido`; puertos de salida para el repositorio de pedidos y para la lectura de producto con tasa vigente y de mesa.
- `infrastructure/persistence`: lee `productos`, `productos_sucursal`, `tasas_impuesto` y `mesas` con SQL propio; **no** importa `catalogo` (como `atencion` ya hace con `sucursales`). Traduce las violaciones de restricción y los errores de trigger a errores de dominio.
- `web`: controller, DTO de petición y respuesta, y mapper; rutas registradas con `Idempotency-Key` donde corresponde.
- Contrato: se añaden las operaciones y esquemas a `openapi` y se amplía la prueba de contrato de I2.

## 5. Pruebas

- **Dominio:** VO de cantidad e importes; reglas de edición.
- **Casos de uso:** apertura por tipo (SALON con capacidad excedida, mesa no habilitada, LLEVAR con mesa, DELIVERY sin reparto); instantánea de precio y tasa; producto inactivo o no disponible; descuento sin permiso; `ADICIONAL` con y sin padre.
- **Integración (Postgres):** rollback de la apertura si falla la cuenta (RF-009); doble apertura simultánea de la misma mesa (solo una gana, la otra recibe 409); cambiar el precio de carta no altera una línea existente (RF-010); cantidad ≤ 0 rechazada; reintento con la misma clave no duplica.
- **Autorización por sede:** usuario de la sede A no lee ni modifica pedidos de la sede B (404/403), con ID conocido (RF-002).
- **Contrato:** respuestas y errores contra el OpenAPI.
- **Arquitectura:** `tests/architecture` sigue verde (`atencion` no importa `catalogo` ni `identidad`).

## 6. Trazabilidad

Al cerrar la rebanada se actualiza el expediente: `estado_backend` de RF-009 y RF-010 a `IMPLEMENTADO` (casos de aceptación `NO_EJECUTADA` hasta la ejecución formal); RF-002 pasa de `PARCIAL` a `IMPLEMENTADO` en lo verificable con pedidos; `avance` de I3 en `backlog_incrementos.json` y su estado a `EN_CURSO`; y la matriz de bajas y formularios se extiende con las entidades nuevas (los pedidos no tienen baja lógica: se cancelan, en R3).

## 7. Ajustes durante el plan

Al escribir el plan (`docs/superpowers/plans/2026-10-27-i3-rebanada1-pedidos.md`) se reconcilió este diseño con el código y el contrato existentes. Las diferencias con las secciones anteriores son:

| Sección | Diseño | Implementado | Motivo |
|---|---|---|---|
| D5, D4 | Permiso nuevo `PEDIDO_DESCONTAR` | `DESCUENTO_AUTORIZAR` (ya existe desde `00001`), asignado a CAJERO; solo se exige para un descuento distinto de 0 (al agregar y al editar una línea) | Evitar un permiso duplicado |
| D5 | `PEDIDO_CREAR` nuevo | Ya existe; `00011` lo asigna a MESERO y CAJERO y crea solo `PEDIDO_VER` y `PEDIDO_EDITAR` | Evitar insertarlo dos veces |
| D2 | `notas_preparacion`, `descuento_total`, `cuenta_id` opcional | `instrucciones`, `descuento_monto`, `cuenta_id` obligatorio | Nombres del contrato `ConsumoAlta` |
| D1 | `reparto` con `ubigeo`, coordenadas y `programado_en` | `reparto` = `DeliveryDatos` (destinatario, teléfono, dirección, referencia, canal) | Los campos extra son de `PUT /pedidos/{id}/delivery` (I6) |
| D4 | Código `permiso_insuficiente` | `sin_permiso` | Código ya usado por todo el módulo |
| D1 | Repetición idempotente: 200 | 201 (el estado original) | Así lo hace `idempotency.Run` |
| D7 | `version_bloqueo` no se expone | Se expone de solo lectura | Lo exige el esquema `Pedidos` |
| §2 | `POST detalles` con `PEDIDO_CREAR` (contrato) | `PEDIDO_EDITAR` | Lo fija D3/D5; el contrato se corrigió |
| D1 | `comensales` por defecto 1 | Obligatorio: el contrato `PedidoAlta` lo declara así y el código responde 422 `comensales_invalidos` si falta o es menor que 1 | Alinearse con el contrato |
| D1 | Conflicto de idempotencia: 409 `idempotencia_conflicto` | 409 `idempotency_key_reutilizada` | Código de la plataforma; las pruebas lo afirman |
| §4, §5 | Prueba de contrato | No existe una prueba automática de OpenAPI en el repositorio; el alineamiento con el contrato se comprueba a mano y las pruebas de integración decodifican las respuestas en estructuras con la forma del contrato | La prueba automática no se llegó a escribir |

### Reglas para las rebanadas siguientes

- Toda escritura que toque las líneas o las cuentas de un pedido debe bloquear el pedido `FOR UPDATE` primero y después la cuenta o el detalle. R2 y R3 mantienen este orden. `EditarDetalle` bloquea pedido, detalle y cuenta; `AgregarDetalle` bloquea pedido, cuenta y detalle padre; ambos son seguros solo porque el bloqueo del pedido va primero.
- Pendiente de decisión (D4): quien no tiene `DESCUENTO_AUTORIZAR` puede bajar la `cantidad` de una línea que ya tiene descuento, con lo que el descuento pesa más sobre la línea. Decidir antes de R3 si se exige el permiso cuando la línea resultante conserva un descuento distinto de 0.
