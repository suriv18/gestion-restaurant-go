# Plantilla backend Go — SoftPrime Solutions

Versión: 1.0.0 · Fuentes consultadas: 15 de octubre de 2026.

Plantilla documental de un monolito modular con Clean Architecture, DDD selectivo, puertos y adaptadores, CQRS lógico y Result. Leer junto con [la guía común](README_PLANTILLAS_BACKEND.md). El módulo `catalogo` es un ejemplo sustituible. El árbol especifica archivos que se implementarán; no es un repositorio ejecutable incluido en esta entrega.

## 1. Decisiones específicas de Go

| Área | Decisión |
|---|---|
| Runtime | Versión estable mantenida de Go, fijada en `go.mod` y en CI al instanciar el proyecto. |
| HTTP | `net/http`; router de la biblioteca estándar como base. |
| Persistencia | `database/sql` y controlador PostgreSQL, por ejemplo `pgx/v5/stdlib`. Usar una sola API de transacciones. |
| SQL | Consultas parametrizadas explícitas. Un generador como sqlc sería una decisión posterior, confinada a infraestructura. |
| Migraciones | Herramienta versionada, por ejemplo Goose, con una única secuencia de migraciones del despliegue. |
| Composición | Constructores y funciones explícitas; no contenedor de DI obligatorio. |
| Logs | `log/slog` con campos estructurados y correlación de solicitudes. |
| Pruebas | `testing`, `httptest`, pruebas de integración contra PostgreSQL. |
| Contratos | OpenAPI versionado; no generadores obligatorios para el núcleo. |

`com.softprimesolutions` identifica a la organización, pero no se usa literalmente como ruta de importación Go. En los ejemplos `example.com/organizacion/proyecto` es un marcador que se sustituye por la ruta real del módulo.

Go organiza cada directorio de código como un paquete. No equivale a crear clases Java dentro de carpetas. Evitar paquetes diminutos sin responsabilidad y ciclos entre `model`, `vo`, `policy` y `service`. `internal` restringe importaciones desde fuera del árbol autorizado; no impide por sí solo que dos módulos de negocio del mismo proyecto se acoplen.

## 2. Estructura del repositorio

```text
proyecto-backend-go/
  go.mod
  go.sum
  README.md
  Makefile
  Dockerfile
  compose.yaml
  .env.example
  .gitignore
  .github/workflows/ci.yml
  api/openapi.yaml
  docs/
    ADD.md
    module-map.md
    adr/
      001-monolito-modular.md
      002-cqrs-result-transacciones.md
      003-persistencia-y-versiones.md
  migrations/
    00001_catalogo_productos.sql
  cmd/
    api/main.go
  internal/
    bootstrap/
      app.go
      router.go
      lifecycle.go
    platform/
      config/config.go
      database/postgres.go
      transaction/runner.go
      security/authenticator.go
      observability/logger.go
      web/
        handler/endpoint.go
        exception/error_handler.go
        middleware/recover.go
        middleware/request_id.go
        middleware/authentication.go
        pagination/query_parser.go
        response/problem.go
    shared/
      kernel/
        result/result.go
        apperror/app_error.go
      application/
        pagination/page_request.go
        pagination/page_result.go
        pagination/cursor_result.go
        context/actor.go
    modules/
      catalogo/
        domain/
        application/
        infrastructure/
  tests/
    architecture/import_rules_test.go
    integration/catalogo_transaction_test.go
    integration/catalogo_pagination_test.go
    contract/productos_http_test.go
    fixtures/productos.sql
```

Los tests unitarios viven junto a los archivos que prueban, por ejemplo `producto_test.go`. La carpeta `tests` contiene pruebas que necesitan varias piezas o infraestructura real. `cmd/api/main.go` se limita a arrancar, atender señales y delegar en `bootstrap`; no registra manualmente todas las reglas del negocio.

## 3. Subestructura completa de un módulo

```text
internal/modules/catalogo/
  domain/
    model/
      producto.go
      estado_producto.go
      producto_test.go
    domainerror/
      invariant_error.go
    vo/
      producto_id.go
      codigo_producto.go
      dinero.go
    policy/
      activacion_producto.go
    service/
      calculo_precio.go
  application/
    dto/
      command/
        crear_producto.go
        cambiar_estado_producto.go
      query/
        obtener_producto.go
        listar_productos.go
      result/
        crear_producto.go
        producto_detalle.go
        producto_resumen.go
    port/
      inbound/
        crear_producto.go
        cambiar_estado_producto.go
        obtener_producto.go
        listar_productos.go
      outbound/
        producto_writer.go
        producto_reader.go
        precio_externo.go
    usecase/
      command/
        crear_producto.go
        cambiar_estado_producto.go
        crear_producto_test.go
      query/
        obtener_producto.go
        listar_productos.go
    mapper/
      producto_result.go
  infrastructure/
    client/
      precio/
        http_client.go
        dto.go
        mapper.go
        adapter.go
    config/
      module.go
      transactional_crear_producto.go
      transactional_cambiar_estado_producto.go
      consistent_listar_productos.go
    persistence/
      read/
        projection/producto_projection.go
        mapper/producto_read_mapper.go
        repository/producto_sql_reader.go
        adapter/producto_read_adapter.go
      write/
        entity/producto_row.go
        mapper/producto_write_mapper.go
        repository/producto_sql_writer.go
        adapter/producto_write_adapter.go
    web/
      dto/
        request/crear_producto_request.go
        request/listar_productos_request.go
        request/cambiar_estado_producto_request.go
        response/producto_response.go
        response/producto_page_response.go
      controller/
        producto_handler.go
        producto_routes.go
      mapper/
        producto_web_mapper.go
        catalogo_error_mapper.go
```

`domainerror` adapta la carpeta `exception`: Go no usa excepciones para sus errores ordinarios. `inbound` y `outbound` conservan el significado de `port/in` y `port/out`, con nombres de paquete claros.

La integración `client/precio` y `service/calculo_precio.go` son ejemplos de extensión. Omitirlos si el producto no necesita precios externos ni ese servicio de dominio. `entity/producto_row.go` describe una fila persistida; no lleva reglas del agregado ni se expone por HTTP.

## 4. Dirección de imports

| Paquete | Puede importar | No debe importar |
|---|---|---|
| `domain/vo`, `domain/domainerror` | Biblioteca estándar y tipos puros autorizados | Modelo, aplicación, SQL, HTTP. |
| `domain/model` | VO y errores del dominio | Policies que a su vez importen model. |
| `domain/policy`, `domain/service` | Model, VO y otros tipos puros necesarios | Infraestructura, contextos HTTP, repositorios. |
| `application` | Dominio, sus puertos/DTO y shared puro | `database/sql`, driver, `net/http`, infraestructura. |
| `infrastructure` | Contratos de aplicación, dominio para mapping y plataforma técnica | Internos de otro módulo. |
| `bootstrap` | Composición pública de módulos y plataforma | Reglas del negocio dentro del ensamblaje. |

No almacenar `context.Context` en agregados ni usarlo como contenedor oculto de `*sql.Tx`. Se pasa como primer parámetro en operaciones que necesitan cancelación/deadline. Un objeto puro de actor/ámbito transporta identidad validada cuando el caso lo necesita; un header sin verificar no es identidad confiable.

Los puertos se definen desde la necesidad del consumidor. Evitar interfaces con todos los métodos de una implementación. Interfaces de aplicación y adaptadores concretos se conectan en `infrastructure/config/module.go`.

## 5. Contratos de ejemplo y Result

Fragmento ilustrativo del puerto de entrada; las rutas y tipos se implementan al crear el proyecto:

```go
package inbound

import (
    "context"

    cmd "example.com/organizacion/proyecto/internal/modules/catalogo/application/dto/command"
    dtoresult "example.com/organizacion/proyecto/internal/modules/catalogo/application/dto/result"
    appresult "example.com/organizacion/proyecto/internal/shared/kernel/result"
)

type CrearProducto interface {
    Execute(context.Context, cmd.CrearProducto) (
        appresult.Result[dtoresult.CrearProducto], error,
    )
}
```

Reglas para `shared/kernel/result`:

- Variante `Ok[T]` construida mediante `Ok(value)` y variante fallida mediante `Fail[T](appError)`; campos internos no exportados.
- El valor cero de `Result[T]` se considera inválido. El ejecutor debe detectarlo y tratarlo como error de programación antes de commit. Los constructores crean los únicos valores válidos.
- Si el segundo valor `error` es no nulo, se ignora Result y se aborta la operación.
- Si `error == nil`, Result debe ser válido: éxito o rechazo esperado. El DTO no es el sobre genérico.
- Usar `errors.Is`/`errors.As` cuando se necesite reconocer causas y preservar contexto con wrapping. No decidir por comparación de mensajes.
- No usar `panic` para una validación o un rechazo del negocio. `recover` no reemplaza comprobar `error`.

Este doble canal se limita a los casos de uso para distinguir negocio y fallo técnico. Un repositorio SQL usa firmas Go ordinarias como `FindByID(ctx, id) (ProductoRow, error)`. Un resultado vacío esperado se define explícitamente en el contrato; no confundirlo con una caída de base de datos.

El dominio puede devolver un error tipado de invariante. El caso de uso convierte solo ese error conocido en un `AppError` esperado. Los errores inesperados continúan por el canal `error`.

## 6. Transacción sin filtrar `sql.Tx` al núcleo

`platform/transaction/runner.go` ofrece una función genérica técnica. `infrastructure/config/transactional_crear_producto.go` implementa el puerto de entrada y usa esa función. Una función genérica de paquete es apropiada; no definir métodos con parámetros de tipo propios, que Go no admite.

Secuencia propuesta:

1. `DB.BeginTx(ctx, options)` y rollback diferido como respaldo.
2. El decorador construye repositorios/adaptadores con **ese mismo `*sql.Tx`** y luego el caso de uso puro.
3. Ejecuta el caso de uso. Si hay error técnico, Result inválido o Result fallido, cierra con rollback.
4. Si Result es éxito, invoca `Commit()` y comprueba su error.
5. Solo después devuelve éxito al handler.

El runner genérico recibe una función de trabajo y un predicado puro que indica si el resultado permite commit; así `platform` no conoce `catalogo`. Los repositorios reciben una interfaz técnica mínima `DBTX` con métodos de consulta/ejecución, implementada por `*sql.DB` y `*sql.Tx`; esa interfaz permanece en infraestructura/plataforma, nunca en aplicación.

Dentro de un comando no usar el `*sql.DB` de fuera para consultas que deben pertenecer a la transacción. Para un proceso entre módulos, el composition root construye los participantes con el mismo `Tx`; no se anidan varios `BeginTx` pretendiendo una transacción atómica.

Si falla el rollback, conservar el error principal y registrar la falla de cierre. Si el commit devuelve un error de comunicación, no afirmar que el cliente puede repetir cualquier comando sin riesgo: contemplar una clave de idempotencia cuando el proceso lo necesite.

## 7. HTTP y equivalente de GlobalExceptionHandler

| Archivo | Responsabilidad |
|---|---|
| `internal/platform/web/handler/endpoint.go` | Adaptar endpoints que retornan error a `http.Handler`. |
| `internal/platform/web/exception/error_handler.go` | Mapear errores técnicos conocidos y producir Problem Details seguro. |
| `internal/platform/web/middleware/recover.go` | Capturar panics inesperados en la goroutine de la solicitud y registrar la causa. |
| `.../web/mapper/catalogo_error_mapper.go` | Mapear Result fallido y códigos del módulo a HTTP. |

Go no necesita una clase llamada `GlobalExceptionHandler`. Estos componentes cumplen sus responsabilidades. El handler no escribe headers ni cuerpo antes de finalizar la operación que podría fallar. Una respuesta parcialmente enviada no se puede reemplazar limpiamente por un problema JSON.

`recover` no captura automáticamente panics de otras goroutines. Evitar trabajo de solicitud que siga corriendo sin ownership tras la respuesta; las tareas durables requieren un diseño específico.

Validar tamaño del cuerpo, JSON y parámetros. Configurar timeouts del servidor y propagación del contexto. Los errores de negocio no se envían con HTTP 200. El contrato de códigos y `application/problem+json` es el de la guía común.

## 8. Lecturas y paginación

`ListarProductos` recibe un DTO con filtros y `PageRequest`. `ProductoReader` devuelve `PageResult[ProductoResumen]`; el repositorio genera SQL paginado y la proyección permanece dentro del adaptador. No hay un `[]Producto` sin límites para un listado público.

`page=0`, `size=25`, máximo 100 por respuesta; orden estable como `nombre, id`. Parámetros SQL para valores y allowlist para columnas de orden. Recorrer filas con `rows.Next()`, cerrar `rows` y comprobar `rows.Err()`.

Si conteo y datos deben coincidir dentro de la respuesta, `consistent_listar_productos.go` crea una transacción corta de lectura con aislamiento acordado y conecta ambos queries al mismo `Tx`. El caso de uso de lectura sigue sin importar SQL.

Para cursor, agregar al módulo DTO y puerto específicos, construir el cursor con el último elemento entregado y consultar `size+1`. La exportación completa tiene su propio caso de uso y política de consistencia; no equivale a mostrar una página gigante.

## 9. Modelado, auditoría y extensiones

`EstadoProducto` representa el negocio; `es_activo` tiene una semántica de habilitación/visibilidad definida por el módulo. Los campos de eliminación se incorporan solo cuando exista ese requisito. No crear un struct embebido universal con todos los campos de auditoría y borrado.

Los VO usan constructores validados y campos no exportados cuando sea necesario preservar invariantes. Para importes, adoptar una representación exacta documentada —unidades menores enteras o decimal— con moneda y redondeo explícitos; no usar `float64` indiscriminadamente.

Un cliente externo se inyecta mediante un puerto pequeño. Añadir timeouts y retries solo para operaciones que puedan reintentarse sin duplicar efectos. Las closures sirven para composición y transacciones; una regla compleja vive en una función con nombre.

## 10. Verificación y arranque de una instancia

Comandos orientativos, ejecutables después de implementar los archivos y fijar dependencias:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/api
go run ./cmd/api
```

Configurar el runner para que pruebas de integración y migraciones dispongan de PostgreSQL. No confundir `go test` exitoso sin esos servicios con una prueba de persistencia real.

Pruebas imprescindibles del proyecto resultante:

- Invariantes y transiciones del agregado sin servidor HTTP.
- Caso de uso con puertos fake, incluido Result fallido.
- Escritura seguida de rechazo: rollback verificado mediante una lectura independiente.
- Error de commit/driver: jamás devuelve éxito.
- Paginación con más de una página y orden duplicado.
- Verificación de imports entre capas y módulos; compilación sin ciclos no prueba toda la arquitectura.

`main` usa apagado ordenado del servidor con un deadline y cierra recursos. `.env.example` solo contiene nombres/valores ilustrativos, no credenciales reales. Las migraciones se ejecutan como un paso controlado; no permitir que cada réplica cree versiones divergentes.

## 11. Fuentes y recursos reales

| Recurso | Uso |
|---|---|
| [Go: organización de módulos](https://go.dev/doc/modules/layout) | Convenciones de paquetes, `cmd` e `internal`. |
| [Go: transacciones](https://go.dev/doc/database/execute-transactions) | Ciclo de `sql.Tx` y uso consistente de sus métodos. |
| [Repositorio oficial golang/example](https://github.com/golang/example) | Ejemplos reales para estudiar código Go; no es una plantilla DDD completa. |
| [Biblioteca net/http](https://pkg.go.dev/net/http) | API del servidor, handlers y timeouts. |
| [pgx: adaptador database/sql](https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib) | Driver propuesto para PostgreSQL. |
| [Goose](https://github.com/pressly/goose) | Herramienta de migraciones propuesta; fijar versión al adoptarla. |

El detalle de capas, contratos Result y decorators es una decisión de esta plantilla. No es una exigencia del lenguaje Go ni de los repositorios citados.
