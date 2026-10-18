# ADD — service-restaurant (backend)

Documento de diseño de arquitectura del backend. Estado: esqueleto implementado (núcleo, migraciones, pruebas, CI); sin módulos de negocio todavía. Base: expediente en `../../docs/planificacion` (ADR-01, ADR-02, ADR-06, ADR-07) y `../../docs/estructura/PLANTILLA_BACKEND_GO.md`.

## 1. Contexto y alcance

Backend de un sistema de restaurante MYPE/PYME: una empresa por base, varias sucursales. PostgreSQL 18 es la autoridad de integridad (SQL V4: 49 tablas, restricciones, funciones y triggers). El backend valida permisos, sucursal, estados y unidades de trabajo.

Fuera de alcance de este esqueleto: autenticación y permisos (I1), módulos de negocio (I2–I8), integración fiscal, frontend.

## 2. Estilo

Monolito modular (ADR-001): un binario `api`, un binario `migrate`, un esquema PostgreSQL (`sch_restaurante`). Clean Architecture con puertos y adaptadores, CQRS lógico y `Result` para rechazos esperados (ADR-002).

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
- `GET /usuarios` no existe en el contrato (ni lo pide RF-033): no se implementó.
- `UsuarioAlta.nombre` se acepta pero no se persiste (la tabla `usuarios` no tiene nombre; pertenece a empleados).
- `GET /roles` y `GET /permisos` declaran el parámetro `sucursal_id`: se ignora (son catálogos globales).
- Las respuestas repetidas por idempotencia no marcan la repetición con un encabezado.
- El token JWT no incluye refresh (el esquema `Token` no lo prevé).

## 9. Pendiente

Recuperación de contraseña, MFA, rate limiting por IP, rotación de claves JWT (`kid`), refresh de sesión; token de edición concurrente (ETag) en PATCH; rol de BD de operación separado del propietario del esquema (ADR-07); limpieza de registros de idempotencia y sesiones expiradas; módulos de negocio de I2 en adelante; contrato HTTP contra `api/openapi.yaml`; despliegue y respaldo (DEC-03, DEC-08).
