# app-restaurant

Frontend de Mesa basado en las 14 pantallas de `docs/planificacion/04_prototipo`.
Angular **21.2.25 LTS**, CLI 21.2.27, TypeScript estricto, Tailwind CSS 4.3.3 y Angular CDK.
Aplicación standalone, zoneless, con rutas diferidas, Signals, Reactive Forms y fuentes alojadas localmente.

## Ejecutar

Requiere Node compatible (validado con 24.14.1) y npm. En PowerShell con scripts restringidos usar `npm.cmd` / `npx.cmd`.

```sh
cd app-restaurant
npm ci
npm start
```

Abrir http://localhost:4200. La entrada es **Iniciar sesión** (`/login`). Seleccionar **Explorar la demo** abre Salón y pedidos con datos de ejemplo y sin credenciales.

### Conexión con el backend

El front llama siempre a `/api/v1/...` (mismo origen) y el servidor de desarrollo lo reenvía al backend mediante `proxy.conf.mjs`; así no hace falta CORS. Levante el backend (`service-restaurant`: `docker compose up -d --build` y la instalación inicial del administrador, ver su README) y reinicie `npm start` si ya estaba corriendo.

```sh
# Backend en otro puerto (por defecto el proxy apunta a http://localhost:18080):
API_TARGET=http://localhost:8080 npm start
```

En producción sirva el front y el API bajo el mismo origen (proxy inverso que enrute `/api` al backend). Para producción, `npm run build` genera `dist/app-restaurant/browser`; configurar el hosting para resolver las rutas SPA a `index.html`.

## Alcance actual

Implementación Angular nativa: no usa iframe, inyección de HTML ni el controlador DOM del prototipo.

- **Login real** contra el backend (`POST /api/v1/auth/login`, JWT con sesión en servidor): validación, estados de carga y errores (credenciales, conexión, servicio), control de visibilidad de contraseña y destino tras entrar. Las pantallas de operación **exigen sesión**; «Explorar la demo» es un modo explícito y separado. La recuperación de contraseña sigue sin existir (pendiente en DEC-06). Ver `LOGIN-DISENO.md`.
- **Equipo y accesos conectado al backend** (`features/identidad`): con sesión real lista a las personas con sus roles y sedes, y permite crear una persona (alta + rol + sede), cambiar su rol, darla de baja con motivo (cierra sus sesiones) y restaurarla. Los errores del servidor (duplicado, último administrador, sede no autorizada…) aparecen dentro del diálogo, que conserva lo escrito; cada formulario usa una `Idempotency-Key` propia, así un reintento no duplica. La pantalla solo ofrece acciones a quien tiene `USUARIO_ADMINISTRAR` (el servidor lo vuelve a comprobar). En modo demostración sigue mostrando los datos de ejemplo.
- **Configuración conectada al backend** (`features/configuracion`): con sesión real edita los datos del negocio (el RUC no se modifica), gestiona sedes (alta, edición, baja con motivo y restauración) y las zonas y mesas de cada sede (número, zona, capacidad, posición opcional, interruptor de habilitación, baja con motivo y restauración). Cada sección se muestra según los permisos `NEGOCIO_VER`/`NEGOCIO_ADMINISTRAR`, `SUCURSAL_VER`/`SUCURSAL_ADMINISTRAR` y `MESA_VER`/`MESA_ADMINISTRAR`: sin permiso de ver la sección no aparece, y sin el de administrar queda en solo lectura; el servidor vuelve a comprobarlo. Los errores del servidor aparecen en el campo o en el diálogo, que conserva lo escrito, y cada formulario usa su propia `Idempotency-Key`. Patrón: `configuracion-page` es un contenedor que elige entre `configuracion-real` (con el `ConfiguracionStore`, provisto en la página) y el HTML de demostración. En modo demostración sigue mostrando los datos de ejemplo.
- Menú de usuario en el header: Mi perfil, Tema (Claro/Oscuro/Sistema) y Cerrar sesión. Preferencia local en `app-restaurant.theme`, compartida por páginas, login y diálogos. Cerrar sesión vuelve al login conservando los datos de la demo.
- Resumen, salón/pedidos, cocina, reservas y delivery.
- Caja, pagos parciales/mixtos, vuelto, devolución y cierre de turno.
- Compras, recepción, inventario por movimientos y kardex.
- Carta, recetas de referencia, disponibilidad, bajas con motivo y restauración.
- Comprobantes de ejemplo, contactos, roles de ejemplo, auditoría CSV y configuración.
- Foco contenido y restaurado en diálogos, Escape, navegación por teclado, menú móvil, estados vacíos y mensajes de error.

**Modo demostración**, identificado permanentemente en la interfaz. Persistencia local en `app-restaurant.demo.v1`, independiente del prototipo. Los datos iniciales son ficticios. No es una aplicación multiusuario ni offline transaccional.

El backend implementa hoy el acceso (`/auth/login`, `/auth/logout`, `/sesion`) y la administración de roles, permisos y usuarios (consumida por «Equipo y accesos») y la configuración del negocio, las sedes, las zonas y las mesas (consumida por «Configuración»); **el resto de las pantallas, incluidos el Salón de pedidos, el plano y el resumen, sigue usando la proyección local de demostración hasta I3** (aún no hay endpoints de pedidos, caja, carta, etc.). Autenticación real ya integrada; permisos por pantalla, sucursales múltiples, concurrencia, impresión operativa, integración bancaria y SUNAT requieren los endpoints y decisiones pendientes del expediente. Los documentos mostrados no tienen validez tributaria.

## Estructura

```text
src/app/
  app.*                    # Raíz y composición de rutas
  shell.*                  # Navegación de las 14 áreas de operación
  core/auth/               # Sesión (token en memoria + sessionStorage), API de acceso, interceptores, guards y modo demo
  core/theme/              # Preferencia persistida y seguimiento del tema del sistema
  core/demo/
    restaurant-store.ts    # Signals y persistencia del modo local
    engine/                # Reglas y semilla del prototipo, con frontera tipada
  features/
    atencion/ produccion/ caja/ abastecimiento/
    catalogos/ identidad/ comprobantes/ consulta/ configuracion/
  shared/
    models/                # Proyecciones de UI y payloads tipados
    ui/                    # Diálogos, campos, cabeceras, importes, iconos
e2e/
  restaurant.spec.ts
  login.spec.ts
  user-menu.spec.ts
  equipo.spec.ts configuracion.spec.ts   # con la API simulada
  *.real-api.spec.ts                    # opt-in, contra el backend real
```

El motor de demostración conserva el JavaScript de reglas del prototipo, formateado y aislado, con declaraciones TypeScript explícitas. No es el dominio de producción. Todas las páginas y stores Angular están en TypeScript estricto. Las clases CSS de referencia mantienen el diseño de Mesa; Tailwind se usa para composición y responsive, y CDK para foco/overlay.

Los IDs numéricos y los céntimos enteros pertenecen exclusivamente a la proyección local. El OpenAPI propuesto usa IDs y decimales como cadenas. Al integrar, crear adaptadores `data-access` por funcionalidad y reemplazar los comandos locales por resultados confirmados por la API. No enviar metadatos de auditoría del navegador como autoridad.

## Verificación

```sh
npm run lint
npm run format:check
npm run test:run
npm run build
npx playwright install chromium
npm run test:e2e
npm run test:e2e:report
```

En Linux CI: `npx playwright install --with-deps chromium`.

- `*.spec.ts`: reglas de operación, atomicidad, pagos, persistencia, validadores, formulario y shell (Vitest del Angular CLI).
- E2E: login, errores, visibilidad de contraseña, ayuda, foco, entrada/salida de demo; rutas, recorrido mesa → cocina → pago mixto → cierre, sobrepago, catálogo/baja/restauración, compras/stock, reservas, foco y navegación.
- Proyectos Playwright: **desktop 1440×1000**, **tablet 820×1180**, **mobile 390×844**, Chromium con emulación táctil en tablet/móvil.
- Cada prueba tiene un contexto aislado; no depende de un backend o datos reales.
- Capturas de login, salón, cocina y carta por dispositivo en `test-results`; trazas y video conservados al fallar. Informe HTML en `playwright-report`.
- La emulación no equivale a validación en dispositivos físicos o Safari/WebKit.

Pipeline: `.github/workflows/app-restaurant-ci.yml`. La revisión y las decisiones de migración están en `docs/estructura/REVISION_FRONTEND_IMPLEMENTACION.md`.

Resultado local: **194 pruebas unitarias/de componentes (24 archivos) y 141 E2E aprobados** (15 omitidas: las del API real, opt-in), además de lint, formato y build. Ver [VERIFICACION.md](VERIFICACION.md) para alcance y evidencia.

## Versiones

Se eligió Angular 21 porque el pedido exige la última línea **en fase LTS**, aunque el documento previo propone Angular 22 en soporte activo. Se conserva la estructura funcional y se sustituye Material/SCSS por Tailwind/CSS + CDK según lo solicitado.

Fuentes consultadas el 17 de octubre de 2026: [soporte Angular](https://angular.dev/reference/releases), [compatibilidad](https://angular.dev/reference/versions), [Tailwind con Angular](https://tailwindcss.com/docs/installation/framework-guides/angular).
