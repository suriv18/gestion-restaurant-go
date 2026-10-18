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

## 8. Pendiente

Autenticación y sesiones (DEC-06, I1); permisos faltantes, idempotencia general y outbox (migraciones nuevas); contrato HTTP contra `api/openapi.yaml`; paginación real; rol de BD de operación separado del propietario del esquema (ADR-07); roles de operación, despliegue y respaldo (DEC-03, DEC-08).
