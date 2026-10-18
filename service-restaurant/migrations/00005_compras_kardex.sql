-- Migracion 00005: Compras, existencias y kardex (I5)
-- Origen: docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql (V4),
-- dividido por dominio sin cambiar las sentencias. Requiere PostgreSQL 18.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
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

ALTER TABLE compras ADD CONSTRAINT fk_compras_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE compras ADD CONSTRAINT fk_compras_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE compra_detalles ADD CONSTRAINT fk_compra_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE compra_detalles ADD CONSTRAINT fk_compra_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimientos_inventario ADD CONSTRAINT fk_movimientos_inventario_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimientos_inventario ADD CONSTRAINT fk_movimientos_inventario_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimiento_inventario_detalles ADD CONSTRAINT fk_movimiento_inventario_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE movimiento_inventario_detalles ADD CONSTRAINT fk_movimiento_inventario_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

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

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_detalle_borrador BEFORE INSERT OR UPDATE OR DELETE ON compra_detalles
FOR EACH ROW EXECUTE FUNCTION fn_detalle_borrador('compras','compra_id');

CREATE TRIGGER a_detalle_borrador BEFORE INSERT OR UPDATE OR DELETE ON movimiento_inventario_detalles
FOR EACH ROW EXECUTE FUNCTION fn_detalle_borrador('movimientos_inventario','movimiento_id');

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_compra BEFORE INSERT OR UPDATE ON compras FOR EACH ROW EXECUTE FUNCTION fn_validar_compra();

CREATE TRIGGER a_no_borrar BEFORE DELETE ON compras FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_movimiento BEFORE INSERT OR UPDATE ON movimientos_inventario
FOR EACH ROW EXECUTE FUNCTION fn_validar_movimiento();

CREATE TRIGGER a_no_borrar BEFORE DELETE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON compras FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON compras FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON compra_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON compra_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON movimiento_inventario_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON movimiento_inventario_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON compras FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','almacen_id|almacenes','proveedor_id|proveedores');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON compra_detalles FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('articulo_id|articulos_inventario','presentacion_id|presentaciones_articulo');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON movimientos_inventario FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('almacen_id|almacenes');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON movimiento_inventario_detalles FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('almacen_id|almacenes','articulo_id|articulos_inventario');

CREATE INDEX idx_compras_fecha ON compras(sucursal_id,fecha_documento);

CREATE INDEX idx_movimientos_stock ON movimientos_inventario(almacen_id,estado,fecha_movimiento);

CREATE INDEX idx_movimientos_pedido ON movimientos_inventario(detalle_pedido_id);

CREATE INDEX idx_movimientos_referencia ON movimientos_inventario(movimiento_referencia_id);

CREATE INDEX idx_kardex_articulo_lote ON movimiento_inventario_detalles(almacen_id,articulo_id,lote_id);

CREATE INDEX idx_kardex_compra_detalle ON movimiento_inventario_detalles(compra_detalle_id);

COMMENT ON COLUMN compras.estado IS 'Estado del proceso de negocio: BORRADOR, CONFIRMADA, CANCELADA. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN movimientos_inventario.estado IS 'Estado del proceso de negocio: BORRADOR, CONFIRMADO. Independiente de eliminacion logica; validar transiciones en el servicio.';

-- +goose Down
DROP VIEW IF EXISTS sch_restaurante.v_compra_totales;
DROP VIEW IF EXISTS sch_restaurante.v_kardex;
DROP VIEW IF EXISTS sch_restaurante.v_existencias;
DROP VIEW IF EXISTS sch_restaurante.v_stock_lotes;
DROP TABLE IF EXISTS sch_restaurante.movimiento_inventario_detalles CASCADE;
DROP TABLE IF EXISTS sch_restaurante.movimientos_inventario CASCADE;
DROP TABLE IF EXISTS sch_restaurante.compra_detalles CASCADE;
DROP TABLE IF EXISTS sch_restaurante.compras CASCADE;
DROP FUNCTION IF EXISTS sch_restaurante.fn_confirmar_movimiento(p_movimiento_id BIGINT);
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_movimiento();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_compra();
DROP FUNCTION IF EXISTS sch_restaurante.fn_detalle_borrador();
