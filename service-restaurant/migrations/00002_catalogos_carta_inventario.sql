-- Migracion 00002: Catalogos, carta, recetas e inventario maestro (I2)
-- Origen: docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql (V4),
-- dividido por dominio sin cambiar las sentencias. Requiere PostgreSQL 18.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
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

CREATE INDEX idx_receta_detalles_articulo ON receta_detalles(articulo_id);

CREATE INDEX idx_productos_categoria ON productos(categoria_id);

CREATE INDEX idx_productos_catalogo_activos ON productos(categoria_id,nombre) WHERE es_activo;

SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO unidades_medida(codigo,nombre,dimension,factor_referencia,codigo_sunat) VALUES
 ('KG','Kilogramo','MASA',1,'KGM'),('G','Gramo','MASA',0.001,'GRM'),
 ('L','Litro','VOLUMEN',1,'LTR'),('ML','Mililitro','VOLUMEN',0.001,'MLT'),
 ('UND','Unidad','UNIDAD',1,'NIU');

INSERT INTO impuestos(codigo,nombre,descripcion) VALUES
 ('IGV_TOTAL','IGV e IPM','Configurar la tasa total aplicable y su vigencia; no asumir beneficio por ser restaurante.');

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

-- +goose Down
DROP VIEW IF EXISTS sch_restaurante.v_carta_disponible;
DROP TABLE IF EXISTS sch_restaurante.receta_detalles CASCADE;
DROP TABLE IF EXISTS sch_restaurante.recetas CASCADE;
DROP TABLE IF EXISTS sch_restaurante.productos_sucursal CASCADE;
DROP TABLE IF EXISTS sch_restaurante.productos CASCADE;
DROP TABLE IF EXISTS sch_restaurante.lotes CASCADE;
DROP TABLE IF EXISTS sch_restaurante.articulos_almacen CASCADE;
DROP TABLE IF EXISTS sch_restaurante.almacenes CASCADE;
DROP TABLE IF EXISTS sch_restaurante.presentaciones_articulo CASCADE;
DROP TABLE IF EXISTS sch_restaurante.articulos_inventario CASCADE;
DROP TABLE IF EXISTS sch_restaurante.estaciones_preparacion CASCADE;
DROP TABLE IF EXISTS sch_restaurante.tasas_impuesto CASCADE;
DROP TABLE IF EXISTS sch_restaurante.impuestos CASCADE;
DROP TABLE IF EXISTS sch_restaurante.categorias CASCADE;
DROP TABLE IF EXISTS sch_restaurante.unidades_medida CASCADE;
