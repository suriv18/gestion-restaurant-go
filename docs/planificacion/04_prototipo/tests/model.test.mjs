import assert from 'node:assert/strict';
import {seed,today} from '../dist/assets/data.js';
import {act,total,paid,balance,stock,expectedCash} from '../dist/assets/model.js';
let passed=0;
const test=(name,fn)=>{fn();passed++;console.log('PASS',name);};
const run=(s,a,p)=>act(s,a,p).state;
test('Abrir, agregar, enviar, consumir una vez, servir, pagar en dos métodos y liberar mesa',()=>{
 let s=seed();const r=act(s,'newOrder',{type:'SALON',table:1,guests:2});s=r.state;const id=r.result;
 s=run(s,'addLine',{order:id,product:1,qty:1,account:1,note:'Sin cebolla'});
 const o=()=>s.orders.find(o=>o.id===id),line=o().lines[0].id;
 assert.equal(total(o()),3200);assert.throws(()=>run(s,'pay',{order:id,account:1,amount:3200,method:'EFECTIVO',received:3200}),/Envía/);
 s=run(s,'send',{order:id});let q=stock(s,1);s=run(s,'lineState',{order:id,line,state:'PREPARACION'});assert.equal(stock(s,1),Math.round((q-.2)*1000)/1000);
 assert.throws(()=>run(s,'lineState',{order:id,line,state:'PREPARACION'}),/transición/);assert.equal(stock(s,1),Math.round((q-.2)*1000)/1000);
 s=run(s,'lineState',{order:id,line,state:'LISTO'});s=run(s,'lineState',{order:id,line,state:'ENTREGADO'});
 const cash=expectedCash(s);s=run(s,'pay',{order:id,account:1,amount:1200,received:2000,method:'EFECTIVO'});assert.equal(expectedCash(s),cash+1200);
 assert.throws(()=>run(s,'addLine',{order:id,product:1,qty:1,account:1}),/pagos/);
 assert.throws(()=>run(s,'pay',{order:id,account:1,amount:2200,method:'YAPE',reference:'123'}),/saldo/);
 s=run(s,'pay',{order:id,account:1,amount:2000,method:'YAPE',reference:'123'});assert.equal(expectedCash(s),cash+1200);assert.equal(balance(s,o()),0);
 s=run(s,'closeOrder',{order:id});assert.ok(o().closedAt);
 assert.doesNotThrow(()=>run(s,'newOrder',{type:'SALON',table:1,guests:2}));
});
test('Recibir una compra aumenta stock una vez y mantiene referencias únicas',()=>{let s=seed();const q=stock(s,1);s=run(s,'receivePurchase',{id:1});assert.equal(stock(s,1),q+5);assert.throws(()=>run(s,'receivePurchase',{id:1}),/procesada/);assert.throws(()=>run(s,'purchase',{supplier:1,reference:'F001-002458',lines:[{item:1,qty:1,cost:2800}]}),/existe/);});
test('Stock negativo y consumo parcial se revierten sin afectar los datos',()=>{let s=seed();const before=structuredClone(s);assert.throws(()=>run(s,'movement',{item:1,type:'MERMA',qty:999,reason:'prueba'}),/negativo/);assert.deepEqual(s,before);s=run(s,'movement',{item:3,type:'MERMA',qty:18.2,reason:'prueba'});s=run(s,'addLine',{order:1042,product:1,qty:1,account:1});s=run(s,'send',{order:1042});const last=s.orders.find(o=>o.id===1042).lines.at(-1),copy=structuredClone(s);assert.throws(()=>run(s,'lineState',{order:1042,line:last.id,state:'PREPARACION'}),/insuficiente/);assert.deepEqual(s,copy);});
test('La baja y restauración preservan identidad, historial y códigos únicos',()=>{let s=seed(),initial=total(s.orders[0]);assert.throws(()=>run(s,'softDelete',{entity:'products',id:1,reason:''}),/obligatorio/);s=run(s,'softDelete',{entity:'products',id:1,reason:'Retiro de carta'});const p=s.products[0];assert.equal(p.es_activo,false);assert.equal(p.eliminado_por,'Valeria Torres');assert.equal(total(s.orders[0]),initial);assert.throws(()=>run(s,'saveProduct',{name:'Otro',code:'PL-001',price:200,category:'Fondos'}),/existe/);s=run(s,'restore',{entity:'products',id:1});assert.equal(s.products[0].id,1);assert.equal(s.products[0].es_activo,true);assert.equal(s.products[0].motivo_eliminacion,null);assert.ok(s.audit.some(e=>e.action==='BAJA LÓGICA'));assert.ok(s.audit.some(e=>e.action==='RESTAURAR'));});
test('Disponibilidad y baja lógica se controlan por separado',()=>{let s=seed();s=run(s,'availability',{id:1});assert.equal(s.products[0].disponible,false);assert.equal(s.products[0].es_activo,true);s=run(s,'softDelete',{entity:'products',id:1,reason:'Baja'});s=run(s,'restore',{entity:'products',id:1});assert.equal(s.products[0].disponible,false);});
test('Reservas rechazan solapamiento pero aceptan intervalos adyacentes',()=>{let s=seed();const r={name:'Prueba',phone:'999999999',table:5,guests:2,date:today(),start:'20:00',end:'21:00'};assert.throws(()=>run(s,'reservation',r),/horario/);assert.doesNotThrow(()=>run(s,'reservation',{...r,start:'20:30',end:'21:30'}));assert.throws(()=>run(s,'reservation',{...r,guests:8}),/capacidad/);});
test('Cuentas separadas conservan total y cobros limitados al saldo individual',()=>{let s=seed(),o=s.orders.find(o=>o.id===1042),amount=total(o);s=run(s,'split',{order:1042,lines:[1]});o=s.orders.find(o=>o.id===1042);assert.equal(total(o),amount);assert.equal(total(o,2),6400);assert.equal(total(o,1),4200);s=run(s,'pay',{order:1042,account:2,amount:6400,method:'EFECTIVO',received:6400});assert.throws(()=>run(s,'pay',{order:1042,account:2,amount:100,method:'YAPE',reference:'X'}),/saldo/);});
test('Precios nuevos no cambian consumos históricos ni documentos',()=>{let s=seed();const initial=total(s.orders[0]);s=run(s,'saveProduct',{id:1,name:'Lomo nuevo',code:'PL-001',price:5000,category:'Fondos',station:'Cocina',type:'PREPARADO'});assert.equal(total(s.orders[0]),initial);assert.equal(s.orders[0].lines[0].name,'Lomo saltado');assert.equal(s.invoices[0].amount,8000);});
test('Devoluciones compensan caja sin reescribir el cobro original',()=>{let s=seed(),cash=expectedCash(s),payment=structuredClone(s.payments[0]);s=run(s,'refund',{payment:1,amount:3000,reason:'Devolución parcial'});assert.equal(expectedCash(s),cash-3000);assert.deepEqual(s.payments[0],payment);assert.throws(()=>run(s,'refund',{payment:1,amount:5100,reason:'Prueba'}),/supera/);assert.equal(balance(s,s.orders.find(o=>o.id===1039)),0);});
test('Cierre, conteo de diferencia y apertura de siguiente turno',()=>{let s=seed();const expected=expectedCash(s);s=run(s,'closeCash',{counted:expected-100});assert.equal(s.session.difference,-100);assert.equal(s.sessionHistory.length,1);assert.throws(()=>run(s,'pay',{order:1040,account:1,amount:100,method:'EFECTIVO',received:100}),/Abre/);s=run(s,'openCash',{opening:50000});assert.equal(expectedCash(s),50000);assert.equal(s.session.id,2);});
test('Delivery exige preparación completa y un repartidor antes de salir',()=>{let s=seed();assert.throws(()=>run(s,'deliveryState',{order:1043,state:'EN_RUTA',courier:'Marco'}),/Cocina/);s=run(s,'lineState',{order:1043,line:8,state:'LISTO'});s=run(s,'lineState',{order:1043,line:9,state:'PREPARACION'});s=run(s,'lineState',{order:1043,line:9,state:'LISTO'});s=run(s,'deliveryState',{order:1043,state:'EN_RUTA',courier:'Marco'});assert.equal(s.orders.find(o=>o.id===1043).courier,'Marco');s=run(s,'deliveryState',{order:1043,state:'ENTREGADO'});assert.equal(s.orders.find(o=>o.id===1043).deliveryState,'ENTREGADO');});
test('Un comprobante simulado se crea solo para pedidos pagados y una sola vez',()=>{let s=seed();assert.throws(()=>run(s,'invoice',{order:1042,type:'Boleta'}),/cobrado/);assert.throws(()=>run(s,'invoice',{order:1039,type:'Boleta'}),/ya tiene/);s=run(s,'invoice',{order:1038,type:'Boleta'});assert.equal(s.invoices.at(-1).amount,3100);assert.equal(s.invoices.at(-1).state,'SIMULADO');});
console.log(`\n${passed} escenarios de negocio verificados.`);

// V4: una mesa usa actividad simple; no almacena auditoría de eliminación.
{
 let s=seed();const id=s.tables.find(t=>!s.orders.some(o=>o.table===t.id&&!o.closedAt&&o.state!=='CANCELADO')).id;
 s=run(s,'softDelete',{entity:'tables',id});const t=s.tables.find(t=>t.id===id);
 assert.equal(t.es_activo,false);assert.equal('eliminado_por' in t,false);assert.equal('eliminado_en' in t,false);assert.equal('motivo_eliminacion' in t,false);
 s=run(s,'restore',{entity:'tables',id});assert.equal(s.tables.find(t=>t.id===id).es_activo,true);
 console.log('PASS Mesa con desactivación simple sin marcadores de eliminación');
}
