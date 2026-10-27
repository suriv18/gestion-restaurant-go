import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { Observable, of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import type { Estacion } from '../../catalogos/data-access/carta.models';
import { EstacionesApi } from '../../catalogos/data-access/estaciones-api';
import { EstacionesStore } from './estaciones-store';

const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

const estacion = (id: string, over: Partial<Estacion> = {}): Estacion => ({
  id,
  sucursal_id: '7',
  codigo: `E${id}`,
  nombre: `Estacion ${id}`,
  es_activo: true,
  ...over,
});

describe('EstacionesStore', () => {
  const api = {
    listar: vi.fn(),
    crear: vi.fn(),
    editar: vi.fn(),
    baja: vi.fn(),
    restaurar: vi.fn(),
  };
  let permisos: string[];

  function setup(): EstacionesStore {
    TestBed.configureTestingModule({
      providers: [
        EstacionesStore,
        { provide: EstacionesApi, useValue: api },
        { provide: AuthSession, useValue: { hasPermission: (p: string) => permisos.includes(p) } },
      ],
    });
    return TestBed.inject(EstacionesStore);
  }

  beforeEach(() => {
    permisos = ['ESTACION_ADMINISTRAR'];
    Object.values(api).forEach((f) => f.mockReset());
    api.listar.mockReturnValue(
      of({ items: [estacion('2'), estacion('1', { es_activo: false })], nextCursor: null }),
    );
  });

  it('administrar depende solo de ESTACION_ADMINISTRAR (ver no basta)', () => {
    expect(setup().puedeAdministrar()).toBe(true);
    TestBed.resetTestingModule();
    permisos = ['ESTACION_VER'];
    expect(setup().puedeAdministrar()).toBe(false);
  });

  it('carga las estaciones de la sede (activas y bajas) ordenadas por código', async () => {
    const store = setup();
    await store.cargar('7');
    expect(api.listar).toHaveBeenCalledWith({ sucursal_id: '7' }, undefined);
    expect(store.ordenadas().map((e) => e.id)).toEqual(['1', '2']);
    expect(store.status()).toBe('ready');
  });

  it('sin sede no llama al servidor y queda vacío', async () => {
    const store = setup();
    await store.cargar(null);
    expect(api.listar).not.toHaveBeenCalled();
    expect(store.estaciones()).toEqual([]);
    expect(store.status()).toBe('ready');
  });

  it('un error de carga queda en error con el detalle y se puede reintentar', async () => {
    api.listar.mockReturnValueOnce(problem(500, 'x', 'boom'));
    const store = setup();
    await store.cargar('7');
    expect(store.status()).toBe('error');
    expect(store.error()).toBeTruthy();
    await store.cargar('7');
    expect(store.status()).toBe('ready');
  });

  it('descarta la respuesta de una sede que ya no es la actual', async () => {
    let soltar!: () => void;
    const lenta = new Promise<void>((r) => (soltar = r));
    api.listar.mockReturnValueOnce(
      new Observable((s) => {
        void lenta.then(() => {
          s.next({ items: [estacion('9')], nextCursor: null });
          s.complete();
        });
      }),
    );
    const store = setup();
    const primera = store.cargar('7');
    api.listar.mockReturnValueOnce(
      of({ items: [estacion('5', { sucursal_id: '8' })], nextCursor: null }),
    );
    await store.cargar('8');
    soltar();
    await primera;
    expect(store.estaciones().map((e) => e.id)).toEqual(['5']);
  });

  describe('alta', () => {
    it('envía el código en mayúsculas con la sede y la clave, y refresca', async () => {
      const store = setup();
      await store.cargar('7');
      api.crear.mockReturnValue(of(estacion('3')));
      const r = await store.crear({ codigo: ' bar-1 ', nombre: ' Barra ' }, 'k1');
      expect(r).toBeUndefined();
      expect(api.crear).toHaveBeenCalledWith(
        { sucursal_id: '7', codigo: 'BAR-1', nombre: 'Barra' },
        'k1',
      );
      expect(api.listar).toHaveBeenCalledTimes(2);
      expect(store.notice()?.kind).toBe('ok');
    });

    it('valida código y nombre antes de llamar', async () => {
      const store = setup();
      await store.cargar('7');
      expect(await store.crear({ codigo: '-X', nombre: 'a' }, 'k')).toMatchObject({
        field: 'codigo',
      });
      expect(await store.crear({ codigo: 'A'.repeat(21), nombre: 'a' }, 'k')).toMatchObject({
        field: 'codigo',
      });
      expect(await store.crear({ codigo: 'OK', nombre: ' ' }, 'k')).toMatchObject({
        field: 'nombre',
      });
      expect(await store.crear({ codigo: 'OK', nombre: 'x'.repeat(81) }, 'k')).toMatchObject({
        field: 'nombre',
      });
      expect(api.crear).not.toHaveBeenCalled();
    });

    it('estacion_duplicada vuelve como error del campo código', async () => {
      const store = setup();
      await store.cargar('7');
      api.crear.mockReturnValue(problem(409, 'estacion_duplicada', 'Ya existe.'));
      expect(await store.crear({ codigo: 'COC', nombre: 'Cocina' }, 'k')).toEqual({
        field: 'codigo',
        message: 'Ya existe.',
      });
    });

    it('sin sede no crea', async () => {
      const store = setup();
      await store.cargar(null);
      expect(typeof (await store.crear({ codigo: 'A', nombre: 'b' }, 'k'))).toBe('string');
      expect(api.crear).not.toHaveBeenCalled();
    });
  });

  describe('edición', () => {
    it('PATCH solo con el nombre; sin cambios no llama', async () => {
      const store = setup();
      await store.cargar('7');
      const e = estacion('2');
      expect(await store.editar(e, { nombre: ' Estacion 2 ' })).toBeUndefined();
      expect(api.editar).not.toHaveBeenCalled();
      api.editar.mockReturnValue(of(e));
      await store.editar(e, { nombre: 'Parrilla' });
      expect(api.editar).toHaveBeenCalledWith('2', { nombre: 'Parrilla' });
    });

    it('nombre_invalido del servidor va al campo nombre', async () => {
      const store = setup();
      api.editar.mockReturnValue(problem(422, 'nombre_invalido', 'Nombre no válido.'));
      expect(await store.editar(estacion('2'), { nombre: 'Otro' })).toEqual({
        field: 'nombre',
        message: 'Nombre no válido.',
      });
    });
  });

  describe('baja y restauración', () => {
    it('baja con clave y refresco', async () => {
      const store = setup();
      await store.cargar('7');
      api.baja.mockReturnValue(of(estacion('2', { es_activo: false })));
      expect(await store.baja(estacion('2'), 'kb')).toBeUndefined();
      expect(api.baja).toHaveBeenCalledWith('2', 'kb');
      expect(api.listar).toHaveBeenCalledTimes(2);
    });

    it('estacion_en_uso se devuelve como texto con el detail', async () => {
      const store = setup();
      api.baja.mockReturnValue(problem(409, 'estacion_en_uso', 'La usan 3 productos activos.'));
      expect(await store.baja(estacion('2'), 'kb')).toBe('La usan 3 productos activos.');
    });

    it('restaurar usa la clave, se libera al terminar y avisa de errores', async () => {
      const store = setup();
      await store.cargar('7');
      api.restaurar.mockReturnValue(of(estacion('1')));
      await store.restaurar(estacion('1', { es_activo: false }), 'kr');
      expect(api.restaurar).toHaveBeenCalledWith('1', 'kr');
      expect(store.busyId()).toBeNull();
      expect(store.notice()?.text).toContain('volvió a estar activa');
      api.restaurar.mockReturnValue(problem(409, 'ya_activa', 'Ya está activa.'));
      await store.restaurar(estacion('1'), 'kr2');
      expect(store.notice()).toEqual({ kind: 'error', text: 'Ya está activa.' });
    });

    it('si el refresco falla tras la escritura avisa con advertencia', async () => {
      const store = setup();
      await store.cargar('7');
      api.baja.mockReturnValue(of(estacion('2')));
      api.listar.mockReturnValue(problem(500, 'x', 'boom'));
      await store.baja(estacion('2'), 'kb');
      expect(store.notice()?.kind).toBe('error');
      expect(store.notice()?.text).toContain('no pudimos actualizar');
    });
  });
});
