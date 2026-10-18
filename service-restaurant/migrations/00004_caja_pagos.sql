-- Migracion 00004: Caja, pagos y devoluciones (I4)
-- Origen: docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql (V4),
-- dividido por dominio sin cambiar las sentencias. Requiere PostgreSQL 18.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
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

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_pago BEFORE INSERT OR UPDATE ON pagos
FOR EACH ROW EXECUTE FUNCTION fn_validar_pago();

CREATE TRIGGER a_no_borrar BEFORE DELETE ON pagos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_devolucion BEFORE INSERT ON devoluciones_pago
FOR EACH ROW EXECUTE FUNCTION fn_validar_devolucion();

CREATE TRIGGER a_inmutable BEFORE UPDATE OR DELETE ON devoluciones_pago
FOR EACH ROW EXECUTE FUNCTION fn_solo_insertar();

-- +goose StatementBegin
CREATE FUNCTION fn_validar_movimiento_caja() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
DECLARE v_estado TEXT;
BEGIN
    SELECT estado INTO STRICT v_estado FROM sesiones_caja WHERE id=NEW.sesion_caja_id FOR UPDATE;
    IF v_estado<>'ABIERTA' THEN RAISE EXCEPTION 'La sesion de caja esta cerrada'; END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER a_validar_movimiento_caja BEFORE INSERT ON movimientos_caja
FOR EACH ROW EXECUTE FUNCTION fn_validar_movimiento_caja();

CREATE TRIGGER a_inmutable BEFORE UPDATE OR DELETE ON movimientos_caja
FOR EACH ROW EXECUTE FUNCTION fn_solo_insertar();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_sesion BEFORE INSERT OR UPDATE ON sesiones_caja
FOR EACH ROW EXECUTE FUNCTION fn_validar_sesion();

CREATE TRIGGER a_no_borrar BEFORE DELETE ON sesiones_caja FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_proteger_metodo BEFORE UPDATE ON metodos_pago FOR EACH ROW EXECUTE FUNCTION fn_proteger_metodo_pago();

CREATE TRIGGER a2_baja_operativa BEFORE UPDATE ON cajas FOR EACH ROW EXECUTE FUNCTION fn_validar_baja_operativa();

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

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON cajas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON sesiones_caja FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('caja_id|cajas','sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON pagos FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('metodo_pago_id|metodos_pago');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON devoluciones_pago FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('metodo_pago_id|metodos_pago');

CREATE INDEX idx_pagos_cuenta ON pagos(cuenta_id,estado);

CREATE INDEX idx_pagos_sesion ON pagos(sesion_caja_id,estado);

CREATE INDEX idx_devoluciones_pago ON devoluciones_pago(pago_id);

CREATE INDEX idx_devoluciones_sesion ON devoluciones_pago(sesion_caja_id);

CREATE INDEX idx_movimientos_caja_sesion ON movimientos_caja(sesion_caja_id);

SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO metodos_pago(codigo,nombre,es_efectivo,requiere_referencia) VALUES
 ('EFECTIVO','Efectivo',true,false),('TARJETA','Tarjeta',false,true),
 ('TRANSFERENCIA','Transferencia',false,true),('YAPE','Yape',false,true),
 ('PLIN','Plin',false,true),('OTROS','Otros',false,false);

COMMENT ON COLUMN metodos_pago.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';

COMMENT ON COLUMN cajas.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha.';

COMMENT ON COLUMN sesiones_caja.estado IS 'Estado del proceso de negocio: ABIERTA, CERRADA. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN pagos.estado IS 'Estado del proceso de negocio: PENDIENTE, CONFIRMADO, FALLIDO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';

-- +goose Down
DROP VIEW IF EXISTS sch_restaurante.v_caja_resumen;
DROP VIEW IF EXISTS sch_restaurante.v_pedido_totales;
DROP VIEW IF EXISTS sch_restaurante.v_cuenta_totales;
DROP TABLE IF EXISTS sch_restaurante.devoluciones_pago CASCADE;
DROP TABLE IF EXISTS sch_restaurante.pagos CASCADE;
DROP TABLE IF EXISTS sch_restaurante.movimientos_caja CASCADE;
DROP TABLE IF EXISTS sch_restaurante.sesiones_caja CASCADE;
DROP TABLE IF EXISTS sch_restaurante.cajas CASCADE;
DROP TABLE IF EXISTS sch_restaurante.metodos_pago CASCADE;
DROP FUNCTION IF EXISTS sch_restaurante.fn_proteger_metodo_pago();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_sesion();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_movimiento_caja();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_devolucion();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_pago();
