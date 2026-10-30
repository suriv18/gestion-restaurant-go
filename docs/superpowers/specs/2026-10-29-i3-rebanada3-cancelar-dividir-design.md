# Diseño I3 (rebanada 3, backend) — Dividir cuentas y cancelar con motivo

Requisitos: RES-RF-011 (dividir cuentas) y RES-RF-014 (cancelar con motivo, pedido y línea), con RES-RF-002 (autorizar por permiso y sede). Código: `service-restaurant/`, módulo `atencion` (`produccion` no cambia). El frontend de Salón y Cocina y el cierre con auditoría quedan fuera (R4); pagos y devoluciones son de I4.

## 1. Alcance

R3 usa las tablas existentes (`pedidos`, `cuentas`, `detalles_pedido`, `comanda_detalles`) sin cambiar el esquema. Solo añade la migración `00013`, que asigna `PEDIDO_ANULAR` al rol CAJERO (ADMIN ya lo tiene desde `00001`). Se escribe con `ON CONFLICT DO NOTHING` y su `Down` retira únicamente lo que insertó.

Fuera de R3: pagos, devoluciones y ajustes físicos (I4, I5); pantalla de Salón y Cocina (R4); anular lo ya entregado a la mesa.

## 2. Superficie REST

| Endpoint | Función | Permiso |
|---|---|---|
| `POST /pedidos/{id}/dividir-cuenta` (operacion_116) | Mueve líneas a una cuenta nueva del mismo pedido. Cuerpo `{detalle_ids, nombre_cuenta?}`; responde 200 con la cuenta nueva | `PEDIDO_CREAR` (según el contrato) |
| `POST /pedidos/{id}/cancelacion` (operacion_117) | Cancela el pedido entero. Cuerpo `{motivo}`; responde 200 con el pedido | `PEDIDO_ANULAR` |
| `POST /pedidos/{id}/detalles/{detalle_id}/anulacion` (**nueva**) | Anula una línea. Cuerpo `{motivo}`; responde 200 con la línea | `PEDIDO_ANULAR` |

Las tres exigen `Idempotency-Key`. Se marcan `x-implementado: true`. La ruta nueva sube el contrato a 121 rutas y 165 operaciones, de modo que `MANIFIESTO.json`, `00_LEEME.txt` y `validacion_entrega.json` actualizan sus conteos. Errores según el contrato de I2 (RF-036): `422` validación, `409` conflicto de estado, `403`/`404` por sede. `motivo` es obligatorio, recortado, de 1 a 2000 caracteres (422).

## 3. Reglas

### 3.1 Dividir cuenta (RF-011)

1. Se bloquea el pedido `FOR UPDATE` primero; después las líneas y las cuentas por `id` ascendente (orden de R1 y R2).
2. El pedido debe estar abierto, las cuentas de origen `ABIERTA` y las líneas `ACTIVO`; si no, 409. Los adicionales son de un solo nivel: R1 (`validarPadre`) rechaza con 422 `padre_invalido` un padre que ya es adicional.
3. `detalle_ids` no puede estar vacío ni traer repetidos (422 `detalle_duplicado`); todos deben ser del pedido (404 si no).
4. Adicionales: al mover una línea padre viajan sus hijas. Un adicional solo, sin su padre en la selección, da 422 `adicional_sin_padre`.
5. Pago confirmado: si alguna cuenta de origen tiene un pago `CONFIRMADO`, 409 `cuenta_con_pago`. Los pagos son de I4, así que hoy no se dispara en uso real; se implementa y se prueba insertando el pago por SQL.
6. La cuenta nueva toma `numero = max + 1` dentro del pedido, `etiqueta = nombre_cuenta` (opcional), estado `ABIERTA`, sin cliente. La respuesta es esa cuenta con sus líneas.
7. Cocina no se altera: las comandas cuelgan de `detalle_pedido_id`, no de la cuenta, así que mover líneas ya enviadas conserva la trazabilidad.
8. Invariante: la suma de `importe_total` de las cuentas antes y después de dividir es idéntica.

### 3.2 Anular una línea (RF-014)

1. Se bloquea el pedido, después la línea y su cuenta, y al final sus `comanda_detalles`. El pedido debe estar abierto, la cuenta `ABIERTA` y la línea `ACTIVO`; si no, 409 (`pedido_cerrado`, `cuenta_cerrada` o `linea_anulada`).
2. Si la línea tiene adicionales hijos, se anulan con ella.
3. Cocina: las líneas de comanda en `PENDIENTE`, `PREPARACION` o `LISTO` pasan a `CANCELADO` con el mismo motivo. Si alguna está `ENTREGADO`, 409 `linea_entregada` y no cambia nada: lo servido no se anula aquí.
4. La línea queda `ANULADO` con `anulado_en`, `anulado_por` y `motivo_anulacion`. No se borra nada.
5. Si la cuenta tiene un pago `CONFIRMADO`, 409 `cuenta_con_pago` (la devolución es de I4).

### 3.3 Cancelar el pedido (RF-014)

1. Se bloquea el pedido primero. Debe estar abierto (409 `pedido_cerrado` si no).
2. 409 `pedido_con_pago` si alguna cuenta tiene un pago `CONFIRMADO`: el consumo confirmado no se devuelve automáticamente. 409 `linea_entregada` si alguna línea de comanda ya fue entregada.
3. En una sola transacción: el pedido pasa a `CANCELADO` con `cerrado_en` y `motivo_cancelacion`; las cuentas `ABIERTA` pasan a `CANCELADA`; las líneas `ACTIVO` a `ANULADO`; las líneas de comanda a `CANCELADO`. Todos llevan el mismo motivo. La mesa queda libre, porque `uq_pedido_mesa_abierta` solo cuenta pedidos con `cerrado_en` nulo.
4. La respuesta es el pedido completo; las líneas se conservan con su auditoría.

### 3.4 Lectura

Hoy ninguna consulta de `atencion` trata `ANULADO`. El total de una cuenta y del pedido pasa a ignorar las líneas `ANULADO`, que siguen visibles con su estado y su motivo. Afecta a la lectura de pedidos de R1 y se cubre con pruebas.

### 3.5 Decisión heredada D4

`EditarDetalle` (R1) exige `DESCUENTO_AUTORIZAR` (403 `sin_permiso_descuento`) cuando la cantidad baja y la línea resultante conserva `descuento_total > 0`. Sin el permiso, antes se podía bajar la cantidad y el descuento pesaba más sobre la línea.

## 4. Arquitectura

Todo en `internal/modules/atencion`, con las capas existentes:

- `domain`: reglas puras: qué estados de cocina admiten anular, qué líneas viajan con un padre.
- `application`: casos de uso `DividirCuenta`, `AnularDetalle` y `CancelarPedido` en `usecase/command`, con DTO y puertos. `PedidoStore` crece con los bloqueos y escrituras necesarios (se detallan en el plan).
- `infrastructure/persistence`: el SQL de `pedidos`, `cuentas`, `detalles_pedido` y, para la cascada a cocina, `comanda_detalles`.
- `web`: controladores y DTO.

Cascada a cocina: SQL en la persistencia de `atencion`, dentro de la misma transacción. R1 ya consulta `comanda_detalles` desde `atencion` y R2 consulta `pedidos` desde `produccion`. `produccion` no cambia y `tests/architecture` debe seguir en verde. Se descartó un puerto implementado por un adaptador de `produccion`: respeta mejor la propiedad de la tabla, pero exige pasar la transacción entre módulos desde el arranque, sin precedente en el repositorio.

Concurrencia: orden de bloqueo pedido, cuentas y líneas por `id` ascendente, y al final las líneas de comanda; el mismo que R1 y R2, así que no hay interbloqueos con el envío a cocina.

## 5. Pruebas

- Unitarias de dominio y casos de uso: estados de cocina que admiten anular, arrastre de adicionales, permisos, motivo y D4.
- Integración con Postgres real, como en R1 y R2:
  - Dividir conserva la suma de importes; con pago confirmado insertado por SQL se rechaza; los adicionales viajan con su padre.
  - Anular una línea enviada cancela su comanda hasta `LISTO`; con una `ENTREGADO` se rechaza sin cambiar nada.
  - Cancelar un pedido deja cuentas, líneas y comandas cancelados con el mismo motivo, libera la mesa y conserva las líneas.
  - Los totales ignoran las líneas anuladas.
  - Permisos y sedes; CAJERO puede anular; idempotencia de las tres rutas.
- Concurrencia: una cancelación y un envío a cocina simultáneos sobre el mismo pedido dejan un resultado coherente (se envió y luego se canceló, o se canceló y el envío falla con 409).
- Arquitectura: `tests/architecture` en verde.

## 6. Cierre del expediente

- RES-RF-011 y RES-RF-014 pasan a `IMPLEMENTADO` en backend (los casos de aceptación siguen `NO_EJECUTADA`).
- Se marcan las operaciones 116 y 117 como implementadas y se añade la ruta nueva al contrato con sus esquemas; `MANIFIESTO.json`, `00_LEEME.txt` y `validacion_entrega.json` actualizan sus conteos (121 rutas, 165 operaciones).
- `avance` de I3 en `backlog_incrementos.json`, `CAMBIOS_1_3.txt`, `SHA256SUMS.txt` y `service-restaurant/docs/module-map.md`.

## 7. Pendientes

- R4: frontend de Salón y Cocina y cierre con auditoría.
- I4: pagos, devoluciones y cierre del pedido con saldo; hasta entonces `cuenta_con_pago` y `pedido_con_pago` se prueban con pagos insertados por SQL.
- `pedidos.estado` (`REGISTRADO/PREPARACION/LISTO/ENTREGADO`) sigue sin derivarse de las comandas; se decide en R4 o I4.
- Una línea con un lote ya entregado no se puede anular ni reducir (R1 prohíbe editar una línea que ya está en cocina); las reducciones parciales de líneas servidas siguen pendientes.
- Un pedido de SALON cuya mesa se deshabilitó (`habilitada = false`) no se puede cancelar, porque `fn_validar_pedido` revisa la mesa en cada UPDATE de `pedidos`; bloquear la deshabilitación de una mesa con pedido abierto queda pendiente.
- Al cancelar un pedido de DELIVERY, `pedidos_delivery.estado` queda en PENDIENTE hasta I6.

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
