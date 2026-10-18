# Diseño: Esqueleto backend Go + cierre de DEC-02 (I0)

## Context
`gestion-restaurant-go` contiene solo planificación (docs/): SQL V4 PostgreSQL 18 (49 tablas), OpenAPI 0.2.0 (112 rutas), prototipo Mesa y `PLANTILLA_BACKEND_GO.md`. No hay código productivo. I0 del backlog es "cerrar alcance y decisiones"; DEC-02 (stack y versiones) sigue PENDIENTE y bloquea I1. Este diseño fija Go como stack y define un esqueleto ejecutable del monolito, sin módulos de negocio.

## Decisiones tomadas (con el usuario)
- **Alcance:** esqueleto Go + cerrar DEC-02. Sin autenticación ni módulos (eso es I1+).
- **Ubicación:** repo git **separado** para el backend, con la estructura de la plantilla en la raíz. `docs/` queda como expediente.
- **Migraciones:** el SQL V4 se **divide por dominio** (seguridad, maestros, pedidos/cocina, caja/pagos, compras/stock, reservas/delivery, comprobantes, auditoría), una migración Goose por dominio, alineadas con I1–I8.
- **BD local/pruebas:** `compose.yaml` (PostgreSQL 18) para desarrollo + `testcontainers-go` para suites de integración.

## Enfoque recomendado
Instanciar `PLANTILLA_BACKEND_GO.md` tal cual: monolito modular, Clean Architecture, puertos/adaptadores, CQRS lógico con `Result`.
- Runtime Go estable fijado en `go.mod` y CI; `net/http` (router estándar); `database/sql` + `pgx/v5/stdlib`; SQL parametrizado; Goose; `log/slog`; `testing` + `httptest`.
- Alternativas descartadas: framework web (chi/gin) y ORM, por no aportar al núcleo y contradecir la plantilla.

## Componentes del esqueleto
1. `cmd/api/main.go` + `internal/bootstrap` (app, router, lifecycle con shutdown ordenado).
2. `internal/platform`: config (env), database (pool pgx), transaction runner (única API de transacciones), observability (slog + request id), web (middleware recover/request_id, errores RFC 9457 `problem`, paginación).
3. `internal/shared/kernel`: `result`, `apperror`; `application/context/actor` (actor de escritura vía `set_config` local a la transacción, según LEEME de la API).
4. `internal/modules/` vacío salvo un módulo ejemplo mínimo (health/versión de esquema) para probar el patrón.
5. `migrations/`: baseline troceado por dominio desde `docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql` (reubicar triggers/funciones/vistas con su dominio; respetar orden de dependencias de FK).
6. `tests/architecture/import_rules_test.go` (reglas de importación), `tests/integration` (testcontainers aplica migraciones y verifica 49 tablas), `tests/contract` (health contra `api/openapi.yaml`).
7. `Makefile`, `Dockerfile`, `compose.yaml`, `.env.example`, `.github/workflows/ci.yml`.

## Flujo de datos
HTTP → middleware (request id, recover) → handler → caso de uso (Command/Query, devuelve `Result`) → puerto → repositorio pgx dentro del `transaction runner` → PostgreSQL. Errores `apperror` se mapean a `problem+json` en un único manejador.

## Documentación a producir
- ADR/ADD del repo backend (001 monolito modular, 002 CQRS/Result/transacciones, 003 persistencia y versiones).
- Actualizar `decisiones_pendientes.json` (DEC-02 → CERRADA con versión de Go, PostgreSQL 18, drivers, evidencia = prueba técnica de migraciones + health en CI).

## Riesgos / puntos abiertos
- Trocear el SQL por dominio puede romper dependencias circulares entre FKs/triggers: se valida aplicando todas las migraciones en orden sobre PostgreSQL 18 real y comparando contra `catalogo_estructural.json` (49 tablas, 664 columnas).
- Docker requerido en Windows para testcontainers.
- DEC-06 (identidad/sesiones) queda fuera: I1.

## Verificación
- `make test` (unit + architecture) verde.
- `make test-integration`: testcontainers PostgreSQL 18 aplica migraciones; assert 49 tablas/664 columnas y que el baseline coincide con el SQL original.
- `docker compose up` + `curl /healthz` y `/readyz` devuelven 200; apagado limpio con SIGTERM.
- CI ejecuta lint, tests y build de la imagen.
