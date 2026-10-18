import { TestBed } from '@angular/core/testing';
import { RestaurantStore } from './restaurant-store';
import { seed, today } from './engine/data.js';
import { act, balance, expectedCash, stock, total } from './engine/model.js';
import { ActionPayloads, RestaurantState } from '../../shared/models/restaurant';
const run = <K extends keyof ActionPayloads>(
  s: RestaurantState,
  action: K,
  payload: ActionPayloads[K],
) => act(s, action, payload).state;
describe('Reglas del restaurante', () => {
  it('completa atención, consumo único, pagos mixtos y liberación de mesa', () => {
    const created = act(seed(), 'newOrder', { type: 'SALON', table: 1, guests: 2 });
    let s = created.state;
    const id = created.result!;
    s = run(s, 'addLine', { order: id, product: 1, qty: 1, account: 1, note: 'Sin cebolla' });
    const order = () => s.orders.find((o) => o.id === id)!;
    const line = order().lines[0].id;
    expect(() =>
      run(s, 'pay', {
        order: id,
        account: 1,
        amount: 3200,
        method: 'EFECTIVO',
        received: 3200,
        reference: '',
      }),
    ).toThrow(/Envía/);
    s = run(s, 'send', { order: id });
    const initial = stock(s, 1);
    s = run(s, 'lineState', { order: id, line, state: 'PREPARACION' });
    expect(stock(s, 1)).toBeCloseTo(initial - 0.2);
    expect(() => run(s, 'lineState', { order: id, line, state: 'PREPARACION' })).toThrow(
      /transición/,
    );
    s = run(s, 'lineState', { order: id, line, state: 'LISTO' });
    s = run(s, 'lineState', { order: id, line, state: 'ENTREGADO' });
    const cash = expectedCash(s);
    s = run(s, 'pay', {
      order: id,
      account: 1,
      amount: 1200,
      received: 2000,
      method: 'EFECTIVO',
      reference: '',
    });
    expect(expectedCash(s)).toBe(cash + 1200);
    expect(() => run(s, 'addLine', { order: id, product: 1, qty: 1, account: 1 })).toThrow(/pagos/);
    expect(() =>
      run(s, 'pay', {
        order: id,
        account: 1,
        amount: 2200,
        method: 'YAPE',
        reference: '123',
        received: 2200,
      }),
    ).toThrow(/saldo/);
    s = run(s, 'pay', {
      order: id,
      account: 1,
      amount: 2000,
      method: 'YAPE',
      reference: '123',
      received: 2000,
    });
    expect(expectedCash(s)).toBe(cash + 1200);
    expect(balance(s, order())).toBe(0);
    s = run(s, 'closeOrder', { order: id });
    expect(order().closedAt).toBeTruthy();
    expect(() => run(s, 'newOrder', { type: 'SALON', table: 1, guests: 2 })).not.toThrow();
  });
  it('revierte todo el consumo si falta uno de los insumos', () => {
    let s = seed();
    s = run(s, 'movement', { item: 3, type: 'MERMA', qty: 18.2, reason: 'Prueba' });
    s = run(s, 'addLine', { order: 1042, product: 1, qty: 1, account: 1 });
    s = run(s, 'send', { order: 1042 });
    const line = s.orders.find((o) => o.id === 1042)!.lines.at(-1)!.id;
    const before = structuredClone(s);
    expect(() => run(s, 'lineState', { order: 1042, line, state: 'PREPARACION' })).toThrow(
      /insuficiente/,
    );
    expect(s).toEqual(before);
    expect(() =>
      run(s, 'movement', { item: 1, type: 'MERMA', qty: 999, reason: 'Prueba' }),
    ).toThrow(/negativo/);
  });
  it('recibe una compra una sola vez y evita referencias duplicadas', () => {
    let s = seed();
    const initial = stock(s, 1);
    s = run(s, 'receivePurchase', { id: 1 });
    expect(stock(s, 1)).toBe(initial + 5);
    expect(() => run(s, 'receivePurchase', { id: 1 })).toThrow(/procesada/);
    expect(() =>
      run(s, 'purchase', {
        supplier: 1,
        reference: 'F001-002458',
        lines: [{ item: 1, qty: 1, cost: 100 }],
      }),
    ).toThrow(/existe/);
  });
  it('preserva identidad y precios históricos al editar o dar de baja', () => {
    let s = seed();
    const initial = total(s.orders[0]);
    s = run(s, 'saveProduct', {
      id: 1,
      name: 'Lomo nuevo',
      code: 'PL-001',
      price: 5000,
      category: 'Fondos',
      station: 'Cocina',
      type: 'PREPARADO',
    });
    expect(total(s.orders[0])).toBe(initial);
    expect(s.orders[0].lines[0].name).toBe('Lomo saltado');
    expect(() => run(s, 'softDelete', { entity: 'products', id: 1, reason: '' })).toThrow(
      /obligatorio/,
    );
    s = run(s, 'softDelete', { entity: 'products', id: 1, reason: 'Retiro' });
    expect(s.products[0].es_activo).toBe(false);
    expect(() =>
      run(s, 'saveProduct', {
        name: 'Otro',
        code: 'PL-001',
        price: 200,
        category: 'Fondos',
        station: 'Cocina',
        type: 'PREPARADO',
      }),
    ).toThrow(/existe/);
    s = run(s, 'restore', { entity: 'products', id: 1 });
    expect(s.products[0].id).toBe(1);
    expect(s.products[0].motivo_eliminacion).toBeNull();
  });
  it('separa disponibilidad de producto y baja lógica', () => {
    let s = run(seed(), 'availability', { id: 1 });
    s = run(s, 'softDelete', { entity: 'products', id: 1, reason: 'Baja' });
    s = run(s, 'restore', { entity: 'products', id: 1 });
    expect(s.products[0].es_activo).toBe(true);
    expect(s.products[0].disponible).toBe(false);
  });
  it('rechaza reservas superpuestas y permite intervalos adyacentes', () => {
    const s = seed();
    const payload = {
      name: 'Prueba',
      phone: '999999999',
      table: 5,
      guests: 2,
      date: today(),
      start: '20:00',
      end: '21:00',
      note: '',
    };
    expect(() => run(s, 'reservation', payload)).toThrow(/horario/);
    expect(() => run(s, 'reservation', { ...payload, start: '20:30', end: '21:30' })).not.toThrow();
    expect(() => run(s, 'reservation', { ...payload, guests: 8 })).toThrow(/capacidad/);
  });
  it('limita cada cobro al saldo de la cuenta dividida', () => {
    let s = run(seed(), 'split', { order: 1042, lines: [1] });
    const o = s.orders.find((o) => o.id === 1042)!;
    expect(total(o, 2)).toBe(6400);
    expect(total(o, 1)).toBe(4200);
    s = run(s, 'pay', {
      order: 1042,
      account: 2,
      amount: 6400,
      received: 6400,
      method: 'EFECTIVO',
      reference: '',
    });
    expect(() =>
      run(s, 'pay', {
        order: 1042,
        account: 2,
        amount: 100,
        received: 100,
        method: 'YAPE',
        reference: 'X',
      }),
    ).toThrow(/saldo/);
  });
  it('registra devoluciones compensatorias sin alterar el pago ni reabrir saldo', () => {
    let s = seed();
    const payment = structuredClone(s.payments[0]);
    const cash = expectedCash(s);
    s = run(s, 'refund', { payment: 1, amount: 3000, reason: 'Parcial' });
    expect(expectedCash(s)).toBe(cash - 3000);
    expect(s.payments[0]).toEqual(payment);
    expect(
      balance(
        s,
        s.orders.find((o) => o.id === 1039)!,
      ),
    ).toBe(0);
    expect(() => run(s, 'refund', { payment: 1, amount: 5100, reason: 'Prueba' })).toThrow(
      /supera/,
    );
  });
  it('conserva diferencias de cierre y reinicia el fondo del siguiente turno', () => {
    let s = seed();
    s = run(s, 'closeCash', { counted: expectedCash(s) - 100 });
    expect(s.session.difference).toBe(-100);
    expect(s.sessionHistory).toHaveLength(1);
    expect(() =>
      run(s, 'pay', {
        order: 1040,
        account: 1,
        amount: 100,
        received: 100,
        method: 'EFECTIVO',
        reference: '',
      }),
    ).toThrow(/Abre/);
    s = run(s, 'openCash', { opening: 50000 });
    expect(expectedCash(s)).toBe(50000);
    expect(s.session.id).toBe(2);
  });
  it('exige preparación y repartidor antes de despachar delivery', () => {
    let s = seed();
    expect(() =>
      run(s, 'deliveryState', { order: 1043, state: 'EN_RUTA', courier: 'Marco' }),
    ).toThrow(/Cocina/);
    s = run(s, 'lineState', { order: 1043, line: 8, state: 'LISTO' });
    s = run(s, 'lineState', { order: 1043, line: 9, state: 'PREPARACION' });
    s = run(s, 'lineState', { order: 1043, line: 9, state: 'LISTO' });
    expect(() => run(s, 'deliveryState', { order: 1043, state: 'EN_RUTA', courier: '' })).toThrow(
      /repartidor/,
    );
    s = run(s, 'deliveryState', { order: 1043, state: 'EN_RUTA', courier: 'Marco' });
    s = run(s, 'deliveryState', { order: 1043, state: 'ENTREGADO' });
    expect(s.orders.find((o) => o.id === 1043)!.deliveryState).toBe('ENTREGADO');
  });
  it('simula un comprobante solo una vez por pedido pagado', () => {
    let s = seed();
    expect(() =>
      run(s, 'invoice', { order: 1042, type: 'Boleta', document: '', customer: '' }),
    ).toThrow(/cobrado/);
    expect(() =>
      run(s, 'invoice', { order: 1039, type: 'Boleta', document: '', customer: '' }),
    ).toThrow(/ya tiene/);
    s = run(s, 'invoice', { order: 1038, type: 'Boleta', document: '', customer: '' });
    expect(s.invoices.at(-1)!.amount).toBe(3100);
  });
  it('desactiva mesas sin marcadores de eliminación y protege las ocupadas', () => {
    let s = seed();
    expect(() => run(s, 'tableEnabled', { id: 4 })).toThrow(/ocupada/);
    s = run(s, 'softDelete', { entity: 'tables', id: 1 });
    expect(s.tables[0].es_activo).toBe(false);
    expect(s.tables[0]).not.toHaveProperty('eliminado_en');
    s = run(s, 'restore', { entity: 'tables', id: 1 });
    expect(s.tables[0].es_activo).toBe(true);
  });
});
describe('RestaurantStore: persistencia y errores', () => {
  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({});
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });
  it('recupera cambios persistidos y mantiene Signals derivados', () => {
    const store = TestBed.inject(RestaurantStore);
    store.execute('newOrder', { type: 'LLEVAR', guests: 1 });
    expect(store.orders()).toHaveLength(5);
    expect(new RestaurantStore().orders()).toHaveLength(5);
  });
  it('no cambia estado ni persistencia tras una operación rechazada', () => {
    const store = TestBed.inject(RestaurantStore);
    const before = store.state();
    expect(store.run('newOrder', { type: 'SALON', table: 4, guests: 1 })).toBe(false);
    expect(store.state()).toBe(before);
    expect(localStorage.getItem('app-restaurant.demo.v1')).toBeNull();
  });
  it('recupera datos dañados sin impedir el arranque', () => {
    localStorage.setItem('app-restaurant.demo.v1', '{broken');
    const store = TestBed.inject(RestaurantStore);
    expect(store.orders()).toHaveLength(4);
    expect(store.storageWarning()).not.toBe('');
  });
  it('avisa cuando almacenamiento no está disponible y permite continuar la sesión', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('Quota', 'QuotaExceededError');
    });
    const store = TestBed.inject(RestaurantStore);
    store.execute('newOrder', { type: 'LLEVAR', guests: 1 });
    expect(store.orders()).toHaveLength(5);
    expect(store.storageWarning()).toContain('solo durarán');
  });
});
