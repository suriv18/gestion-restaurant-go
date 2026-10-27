# ADD — service-restaurant (backend)

Documento de diseño de arquitectura del backend. Estado: esqueleto implementado (núcleo, migraciones, pruebas, CI); módulos `identidad` (I1, más negocio y sucursales en I2) y `atencion` (I2: zonas y mesas). Base: expediente en `../../docs/planificacion` (ADR-01, ADR-02, ADR-06, ADR-07) y `../../docs/estructura/PLANTILLA_BACKEND_GO.md`.

## 1. Contexto y alcance

Backend de un sistema de restaurante MYPE/PYME: una empresa por base, varias sucursales. PostgreSQL 18 es la autoridad de integridad (SQL V4: 49 tablas, restricciones, funciones y triggers). El backend valida permisos, sucursal, estados y unidades de trabajo.

Fuera de alcance de este esqueleto: autenticación y permisos (I1), módulos de negocio (I2–I8), integración fiscal, frontend.

## 2. Estilo

Índice de ADR (`docs/adr/`): 001 monolito modular, 002 CQRS/Result/transacciones, 003 persistencia y versiones, 004 acceso JWT/sesión y tablas técnicas, 005 baja de mesa con motivo. Monolito modular (ADR-001): un binario `api`, un binario `migrate`, un esquema PostgreSQL (`sch_restaurante`). Clean Architecture con puertos y adaptadores, CQRS lógico y `Result` para rechazos esperados (ADR-002).

## 3. Estructura

```text
cmd/api, cmd/migrate        arranque; delegan en bootstrap / platform
internal/bootstrap          composición (app, router, ciclo de vida)
internal/platform           técnica transversal: config, database, transaction,
                            observability, security, web (handler, exception,
                            middleware, pagination, response)
internal/shared             kernel puro (result, apperror) y DTO de aplicación
internal/modules/<m>        un módulo de negocio: domain, application,
                            infrastructure, web
migrations                  secuencia única de migraciones Goose (embebidas)
tests                       architecture, integration, contract, fixtures
```

Cada módulo tiene cuatro capas. **`web` es el adaptador de entrada y vive fuera de `infrastructure`** (decisión de este proyecto; la plantilla lo ubica dentro). Reglas completas en `module-map.md` y verificadas por `tests/architecture`.

## 4. Flujo de una solicitud

HTTP → `RequestID` → `Recover` → `MaxBytesHandler` → handler (`Endpoint` que devuelve `error`) → puerto de entrada → decorador transaccional (`infrastructure/config`) → caso de uso → puertos de salida → adaptadores SQL sobre el **mismo `*sql.Tx`**.

- Rechazo esperado: `Result` fallido con `AppError` → rollback → `application/problem+json` con el estado HTTP de su `Kind`.
- Error técnico o panic: rollback, 500 opaco (sin detalles internos), causa en el log con `request_id`.
- Éxito: commit comprobado; solo después se responde.

## 5. Transacciones y actor

`platform/transaction.Run` es la única API de transacciones. Hace commit solo si el trabajo no devuelve error y el predicado (`result.CanCommit`) lo permite; un fallo de commit nunca se informa como éxito. READ COMMITTED (predeterminado) es obligatorio para cobros, devoluciones y stock.

La V4 exige un actor por transacción: `set_config('app.usuario_id', <id>, true)`, tomado de la sesión validada (nunca de un header o del cuerpo). Sin actor, la base rechaza la escritura. El usuario técnico `__sistema__` no autentica.

## 6. Datos y migraciones

El SQL V4 se dividió en 6 migraciones por dominio sin cambiar sus sentencias (esquema verificado idéntico al original). Ver ADR-003. Las migraciones se ejecutan como paso controlado (`cmd/migrate`), no al arrancar cada réplica.

## 7. Verificación

| Capa | Mecanismo |
|---|---|
| Unidad | `go test` junto al código |
| Arquitectura | `tests/architecture`: dirección de imports; reglas probadas con casos sintéticos |
| Integración | `tests/integration`: testcontainers + PostgreSQL 18; esquema (49/664/187), semillas, actor, runner (commit, rechazo, error técnico, fallo de commit) |
| CI | GitHub Actions (`.github/workflows/service-restaurant-ci.yml`): formato, vet, `-race`, integración, build, imagen |

## 8. Estado e implementación de I1 (identidad)

Implementado: acceso (login, logout, sesión; JWT HS256 + sesión en BD, ADR-004), autorización por permiso y sede (`Actor.Can`), idempotencia genérica (`platform/idempotency`), opción de actor en transacciones y administración de accesos (RES-RF-033: roles, permisos, usuarios, asignaciones, baja y restauración).

Reglas de seguridad de la administración, verificadas con pruebas (y por mutación):
- Todas las rutas exigen `USUARIO_ADMINISTRAR` en alguna sede; las asignaciones solo agregan o quitan sedes donde el administrador lo tiene.
- Ningún cambio puede dejar al sistema sin un administrador efectivo (se aplica, se cuenta y, si queda en 0, se revierte).
- Nadie se da de baja a sí mismo; el actor técnico `__sistema__` es invisible para la API (404).
- La baja de un usuario revoca sus sesiones y restaurarlo no las revive.
- La autoría (`creado_por`/`actualizado_por`) sale del actor autenticado (`transaction.WithActor`), nunca del cuerpo.

### Desviaciones y huecos del contrato OpenAPI 0.2.0
- `POST /auth/logout` no existe en el contrato; se agregó y debe incorporarse al OpenAPI.
- `GET /usuarios` y `GET /sucursales-asignables` **no existen en el contrato** y se agregaron porque la pantalla «Equipo y accesos» no puede funcionar sin ellos: el primero lista al equipo con sus roles y sedes vigentes (excluye siempre al actor técnico, devuelve las bajas con su motivo y nunca el hash); el segundo ofrece solo las sedes donde el administrador tiene `USUARIO_ADMINISTRAR`, que es exactamente el conjunto que la regla de asignación le permite tocar. Ambos exigen `USUARIO_ADMINISTRAR` y se incorporaron al OpenAPI en el paso 3 de I2.
- No hay endpoints para asignar permisos a un rol ni para consultar un usuario por id; hoy la asignación rol-permiso solo se hace por SQL.
- `UsuarioAlta.nombre` se acepta pero no se persiste (la tabla `usuarios` no tiene nombre; pertenece a empleados). Para guardar el nombre, el alta acepta un objeto `empleado` (`tipo_documento`, `numero_documento`, `nombres`, `apellidos`, `telefono`, `cargo`) que crea al empleado vinculado en la misma transacción y exige `EMPLEADO_ADMINISTRAR`; `GET /usuarios` devuelve ese `empleado` (id, nombres, apellidos, cargo) o `null`.
- `/empleados` (`GET`, `POST`, `GET/PATCH /{id}`, `/baja`, `/restauracion`) no existe en el contrato 0.2.0; se agregó con los permisos `EMPLEADO_VER` y `EMPLEADO_ADMINISTRAR` (migración 00009). La baja de un empleado se rechaza (409 `empleado_con_usuario_activo`) mientras su usuario siga activo.
- `/clientes` y `/proveedores` se implementan con la baja con cuerpo `{motivo}` (RES-RF-030), sin el filtro `sucursal_id` (maestros globales) y con los permisos `CLIENTE_*` y `PROVEEDOR_*` (migración 00009); un documento dado de baja sigue reservado.
- Carta (rebanada 3 de I2), contrato 0.2.0 corregido: `limit` por defecto 25 (no 30); `TasasImpuestoEdicion` sin `impuesto_id` y `EstacionesPreparacionEdicion` sin `sucursal_id` (se crea otra tasa; una estación no cambia de sede); decimales sin signo (tasa 0-100 con 4 decimales, precio 2, factor 6), normalizados al responder; `GET /tasas-impuesto` y `GET /categorias` sin `sucursal_id` (maestros globales), el primero con `impuesto_id`; `GET /productos` con `es_activo` por defecto solo activos y la extensión `todos` (categorías y clientes sin parámetro listan todos), `categoria_id`, `disponible` y `sucursal_id`, que añade `en_sucursal` (`disponible`, `precio_sucursal`, `precio_efectivo`, `estacion_id`; null sin fila activa en esa sede).
- `GET /impuestos` y `GET /impuestos/{id}` no existían en el contrato y se agregaron (solo lectura; sin altas ni ediciones por API). `PUT /productos/{id}/disponibilidad` acepta, como extensión, `precio_venta` y `estacion_id` opcionales y anulables; sin fila en `productos_sucursal` el producto no se ofrece en la sede. Permisos nuevos de la migración 00010: `CARTA_VER`, `CARTA_ADMINISTRAR`, `CARTA_DISPONIBILIDAD`, `ESTACION_VER`, `ESTACION_ADMINISTRAR`, `IMPUESTO_VER`, `IMPUESTO_ADMINISTRAR`; `CARTA_ADMINISTRAR` también habilita la lectura de impuestos y tasas.
- Un producto `REVENTA` no se puede crear hasta I5 (no hay API de artículos; 422 `referencia_invalida`). El 403 `sede_no_autorizada` y el 422 por sede inactiva de las estaciones no son alcanzables por HTTP (los permisos efectivos cruzan roles y sedes activas); están cubiertos por pruebas unitarias. Revisión final: una estación de otra sede, inexistente o inactiva en `PUT /productos/{id}/disponibilidad` responde el mismo 422 `referencia_invalida` (sin revelar estaciones de otras sedes); `impuesto_id` de las tasas se valida en el caso de uso tras el permiso (400 si está mal formado, 422 si falta).
- Pendiente de contrato: `/clientes`, `/proveedores` y `/empleados` siguen con `x-implementado: false` en el OpenAPI.
- `GET /roles` y `GET /permisos` declaran el parámetro `sucursal_id`: se ignora (son catálogos globales).
- Las respuestas repetidas por idempotencia no marcan la repetición con un encabezado.
- El token JWT no incluye refresh (el esquema `Token` no lo prevé).

## 9. Pendiente

Recuperación de contraseña, MFA, rate limiting por IP, rotación de claves JWT (`kid`), refresh de sesión; token de edición concurrente (ETag) en PATCH; rol de BD de operación separado del propietario del esquema (ADR-07); limpieza de registros de idempotencia y sesiones expiradas; frontend de la carta, de impuestos y tasas (la rebanada 3 de I2 es solo backend); antes de I3, impedir editar tasas con uso en ventas; módulos de negocio de I3 en adelante; contrato HTTP en `../docs/planificacion/03_api/openapi.yaml`; despliegue y respaldo (DEC-03, DEC-08).
