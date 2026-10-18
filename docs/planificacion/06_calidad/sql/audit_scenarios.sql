SET search_path=sch_restaurante,pg_catalog;
SELECT pg_temp.assert_true((SELECT count(*)=49 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='creado_por' AND is_nullable='NO'),'49 creador no nulo');
SELECT pg_temp.assert_true((SELECT count(*)=46 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='actualizado_por' AND is_nullable='NO'),'46 editor no nulo');
SELECT pg_temp.assert_true((SELECT count(*)=26 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='es_activo' AND is_nullable='NO'),'26 indicadores de actividad');
SELECT pg_temp.assert_true((SELECT count(*)=0 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='activo'),'nomenclatura unica');
SELECT pg_temp.assert_true((SELECT creado_por=2 AND actualizado_por=2 FROM categorias WHERE id=1),'autor contextual, no ID suministrado');
SELECT pg_temp.expect_error('UPDATE categorias SET creado_por=1 WHERE id=1','creacion son inmutables');
SELECT set_config('app.usuario_id','',false);
SELECT pg_temp.expect_error($q$INSERT INTO categorias(nombre) VALUES('SIN_ACTOR')$q$,'app.usuario_id');
SELECT set_config('app.usuario_id','999999',false);
SELECT pg_temp.expect_error($q$INSERT INTO categorias(nombre) VALUES('ACTOR_INVALIDO')$q$,'Actor de auditoria');
SELECT set_config('app.usuario_id','2',false);
INSERT INTO usuarios(username,password_hash) VALUES ('editor','HASH_DE_PRUEBA_NO_UTILIZABLE');
SELECT set_config('app.usuario_id',(SELECT id::text FROM usuarios WHERE username='editor'),false);
UPDATE categorias SET descripcion='Modificado por editor' WHERE id=1;
SELECT pg_temp.assert_true((SELECT creado_por=2 AND actualizado_por=(SELECT id FROM usuarios WHERE username='editor') FROM categorias WHERE id=1),'autor original preservado, editor nuevo');
INSERT INTO productos_sucursal(producto_id,sucursal_id) VALUES (1,1);
SELECT pg_temp.assert_true((SELECT count(*)=1 FROM v_carta_disponible WHERE producto_id=1),'carta activa');
UPDATE categorias SET es_activo=false WHERE id=1;
SELECT pg_temp.assert_true((SELECT actualizado_por=(SELECT id FROM usuarios WHERE username='editor') AND NOT es_activo FROM categorias WHERE id=1),'autoria de desactivacion');
SELECT pg_temp.assert_true((SELECT count(*)=0 FROM v_carta_disponible WHERE producto_id=1),'categoria de baja oculta selector');
SELECT pg_temp.assert_true((SELECT total=100 FROM v_comprobante_totales WHERE comprobante_id=1),'baja no cambia documento historico');
SELECT pg_temp.expect_error($q$INSERT INTO productos(categoria_id,codigo,nombre,tipo,precio_venta,afectacion_igv,impuesto_id)
VALUES (1,'NUEVO','Plato','PREPARADO',10,'10',1)$q$,'dado de baja');
SELECT pg_temp.expect_error($q$INSERT INTO categorias(nombre) VALUES('PRUEBA')$q$,'unique');
SELECT pg_temp.expect_error('DELETE FROM categorias WHERE id=1','No se permite borrar');
UPDATE categorias SET es_activo=true WHERE id=1;
SELECT pg_temp.assert_true((SELECT es_activo FROM categorias WHERE id=1),'reactivacion simple');
SELECT pg_temp.assert_true((SELECT count(*)>=1 FROM auditoria_eventos WHERE tabla='categorias' AND registro_id=1 AND datos_despues->>'es_activo'='false'),'desactivacion permanece en auditoria');
SELECT pg_temp.assert_true((SELECT count(*)=1 FROM v_carta_disponible WHERE producto_id=1),'restauracion de carta');
SELECT pg_temp.assert_true((SELECT count(*)=0 FROM auditoria_eventos WHERE tabla='usuarios' AND ((datos_antes ? 'password_hash') OR (datos_despues ? 'password_hash'))),'sin hashes en auditoria');

INSERT INTO usuarios_roles(usuario_id,rol_id) SELECT id,1 FROM usuarios WHERE username='editor';
INSERT INTO usuarios_sucursales(usuario_id,sucursal_id) SELECT id,1 FROM usuarios WHERE username='editor';
SELECT pg_temp.assert_true((SELECT count(*)=10 FROM v_permisos_efectivos WHERE usuario_id=(SELECT id FROM usuarios WHERE username='editor')),'permisos efectivos');
UPDATE usuarios_roles SET es_activo=false WHERE usuario_id=(SELECT id FROM usuarios WHERE username='editor');
SELECT pg_temp.assert_true((SELECT count(*)=0 FROM v_permisos_efectivos WHERE usuario_id=(SELECT id FROM usuarios WHERE username='editor')),'asignacion desactivada revoca permisos');
UPDATE usuarios_roles SET es_activo=true WHERE usuario_id=(SELECT id FROM usuarios WHERE username='editor');
UPDATE roles SET es_activo=false WHERE id=1;
SELECT pg_temp.assert_true((SELECT count(*)=0 FROM v_permisos_efectivos WHERE usuario_id=(SELECT id FROM usuarios WHERE username='editor')),'rol de baja revoca permisos');
UPDATE roles SET es_activo=true WHERE id=1;
SELECT pg_temp.expect_error($q$INSERT INTO usuarios_roles(usuario_id,rol_id) VALUES (1,1)$q$,'actor tecnico');
SELECT pg_temp.expect_error($q$UPDATE usuarios SET es_activo=false,motivo_eliminacion='No permitido' WHERE es_sistema$q$,'actor tecnico');

UPDATE usuarios SET es_activo=false,motivo_eliminacion='Usuario retirado' WHERE username='test';
SELECT set_config('app.usuario_id','2',false);
SELECT pg_temp.expect_error($q$UPDATE categorias SET descripcion='No permitido' WHERE id=1$q$,'inactivo');
SELECT set_config('app.usuario_id',(SELECT id::text FROM usuarios WHERE username='editor'),false);
UPDATE usuarios SET es_activo=true WHERE username='test';

UPDATE articulos_inventario SET es_activo=false,motivo_eliminacion='No comprar mas' WHERE id=1;
SELECT pg_temp.assert_true((SELECT stock_actual=57 FROM v_existencias WHERE articulo_id=1),'stock historico visible aunque maestro inactivo');
SELECT pg_temp.assert_true((SELECT sum(cantidad_neta)=57 FROM v_kardex WHERE articulo_id=1),'kardex no se filtra por maestro');
UPDATE articulos_inventario SET es_activo=true WHERE id=1;
UPDATE metodos_pago SET es_activo=false WHERE id=1;
SELECT pg_temp.assert_true((SELECT efectivo_esperado=45 FROM v_caja_resumen WHERE sesion_caja_id=1),'medio inactivo no altera cierre');
UPDATE metodos_pago SET es_activo=true WHERE id=1;
SELECT pg_temp.expect_error('UPDATE devoluciones_pago SET monto=1 WHERE id=1','inmutable');

INSERT INTO sesiones_caja(caja_id,sucursal_id,abierto_por,fondo_inicial) VALUES (1,1,2,0);
SELECT pg_temp.expect_error($q$UPDATE cajas SET es_activo=false,motivo_eliminacion='Cerrar' WHERE id=1$q$,'Cerrar la sesion');
UPDATE sesiones_caja SET estado='CERRADA',cierre_en=clock_timestamp(),cerrado_por=2,efectivo_contado=0 WHERE estado='ABIERTA';
UPDATE comprobantes SET estado='ANULADO',anulado_en=clock_timestamp(),motivo_anulacion='Prueba de guardia con actualizado_por' WHERE id=1;
SELECT pg_temp.assert_true((SELECT actualizado_por=(SELECT id FROM usuarios WHERE username='editor') FROM comprobantes WHERE id=1),'anulacion fiscal compatible con auditoria');

-- Regresion V4: distribucion de campos, estados y baja de maestros principales.
SELECT pg_temp.assert_true((SELECT count(*)=9 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='eliminado_en'),'9 maestros con fecha de baja');
SELECT pg_temp.assert_true((SELECT count(*)=9 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='eliminado_por'),'9 maestros con actor de baja');
SELECT pg_temp.assert_true((SELECT count(*)=9 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='motivo_eliminacion'),'9 maestros con motivo');
SELECT pg_temp.assert_true((SELECT count(*)=13 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND column_name='estado' AND table_name IN (SELECT table_name FROM information_schema.tables WHERE table_schema='sch_restaurante' AND table_type='BASE TABLE')),'13 tablas con estado de negocio');
SELECT pg_temp.assert_true((SELECT count(*)=0 FROM information_schema.columns WHERE table_schema='sch_restaurante' AND table_name IN ('roles_permisos','usuarios_roles','usuarios_sucursales','articulos_almacen','productos_sucursal') AND column_name IN ('eliminado_por','eliminado_en','motivo_eliminacion')),'relaciones sin marcadores de eliminacion');
SELECT pg_temp.expect_error('UPDATE productos SET es_activo=false WHERE id=1','motivo_eliminacion');
UPDATE productos SET es_activo=false,motivo_eliminacion='Producto descontinuado' WHERE id=1;
SELECT pg_temp.assert_true((SELECT eliminado_por=(SELECT id FROM usuarios WHERE username='editor') AND eliminado_en IS NOT NULL FROM productos WHERE id=1),'marcadores del maestro completos');
UPDATE productos SET es_activo=true WHERE id=1;
SELECT pg_temp.assert_true((SELECT eliminado_en IS NULL AND eliminado_por IS NULL AND motivo_eliminacion IS NULL FROM productos WHERE id=1),'restauracion de maestro limpia marcadores');
SELECT pg_temp.assert_true((SELECT count(*)>0 FROM auditoria_eventos WHERE tabla='productos' AND datos_despues->>'motivo_eliminacion'='Producto descontinuado'),'motivo historico preservado');
SELECT pg_temp.expect_error($q$UPDATE reservas SET estado='INEXISTENTE' WHERE id=1$q$,'check');
