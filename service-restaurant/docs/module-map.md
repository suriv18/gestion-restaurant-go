# Mapa de módulos

## Módulos

Hoy existe un único módulo: `catalogo`, **esqueleto de ejemplo sustituible** de la plantilla (archivos con solo `package`). Los módulos de negocio previstos por el expediente (ADR-01) y su incremento son:

| Módulo (ADR-01) | Tablas (migración) | Incremento |
|---|---|---|
| Identidad | negocio, sucursales, roles, permisos, usuarios, empleados… (`00001`) | I1 |
| Catálogos | unidades, categorías, impuestos, productos, recetas, inventario maestro (`00002`) | I2 |
| Atención | zonas, mesas, reservas, pedidos, cuentas, detalles (`00003`) | I3 (reservas: I6) |
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

Ningún módulo importa internos de otro; la coordinación entre módulos ocurre en la composición (mismo `*sql.Tx`), no por importación.
