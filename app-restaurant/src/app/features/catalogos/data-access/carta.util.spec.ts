import {
  compararFechas,
  factorValido,
  fechaValida,
  formatearSoles,
  formatearTasa,
  normalizarDecimal,
  precioValido,
  tasaValida,
  vigenteEn,
} from './carta.util';

describe('Utilidades de la carta', () => {
  it('formatea soles solo con texto', () => {
    expect(formatearSoles('25.5')).toBe('S/ 25.50');
    expect(formatearSoles('25')).toBe('S/ 25.00');
    expect(formatearSoles('25.00')).toBe('S/ 25.00');
    expect(formatearSoles('0.1')).toBe('S/ 0.10');
    expect(formatearSoles('007.5')).toBe('S/ 7.50');
    expect(formatearSoles('12345678901234567890.99')).toBe('S/ 12345678901234567890.99');
    expect(formatearSoles('abc')).toBe('abc');
    expect(formatearSoles(null)).toBe('');
  });

  it('valida precios (>= 0, hasta 2 decimales)', () => {
    for (const ok of ['0', '25', '25.5', '25.50', '0.05']) expect(precioValido(ok)).toBe(true);
    for (const mal of ['', '-1', '25.555', '1,5', 'abc', '.5', '25.']) {
      expect(precioValido(mal)).toBe(false);
    }
  });

  it('valida el factor de consumo (> 0, hasta 6 decimales)', () => {
    for (const ok of ['1', '0.5', '0.000001', '12.250000']) expect(factorValido(ok)).toBe(true);
    for (const mal of ['0', '0.0', '-1', '1.1234567', '', 'x']) {
      expect(factorValido(mal)).toBe(false);
    }
  });

  it('valida tasas entre 0 y 100 con hasta 4 decimales', () => {
    for (const ok of ['0', '18', '10.5', '18.0000', '100', '100.0000', '99.9999']) {
      expect(tasaValida(ok)).toBe(true);
    }
    for (const mal of ['100.0001', '101', '-1', '18.12345', '', '1,5', '1000', '999']) {
      expect(tasaValida(mal)).toBe(false);
    }
  });

  it('valida fechas ISO reales', () => {
    expect(fechaValida('2026-02-28')).toBe(true);
    expect(fechaValida('2028-02-29')).toBe(true);
    expect(fechaValida('2026-02-29')).toBe(false);
    expect(fechaValida('2026-13-01')).toBe(false);
    expect(fechaValida('26-01-01')).toBe(false);
  });

  it('normaliza decimales para compararlos', () => {
    expect(normalizarDecimal('25.50')).toBe('25.5');
    expect(normalizarDecimal('25.00')).toBe('25');
    expect(normalizarDecimal('1.000000')).toBe('1');
    expect(normalizarDecimal('007')).toBe('7');
    expect(normalizarDecimal(' 0.10 ')).toBe('0.1');
    expect(normalizarDecimal('abc')).toBe('abc');
  });

  it('formatea tasas', () => {
    expect(formatearTasa('18.0000')).toBe('18 %');
    expect(formatearTasa('10.5000')).toBe('10.5 %');
  });

  it('compara vigencias como texto', () => {
    expect(compararFechas('2026-01-01', '2026-10-25')).toBeLessThan(0);
    expect(compararFechas('2026-10-25', '2026-10-25')).toBe(0);
    expect(compararFechas('2027-01-01', '2026-10-25')).toBeGreaterThan(0);
    expect(vigenteEn('2026-01-01', null, '2026-10-25')).toBe(true);
    expect(vigenteEn('2026-01-01', '2026-10-25', '2026-10-25')).toBe(false);
    expect(vigenteEn('2026-01-01', '2026-10-26', '2026-10-25')).toBe(true);
    expect(vigenteEn('2026-11-01', null, '2026-10-25')).toBe(false);
  });
});
