import { RestaurantState } from '../../shared/models/restaurant';
type Row = Record<string, unknown>;
type Shape = Record<string, 'string' | 'number' | 'boolean'>;
const meta: Shape = {
  id: 'number',
  name: 'string',
  es_activo: 'boolean',
  creado_por: 'string',
  creado_en: 'string',
  actualizado_por: 'string',
  actualizado_en: 'string',
};
const event: Shape = { id: 'number', at: 'string', actor: 'string' };
const line: Shape = {
  id: 'number',
  productId: 'number',
  name: 'string',
  price: 'number',
  qty: 'number',
  state: 'string',
  account: 'number',
  note: 'string',
  sent: 'boolean',
  consumed: 'boolean',
  station: 'string',
};
const session: Shape = {
  id: 'number',
  state: 'string',
  opening: 'number',
  openedAt: 'string',
  actor: 'string',
};
function record(v: unknown): v is Row {
  return !!v && typeof v === 'object' && !Array.isArray(v);
}
function matches(v: unknown, shape: Shape): v is Row {
  return (
    record(v) &&
    Object.entries(shape).every(
      ([key, type]) => typeof v[key] === type && (type !== 'number' || Number.isFinite(v[key])),
    )
  );
}
function rows(v: unknown, shape: Shape): v is Row[] {
  return Array.isArray(v) && v.every((row) => matches(row, shape));
}
/** Valida el almacenamiento no confiable antes de que llegue a las plantillas. No autoriza operaciones. */
export function isRestaurantState(v: unknown): v is RestaurantState {
  if (
    !record(v) ||
    v['version'] !== 1 ||
    !matches(v['settings'], {
      name: 'string',
      branch: 'string',
      address: 'string',
      currency: 'string',
    }) ||
    !matches(v['session'], session)
  )
    return false;
  if (!['ABIERTA', 'CERRADA'].includes(String(v['session']['state']))) return false;
  const collections: Record<string, Shape> = {
    products: {
      ...meta,
      code: 'string',
      category: 'string',
      price: 'number',
      station: 'string',
      disponible: 'boolean',
      type: 'string',
    },
    ingredients: {
      ...meta,
      code: 'string',
      unit: 'string',
      min: 'number',
      cost: 'number',
      opening: 'number',
    },
    tables: { ...meta, zone: 'string', capacity: 'number', habilitada: 'boolean' },
    orders: {
      id: 'number',
      type: 'string',
      guests: 'number',
      waiter: 'string',
      customer: 'string',
      state: 'string',
      createdAt: 'string',
    },
    payments: {
      ...event,
      order: 'number',
      account: 'number',
      method: 'string',
      amount: 'number',
      received: 'number',
      reference: 'string',
      session: 'number',
    },
    refunds: {
      ...event,
      payment: 'number',
      amount: 'number',
      method: 'string',
      reason: 'string',
      session: 'number',
    },
    cashMovements: { ...event, amount: 'number', reason: 'string', session: 'number' },
    sessionHistory: session,
    reservations: {
      id: 'number',
      name: 'string',
      phone: 'string',
      table: 'number',
      guests: 'number',
      date: 'string',
      start: 'string',
      end: 'string',
      state: 'string',
      note: 'string',
    },
    suppliers: { ...meta, document: 'string', phone: 'string', email: 'string' },
    customers: { ...meta, document: 'string', phone: 'string', email: 'string' },
    users: { ...meta, username: 'string', role: 'string', enabled: 'boolean' },
    purchases: {
      id: 'number',
      supplier: 'number',
      reference: 'string',
      state: 'string',
      at: 'string',
    },
    movements: { ...event, item: 'number', qty: 'number', type: 'string', note: 'string' },
    invoices: {
      id: 'number',
      number: 'string',
      type: 'string',
      order: 'number',
      customer: 'string',
      document: 'string',
      amount: 'number',
      state: 'string',
      at: 'string',
    },
    audit: { ...event, entity: 'string', record: 'string', action: 'string', detail: 'string' },
  };
  if (!Object.entries(collections).every(([key, shape]) => rows(v[key], shape))) return false;
  const orders = v['orders'] as Row[];
  if (
    !orders.every(
      (o) =>
        rows(o['lines'], line) &&
        Array.isArray(o['accounts']) &&
        o['accounts'].every((a) => Number.isSafeInteger(a)) &&
        (o['table'] === null || Number.isSafeInteger(o['table'])) &&
        (o['closedAt'] === null || typeof o['closedAt'] === 'string'),
    )
  )
    return false;
  if (
    !(v['purchases'] as Row[]).every((p) =>
      rows(p['lines'], { item: 'number', qty: 'number', cost: 'number' }),
    )
  )
    return false;
  if (!(v['invoices'] as Row[]).every((i) => i['lines'] === undefined || rows(i['lines'], line)))
    return false;
  return (
    record(v['recipes']) &&
    Object.values(v['recipes']).every((recipe) => rows(recipe, { item: 'number', qty: 'number' }))
  );
}
