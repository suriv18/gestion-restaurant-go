# Mesa · Prototipo de gestión de restaurante

Prototipo navegable en español basado en el modelo `sch_restaurante` de PostgreSQL, versión V3 de auditoría y baja lógica. Incluye 14 pantallas y datos ficticios. Es una aplicación estática, sin dependencias de ejecución ni conexión al servidor.

## Recorrido principal

1. En **Salón y pedidos**, elegir una mesa disponible y crear un pedido.
2. Agregar productos, cantidades e indicaciones. Enviar la comanda.
3. En **Cocina**, comenzar, terminar y servir cada producto. El inicio registra el consumo de la receta de ejemplo una sola vez.
4. En el pedido o **Caja y cobros**, cobrar una cuenta con efectivo, Yape, Plin, tarjeta o transferencia. Se permiten pagos parciales y mixtos.
5. Cerrar el pedido cuando esté completamente pagado y servido para liberar la mesa.

También se puede dividir un pedido por líneas antes del primer pago, recibir compras, registrar mermas o ajustes, consultar kardex, reservar mesas, despachar delivery, simular comprobantes y revisar auditoría.

## Correspondencia con la base de datos

| Pantalla | Tablas de referencia |
| --- | --- |
| Salón y pedidos | zonas, mesas, pedidos, cuentas, detalles_pedido |
| Cocina | comandas, comanda_detalles, estaciones_preparacion |
| Reservas | reservas, mesas |
| Delivery | pedidos_delivery, empleados |
| Caja y cobros | cajas, sesiones_caja, movimientos_caja, pagos, devoluciones_pago, metodos_pago |
| Compras | compras, compra_detalles, proveedores |
| Inventario | articulos_inventario, articulos_almacen, almacenes, movimientos_inventario, movimiento_inventario_detalles |
| Carta y recetas | productos, productos_sucursal, categorias, recetas, receta_detalles |
| Comprobantes | comprobantes, comprobante_detalles, series_comprobantes |
| Directorio | clientes, proveedores |
| Equipo y accesos | usuarios, roles, usuarios_roles, usuarios_sucursales |
| Configuración | negocio, sucursales, mesas |
| Auditoría | auditoria_eventos y campos de auditoría de los maestros |

El estado del navegador es una proyección simplificada de estas entidades. No replica físicamente todas las tablas ni todos sus campos.

## Reglas implementadas en la demostración

- Una mesa habilitada admite un solo pedido abierto; respeta su capacidad.
- Los consumos conservan nombre y precio históricos. Cambiar la carta no cambia ventas anteriores.
- La cuenta se puede dividir por líneas completas antes del primer pago. Después del primer cobro se bloquean cambios en los consumos.
- El cobro no supera el saldo. El vuelto se calcula a partir del efectivo recibido; el saldo y la caja solo reciben el importe aplicado.
- Los pagos digitales no suman al efectivo esperado. Las devoluciones son registros separados, sin modificar los pagos originales ni reabrir saldos automáticamente.
- La caja registra fondo, entradas, salidas, efectivo esperado, contado y diferencia por turno.
- Las reservas pendientes o confirmadas no admiten intervalos superpuestos en una misma mesa.
- El inventario se calcula a partir de movimientos. No hay un campo editable de stock ni se permiten salidas que dejen stock negativo.
- Una compra confirmada se recibe una vez. Las cantidades de compra usan la unidad base en esta versión del prototipo.
- La baja lógica de los maestros principales exige motivo y conserva identidad, código, usuario y fecha. Se puede restaurar. Las mesas solo cambian es_activo, sin motivo ni marcadores de eliminación; la auditoría registra el cambio. La disponibilidad de producto y la habilitación de mesa son controles independientes.
- Los documentos operativos se cancelan por estado; no se eliminan mediante `es_activo`.
- La auditoría visible es de solo lectura dentro de la aplicación. Su persistencia local no constituye un registro seguro o inmutable de producción.

## Alcance y límites

Datos guardados con `localStorage` únicamente en el navegador. Sesión fija de Administradora para la demostración. Las opciones de rol ilustran la configuración y no aplican autorización real. No se usan credenciales ni servicios externos.

No hay conexión PostgreSQL, API, autenticación, concurrencia multiusuario, impresora de comandas, integración bancaria o emisión a SUNAT. Los comprobantes son ejemplos sin validez tributaria y no calculan un desglose tributario legal. Las recetas son ejemplos de referencia, no fichas técnicas validadas. Los lotes, caducidades, conversiones de presentación, edición versionada de recetas, cuentas por pagar y administración completa de permisos quedan para la aplicación conectada al backend.

Las reservas abren pedidos al recibir clientes y los delivery se despachan al estar listos; no se integra geolocalización. La cancelación de un pedido conserva los consumos de inventario ya registrados. Las devoluciones de stock requieren un movimiento compensatorio.

La vista Resumen agrega los registros de la demostración. No pretende ser un informe contable ni un reporte tributario diario.

## Archivos y ejecución

- `dist/index.html`: documento de entrada.
- `dist/assets/styles.css`: diseño adaptable a escritorio, tablet y móvil.
- `dist/assets/data.js`: datos ficticios iniciales.
- `dist/assets/model.js`: reglas de negocio y operaciones atómicas sobre una copia de estado.
- `dist/assets/app.js`: pantallas, formularios, navegación y persistencia local.
- `servidor.mjs`: servidor local de demostración, solo en 127.0.0.1.

No hay proceso de compilación. El alojamiento sirve `dist/`. Para un entorno de desarrollo con servidor estático, usar su ruta raíz como documento de entrada; los módulos JavaScript requieren HTTP(S).

## Validación realizada

`npm test` ejecuta 13 escenarios de negocio con Node: atención completa, pagos mixtos y sobrepago, consumo único y atómico, recepción única, bajas/restauraciones, precios históricos, reservas, división de cuentas, devoluciones, cierres de caja, delivery, documentos simulados y desactivación/reactivación de mesa sin campos de eliminación.

`tests/ui.test.mjs` usa jsdom 30.1.2 para verificar las 14 rutas y los flujos principales mediante el DOM, incluidos mensajes de error, formularios y escape de contenido. Para ejecutarlo, instalar jsdom como dependencia local de desarrollo o indicar `MESA_TEST_PACKAGE` con la ruta al package.json del entorno de pruebas que lo contiene.

Pasaron la validación de sintaxis, los 13 escenarios y las pruebas de DOM. No se realizó inspección visual en un navegador real porque el control de navegador no estaba disponible en el entorno. Los estilos incluyen puntos de adaptación a 1550, 1200, 1000, 760 y 430 píxeles, foco visible, diálogo con gestión de foco, etiquetas accesibles y respeto a movimiento reducido.
