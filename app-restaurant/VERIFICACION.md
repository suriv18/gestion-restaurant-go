# Verificación local

Fecha: 22 de octubre de 2026, America/Lima. Windows, Node 24.14.1, npm 11.14.0.

| Comprobación | Resultado |
| --- | --- |
| Angular ESLint (TypeScript y plantillas accesibles) | Sin errores ni advertencias |
| Prettier | Todos los archivos comprobados cumplen formato |
| Vitest / Angular CLI | **194 pruebas aprobadas, 24 archivos spec** (incluye sesión, interceptores, guards, login, estado de Equipo y accesos, estado y secciones de Configuración y el diálogo asíncrono) |
| Build de producción | Aprobado; entrada 334.09 kB, transferencia estimada 89.15 kB, dentro del presupuesto de 500 kB |
| Playwright | **141 pruebas aprobadas**, 0 fallos; 15 omitidas (las del API real, que son opt-in) |
| Integración con el API real | `e2e/login.real-api.spec.ts` y `e2e/equipo.real-api.spec.ts` aprobados contra el stack de Docker: login y cierre que revoca el token; alta de una persona, duplicado rechazado con el mensaje del servidor, cambio de rol, baja (la persona recibe 401 al entrar), restauración, y 403 real para quien no administra |
| Integración de Configuración con el API real | `e2e/configuracion.real-api.spec.ts` escrita (sede con su aviso de éxito —sin esperar verla en el listado, porque el backend no se la asigna a quien la crea—, código repetido, zona, mesa, baja con motivo y restauración); **no se ha ejecutado contra el API real**: requiere `E2E_API_USER` y `E2E_API_PASSWORD`. Sin ellas queda omitida (verificado) |
| Rutas | Login y las 14 pantallas de operación cargan sin errores JavaScript y sin desbordamiento horizontal en los tres tamaños |

## Entornos E2E

| Proyecto | Viewport CSS | Motor |
| --- | --- | --- |
| desktop | 1440 × 1000 | Chromium |
| tablet | 820 × 1180 | Chromium con emulación táctil de iPad |
| mobile | 390 × 844 | Chromium con emulación de Pixel |

La emulación comprueba adaptación e interacción. No sustituye pruebas en hardware físico o motores Safari/Firefox.

## Recorridos verificados en cada entorno

1. Dividir varias líneas conservando total y saldos por cuenta.
2. Crear compra con varios insumos y consultar detalle.
3. Visitar las 14 rutas, comprobar errores y anchura, capturar salón/cocina/carta.
4. Abrir mesa → agregar plato → enviar comanda → preparar → servir → efectivo/Yape → cierre → recargar → comprobar inventario.
5. Rechazar sobrepago y conservar los campos para corregirlo.
6. Crear, filtrar, pausar, dar de baja y restaurar producto sin duplicar identidad.
7. Recibir una compra una vez y rechazar salida con stock negativo.
8. Rechazar reservas superpuestas y corregir el horario.
9. Mantener foco dentro del diálogo, cerrar con Escape y restaurar foco.
10. Navegar por menú, volver atrás y resolver ruta inexistente.
11. Abrir login desde la raíz, capturar la pantalla, entrar a la demo, salir y comprobar que las pantallas vuelven a exigir cuenta.
12. Validar campos y enfocar el error antes de llamar al servidor; iniciar sesión (token solo en sessionStorage, contraseña nunca almacenada); volver al destino pedido y descartar destinos externos; credenciales incorrectas sin sesión; falta de conexión y fallo del servicio; cerrar sesión (revoca en el servidor); recarga que conserva o descarta la sesión según el servidor.
13. Saltar al formulario con teclado, abrir ayuda/recuperación y restaurar el foco.
14. Abrir el menú de usuario con teclado, consultar Mi perfil, cerrar con Escape/clic exterior y restaurar el foco.
15. Persistir el tema, recorrer las 14 pantallas en oscuro, comprobar diálogos y seguir cambios de la apariencia del sistema.
16. Cerrar sesión desde el header, volver al login y conservar tanto el tema como los datos de la demo.
17. Configuración (con la API simulada): negocio editable con el RUC fijo; alta, edición, baja con motivo y restauración de sede, con el aviso de asignación; zonas (incluido el 409 de zona con mesas) y mesas con su interruptor, baja con motivo y restauración; solo lectura con `MESA_VER`; sesión sin permisos de configuración; modo demostración.

## Evidencia y reproducción

Informe HTML: `playwright-report/index.html` (abrir con `npm run test:e2e:report`).
Capturas del salón, cocina y carta: `test-results/restaurant-las-14-rutas-*/` por proyecto.
Capturas del login: `test-results/login-el-acceso-*/login.png` por proyecto.
Capturas del menú y de las páginas en oscuro: `test-results/user-menu-*/` por proyecto.
Las capturas se inspeccionaron visualmente. Los artefactos se regeneran en cada ejecución y no se versionan.

```sh
npm run lint
npm run format:check
npm run test:run
npm run build
npm run test:e2e
```

Vitest usa un máximo de 2 procesos para reducir el consumo de recursos. Las pruebas de tema cubren persistencia, cambios del sistema, valores inválidos, storage bloqueado y eliminación de listeners.

La suite valida el frontend y el motor local de demostración. No certifica autenticación real, integración HTTP, concurrencia, PostgreSQL, cobros bancarios o emisión tributaria.
