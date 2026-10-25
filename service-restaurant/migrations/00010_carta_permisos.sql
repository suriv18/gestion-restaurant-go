-- Migracion 00010: carta, estaciones de preparacion e impuestos (I2, rebanada 3).
-- Siete permisos nuevos. Las tablas ya existen desde 00002 con baja, auditoria y unicidad.
-- Requiere PostgreSQL 18 y las migraciones 00001-00009.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO permisos(codigo,descripcion) VALUES
 ('CARTA_VER','Consultar categorias y productos de la carta'),
 ('CARTA_ADMINISTRAR','Crear, editar, dar de baja y restaurar categorias y productos; fijar precio y estacion por sede'),
 ('CARTA_DISPONIBILIDAD','Pausar o reactivar la disponibilidad de un producto en una sede'),
 ('ESTACION_VER','Consultar estaciones de preparacion'),
 ('ESTACION_ADMINISTRAR','Crear, editar, dar de baja y restaurar estaciones de preparacion'),
 ('IMPUESTO_VER','Consultar impuestos y tasas'),
 ('IMPUESTO_ADMINISTRAR','Crear y editar tasas de impuesto');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='ADMIN'
  AND p.codigo IN ('CARTA_VER','CARTA_ADMINISTRAR','CARTA_DISPONIBILIDAD','ESTACION_VER','ESTACION_ADMINISTRAR','IMPUESTO_VER','IMPUESTO_ADMINISTRAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO','COCINA','ALMACEN') AND p.codigo='CARTA_VER';

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='COCINA' AND p.codigo='CARTA_DISPONIBILIDAD';

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','COCINA') AND p.codigo='ESTACION_VER';

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
ALTER TABLE permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos WHERE permiso_id IN (SELECT id FROM permisos WHERE codigo IN
 ('CARTA_VER','CARTA_ADMINISTRAR','CARTA_DISPONIBILIDAD','ESTACION_VER','ESTACION_ADMINISTRAR','IMPUESTO_VER','IMPUESTO_ADMINISTRAR'));
DELETE FROM permisos WHERE codigo IN
 ('CARTA_VER','CARTA_ADMINISTRAR','CARTA_DISPONIBILIDAD','ESTACION_VER','ESTACION_ADMINISTRAR','IMPUESTO_VER','IMPUESTO_ADMINISTRAR');
ALTER TABLE permisos ENABLE TRIGGER a_no_borrar;
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
