-- Migracion 00001: Negocio, seguridad, personas y auditoria (I1)
-- Origen: docs/planificacion/02_base_datos/modelo_de_base_de_datos_para_restaurante.sql (V4),
-- dividido por dominio sin cambiar las sentencias. Requiere PostgreSQL 18.

-- +goose Up
CREATE SCHEMA sch_restaurante;
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
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

ALTER TABLE auditoria_eventos ADD CONSTRAINT fk_auditoria_eventos_creador FOREIGN KEY (creado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

CREATE UNIQUE INDEX uq_usuario_sistema ON usuarios(es_sistema) WHERE es_sistema;

-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION fn_no_borrar() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    RAISE EXCEPTION 'No se permite borrar %. Use anulacion o una operacion compensatoria.', TG_TABLE_NAME;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION fn_solo_insertar() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    RAISE EXCEPTION '% conserva historia inmutable; registrar una operacion compensatoria.', TG_TABLE_NAME;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a_inmutable BEFORE UPDATE OR DELETE ON auditoria_eventos FOR EACH ROW EXECUTE FUNCTION fn_solo_insertar();

-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose StatementBegin
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
-- +goose StatementEnd

CREATE TRIGGER a2_proteger_usuario_sistema BEFORE INSERT OR UPDATE ON usuarios
FOR EACH ROW EXECUTE FUNCTION fn_proteger_usuario_sistema();

-- +goose StatementBegin
CREATE FUNCTION fn_no_roles_sistema() RETURNS trigger
LANGUAGE plpgsql SET search_path = sch_restaurante, pg_catalog AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM usuarios WHERE id=NEW.usuario_id AND es_sistema) THEN
        RAISE EXCEPTION 'El actor tecnico no puede recibir roles ni acceso interactivo a sucursales';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER a2_no_roles_sistema BEFORE INSERT OR UPDATE ON usuarios_roles
FOR EACH ROW EXECUTE FUNCTION fn_no_roles_sistema();

CREATE TRIGGER a2_no_roles_sistema BEFORE INSERT OR UPDATE ON usuarios_sucursales
FOR EACH ROW EXECUTE FUNCTION fn_no_roles_sistema();

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

CREATE TRIGGER a0_metadatos BEFORE INSERT ON auditoria_eventos FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('false','false');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON sucursales FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('negocio_id|negocio');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON roles_permisos FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('rol_id|roles','permiso_id|permisos');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON usuarios_roles FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('usuario_id|usuarios','rol_id|roles');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON usuarios_sucursales FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('usuario_id|usuarios','sucursal_id|sucursales');

CREATE TRIGGER a1_referencias_activas BEFORE INSERT OR UPDATE ON empleados FOR EACH ROW EXECUTE FUNCTION fn_referencias_activas('usuario_id|usuarios');

CREATE INDEX idx_auditoria_registro ON auditoria_eventos(tabla,registro_id,creado_en);

CREATE INDEX idx_auditoria_usuario ON auditoria_eventos(creado_por,creado_en);

CREATE INDEX idx_clientes_nombre_activos ON clientes(nombre_razon_social) WHERE es_activo;

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

-- +goose Down
DROP SCHEMA sch_restaurante CASCADE;
