import { MoneyPipe, toCents } from './money';
describe('Importes de la demostración', () => {
  it('convierte decimales sin redondear silenciosamente', () => {
    expect(toCents('0.29')).toBe(29);
    expect(toCents('32')).toBe(3200);
    expect(toCents('32.5')).toBe(3250);
    expect(toCents(' 120.09 ')).toBe(12009);
    expect(toCents('0')).toBe(0);
  });
  it.each(['-1', '1.999', '1e3', 'NaN', 'Infinity', '', '0x10', '10,50', '9007199254740991'])(
    'rechaza el importe inválido %s',
    (value) => expect(() => toCents(value)).toThrow(),
  );
  it('presenta céntimos en soles', () =>
    expect(new MoneyPipe().transform(3250)).toMatch(/32[.,]50/));
});
