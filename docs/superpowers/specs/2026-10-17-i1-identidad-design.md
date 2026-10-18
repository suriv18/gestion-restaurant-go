# Diseño I1 — Identidad: acceso, autorización y administración de accesos

Fecha: 2026-10-17 · Estado: aprobado por secciones en conversación; pendiente de revisión del documento.
Requisitos: RES-RF-001 (autenticar), RES-RF-002 (autorizar por permiso y sede), RES-RF-033 (roles y asignaciones). Decisiones: DEC-06 (identidad y sesiones), ADR-03, ADR-04. Código: `service-restaurant/`.

## 1. Decisiones

| Tema | Decisión |
|---|---|
| Token (DEC-06) | Token opaco aleatorio de 32 bytes (base64url); en BD solo su hash SHA-256. Sin JWT ni refresh (el contrato `Token` solo trae `access_token`). |
| Primer administrador | Binario `cmd/admin`, ejecutado una vez por el instalador. Sin endpoint de alta pública. |
| Alcance | Todo I1: RF-001, RF-002 y RF-033. |
| Idempotencia (ADR-04) | Componente genérico dentro de I1: lo exigen las escrituras de RF-033 y lo reusarán I3/I4. |
| Contraseñas | argon2id, formato PHC (cumple `length(password_hash) >= 20`). Parámetros bajos solo en pruebas. |
| Expiración (ajustable) | 12 h absolutas y 2 h de inactividad. |
| Bloqueo | 5 intentos fallidos → `bloqueado_hasta = ahora + 15 min`. |

## 2. Módulo y datos

Módulo nuevo `internal/modules/identidad` con `domain`, `application`, `infrastructure` y `web` (web fuera de `infrastructure`). Dos áreas: acceso (login, logout, sesión, autorización) y administración de accesos (usuarios, roles, permisos, asignaciones). `catalogo` sigue siendo el esqueleto de ejemplo.

Migración nueva `00007_sesiones_idempotencia` con dos tablas **técnicas**, sin triggers de metadatos ni de auditoría (desviación deliberada de la convención V4: actualizarían `ultimo_uso` en cada petición y generarían un evento de auditoría por petición; se registrará en un ADR):

- `sesiones`: id, `usuario_id`, `token_hash` (único), `creado_en`, `ultimo_uso_en` (se actualiza como máximo una vez por minuto), `expira_inactividad_en`, `expira_absoluta_en`, `revocada_en`.
- `idempotencia`: `clave` (UUID), `usuario_id`, `operacion`, `huella` (SHA-256 del contenido), `estado`, `respuesta_status`, `respuesta_cuerpo`, `creado_en`, `expira_en` (24 h). Único por (`clave`, `usuario_id`, `operacion`).

Permisos efectivos: vista `v_permisos_efectivos` de la V4 (todos los eslabones activos: usuario, rol, permiso, sucursal, negocio). No se agregan permisos: `USUARIO_ADMINISTRAR` protege todas las rutas de administración.

## 3. Flujos

**Login (`POST /auth/login`).** Normaliza `lower(btrim(username))`. Si el usuario no existe, calcula igualmente un hash contra uno ficticio (tiempos similares). Responde 401 con cuerpo idéntico si: contraseña incorrecta, usuario inactivo, `__sistema__` o bloqueado. Un fallo real suma `intentos_fallidos` (actor `__sistema__`, porque aún no hay usuario autenticado); al quinto fija `bloqueado_hasta`. Éxito: `intentos_fallidos = 0`, `ultimo_acceso_en` (actor: el usuario), crea la sesión y devuelve `{access_token, token_type: "Bearer", expires_in}`. `POST /auth/logout` (ruta nueva, a agregar al OpenAPI) revoca la sesión actual.

**Autenticación por petición.** Middleware: `Authorization: Bearer` → hash → sesión vigente, no revocada, usuario activo y no sistema → permisos efectivos → `Actor` con mapa sede → permisos; `actor.Can(permiso, sedeId)` es en memoria. 2 consultas por petición; revocar o dar de baja surte efecto inmediato.

**Autorización.**
- Rutas sin sede propia (roles, permisos, usuarios): `USUARIO_ADMINISTRAR` en al menos una sede.
- Asignaciones: un administrador solo asigna sedes donde él tiene `USUARIO_ADMINISTRAR`.
- Recursos de una sede (módulos futuros): el caso de uso comprueba `Can(permiso, sedeDelRecurso)`; sede ajena → 404 para no revelar su existencia.

**Actor en escrituras.** `platform/transaction` ofrece una opción para fijar `app.usuario_id` (`set_config(..., true)`) al abrir la transacción con el actor autenticado (decisión 12 de la V4), válida para todos los módulos.

**Idempotencia.** Dentro de la misma transacción del comando: `INSERT … ON CONFLICT DO NOTHING`; misma huella y completada → se devuelve la respuesta guardada; huella distinta → 409; en curso por otra transacción → espera con `SELECT … FOR UPDATE` y aplica lo anterior. Como el registro comparte transacción con el efecto, un rechazo (422) o un fallo hace rollback y no se guarda nada. Se autoriza antes de revelar un resultado previo; la clave está acotada al usuario.

**Administración (RF-033).** Alta, edición, baja y restauración con auditoría por los triggers V4. La baja de maestros exige motivo; la restauración exige confirmación. `PUT /usuarios/{id}/asignaciones` reemplaza el conjunto de roles y sedes reactivando asignaciones existentes (regla 16: no duplicar). El alta recibe la contraseña en texto y guarda solo el hash; nada de texto plano en logs ni auditoría. El actor técnico no recibe roles y los roles de sistema no se tocan: lo imponen triggers V4 y el backend traduce sus errores a 409/422.

## 4. API

Base `/api/v1`.

| Ruta | Acceso |
|---|---|
| `POST /auth/login` | pública |
| `POST /auth/logout` (nueva) | autenticada |
| `GET /sesion` | autenticada |
| `GET/POST /roles`, `GET/PATCH /roles/{id}`, `POST /roles/{id}/baja`, `POST /roles/{id}/restauracion` | `USUARIO_ADMINISTRAR` |
| `GET/POST /permisos`, `GET/PATCH /permisos/{id}`, `POST /permisos/{id}/baja`, `POST /permisos/{id}/restauracion` | `USUARIO_ADMINISTRAR` |
| `POST /usuarios`, `PUT /usuarios/{id}/asignaciones`, `POST /usuarios/{id}/baja`, `POST /usuarios/{id}/restauracion` | `USUARIO_ADMINISTRAR` |

Los `POST` de creación, baja y restauración exigen `Idempotency-Key` (UUID); sin ella → 400. Listados con `limit` (máx. 100) y `cursor`, orden estable por id. Errores en `application/problem+json`: 401 (credenciales/token inválidos, expirados o revocados; cuerpo siempre idéntico), 403 (falta permiso), 404 (inexistente o de otra sede), 409 (conflicto, incluida clave de idempotencia reutilizada con otro contenido), 422 (regla).

Hueco del contrato anotado, no resuelto aquí: no existe `GET /usuarios` en el OpenAPI y RF-033 no lo pide.

`cmd/admin`: crea el primer usuario con rol ADMIN y una sede; lee la contraseña de stdin o de una variable de entorno; rechaza contraseñas débiles; se niega a correr si ya existe un usuario humano. Se incluye en la imagen Docker.

## 5. Pruebas

- Unitarias: hash/verificación de contraseñas, generación y hash de tokens, `Actor.Can`, huella de idempotencia, política de bloqueo.
- Integración (PostgreSQL 18): login correcto/incorrecto, usuario inactivo, `__sistema__`, bloqueo tras 5 intentos; sesión expirada y revocada; rol inactivo deja de conceder permisos; sede ajena → 404; idempotencia (repetir igual, repetir distinto, simultáneas, reintento tras fallo); alta/baja/restauración de usuario con actor y auditoría y sin secretos en `auditoria_eventos`.
- Arquitectura: `tests/architecture` cubre el módulo nuevo.

## 6. Entrega

Tres pasos, cada uno con CI en verde:
1. Núcleo de acceso: migración 00007, contraseñas, sesiones, login, sesión, middleware, `cmd/admin`.
2. Idempotencia genérica y opción de actor en `transaction`.
3. Administración (RF-033).

## 7. Riesgos y pendientes

- La migración 00007 se aparta de la convención de auditoría V4 (ADR a escribir).
- Argon2id con parámetros altos ralentiza los tests: parámetros bajos en pruebas, valor sensato por defecto en producción.
- DEC-06 se cierra con este diseño una vez implementado; DEC-02 gana la prueba con autenticación que pide ADR-06, pero siguen faltando la comparación de alternativas y la confirmación de un responsable técnico.
- Fuera de alcance: rol de BD de operación separado del propietario del esquema (ADR-07), recuperación de contraseña, MFA, rate limiting por IP.
