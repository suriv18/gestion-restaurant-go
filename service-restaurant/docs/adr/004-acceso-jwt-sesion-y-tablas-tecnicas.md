# ADR-004 — Acceso: JWT con sesión en BD y tablas técnicas

Estado: aceptado para este repositorio (2026-10-17). Aporta a DEC-06 (identidad y sesiones) y a ADR-03/ADR-04 del expediente.

## Contexto

I1 necesita autenticar usuarios humanos, revocar accesos al instante (RES-RF-033: "un rol inactivo deja de conceder permisos") y autorizar por permiso y sede (RES-RF-002). El contrato OpenAPI declara `bearerFormat: JWT`. Se evaluó primero un token opaco; el equipo pidió JWT por seguridad e interoperabilidad con el contrato.

## Decisión

- **JWT HS256 + sesión en BD (híbrido).** El bearer es un JWT firmado con `sid` (id de sesión), `sub` (usuario), `iss`, `aud`, `iat`, `exp` (= vencimiento absoluto, 12 h). El verificador fija el algoritmo (rechaza `none` y otros), exige `exp`, `iss` y `aud`. En cada petición `AutenticarToken` además busca la sesión `sid`: debe estar vigente (no revocada, dentro de la inactividad de 2 h y del plazo absoluto), pertenecer al `sub` y ser de un usuario activo y no técnico; luego carga los permisos efectivos desde `v_permisos_efectivos`.
- **Por qué no JWT sin estado:** no se podría revocar antes de `exp`, ni reflejar una baja de usuario o un rol desactivado hasta que venza; incumpliría RES-RF-033.
- **Costo:** 2 consultas por petición; aceptable en un monolito que ya consulta la BD. El uso de la sesión se registra como máximo una vez por minuto.
- **Clave:** `JWT_SECRET` (>= 32 bytes) obligatoria; la API no arranca sin ella. Sin rotación (`kid`) por ahora; HS256 es suficiente mientras solo este monolito verifica tokens (si otro servicio debiera verificar, pasar a EdDSA/RS256 con clave pública).
- **Contraseñas:** argon2id en formato PHC (parámetros dentro del hash, por lo que subir el costo no invalida hashes previos). El rechazo de login es siempre idéntico (usuario inexistente, inactivo, técnico, bloqueado o contraseña errónea) y gasta el mismo tiempo de hash. 5 fallos bloquean 15 min.
- **Un rechazo de login se confirma.** Registrar el fallo (`intentos_fallidos`) debe sobrevivir al 401; el decorador transaccional de login hace commit en ese caso y rollback en cualquier otro rechazo o error.

## Tablas técnicas (`00007_sesiones_idempotencia`)

`sesiones` e `idempotencia` **no llevan** los triggers de metadatos ni de auditoría de la V4 (desviación deliberada de la convención): `sesiones.ultimo_uso_en` cambia durante el uso y la auditoría V4 generaría un evento por petición; `idempotencia` es un registro efimero (24 h). La trazabilidad de accesos queda en los eventos de `usuarios` (intentos, último acceso) que sí se auditan, sin hashes de contraseña.

## Consecuencias

El flujo de acceso es revocable al instante y compatible con el contrato JWT. Cada módulo futuro obtiene un `Actor` con permisos por sede y no vuelve a tocar la autenticación. Pendiente: refresh, MFA, recuperación de contraseña, rate limiting por IP, rotación de claves.

## Verificación

`tests/integration/identidad_http_test.go` (PostgreSQL 18): login, sesión, logout con revocación inmediata, baja de usuario, sede retirada, inactividad, bloqueo persistente (verificado por mutación: la prueba falla si el rechazo se revierte), rechazos indistinguibles, validación del cuerpo y auditoría sin secretos. Unitarias: política de bloqueo y sesión, argon2id, JWT manipulados (`none`, HS512, otra clave, otro emisor/audiencia, sin `exp`).
