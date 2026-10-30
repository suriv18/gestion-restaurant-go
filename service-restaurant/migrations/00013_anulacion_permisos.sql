-- Migracion 00013: permiso de anulacion para el cajero (I3, rebanada 3).
-- PEDIDO_ANULAR existe desde 00001 y solo lo tiene ADMIN (00001 asigna todos los permisos a ADMIN):
-- aqui se asigna al rol CAJERO. Anular es una accion de supervision: MESERO y COCINA no la reciben.
-- Requiere PostgreSQL 18 y las migraciones 00001-00012.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='CAJERO' AND p.codigo='PEDIDO_ANULAR'
  AND NOT EXISTS (SELECT 1 FROM roles_permisos x WHERE x.rol_id=r.id AND x.permiso_id=p.id);

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo='CAJERO')
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='PEDIDO_ANULAR');
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
