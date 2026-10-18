# ADR-001 — Monolito modular con transacciones locales

Estado: aceptado para este repositorio (2026-10-17). Instancia en código el ADR-01 del expediente (`docs/planificacion/01_documentacion/registro_adr.json`, estado allí: PROPUESTO).

## Contexto

Abrir un pedido, cobrar y confirmar movimientos de stock deben confirmarse o revertirse juntos. El SQL V4 reúne el negocio en un esquema. No hay necesidad medida de despliegues independientes por dominio.

## Decisión

- Un backend como **monolito modular**: un binario `api`, módulos con fronteras por paquete (`internal/modules/<m>`).
- Un caso de uso puede coordinar varios módulos dentro de una única transacción PostgreSQL.
- Capas por módulo: `domain`, `application`, `infrastructure` y `web`. **`web` (adaptador de entrada) se ubica fuera de `infrastructure`**, a diferencia de la plantilla, para que la entrada HTTP y los adaptadores de salida (persistencia, clientes externos) sean hermanos y se puedan reglamentar por separado.
- Los límites se hacen cumplir con `tests/architecture`, porque compilar sin ciclos no prueba la arquitectura ni impide que un módulo importe internos de otro (`internal` no lo evita).

## Alternativas descartadas

- Microservicios por dominio: coordinación distribuida y operación adicional sin necesidad actual.
- Monolito sin límites internos: favorece accesos cruzados y reglas duplicadas.

## Consecuencias

Menos componentes que desplegar y pruebas integradas simples. Exige disciplina de dependencias (la prueba de arquitectura la automatiza) y que un módulo no escriba en tablas de otro. Se reconsidera si una necesidad medida de escala, equipos o aislamiento supera el costo de coordinación.

## Verificación

`go test ./tests/architecture/...`; pruebas de integración de rollback (`tests/integration`). Aún no hay casos de uso multi-módulo que probar con fallos inyectados.
