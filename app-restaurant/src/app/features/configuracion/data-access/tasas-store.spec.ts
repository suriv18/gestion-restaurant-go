import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import type { Impuesto, TasaImpuesto } from '../../catalogos/data-access/carta.models';
import { ImpuestosApi } from '../../catalogos/data-access/impuestos-api';
import { TasasStore, type ValoresTasa } from './tasas-store';

const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

const impuesto = (id: string, over: Partial<Impuesto> = {}): Impuesto => ({
  id,
  codigo: `I${id}`,
  nombre: `Impuesto ${id}`,
  descripcion: null,
  es_activo: true,
  ...over,
});
const tasa = (id: string, impuestoId: string, over: Partial<TasaImpuesto> = {}): TasaImpuesto => ({
  id,
  impuesto_id: impuestoId,
  tasa: '18.0000',
  vigente_desde: '2026-01-01',
  vigente_hasta: null,
  fundamento: null,
  ...over,
});
const valores = (over: Partial<ValoresTasa> = {}): ValoresTasa => ({
  tasa: '18',
  vigente_desde: '2026-01-01',
  vigente_hasta: '',
  fundamento: '',
  ...over,
});
const pagina = <T>(items: T[]) => of({ items, nextCursor: null });

describe('TasasStore', () => {
  const api = { impuestos: vi.fn(), tasas: vi.fn(), crearTasa: vi.fn(), editarTasa: vi.fn() };
  let permisos: string[];

  function setup(): TasasStore {
    TestBed.configureTestingModule({
      providers: [
        TasasStore,
        { provide: ImpuestosApi, useValue: api },
        { provide: AuthSession, useValue: { hasPermission: (p: string) => permisos.includes(p) } },
      ],
    });
    return TestBed.inject(TasasStore);
  }

  beforeEach(() => {
    permisos = ['IMPUESTO_ADMINISTRAR'];
    Object.values(api).forEach((f) => f.mockReset());
    api.impuestos.mockReturnValue(pagina([impuesto('1'), impuesto('2')]));
    api.tasas.mockImplementation((id: string) =>
      id === '1'
        ? pagina([
            tasa('a', '1', { vigente_desde: '2020-01-01', vigente_hasta: '2026-01-01' }),
            tasa('b', '1'),
          ])
        : pagina([]),
    );
  });

  it('administrar solo con IMPUESTO_ADMINISTRAR (IMPUESTO_VER o CARTA_ADMINISTRAR no bastan)', () => {
    expect(setup().puedeAdministrar()).toBe(true);
    for (const p of ['IMPUESTO_VER', 'CARTA_ADMINISTRAR']) {
      TestBed.resetTestingModule();
      permisos = [p];
      expect(setup().puedeAdministrar()).toBe(false);
    }
  });

  it('carga impuestos y las tasas de cada uno, de la más reciente a la más antigua', async () => {
    const store = setup();
    await store.cargar();
    expect(api.tasas).toHaveBeenCalledWith('1', undefined);
    expect(api.tasas).toHaveBeenCalledWith('2', undefined);
    expect(store.tasas()['1'].map((t) => t.id)).toEqual(['b', 'a']);
    expect(store.tasas()['2']).toEqual([]);
    expect(store.status()).toBe('ready');
  });

  it('si fallan los impuestos queda en error y reintenta', async () => {
    api.impuestos.mockReturnValueOnce(problem(500, 'x', 'boom'));
    const store = setup();
    await store.cargar();
    expect(store.status()).toBe('error');
    await store.cargar();
    expect(store.status()).toBe('ready');
  });

  it('carga parcial: el fallo de un impuesto no tumba a los demás y se reintenta solo ese', async () => {
    api.tasas.mockImplementation((id: string) =>
      id === '2' ? problem(500, 'x', 'boom') : pagina([tasa('b', '1')]),
    );
    const store = setup();
    await store.cargar();
    expect(store.status()).toBe('ready');
    expect(store.tasas()['1']).toHaveLength(1);
    expect(store.errores()['2']).toBeTruthy();
    api.tasas.mockImplementation(() => pagina([tasa('c', '2')]));
    await store.reintentar('2');
    expect(store.errores()['2']).toBeUndefined();
    expect(store.tasas()['2'].map((t) => t.id)).toEqual(['c']);
  });

  describe('alta', () => {
    it('envía la tasa tal como se escribió, con clave, y refresca solo ese impuesto', async () => {
      const store = setup();
      await store.cargar();
      api.crearTasa.mockReturnValue(of(tasa('n', '2')));
      const r = await store.crear(
        impuesto('2'),
        valores({ tasa: '10.5', vigente_hasta: '2027-01-01', fundamento: ' Ley X ' }),
        'k1',
      );
      expect(r).toBeUndefined();
      expect(api.crearTasa).toHaveBeenCalledWith(
        {
          impuesto_id: '2',
          tasa: '10.5',
          vigente_desde: '2026-01-01',
          vigente_hasta: '2027-01-01',
          fundamento: 'Ley X',
        },
        'k1',
      );
      expect(api.tasas).toHaveBeenCalledTimes(3);
      expect(store.notice()?.kind).toBe('ok');
    });

    it('vigente_hasta y fundamento vacíos viajan como null', async () => {
      const store = setup();
      api.crearTasa.mockReturnValue(of(tasa('n', '2')));
      await store.crear(impuesto('2'), valores(), 'k1');
      expect(api.crearTasa.mock.calls[0][0]).toMatchObject({
        vigente_hasta: null,
        fundamento: null,
      });
    });

    it.each([
      [valores({ tasa: '100.1' }), 'tasa'],
      [valores({ tasa: '' }), 'tasa'],
      [valores({ vigente_desde: '2026-02-30' }), 'vigente_desde'],
      [valores({ vigente_hasta: 'x' }), 'vigente_hasta'],
      [valores({ vigente_hasta: '2026-01-01' }), 'vigente_hasta'],
      [valores({ vigente_hasta: '2025-12-31' }), 'vigente_hasta'],
      [valores({ fundamento: 'x'.repeat(2001) }), 'fundamento'],
    ])('valida antes de llamar (%#)', async (v, campo) => {
      const store = setup();
      expect(await store.crear(impuesto('1'), v, 'k')).toMatchObject({ field: campo });
      expect(api.crearTasa).not.toHaveBeenCalled();
    });

    it.each([
      ['vigencia_solapada', 'vigente_desde'],
      ['vigencia_invalida', 'vigente_hasta'],
      ['tasa_invalida', 'tasa'],
      ['vigente_desde_invalido', 'vigente_desde'],
      ['vigente_hasta_invalido', 'vigente_hasta'],
      ['fundamento_invalido', 'fundamento'],
    ])('el rechazo %s se muestra en el campo %s', async (codigo, campo) => {
      const store = setup();
      api.crearTasa.mockReturnValue(problem(409, codigo, 'Mensaje del servidor.'));
      expect(await store.crear(impuesto('1'), valores(), 'k')).toEqual({
        field: campo,
        message: 'Mensaje del servidor.',
      });
    });
  });

  describe('edición', () => {
    const actual = tasa('b', '1', { fundamento: 'Ley' });

    it('sin cambios no llama (18 = 18.0000)', async () => {
      const store = setup();
      expect(await store.editar(actual, valores({ fundamento: 'Ley' }))).toBeUndefined();
      expect(api.editarTasa).not.toHaveBeenCalled();
    });

    it('PATCH solo con lo cambiado, sin clave; permite vaciar la vigencia y el fundamento', async () => {
      const store = setup();
      await store.cargar();
      api.editarTasa.mockReturnValue(of(actual));
      await store.editar(actual, valores({ tasa: '17.5', fundamento: 'Ley' }));
      expect(api.editarTasa).toHaveBeenCalledWith('b', { tasa: '17.5' });
      const cerrada = tasa('c', '1', { vigente_hasta: '2027-01-01', fundamento: 'Ley' });
      await store.editar(cerrada, valores({ fundamento: '' }));
      expect(api.editarTasa).toHaveBeenLastCalledWith('c', {
        vigente_hasta: null,
        fundamento: null,
      });
    });

    it('un rechazo del servidor apunta al campo', async () => {
      const store = setup();
      api.editarTasa.mockReturnValue(problem(409, 'vigencia_solapada', 'Se cruza con otra.'));
      expect(await store.editar(actual, valores({ vigente_desde: '2025-01-01' }))).toEqual({
        field: 'vigente_desde',
        message: 'Se cruza con otra.',
      });
    });
  });
});
