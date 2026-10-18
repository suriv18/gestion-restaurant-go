import { seed } from './engine/data.js';
import { isRestaurantState } from './state-validator';
describe('Validación del estado persistido', () => {
  it('acepta el estado inicial', () => expect(isRestaurantState(seed())).toBe(true));
  it('rechaza registros incompletos que romperían las pantallas', () => {
    const data = seed();
    const broken = { ...data, products: [{ id: 1 }] };
    expect(isRestaurantState(broken)).toBe(false);
    expect(isRestaurantState({ ...data, orders: [null] })).toBe(false);
    expect(isRestaurantState({ ...data, recipes: 'incorrecto' })).toBe(false);
  });
  it('rechaza valores de inventario no finitos y líneas inválidas', () => {
    const data = seed();
    expect(
      isRestaurantState({ ...data, movements: [{ ...data.movements[0], qty: Infinity }] }),
    ).toBe(false);
    expect(isRestaurantState({ ...data, orders: [{ ...data.orders[0], lines: [{}] }] })).toBe(
      false,
    );
  });
});
