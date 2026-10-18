-- Migracion 00003: Salon, reservas, pedidos, cuentas y cocina (I3)
-- Origen: docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql (V4),
-- dividido por dominio sin cambiar las sentencias. Requiere PostgreSQL 18.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
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

CREATE VIEW v_mesas_estado AS
SELECT m.id AS mesa_id,m.sucursal_id,m.numero_mesa,m.capacidad,m.es_activo,
 CASE WHEN NOT m.es_activo THEN 'BAJA_LOGICA'
      WHEN NOT m.habilitada THEN 'NO_HABILITADA'
      WHEN p.id IS NOT NULL THEN 'OCUPADA' ELSE 'DISPONIBLE' END AS estado_actual,
 p.id AS pedido_activo_id
FROM mesas m LEFT JOIN pedidos p ON p.mesa_id=m.id AND p.cerrado_en IS NULL;

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_reserva BEFORE INSERT OR UPDATE ON reservas
FOR EACH ROW EXECUTE FUNCTION fn_validar_reserva();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_guardar_detalle BEFORE INSERT OR UPDATE OR DELETE ON detalles_pedido
FOR EACH ROW EXECUTE FUNCTION fn_guardar_detalle_pedido();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_cuenta BEFORE INSERT OR UPDATE ON cuentas
FOR EACH ROW EXECUTE FUNCTION fn_validar_cuenta();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_validar_pedido BEFORE INSERT OR UPDATE ON pedidos
FOR EACH ROW EXECUTE FUNCTION fn_validar_pedido();

-- +goose StatementBegin
CREATE FUNCTION fn_validar_delivery() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pedidos WHERE id=NEW.pedido_id AND tipo='DELIVERY') THEN
        RAISE EXCEPTION 'Solo pedidos DELIVERY pueden registrar datos de reparto';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER a_validar_delivery BEFORE INSERT OR UPDATE ON pedidos_delivery
FOR EACH ROW EXECUTE FUNCTION fn_validar_delivery();

CREATE TRIGGER a_no_borrar BEFORE DELETE ON pedidos FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

CREATE TRIGGER a_no_borrar BEFORE DELETE ON cuentas FOR EACH ROW EXECUTE FUNCTION fn_no_borrar();

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a2_baja_operativa BEFORE UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_validar_baja_operativa();

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

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON zonas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','zona_id|zonas');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON reservas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','mesa_id|mesas','cliente_id|clientes');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON pedidos FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','mesa_id|mesas','cliente_id|clientes','mesero_id|empleados');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON cuentas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('cliente_id|clientes');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON detalles_pedido FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('producto_id|productos');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON comandas FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('sucursal_id|sucursales','estacion_id|estaciones_preparacion');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON pedidos_delivery FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('repartidor_id|empleados');

CREATE INDEX idx_reservas_sucursal_inicio ON reservas(sucursal_id,inicio);

CREATE INDEX idx_pedidos_sucursal_fecha ON pedidos(sucursal_id,abierto_en);

CREATE INDEX idx_pedidos_cliente ON pedidos(cliente_id);

CREATE INDEX idx_pedidos_pendientes ON pedidos(sucursal_id,estado) WHERE cerrado_en IS NULL;

CREATE INDEX idx_detalles_cuenta ON detalles_pedido(cuenta_id);

CREATE INDEX idx_detalles_producto ON detalles_pedido(producto_id);

CREATE INDEX idx_comandas_pedido ON comandas(pedido_id,enviado_en);

CREATE INDEX idx_comanda_detalles_pedido ON comanda_detalles(detalle_pedido_id);

CREATE INDEX idx_comanda_detalles_estado ON comanda_detalles(estado);

COMMENT ON COLUMN zonas.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';

COMMENT ON COLUMN mesas.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';

COMMENT ON COLUMN reservas.estado IS 'Estado del proceso de negocio: PENDIENTE, CONFIRMADA, ATENDIDA, CANCELADA, NO_ASISTIO. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN pedidos.estado IS 'Estado del proceso de negocio: REGISTRADO, PREPARACION, LISTO, ENTREGADO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN cuentas.estado IS 'Estado del proceso de negocio: ABIERTA, CERRADA, CANCELADA. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN detalles_pedido.estado IS 'Estado del proceso de negocio: ACTIVO, ANULADO. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN comanda_detalles.estado IS 'Estado del proceso de negocio: PENDIENTE, PREPARACION, LISTO, ENTREGADO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';

COMMENT ON COLUMN pedidos_delivery.estado IS 'Estado del proceso de negocio: PENDIENTE, ASIGNADO, EN_RUTA, ENTREGADO, FALLIDO, CANCELADO. Independiente de eliminacion logica; validar transiciones en el servicio.';

-- +goose Down
DROP VIEW IF EXISTS sch_restaurante.v_mesas_estado;
DROP TABLE IF EXISTS sch_restaurante.pedidos_delivery CASCADE;
DROP TABLE IF EXISTS sch_restaurante.comanda_detalles CASCADE;
DROP TABLE IF EXISTS sch_restaurante.comandas CASCADE;
DROP TABLE IF EXISTS sch_restaurante.detalles_pedido CASCADE;
DROP TABLE IF EXISTS sch_restaurante.cuentas CASCADE;
DROP TABLE IF EXISTS sch_restaurante.pedidos CASCADE;
DROP TABLE IF EXISTS sch_restaurante.reservas CASCADE;
DROP TABLE IF EXISTS sch_restaurante.mesas CASCADE;
DROP TABLE IF EXISTS sch_restaurante.zonas CASCADE;
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_baja_operativa();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_delivery();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_pedido();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_cuenta();
DROP FUNCTION IF EXISTS sch_restaurante.fn_guardar_detalle_pedido();
DROP FUNCTION IF EXISTS sch_restaurante.fn_validar_reserva();
