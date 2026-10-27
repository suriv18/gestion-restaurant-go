# Diseño I2 (rebanada 4) — Cierre de I2 con auditoría de bajas y formularios

Requisitos: RES-RF-030 (bajas y restauraciones) y RES-RF-036 (formularios y errores), transversales a I2, más el cierre documental del incremento. Las rebanadas 1 a 3 (backend y frontend) están fusionadas en `main` (PR #1 a #6). Esta rebanada no añade funcionalidad de negocio: verifica lo construido contra los criterios de aceptación, fija el resultado con pruebas y corrige los huecos que aparezcan.

## 1. Alcance
- **Entidades de I2 auditadas.**
  - Maestros con baja y motivo (7 de los 10 del expediente): sucursales, usuarios, empleados, clientes, proveedores, productos y mesas. Artículos de inventario, almacenes y cajas llegan en I4/I5.
  - Desactivación simple (sin motivo): categorías, estaciones de preparación, zonas, disponibilidad por sede (`productos_sucursal`) y las de I1 que tengan acción en pantalla (roles, asignación de sedes).
  - Sin baja: negocio (singleton, solo edición) y tasas de impuesto (vigencias; se editan, no se dan de baja). Se anotan como «no aplica» en la matriz.
- **Fuera:** maestros y pantallas de I3–I5, el rediseño de los diálogos y un componente genérico de baja (YAGNI: el patrón `Dialogs.form` + `newIdempotencyKey()` ya es común).

## 2. Decisiones
**D1. Estado de RF-030 y RF-036: `PARCIAL`, no `IMPLEMENTADO`.** Faltan tres maestros y varias de las 14 pantallas, que caen en I3–I5. Cada requisito lleva escrita su situación (qué entidades cumplen y qué falta). Los casos RES-CP-030 y RES-CP-036 siguen `NO_EJECUTADA`: no se ejecutan como tales en esta rebanada.

**D2. La auditoría produce una matriz, y la matriz se apoya en pruebas automatizadas, no en inspección manual.** Una constatación sin prueba que la fije no cuenta como verificada.

## 3. Matriz de auditoría
Filas: las entidades del §1. Columnas (criterios de aceptación de RF-030 y RF-036):

| Grupo | Criterio |
|---|---|
| Backend, baja | Los maestros exigen motivo y guardan motivo, usuario y fecha de baja; las desactivaciones simples no los piden ni los guardan |
| Backend, baja | El actor sale de la sesión, nunca del cuerpo (los campos de auditoría enviados se ignoran o se rechazan) |
| Backend, baja | Baja y restauración exigen `Idempotency-Key`; sin ella, 400; con la misma clave, el mismo resultado sin duplicar |
| Backend, restauración | Reactivar conserva el id y limpia el motivo |
| Backend, borrado | `DELETE` no está permitido (404 o 405, nunca 2xx) |
| Backend, historial | La auditoría conserva actor y cambios de cada baja y restauración |
| Frontend | La baja de un maestro pide motivo; la desactivación simple pide solo confirmación |
| Frontend | El doble clic en confirmar no duplica el comando (una `Idempotency-Key` por intento) |
| Frontend | Un error de campo se muestra en el campo, conserva lo escrito y lleva el foco |
| Frontend | Las acciones dependen del permiso y del estado (activo/de baja) |
| Frontend | Un 403 en una sección muestra el aviso de permiso y no rompe la pantalla |
| Frontend | Las bajas no se ocultan: hay filtro o interruptor para verlas y restaurarlas |

Cada celda se marca «cumple» (con el nombre de la prueba que la fija), «hueco» (con su arreglo) o «no aplica» (con la razón). La matriz se entrega en `docs/superpowers/specs/` junto a este diseño, como `2026-10-26-i2-rebanada4-matriz-bajas-formularios.md`.

## 4. Pruebas que fijan el resultado
- **Go.** Una prueba parametrizada por entidad para los criterios de backend. Se aprovecha la infraestructura de pruebas de integración del repositorio; no se duplica lo que ya cubre una prueba existente: la celda apunta a ella.
- **Frontend unitario.** Un spec por tipo de baja (con motivo y simple) que recorre las secciones reales con el store simulado: confirmación/motivo, una sola llamada con doble clic, error de campo y botones por permiso y estado.
- **E2E (simulado).** Se añade el doble clic solo donde la matriz muestre que falta cobertura; el resto ya está en los specs de cada pantalla.
- **API real.** Las pruebas opt-in existentes se reutilizan; no se amplían salvo que un hueco de backend lo exija. Lo que no se ejecute contra el API real se declara.

## 5. Huecos y correcciones
Los huecos se listan en la matriz **antes** de corregir nada. Cada uno se corrige con el cambio mínimo (backend, frontend o contrato) y su prueba. Si un hueco exige una decisión de producto (p. ej. añadir motivo a una entidad que hoy no lo tiene), se detiene y se pregunta; no se decide en silencio. Un hueco que dependa de I3–I5 se anota como pendiente de ese incremento.

## 6. Cierre documental
- `backlog_incrementos.json`: el «avance» de I2 recoge las cuatro rebanadas (hoy dice que falta el frontend de la carta). El estado del incremento lo decide el usuario al cerrar la rebanada; este diseño no lo cambia por su cuenta.
- `requisitos_y_trazabilidad.json` y `.html`: RF-030 y RF-036 pasan a `PARCIAL` con su situación; RF-005 y RF-027 conservan la suya.
- `openapi.yaml`: `x-implementado: true` solo en las operaciones realmente implementadas y probadas (clientes, proveedores, empleados y lo que la auditoría confirme). Las de módulos futuros no se tocan.
- `CAMBIOS_1_3.txt`, `MANIFIESTO.json` y `SHA256SUMS.txt` se actualizan y los checksums se recalculan y verifican, como en las rebanadas anteriores.
- `service-restaurant/docs/ADD.md` y `module-map.md` solo si un cambio de backend los afecta.

## 7. Entrega y verificación
- Una rama `i2-rebanada4-cierre-auditoria`, commits pequeños (matriz y pruebas, correcciones de huecos, expediente) y un PR contra `main`.
- Antes de afirmar que algo está hecho: `lint`, `prettier`, pruebas unitarias del frontend, `go vet` y `go test` (con integración si hay Postgres disponible), E2E simulados, y la verificación de checksums del expediente. Lo no ejecutado se declara.

## 8. Riesgos
- **Alcance elástico.** Una auditoría puede crecer sin fin. Se acota a las entidades del §1 y a los criterios del §3; lo demás se anota como pendiente de I3–I5.
- **Pruebas de integración de Go.** Necesitan Postgres; si no está disponible en el entorno, se declara y se apoya en el stack de Docker temporal, como en la rebanada 3.
- **Tentación de declarar «cumple» sin prueba.** Mitigado por D2.
