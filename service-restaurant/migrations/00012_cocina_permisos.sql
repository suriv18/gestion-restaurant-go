-- Migracion 00012: permiso de cocina (I3, rebanada 2).
-- COCINA_OPERAR existe desde 00001 y solo lo tiene ADMIN (00001 asigna todos los permisos a ADMIN):
-- aqui se asigna al rol COCINA. PEDIDO_VER para COCINA ya lo dio 00011.
-- Requiere PostgreSQL 18 y las migraciones 00001-00011.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='COCINA' AND p.codigo='COCINA_OPERAR'
  AND NOT EXISTS (SELECT 1 FROM roles_permisos x WHERE x.rol_id=r.id AND x.permiso_id=p.id);

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo='COCINA')
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='COCINA_OPERAR');
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
