SET search_path=sch_restaurante,pg_catalog;
SELECT set_config('app.usuario_id',(SELECT id::text FROM usuarios WHERE es_sistema),false);
CREATE FUNCTION pg_temp.expect_error(p_sql text,p_fragment text) RETURNS void LANGUAGE plpgsql AS $$
DECLARE caught boolean:=false;
BEGIN
 BEGIN
  EXECUTE p_sql;
 EXCEPTION WHEN others THEN
  caught:=true;
  IF position(lower(p_fragment) in lower(SQLERRM))=0 THEN
   RAISE EXCEPTION 'Error inesperado: %; esperado: %',SQLERRM,p_fragment;
  END IF;
 END;
 IF NOT caught THEN RAISE EXCEPTION 'Falto rechazo de: %',p_sql; END IF;
END;
$$;
CREATE FUNCTION pg_temp.assert_true(v boolean,description text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN IF v IS DISTINCT FROM true THEN RAISE EXCEPTION 'Fallo: %',description; END IF; END;
$$;
INSERT INTO negocio(ruc,razon_social,nombre_comercial,direccion_fiscal) VALUES ('20000000001','PRUEBA','PRUEBA','PRUEBA');
INSERT INTO sucursales(negocio_id,codigo,nombre,direccion) VALUES (1,'PRUEBA','PRUEBA','PRUEBA');
INSERT INTO usuarios(username,password_hash) VALUES ('test','HASH_DE_PRUEBA_NO_UTILIZABLE');
SELECT set_config('app.usuario_id',(SELECT id::text FROM usuarios WHERE username='test'),false);
INSERT INTO zonas(sucursal_id,nombre) VALUES (1,'SALON');
INSERT INTO mesas(sucursal_id,zona_id,numero_mesa,capacidad) VALUES (1,1,'M01',4);
INSERT INTO categorias(nombre) VALUES ('PRUEBA');
INSERT INTO productos(categoria_id,codigo,nombre,tipo,precio_venta,afectacion_igv,impuesto_id)
VALUES (1,'P1','Plato','PREPARADO',50,'10',1);
INSERT INTO pedidos(sucursal_id,clave_idempotencia,mesa_id,tipo,creado_por)
VALUES (1,gen_random_uuid(),1,'SALON',1);
INSERT INTO cuentas(pedido_id,numero,creado_por) VALUES (1,1,1);
INSERT INTO detalles_pedido(pedido_id,cuenta_id,producto_id,descripcion,cantidad,precio_unitario,afectacion_igv,tasa_impuesto,creado_por)
VALUES (1,1,1,'Plato',2,50,'10',18,1);
SELECT pg_temp.assert_true((SELECT total=100 FROM v_cuenta_totales WHERE cuenta_id=1),'total de cuenta');
INSERT INTO cajas(sucursal_id,codigo,nombre) VALUES (1,'C1','Caja');
INSERT INTO sesiones_caja(caja_id,sucursal_id,abierto_por,fondo_inicial) VALUES (1,1,1,10);
INSERT INTO pagos(cuenta_id,sesion_caja_id,metodo_pago_id,monto,efectivo_recibido,estado,clave_idempotencia,creado_por)
VALUES (1,1,1,60,100,'CONFIRMADO',gen_random_uuid(),1);
SELECT pg_temp.assert_true((SELECT vuelto=40 FROM pagos WHERE id=1),'vuelto de efectivo');
SELECT pg_temp.expect_error($q$INSERT INTO pagos(cuenta_id,sesion_caja_id,metodo_pago_id,monto,efectivo_recibido,estado,clave_idempotencia,creado_por)
VALUES (1,1,1,50,50,'CONFIRMADO',gen_random_uuid(),1)$q$,'supera el saldo');
INSERT INTO pagos(cuenta_id,sesion_caja_id,metodo_pago_id,monto,estado,proveedor_operacion,referencia_operacion,clave_idempotencia,creado_por)
VALUES (1,1,4,40,'CONFIRMADO','YAPE','TEST-001',gen_random_uuid(),1);
SELECT pg_temp.assert_true((SELECT estado_cobro='PAGADO' AND cobrado=100 FROM v_cuenta_totales WHERE cuenta_id=1),'pago mixto');
SELECT pg_temp.expect_error('UPDATE detalles_pedido SET cantidad=3 WHERE id=1','ya no son editables');
SELECT pg_temp.expect_error('UPDATE pagos SET monto=59 WHERE id=1','inmutable');
INSERT INTO devoluciones_pago(pago_id,sesion_caja_id,metodo_pago_id,monto,motivo,clave_idempotencia,autorizado_por)
VALUES (1,1,1,20,'Prueba',gen_random_uuid(),1);
SELECT pg_temp.expect_error($q$INSERT INTO devoluciones_pago(pago_id,sesion_caja_id,metodo_pago_id,monto,motivo,clave_idempotencia,autorizado_por)
VALUES (1,1,1,50,'Prueba',gen_random_uuid(),1)$q$,'supera el cobro');
SELECT pg_temp.assert_true((SELECT estado_cobro='REEMBOLSADO_PARCIAL' AND cobro_neto=80 FROM v_cuenta_totales WHERE cuenta_id=1),'devolucion parcial');
INSERT INTO movimientos_caja(sesion_caja_id,tipo,monto,concepto,creado_por,clave_idempotencia)
VALUES (1,'EGRESO',5,'Prueba',1,gen_random_uuid());
SELECT pg_temp.assert_true((SELECT efectivo_esperado=45 FROM v_caja_resumen WHERE sesion_caja_id=1),'arqueo no incluye Yape');
UPDATE cuentas SET estado='CERRADA' WHERE id=1;
UPDATE sesiones_caja SET estado='CERRADA',cierre_en=clock_timestamp(),cerrado_por=1,efectivo_contado=45 WHERE id=1;
SELECT pg_temp.assert_true((SELECT diferencia=0 FROM v_caja_resumen WHERE sesion_caja_id=1),'cierre cuadrado');
SELECT pg_temp.expect_error($q$UPDATE metodos_pago SET es_efectivo=false WHERE codigo='EFECTIVO'$q$,'No reinterpretar');
SELECT pg_temp.expect_error($q$INSERT INTO movimientos_caja(sesion_caja_id,tipo,monto,concepto,creado_por,clave_idempotencia)
VALUES(1,'INGRESO',10,'Prueba',1,gen_random_uuid())$q$,'cerrada');
UPDATE pedidos SET estado='ENTREGADO',cerrado_en=clock_timestamp() WHERE id=1;

INSERT INTO reservas(sucursal_id,mesa_id,nombre_contacto,inicio,fin,cantidad_personas,creado_por)
VALUES (1,1,'Prueba','2026-11-01 12:00-05','2026-11-01 13:00-05',2,1);
SELECT pg_temp.expect_error($q$INSERT INTO reservas(sucursal_id,mesa_id,nombre_contacto,inicio,fin,cantidad_personas,creado_por)
VALUES (1,1,'Prueba','2026-11-01 12:30-05','2026-11-01 13:30-05',2,1)$q$,'exclusion');
INSERT INTO reservas(sucursal_id,mesa_id,nombre_contacto,inicio,fin,cantidad_personas,creado_por)
VALUES (1,1,'Prueba','2026-11-01 13:00-05','2026-11-01 14:00-05',2,1);
SELECT pg_temp.expect_error($q$INSERT INTO reservas(sucursal_id,mesa_id,nombre_contacto,inicio,fin,cantidad_personas,creado_por)
VALUES (1,1,'Prueba','2026-11-02 12:00-05','2026-11-02 13:00-05',20,1)$q$,'capacidad');

INSERT INTO series_comprobantes(sucursal_id,tipo,serie) VALUES (1,'03','B001');
INSERT INTO comprobantes(sucursal_id,cuenta_id,serie_id,tipo,serie,numero,moneda,emisor_ruc,emisor_razon_social,emisor_direccion,receptor_nombre,clave_idempotencia,creado_por)
VALUES (1,1,1,'03','B001',fn_siguiente_correlativo(1),'PEN','20000000001','PRUEBA','PRUEBA','PRUEBA',gen_random_uuid(),1);
INSERT INTO comprobante_detalles(comprobante_id,numero_linea,detalle_pedido_id,descripcion,unidad_sunat,cantidad,precio_unitario,afectacion_igv,tasa_impuesto)
VALUES (1,1,1,'Plato','NIU',2,50,'10',18);
UPDATE comprobantes SET estado='EMITIDO' WHERE id=1;
SELECT pg_temp.assert_true((SELECT total=100 FROM v_comprobante_totales WHERE comprobante_id=1),'total comprobante');
SELECT pg_temp.expect_error('UPDATE comprobante_detalles SET precio_unitario=51 WHERE id=1','inmutables');
SELECT pg_temp.expect_error($q$UPDATE comprobantes SET receptor_nombre='OTRO' WHERE id=1$q$,'inmutables');

INSERT INTO almacenes(sucursal_id,codigo,nombre) VALUES (1,'A1','Almacen');
INSERT INTO articulos_inventario(codigo,nombre,tipo,unidad_base_id) VALUES ('I1','Arroz','INSUMO',1);
INSERT INTO articulos_almacen(almacen_id,articulo_id,stock_minimo) VALUES (1,1,2);
INSERT INTO movimientos_inventario(almacen_id,tipo,direccion,motivo,creado_por,clave_idempotencia)
VALUES (1,'APERTURA',1,'Saldo inicial',1,gen_random_uuid());
INSERT INTO movimiento_inventario_detalles(movimiento_id,almacen_id,articulo_id,cantidad,costo_unitario)
VALUES (1,1,1,10,5);
SELECT fn_confirmar_movimiento(1);
SELECT fn_confirmar_movimiento(1);
SELECT pg_temp.assert_true((SELECT stock_actual=10 FROM v_existencias WHERE articulo_id=1),'apertura idempotente');
SELECT pg_temp.expect_error('UPDATE movimiento_inventario_detalles SET cantidad=11 WHERE id=1','inmutables');
SELECT pg_temp.expect_error($q$UPDATE movimientos_inventario SET motivo='otro' WHERE id=1$q$,'inmutable');
INSERT INTO movimientos_inventario(almacen_id,tipo,direccion,motivo,creado_por,clave_idempotencia)
VALUES (1,'MERMA',-1,'Prueba',1,gen_random_uuid());
INSERT INTO movimiento_inventario_detalles(movimiento_id,almacen_id,articulo_id,cantidad,costo_unitario)
VALUES (2,1,1,11,5);
SELECT pg_temp.expect_error('SELECT fn_confirmar_movimiento(2)','Stock insuficiente');
UPDATE movimiento_inventario_detalles SET cantidad=3 WHERE movimiento_id=2;
SELECT fn_confirmar_movimiento(2);
SELECT pg_temp.assert_true((SELECT stock_actual=7 FROM v_existencias WHERE articulo_id=1),'salida de stock');

INSERT INTO proveedores(tipo_documento,numero_documento,razon_social) VALUES ('6','20000000002','PROVEEDOR PRUEBA');
INSERT INTO compras(sucursal_id,almacen_id,proveedor_id,tipo_documento,serie,numero,fecha_documento,creado_por)
VALUES (1,1,1,'01','F001','1',current_date,1);
INSERT INTO compra_detalles(compra_id,numero_linea,articulo_id,cantidad_compra,factor_unidad_base,precio_unitario,tasa_impuesto)
VALUES (1,1,1,1,50,250,18);
UPDATE compras SET estado='CONFIRMADA' WHERE id=1;
INSERT INTO movimientos_inventario(almacen_id,tipo,direccion,compra_id,motivo,creado_por,clave_idempotencia)
VALUES (1,'COMPRA',1,1,'Compra',1,gen_random_uuid());
INSERT INTO movimiento_inventario_detalles(movimiento_id,almacen_id,articulo_id,compra_detalle_id,cantidad,costo_unitario)
VALUES (3,1,1,1,49,5);
SELECT pg_temp.expect_error('SELECT fn_confirmar_movimiento(3)','exactamente');
UPDATE movimiento_inventario_detalles SET cantidad=50 WHERE movimiento_id=3;
SELECT fn_confirmar_movimiento(3);
SELECT pg_temp.assert_true((SELECT stock_actual=57 FROM v_existencias WHERE articulo_id=1),'recepcion de compra y conversion');
SELECT pg_temp.expect_error('UPDATE compra_detalles SET factor_unidad_base=100 WHERE id=1','inmutables');
SELECT pg_temp.expect_error('DELETE FROM auditoria_eventos WHERE id=(SELECT min(id) FROM auditoria_eventos)','inmutable');
