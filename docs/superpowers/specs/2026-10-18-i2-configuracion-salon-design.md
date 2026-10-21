# Diseño I2 (rebanada 1) — Configuración y salón: negocio, sucursales, zonas y mesas

Fecha: 2026-10-18 · Estado: aprobado por secciones en conversación; pendiente de revisión del documento.
Requisitos: RES-RF-003 (negocio y sedes), RES-RF-008 (mesas habilitadas), con RES-RF-030 (bajas y restauraciones) y RES-RF-036 (formularios y errores) en lo que toca a estas entidades. Decisiones relacionadas: DEC-07 (reglas de atención, sigue pendiente), ADR-02, ADR-03, ADR-04. Código: `service-restaurant/` y `app-restaurant/`.

## 0. Alcance y partición de I2

El backlog agrupa en I2 siete requisitos (RF-003, 004, 005, 008, 027, 030, 036; 36–56 h). Se parte en rebanadas:

- **Rebanada 1 (este documento):** negocio, sucursales, zonas y mesas, con su pantalla de Configuración.
- **Rebanada 2 (otro diseño):** carta y productos por sede (RF-005), clientes y proveedores (RF-004) y tasas de impuesto (RF-027).

Quedan fuera de esta rebanada: tasas, series, cajas, almacenes y estaciones de preparación (que la guía de pantallas también ubica en Configuración), la ocupación de mesas (depende de pedidos, I3) y las reservas (I6). El Salón de pedidos, el plano y el resumen siguen en demostración hasta I3.

## 1. Decisiones

| Tema | Decisión |
|---|---|
| Módulos | Negocio y sucursales se agregan a `identidad`; zonas y mesas van en un módulo nuevo `atencion` (según `module-map.md`, que I3 ampliará con pedidos). |
| Permisos | Seis permisos nuevos: `NEGOCIO_VER`, `NEGOCIO_ADMINISTRAR`, `SUCURSAL_VER`, `SUCURSAL_ADMINISTRAR`, `MESA_VER`, `MESA_ADMINISTRAR`. Zonas y mesas comparten los dos `MESA_*`. |
| Baja de mesa | **Con motivo** (decisión del equipo). Se aparta del expediente, que define `mesas` como `ACTIVIDAD_SIMPLE` y cuyo contrato dice «no recibe motivo». Ver §2 y §7. |
| Baja de zona | Desactivación simple, sin motivo (como en V4). |
| Negocio | Singleton: solo `GET` y `PATCH`. Lo crea `cmd/admin` en la instalación. |
| Idempotencia (ADR-04) | Los `POST` de creación, baja y restauración exigen `Idempotency-Key`, igual que en I1. |
| Frontend | Configuración pasa a la API real y se agrega el mantenimiento de zonas y mesas. El Salón sigue en demo. |

## 2. Migración `00008_configuracion_salon`

**Permisos.** Se insertan los seis permisos. El rol `ADMIN` recibe los seis; `MESERO` y `CAJERO` reciben `MESA_VER` (I3 e I4 lo necesitarán y es solo lectura). `COCINA` y `ALMACEN` no reciben nada. No se tocan asignaciones existentes.

**Baja de mesa con motivo.** A `mesas` se agregan `eliminado_en TIMESTAMPTZ`, `eliminado_por BIGINT` y `motivo_eliminacion TEXT`, con el mismo `CHECK` que `sucursales` (activa: los tres nulos; inactiva: los tres presentes y el motivo no vacío). El trigger `a0_metadatos` de `mesas` se recrea con `fn_metadatos_fila('true','true')`, que ya sabe exigir el motivo, fijar actor y fecha de baja y limpiarlos al restaurar. `zz_auditoria` sigue registrando el evento completo. Las filas existentes de `mesas` quedan válidas: las inactivas no existen todavía en una base nueva; si las hubiera, la migración les asigna el motivo `Baja anterior a la migración 00008` y los marcadores a partir de `actualizado_en`/`actualizado_por`.

**Sin cambios:** `zonas` (sin motivo), el trigger `a2_baja_operativa` (ya impide dar de baja una mesa con pedido abierto) y `fn_referencias_activas` (valida que la sede y la zona referenciadas estén activas).

**Consecuencia documental:** `mesas` pasa a ser el décimo maestro con baja y motivo. Se actualizan RF-030 («9 maestros» → 10; «17 desactivaciones simples» → 16), `politica_campos_por_tabla.json`, el diccionario de datos, `matriz_auditoria.html` y el contrato (§6).

## 3. Autorización

Los permisos efectivos siguen viniendo de `v_permisos_efectivos` como pares (sede, permiso); `actor.Can(permiso, sedeId)` es en memoria.

| Recurso | Regla |
|---|---|
| Negocio | Leer: `NEGOCIO_VER` o `NEGOCIO_ADMINISTRAR` en al menos una sede. Editar: `NEGOCIO_ADMINISTRAR` en al menos una sede. La empresa es única y no pertenece a una sede. |
| Sucursales | Listar y leer: solo las sedes donde el actor tiene `SUCURSAL_VER` o `SUCURSAL_ADMINISTRAR`. Editar, baja y restauración: `SUCURSAL_ADMINISTRAR` en esa sede. Crear: `SUCURSAL_ADMINISTRAR` en al menos una sede. |
| Zonas y mesas | `MESA_VER` o `MESA_ADMINISTRAR` (lectura) y `MESA_ADMINISTRAR` (escritura) en la sede del recurso. Una sede ajena responde 404 para no revelar su existencia. |

Quien crea una sede no tiene acceso a ella hasta que se la asignen en Equipo y accesos (las asignaciones son por sede). Es coherente con I1 y la interfaz lo avisa (§5).

**Sedes inactivas.** `v_permisos_efectivos` solo concede permisos sobre sedes activas, de modo que al dar de baja una sede el actor pierde sus permisos sobre ella. Para poder restaurarla, una sede inactiva la ven y la gestionan quienes tienen `SUCURSAL_ADMINISTRAR` en alguna sede; las activas siguen la regla de la tabla. Consecuencia: quien tenía asignada solo esa sede queda sin acceso hasta que otro administrador la restaure o lo reasigne.

## 4. API

Base `/api/v1`. Los `POST` de creación, baja y restauración exigen `Idempotency-Key` (UUID); sin ella → 400. Listados con `limit` (máx. 100) y `cursor`, orden estable por id, y filtros `sucursal_id` y `es_activo`. Errores en `application/problem+json` con los códigos de I1: 400, 401, 403, 404 (inexistente o de otra sede), 409 (duplicado o conflicto) y 422 (regla); los errores de campo indican el campo (RF-036).

| Ruta | Acceso |
|---|---|
| `GET /negocio`, `PATCH /negocio` | §3 |
| `GET/POST /sucursales`, `GET/PATCH /sucursales/{id}`, `POST …/baja` (con `motivo`), `POST …/restauracion` | §3 |
| `GET/POST /zonas`, `GET/PATCH /zonas/{id}`, `POST …/baja`, `POST …/restauracion` | §3 |
| `GET/POST /mesas`, `GET/PATCH /mesas/{id}`, `POST …/baja` (con `motivo`), `POST …/restauracion` | §3 |

**Reglas de negocio**

- **Negocio:** `PATCH` parcial de razón social, nombre comercial, dirección fiscal, ubigeo, teléfono, email, moneda y zona horaria. El RUC y el `singleton` no se editan. La moneda cumple `^[A-Z]{3}$`; la zona horaria se valida como zona IANA; el ubigeo, 6 dígitos.
- **Sucursales:** el código es único. `negocio_id` no se acepta del cliente. No se puede dar de baja la última sede activa del negocio (422). Dar de baja una sede no modifica sus operaciones. La restauración conserva el ID (regla 16: no duplicar).
- **Zonas:** nombre único por sede. La baja se rechaza con 409 si la zona tiene mesas activas.
- **Mesas:** número único por sede; la zona debe pertenecer a la misma sede (clave compuesta); capacidad > 0. `habilitada` se cambia con `PATCH`, sin motivo. Una mesa dada de baja no se edita ni se habilita hasta restaurarla. La ocupación no se expone (se deriva de pedidos, I3).
- **Traducción de errores:** las violaciones de unicidad, de `CHECK` y de los triggers V4 se traducen a 409/422 con un mensaje por campo, como en I1.
- **Actor:** `creado_por`, `actualizado_por` y los marcadores de baja nunca se aceptan del cliente; salen de la sesión mediante `app.usuario_id`.

## 5. Estructura del backend y casos de uso

Mismo esquema CQRS de I1 (`domain`, `application`, `infrastructure`, `web` por módulo):

- `identidad`: se agregan comandos y consultas de negocio y sucursales (`ObtenerNegocio`, `EditarNegocio`, `ListarSucursales`, `ObtenerSucursal`, `CrearSucursal`, `EditarSucursal`, `DarBajaSucursal`, `RestaurarSucursal`). La consulta existente `sucursales-asignables` sigue en este módulo, que ya es dueño de la tabla.
- `atencion` (nuevo): zonas y mesas, con puertos de lectura y escritura propios.
- `atencion` no importa `identidad`: la existencia de la sede la garantizan la clave foránea y `fn_referencias_activas`, y el adaptador traduce el error. La coordinación ocurre en la composición, como exige la regla de imports.
- `tests/architecture` pasa a cubrir `atencion`.

## 6. Contrato y documentación

- En `docs/planificacion/03_api/openapi.yaml`: `x-permiso` de las operaciones de esta rebanada con los permisos nuevos; baja de mesa con cuerpo `{motivo}` (sustituye «No recibe motivo»); se agregan `GET /usuarios` y `GET /sucursales-asignables` (implementados en I1 sin contrato).
- `service-restaurant/api/openapi.yaml` pesa 0 bytes desde el primer commit y el README ya remite al contrato de `docs/`: se elimina el archivo vacío.
- ADR 005 («Baja de mesa con motivo») en `service-restaurant/docs/adr/`, con el alcance de §2; se actualizan `docs/module-map.md` y el README.
- Expediente: RF-003 y RF-008 pasan a IMPLEMENTADO al cerrarse la rebanada; RF-030 y la matriz de auditoría reflejan el décimo maestro.

## 7. Frontend

Sigue el patrón de Equipo y accesos: un contenedor elige entre real y demo, y un store propio de la página trae todo del servidor.

- `features/configuracion/` (nueva): `data-access/configuracion-api.ts` y `configuracion.models.ts` (IDs como cadenas); `data-access/configuracion-store.ts` (`@Injectable()` provisto en la página, con `status`, `loadError`, `notice`, `busyId`, carga con paginación acotada); `configuracion-real.ts` (pantalla con datos reales).
- `configuracion-page.ts` queda como contenedor: con sesión real (`auth.isAuthenticated()`) muestra la pantalla real; en modo demo conserva la de ejemplo, igual que `usuarios-page.ts`.
- Lo que se muestra depende de `auth.hasPermission()`; no es seguridad, el servidor vuelve a validar.

**Pantalla real**

- **Tu restaurante:** formulario de negocio; el RUC se muestra sin poder editarse; sin `NEGOCIO_ADMINISTRAR` queda en solo lectura; sin ningún permiso de negocio la sección no aparece.
- **Sedes:** lista con alta, edición, baja (pide motivo) y restauración. Tras crear una sede, aviso: «Asígnala a tu equipo en Equipo y accesos para que alguien pueda operarla».
- **Zonas y mesas:** agrupadas por sede (con selector si hay varias). Zonas: alta, edición, baja y restauración; con mesas activas se muestra el 409 del servidor. Mesas: número, zona, capacidad y posición opcional; interruptor de habilitación sin motivo; baja con motivo y restauración.
- Se conserva «Restablecer demostración» solo en modo demo.

**Comportamiento común (RF-036):** los diálogos reutilizan `Dialogs.form` y `newIdempotencyKey()` (el doble clic no duplica el comando); los errores de campo se muestran en el campo y conservan lo escrito; los demás errores usan `describeProblem()`; un 403 en una sección muestra el mensaje de falta de permiso en lugar de una pantalla rota. Las acciones devuelven el texto de error al diálogo, que no se cierra hasta que salga bien.

## 8. Pruebas

- **Unitarias (Go):** reglas de dominio de sucursal, zona y mesa; políticas de autorización.
- **Integración (PostgreSQL 18):** sede ajena → 404; baja de zona con mesas activas → 409; baja de mesa sin motivo → 422; número de mesa y código de sede duplicados → 409; no dar de baja la última sede activa → 422; restauración conserva el ID; idempotencia (repetir igual, repetir distinto); usuario con solo `*_VER` no escribe (403); ADMIN recibe los seis permisos y MESERO/CAJERO solo `MESA_VER`; el motivo y el actor de la baja de mesa quedan en la fila y en `auditoria_eventos`; la migración 00008 sube y baja limpia.
- **Arquitectura:** `tests/architecture` cubre `atencion`.
- **Frontend:** `configuracion-store.spec.ts` (Vitest) con carga, errores, permisos y acciones; E2E de Playwright `configuracion.spec.ts` en modo demo y `configuracion.real-api.spec.ts` contra la API real, como los de login y equipo; la CI del front sigue en verde.

## 9. Entrega

Cuatro pasos, cada uno con CI en verde:

1. Migración 00008, permisos, negocio y sucursales.
2. Módulo `atencion`: zonas y mesas.
3. Contrato, ADR 005 y documentos del expediente (§6).
4. Frontend (§7), una vez el backend de los pasos 1 a 3 esté integrado.

## 10. Riesgos y pendientes

- **Desviación del expediente:** la baja de mesa con motivo contradice `politica_campos_por_tabla.json`, el contrato y RF-030; se corrige en el paso 3 y queda registrada en el ADR 005. Si el equipo revierte la decisión, basta omitir las columnas y el trigger de §2.
- **Sede recién creada sin acceso:** depende de la asignación manual en Equipo y accesos; si resulta incómodo, un cambio posterior podría asignar automáticamente al creador (decisión fuera de esta rebanada).
- **Administradores sin sede activa:** si se da de baja la única sede asignada a todos los administradores de sedes, nadie podría restaurarla por la API; se resuelve con cmd/admin o por SQL. Mitigación posible fuera de esta rebanada: impedir la baja si deja a la empresa sin administradores de sedes.
- **DEC-07 (reglas de atención)** sigue pendiente; afecta la reserva y ocupación de mesas, no esta rebanada.
- **Mesa deshabilitada y reservas:** la interacción con `reservas` (I6) se define allí.
- Fuera de alcance: tasas, series, cajas, almacenes, estaciones, ocupación de mesas, plano del salón con posiciones editables en pantalla y la rebanada 2 de I2.
