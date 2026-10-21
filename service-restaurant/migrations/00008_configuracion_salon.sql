-- Migracion 00008: configuracion y salon (I2, rebanada 1).
-- Seis permisos nuevos (negocio, sucursal, mesa) y baja con motivo en mesas (ADR 005).
-- Requiere PostgreSQL 18 y las migraciones 00001-00007.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO permisos(codigo,descripcion) VALUES
 ('NEGOCIO_VER','Consultar los datos del negocio'),
 ('NEGOCIO_ADMINISTRAR','Editar los datos del negocio'),
 ('SUCURSAL_VER','Consultar sucursales'),
 ('SUCURSAL_ADMINISTRAR','Crear, editar, dar de baja y restaurar sucursales'),
 ('MESA_VER','Consultar zonas y mesas'),
 ('MESA_ADMINISTRAR','Administrar zonas y mesas');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='ADMIN'
  AND p.codigo IN ('NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_VER','MESA_ADMINISTRAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO') AND p.codigo='MESA_VER';

-- Baja de mesa con motivo (decision del equipo; el expediente la definia como actividad simple).
ALTER TABLE mesas
    ADD COLUMN eliminado_en TIMESTAMPTZ,
    ADD COLUMN eliminado_por BIGINT,
    ADD COLUMN motivo_eliminacion TEXT;

-- Filas inactivas previas: se completan antes de exigir la restriccion. El trigger vigente
-- (sin baja logica) permite el UPDATE y la auditoria lo registra.
UPDATE mesas
SET eliminado_en = actualizado_en, eliminado_por = actualizado_por,
    motivo_eliminacion = 'Baja anterior a la migración 00008'
WHERE NOT es_activo;

ALTER TABLE mesas ADD CONSTRAINT ck_mesas_baja CHECK (
    (es_activo AND eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL)
    OR (NOT es_activo AND eliminado_en IS NOT NULL AND eliminado_por IS NOT NULL
        AND motivo_eliminacion IS NOT NULL AND btrim(motivo_eliminacion) <> ''));

ALTER TABLE mesas ADD CONSTRAINT fk_mesas_eliminador FOREIGN KEY (eliminado_por) REFERENCES usuarios(id) ON DELETE RESTRICT;

DROP TRIGGER a0_metadatos ON mesas;
CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','true');

COMMENT ON COLUMN mesas.es_activo IS 'Baja administrativa del maestro; FALSE exige motivo y completa actor/fecha (ADR 005).';

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

DROP TRIGGER a0_metadatos ON mesas;
CREATE TRIGGER a0_metadatos BEFORE INSERT OR UPDATE ON mesas FOR EACH ROW EXECUTE FUNCTION fn_metadatos_fila('true','false');

ALTER TABLE mesas DROP CONSTRAINT fk_mesas_eliminador;
ALTER TABLE mesas DROP CONSTRAINT ck_mesas_baja;
ALTER TABLE mesas DROP COLUMN eliminado_en, DROP COLUMN eliminado_por, DROP COLUMN motivo_eliminacion;
COMMENT ON COLUMN mesas.es_activo IS 'Actividad administrativa; FALSE desactiva la fila sin campos de eliminacion.';

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
ALTER TABLE permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos WHERE permiso_id IN (SELECT id FROM permisos WHERE codigo IN
 ('NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_VER','MESA_ADMINISTRAR'));
DELETE FROM permisos WHERE codigo IN
 ('NEGOCIO_VER','NEGOCIO_ADMINISTRAR','SUCURSAL_VER','SUCURSAL_ADMINISTRAR','MESA_VER','MESA_ADMINISTRAR');
ALTER TABLE permisos ENABLE TRIGGER a_no_borrar;
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
