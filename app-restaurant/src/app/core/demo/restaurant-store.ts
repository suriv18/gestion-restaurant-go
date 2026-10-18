import { isRestaurantState } from './state-validator';
import { Injectable, computed, signal } from '@angular/core';
import { ActionPayloads, Order, RestaurantState } from '../../shared/models/restaurant';
import { seed } from './engine/data.js';
import { act, balance, expectedCash, openOrders, paid, stock, total } from './engine/model.js';

const STORAGE_KEY = 'app-restaurant.demo.v1';
@Injectable({ providedIn: 'root' })
export class RestaurantStore {
  private readonly current = signal<RestaurantState>(seed());
  readonly state = this.current.asReadonly();
  readonly notice = signal('');
  readonly storageWarning = signal('');
  readonly orders = computed(() => openOrders(this.state()));
  readonly cash = computed(() => expectedCash(this.state()));
  readonly due = computed(() => this.orders().reduce((sum, order) => sum + this.balance(order), 0));
  readonly ready = computed(
    () =>
      this.orders()
        .flatMap((o) => o.lines)
        .filter((l) => l.state === 'LISTO').length,
  );
  readonly lowStock = computed(() =>
    this.state().ingredients.filter((i) => i.es_activo && this.stock(i.id) < i.min),
  );
  readonly enabledTables = computed(() =>
    this.state().tables.filter((t) => t.es_activo && t.habilitada),
  );
  readonly occupied = computed(() => this.orders().filter((o) => o.table !== null).length);
  private timer?: ReturnType<typeof setTimeout>;

  constructor() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed: unknown = JSON.parse(raw);
        if (isRestaurantState(parsed)) this.current.set(parsed);
        else
          this.storageWarning.set(
            'Los datos guardados no son compatibles. Se cargaron ejemplos nuevos.',
          );
      }
    } catch {
      this.storageWarning.set(
        'No se pudieron recuperar los datos locales. La sesión usa datos de ejemplo.',
      );
    }
  }
  execute<K extends keyof ActionPayloads>(
    action: K,
    payload: ActionPayloads[K],
  ): number | undefined {
    const result = act(this.state(), action, payload);
    this.persist(result.state);
    this.current.set(result.state);
    this.notify('Cambio guardado en la demostración.');
    return result.result;
  }
  run<K extends keyof ActionPayloads>(action: K, payload: ActionPayloads[K]): boolean {
    try {
      this.execute(action, payload);
      return true;
    } catch (error) {
      this.notify(error instanceof Error ? error.message : 'No se pudo completar la acción.');
      return false;
    }
  }
  reset(): void {
    const state = seed();
    this.persist(state);
    this.current.set(state);
    this.notify('Datos de demostración restablecidos.');
  }
  total = total;
  balance(order: Order, account?: number): number {
    return balance(this.state(), order, account);
  }
  paid(order: Order, account?: number): number {
    return paid(this.state(), order, account);
  }
  stock(id: number): number {
    return stock(this.state(), id);
  }
  tableName(order: Order): string {
    return (
      this.state().tables.find((t) => t.id === order.table)?.name ??
      (order.type === 'DELIVERY' ? 'Delivery' : 'Para llevar')
    );
  }
  notify(message: string): void {
    clearTimeout(this.timer);
    this.notice.set(message);
    this.timer = setTimeout(() => this.notice.set(''), 6000);
  }
  private persist(state: RestaurantState): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
      this.storageWarning.set('');
    } catch {
      this.storageWarning.set(
        'El navegador no permite guardar. Los cambios solo durarán durante esta sesión.',
      );
    }
  }
}
