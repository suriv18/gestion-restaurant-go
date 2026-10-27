-- Migracion 00009: directorio de clientes, proveedores y empleados (I2, rebanada 2).
-- Seis permisos nuevos. Las tablas ya existen desde 00001 con baja con motivo, auditoria y unicidad.
-- Requiere PostgreSQL 18 y las migraciones 00001-00008.

-- +goose Up
SET LOCAL search_path = sch_restaurante, pg_catalog;
SET LOCAL TIME ZONE 'America/Lima';
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

INSERT INTO permisos(codigo,descripcion) VALUES
 ('CLIENTE_VER','Consultar clientes'),
 ('CLIENTE_ADMINISTRAR','Crear, editar, dar de baja y restaurar clientes'),
 ('PROVEEDOR_VER','Consultar proveedores'),
 ('PROVEEDOR_ADMINISTRAR','Crear, editar, dar de baja y restaurar proveedores'),
 ('EMPLEADO_VER','Consultar empleados'),
 ('EMPLEADO_ADMINISTRAR','Crear, editar, dar de baja y restaurar empleados');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='ADMIN'
  AND p.codigo IN ('CLIENTE_VER','CLIENTE_ADMINISTRAR','PROVEEDOR_VER','PROVEEDOR_ADMINISTRAR','EMPLEADO_VER','EMPLEADO_ADMINISTRAR');

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo IN ('MESERO','CAJERO') AND p.codigo='CLIENTE_VER';

INSERT INTO roles_permisos(rol_id,permiso_id)
SELECT r.id,p.id FROM roles r CROSS JOIN permisos p
WHERE r.codigo='ALMACEN' AND p.codigo IN ('PROVEEDOR_VER','PROVEEDOR_ADMINISTRAR');

-- +goose Down
SET LOCAL search_path = sch_restaurante, pg_catalog;
SELECT set_config('app.usuario_id', (SELECT id::text FROM usuarios WHERE es_sistema), true);

ALTER TABLE roles_permisos DISABLE TRIGGER a_no_borrar;
ALTER TABLE permisos DISABLE TRIGGER a_no_borrar;
DELETE FROM roles_permisos WHERE permiso_id IN (SELECT id FROM permisos WHERE codigo IN
 ('CLIENTE_VER','CLIENTE_ADMINISTRAR','PROVEEDOR_VER','PROVEEDOR_ADMINISTRAR','EMPLEADO_VER','EMPLEADO_ADMINISTRAR'));
DELETE FROM permisos WHERE codigo IN
 ('CLIENTE_VER','CLIENTE_ADMINISTRAR','PROVEEDOR_VER','PROVEEDOR_ADMINISTRAR','EMPLEADO_VER','EMPLEADO_ADMINISTRAR');
ALTER TABLE permisos ENABLE TRIGGER a_no_borrar;
ALTER TABLE roles_permisos ENABLE TRIGGER a_no_borrar;
