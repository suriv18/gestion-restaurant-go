# Revisión e implementación de app-restaurant

Fecha: 17 de octubre de 2026 (America/Lima).

## Evidencia revisada

- ESTRUCTURA_FRONTEND_RESTAURANTE.md: arquitectura por funcionalidades, rutas diferidas, estado, formularios, contratos y pruebas.
- Requisitos y trazabilidad, registro maestro, backlog, decisiones pendientes y plan de pruebas de `docs/planificacion`.
- SQL V4 y catálogo estructural: **49 tablas**; maestros con baja lógica selectiva, mesas con actividad simple, documentos operativos por estado.
- OpenAPI de planificación: contrato **propuesto**, operaciones con `x-implementado: false`.
- Prototipo: 14 pantallas, datos, estilos, motor de reglas y pruebas. Se inspeccionó el salón en navegador antes de migrar.
- Backend Go: router real y mapa de módulos. El catálogo contiene archivos de plantilla; el router solo registra salud y disponibilidad.

## Decisiones aplicadas

| Tema | Implementación |
| --- | --- |
| Nombre | `app-restaurant`, solicitado por el usuario |
| Framework | Angular 21.2.25: última línea en LTS según la fuente oficial consultada; Angular 22 está en fase activa |
| UI | Identidad de Mesa, Tailwind 4, CSS de referencia y Angular CDK para diálogos |
| Arquitectura | Componentes standalone OnPush, Signals/computed, Reactive Forms, rutas diferidas por funcionalidad |
| Tipos | TypeScript y plantillas estrictos; proyecciones de UI separadas del futuro transporte HTTP |
| Persistencia inicial | Demostración explícita en localStorage; sin falsa autenticación o integración |
| Reglas locales | Motor del prototipo aislado, con contratos tipados y pruebas de comportamiento |
| Fuentes | DM Sans y Manrope locales; no dependen de Google Fonts en ejecución |
| Validación | Vitest, Angular ESLint, Prettier, build y Playwright desktop/tablet/mobile |

La decisión Angular 21/Tailwind sustituye las selecciones Angular 22/Material/SCSS del documento previo para esta entrega. No modifica retroactivamente el expediente ni declara resueltas las decisiones de negocio pendientes.

## Pantallas y entidades

| Pantallas | Tablas que representan |
| --- | --- |
| Salón y pedidos | zonas, mesas, pedidos, cuentas, detalles_pedido |
| Cocina | comandas, comanda_detalles, estaciones_preparacion |
| Reservas / delivery | reservas, pedidos_delivery, empleados |
| Caja | cajas, sesiones_caja, movimientos_caja, pagos, devoluciones_pago, metodos_pago |
| Compras / inventario | compras, compra_detalles, articulos_inventario, articulos_almacen, almacenes, movimientos_inventario, movimiento_inventario_detalles |
| Carta y recetas | productos, productos_sucursal, categorias, recetas, receta_detalles |
| Comprobantes | comprobantes, comprobante_detalles, series_comprobantes |
| Contactos / equipo | clientes, proveedores, usuarios, roles, usuarios_roles, usuarios_sucursales |
| Configuración / auditoría | negocio, sucursales, mesas, auditoria_eventos |
| Resumen | Agregados de la proyección local |

La UI no replica 49 CRUD independientes. Conserva las tareas y recorridos del prototipo. Los estados agregados de la demostración no se deben copiar directamente a columnas SQL: por ejemplo, el estado de preparación del consumo local combina información que en SQL pertenece a detalle/comanda.

## Invariantes conservadas

Una mesa admite un pedido abierto y respeta su capacidad. Precios y nombres históricos no cambian al editar la carta. El primer pago bloquea modificaciones de consumos. El pago no supera el saldo de su cuenta; el vuelto no aumenta el ingreso aplicado. Los cobros digitales no aumentan el efectivo esperado. Las devoluciones son compensaciones separadas.

La receta se consume al iniciar preparación, una sola vez y de forma atómica en la proyección local. Las salidas no pueden dejar stock negativo. Las compras se reciben una vez. Reservas activas no se superponen. El cierre exige pago completo y productos entregados.

Los maestros conservan identidad al darse de baja/restaurarse; las mesas se desactivan sin inventar marcadores de eliminación. Los documentos se cancelan por estado. Los comprobantes son explícitamente simulados.

## Integración posterior

1. Resolver autenticación/sesión (DEC-06), permisos y sucursal en el servidor.
2. Estabilizar IDs/decimales como cadenas, paginación y Problem Details del contrato.
3. Implementar adaptadores HTTP por funcionalidad y reemplazar la proyección local con resultados confirmados.
4. Añadir idempotencia y recuperación de resultado indeterminado en cobros; no reintentar escrituras financieras automáticamente.
5. Incorporar refresco/concurrencia de cocina y pedidos y validación multiusuario.
6. Completar inventario avanzado (lotes, vencimientos, conversiones), recetas versionadas y emisión real cuando existan contratos.
7. Ejecutar pruebas de integración contra PostgreSQL/API. Las pruebas de este frontend no certifican reglas transaccionales del backend.

No se incorporaron endpoints inventados, credenciales locales ni guards que aparenten seguridad de producción.
