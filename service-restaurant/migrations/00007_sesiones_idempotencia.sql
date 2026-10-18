-- Migracion 00007: sesiones de acceso e idempotencia de escrituras (I1).
-- Tablas TECNICAS: deliberadamente sin los triggers de metadatos ni de auditoria de la V4.
-- Motivo: sesiones.ultimo_uso_en se actualiza durante el uso y la auditoria V4 registraria un
-- evento por peticion; idempotencia es un registro efimero (24 h). Ver docs/adr/004.
-- Requiere PostgreSQL 18 y las migraciones 00001-00006.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';

CREATE TABLE sesiones (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    usuario_id BIGINT NOT NULL REFERENCES usuarios(id),
    creada_en TIMESTAMPTZ NOT NULL,
    ultimo_uso_en TIMESTAMPTZ NOT NULL,
    expira_inactividad_en TIMESTAMPTZ NOT NULL,
    expira_absoluta_en TIMESTAMPTZ NOT NULL,
    revocada_en TIMESTAMPTZ,
    CHECK (ultimo_uso_en >= creada_en),
    CHECK (expira_inactividad_en <= expira_absoluta_en),
    CHECK (expira_absoluta_en > creada_en)
);

CREATE INDEX idx_sesiones_usuario_vigentes ON sesiones (usuario_id) WHERE revocada_en IS NULL;

COMMENT ON TABLE sesiones IS 'Sesiones de acceso. El JWT identifica la sesion por su id (claim sid); vigencia, revocacion y permisos se comprueban aqui en cada peticion. Sin triggers de auditoria (tabla tecnica).';

CREATE TABLE idempotencia (
    clave UUID NOT NULL,
    usuario_id BIGINT NOT NULL REFERENCES usuarios(id),
    operacion VARCHAR(120) NOT NULL CHECK (btrim(operacion) <> ''),
    huella BYTEA NOT NULL CHECK (octet_length(huella) = 32),
    estado VARCHAR(12) NOT NULL CHECK (estado IN ('EN_CURSO', 'COMPLETADA')),
    respuesta_status INTEGER CHECK (respuesta_status BETWEEN 100 AND 599),
    respuesta_cuerpo BYTEA,
    creada_en TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expira_en TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (clave, usuario_id, operacion),
    CHECK ((estado = 'COMPLETADA') = (respuesta_status IS NOT NULL)),
    CHECK (expira_en > creada_en)
);

CREATE INDEX idx_idempotencia_expira ON idempotencia (expira_en);

COMMENT ON TABLE idempotencia IS 'Una clave por intencion (ADR-04): misma huella repite el efecto guardado; distinta huella es conflicto. Registro efimero (expira_en); sin triggers de auditoria (tabla tecnica).';

-- +goose Down
DROP TABLE IF EXISTS sch_restaurante.idempotencia;
DROP TABLE IF EXISTS sch_restaurante.sesiones;
