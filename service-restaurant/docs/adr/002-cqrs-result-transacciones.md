# ADR-002 — CQRS lógico, Result y transacciones sin filtrar `sql.Tx`

Estado: aceptado para este repositorio (2026-10-17). Decisión de la plantilla, no una exigencia del lenguaje.

## Contexto

Los casos de uso deben distinguir un rechazo esperado del negocio de un fallo técnico, y las operaciones de escritura deben ser atómicas sin que la aplicación conozca `database/sql`.

## Decisión

- **CQRS lógico:** comandos y consultas son casos de uso distintos (`usecase/command`, `usecase/query`) con DTO propios; lecturas con proyecciones y paginación (`page=0`, `size=25`, máximo 100).
- **`Result[T]`** (`internal/shared/kernel/result`): éxito o `AppError` esperado. El valor cero es inválido; solo `Ok` y `Fail` crean valores válidos. Doble canal solo en casos de uso: si `error != nil` se ignora el `Result` y se aborta; si `error == nil`, el `Result` debe ser válido.
- **Transacciones:** `platform/transaction.Run` ejecuta el trabajo en una `sql.Tx`; hace commit solo si no hay error y `result.CanCommit` lo permite; rollback en cualquier otro caso (el fallo de rollback se registra sin perder el error principal). Un decorador por caso de uso en `infrastructure/config` construye los repositorios con **ese mismo `*sql.Tx`**; la aplicación nunca ve `sql.Tx`. No se anidan `BeginTx`.
- Errores HTTP: `application/problem+json`; los errores no reconocidos devuelven 500 opaco y se registran con `request_id`.

## Consecuencias

La aplicación y el dominio no importan SQL ni HTTP (verificado por `tests/architecture`). El costo es un decorador transaccional por caso de uso. Tras un error de comunicación en el commit no se puede afirmar que el cliente pueda reintentar cualquier comando: los procesos que lo necesiten requieren clave de idempotencia (pendiente, ver ADD §8).

## Verificación

`tests/integration/catalogo_transaction_test.go` (nombre heredado de la plantilla; prueba el runner): commit permitido, rollback por rechazo y por error técnico verificados con lectura independiente, y fallo de commit (restricción diferida) que nunca devuelve éxito.
