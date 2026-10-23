import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import type { Mesa, Negocio, Sucursal, Zona } from '../configuracion.models';
import { ConfiguracionApi } from './configuracion-api';
import { ConfiguracionStore } from './configuracion-store';

const TODOS = [
  'NEGOCIO_VER',
  'NEGOCIO_ADMINISTRAR',
  'SUCURSAL_VER',
  'SUCURSAL_ADMINISTRAR',
  'MESA_VER',
  'MESA_ADMINISTRAR',
];

const api = {
  negocio: vi.fn(),
  editarNegocio: vi.fn(),
  sucursales: vi.fn(),
  crearSucursal: vi.fn(),
  editarSucursal: vi.fn(),
  bajaSucursal: vi.fn(),
  restaurarSucursal: vi.fn(),
  zonas: vi.fn(),
  crearZona: vi.fn(),
  editarZona: vi.fn(),
  bajaZona: vi.fn(),
  restaurarZona: vi.fn(),
  mesas: vi.fn(),
  crearMesa: vi.fn(),
  editarMesa: vi.fn(),
  bajaMesa: vi.fn(),
  restaurarMesa: vi.fn(),
};

let permisos: string[];

const audit = {
  creado_en: '2026-10-21T10:00:00Z',
  creado_por: '1',
  actualizado_en: '2026-10-21T10:00:00Z',
  actualizado_por: '1',
};
const negocio = (over: Partial<Negocio> = {}): Negocio => ({
  id: '1',
  singleton: true,
  ruc: '20123456789',
  razon_social: 'Mi Restaurante SAC',
  nombre_comercial: 'Mi Restaurante',
  direccion_fiscal: 'Av. Principal 123',
  ubigeo: null,
  telefono: null,
  email: null,
  moneda: 'PEN',
  zona_horaria: 'America/Lima',
  es_activo: true,
  ...audit,
  ...over,
});
const sede = (id: string, over: Partial<Sucursal> = {}): Sucursal => ({
  id,
  negocio_id: '1',
  codigo: 'S' + id,
  nombre: 'Sede ' + id,
  direccion: 'Calle ' + id,
  ubigeo: null,
  codigo_establecimiento_sunat: null,
  telefono: null,
  es_activo: true,
  motivo_eliminacion: null,
  eliminado_en: null,
  eliminado_por: null,
  ...audit,
  ...over,
});
const zona = (id: string, sucursal: string, over: Partial<Zona> = {}): Zona => ({
  id,
  sucursal_id: sucursal,
  nombre: 'Zona ' + id,
  orden_visualizacion: 0,
  es_activo: true,
  ...audit,
  ...over,
});
const mesa = (id: string, zonaId: string, sucursal: string, over: Partial<Mesa> = {}): Mesa => ({
  id,
  sucursal_id: sucursal,
  zona_id: zonaId,
  numero_mesa: id,
  capacidad: 4,
  habilitada: true,
  posicion_x: null,
  posicion_y: null,
  es_activo: true,
  motivo_eliminacion: null,
  eliminado_en: null,
  eliminado_por: null,
  ...audit,
  ...over,
});
const lista = <T>(items: T[]) => of({ items, nextCursor: null });
const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

function setup(): ConfiguracionStore {
  TestBed.configureTestingModule({
    providers: [
      ConfiguracionStore,
      { provide: ConfiguracionApi, useValue: api },
      {
        provide: AuthSession,
        useValue: { hasPermission: (p: string) => permisos.includes(p), user: signal({ id: '2' }) },
      },
    ],
  });
  return TestBed.inject(ConfiguracionStore);
}

describe('Estado de Configuración', () => {
  beforeEach(() => {
    permisos = TODOS;
    Object.values(api).forEach((f) => f.mockReset());
    api.negocio.mockReturnValue(of(negocio()));
    api.sucursales.mockReturnValue(lista([sede('1'), sede('2', { es_activo: false })]));
    api.zonas.mockReturnValue(lista([zona('10', '1')]));
    api.mesas.mockReturnValue(lista([mesa('100', '10', '1')]));
  });

  it('carga negocio, sedes, zonas y mesas con todos los permisos', async () => {
    const store = setup();
    await store.load();
    expect(store.status()).toBe('ready');
    expect(store.negocio()?.nombre_comercial).toBe('Mi Restaurante');
    expect(store.sedes()).toHaveLength(2);
    expect(store.zonas()).toHaveLength(1);
    expect(store.mesas()).toHaveLength(1);
    expect(store.puedeEditarNegocio()).toBe(true);
  });

  it('un mesero (solo MESA_VER) solo pide zonas y mesas y muestra «Sede {id}»', async () => {
    permisos = ['MESA_VER'];
    const store = setup();
    await store.load();
    expect(api.negocio).not.toHaveBeenCalled();
    expect(api.sucursales).not.toHaveBeenCalled();
    expect(store.zonas()).toHaveLength(1);
    expect(store.mesas()).toHaveLength(1);
    expect(store.sedesSalon()).toEqual([{ id: '1', nombre: 'Sede 1' }]);
    expect(store.puedeEditarSalon()).toBe(false);
  });

  it('sin ningún permiso no llama a la API', async () => {
    permisos = [];
    const store = setup();
    await store.load();
    expect(store.sinAcceso()).toBe(true);
    expect(Object.values(api).every((f) => f.mock.calls.length === 0)).toBe(true);
    expect(store.status()).toBe('ready');
  });

  it('un fallo en una sección no tumba las demás', async () => {
    api.negocio.mockReturnValue(problem(403, 'sin_permiso', 'No tienes permiso.'));
    const store = setup();
    await store.load();
    expect(store.errors().negocio).toBe('No tienes permiso para realizar esta acción.');
    expect(store.errors().salon).toBeUndefined();
    expect(store.zonas()).toHaveLength(1);
  });

  it('un fallo al cargar las sedes lo marca en su sección y las demás cargan', async () => {
    api.sucursales.mockReturnValue(problem(500, 'error_interno', 'Falló.'));
    const store = setup();
    await store.load();
    expect(store.errors().sedes).toBeTruthy();
    expect(store.sedes()).toEqual([]);
    expect(store.negocio()).not.toBeNull();
    expect(store.zonas()).toHaveLength(1);
    expect(store.mesas()).toHaveLength(1);
    expect(store.status()).toBe('ready');
  });

  it('si las sedes no cargan, el salón sigue operable con las sedes que salen de los datos', async () => {
    api.sucursales.mockReturnValue(problem(403, 'prohibido', 'No tienes permiso.'));
    const store = setup();
    await store.load();
    expect(store.errors().sedes).toBeTruthy();
    expect(store.sedesSalon()).toEqual([{ id: '1', nombre: 'Sede 1' }]);
    expect(store.sedeActual()).toBe('1');
    expect(store.zonasSede()).toHaveLength(1);
  });

  it('las sedes del salón son solo las activas con nombre real cuando se pueden ver', async () => {
    const store = setup();
    await store.load();
    expect(store.sedesSalon()).toEqual([{ id: '1', nombre: 'Sede 1' }]);
    expect(store.sedeActual()).toBe('1');
  });

  it('editarNegocio guarda los cambios y avisa', async () => {
    api.editarNegocio.mockReturnValue(of(negocio({ nombre_comercial: 'Nuevo' })));
    const store = setup();
    await store.load();
    const result = await store.editarNegocio({ nombre_comercial: 'Nuevo' });
    expect(result).toBeUndefined();
    expect(store.negocio()?.nombre_comercial).toBe('Nuevo');
    expect(store.notice()?.kind).toBe('ok');
  });

  it('editarNegocio sin cambios no llama a la API', async () => {
    const store = setup();
    await store.load();
    expect(await store.editarNegocio({})).toBeUndefined();
    expect(api.editarNegocio).not.toHaveBeenCalled();
  });

  it('un rechazo de campo vuelve como {field, message}', async () => {
    api.editarNegocio.mockReturnValue(problem(422, 'moneda_invalida', 'La moneda no es válida.'));
    const store = setup();
    await store.load();
    expect(await store.editarNegocio({ moneda: 'xx' })).toEqual({
      field: 'moneda',
      message: 'La moneda no es válida.',
    });
  });

  it('crearSede refresca la lista y recuerda asignarla en Equipo y accesos', async () => {
    api.crearSucursal.mockReturnValue(of(sede('3')));
    api.sucursales
      .mockReturnValueOnce(lista([sede('1')]))
      .mockReturnValue(lista([sede('1'), sede('3')]));
    const store = setup();
    await store.load();
    const result = await store.crearSede(
      { codigo: 'S3', nombre: 'Sede 3', direccion: 'Calle 3' },
      'k1',
    );
    expect(result).toBeUndefined();
    expect(api.crearSucursal).toHaveBeenCalledWith(
      { codigo: 'S3', nombre: 'Sede 3', direccion: 'Calle 3' },
      'k1',
    );
    expect(store.sedes()).toHaveLength(2);
    expect(store.notice()?.text).toContain('Asígnala a tu equipo en Equipo y accesos');
  });

  it('código duplicado de sede vuelve como error del campo codigo', async () => {
    api.crearSucursal.mockReturnValue(problem(409, 'codigo_duplicado', 'Ya existe ese código.'));
    const store = setup();
    await store.load();
    expect(await store.crearSede({ codigo: 'S1', nombre: 'x', direccion: 'y' }, 'k1')).toEqual({
      field: 'codigo',
      message: 'Ya existe ese código.',
    });
  });

  it('editarSede envía solo lo cambiado y no llama si nada cambió', async () => {
    api.editarSucursal.mockReturnValue(of(sede('1', { nombre: 'Centro' })));
    const store = setup();
    await store.load();
    const original = store.sedes()[0];
    expect(await store.editarSede(original, {})).toBeUndefined();
    expect(api.editarSucursal).not.toHaveBeenCalled();
    await store.editarSede(original, { nombre: 'Centro' });
    expect(api.editarSucursal).toHaveBeenCalledWith('1', { nombre: 'Centro' });
    expect(store.notice()).toEqual({
      kind: 'ok',
      text: 'Guardamos los cambios de la sede Sede 1.',
    });
    expect(api.sucursales).toHaveBeenCalledTimes(2);
  });

  it('si la lista no se pudo refrescar tras guardar, el aviso no es de éxito simple', async () => {
    api.crearSucursal.mockReturnValue(of(sede('3')));
    api.sucursales
      .mockReturnValueOnce(lista([sede('1')]))
      .mockReturnValue(problem(500, 'error_interno', 'Falló.'));
    const store = setup();
    await store.load();
    const result = await store.crearSede({ codigo: 'S3', nombre: 'Sede 3', direccion: 'x' }, 'k1');
    expect(result).toBeUndefined();
    expect(store.notice()?.kind).toBe('error');
    expect(store.notice()?.text).toContain('Se creó la sede Sede 3');
    expect(store.notice()?.text).toContain('no pudimos actualizar la lista de sedes');
    expect(store.sedes()).toHaveLength(1);
  });

  it('restaurarSede con rechazo avisa del error y libera el bloqueo', async () => {
    api.restaurarSucursal.mockReturnValue(problem(409, 'conflicto', 'No se pudo restaurar.'));
    const store = setup();
    await store.load();
    await store.restaurarSede(store.sedes()[1], 'k1');
    expect(store.notice()).toEqual({ kind: 'error', text: 'No se pudo restaurar.' });
    expect(store.busyId()).toBeNull();
  });

  it('bajaSede con la última sede activa devuelve el mensaje del servidor', async () => {
    api.bajaSucursal.mockReturnValue(
      problem(422, 'ultima_sede_activa', 'No se puede dar de baja la última sucursal activa.'),
    );
    const store = setup();
    await store.load();
    expect(await store.bajaSede(store.sedes()[0], 'Cierre', 'k1')).toBe(
      'No se puede dar de baja la última sucursal activa.',
    );
  });

  it('crearZona refresca zonas y mesas y avisa', async () => {
    api.crearZona.mockReturnValue(of(zona('11', '1', { nombre: 'Terraza' })));
    const store = setup();
    await store.load();
    api.zonas.mockReturnValue(lista([zona('10', '1'), zona('11', '1', { nombre: 'Terraza' })]));
    const result = await store.crearZona({ sucursal_id: '1', nombre: 'Terraza' }, 'k1');
    expect(result).toBeUndefined();
    expect(store.zonas()).toHaveLength(2);
    expect(store.notice()?.text).toContain('Terraza');
  });

  it('nombre duplicado de zona vuelve como error del campo nombre', async () => {
    api.crearZona.mockReturnValue(
      problem(409, 'nombre_duplicado', 'Ya existe una zona con ese nombre.'),
    );
    const store = setup();
    await store.load();
    expect(await store.crearZona({ sucursal_id: '1', nombre: 'Zona 10' }, 'k1')).toEqual({
      field: 'nombre',
      message: 'Ya existe una zona con ese nombre.',
    });
  });

  it('bajaZona con mesas activas devuelve el mensaje del servidor como aviso general', async () => {
    api.bajaZona.mockReturnValue(problem(409, 'zona_con_mesas', 'La zona tiene mesas activas.'));
    const store = setup();
    await store.load();
    expect(await store.bajaZona(store.zonas()[0], 'k1')).toBe('La zona tiene mesas activas.');
  });

  it('editarZona sin cambios no llama a la API', async () => {
    const store = setup();
    await store.load();
    const z = store.zonas()[0];
    expect(await store.editarZona(z, { nombre: z.nombre, orden_visualizacion: 0 })).toBeUndefined();
    expect(api.editarZona).not.toHaveBeenCalled();
  });

  it('crearMesa con número repetido vuelve como error del campo numero_mesa', async () => {
    api.crearMesa.mockReturnValue(problem(409, 'numero_mesa_duplicado', 'Ya existe esa mesa.'));
    const store = setup();
    await store.load();
    expect(
      await store.crearMesa(
        { sucursal_id: '1', zona_id: '10', numero_mesa: '100', capacidad: 4 },
        'k1',
      ),
    ).toEqual({ field: 'numero_mesa', message: 'Ya existe esa mesa.' });
  });

  it('editarMesa envía solo lo cambiado, incluido vaciar la posición con null', async () => {
    api.editarMesa.mockReturnValue(of(mesa('100', '10', '1', { capacidad: 6 })));
    const store = setup();
    await store.load();
    const m = store.mesas()[0];
    await store.editarMesa(m, {
      zona_id: '10',
      numero_mesa: '100',
      capacidad: 6,
      posicion_x: null,
      posicion_y: null,
    });
    expect(api.editarMesa).toHaveBeenCalledWith('100', { capacidad: 6 });
  });

  it('habilitarMesa alterna el estado en la fila y no deja avisos de error', async () => {
    api.editarMesa.mockReturnValue(of(mesa('100', '10', '1', { habilitada: false })));
    const store = setup();
    await store.load();
    await store.habilitarMesa(store.mesas()[0]);
    expect(api.editarMesa).toHaveBeenCalledWith('100', { habilitada: false });
    expect(store.mesas()[0].habilitada).toBe(false);
    expect(store.busyId()).toBeNull();
  });

  it('habilitarMesa con error muestra el aviso y libera el bloqueo', async () => {
    api.editarMesa.mockReturnValue(problem(409, 'mesa_inactiva', 'La mesa está dada de baja.'));
    const store = setup();
    await store.load();
    await store.habilitarMesa(store.mesas()[0]);
    expect(store.notice()).toEqual({ kind: 'error', text: 'La mesa está dada de baja.' });
    expect(store.busyId()).toBeNull();
  });

  it('bajaMesa sin motivo del servidor vuelve como error del campo motivo', async () => {
    api.bajaMesa.mockReturnValue(problem(422, 'motivo_requerido', 'Indica el motivo.'));
    const store = setup();
    await store.load();
    expect(await store.bajaMesa(store.mesas()[0], 'x', 'k1')).toEqual({
      field: 'motivo',
      message: 'Indica el motivo.',
    });
  });

  it('restaurarMesa con la zona dada de baja avisa con el mensaje del servidor', async () => {
    api.restaurarMesa.mockReturnValue(
      problem(
        422,
        'regla_de_datos',
        'No se puede restaurar la mesa porque su zona está dada de baja.',
      ),
    );
    const store = setup();
    await store.load();
    await store.restaurarMesa(store.mesas()[0], 'k1');
    expect(store.notice()?.kind).toBe('error');
    expect(store.notice()?.text).toContain('su zona está dada de baja');
  });

  it('si zonas y mesas no se pudieron refrescar tras guardar, el aviso no es de éxito simple', async () => {
    api.crearZona.mockReturnValue(of(zona('11', '1', { nombre: 'Terraza' })));
    const store = setup();
    await store.load();
    api.zonas.mockReturnValue(problem(500, 'error_interno', 'Falló.'));
    const result = await store.crearZona({ sucursal_id: '1', nombre: 'Terraza' }, 'k1');
    expect(result).toBeUndefined();
    expect(store.notice()?.kind).toBe('error');
    expect(store.notice()?.text).toContain('Se creó la zona Terraza');
    expect(store.notice()?.text).toContain('no pudimos actualizar la lista de zonas y mesas');
    expect(store.zonas()).toHaveLength(1);
  });

  it('restaurarSede avisa y no permite dos acciones a la vez', async () => {
    let liberar: (v: Sucursal) => void = () => undefined;
    api.restaurarSucursal.mockReturnValue({
      subscribe: (o: { next: (v: Sucursal) => void; complete: () => void }) => {
        liberar = (v) => {
          o.next(v);
          o.complete();
        };
        return { unsubscribe() {} };
      },
    });
    const store = setup();
    await store.load();
    const inactiva = store.sedes()[1];
    const primera = store.restaurarSede(inactiva, 'k1');
    expect(store.busyId()).toBe('2');
    await store.restaurarSede(inactiva, 'k2');
    expect(api.restaurarSucursal).toHaveBeenCalledTimes(1);
    api.sucursales.mockReturnValue(lista([sede('1'), sede('2')]));
    liberar(sede('2'));
    await primera;
    expect(store.busyId()).toBeNull();
    expect(store.notice()).toEqual({ kind: 'ok', text: 'La sede Sede 2 volvió a estar activa.' });
    expect(store.sedes()[1].es_activo).toBe(true);
  });
});
