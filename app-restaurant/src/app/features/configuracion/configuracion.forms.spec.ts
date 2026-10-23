import { cambiosDe, entero, opcional, texto } from './configuracion.forms';

describe('Normalización de formularios de Configuración', () => {
  it('texto recorta espacios', () => {
    expect(texto('  Terraza ')).toBe('Terraza');
  });

  it('opcional devuelve null si está vacío y el texto recortado si no', () => {
    expect(opcional('   ')).toBeNull();
    expect(opcional(' 12.50 ')).toBe('12.50');
  });

  it('entero convierte a número', () => {
    expect(entero('8')).toBe(8);
    expect(entero(' 12 ')).toBe(12);
  });

  it('cambiosDe devuelve solo los campos que cambiaron, incluido el vaciado a null', () => {
    const actual = { nombre: 'Salón', orden: 1, telefono: '999', ubigeo: null };
    const nuevo = { nombre: 'Salón', orden: 2, telefono: null, ubigeo: null };
    expect(cambiosDe(actual, nuevo)).toEqual({ orden: 2, telefono: null });
  });

  it('cambiosDe devuelve un objeto vacío si nada cambió', () => {
    expect(cambiosDe({ a: 'x' }, { a: 'x' })).toEqual({});
  });
});
