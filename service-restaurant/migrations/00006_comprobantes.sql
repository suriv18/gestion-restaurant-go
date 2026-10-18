-- Migracion 00006: Comprobantes electronicos y trazabilidad de emision (I7)
-- Origen: docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql (V4),
-- dividido por dominio sin cambiar las sentencias. Requiere PostgreSQL 18.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
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

ALTER TABLE series_comprobantes ADD CONSTRAINT fk_series_comprobantes_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE series_comprobantes ADD CONSTRAINT fk_series_comprobantes_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comprobantes ADD CONSTRAINT fk_comprobantes_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comprobantes ADD CONSTRAINT fk_comprobantes_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comprobante_detalles ADD CONSTRAINT fk_comprobante_detalles_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE comprobante_detalles ADD CONSTRAINT fk_comprobante_detalles_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE envios_comprobante ADD CONSTRAINT fk_envios_comprobante_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

ALTER TABLE envios_comprobante ADD CONSTRAINT fk_envios_comprobante_editor FOREIGN KEY (actualizado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

CREATE VIEW v_comprobante_totales AS
SELECT c.id AS comprobante_id,c.tipo,c.serie,c.numero,c.moneda,c.estado,
 coalesce(sum(d.base_imponible) FILTER (WHERE d.afectacion_igv='10'),0) AS gravado,
 coalesce(sum(d.base_imponible) FILTER (WHERE d.afectacion_igv='20'),0) AS exonerado,
 coalesce(sum(d.base_imponible) FILTER (WHERE d.afectacion_igv='30'),0) AS inafecto,
 coalesce(sum(d.importe_total-d.base_imponible),0) AS impuesto,
 coalesce(sum(d.importe_total),0) AS total,
 CASE WHEN c.tipo='07' THEN -1 ELSE 1 END AS signo_documento
FROM comprobantes c LEFT JOIN comprobante_detalles d ON d.comprobante_id=c.id GROUP BY c.id;

CREATE TRIGGER a_detalle_borrador BEFORE INSERT OR UPDATE OR DELETE ON comprobante_detalles
FOR EACH ROW EXECUTE FUNCTION fn_detalle_borrador('comprobantes','comprobante_id');

-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_comprobante BEFORE INSERT OR UPDATE ON comprobantes
FOR EACH ROW EXECUTE FUNCTION fn_validar_comprobante();

CREATE TRIGGER a_no_borrar BEFORE DELETE ON comprobantes FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE UNIQUE INDEX uq_comprobante_principal_cuenta ON comprobantes(cuenta_id)
WHERE tipo IN ('01','03') AND estado='EMITIDO';

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER a_no_borrar BEFORE DELETE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON comprobantes FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON comprobantes FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON comprobante_detalles FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON comprobante_detalles FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON envios_comprobante FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

CREATE TRIGGER zz_auditoria AFTER INSERT OR UPDATE OR DELETE ON envios_comprobante FOR EACH ROW EXECUTE FUNCTION fn_auditar();

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON series_comprobantes FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

CREATE INDEX idx_comprobantes_cuenta ON comprobantes(cuenta_id);

CREATE INDEX idx_comprobantes_fecha ON comprobantes(sucursal_id,fecha_emision);

CREATE INDEX idx_comprobantes_referencia ON comprobantes(comprobante_referencia_id);

COMMENT ON COLUMN series_comprobantes.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';

COMMENT ON COLUMN comprobantes.estado IS 'Estado del proceso de negocio: BORRADOR, EMITIDO, ANULADO. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN envios_comprobante.estado IS 'Estado del proceso de negocio: PENDIENTE, ENVIADO, ACEPTADO, ACEPTADO_OBSERVADO, RECHAZADO, ERROR, BAJA_ACEPTADA. Independiente de eliminacion logica; validar transiciones en el servicio.';

-- +goose Down
DROP VIEW IF EXISTS sch_restaurante.v_comprobante_totales;
DROP TABLE IF EXISTS sch_restaurante.envios_comprobante CASCADE;
DROP TABLE IF EXISTS sch_restaurante.comprobante_detalles CASCADE;
DROP TABLE IF EXISTS sch_restaurante.comprobantes CASCADE;
DROP TABLE IF EXISTS sch_restaurante.series_comprobantes CASCADE;
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_comprobante();
DROP FUNCTION IF EXISTS sch_restaurante.fn_siguiente_correlativo(p_serie_id BIGINT);
