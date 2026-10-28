# Mapa de módulos

## Módulos

Módulos hoy: **`identidad`** (implementado en I1: acceso, autorización y administración de accesos; en I2, negocio y sucursales), **`atencion`** (zonas y mesas, implementado en I2; pedidos, cuentas y consumos, implementado en I3 rebanada 1) y **`catalogo`** (implementado en I2: clientes y proveedores en la rebanada 2; categorías, impuestos de solo lectura, tasas de impuesto, estaciones de preparación, productos y su disponibilidad por sede en la rebanada 3). En `catalogo` sigue presente, sin usarse, el **esqueleto de ejemplo** de la plantilla: 51 archivos de una sola línea `package x` con nombres `*producto*` (p. ej. `crear_producto.go`, `producto_handler.go`, `infrastructure/client/precio`) que conviven con las implementaciones reales (`productos.go`, `productos_handler.go`) y deben borrarse en una limpieza aparte. Los empleados viven en `identidad`. Los módulos de negocio previstos por el expediente (ADR-01) y su incremento son:

| Módulo (ADR-01) | Tablas (migración) | Incremento |
|---|---|---|
| Identidad | negocio y sucursales (administración en I2), roles, permisos, usuarios, empleados… (`00001`) | I1 / I2 |
| Catálogos | categorías, impuestos, tasas, estaciones, productos y disponibilidad por sede (`00002`, implementados en I2); unidades, recetas e inventario maestro quedan para I5 | I2 / I5 |
| Atención | zonas y mesas (`00003`, I2) · pedidos, cuentas, detalles y datos de reparto (`00003`, I3 rebanada 1; permisos en `00011`) · reservas (I6) | I2 / I3 (reservas: I6) |
| Producción | comandas, comanda_detalles, delivery (`00003`) | I3 / I6 |
| Caja | métodos de pago, cajas, sesiones, movimientos, pagos, devoluciones (`00004`) | I4 |
| Abastecimiento | compras, movimientos de inventario, kardex (`00005`) | I5 |
| Comprobantes | series, comprobantes, detalles, envíos (`00006`) | I7 |
| Consulta | vistas y reportes por sede | I8 |

La asignación módulo ↔ tabla es una propuesta; se confirma al implementar cada incremento. `usuarios` es referenciada por casi todas las tablas (`creado_por`/`actualizado_por`), por lo que Identidad se migra primero.

## Capas de un módulo

```text
internal/modules/<m>/
  domain/          model, vo, domainerror, policy, service
  application/     dto (command, query, result), port (inbound, outbound), usecase, mapper
  infrastructure/  persistence (read, write), client, config (composición del módulo)
  web/             controller, dto (request, response), mapper   ← adaptador de entrada
```

## Dirección de imports (verificada por `tests/architecture`)

| Paquete | Puede importar | No puede importar |
|---|---|---|
| `domain/vo`, `domain/domainerror` | solo biblioteca estándar | todo lo interno |
| `domain/model` | `vo`, `domainerror`, `shared/kernel` | `policy`, `service` |
| `domain/policy`, `domain/service` | `model`, `vo`, `domainerror`, `shared/kernel` | aplicación, infraestructura, web |
| dominio y `application` | (ver arriba / dominio propio, `shared`) | `database/sql`, `net/http`, drivers, `infrastructure`, `web`, `platform` |
| `infrastructure` | aplicación, dominio, `platform`, `shared` | otro módulo; `web` salvo `infrastructure/config` |
| `web` | puertos y DTO de `application`, `shared`, `platform/web` | dominio, `infrastructure` |
| `shared` | solo lo propio y estándar | módulos, `platform`, `bootstrap`, SQL/HTTP |
| `platform` | `shared` | módulos, `bootstrap` |
| `bootstrap` | `platform`, y de un módulo solo `infrastructure/config` | reglas de negocio |
| `cmd` | `bootstrap`, `platform` | módulos |

`tests/architecture` cubre también `atencion`, que no importa `identidad`. Ningún módulo importa internos de otro; la coordinación entre módulos ocurre en la composición (mismo `*sql.Tx`), no por importación.
