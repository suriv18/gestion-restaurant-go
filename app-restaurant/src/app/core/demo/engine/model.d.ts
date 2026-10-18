import { RestaurantState, Order, ActionPayloads } from '../../../shared/models/restaurant';
export function act<K extends keyof ActionPayloads>(
  input: RestaurantState,
  action: K,
  payload: ActionPayloads[K],
): { state: RestaurantState; result?: number };
export function total(order: Order, account?: number): number;
export function paid(state: RestaurantState, order: Order, account?: number): number;
export function balance(state: RestaurantState, order: Order, account?: number): number;
export function refunded(state: RestaurantState, order: Order): number;
export function stock(state: RestaurantState, id: number): number;
export function expectedCash(state: RestaurantState): number;
export function openOrders(state: RestaurantState): Order[];
export function money(value: number): string;
