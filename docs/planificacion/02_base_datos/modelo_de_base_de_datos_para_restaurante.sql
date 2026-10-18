-- ============================================================================
-- RESTAURANTE MYPE / PYME - MODELO COMPLETO V4: AUDITORIA SELECTIVA Y ESTADOS
-- Motor objetivo: PostgreSQL 18. Fecha: 2026-10-12.
-- Instalacion NUEVA, no migracion del script original. No contiene DROP ni datos
-- ficticios del negocio. Si sch_restaurante existe, la instalacion se detiene.
-- Ejecutar TODO el archivo con un usuario propietario del esquema:
--   psql -v ON_ERROR_STOP=1 -d nombre_base -f modelo_de_base_de_datos_para_restaurante.sql
-- Tambien puede ejecutarse como script completo desde DBeaver / pgAdmin.
-- No crea base de datos, usuarios PostgreSQL ni contrasenas predeterminadas.
-- Crea un actor tecnico __sistema__ sin password ni roles, exclusivamente para
-- instalacion/procesos internos; no es una cuenta de acceso humano.
--
-- ALCANCE: una empresa/RUC por base, varias sucursales, salones y almacenes;
-- carta, recetas versionadas, reservas sin solapamientos, pedidos, cuentas,
-- cocina, delivery, pagos mixtos, devoluciones, caja, compras, kardex,
-- comprobantes, configuracion tributaria y auditoria.
-- No implementa planillas, contabilidad completa, combos configurables,
-- integracion SUNAT/FCM/pasarelas ni sincronizacion offline.
--
-- DECISIONES:
-- 1. Los precios de venta INCLUYEN impuestos; descuento por linea en moneda.
-- 2. Un detalle pertenece a una cuenta; un pedido puede tener varias cuentas.
--    Para dividir cantidades de un plato, el servicio divide sus lineas antes
--    del primer cobro y conserva su trazabilidad de cocina.
-- 3. Los adicionales y el envio cobrable son lineas propias del pedido.
-- 4. El stock se obtiene del kardex CONFIRMADO; no hay stock_actual editable.
-- 5. No hay descuento automatico de recetas al cobrar. El servicio genera el
--    consumo una sola vez al preparar/despachar, con clave de idempotencia.
-- 6. Corregir stock confirmado requiere otro movimiento, nunca borrar historia.
-- 7. Tasas tributarias SIN semillas numericas: configurar segun vigencia y
--    condicion del contribuyente. Cada venta conserva la tasa utilizada.
-- 8. Cabeceras operativas usan vistas para totales; comprobantes guardan sus
--    propias lineas historicas. No calcular ventas antiguas con la carta actual.
-- 9. Funciones son SECURITY INVOKER. El backend debe usar un rol de BD con
--    permisos limitados, transacciones y autorizacion por operacion.
-- 10. Para auditoria: SELECT set_config('app.usuario_id','123',true) dentro
--     de cada transaccion autenticada. No usar sesiones reutilizadas sin reset.
-- 11. Cobros, devoluciones y stock exigen READ COMMITTED (predeterminado en PG).
--     Las funciones rechazan otro aislamiento; el backend debe respetarlo.
-- 12. creado_por/actualizado_por son FK NOT NULL a usuarios; los completa un
--     trigger desde app.usuario_id. Es obligatorio establecer actor en cada
--     transaccion. No se toma el actor que envie el cliente HTTP.
-- 13. es_activo controla la actividad administrativa en 26 tablas. SOLO 9
--     maestros principales tienen eliminado_en, eliminado_por y motivo_eliminacion.
--     En las otras 17, desactivar/reactivar solo cambia es_activo; el actor y
--     el historial quedan en actualizado_por/en y auditoria_eventos.
--     estado expresa el proceso de negocio en 13 tablas; no equivale a una baja.
-- 14. Documentos operativos NO tienen es_activo: se anulan por estado, y los
--     saldos solo cambian por operaciones compensatorias. Nunca filtrar kardex,
--     caja o historicos comerciales por es_activo del maestro relacionado.
-- 15. En 46 tablas editables: creado_en/por y actualizado_en/por. En las 3
--     inmutables (auditoria_eventos, movimientos_caja, devoluciones_pago):
--     solo creado_en/por. No se agregan campos de actualizacion sin significado.
-- 16. La baja no libera RUC, documento, username, codigo ni serie: restaurar
--     la misma identidad. Tampoco duplica roles/asignaciones al reactivarlos.
--
-- FUENTES DE DISENO (consultadas 2026-10-12):
-- https://www.postgresql.org/docs/18/ddl-constraints.html
-- https://www.postgresql.org/docs/18/rangetypes.html
-- https://www.odoo.com/documentation/19.0/applications/sales/point_of_sale/restaurant.html
-- https://www.odoo.com/documentation/19.0/applications/sales/point_of_sale/use.html
-- https://cpe.sunat.gob.pe/tipos_de_comprobantes/boleta
-- https://cpe.sunat.gob.pe/tipos_de_comprobantes/nota_de_credito
-- https://orientacion.sunat.gob.pe/reduccion-del-igv-para-restaurantes-hoteles-y-alojamiento-turisticos-1
-- ============================================================================
BEGIN;
CREATE SCHEMA sch_restaurante;
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';

-- 01. NEGOCIO, SEGURIDAD Y PERSONAS

CREATE TABLE negocio (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    singleton BOOLEAN NOT NULL DEFAULT TRUE UNIQUE CHECK (singleton),
    ruc VARCHAR(11) NOT NULL UNIQUE CHECK (ruc ~ '^[0-9]{11}$'),
    razon_social VARCHAR(250) NOT NULL CHECK (btrim(razon_social) <> ''),
    nombre_comercial VARCHAR(150) NOT NULL,
    direccion_fiscal TEXT NOT NULL,
    ubigeo VARCHAR(6) CHECK (ubigeo ~ '^[0-9]{6}$'),
    telefono VARCHAR(30), email VARCHAR(254),
    moneda CHAR(3) NOT NULL DEFAULT 'PEN' CHECK (moneda ~ '^[A-Z]{3}$'),
    zona_horaria VARCHAR(60) NOT NULL DEFAULT 'America/Lima',
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE negocio IS 'Una unica empresa por base. La configuracion tributaria es explicita.';

CREATE TABLE sucursales (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    negocio_id BIGINT NOT NULL REFERENCES negocio(id),
    codigo VARCHAR(20) NOT NULL UNIQUE,
    nombre VARCHAR(120) NOT NULL,
    direccion TEXT NOT NULL,
    ubigeo VARCHAR(6) CHECK (ubigeo ~ '^[0-9]{6}$'),
    codigo_establecimiento_sunat VARCHAR(10),
    telefono VARCHAR(30), es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE sucursales IS 'Locales operativos de la misma empresa.';

CREATE TABLE roles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo VARCHAR(40) NOT NULL UNIQUE,
    nombre VARCHAR(80) NOT NULL, descripcion TEXT,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE roles IS 'Roles de aplicacion; no son roles PostgreSQL.';

CREATE TABLE permisos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo VARCHAR(80) NOT NULL UNIQUE,
    descripcion VARCHAR(250) NOT NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    es_activo BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON TABLE permisos IS 'Permisos funcionales que debe verificar el backend.';

CREATE TABLE roles_permisos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rol_id BIGINT NOT NULL REFERENCES roles(id),
    permiso_id BIGINT NOT NULL REFERENCES permisos(id),
    UNIQUE (rol_id, permiso_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    es_activo BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON TABLE roles_permisos IS 'Asignacion de permisos a roles.';

CREATE TABLE usuarios (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username VARCHAR(80) NOT NULL CHECK (btrim(username) <> ''),
    password_hash TEXT,
    es_sistema BOOLEAN NOT NULL DEFAULT FALSE,
    CHECK ((es_sistema AND password_hash IS NULL)
        OR (NOT es_sistema AND password_hash IS NOT NULL AND length(password_hash) >= 20)),
    email VARCHAR(254), es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    ultimo_acceso_en TIMESTAMPTZ,
    bloqueado_hasta TIMESTAMPTZ,
    intentos_fallidos INTEGER NOT NULL DEFAULT 0 CHECK (intentos_fallidos >= 0),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE usuarios IS 'Credenciales con hash robusto generado por el backend; nunca texto plano.';

CREATE UNIQUE INDEX uq_usuarios_username ON usuarios (lower(btrim(username)));

CREATE TABLE usuarios_roles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    usuario_id BIGINT NOT NULL REFERENCES usuarios(id),
    rol_id BIGINT NOT NULL REFERENCES roles(id),
    UNIQUE (usuario_id, rol_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    es_activo BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON TABLE usuarios_roles IS 'Un usuario puede desempeñar varios roles.';

CREATE TABLE usuarios_sucursales (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    usuario_id BIGINT NOT NULL REFERENCES usuarios(id),
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    UNIQUE (usuario_id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    es_activo BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON TABLE usuarios_sucursales IS 'Ambito de acceso por sucursal; debe validarlo la aplicacion.';

CREATE TABLE empleados (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    usuario_id BIGINT UNIQUE REFERENCES usuarios(id),
    tipo_documento VARCHAR(10) NOT NULL,
    numero_documento VARCHAR(20) NOT NULL,
    nombres VARCHAR(120) NOT NULL, apellidos VARCHAR(120) NOT NULL,
    telefono VARCHAR(30), cargo VARCHAR(80),
    fecha_ingreso DATE NOT NULL DEFAULT CURRENT_DATE,
    fecha_cese DATE, es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (tipo_documento, numero_documento),
    CHECK (fecha_cese IS NULL OR fecha_cese >= fecha_ingreso),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE empleados IS 'Personal operativo; no reemplaza un modulo de planillas.';

CREATE TABLE clientes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tipo_documento VARCHAR(10), numero_documento VARCHAR(20),
    nombre_razon_social VARCHAR(250) NOT NULL,
    telefono VARCHAR(30), email VARCHAR(254), direccion TEXT,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK ((tipo_documento IS NULL) = (numero_documento IS NULL)),
    UNIQUE (tipo_documento, numero_documento),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE clientes IS 'Cliente opcional del pedido. El comprobante conserva una copia historica.';

CREATE TABLE proveedores (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tipo_documento VARCHAR(10) NOT NULL,
    numero_documento VARCHAR(20) NOT NULL,
    razon_social VARCHAR(250) NOT NULL,
    contacto_nombre VARCHAR(150), telefono VARCHAR(30), email VARCHAR(254),
    direccion TEXT, es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (tipo_documento, numero_documento),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE proveedores IS 'Proveedores de insumos y articulos de reventa.';

-- 02. CATALOGOS, CARTA, RECETAS E INVENTARIO MAESTRO

CREATE TABLE unidades_medida (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo VARCHAR(15) NOT NULL UNIQUE,
    nombre VARCHAR(60) NOT NULL,
    dimension VARCHAR(15) NOT NULL CHECK (dimension IN ('MASA','VOLUMEN','UNIDAD')),
    factor_referencia NUMERIC(18,9) NOT NULL CHECK (factor_referencia > 0),
    codigo_sunat VARCHAR(10), es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE unidades_medida IS 'Referencias: kg, litro y unidad. Conversion solo dentro de la misma dimension.';

CREATE TABLE categorias (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL UNIQUE,
    descripcion TEXT, orden_visualizacion INTEGER NOT NULL DEFAULT 0,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE categorias IS 'Categorias de la carta.';

CREATE TABLE impuestos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo VARCHAR(30) NOT NULL UNIQUE,
    nombre VARCHAR(120) NOT NULL,
    descripcion TEXT, es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE impuestos IS 'Tributo aplicable a ventas. IGV_TOTAL representa la tasa combinada IGV/IPM.';

CREATE TABLE tasas_impuesto (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    impuesto_id BIGINT NOT NULL REFERENCES impuestos(id),
    tasa NUMERIC(7,4) NOT NULL CHECK (tasa >= 0 AND tasa <= 100),
    vigente_desde DATE NOT NULL, vigente_hasta DATE,
    fundamento TEXT,
    CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde),
    EXCLUDE USING gist (int8range(impuesto_id, impuesto_id, '[]') WITH &&,
        daterange(vigente_desde, vigente_hasta, '[)') WITH &&),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE tasas_impuesto IS 'Vigencias [desde,hasta), sin solapamientos. No se insertan tasas por defecto.';

CREATE TABLE estaciones_preparacion (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    codigo VARCHAR(20) NOT NULL, nombre VARCHAR(80) NOT NULL,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (sucursal_id, codigo), UNIQUE (id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE estaciones_preparacion IS 'Cocina, barra u otras estaciones de preparacion.';

CREATE TABLE articulos_inventario (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo VARCHAR(40) NOT NULL UNIQUE,
    nombre VARCHAR(180) NOT NULL,
    tipo VARCHAR(20) NOT NULL CHECK (tipo IN ('INSUMO','REVENTA','ENVASE','SEMIELABORADO')),
    unidad_base_id BIGINT NOT NULL REFERENCES unidades_medida(id),
    controla_lote BOOLEAN NOT NULL DEFAULT FALSE,
    controla_vencimiento BOOLEAN NOT NULL DEFAULT FALSE,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (NOT controla_vencimiento OR controla_lote),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE articulos_inventario IS 'Sustituye ingredientes e incluye bebidas, envases y semielaborados; sin saldo editable.';

CREATE TABLE presentaciones_articulo (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    articulo_id BIGINT NOT NULL REFERENCES articulos_inventario(id),
    nombre VARCHAR(100) NOT NULL,
    factor_unidad_base NUMERIC(18,6) NOT NULL CHECK (factor_unidad_base > 0),
    codigo_barras VARCHAR(50), es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (articulo_id, nombre), UNIQUE (id, articulo_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE presentaciones_articulo IS 'Ejemplo: saco 50 kg o caja 12 unidades. El factor es especifico del articulo.';

CREATE TABLE almacenes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    codigo VARCHAR(20) NOT NULL, nombre VARCHAR(100) NOT NULL,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (sucursal_id, codigo), UNIQUE (id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE almacenes IS 'Almacenes fisicos por sucursal.';

CREATE TABLE articulos_almacen (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    almacen_id BIGINT NOT NULL REFERENCES almacenes(id),
    articulo_id BIGINT NOT NULL REFERENCES articulos_inventario(id),
    stock_minimo NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (stock_minimo >= 0),
    stock_maximo NUMERIC(18,6), es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (almacen_id, articulo_id),
    CHECK (stock_maximo IS NULL OR stock_maximo >= stock_minimo),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE articulos_almacen IS 'Configuracion y fila de bloqueo para serializar movimientos del articulo/almacen.';

CREATE TABLE lotes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    articulo_id BIGINT NOT NULL REFERENCES articulos_inventario(id),
    codigo VARCHAR(80) NOT NULL,
    fecha_elaboracion DATE, fecha_vencimiento DATE,
    estado VARCHAR(15) NOT NULL DEFAULT 'LIBERADO' CHECK (estado IN ('LIBERADO','BLOQUEADO','RETIRADO')),
    motivo_estado TEXT,
    CHECK (estado = 'LIBERADO' OR (motivo_estado IS NOT NULL AND btrim(motivo_estado) <> '')),
    UNIQUE (articulo_id, codigo), UNIQUE (id, articulo_id),
    CHECK (fecha_elaboracion IS NULL OR fecha_vencimiento IS NULL
           OR fecha_vencimiento >= fecha_elaboracion),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE lotes IS 'Identidad del lote. Estado de calidad independiente del saldo; no admite baja que oculte existencias.';

CREATE TABLE productos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    categoria_id BIGINT NOT NULL REFERENCES categorias(id),
    codigo VARCHAR(40) NOT NULL UNIQUE,
    nombre VARCHAR(180) NOT NULL, descripcion TEXT,
    tipo VARCHAR(20) NOT NULL CHECK (tipo IN ('PREPARADO','REVENTA','ADICIONAL','SERVICIO')),
    articulo_id BIGINT REFERENCES articulos_inventario(id),
    factor_consumo NUMERIC(18,6) NOT NULL DEFAULT 1 CHECK (factor_consumo > 0),
    precio_venta NUMERIC(14,2) NOT NULL CHECK (precio_venta >= 0),
    afectacion_igv CHAR(2) NOT NULL CHECK (afectacion_igv IN ('10','20','30')),
    impuesto_id BIGINT REFERENCES impuestos(id),
    imagen_url TEXT, es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK ((tipo = 'REVENTA') = (articulo_id IS NOT NULL)),
    CHECK (afectacion_igv <> '10' OR impuesto_id IS NOT NULL),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE productos IS 'Carta: precios finales con impuesto. 10 gravado, 20 exonerado, 30 inafecto; no incluye operaciones gratuitas especiales.';

CREATE TABLE productos_sucursal (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    producto_id BIGINT NOT NULL REFERENCES productos(id),
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    estacion_id BIGINT,
    precio_venta NUMERIC(14,2) CHECK (precio_venta >= 0),
    disponible BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (producto_id, sucursal_id),
    FOREIGN KEY (estacion_id, sucursal_id) REFERENCES estaciones_preparacion(id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    es_activo BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON TABLE productos_sucursal IS 'Disponibilidad, precio opcional y estacion por sucursal; NULL usa precio general.';

CREATE TABLE recetas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    producto_id BIGINT NOT NULL REFERENCES productos(id),
    version INTEGER NOT NULL CHECK (version > 0),
    nombre VARCHAR(150) NOT NULL,
    rendimiento_porciones NUMERIC(14,3) NOT NULL CHECK (rendimiento_porciones > 0),
    instrucciones TEXT, vigente_desde DATE NOT NULL,
    vigente_hasta DATE,
    UNIQUE (producto_id, version), UNIQUE (id, producto_id),
    CHECK (vigente_hasta IS NULL OR vigente_hasta > vigente_desde),
    EXCLUDE USING gist (int8range(producto_id, producto_id, '[]') WITH &&,
        daterange(vigente_desde, vigente_hasta, '[)') WITH &&),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE recetas IS 'Cabecera versionada. Para cambiar composicion utilizada, crear otra version.';

CREATE TABLE receta_detalles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    receta_id BIGINT NOT NULL REFERENCES recetas(id),
    articulo_id BIGINT NOT NULL REFERENCES articulos_inventario(id),
    cantidad_base NUMERIC(18,6) NOT NULL CHECK (cantidad_base > 0),
    notas TEXT,
    UNIQUE (receta_id, articulo_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE receta_detalles IS 'Cantidad bruta requerida para todo el rendimiento, en unidad base; incluir merma tecnica prevista.';

-- 03. SALON, RESERVAS, PEDIDOS, CUENTAS Y COCINA

CREATE TABLE zonas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    nombre VARCHAR(80) NOT NULL,
    orden_visualizacion INTEGER NOT NULL DEFAULT 0,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (sucursal_id, nombre), UNIQUE (id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE zonas IS 'Salones, terrazas o niveles del local.';

CREATE TABLE mesas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id), zona_id BIGINT NOT NULL,
    numero_mesa VARCHAR(20) NOT NULL,
    capacidad INTEGER NOT NULL CHECK (capacidad > 0),
    habilitada BOOLEAN NOT NULL DEFAULT TRUE,
    posicion_x NUMERIC(10,2), posicion_y NUMERIC(10,2),
    UNIQUE (sucursal_id, numero_mesa), UNIQUE (id, sucursal_id),
    FOREIGN KEY (zona_id, sucursal_id) REFERENCES zonas(id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    es_activo BOOLEAN NOT NULL DEFAULT TRUE
);

COMMENT ON TABLE mesas IS 'La ocupacion se deriva de pedidos abiertos; reservas dependen del intervalo, no de un estado permanente.';

CREATE TABLE reservas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id), mesa_id BIGINT NOT NULL,
    cliente_id BIGINT REFERENCES clientes(id),
    nombre_contacto VARCHAR(150) NOT NULL, telefono_contacto VARCHAR(30),
    inicio TIMESTAMPTZ NOT NULL, fin TIMESTAMPTZ NOT NULL,
    cantidad_personas INTEGER NOT NULL CHECK (cantidad_personas > 0),
    estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE'
        CHECK (estado IN ('PENDIENTE','CONFIRMADA','ATENDIDA','CANCELADA','NO_ASISTIO')),
    observaciones TEXT, creado_por BIGINT NOT NULL,
    FOREIGN KEY (mesa_id, sucursal_id) REFERENCES mesas(id, sucursal_id),
    UNIQUE (id, sucursal_id), CHECK (fin > inicio),
    EXCLUDE USING gist (int8range(mesa_id, mesa_id, '[]') WITH &&,
        tstzrange(inicio, fin, '[)') WITH &&)
        WHERE (estado IN ('PENDIENTE','CONFIRMADA')),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE reservas IS 'Reserva de una mesa, sin solapamientos activos. Para grupos con varias mesas ampliar asignaciones.';

CREATE TABLE pedidos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    codigo_publico UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    clave_idempotencia UUID NOT NULL UNIQUE,
    mesa_id BIGINT, reserva_id BIGINT,
    cliente_id BIGINT REFERENCES clientes(id),
    mesero_id BIGINT REFERENCES empleados(id),
    tipo VARCHAR(15) NOT NULL CHECK (tipo IN ('SALON','LLEVAR','DELIVERY')),
    estado VARCHAR(20) NOT NULL DEFAULT 'REGISTRADO'
        CHECK (estado IN ('REGISTRADO','PREPARACION','LISTO','ENTREGADO','CANCELADO')),
    moneda CHAR(3) NOT NULL DEFAULT 'PEN' CHECK (moneda ~ '^[A-Z]{3}$'),
    comensales INTEGER NOT NULL DEFAULT 1 CHECK (comensales > 0),
    abierto_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cerrado_en TIMESTAMPTZ,
    notas TEXT, motivo_cancelacion TEXT,
    creado_por BIGINT NOT NULL,
    version_bloqueo INTEGER NOT NULL DEFAULT 0 CHECK (version_bloqueo >= 0),
    UNIQUE (id, sucursal_id),
    FOREIGN KEY (mesa_id, sucursal_id) REFERENCES mesas(id, sucursal_id),
    FOREIGN KEY (reserva_id, sucursal_id) REFERENCES reservas(id, sucursal_id),
    CHECK ((tipo = 'SALON') = (mesa_id IS NOT NULL)),
    CHECK (cerrado_en IS NULL OR cerrado_en >= abierto_en),
    CHECK (estado <> 'CANCELADO' OR (cerrado_en IS NOT NULL AND btrim(motivo_cancelacion) <> '' AND motivo_cancelacion IS NOT NULL)),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE pedidos IS 'Un pedido activo por mesa. Atencion y cobro son dimensiones independientes; cobro se consulta en vistas.';

CREATE UNIQUE INDEX uq_pedido_mesa_abierta ON pedidos(mesa_id) WHERE mesa_id IS NOT NULL AND cerrado_en IS NULL;

CREATE TABLE cuentas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pedido_id BIGINT NOT NULL REFERENCES pedidos(id),
    numero INTEGER NOT NULL CHECK (numero > 0),
    cliente_id BIGINT REFERENCES clientes(id),
    etiqueta VARCHAR(80),
    estado VARCHAR(15) NOT NULL DEFAULT 'ABIERTA' CHECK (estado IN ('ABIERTA','CERRADA','CANCELADA')),
    cerrado_en TIMESTAMPTZ, motivo_cancelacion TEXT,
    creado_por BIGINT NOT NULL,
    UNIQUE (pedido_id, numero), UNIQUE (id, pedido_id),
    CHECK (estado <> 'CERRADA' OR cerrado_en IS NOT NULL),
    CHECK (estado <> 'CANCELADA' OR (motivo_cancelacion IS NOT NULL AND btrim(motivo_cancelacion) <> '')),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE cuentas IS 'Subcuentas por pedido para separar consumos y comprobantes; cada una admite multiples pagos.';

CREATE TABLE detalles_pedido (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pedido_id BIGINT NOT NULL REFERENCES pedidos(id), cuenta_id BIGINT NOT NULL,
    producto_id BIGINT NOT NULL REFERENCES productos(id),
    receta_id BIGINT, detalle_padre_id BIGINT,
    descripcion VARCHAR(250) NOT NULL,
    cantidad NUMERIC(14,3) NOT NULL CHECK (cantidad > 0),
    precio_unitario NUMERIC(14,6) NOT NULL CHECK (precio_unitario >= 0),
    descuento_total NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (descuento_total >= 0),
    afectacion_igv CHAR(2) NOT NULL CHECK (afectacion_igv IN ('10','20','30')),
    tasa_impuesto NUMERIC(7,4) NOT NULL CHECK (tasa_impuesto >= 0 AND tasa_impuesto <= 100),
    importe_total NUMERIC(14,2) GENERATED ALWAYS AS
        (round(cantidad * precio_unitario,2) - descuento_total) STORED,
    base_imponible NUMERIC(14,2) GENERATED ALWAYS AS
        (round((round(cantidad * precio_unitario,2) - descuento_total) / (1 + tasa_impuesto / 100),2)) STORED,
    estado VARCHAR(15) NOT NULL DEFAULT 'ACTIVO' CHECK (estado IN ('ACTIVO','ANULADO')),
    notas_preparacion TEXT,
    anulado_en TIMESTAMPTZ, anulado_por BIGINT REFERENCES usuarios(id), motivo_anulacion TEXT,
    creado_por BIGINT NOT NULL,
    UNIQUE (id, pedido_id),
    FOREIGN KEY (cuenta_id, pedido_id) REFERENCES cuentas(id, pedido_id),
    FOREIGN KEY (receta_id, producto_id) REFERENCES recetas(id, producto_id),
    FOREIGN KEY (detalle_padre_id, pedido_id) REFERENCES detalles_pedido(id, pedido_id),
    CHECK (detalle_padre_id IS NULL OR detalle_padre_id <> id),
    CHECK (descuento_total <= round(cantidad * precio_unitario,2)),
    CHECK (afectacion_igv = '10' OR tasa_impuesto = 0),
    CHECK (estado <> 'ANULADO' OR (anulado_en IS NOT NULL AND anulado_por IS NOT NULL
        AND motivo_anulacion IS NOT NULL AND btrim(motivo_anulacion) <> '')),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE detalles_pedido IS 'Snapshot comercial. Adicionales como lineas hijas; envio como producto SERVICIO. Precios incluyen tributos.';

CREATE TABLE comandas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pedido_id BIGINT NOT NULL,
    sucursal_id BIGINT NOT NULL, estacion_id BIGINT NOT NULL,
    clave_idempotencia UUID NOT NULL UNIQUE,
    enviado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    enviado_por BIGINT NOT NULL REFERENCES usuarios(id), notas TEXT,
    UNIQUE (id, pedido_id),
    FOREIGN KEY (pedido_id, sucursal_id) REFERENCES pedidos(id, sucursal_id),
    FOREIGN KEY (estacion_id, sucursal_id) REFERENCES estaciones_preparacion(id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE comandas IS 'Cada envio a cocina/barra tiene identidad propia; reimprimir no crea otro envio.';

CREATE TABLE comanda_detalles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    comanda_id BIGINT NOT NULL, pedido_id BIGINT NOT NULL, detalle_pedido_id BIGINT NOT NULL,
    cantidad NUMERIC(14,3) NOT NULL CHECK (cantidad > 0),
    descripcion VARCHAR(250) NOT NULL, instrucciones TEXT,
    estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE'
        CHECK (estado IN ('PENDIENTE','PREPARACION','LISTO','ENTREGADO','CANCELADO')),
    iniciado_en TIMESTAMPTZ, listo_en TIMESTAMPTZ, entregado_en TIMESTAMPTZ,
    motivo_cancelacion TEXT,
    FOREIGN KEY (comanda_id, pedido_id) REFERENCES comandas(id, pedido_id),
    FOREIGN KEY (detalle_pedido_id, pedido_id) REFERENCES detalles_pedido(id, pedido_id),
    UNIQUE (comanda_id, detalle_pedido_id),
    CHECK (listo_en IS NULL OR iniciado_en IS NULL OR listo_en >= iniciado_en),
    CHECK (entregado_en IS NULL OR listo_en IS NULL OR entregado_en >= listo_en),
    CHECK (estado <> 'CANCELADO' OR (motivo_cancelacion IS NOT NULL AND btrim(motivo_cancelacion) <> '')),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE comanda_detalles IS 'Permite preparar y entregar en tandas. El servicio controla la cantidad acumulada enviada por linea.';

CREATE TABLE pedidos_delivery (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pedido_id BIGINT NOT NULL UNIQUE REFERENCES pedidos(id),
    destinatario VARCHAR(150) NOT NULL, telefono VARCHAR(30) NOT NULL,
    direccion TEXT NOT NULL, referencia TEXT, ubigeo VARCHAR(6),
    latitud NUMERIC(10,7) CHECK (latitud BETWEEN -90 AND 90),
    longitud NUMERIC(10,7) CHECK (longitud BETWEEN -180 AND 180),
    canal VARCHAR(40) NOT NULL DEFAULT 'PROPIO', codigo_externo VARCHAR(100),
    repartidor_id BIGINT REFERENCES empleados(id),
    estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE'
        CHECK (estado IN ('PENDIENTE','ASIGNADO','EN_RUTA','ENTREGADO','FALLIDO','CANCELADO')),
    programado_en TIMESTAMPTZ, salida_en TIMESTAMPTZ, entrega_en TIMESTAMPTZ,
    costo_logistico NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (costo_logistico >= 0),
    UNIQUE (canal, codigo_externo),
    CHECK (entrega_en IS NULL OR salida_en IS NULL OR entrega_en >= salida_en),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE pedidos_delivery IS 'Direccion historica de entrega. El cobro de envio se registra como detalle de pedido, no se suma aqui.';

-- 04. CAJA, PAGOS Y DEVOLUCIONES

CREATE TABLE metodos_pago (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    codigo VARCHAR(25) NOT NULL UNIQUE,
    nombre VARCHAR(80) NOT NULL,
    es_efectivo BOOLEAN NOT NULL DEFAULT FALSE,
    requiere_referencia BOOLEAN NOT NULL DEFAULT FALSE,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE metodos_pago IS 'Catalogo extensible de medios de cobro.';

CREATE TABLE cajas (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    codigo VARCHAR(20) NOT NULL, nombre VARCHAR(80) NOT NULL,
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (sucursal_id, codigo), UNIQUE (id, sucursal_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en),
    eliminado_en TIMESTAMPTZ,
    eliminado_por BIGINT,
    motivo_eliminacion TEXT,
    CHECK ((es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
        OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
            AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''))
);

COMMENT ON TABLE cajas IS 'Punto de cobro fisico o virtual.';

CREATE TABLE sesiones_caja (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    caja_id BIGINT NOT NULL, sucursal_id BIGINT NOT NULL,
    moneda CHAR(3) NOT NULL DEFAULT 'PEN' CHECK (moneda ~ '^[A-Z]{3}$'),
    abierto_por BIGINT NOT NULL REFERENCES usuarios(id), cerrado_por BIGINT REFERENCES usuarios(id),
    apertura_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, cierre_en TIMESTAMPTZ,
    estado VARCHAR(10) NOT NULL DEFAULT 'ABIERTA' CHECK (estado IN ('ABIERTA','CERRADA')),
    fondo_inicial NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (fondo_inicial >= 0),
    efectivo_contado NUMERIC(14,2) CHECK (efectivo_contado >= 0),
    observaciones TEXT,
    FOREIGN KEY (caja_id, sucursal_id) REFERENCES cajas(id, sucursal_id),
    CHECK (cierre_en IS NULL OR cierre_en >= apertura_en),
    CHECK ((estado = 'ABIERTA' AND cierre_en IS NULL AND cerrado_por IS NULL)
        OR (estado = 'CERRADA' AND cierre_en IS NOT NULL AND cerrado_por IS NOT NULL AND efectivo_contado IS NOT NULL)),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE sesiones_caja IS 'Turno de caja. Efectivo esperado y diferencia se calculan en una vista.';

CREATE UNIQUE INDEX uq_caja_sesion_abierta ON sesiones_caja(caja_id) WHERE estado = 'ABIERTA';

CREATE TABLE movimientos_caja (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sesion_caja_id BIGINT NOT NULL REFERENCES sesiones_caja(id),
    tipo VARCHAR(10) NOT NULL CHECK (tipo IN ('INGRESO','EGRESO')),
    monto NUMERIC(14,2) NOT NULL CHECK (monto > 0),
    concepto VARCHAR(200) NOT NULL, referencia VARCHAR(100),
    creado_por BIGINT NOT NULL,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    clave_idempotencia UUID NOT NULL UNIQUE
);

COMMENT ON TABLE movimientos_caja IS 'Movimientos extraordinarios de EFECTIVO. No duplicar cobros ni devoluciones en esta tabla.';

CREATE TABLE pagos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cuenta_id BIGINT NOT NULL REFERENCES cuentas(id),
    sesion_caja_id BIGINT NOT NULL REFERENCES sesiones_caja(id),
    metodo_pago_id BIGINT NOT NULL REFERENCES metodos_pago(id),
    monto NUMERIC(14,2) NOT NULL CHECK (monto > 0),
    efectivo_recibido NUMERIC(14,2) CHECK (efectivo_recibido >= monto),
    vuelto NUMERIC(14,2) GENERATED ALWAYS AS (coalesce(efectivo_recibido - monto,0)) STORED,
    estado VARCHAR(15) NOT NULL DEFAULT 'PENDIENTE'
        CHECK (estado IN ('PENDIENTE','CONFIRMADO','FALLIDO','CANCELADO')),
    proveedor_operacion VARCHAR(60), referencia_operacion VARCHAR(120),
    clave_idempotencia UUID NOT NULL UNIQUE,
    creado_por BIGINT NOT NULL,
    confirmado_en TIMESTAMPTZ,
    UNIQUE (proveedor_operacion, referencia_operacion),
    CHECK ((proveedor_operacion IS NULL) = (referencia_operacion IS NULL)),
    CHECK (estado <> 'CONFIRMADO' OR confirmado_en IS NOT NULL),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE pagos IS 'Multiples cobros por cuenta. monto es el aplicado a la deuda, sin vuelto ni propina.';

CREATE TABLE devoluciones_pago (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pago_id BIGINT NOT NULL REFERENCES pagos(id),
    sesion_caja_id BIGINT NOT NULL REFERENCES sesiones_caja(id),
    metodo_pago_id BIGINT NOT NULL REFERENCES metodos_pago(id),
    monto NUMERIC(14,2) NOT NULL CHECK (monto > 0),
    motivo TEXT NOT NULL CHECK (btrim(motivo) <> ''),
    referencia_operacion VARCHAR(120),
    clave_idempotencia UUID NOT NULL UNIQUE,
    autorizado_por BIGINT NOT NULL REFERENCES usuarios(id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL
);

COMMENT ON TABLE devoluciones_pago IS 'Solo devoluciones efectivamente ejecutadas; suma limitada al pago confirmado original.';

-- 05. COMPRAS Y KARDEX

CREATE TABLE compras (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    almacen_id BIGINT NOT NULL,
    proveedor_id BIGINT NOT NULL REFERENCES proveedores(id),
    tipo_documento VARCHAR(10) NOT NULL,
    serie VARCHAR(20) NOT NULL, numero VARCHAR(40) NOT NULL,
    fecha_documento DATE NOT NULL,
    fecha_recepcion TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    moneda CHAR(3) NOT NULL DEFAULT 'PEN' CHECK (moneda ~ '^[A-Z]{3}$'),
    tipo_cambio NUMERIC(18,6) NOT NULL DEFAULT 1 CHECK (tipo_cambio > 0),
    condicion_pago VARCHAR(10) NOT NULL DEFAULT 'CONTADO' CHECK (condicion_pago IN ('CONTADO','CREDITO')),
    fecha_vencimiento DATE,
    estado VARCHAR(15) NOT NULL DEFAULT 'BORRADOR' CHECK (estado IN ('BORRADOR','CONFIRMADA','CANCELADA')),
    notas TEXT, creado_por BIGINT NOT NULL,
    UNIQUE (proveedor_id, tipo_documento, serie, numero), UNIQUE (id, almacen_id),
    FOREIGN KEY (almacen_id, sucursal_id) REFERENCES almacenes(id, sucursal_id),
    CHECK (fecha_vencimiento IS NULL OR fecha_vencimiento >= fecha_documento),
    CHECK (condicion_pago <> 'CREDITO' OR fecha_vencimiento IS NOT NULL),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE compras IS 'Compra con recepcion unica. Pagos a proveedores y recepciones parciales requieren ampliacion del modulo.';

CREATE TABLE compra_detalles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    compra_id BIGINT NOT NULL REFERENCES compras(id),
    numero_linea INTEGER NOT NULL CHECK (numero_linea > 0),
    articulo_id BIGINT NOT NULL REFERENCES articulos_inventario(id),
    presentacion_id BIGINT,
    cantidad_compra NUMERIC(18,6) NOT NULL CHECK (cantidad_compra > 0),
    factor_unidad_base NUMERIC(18,6) NOT NULL CHECK (factor_unidad_base > 0),
    cantidad_base NUMERIC(18,6) GENERATED ALWAYS AS (cantidad_compra * factor_unidad_base) STORED,
    precio_unitario NUMERIC(18,6) NOT NULL CHECK (precio_unitario >= 0),
    descuento_total NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (descuento_total >= 0),
    tasa_impuesto NUMERIC(7,4) NOT NULL CHECK (tasa_impuesto BETWEEN 0 AND 100),
    importe_total NUMERIC(14,2) GENERATED ALWAYS AS (round(cantidad_compra * precio_unitario,2) - descuento_total) STORED,
    lote_id BIGINT,
    UNIQUE (compra_id, numero_linea), UNIQUE (id, articulo_id),
    FOREIGN KEY (presentacion_id, articulo_id) REFERENCES presentaciones_articulo(id, articulo_id),
    FOREIGN KEY (lote_id, articulo_id) REFERENCES lotes(id, articulo_id),
    CHECK (descuento_total <= round(cantidad_compra * precio_unitario,2)),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE compra_detalles IS 'Detalle de compra. Factor, cantidad y precio son historicos; unidad de compra descrita por presentacion.';

CREATE TABLE movimientos_inventario (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    almacen_id BIGINT NOT NULL REFERENCES almacenes(id),
    tipo VARCHAR(25) NOT NULL CHECK (tipo IN ('APERTURA','COMPRA','CONSUMO','REVENTA','MERMA',
        'AJUSTE_ENTRADA','AJUSTE_SALIDA','DEVOLUCION_COMPRA','DEVOLUCION_VENTA','TRANSFERENCIA_ENTRADA','TRANSFERENCIA_SALIDA')),
    direccion SMALLINT NOT NULL CHECK (direccion IN (-1,1)),
    estado VARCHAR(12) NOT NULL DEFAULT 'BORRADOR' CHECK (estado IN ('BORRADOR','CONFIRMADO')),
    compra_id BIGINT, detalle_pedido_id BIGINT REFERENCES detalles_pedido(id),
    movimiento_referencia_id BIGINT REFERENCES movimientos_inventario(id),
    motivo TEXT NOT NULL CHECK (btrim(motivo) <> ''),
    fecha_movimiento TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmado_en TIMESTAMPTZ,
    creado_por BIGINT NOT NULL,
    clave_idempotencia UUID NOT NULL UNIQUE,
    FOREIGN KEY (compra_id, almacen_id) REFERENCES compras(id, almacen_id),
    UNIQUE (id, almacen_id),
    CHECK (movimiento_referencia_id IS NULL OR movimiento_referencia_id <> id),
    CHECK ((direccion = 1 AND tipo IN ('APERTURA','COMPRA','AJUSTE_ENTRADA','DEVOLUCION_VENTA','TRANSFERENCIA_ENTRADA'))
        OR (direccion = -1 AND tipo IN ('CONSUMO','REVENTA','MERMA','AJUSTE_SALIDA','DEVOLUCION_COMPRA','TRANSFERENCIA_SALIDA'))),
    CHECK (tipo <> 'COMPRA' OR compra_id IS NOT NULL),
    CHECK (tipo NOT IN ('CONSUMO','REVENTA') OR detalle_pedido_id IS NOT NULL),
    CHECK (tipo NOT IN ('DEVOLUCION_COMPRA','DEVOLUCION_VENTA','TRANSFERENCIA_ENTRADA') OR movimiento_referencia_id IS NOT NULL),
    CHECK (estado <> 'CONFIRMADO' OR confirmado_en IS NOT NULL),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE movimientos_inventario IS 'Documento de stock. Confirmar valida saldos y congela cabecera/detalles. Transferencias deben generarse como par atomico por el servicio.';

CREATE UNIQUE INDEX uq_movimiento_compra ON movimientos_inventario(compra_id) WHERE tipo = 'COMPRA';

CREATE TABLE movimiento_inventario_detalles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    movimiento_id BIGINT NOT NULL,
    almacen_id BIGINT NOT NULL, articulo_id BIGINT NOT NULL,
    lote_id BIGINT, compra_detalle_id BIGINT,
    cantidad NUMERIC(18,6) NOT NULL CHECK (cantidad > 0),
    costo_unitario NUMERIC(18,6) NOT NULL CHECK (costo_unitario >= 0),
    costo_total NUMERIC(18,6) GENERATED ALWAYS AS (cantidad * costo_unitario) STORED,
    FOREIGN KEY (movimiento_id, almacen_id) REFERENCES movimientos_inventario(id, almacen_id),
    FOREIGN KEY (almacen_id, articulo_id) REFERENCES articulos_almacen(almacen_id, articulo_id),
    FOREIGN KEY (lote_id, articulo_id) REFERENCES lotes(id, articulo_id),
    FOREIGN KEY (compra_detalle_id, articulo_id) REFERENCES compra_detalles(id, articulo_id),
    UNIQUE NULLS NOT DISTINCT (movimiento_id, articulo_id, lote_id, compra_detalle_id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE movimiento_inventario_detalles IS 'Cantidades positivas en unidad base; signo en cabecera. Costo historico en moneda del negocio, calculado por el servicio.';

-- 06. COMPROBANTES ELECTRONICOS Y TRAZABILIDAD DE EMISION

CREATE TABLE series_comprobantes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    tipo CHAR(2) NOT NULL CHECK (tipo IN ('01','03','07','08')),
    serie VARCHAR(4) NOT NULL CHECK (serie ~ '^[A-Z0-9]{4}$'),
    ultimo_correlativo BIGINT NOT NULL DEFAULT 0 CHECK (ultimo_correlativo >= 0),
    administracion VARCHAR(10) NOT NULL DEFAULT 'LOCAL' CHECK (administracion IN ('LOCAL','PROVEEDOR')),
    es_activo BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (tipo, serie), UNIQUE (id, sucursal_id, tipo, serie),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE series_comprobantes IS 'Una serie por tipo y empresa. fn_siguiente_correlativo usa UPDATE atomico; nunca MAX+1.';

CREATE TABLE comprobantes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sucursal_id BIGINT NOT NULL REFERENCES sucursales(id),
    cuenta_id BIGINT NOT NULL REFERENCES cuentas(id),
    serie_id BIGINT NOT NULL, tipo CHAR(2) NOT NULL CHECK (tipo IN ('01','03','07','08')),
    serie VARCHAR(4) NOT NULL, numero BIGINT NOT NULL CHECK (numero > 0),
    fecha_emision TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    moneda CHAR(3) NOT NULL CHECK (moneda ~ '^[A-Z]{3}$'),
    emisor_ruc VARCHAR(11) NOT NULL CHECK (emisor_ruc ~ '^[0-9]{11}$'),
    emisor_razon_social VARCHAR(250) NOT NULL, emisor_direccion TEXT NOT NULL,
    receptor_tipo_documento VARCHAR(10), receptor_numero_documento VARCHAR(20),
    receptor_nombre VARCHAR(250) NOT NULL, receptor_direccion TEXT, receptor_email VARCHAR(254),
    comprobante_referencia_id BIGINT REFERENCES comprobantes(id),
    codigo_motivo VARCHAR(10), descripcion_motivo TEXT,
    estado VARCHAR(12) NOT NULL DEFAULT 'BORRADOR' CHECK (estado IN ('BORRADOR','EMITIDO','ANULADO')),
    emitido_en TIMESTAMPTZ, anulado_en TIMESTAMPTZ, motivo_anulacion TEXT,
    clave_idempotencia UUID NOT NULL UNIQUE,
    creado_por BIGINT NOT NULL,
    UNIQUE (tipo, serie, numero),
    FOREIGN KEY (serie_id, sucursal_id, tipo, serie) REFERENCES series_comprobantes(id, sucursal_id, tipo, serie),
    CHECK ((receptor_tipo_documento IS NULL) = (receptor_numero_documento IS NULL)),
    CHECK (tipo <> '01' OR (receptor_tipo_documento IS NOT NULL AND receptor_numero_documento IS NOT NULL)),
    CHECK ((tipo IN ('07','08')) = (comprobante_referencia_id IS NOT NULL)),
    CHECK (comprobante_referencia_id IS NULL OR comprobante_referencia_id <> id),
    CHECK (tipo NOT IN ('07','08') OR (codigo_motivo IS NOT NULL AND descripcion_motivo IS NOT NULL)),
    CHECK (estado = 'BORRADOR' OR emitido_en IS NOT NULL),
    CHECK (estado <> 'ANULADO' OR (anulado_en IS NOT NULL AND motivo_anulacion IS NOT NULL)),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE comprobantes IS 'Datos historicos del CPE. El conector debe validar catalogos, UBL, reglas fiscales y mecanismo de anulacion aplicable.';

CREATE TABLE comprobante_detalles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    comprobante_id BIGINT NOT NULL REFERENCES comprobantes(id),
    numero_linea INTEGER NOT NULL CHECK (numero_linea > 0),
    detalle_pedido_id BIGINT REFERENCES detalles_pedido(id),
    codigo_producto VARCHAR(40), descripcion VARCHAR(250) NOT NULL,
    unidad_sunat VARCHAR(10) NOT NULL,
    cantidad NUMERIC(14,3) NOT NULL CHECK (cantidad > 0),
    precio_unitario NUMERIC(14,6) NOT NULL CHECK (precio_unitario >= 0),
    descuento_total NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (descuento_total >= 0),
    afectacion_igv CHAR(2) NOT NULL CHECK (afectacion_igv IN ('10','20','30')),
    tasa_impuesto NUMERIC(7,4) NOT NULL CHECK (tasa_impuesto BETWEEN 0 AND 100),
    importe_total NUMERIC(14,2) GENERATED ALWAYS AS (round(cantidad * precio_unitario,2) - descuento_total) STORED,
    base_imponible NUMERIC(14,2) GENERATED ALWAYS AS
        (round((round(cantidad * precio_unitario,2) - descuento_total)/(1+tasa_impuesto/100),2)) STORED,
    UNIQUE (comprobante_id, numero_linea),
    CHECK (descuento_total <= round(cantidad * precio_unitario,2)),
    CHECK (afectacion_igv = '10' OR tasa_impuesto = 0),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE comprobante_detalles IS 'Snapshot tributario independiente de cambios posteriores en productos y pedidos.';

CREATE TABLE envios_comprobante (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    comprobante_id BIGINT NOT NULL REFERENCES comprobantes(id),
    numero_intento INTEGER NOT NULL CHECK (numero_intento > 0),
    proveedor VARCHAR(80) NOT NULL,
    id_externo VARCHAR(150), ticket VARCHAR(150),
    estado VARCHAR(25) NOT NULL CHECK (estado IN ('PENDIENTE','ENVIADO','ACEPTADO','ACEPTADO_OBSERVADO','RECHAZADO','ERROR','BAJA_ACEPTADA')),
    codigo_respuesta VARCHAR(30), descripcion_respuesta TEXT,
    xml_uri TEXT, cdr_uri TEXT, pdf_uri TEXT, hash_documento TEXT,
    enviado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, respondido_en TIMESTAMPTZ,
    UNIQUE (comprobante_id, numero_intento),
    CHECK (respondido_en IS NULL OR respondido_en >= enviado_en),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    creado_por BIGINT NOT NULL,
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actualizado_por BIGINT NOT NULL,
    CHECK (actualizado_en >= creado_en)
);

COMMENT ON TABLE envios_comprobante IS 'Intentos y respuesta SUNAT/PSE/OSE. URI de archivos protegidos; no guardar credenciales ni claves privadas aqui.';

CREATE TABLE auditoria_eventos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tabla VARCHAR(80) NOT NULL,
    registro_id BIGINT NOT NULL,
    accion VARCHAR(10) NOT NULL CHECK (accion IN ('INSERT','UPDATE','DELETE')),
    creado_por BIGINT NOT NULL, usuario_bd TEXT NOT NULL DEFAULT CURRENT_USER,
    transaccion BIGINT NOT NULL DEFAULT txid_current(),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    datos_antes JSONB, datos_despues JSONB
);

COMMENT ON TABLE auditoria_eventos IS 'Auditoria de operaciones seleccionadas; no incluye hashes de contrasena. Restringir acceso y definir retencion.';

-- 06B. CLAVES DE AUDITORIA Y ACTOR TECNICO DE INSTALACION

ALTER TABLE negocio ADD CONSTRAINT fk_negocio_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE negocio ADD CONSTRAINT fk_negocio_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE sucursales ADD CONSTRAINT fk_sucursales_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE sucursales ADD CONSTRAINT fk_sucursales_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE sucursales ADD CONSTRAINT fk_sucursales_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE roles ADD CONSTRAINT fk_roles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE roles ADD CONSTRAINT fk_roles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE permisos ADD CONSTRAINT fk_permisos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE permisos ADD CONSTRAINT fk_permisos_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE roles_permisos ADD CONSTRAINT fk_roles_permisos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE roles_permisos ADD CONSTRAINT fk_roles_permisos_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE usuarios ADD CONSTRAINT fk_usuarios_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE usuarios ADD CONSTRAINT fk_usuarios_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE usuarios ADD CONSTRAINT fk_usuarios_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE usuarios_roles ADD CONSTRAINT fk_usuarios_roles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE usuarios_roles ADD CONSTRAINT fk_usuarios_roles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE usuarios_sucursales ADD CONSTRAINT fk_usuarios_sucursales_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE usuarios_sucursales ADD CONSTRAINT fk_usuarios_sucursales_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE empleados ADD CONSTRAINT fk_empleados_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE empleados ADD CONSTRAINT fk_empleados_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE empleados ADD CONSTRAINT fk_empleados_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE clientes ADD CONSTRAINT fk_clientes_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE clientes ADD CONSTRAINT fk_clientes_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE clientes ADD CONSTRAINT fk_clientes_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE proveedores ADD CONSTRAINT fk_proveedores_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE proveedores ADD CONSTRAINT fk_proveedores_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE proveedores ADD CONSTRAINT fk_proveedores_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE unidades_medida ADD CONSTRAINT fk_unidades_medida_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE unidades_medida ADD CONSTRAINT fk_unidades_medida_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE categorias ADD CONSTRAINT fk_categorias_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE categorias ADD CONSTRAINT fk_categorias_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE impuestos ADD CONSTRAINT fk_impuestos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE impuestos ADD CONSTRAINT fk_impuestos_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE tasas_impuesto ADD CONSTRAINT fk_tasas_impuesto_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE tasas_impuesto ADD CONSTRAINT fk_tasas_impuesto_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE estaciones_preparacion ADD CONSTRAINT fk_estaciones_preparacion_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE estaciones_preparacion ADD CONSTRAINT fk_estaciones_preparacion_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE articulos_inventario ADD CONSTRAINT fk_articulos_inventario_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE articulos_inventario ADD CONSTRAINT fk_articulos_inventario_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE articulos_inventario ADD CONSTRAINT fk_articulos_inventario_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE presentaciones_articulo ADD CONSTRAINT fk_presentaciones_articulo_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE presentaciones_articulo ADD CONSTRAINT fk_presentaciones_articulo_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE almacenes ADD CONSTRAINT fk_almacenes_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE almacenes ADD CONSTRAINT fk_almacenes_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE almacenes ADD CONSTRAINT fk_almacenes_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE articulos_almacen ADD CONSTRAINT fk_articulos_almacen_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE articulos_almacen ADD CONSTRAINT fk_articulos_almacen_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE lotes ADD CONSTRAINT fk_lotes_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE lotes ADD CONSTRAINT fk_lotes_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE productos ADD CONSTRAINT fk_productos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE productos ADD CONSTRAINT fk_productos_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE productos ADD CONSTRAINT fk_productos_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE productos_sucursal ADD CONSTRAINT fk_productos_sucursal_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE productos_sucursal ADD CONSTRAINT fk_productos_sucursal_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE recetas ADD CONSTRAINT fk_recetas_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE recetas ADD CONSTRAINT fk_recetas_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE receta_detalles ADD CONSTRAINT fk_receta_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE receta_detalles ADD CONSTRAINT fk_receta_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE zonas ADD CONSTRAINT fk_zonas_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE zonas ADD CONSTRAINT fk_zonas_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE mesas ADD CONSTRAINT fk_mesas_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE mesas ADD CONSTRAINT fk_mesas_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE reservas ADD CONSTRAINT fk_reservas_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE reservas ADD CONSTRAINT fk_reservas_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE pedidos ADD CONSTRAINT fk_pedidos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE pedidos ADD CONSTRAINT fk_pedidos_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE cuentas ADD CONSTRAINT fk_cuentas_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE cuentas ADD CONSTRAINT fk_cuentas_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE detalles_pedido ADD CONSTRAINT fk_detalles_pedido_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE detalles_pedido ADD CONSTRAINT fk_detalles_pedido_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comandas ADD CONSTRAINT fk_comandas_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comandas ADD CONSTRAINT fk_comandas_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comanda_detalles ADD CONSTRAINT fk_comanda_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comanda_detalles ADD CONSTRAINT fk_comanda_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE pedidos_delivery ADD CONSTRAINT fk_pedidos_delivery_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE pedidos_delivery ADD CONSTRAINT fk_pedidos_delivery_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE metodos_pago ADD CONSTRAINT fk_metodos_pago_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE metodos_pago ADD CONSTRAINT fk_metodos_pago_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE cajas ADD CONSTRAINT fk_cajas_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE cajas ADD CONSTRAINT fk_cajas_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE cajas ADD CONSTRAINT fk_cajas_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE sesiones_caja ADD CONSTRAINT fk_sesiones_caja_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE sesiones_caja ADD CONSTRAINT fk_sesiones_caja_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimientos_caja ADD CONSTRAINT fk_movimientos_caja_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE pagos ADD CONSTRAINT fk_pagos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE pagos ADD CONSTRAINT fk_pagos_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE devoluciones_pago ADD CONSTRAINT fk_devoluciones_pago_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE compras ADD CONSTRAINT fk_compras_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE compras ADD CONSTRAINT fk_compras_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE compra_detalles ADD CONSTRAINT fk_compra_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE compra_detalles ADD CONSTRAINT fk_compra_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimientos_inventario ADD CONSTRAINT fk_movimientos_inventario_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimientos_inventario ADD CONSTRAINT fk_movimientos_inventario_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimiento_inventario_detalles ADD CONSTRAINT fk_movimiento_inventario_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimiento_inventario_detalles ADD CONSTRAINT fk_movimiento_inventario_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE series_comprobantes ADD CONSTRAINT fk_series_comprobantes_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE series_comprobantes ADD CONSTRAINT fk_series_comprobantes_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;


ALTER TABLE comprobantes ADD CONSTRAINT fk_comprobantes_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comprobantes ADD CONSTRAINT fk_comprobantes_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comprobante_detalles ADD CONSTRAINT fk_comprobante_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comprobante_detalles ADD CONSTRAINT fk_comprobante_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE envios_comprobante ADD CONSTRAINT fk_envios_comprobante_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE envios_comprobante ADD CONSTRAINT fk_envios_comprobante_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE auditoria_eventos ADD CONSTRAINT fk_auditoria_eventos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

CREATE UNIQUE INDEX uq_usuario_sistema ON usuarios(es_sistema) WHERE es_sistema;
DO $$
DECLARE v_id BIGINT;
BEGIN
    v_id := nextval(pg_get_serial_sequence('sch_restaurante.usuarios','id'));
    INSERT INTO usuarios(id,username,password_hash,es_sistema,creado_por,actualizado_por)
    OVERRIDING SYSTEM VALUE VALUES (v_id,'__sistema__',NULL,TRUE,v_id,v_id);
    PERFORM set_config('app.usuario_id',v_id::text,true);
    INSERT INTO auditoria_eventos(tabla,registro_id,accion,creado_por,datos_despues)
    SELECT 'usuarios',id,'INSERT',v_id,to_jsonb(u)-'password_hash' FROM usuarios u WHERE id=v_id;
END;
$$;

-- 07. VISTAS OPERATIVAS: TOTALES Y SALDOS SIN DUPLICAR ESTADO

CREATE VIEW v_cuenta_totales AS
WITH lineas AS (
 SELECT cuenta_id, sum(base_imponible) AS base, sum(importe_total-base_imponible) AS impuesto,
        sum(descuento_total) AS descuento, sum(importe_total) AS total
 FROM detalles_pedido WHERE estado='ACTIVO' GROUP BY cuenta_id
), cobros AS (
 SELECT cuenta_id,sum(monto) AS cobrado FROM pagos WHERE estado='CONFIRMADO' GROUP BY cuenta_id
), devoluciones AS (
 SELECT p.cuenta_id,sum(d.monto) AS devuelto FROM devoluciones_pago d
 JOIN pagos p ON p.id=d.pago_id GROUP BY p.cuenta_id
)
SELECT c.id AS cuenta_id,c.pedido_id,c.estado,
 coalesce(l.base,0) AS base_imponible,coalesce(l.impuesto,0) AS impuesto,
 coalesce(l.descuento,0) AS descuento,coalesce(l.total,0) AS total,
 coalesce(p.cobrado,0) AS cobrado,coalesce(d.devuelto,0) AS devuelto,
 coalesce(p.cobrado,0)-coalesce(d.devuelto,0) AS cobro_neto,
 greatest(coalesce(l.total,0)-coalesce(p.cobrado,0),0) AS pendiente_cobro_original,
 CASE WHEN coalesce(d.devuelto,0)>0 AND d.devuelto=p.cobrado THEN 'REEMBOLSADO'
      WHEN coalesce(d.devuelto,0)>0 THEN 'REEMBOLSADO_PARCIAL'
      WHEN coalesce(l.total,0)=0 THEN 'SIN_CARGO'
      WHEN coalesce(p.cobrado,0)>=l.total THEN 'PAGADO'
      WHEN coalesce(p.cobrado,0)>0 THEN 'PARCIAL' ELSE 'PENDIENTE' END AS estado_cobro
FROM cuentas c LEFT JOIN lineas l ON l.cuenta_id=c.id
LEFT JOIN cobros p ON p.cuenta_id=c.id LEFT JOIN devoluciones d ON d.cuenta_id=c.id;
COMMENT ON VIEW v_cuenta_totales IS 'Reembolsos no reabren automaticamente la deuda original. Ajustes comerciales posteriores requieren notas de credito/debito.';

CREATE VIEW v_pedido_totales AS
SELECT p.id AS pedido_id, p.sucursal_id, p.tipo, p.estado, p.moneda,
 coalesce(sum(c.total),0) AS total, coalesce(sum(c.cobrado),0) AS cobrado,
 coalesce(sum(c.devuelto),0) AS devuelto, coalesce(sum(c.cobro_neto),0) AS cobro_neto
FROM pedidos p LEFT JOIN v_cuenta_totales c ON c.pedido_id=p.id
GROUP BY p.id;

CREATE VIEW v_stock_lotes AS
SELECT d.almacen_id,d.articulo_id,d.lote_id,
 sum(m.direccion*d.cantidad) AS stock
FROM movimiento_inventario_detalles d
JOIN movimientos_inventario m ON m.id=d.movimiento_id
WHERE m.estado='CONFIRMADO'
GROUP BY d.almacen_id,d.articulo_id,d.lote_id;

CREATE VIEW v_existencias AS
SELECT a.almacen_id,a.articulo_id,a.es_activo AS asignacion_es_activo,a.stock_minimo,a.stock_maximo,
 coalesce(s.stock,0) AS stock_actual,
 coalesce(s.stock,0)<a.stock_minimo AS bajo_minimo
FROM articulos_almacen a
LEFT JOIN (SELECT almacen_id,articulo_id,sum(stock) AS stock FROM v_stock_lotes
 GROUP BY almacen_id,articulo_id) s USING (almacen_id,articulo_id);

CREATE VIEW v_kardex AS
SELECT m.id AS movimiento_id,d.id AS detalle_id,m.fecha_movimiento,m.confirmado_en,
 m.tipo,m.almacen_id,d.articulo_id,d.lote_id,m.direccion,
 d.cantidad,d.costo_unitario,m.direccion*d.cantidad AS cantidad_neta,
 m.direccion*d.costo_total AS costo_neto,m.compra_id,m.detalle_pedido_id,
 m.movimiento_referencia_id,m.creado_por,m.motivo
FROM movimientos_inventario m JOIN movimiento_inventario_detalles d ON d.movimiento_id=m.id
WHERE m.estado='CONFIRMADO';

CREATE VIEW v_compra_totales AS
SELECT c.id AS compra_id,c.moneda,c.estado,
 coalesce(sum(d.importe_total),0) AS total,
 coalesce(sum(round(d.importe_total/(1+d.tasa_impuesto/100),2)),0) AS base_imponible
FROM compras c LEFT JOIN compra_detalles d ON d.compra_id=c.id GROUP BY c.id;

CREATE VIEW v_comprobante_totales AS
SELECT c.id AS comprobante_id,c.tipo,c.serie,c.numero,c.moneda,c.estado,
 coalesce(sum(d.base_imponible) FILTER (WHERE d.afectacion_igv='10'),0) AS gravado,
 coalesce(sum(d.base_imponible) FILTER (WHERE d.afectacion_igv='20'),0) AS exonerado,
 coalesce(sum(d.base_imponible) FILTER (WHERE d.afectacion_igv='30'),0) AS inafecto,
 coalesce(sum(d.importe_total-d.base_imponible),0) AS impuesto,
 coalesce(sum(d.importe_total),0) AS total,
 CASE WHEN c.tipo='07' THEN -1 ELSE 1 END AS signo_documento
FROM comprobantes c LEFT JOIN comprobante_detalles d ON d.comprobante_id=c.id GROUP BY c.id;

CREATE VIEW v_caja_resumen AS
WITH cobros AS (
 SELECT p.sesion_caja_id,sum(p.monto) AS total,
 sum(p.monto) FILTER (WHERE mp.es_efectivo) AS efectivo
 FROM pagos p JOIN metodos_pago mp ON mp.id=p.metodo_pago_id
 WHERE p.estado='CONFIRMADO' GROUP BY p.sesion_caja_id
), devoluciones AS (
 SELECT d.sesion_caja_id,sum(d.monto) FILTER (WHERE mp.es_efectivo) AS efectivo
 FROM devoluciones_pago d JOIN metodos_pago mp ON mp.id=d.metodo_pago_id GROUP BY d.sesion_caja_id
), movimientos AS (
 SELECT sesion_caja_id,sum(CASE WHEN tipo='INGRESO' THEN monto ELSE -monto END) AS neto
 FROM movimientos_caja GROUP BY sesion_caja_id
)
SELECT s.id AS sesion_caja_id,s.caja_id,s.sucursal_id,s.moneda,s.estado,s.fondo_inicial,
 coalesce(c.total,0) AS cobros_todos_medios,coalesce(c.efectivo,0) AS cobros_efectivo,
 coalesce(d.efectivo,0) AS devoluciones_efectivo,coalesce(m.neto,0) AS movimientos_extraordinarios,
 s.fondo_inicial+coalesce(c.efectivo,0)-coalesce(d.efectivo,0)+coalesce(m.neto,0) AS efectivo_esperado,
 s.efectivo_contado,
 s.efectivo_contado-(s.fondo_inicial+coalesce(c.efectivo,0)-coalesce(d.efectivo,0)+coalesce(m.neto,0)) AS diferencia
FROM sesiones_caja s LEFT JOIN cobros c ON c.sesion_caja_id=s.id
LEFT JOIN devoluciones d ON d.sesion_caja_id=s.id LEFT JOIN movimientos m ON m.sesion_caja_id=s.id;

CREATE VIEW v_mesas_estado AS
SELECT m.id AS mesa_id,m.sucursal_id,m.numero_mesa,m.capacidad,m.es_activo,
 CASE WHEN NOT m.es_activo THEN 'BAJA_LOGICA'
      WHEN NOT m.habilitada THEN 'NO_HABILITADA'
      WHEN p.id IS NOT NULL THEN 'OCUPADA' ELSE 'DISPONIBLE' END AS estado_actual,
 p.id AS pedido_activo_id
FROM mesas m LEFT JOIN pedidos p ON p.mesa_id=m.id AND p.cerrado_en IS NULL;

-- 08. FUNCIONES Y REGLAS DE INTEGRIDAD TRANSACCIONAL
CREATE FUNCTION fn_exigir_read_committed() RETURNS void
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    -- Los agregados se consultan despues de adquirir bloqueos. READ COMMITTED
    -- permite ver la confirmacion anterior; evita usar un snapshot antiguo.
    IF current_setting('transaction_isolation') <> 'read committed' THEN
        RAISE EXCEPTION 'Estas operaciones requieren aislamiento READ COMMITTED';
    END IF;
END;
$$;

CREATE FUNCTION fn_no_borrar() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    RAISE EXCEPTION 'No se permite borrar %. Use anulacion o una operacion compensatoria.', TG_TABLE_NAME;
END;
$$;

CREATE FUNCTION fn_solo_insertar() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    RAISE EXCEPTION '% conserva historia inmutable; registrar una operacion compensatoria.', TG_TABLE_NAME;
END;
$$;

CREATE FUNCTION fn_auditar() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_antes JSONB; v_despues JSONB;
BEGIN
    IF TG_OP<>'INSERT' THEN v_antes:=to_jsonb(OLD); END IF;
    IF TG_OP<>'DELETE' THEN v_despues:=to_jsonb(NEW); END IF;
    -- No registrar el material de autenticacion en las imagenes de auditoria.
    IF TG_TABLE_NAME='usuarios' THEN
        v_antes:=v_antes-'password_hash'; v_despues:=v_despues-'password_hash';
    END IF;
    INSERT INTO auditoria_eventos(tabla,registro_id,accion,creado_por,datos_antes,datos_despues)
    VALUES (TG_TABLE_NAME,CASE WHEN TG_OP='DELETE' THEN OLD.id ELSE NEW.id END,
        TG_OP, fn_usuario_actor(), v_antes, v_despues);
    RETURN NULL;
END;
$$;

CREATE FUNCTION fn_validar_reserva() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_mesa mesas%ROWTYPE;
BEGIN
    SELECT * INTO STRICT v_mesa FROM mesas WHERE id=NEW.mesa_id FOR SHARE;
    IF NEW.estado IN ('PENDIENTE','CONFIRMADA') AND
        (NOT v_mesa.es_activo OR NOT v_mesa.habilitada OR NEW.cantidad_personas>v_mesa.capacidad) THEN
        RAISE EXCEPTION 'Mesa no habilitada o capacidad insuficiente para la reserva';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_reserva BEFORE INSERT OR UPDATE ON reservas
FOR EACH ROW EXECUTE FUNCTION fn_validar_reserva();

CREATE FUNCTION fn_guardar_detalle_pedido() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_old BIGINT; v_new BIGINT; v_cuenta RECORD;
BEGIN
    IF TG_OP<>'INSERT' THEN v_old:=OLD.cuenta_id; END IF;
    IF TG_OP<>'DELETE' THEN v_new:=NEW.cuenta_id; END IF;
    -- El mismo bloqueo usa la confirmacion del pago: evita cambiar total mientras se cobra.
    FOR v_cuenta IN SELECT * FROM cuentas WHERE id IN (v_old,v_new) ORDER BY id FOR UPDATE LOOP
        IF v_cuenta.estado<>'ABIERTA' OR EXISTS (
            SELECT 1 FROM pagos WHERE cuenta_id=v_cuenta.id AND estado='CONFIRMADO') THEN
            RAISE EXCEPTION 'Cuenta % cerrada o con cobros: sus consumos ya no son editables',v_cuenta.id;
        END IF;
        IF EXISTS (SELECT 1 FROM pedidos WHERE id=v_cuenta.pedido_id AND cerrado_en IS NOT NULL) THEN
            RAISE EXCEPTION 'El pedido ya esta cerrado';
        END IF;
    END LOOP;
    IF TG_OP='UPDATE' AND NEW.pedido_id<>OLD.pedido_id THEN
        RAISE EXCEPTION 'No se permite trasladar una linea a otro pedido';
    END IF;
    IF TG_OP='DELETE' AND EXISTS (SELECT 1 FROM comanda_detalles WHERE detalle_pedido_id=OLD.id) THEN
        RAISE EXCEPTION 'Una linea enviada a cocina debe anularse, no borrarse';
    END IF;
    RETURN CASE WHEN TG_OP='DELETE' THEN OLD ELSE NEW END;
END;
$$;
CREATE TRIGGER a_guardar_detalle BEFORE INSERT OR UPDATE OR DELETE ON detalles_pedido
FOR EACH ROW EXECUTE FUNCTION fn_guardar_detalle_pedido();

CREATE FUNCTION fn_validar_cuenta() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_total NUMERIC; v_cobrado NUMERIC;
BEGIN
    PERFORM id FROM pedidos WHERE id=NEW.pedido_id AND cerrado_en IS NULL FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION 'El pedido no existe o ya esta cerrado'; END IF;
    IF TG_OP='UPDATE' THEN
        IF NEW.pedido_id<>OLD.pedido_id THEN RAISE EXCEPTION 'No se puede cambiar el pedido de una cuenta'; END IF;
        IF OLD.estado<>'ABIERTA' THEN RAISE EXCEPTION 'La cuenta ya esta cerrada/cancelada'; END IF;
    END IF;
    IF TG_OP='INSERT' AND NEW.estado<>'ABIERTA' THEN RAISE EXCEPTION 'Crear la cuenta en estado ABIERTA'; END IF;
    IF NEW.estado='CERRADA' THEN
        SELECT total,cobrado INTO v_total,v_cobrado FROM v_cuenta_totales WHERE cuenta_id=NEW.id;
        IF v_cobrado<v_total THEN RAISE EXCEPTION 'No se puede cerrar una cuenta con saldo pendiente'; END IF;
        IF EXISTS (SELECT 1 FROM pagos WHERE cuenta_id=NEW.id AND estado='PENDIENTE') THEN
            RAISE EXCEPTION 'Resolver pagos pendientes antes de cerrar la cuenta';
        END IF;
        NEW.cerrado_en:=coalesce(NEW.cerrado_en,clock_timestamp());
    ELSIF NEW.estado='CANCELADA' THEN
        IF EXISTS (SELECT 1 FROM pagos WHERE cuenta_id=NEW.id AND estado='CONFIRMADO')
           OR EXISTS (SELECT 1 FROM detalles_pedido WHERE cuenta_id=NEW.id AND estado='ACTIVO') THEN
            RAISE EXCEPTION 'Anular lineas antes de cancelar una cuenta; una cuenta cobrada requiere devolucion';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_cuenta BEFORE INSERT OR UPDATE ON cuentas
FOR EACH ROW EXECUTE FUNCTION fn_validar_cuenta();

CREATE FUNCTION fn_validar_pedido() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF TG_OP='UPDATE' THEN
        IF NEW.sucursal_id<>OLD.sucursal_id OR NEW.moneda<>OLD.moneda THEN
            RAISE EXCEPTION 'Sucursal y moneda del pedido son inmutables';
        END IF;
        IF OLD.cerrado_en IS NOT NULL THEN RAISE EXCEPTION 'El pedido ya esta cerrado'; END IF;
        NEW.version_bloqueo:=OLD.version_bloqueo+1;
    END IF;
    IF NEW.mesa_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM mesas WHERE id=NEW.mesa_id AND es_activo AND habilitada) THEN
        RAISE EXCEPTION 'La mesa no esta habilitada';
    END IF;
    IF NEW.cerrado_en IS NOT NULL AND EXISTS (
        SELECT 1 FROM cuentas WHERE pedido_id=NEW.id AND estado='ABIERTA') THEN
        RAISE EXCEPTION 'Cerrar o cancelar las cuentas antes de cerrar el pedido';
    END IF;
    IF NEW.estado='CANCELADO' AND EXISTS (
        SELECT 1 FROM cuentas c JOIN pagos p ON p.cuenta_id=c.id
        WHERE c.pedido_id=NEW.id AND p.estado='CONFIRMADO') THEN
        RAISE EXCEPTION 'Pedido con cobros: registrar devolucion y documento fiscal correspondiente';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_pedido BEFORE INSERT OR UPDATE ON pedidos
FOR EACH ROW EXECUTE FUNCTION fn_validar_pedido();

CREATE FUNCTION fn_validar_delivery() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pedidos WHERE id=NEW.pedido_id AND tipo='DELIVERY') THEN
        RAISE EXCEPTION 'Solo pedidos DELIVERY pueden registrar datos de reparto';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_delivery BEFORE INSERT OR UPDATE ON pedidos_delivery
FOR EACH ROW EXECUTE FUNCTION fn_validar_delivery();

CREATE FUNCTION fn_validar_pago() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_cuenta cuentas%ROWTYPE; v_pedido pedidos%ROWTYPE;
    v_sesion sesiones_caja%ROWTYPE; v_metodo metodos_pago%ROWTYPE;
    v_total NUMERIC; v_cobrado NUMERIC;
BEGIN
    PERFORM fn_exigir_read_committed();
    IF TG_OP='UPDATE' THEN
        IF OLD.estado='CONFIRMADO' THEN RAISE EXCEPTION 'Pago confirmado inmutable; registrar una devolucion'; END IF;
        IF NEW.cuenta_id<>OLD.cuenta_id OR NEW.sesion_caja_id<>OLD.sesion_caja_id THEN
            RAISE EXCEPTION 'No se puede cambiar cuenta o sesion de un pago';
        END IF;
    END IF;
    SELECT * INTO STRICT v_cuenta FROM cuentas WHERE id=NEW.cuenta_id FOR UPDATE;
    SELECT * INTO STRICT v_pedido FROM pedidos WHERE id=v_cuenta.pedido_id;
    SELECT * INTO STRICT v_sesion FROM sesiones_caja WHERE id=NEW.sesion_caja_id FOR UPDATE;
    SELECT * INTO STRICT v_metodo FROM metodos_pago WHERE id=NEW.metodo_pago_id FOR SHARE;
    IF v_cuenta.estado<>'ABIERTA' OR v_pedido.cerrado_en IS NOT NULL THEN
        RAISE EXCEPTION 'La cuenta o el pedido no admiten cobros';
    END IF;
    IF v_sesion.estado<>'ABIERTA' OR v_sesion.sucursal_id<>v_pedido.sucursal_id OR v_sesion.moneda<>v_pedido.moneda THEN
        RAISE EXCEPTION 'Sesion cerrada o sucursal/moneda incompatible';
    END IF;
    IF NOT v_metodo.es_activo THEN RAISE EXCEPTION 'Metodo de pago inactivo'; END IF;
    IF v_metodo.es_efectivo AND NEW.efectivo_recibido IS NULL THEN
        RAISE EXCEPTION 'Indicar efectivo recibido';
    ELSIF NOT v_metodo.es_efectivo AND NEW.efectivo_recibido IS NOT NULL THEN
        RAISE EXCEPTION 'El efectivo recibido corresponde solo a pagos en efectivo';
    END IF;
    IF NEW.estado='CONFIRMADO' THEN
        IF v_metodo.requiere_referencia AND (NEW.referencia_operacion IS NULL OR btrim(NEW.referencia_operacion)='') THEN
            RAISE EXCEPTION 'Indicar referencia externa de la operacion confirmada';
        END IF;
        SELECT total,cobrado INTO v_total,v_cobrado FROM v_cuenta_totales WHERE cuenta_id=NEW.cuenta_id;
        IF NEW.monto>v_total-v_cobrado THEN RAISE EXCEPTION 'El pago supera el saldo pendiente de la cuenta'; END IF;
        NEW.confirmado_en:=coalesce(NEW.confirmado_en,clock_timestamp());
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_pago BEFORE INSERT OR UPDATE ON pagos
FOR EACH ROW EXECUTE FUNCTION fn_validar_pago();
CREATE TRIGGER a_no_borrar BEFORE DELETE ON pagos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE FUNCTION fn_validar_devolucion() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_pago pagos%ROWTYPE; v_sesion sesiones_caja%ROWTYPE;
    v_pedido pedidos%ROWTYPE; v_devuelto NUMERIC;
BEGIN
    PERFORM fn_exigir_read_committed();
    SELECT * INTO STRICT v_pago FROM pagos WHERE id=NEW.pago_id FOR UPDATE;
    SELECT * INTO STRICT v_sesion FROM sesiones_caja WHERE id=NEW.sesion_caja_id FOR UPDATE;
    SELECT p.* INTO STRICT v_pedido FROM pedidos p JOIN cuentas c ON c.pedido_id=p.id WHERE c.id=v_pago.cuenta_id;
    IF v_pago.estado<>'CONFIRMADO' THEN RAISE EXCEPTION 'Solo se devuelve un pago confirmado'; END IF;
    IF v_sesion.estado<>'ABIERTA' OR v_sesion.sucursal_id<>v_pedido.sucursal_id OR v_sesion.moneda<>v_pedido.moneda THEN
        RAISE EXCEPTION 'Sesion de devolucion cerrada o incompatible';
    END IF;
    SELECT coalesce(sum(monto),0) INTO v_devuelto FROM devoluciones_pago WHERE pago_id=NEW.pago_id;
    IF v_devuelto+NEW.monto>v_pago.monto THEN RAISE EXCEPTION 'La devolucion acumulada supera el cobro original'; END IF;
    IF NOT EXISTS (SELECT 1 FROM metodos_pago WHERE id=NEW.metodo_pago_id AND es_activo) THEN
        RAISE EXCEPTION 'Metodo de devolucion inactivo';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_devolucion BEFORE INSERT ON devoluciones_pago
FOR EACH ROW EXECUTE FUNCTION fn_validar_devolucion();
CREATE TRIGGER a_inmutable BEFORE UPDATE OR DELETE ON devoluciones_pago
FOR EACH ROW EXECUTE FUNCTION fn_solo_insertar();

CREATE FUNCTION fn_validar_movimiento_caja() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_estado TEXT;
BEGIN
    SELECT estado INTO STRICT v_estado FROM sesiones_caja WHERE id=NEW.sesion_caja_id FOR UPDATE;
    IF v_estado<>'ABIERTA' THEN RAISE EXCEPTION 'La sesion de caja esta cerrada'; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_movimiento_caja BEFORE INSERT ON movimientos_caja
FOR EACH ROW EXECUTE FUNCTION fn_validar_movimiento_caja();
CREATE TRIGGER a_inmutable BEFORE UPDATE OR DELETE ON movimientos_caja
FOR EACH ROW EXECUTE FUNCTION fn_solo_insertar();

CREATE FUNCTION fn_validar_sesion() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF TG_OP='INSERT' AND NEW.estado<>'ABIERTA' THEN RAISE EXCEPTION 'Abrir sesion antes de cerrarla'; END IF;
    IF TG_OP='UPDATE' THEN
        IF OLD.estado='CERRADA' THEN RAISE EXCEPTION 'Sesion cerrada inmutable'; END IF;
        IF NEW.caja_id<>OLD.caja_id OR NEW.sucursal_id<>OLD.sucursal_id OR NEW.moneda<>OLD.moneda
           OR NEW.fondo_inicial<>OLD.fondo_inicial OR NEW.apertura_en<>OLD.apertura_en THEN
            RAISE EXCEPTION 'No cambiar datos de apertura; registrar movimientos extraordinarios';
        END IF;
        IF NEW.estado='CERRADA' AND EXISTS (
            SELECT 1 FROM pagos WHERE sesion_caja_id=NEW.id AND estado='PENDIENTE') THEN
            RAISE EXCEPTION 'Resolver pagos pendientes antes de cerrar caja';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_sesion BEFORE INSERT OR UPDATE ON sesiones_caja
FOR EACH ROW EXECUTE FUNCTION fn_validar_sesion();
CREATE TRIGGER a_no_borrar BEFORE DELETE ON sesiones_caja FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE FUNCTION fn_proteger_metodo_pago() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF NEW.es_efectivo<>OLD.es_efectivo AND (
        EXISTS (SELECT 1 FROM pagos WHERE metodo_pago_id=OLD.id)
        OR EXISTS (SELECT 1 FROM devoluciones_pago WHERE metodo_pago_id=OLD.id)) THEN
        RAISE EXCEPTION 'No reinterpretar un medio usado: crear otro medio de pago';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_proteger_metodo BEFORE UPDATE ON metodos_pago FOR EACH ROW EXECUTE FUNCTION fn_proteger_metodo_pago();

-- Guardas de documentos: detalles editables solamente con padre BORRADOR.
CREATE FUNCTION fn_detalle_borrador() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_old BIGINT; v_new BIGINT; v_row RECORD; v_col TEXT:=TG_ARGV[1];
BEGIN
    IF TG_OP<>'INSERT' THEN v_old:=(to_jsonb(OLD)->>v_col)::bigint; END IF;
    IF TG_OP<>'DELETE' THEN v_new:=(to_jsonb(NEW)->>v_col)::bigint; END IF;
    FOR v_row IN EXECUTE format('SELECT id,estado FROM sch_restaurante.%I WHERE id=$1 OR id=$2 ORDER BY id FOR UPDATE',TG_ARGV[0])
        USING v_old,v_new LOOP
        IF v_row.estado<>'BORRADOR' THEN RAISE EXCEPTION 'Documento % ya confirmado/emitido: detalles inmutables',v_row.id; END IF;
    END LOOP;
    RETURN CASE WHEN TG_OP='DELETE' THEN OLD ELSE NEW END;
END;
$$;
CREATE TRIGGER a_detalle_borrador BEFORE INSERT OR UPDATE OR DELETE ON compra_detalles
FOR EACH ROW EXECUTE FUNCTION fn_detalle_borrador('compras','compra_id');
CREATE TRIGGER a_detalle_borrador BEFORE INSERT OR UPDATE OR DELETE ON movimiento_inventario_detalles
FOR EACH ROW EXECUTE FUNCTION fn_detalle_borrador('movimientos_inventario','movimiento_id');
CREATE TRIGGER a_detalle_borrador BEFORE INSERT OR UPDATE OR DELETE ON comprobante_detalles
FOR EACH ROW EXECUTE FUNCTION fn_detalle_borrador('comprobantes','comprobante_id');

CREATE FUNCTION fn_validar_compra() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF TG_OP='INSERT' AND NEW.estado<>'BORRADOR' THEN RAISE EXCEPTION 'Crear compra en BORRADOR'; END IF;
    IF TG_OP='UPDATE' THEN
        IF OLD.estado<>'BORRADOR' THEN RAISE EXCEPTION 'Compra confirmada/cancelada inmutable'; END IF;
        IF NEW.estado='CONFIRMADA' AND NOT EXISTS (SELECT 1 FROM compra_detalles WHERE compra_id=NEW.id) THEN
            RAISE EXCEPTION 'La compra no tiene detalles';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_compra BEFORE INSERT OR UPDATE ON compras FOR EACH ROW EXECUTE FUNCTION fn_validar_compra();
CREATE TRIGGER a_no_borrar BEFORE DELETE ON compras FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE FUNCTION fn_validar_movimiento() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE r RECORD; v_stock NUMERIC; v_compra compra_detalles%ROWTYPE;
BEGIN
    PERFORM fn_exigir_read_committed();
    IF TG_OP='INSERT' AND NEW.estado<>'BORRADOR' THEN RAISE EXCEPTION 'Crear movimiento en BORRADOR y luego confirmar'; END IF;
    IF TG_OP='UPDATE' THEN
        IF OLD.estado='CONFIRMADO' THEN RAISE EXCEPTION 'Kardex confirmado inmutable; generar movimiento compensatorio'; END IF;
        IF NEW.estado='CONFIRMADO' THEN
            IF NOT EXISTS (SELECT 1 FROM movimiento_inventario_detalles WHERE movimiento_id=NEW.id) THEN
                RAISE EXCEPTION 'No hay detalles de inventario';
            END IF;
            IF NEW.tipo='COMPRA' AND NOT EXISTS (SELECT 1 FROM compras WHERE id=NEW.compra_id AND estado='CONFIRMADA') THEN
                RAISE EXCEPTION 'Confirmar la compra antes de confirmar su ingreso';
            END IF;
            IF NEW.detalle_pedido_id IS NOT NULL AND NOT EXISTS (
                SELECT 1 FROM detalles_pedido d JOIN pedidos p ON p.id=d.pedido_id
                JOIN almacenes a ON a.sucursal_id=p.sucursal_id
                WHERE d.id=NEW.detalle_pedido_id AND a.id=NEW.almacen_id) THEN
                RAISE EXCEPTION 'El almacen y el pedido deben pertenecer a la misma sucursal';
            END IF;
            -- Bloqueos en orden fijo; todas las confirmaciones pasan por estas filas.
            PERFORM a.id FROM articulos_almacen a
            WHERE a.almacen_id=NEW.almacen_id AND a.articulo_id IN (
                SELECT articulo_id FROM movimiento_inventario_detalles WHERE movimiento_id=NEW.id)
            ORDER BY a.articulo_id FOR UPDATE;
            FOR r IN SELECT d.*,a.controla_lote,a.controla_vencimiento,l.fecha_vencimiento,l.estado AS estado_lote
                FROM movimiento_inventario_detalles d JOIN articulos_inventario a ON a.id=d.articulo_id
                LEFT JOIN lotes l ON l.id=d.lote_id
                WHERE d.movimiento_id=NEW.id LOOP
                IF NEW.tipo IN ('CONSUMO','REVENTA') AND r.lote_id IS NOT NULL AND r.estado_lote<>'LIBERADO' THEN
                    RAISE EXCEPTION 'Lote % no liberado para consumo o venta',r.lote_id;
                END IF;
                IF r.controla_lote AND r.lote_id IS NULL THEN RAISE EXCEPTION 'Articulo % requiere lote',r.articulo_id; END IF;
                IF r.controla_vencimiento AND r.fecha_vencimiento IS NULL THEN RAISE EXCEPTION 'Articulo % requiere vencimiento',r.articulo_id; END IF;
                IF NEW.tipo='COMPRA' THEN
                    IF r.compra_detalle_id IS NULL THEN RAISE EXCEPTION 'Ingreso COMPRA requiere detalle origen'; END IF;
                    SELECT * INTO STRICT v_compra FROM compra_detalles WHERE id=r.compra_detalle_id;
                    IF v_compra.compra_id<>NEW.compra_id OR v_compra.lote_id IS DISTINCT FROM r.lote_id THEN
                        RAISE EXCEPTION 'Detalle/lote de compra incompatible con el ingreso';
                    END IF;
                END IF;
            END LOOP;
            IF NEW.tipo='COMPRA' AND EXISTS (
                SELECT 1 FROM compra_detalles cd WHERE cd.compra_id=NEW.compra_id
                AND cd.cantidad_base<>coalesce((SELECT sum(md.cantidad) FROM movimiento_inventario_detalles md
                     WHERE md.movimiento_id=NEW.id AND md.compra_detalle_id=cd.id),0)) THEN
                RAISE EXCEPTION 'El ingreso debe cubrir exactamente cada detalle de la compra';
            END IF;
            IF NEW.direccion=-1 THEN
                FOR r IN SELECT articulo_id,lote_id,sum(cantidad) AS cantidad
                    FROM movimiento_inventario_detalles WHERE movimiento_id=NEW.id GROUP BY articulo_id,lote_id LOOP
                    SELECT coalesce(sum(stock),0) INTO v_stock FROM v_stock_lotes
                    WHERE almacen_id=NEW.almacen_id AND articulo_id=r.articulo_id AND lote_id IS NOT DISTINCT FROM r.lote_id;
                    IF v_stock<r.cantidad THEN
                        RAISE EXCEPTION 'Stock insuficiente articulo %, lote %: disponible %, salida %',r.articulo_id,r.lote_id,v_stock,r.cantidad;
                    END IF;
                END LOOP;
            END IF;
            NEW.confirmado_en:=clock_timestamp();
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_movimiento BEFORE INSERT OR UPDATE ON movimientos_inventario
FOR EACH ROW EXECUTE FUNCTION fn_validar_movimiento();
CREATE TRIGGER a_no_borrar BEFORE DELETE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE FUNCTION fn_confirmar_movimiento(p_movimiento_id BIGINT) RETURNS void
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    -- Repetir esta llamada sobre el mismo documento es idempotente.
    UPDATE movimientos_inventario SET estado='CONFIRMADO'
    WHERE id=p_movimiento_id AND estado='BORRADOR';
    IF NOT FOUND AND NOT EXISTS (SELECT 1 FROM movimientos_inventario WHERE id=p_movimiento_id AND estado='CONFIRMADO') THEN
        RAISE EXCEPTION 'Movimiento % inexistente',p_movimiento_id;
    END IF;
END;
$$;

CREATE FUNCTION fn_siguiente_correlativo(p_serie_id BIGINT) RETURNS bigint
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_numero BIGINT;
BEGIN
    UPDATE series_comprobantes SET ultimo_correlativo=ultimo_correlativo+1
    WHERE id=p_serie_id AND es_activo AND administracion='LOCAL'
    RETURNING ultimo_correlativo INTO v_numero;
    IF NOT FOUND THEN RAISE EXCEPTION 'Serie inexistente, inactiva o administrada por proveedor'; END IF;
    RETURN v_numero;
END;
$$;

CREATE FUNCTION fn_validar_comprobante() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_pedido pedidos%ROWTYPE; v_total_documento NUMERIC; v_total_cuenta NUMERIC;
BEGIN
    SELECT p.* INTO STRICT v_pedido FROM pedidos p JOIN cuentas c ON c.pedido_id=p.id WHERE c.id=NEW.cuenta_id;
    IF v_pedido.sucursal_id<>NEW.sucursal_id OR v_pedido.moneda<>NEW.moneda THEN
        RAISE EXCEPTION 'Comprobante y cuenta deben tener la misma sucursal y moneda';
    END IF;
    IF NEW.tipo IN ('07','08') AND NOT EXISTS (
        SELECT 1 FROM comprobantes c WHERE c.id=NEW.comprobante_referencia_id
        AND c.cuenta_id=NEW.cuenta_id AND c.sucursal_id=NEW.sucursal_id
        AND c.moneda=NEW.moneda AND c.tipo IN ('01','03') AND c.estado='EMITIDO') THEN
        RAISE EXCEPTION 'La nota debe referenciar una factura/boleta emitida de la misma cuenta';
    END IF;
    IF TG_OP='INSERT' AND NEW.estado<>'BORRADOR' THEN RAISE EXCEPTION 'Crear comprobante en BORRADOR'; END IF;
    IF TG_OP='UPDATE' THEN
        IF OLD.estado='ANULADO' THEN RAISE EXCEPTION 'Comprobante anulado inmutable'; END IF;
        IF OLD.estado='EMITIDO' THEN
            IF NEW.estado<>'ANULADO' OR
               (to_jsonb(NEW)-ARRAY['estado','anulado_en','motivo_anulacion','actualizado_en','actualizado_por']) IS DISTINCT FROM
               (to_jsonb(OLD)-ARRAY['estado','anulado_en','motivo_anulacion','actualizado_en','actualizado_por']) THEN
                RAISE EXCEPTION 'Comprobante emitido: solo registrar anulacion fiscal; importes y datos son inmutables';
            END IF;
        ELSIF NEW.estado='ANULADO' THEN RAISE EXCEPTION 'No anular un borrador como si hubiera sido emitido';
        END IF;
        IF NEW.estado='EMITIDO' THEN
            IF NOT EXISTS (SELECT 1 FROM comprobante_detalles WHERE comprobante_id=NEW.id) THEN
                RAISE EXCEPTION 'Comprobante sin detalles';
            END IF;
            IF EXISTS (SELECT 1 FROM comprobante_detalles cd JOIN detalles_pedido dp ON dp.id=cd.detalle_pedido_id
                WHERE cd.comprobante_id=NEW.id AND dp.cuenta_id<>NEW.cuenta_id) THEN
                RAISE EXCEPTION 'Las lineas del comprobante deben corresponder a su cuenta';
            END IF;
            IF NEW.tipo IN ('01','03') THEN
                SELECT total INTO v_total_documento FROM v_comprobante_totales WHERE comprobante_id=NEW.id;
                SELECT total INTO v_total_cuenta FROM v_cuenta_totales WHERE cuenta_id=NEW.cuenta_id;
                IF v_total_documento<>v_total_cuenta THEN
                    RAISE EXCEPTION 'El comprobante principal debe cubrir el total de la cuenta';
                END IF;
            END IF;
            NEW.emitido_en:=coalesce(NEW.emitido_en,clock_timestamp());
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a_validar_comprobante BEFORE INSERT OR UPDATE ON comprobantes
FOR EACH ROW EXECUTE FUNCTION fn_validar_comprobante();
CREATE TRIGGER a_no_borrar BEFORE DELETE ON comprobantes FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();
CREATE UNIQUE INDEX uq_comprobante_principal_cuenta ON comprobantes(cuenta_id)
WHERE tipo IN ('01','03') AND estado='EMITIDO';
CREATE TRIGGER a_no_borrar BEFORE DELETE ON pedidos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();
CREATE TRIGGER a_no_borrar BEFORE DELETE ON cuentas FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();
CREATE TRIGGER a_inmutable BEFORE UPDATE OR DELETE ON auditoria_eventos FOR EACH ROW EXECUTE FUNCTION fn_solo_insertar();

-- 08B. POLITICA DE AUDITORIA, BAJA LOGICA Y ACTIVIDAD
-- El backend define app.usuario_id desde su autenticacion, dentro de BEGIN/COMMIT.
-- No hay valor por defecto ni sustitucion silenciosa por el usuario tecnico.
CREATE FUNCTION fn_usuario_actor() RETURNS bigint
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_texto TEXT; v_id BIGINT;
BEGIN
    v_texto:=nullif(current_setting('app.usuario_id',true),'');
    IF v_texto IS NULL OR v_texto !~ '^[0-9]+$' THEN
        RAISE EXCEPTION 'Establecer app.usuario_id al usuario autenticado dentro de la transaccion';
    END IF;
    v_id:=v_texto::bigint;
    PERFORM id FROM usuarios WHERE id=v_id AND es_activo FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION 'Actor de auditoria inexistente o inactivo'; END IF;
    RETURN v_id;
END;
$$;

CREATE FUNCTION fn_metadatos_fila() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_actor BIGINT:=fn_usuario_actor(); v_ahora TIMESTAMPTZ:=clock_timestamp();
    v_mutable BOOLEAN:=TG_ARGV[0]::boolean; v_soft BOOLEAN:=TG_ARGV[1]::boolean;
BEGIN
    IF TG_OP='INSERT' THEN
        -- La autoria depende del contexto confiable, nunca del ID enviado por UI.
        NEW.creado_por:=v_actor;
        NEW.creado_en:=v_ahora;
    ELSE
        IF NEW.creado_por IS DISTINCT FROM OLD.creado_por OR NEW.creado_en IS DISTINCT FROM OLD.creado_en THEN
            RAISE EXCEPTION 'La autoria y fecha originales de creacion son inmutables';
        END IF;
    END IF;
    IF v_mutable THEN
        NEW.actualizado_por:=v_actor;
        NEW.actualizado_en:=v_ahora;
    END IF;
    IF v_soft THEN
        IF TG_OP='INSERT' THEN
            IF NOT NEW.es_activo THEN
                IF NEW.motivo_eliminacion IS NULL OR btrim(NEW.motivo_eliminacion)='' THEN
                    RAISE EXCEPTION 'Una baja logica requiere motivo_eliminacion';
                END IF;
                NEW.eliminado_en:=v_ahora; NEW.eliminado_por:=v_actor;
            ELSIF NEW.eliminado_en IS NOT NULL OR NEW.eliminado_por IS NOT NULL OR NEW.motivo_eliminacion IS NOT NULL THEN
                RAISE EXCEPTION 'Un registro activo no admite marcadores de eliminacion';
            END IF;
        ELSIF OLD.es_activo AND NOT NEW.es_activo THEN
            IF NEW.motivo_eliminacion IS NULL OR btrim(NEW.motivo_eliminacion)='' THEN
                RAISE EXCEPTION 'Una baja logica requiere motivo_eliminacion';
            END IF;
            NEW.eliminado_en:=v_ahora; NEW.eliminado_por:=v_actor;
        ELSIF NOT OLD.es_activo AND NEW.es_activo THEN
            -- El evento completo de la baja anterior permanece en auditoria_eventos.
            NEW.eliminado_en:=NULL; NEW.eliminado_por:=NULL; NEW.motivo_eliminacion:=NULL;
        ELSIF (NEW.eliminado_en,NEW.eliminado_por,NEW.motivo_eliminacion) IS DISTINCT FROM
              (OLD.eliminado_en,OLD.eliminado_por,OLD.motivo_eliminacion) THEN
            RAISE EXCEPTION 'Los marcadores de baja solo cambian al eliminar/restaurar el registro';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE FUNCTION fn_referencias_activas() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_arg TEXT; v_campo TEXT; v_tabla TEXT; v_id BIGINT; v_valido BOOLEAN;
    v_new JSONB:=to_jsonb(NEW); v_old JSONB; v_reactivar BOOLEAN:=FALSE;
BEGIN
    IF TG_OP='UPDATE' THEN
        v_old:=to_jsonb(OLD);
        v_reactivar:=coalesce((v_new->>'es_activo')::boolean,FALSE)
            AND NOT coalesce((v_old->>'es_activo')::boolean,TRUE);
    END IF;
    -- No exigir padres activos para registrar una BAJA de una asignacion vieja.
    IF v_new ? 'es_activo' AND NOT (v_new->>'es_activo')::boolean THEN RETURN NEW; END IF;
    FOREACH v_arg IN ARRAY TG_ARGV LOOP
        v_campo:=split_part(v_arg,'|',1); v_tabla:=split_part(v_arg,'|',2);
        v_id:=(v_new->>v_campo)::bigint;
        IF v_id IS NOT NULL AND (TG_OP='INSERT' OR v_reactivar OR
            (v_new->>v_campo) IS DISTINCT FROM (v_old->>v_campo)) THEN
            v_valido:=NULL;
            EXECUTE format('SELECT TRUE FROM sch_restaurante.%I WHERE id=$1 AND es_activo FOR SHARE',v_tabla)
            INTO v_valido USING v_id;
            IF v_valido IS DISTINCT FROM TRUE THEN
                RAISE EXCEPTION 'Referencia %=%: maestro % inexistente o dado de baja',v_campo,v_id,v_tabla;
            END IF;
        END IF;
    END LOOP;
    RETURN NEW;
END;
$$;

CREATE FUNCTION fn_proteger_usuario_sistema() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF TG_OP='INSERT' AND NEW.es_sistema THEN RAISE EXCEPTION 'El actor tecnico solo se crea durante la instalacion'; END IF;
    IF TG_OP='UPDATE' THEN
        IF OLD.es_sistema OR NEW.es_sistema<>OLD.es_sistema THEN
            RAISE EXCEPTION 'El actor tecnico y el tipo de usuario son inmutables';
        END IF;
        IF NOT NEW.es_activo AND NEW.id=fn_usuario_actor() THEN
            RAISE EXCEPTION 'La baja del usuario debe realizarla otro administrador activo';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a2_proteger_usuario_sistema BEFORE INSERT OR UPDATE ON usuarios
FOR EACH ROW EXECUTE FUNCTION fn_proteger_usuario_sistema();

CREATE FUNCTION fn_no_roles_sistema() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM usuarios WHERE id=NEW.usuario_id AND es_sistema) THEN
        RAISE EXCEPTION 'El actor tecnico no puede recibir roles ni acceso interactivo a sucursales';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a2_no_roles_sistema BEFORE INSERT OR UPDATE ON usuarios_roles
FOR EACH ROW EXECUTE FUNCTION fn_no_roles_sistema();
CREATE TRIGGER a2_no_roles_sistema BEFORE INSERT OR UPDATE ON usuarios_sucursales
FOR EACH ROW EXECUTE FUNCTION fn_no_roles_sistema();

-- Evitar que la baja fisica de un maestro oculte operaciones abiertas.
-- No hay cascada de borrado. Los historicos siguen referenciando el mismo ID.
CREATE FUNCTION fn_validar_baja_operativa() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF NOT OLD.es_activo OR NEW.es_activo THEN RETURN NEW; END IF;
    IF TG_TABLE_NAME='mesas' AND EXISTS (
        SELECT 1 FROM pedidos WHERE mesa_id=OLD.id AND cerrado_en IS NULL) THEN
        RAISE EXCEPTION 'Cerrar el pedido de la mesa antes de darla de baja';
    ELSIF TG_TABLE_NAME='cajas' AND EXISTS (
        SELECT 1 FROM sesiones_caja WHERE caja_id=OLD.id AND estado='ABIERTA') THEN
        RAISE EXCEPTION 'Cerrar la sesion de caja antes de darla de baja';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER a2_baja_operativa BEFORE UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_validar_baja_operativa();
CREATE TRIGGER a2_baja_operativa BEFORE UPDATE ON cajas FOR EACH ROW EXECUTE FUNCTION fn_validar_baja_operativa();

CREATE VIEW v_permisos_efectivos AS
SELECT DISTINCT u.id AS usuario_id,us.sucursal_id,p.codigo AS permiso
FROM usuarios u JOIN usuarios_roles ur ON ur.usuario_id=u.id AND ur.es_activo
JOIN roles r ON r.id=ur.rol_id AND r.es_activo
JOIN roles_permisos rp ON rp.rol_id=r.id AND rp.es_activo
JOIN permisos p ON p.id=rp.permiso_id AND p.es_activo
JOIN usuarios_sucursales us ON us.usuario_id=u.id AND us.es_activo
JOIN sucursales s ON s.id=us.sucursal_id AND s.es_activo
JOIN negocio n ON n.id=s.negocio_id AND n.es_activo
WHERE u.es_activo AND NOT u.es_sistema;
COMMENT ON VIEW v_permisos_efectivos IS 'Permisos efectivos: deben estar activos todos los eslabones. No es un mecanismo PostgreSQL RLS.';

CREATE VIEW v_carta_disponible AS
SELECT ps.sucursal_id,p.id AS producto_id,p.codigo,p.nombre,p.tipo,p.categoria_id,
       coalesce(ps.precio_venta,p.precio_venta) AS precio_venta,
       p.afectacion_igv,p.impuesto_id,ps.estacion_id
FROM productos p JOIN categorias c ON c.id=p.categoria_id AND c.es_activo
JOIN productos_sucursal ps ON ps.producto_id=p.id AND ps.es_activo AND ps.disponible
JOIN sucursales s ON s.id=ps.sucursal_id AND s.es_activo
JOIN negocio n ON n.id=s.negocio_id AND n.es_activo
WHERE p.es_activo;
COMMENT ON VIEW v_carta_disponible IS 'Selector operativo; no usar para reconstruir ventas antiguas.';

-- MATRIZ DE APLICACION (todas las 49 tablas conservan creado_en/creado_por)
-- AUDITORIA COMPLETA + BAJA CON TRES MARCADORES (9):
-- sucursales, usuarios, empleados, clientes, proveedores, articulos_inventario,
-- almacenes, productos, cajas.
-- ACTIVIDAD SIMPLE CON es_activo, SIN MARCADORES DE ELIMINACION (17):
-- negocio, roles, permisos, roles_permisos, usuarios_roles, usuarios_sucursales,
-- unidades_medida, categorias, impuestos, estaciones_preparacion,
-- presentaciones_articulo, articulos_almacen, productos_sucursal, zonas,
-- mesas, metodos_pago, series_comprobantes.
-- AUDITORIA COMPLETA, SIN BAJA LOGICA (20):
-- tasas_impuesto, lotes, recetas, receta_detalles, reservas, pedidos, cuentas,
-- detalles_pedido, comandas, comanda_detalles, pedidos_delivery, sesiones_caja,
-- pagos, compras, compra_detalles, movimientos_inventario,
-- movimiento_inventario_detalles, comprobantes, comprobante_detalles,
-- envios_comprobante.
-- SOLO CREACION / INMUTABLES (3):
-- movimientos_caja, devoluciones_pago, auditoria_eventos.
--
-- ESTADO DE NEGOCIO:
-- lotes: LIBERADO/BLOQUEADO/RETIRADO (un lote bloqueado sigue teniendo stock).
-- reservas, pedidos, cuentas, detalles_pedido, comanda_detalles,
-- pedidos_delivery, sesiones_caja, pagos, compras, movimientos_inventario,
-- comprobantes y envios_comprobante: mantienen sus estados tipados por CHECK.
-- comandas: estado por sus lineas; no duplicar un estado de cabecera sin regla.
-- compra_detalles, movimiento_inventario_detalles, comprobante_detalles:
-- editabilidad derivada del estado de su cabecera, sin es_activo individual.
-- recetas/tasas: conservan intervalos de vigencia y versiones, sin un booleano
-- que borre vigencias historicas. receta_detalles depende de su receta.
-- Maestros simples no necesitan un estado adicional ACTIVO/INACTIVO que duplique
-- es_activo. Para usuarios, bloqueo temporal no implica borrado logico.

-- 09. TRIGGERS AUTOMATICOS DE AUTORIA, BAJA LOGICA Y TRAZA

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON negocio FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON negocio FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON negocio FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON sucursales FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON sucursales FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON sucursales FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON roles FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON roles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON permisos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON permisos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON permisos FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON roles_permisos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON roles_permisos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON roles_permisos FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON usuarios FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON usuarios FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON usuarios FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON usuarios_roles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON usuarios_roles FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON usuarios_roles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON usuarios_sucursales FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON usuarios_sucursales FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON usuarios_sucursales FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON empleados FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON empleados FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON empleados FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON clientes FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON clientes FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON clientes FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON proveedores FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON proveedores FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON proveedores FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON unidades_medida FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON unidades_medida FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON unidades_medida FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON categorias FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON categorias FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON categorias FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON impuestos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON impuestos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON impuestos FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON tasas_impuesto FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON tasas_impuesto FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON estaciones_preparacion FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON estaciones_preparacion FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON estaciones_preparacion FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON articulos_inventario FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON articulos_inventario FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON articulos_inventario FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON presentaciones_articulo FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON presentaciones_articulo FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON presentaciones_articulo FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON almacenes FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON almacenes FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON almacenes FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON articulos_almacen FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON articulos_almacen FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON articulos_almacen FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON lotes FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON lotes FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON productos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON productos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON productos FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON productos_sucursal FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON productos_sucursal FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON productos_sucursal FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON recetas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON recetas FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON receta_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON receta_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON zonas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON zonas FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON zonas FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON reservas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON reservas FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON pedidos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON pedidos FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON cuentas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON cuentas FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON detalles_pedido FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON detalles_pedido FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON comandas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON comandas FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON comanda_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON comanda_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON pedidos_delivery FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON pedidos_delivery FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON metodos_pago FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON metodos_pago FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON metodos_pago FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON cajas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON cajas FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON cajas FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON sesiones_caja FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON sesiones_caja FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT ON movimientos_caja FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('false','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON movimientos_caja FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON pagos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON pagos FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT ON devoluciones_pago FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('false','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON devoluciones_pago FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON compras FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON compras FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON compra_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON compra_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON movimiento_inventario_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON movimiento_inventario_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON comprobantes FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON comprobantes FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON comprobante_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON comprobante_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON envios_comprobante FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON envios_comprobante FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT ON auditoria_eventos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('false','false');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON sucursales FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('negocio_id|negocio');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON roles_permisos FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('rol_id|roles','permiso_id|permisos');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON usuarios_roles FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('usuario_id|usuarios','rol_id|roles');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON usuarios_sucursales FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('usuario_id|usuarios','sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON empleados FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('usuario_id|usuarios');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON tasas_impuesto FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('impuesto_id|impuestos');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON estaciones_preparacion FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON articulos_inventario FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('unidad_base_id|unidades_medida');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON presentaciones_articulo FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('articulo_id|articulos_inventario');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON almacenes FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON articulos_almacen FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('almacen_id|almacenes','articulo_id|articulos_inventario');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON lotes FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('articulo_id|articulos_inventario');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON productos FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('categoria_id|categorias','articulo_id|articulos_inventario','impuesto_id|impuestos');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON productos_sucursal FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('producto_id|productos','sucursal_id|sucursales','estacion_id|estaciones_preparacion');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON recetas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('producto_id|productos');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON receta_detalles FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('articulo_id|articulos_inventario');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON zonas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','zona_id|zonas');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON reservas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','mesa_id|mesas','cliente_id|clientes');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON pedidos FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','mesa_id|mesas','cliente_id|clientes','mesero_id|empleados');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON cuentas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('cliente_id|clientes');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON detalles_pedido FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('producto_id|productos');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON comandas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','estacion_id|estaciones_preparacion');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON pedidos_delivery FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('repartidor_id|empleados');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON cajas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON sesiones_caja FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('caja_id|cajas','sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON pagos FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('metodo_pago_id|metodos_pago');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON devoluciones_pago FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('metodo_pago_id|metodos_pago');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON compras FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','almacen_id|almacenes','proveedor_id|proveedores');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON compra_detalles FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('articulo_id|articulos_inventario','presentacion_id|presentaciones_articulo');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('almacen_id|almacenes');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON movimiento_inventario_detalles FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('almacen_id|almacenes','articulo_id|articulos_inventario');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

-- 10. INDICES DE ACCESO Y CLAVES FORANEAS

CREATE INDEX idx_reservas_sucursal_inicio ON reservas(sucursal_id,inicio);

CREATE INDEX idx_pedidos_sucursal_fecha ON pedidos(sucursal_id,abierto_en);

CREATE INDEX idx_pedidos_cliente ON pedidos(cliente_id);

CREATE INDEX idx_pedidos_pendientes ON pedidos(sucursal_id,estado) WHERE cerrado_en IS NULL;

CREATE INDEX idx_detalles_cuenta ON detalles_pedido(cuenta_id);

CREATE INDEX idx_detalles_producto ON detalles_pedido(producto_id);

CREATE INDEX idx_comandas_pedido ON comandas(pedido_id,enviado_en);

CREATE INDEX idx_comanda_detalles_pedido ON comanda_detalles(detalle_pedido_id);

CREATE INDEX idx_comanda_detalles_estado ON comanda_detalles(estado);

CREATE INDEX idx_pagos_cuenta ON pagos(cuenta_id,estado);

CREATE INDEX idx_pagos_sesion ON pagos(sesion_caja_id,estado);

CREATE INDEX idx_devoluciones_pago ON devoluciones_pago(pago_id);

CREATE INDEX idx_devoluciones_sesion ON devoluciones_pago(sesion_caja_id);

CREATE INDEX idx_movimientos_caja_sesion ON movimientos_caja(sesion_caja_id);

CREATE INDEX idx_compras_fecha ON compras(sucursal_id,fecha_documento);

CREATE INDEX idx_receta_detalles_articulo ON receta_detalles(articulo_id);

CREATE INDEX idx_productos_categoria ON productos(categoria_id);

CREATE INDEX idx_movimientos_stock ON movimientos_inventario(almacen_id,estado,fecha_movimiento);

CREATE INDEX idx_movimientos_pedido ON movimientos_inventario(detalle_pedido_id);

CREATE INDEX idx_movimientos_referencia ON movimientos_inventario(movimiento_referencia_id);

CREATE INDEX idx_kardex_articulo_lote ON movimiento_inventario_detalles(almacen_id,articulo_id,lote_id);

CREATE INDEX idx_kardex_compra_detalle ON movimiento_inventario_detalles(compra_detalle_id);

CREATE INDEX idx_comprobantes_cuenta ON comprobantes(cuenta_id);

CREATE INDEX idx_comprobantes_fecha ON comprobantes(sucursal_id,fecha_emision);

CREATE INDEX idx_comprobantes_referencia ON comprobantes(comprobante_referencia_id);

CREATE INDEX idx_auditoria_registro ON auditoria_eventos(tabla,registro_id,creado_en);

CREATE INDEX idx_auditoria_usuario ON auditoria_eventos(creado_por,creado_en);

CREATE INDEX idx_productos_catalogo_activos ON productos(categoria_id,nombre) WHERE es_activo;

CREATE INDEX idx_clientes_nombre_activos ON clientes(nombre_razon_social) WHERE es_activo;

-- 11. SEMILLAS TECNICAS (SIN DATOS FICTICIOS NI USUARIOS ADMIN)
INSERT INTO unidades_medida(codigo,nombre,dimension,factor_referencia,codigo_sunat) VALUES
 ('KG','Kilogramo','MASA',1,'KGM'),('G','Gramo','MASA',0.001,'GRM'),
 ('L','Litro','VOLUMEN',1,'LTR'),('ML','Mililitro','VOLUMEN',0.001,'MLT'),
 ('UND','Unidad','UNIDAD',1,'NIU');
INSERT INTO impuestos(codigo,nombre,descripcion) VALUES
 ('IGV_TOTAL','IGV e IPM','Configurar la tasa total aplicable y su vigencia; no asumir beneficio por ser restaurante.');
INSERT INTO metodos_pago(codigo,nombre,es_efectivo,requiere_referencia) VALUES
 ('EFECTIVO','Efectivo',true,false),('TARJETA','Tarjeta',false,true),
 ('TRANSFERENCIA','Transferencia',false,true),('YAPE','Yape',false,true),
 ('PLIN','Plin',false,true),('OTROS','Otros',false,false);
INSERT INTO roles(codigo,nombre) VALUES
 ('ADMIN','Administrador'),('CAJERO','Cajero'),('MESERO','Mesero'),
 ('COCINA','Cocina'),('ALMACEN','Almacen');
INSERT INTO permisos(codigo,descripcion) VALUES
 ('PEDIDO_CREAR','Registrar pedidos'),('PEDIDO_ANULAR','Anular pedidos'),
 ('DESCUENTO_AUTORIZAR','Autorizar descuentos'),('CAJA_OPERAR','Abrir, cobrar y cerrar caja'),
 ('PAGO_DEVOLVER','Autorizar devoluciones'),('COMPRA_REGISTRAR','Registrar compras'),
 ('INVENTARIO_AJUSTAR','Confirmar ajustes de inventario'),('COCINA_OPERAR','Gestionar preparacion'),
 ('COMPROBANTE_EMITIR','Emitir comprobantes'),('USUARIO_ADMINISTRAR','Gestionar usuarios y roles');
INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p WHERE r.codigo='ADMIN';
-- Los demas roles quedan sin permisos hasta que el administrador los configure.

-- 12. REGLAS DEL SERVICIO / PUESTA EN MARCHA
-- Configurar negocio, sucursal, usuarios/roles, caja, almacen, tasas y series.
-- Conceder solo los permisos necesarios al rol PostgreSQL del backend; no usar
-- el propietario del esquema como usuario de aplicacion. Los permisos de esta
-- base son funcionales: no sustituyen GRANT ni seguridad de endpoints.
-- AUDITORIA: abrir transaccion y establecer SET LOCAL app.usuario_id al usuario
-- autenticado. Se rechaza escritura sin actor o con actor inactivo. El backend
-- asigna contexto confiable, nunca acepta ese ID directamente del formulario.
-- Primera alta humana, por el instalador: BEGIN; SELECT set_config('app.usuario_id',
-- (SELECT id::text FROM sch_restaurante.usuarios WHERE es_sistema),true);
-- INSERT INTO sch_restaurante.usuarios(username,password_hash) VALUES (...);
-- asignar roles y sucursales al usuario creado, sin dar roles a __sistema__;
-- COMMIT. El login debe exigir es_activo AND NOT es_sistema y validar el hash.
-- El actor tecnico no autentica: no tiene password ni roles y no es modificable.
-- BAJA DE MAESTRO PRINCIPAL: UPDATE productos SET es_activo=FALSE,
-- motivo_eliminacion='motivo' WHERE id=...;
-- DESACTIVAR CATALOGO/RELACION: UPDATE roles_permisos SET es_activo=FALSE WHERE id=...;
-- REACTIVAR: UPDATE tabla SET es_activo=TRUE WHERE id=...; el historial de la
-- baja queda en auditoria_eventos aunque se limpien los marcadores de baja.
-- Las claves de negocio siguen siendo unicas aun despues de la baja.
-- El FK garantiza existencia, no actividad. Los triggers comprueban actividad
-- al insertar, cambiar referencias o reactivar; no rompen referencias antiguas.
-- Desactivar un padre no desactiva sus hijos en cascada. Los selectores y la
-- autorizacion deben comprobar la cadena de maestros activos (usar vistas).
--
-- PEDIDOS: validar mesa habilitada, transiciones de atencion, capacidad y
-- usuario/sucursal. Abrir pedido+cuenta en una transaccion. No mover cabeceras
-- entre sucursales tras registrar operaciones. Para cambiar mesa, actualizar
-- mesa_id en transaccion y conservar auditoria.
-- COCINA: impedir cantidades enviadas por encima de la linea, consumir receta
-- historica y usar una clave idempotente estable por evento de preparacion.
-- Una anulacion postpreparacion puede ser MERMA, no una devolucion automatica.
-- PAGOS: moneda se toma del pedido y se verifica con la sesion. Confirmar solo
-- tras verificar la operacion. Registrar devoluciones cuando realmente ocurren.
-- Las propinas no se incluyen en el alcance; no mezclarlas con monto aplicado.
-- COMPRAS: confirmar documento y generar movimiento COMPRA con cada detalle
-- en la misma transaccion. Este alcance tiene una recepcion por compra.
-- STOCK: capturar costo unitario segun politica de valorizacion (p.ej. promedio
-- ponderado) bajo el mismo bloqueo usado al confirmar. La BD no determina por
-- si sola el costo de salida, impuestos recuperables ni gastos capitalizables.
-- Transferencias: salida+entrada, mismo articulo/lote/cantidad y costo, ambos
-- documentos confirmados en una transaccion; el backend valida equivalencia.
-- Nunca habilitar stock negativo. fn_confirmar_movimiento bloquea articulos
-- en orden y valida cantidades disponibles; manejar reintentos por deadlock.
-- VENCIMIENTOS: confirmar presencia de fecha para articulos que la exigen;
-- politica FEFO, alertas y bloqueo de lotes vencidos corresponden al servicio.
-- RECETAS: no modificar una composicion que ya fue usada; crear nueva version.
-- COMPROBANTES: copiar datos historicos, validar tipo de receptor, UBL/catalogos,
-- redondeo fiscal, reglas de identificacion y no duplicar facturacion de cuenta.
-- fn_siguiente_correlativo solo para series locales; si un proveedor administra
-- correlativos, usar el numero devuelto por este. Nunca ejecutar ambos caminos.
-- EMISION: es una integracion externa, no una llamada dentro de un trigger.
-- Registrar envios/reintentos. ANULADO solo tras resolver el flujo fiscal.
-- Tasas/medios de pago usados deben inactivarse o versionarse, no reinterpretarse.
-- Alcance tributario: operaciones onerosas 10/20/30; otros tributos y operaciones
-- especiales requieren ampliar columnas/reglas del adaptador antes de usarlos.
--
-- Consultas de uso:
-- SELECT * FROM sch_restaurante.v_pedido_totales;
-- SELECT * FROM sch_restaurante.v_cuenta_totales;
-- SELECT * FROM sch_restaurante.v_existencias;
-- SELECT * FROM sch_restaurante.v_kardex;
-- SELECT * FROM sch_restaurante.v_caja_resumen;
-- SELECT * FROM sch_restaurante.v_comprobante_totales;
-- 13. SIGNIFICADO EXPLICITO DE ACTIVIDAD, AUDITORIA Y ESTADO
COMMENT ON COLUMN negocio.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN sucursales.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN roles.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN permisos.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN roles_permisos.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN usuarios.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN usuarios_roles.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN usuarios_sucursales.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN empleados.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN clientes.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN proveedores.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN unidades_medida.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN categorias.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN impuestos.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN estaciones_preparacion.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN articulos_inventario.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN presentaciones_articulo.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN almacenes.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN articulos_almacen.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN lotes.estado IS 'Estado del proceso de negocio: LIBERADO, BLOQUEADO, RETIRADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN productos.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN productos_sucursal.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN zonas.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN mesas.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN reservas.estado IS 'Estado del proceso de negocio: PENDIENTE, CONFIRMADA, ATENDIDA, CANCELADA, NO_ASISTIO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN pedidos.estado IS 'Estado del proceso de negocio: REGISTRADO, PREPARACION, LISTO, ENTREGADO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN cuentas.estado IS 'Estado del proceso de negocio: ABIERTA, CERRADA, CANCELADA. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN detalles_pedido.estado IS 'Estado del proceso de negocio: ACTIVO, ANULADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN comanda_detalles.estado IS 'Estado del proceso de negocio: PENDIENTE, PREPARACION, LISTO, ENTREGADO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN pedidos_delivery.estado IS 'Estado del proceso de negocio: PENDIENTE, ASIGNADO, EN_RUTA, ENTREGADO, FALLIDO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN metodos_pago.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN cajas.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';
COMMENT ON COLUMN sesiones_caja.estado IS 'Estado del proceso de negocio: ABIERTA, CERRADA. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN pagos.estado IS 'Estado del proceso de negocio: PENDIENTE, CONFIRMADO, FALLIDO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN compras.estado IS 'Estado del proceso de negocio: BORRADOR, CONFIRMADA, CANCELADA. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN movimientos_inventario.estado IS 'Estado del proceso de negocio: BORRADOR, CONFIRMADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN series_comprobantes.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';
COMMENT ON COLUMN comprobantes.estado IS 'Estado del proceso de negocio: BORRADOR, EMITIDO, ANULADO. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMENT ON COLUMN envios_comprobante.estado IS 'Estado del proceso de negocio: PENDIENTE, ENVIADO, ACEPTADO, ACEPTADO_OBSERVADO, RECHAZADO, ERROR, BAJA_ACEPTADA. Independiente de eliminacion logica; validar transiciones en el servicio.';
COMMIT;
