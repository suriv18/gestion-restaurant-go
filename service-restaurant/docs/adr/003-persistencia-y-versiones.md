# ADR-003 — Persistencia, migraciones y versiones

Estado: aceptado para este repositorio (2026-10-17). Aporta evidencia a DEC-02 (backend) y a ADR-06/ADR-07 del expediente; **no cierra DEC-02** (ver "Límites").

## Decisión

| Área | Elección |
|---|---|
| Lenguaje | Go 1.26.5 (fijado en `go.mod`) |
| Base de datos | PostgreSQL 18 (`postgres:18` en compose y pruebas) |
| Acceso | `database/sql` + `github.com/jackc/pgx/v5` v5.11.0 (`stdlib`); SQL parametrizado, sin ORM |
| Migraciones | `github.com/pressly/goose/v3` v3.29.0, SQL embebido (`migrations/migrations.go`), ejecutado por `cmd/migrate` |
| Pruebas de integración | `testcontainers-go` v0.44.0 con `postgres:18` |
| Logs | `log/slog` JSON |
| HTTP | `net/http` con `ServeMux` estándar |

## Migraciones por dominio

El SQL V4 (`docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql`) es el baseline y se dividió **sin cambiar sentencias** en seis migraciones alineadas con los incrementos: `00001` seguridad/personas/auditoría (I1), `00002` catálogos (I2), `00003` salón/pedidos/cocina (I3), `00004` caja (I4), `00005` compras/kardex (I5), `00006` comprobantes (I7). Cada una incluye `Down`. Comprobado contra PostgreSQL 18: el volcado de esquema (`pg_dump --schema-only`) coincide con el del archivo original (49 tablas, 664 columnas, 187 triggers, mismas semillas); bajar las seis y volver a subirlas funciona.

Detalles que importan:

- Las migraciones fijan `SET LOCAL search_path = sch_restaurante, pg_catalog`; por eso la tabla de versiones de Goose se califica como `public.goose_db_version`.
- Las semillas de migraciones posteriores a la primera fijan el actor técnico con `set_config('app.usuario_id', …)`.
- Los cambios nuevos van como migraciones adicionales (permisos faltantes, idempotencia, outbox, token de edición concurrente), nunca editando las seis existentes.

## Límites (qué falta para cerrar DEC-02)

ADR-06 exige comparar al menos dos combinaciones de stack y una prueba corta con **autenticación**, transacción parametrizada y consulta de catálogo, registrando fecha y responsable real. Aquí:

- Hecho: transacción parametrizada con rollback verificado, empaquetado (imagen de 35 MB), migraciones y CI en verde.
- **No hecho:** comparación documentada de alternativas, prueba con autenticación (I1/DEC-06), consulta de catálogo, confirmación del responsable técnico. La elección de Go fue indicada por el equipo; no se evaluó otra pila.
- Pendiente de ADR-07: rol de BD de operación separado del propietario del esquema; ensayo de recuperación y migración sobre copia representativa.

## Verificación

`tests/integration/migrations_test.go` (esquema, semillas, actor) y `catalogo_transaction_test.go` (runner), ejecutados en CI con PostgreSQL 18.
