-- Migracion 00011: permisos de pedidos (I3, rebanada 1).
-- PEDIDO_VER y PEDIDO_EDITAR son nuevos. PEDIDO_CREAR y DESCUENTO_AUTORIZAR existen desde 00001 solo para ADMIN:
-- aqui se asignan a MESERO y CAJERO (PEDIDO_CREAR) y a CAJERO (DESCUENTO_AUTORIZAR).
-- Requiere PostgreSQL 18 y las migraciones 00001-00010.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO permisos(codigo,descripcion) VALUES
 ('PEDIDO_VER','Consultar pedidos, cuentas y consumos de la sede'),
 ('PEDIDO_EDITAR','Agregar y editar consumos de un pedido abierto');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='ADMIN' AND p.codigo IN ('PEDIDO_VER','PEDIDO_EDITAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO','COCINA') AND p.codigo='PEDIDO_VER';

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO') AND p.codigo IN ('PEDIDO_CREAR','PEDIDO_EDITAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='CAJERO' AND p.codigo='DESCUENTO_AUTORIZAR';

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
ALTER TABLE permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos WHERE permiso_id IN (SELECT id FROM permisos WHERE codigo IN ('PEDIDO_VER','PEDIDO_EDITAR'));
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo IN ('MESERO','CAJERO'))
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='PEDIDO_CREAR');
DELETE FROM roles_permisos
 WHERE rol_id IN (SELECT id FROM roles WHERE codigo='CAJERO')
   AND permiso_id IN (SELECT id FROM permisos WHERE codigo='DESCUENTO_AUTORIZAR');
DELETE FROM permisos WHERE codigo IN ('PEDIDO_VER','PEDIDO_EDITAR');
ALTER TABLE permisos ENABLE TRIGGER a_no_borrar;
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
