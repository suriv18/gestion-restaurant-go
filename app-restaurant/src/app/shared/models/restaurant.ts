/** Proyección local del prototipo. Los IDs y céntimos NO son DTO del API propuesto. */
export interface Metadata {
  es_activo: boolean;
  creado_por: string;
  creado_en: string;
  actualizado_por: string;
  actualizado_en: string;
  eliminado_por?: string | null;
  eliminado_en?: string | null;
  motivo_eliminacion?: string | null;
}
export interface Product extends Metadata {
  id: number;
  name: string;
  code: string;
  category: string;
  price: number;
  station: string;
  disponible: boolean;
  type: string;
}
export interface Table extends Metadata {
  id: number;
  name: string;
  zone: string;
  capacity: number;
  habilitada: boolean;
}
export type LineState = 'NUEVO' | 'PENDIENTE' | 'PREPARACION' | 'LISTO' | 'ENTREGADO' | 'CANCELADO';
export interface OrderLine {
  id: number;
  productId: number;
  name: string;
  price: number;
  qty: number;
  state: LineState;
  account: number;
  note: string;
  sent: boolean;
  consumed: boolean;
  station: string;
}
export interface Order {
  id: number;
  table: number | null;
  type: 'SALON' | 'LLEVAR' | 'DELIVERY';
  guests: number;
  waiter: string;
  customer: string;
  state: string;
  createdAt: string;
  closedAt: string | null;
  accounts: number[];
  lines: OrderLine[];
  phone?: string;
  address?: string;
  deliveryState?: string | null;
  courier?: string;
  cancelReason?: string;
}
export interface Ingredient extends Metadata {
  id: number;
  name: string;
  code: string;
  unit: string;
  min: number;
  cost: number;
  opening: number;
}
export interface Contact extends Metadata {
  id: number;
  name: string;
  document: string;
  phone: string;
  email: string;
}
export interface User extends Metadata {
  id: number;
  name: string;
  username: string;
  role: string;
  enabled: boolean;
}
export interface Reservation {
  id: number;
  name: string;
  phone: string;
  table: number;
  guests: number;
  date: string;
  start: string;
  end: string;
  state: string;
  note: string;
}
export type PaymentMethod = 'EFECTIVO' | 'YAPE' | 'PLIN' | 'TARJETA' | 'TRANSFERENCIA';
export interface Payment {
  id: number;
  order: number;
  account: number;
  method: PaymentMethod;
  amount: number;
  received: number;
  reference: string;
  at: string;
  actor: string;
  session: number;
}
export interface Refund {
  id: number;
  payment: number;
  amount: number;
  method: PaymentMethod;
  reason: string;
  at: string;
  actor: string;
  session: number;
}
export interface CashSession {
  id: number;
  state: 'ABIERTA' | 'CERRADA';
  opening: number;
  openedAt: string;
  actor: string;
  counted?: number;
  expected?: number;
  difference?: number;
  closedAt?: string;
}
export interface CashMovement {
  id: number;
  amount: number;
  reason: string;
  session: number;
  at: string;
  actor: string;
}
export interface Movement {
  id: number;
  item: number;
  qty: number;
  type: string;
  note: string;
  at: string;
  actor: string;
}
export interface PurchaseLine {
  item: number;
  qty: number;
  cost: number;
}
export interface Purchase {
  id: number;
  supplier: number;
  reference: string;
  state: string;
  at: string;
  lines: PurchaseLine[];
  receivedAt?: string;
}
export interface Invoice {
  id: number;
  number: string;
  type: string;
  order: number;
  customer: string;
  document: string;
  amount: number;
  state: string;
  at: string;
  lines?: OrderLine[];
}
export interface AuditEvent {
  id: number;
  at: string;
  actor: string;
  entity: string;
  record: string;
  action: string;
  detail: string;
}
export interface Settings {
  name: string;
  branch: string;
  address: string;
  currency: string;
}
export interface RestaurantState {
  version: number;
  settings: Settings;
  products: Product[];
  ingredients: Ingredient[];
  recipes: Record<number, { item: number; qty: number }[]>;
  tables: Table[];
  orders: Order[];
  payments: Payment[];
  refunds: Refund[];
  session: CashSession;
  cashMovements: CashMovement[];
  sessionHistory: CashSession[];
  reservations: Reservation[];
  suppliers: Contact[];
  customers: Contact[];
  users: User[];
  purchases: Purchase[];
  movements: Movement[];
  invoices: Invoice[];
  audit: AuditEvent[];
}
export type MasterEntity =
  'products' | 'customers' | 'suppliers' | 'users' | 'tables' | 'ingredients';
export interface ActionPayloads {
  newOrder: {
    type: Order['type'];
    table?: number;
    guests: number;
    customer?: string;
    phone?: string;
    address?: string;
  };
  addLine: { order: number; product: number; qty: number; account: number; note?: string };
  removeLine: { order: number; line: number };
  send: { order: number };
  lineState: { order: number; line: number; state: LineState };
  split: { order: number; lines: number[] };
  cancelOrder: { order: number; reason: string };
  pay: {
    order: number;
    account: number;
    method: PaymentMethod;
    amount: number;
    received: number;
    reference: string;
  };
  closeOrder: { order: number };
  refund: { payment: number; amount: number; reason: string };
  cashMovement: { type: 'ENTRADA' | 'SALIDA'; amount: number; reason: string };
  closeCash: { counted: number };
  openCash: { opening: number };
  reservation: Omit<Reservation, 'id' | 'state'>;
  reservationState: { id: number; state: string };
  deliveryState: { order: number; state: string; courier?: string };
  movement: { item: number; type: string; qty: number; reason: string };
  purchase: { supplier: number; reference: string; lines: PurchaseLine[] };
  receivePurchase: { id: number };
  cancelPurchase: { id: number };
  saveProduct: {
    id?: number;
    name: string;
    code: string;
    price: number;
    category: string;
    station: string;
    type: string;
  };
  availability: { id: number };
  softDelete: { entity: MasterEntity; id: number; reason?: string };
  restore: { entity: MasterEntity; id: number };
  saveContact: {
    entity: 'customers' | 'suppliers';
    id?: number;
    name: string;
    document: string;
    phone: string;
    email: string;
  };
  userRole: { id: number; role: string };
  tableEnabled: { id: number };
  settings: Omit<Settings, 'currency'>;
  invoice: { order: number; type: string; document: string; customer: string };
}
