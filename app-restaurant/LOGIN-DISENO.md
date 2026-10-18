# Acceso a Mesa

## Referencias consultadas

Revisión web del 17 de octubre de 2026:

- [Toast: acceso a sus productos](https://support.toasttab.com/en/article/Log-in-to-a-Toast-Product-or-Device). Referencia funcional de acceso por credenciales, recuperación y distinción entre cuenta web y PIN operativo.
- [Nielsen Norman Group: visibilidad de contraseñas](https://www.nngroup.com/articles/stop-password-masking/). Se incorpora un control explícito para revelar u ocultar el valor. Se mantiene oculto inicialmente, considerando dispositivos compartidos en restaurantes.
- [Square: acceso](https://app.squareup.com/login). Se consultó la página pública, pero su contenido requiere JavaScript y no quedó disponible en el lector web; no se usó como referencia visual verificada.

La composición es propia y conserva la identidad del prototipo local: verde bosque, papel cálido, detalles dorados, DM Sans y Manrope. La ilustración vectorial de una mesa puesta está incluida en la plantilla y no descarga imágenes externas.

## Comportamiento

`/` redirige a `/login`. La página pública carga por separado de la navegación de operación. Las 14 pantallas de operación **exigen sesión** (`authGuard`); sin ella se redirige a `/login?returnUrl=<ruta>` y, tras entrar, se vuelve a esa ruta (solo rutas internas: se descartan `//host`, `https://…` y similares). Con una sesión iniciada, `/login` redirige al resumen.

**Modo demostración.** `Explorar la demo` activa un modo explícito, recordado solo en la pestaña (`sessionStorage`, clave `app-restaurant.demo-mode`), para recorrer las pantallas con datos de ejemplo sin cuenta. No es una sesión ni una credencial. `Cerrar sesión` lo desactiva; iniciar sesión también.

**Autenticación real** (contrato del backend, ADR-004):

1. `POST /api/v1/auth/login` con `{ username, password }` devuelve un JWT (`access_token`). El campo es **usuario**, no correo: el servidor autentica por `username`.
2. Con el token se pide `GET /api/v1/sesion` (usuario, sucursales y permisos vigentes). La interfaz solo se considera autenticada cuando esa respuesta llega.
3. El servidor valida la sesión en cada petición: revocar, dar de baja al usuario o desactivar un rol surte efecto de inmediato. Un `401` del API a una llamada autenticada cierra la sesión local y lleva al login con el aviso «Tu sesión venció».
4. `Cerrar sesión` llama a `POST /api/v1/auth/logout` (revoca en el servidor) y limpia lo local aunque el servidor no responda.

**Dónde vive el token.** Solo `AuthSession` (`core/auth`) lo conoce. Está en memoria y se copia a `sessionStorage` únicamente para sobrevivir a una recarga de la pestaña; se borra al cerrar la pestaña, al cerrar sesión y ante cualquier `401`. Al arrancar se valida contra `/sesion`. No se guardan la contraseña ni datos del usuario. Contrapartida conocida: como cualquier token accesible por JavaScript, un XSS podría leerlo; por eso la sesión es revocable en el servidor y dura como máximo 12 h (2 h de inactividad). Una alternativa más robusta (cookie `HttpOnly`) requiere cambios coordinados en el backend.

El interceptor adjunta `Authorization: Bearer` solo a `/api/v1/…` (mismo origen), nunca a otros destinos ni al propio login.

**Errores.** El servidor responde igual ante usuario inexistente, contraseña errónea, cuenta inactiva o bloqueada (5 intentos fallidos bloquean 15 minutos); la interfaz tampoco los distingue: «Usuario o contraseña incorrectos…». Se diferencian la falta de conexión y el fallo del servicio. Tras un intento fallido se vacía la contraseña y el foco vuelve a ella. El botón se desactiva mientras se envía, para impedir envíos dobles.

Los campos incluyen etiquetas visibles, autocomplete, errores asociados mediante ARIA y foco en el primer campo inválido. El formulario funciona con Enter. Los diálogos de ayuda usan el foco contenido/restaurado de Angular CDK.

**Pendiente.** Recuperación de contraseña (no existe en el backend; el enlace «¿La olvidaste?» solo explica que debe pedirse ayuda a quien administra el restaurante), MFA y renovación de sesión.

## Adaptación y pruebas

- Escritorio: panel editorial ilustrado y formulario independiente.
- Tablet: dos columnas ajustadas al ancho disponible.
- Móvil: cabecera compacta y formulario en una columna, inputs de 16 px y controles principales de al menos 44 px.
- Ilustración decorativa excluida del árbol accesible; enlace para saltar al formulario y reducción de transiciones según preferencias del sistema.
- Unitarias: sesión (login, restauración, logout, vencimiento), interceptores, guards y destino seguro, formulario de login.
- Playwright con el backend simulado (deterministas): redirección sin sesión, login, destino, errores, cierre de sesión, recarga y revocación. Ver `VERIFICACION.md`.
- Integración con el API real: `e2e/login.real-api.spec.ts` (opt-in con `E2E_API_USER` y `E2E_API_PASSWORD`).
