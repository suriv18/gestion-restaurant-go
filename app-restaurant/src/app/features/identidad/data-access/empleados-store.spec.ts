import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { AccesosApi } from './accesos-api';
import { UsuarioEquipo } from './accesos.models';
import { EmpleadosApi } from './empleados-api';
import { Empleado } from './empleados.models';
import { EmpleadosStore, ValoresEmpleado } from './empleados-store';

const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

const empleado = (id: string, over: Partial<Empleado> = {}): Empleado => ({
  id,
  usuario_id: null,
  username: null,
  tipo_documento: 'DNI',
  numero_documento: '12345678',
  nombres: 'María',
  apellidos: 'Quispe',
  telefono: null,
  cargo: null,
  fecha_ingreso: '2026-03-15',
  fecha_cese: null,
  es_activo: true,
  motivo_eliminacion: null,
  ...over,
});

const usuario = (id: string, over: Partial<UsuarioEquipo> = {}): UsuarioEquipo => ({
  id,
  username: `u${id}`,
  email: null,
  es_activo: true,
  ultimo_acceso_en: null,
  bloqueado_hasta: null,
  motivo_baja: null,
  empleado: null,
  roles: [],
  sucursales: [],
  ...over,
});

const valores = (over: Partial<ValoresEmpleado> = {}): ValoresEmpleado => ({
  nombres: 'María',
  apellidos: 'Quispe',
  tipo_documento: 'DNI',
  numero_documento: '12345678',
  cargo: '',
  telefono: '',
  fecha_ingreso: '2026-03-15',
  fecha_cese: '',
  usuario_id: '',
  ...over,
});

describe('Estado de Empleados', () => {
  const api = {
    empleados: vi.fn(),
    crear: vi.fn(),
    editar: vi.fn(),
    baja: vi.fn(),
    restaurar: vi.fn(),
  };
  const accesos = { usuarios: vi.fn() };
  let permisos: string[];

  function setup(): EmpleadosStore {
    TestBed.configureTestingModule({
      providers: [
        EmpleadosStore,
        { provide: EmpleadosApi, useValue: api },
        { provide: AccesosApi, useValue: accesos },
        {
          provide: AuthSession,
          useValue: {
            hasPermission: (p: string) => permisos.includes(p),
            user: signal({ id: '2' }),
          },
        },
      ],
    });
    return TestBed.inject(EmpleadosStore);
  }

  beforeEach(() => {
    permisos = ['EMPLEADO_VER', 'EMPLEADO_ADMINISTRAR', 'USUARIO_ADMINISTRAR'];
    Object.values(api).forEach((f) => f.mockReset());
    accesos.usuarios.mockReset();
    api.empleados.mockReturnValue(of({ items: [empleado('1')], nextCursor: null }));
    accesos.usuarios.mockReturnValue(
      of({
        items: [
          usuario('2'),
          usuario('3', { es_activo: false }),
          usuario('4', { empleado: { id: '1', nombres: 'M', apellidos: 'Q', cargo: null } }),
        ],
        nextCursor: null,
      }),
    );
  });

  it('carga los empleados y el equipo para poder vincular', async () => {
    const store = setup();
    await store.load();
    expect(store.status()).toBe('ready');
    expect(store.empleados()).toHaveLength(1);
    expect(store.puedeVincular()).toBe(true);
  });

  it('solo ofrece usuarios activos sin empleado, más el actual del empleado', async () => {
    const store = setup();
    await store.load();
    expect(store.usuariosVinculables().map((u) => u.id)).toEqual(['2']);
    expect(store.usuariosVinculables(empleado('1', { usuario_id: '4' })).map((u) => u.id)).toEqual([
      '2',
      '4',
    ]);
  });

  it('sin permiso para el equipo no lo consulta ni deja vincular', async () => {
    permisos = ['EMPLEADO_VER', 'EMPLEADO_ADMINISTRAR'];
    const store = setup();
    await store.load();
    expect(accesos.usuarios).not.toHaveBeenCalled();
    expect(store.puedeVincular()).toBe(false);
  });

  it('sin permiso de ver no llama al servidor', async () => {
    permisos = [];
    const store = setup();
    await store.load();
    expect(store.puedeVer()).toBe(false);
    expect(api.empleados).not.toHaveBeenCalled();
  });

  it('crea con textos recortados, vacíos como null y la clave del formulario', async () => {
    api.crear.mockReturnValue(of(empleado('9')));
    const store = setup();
    await store.load();
    const resultado = await store.guardar(
      undefined,
      valores({ nombres: '  María ', cargo: 'Mesera', fecha_ingreso: '' }),
      'k1',
    );
    expect(resultado).toBeUndefined();
    expect(api.crear).toHaveBeenCalledWith(
      {
        tipo_documento: 'DNI',
        numero_documento: '12345678',
        nombres: 'María',
        apellidos: 'Quispe',
        telefono: null,
        cargo: 'Mesera',
        fecha_ingreso: null,
        usuario_id: null,
      },
      'k1',
    );
    expect(store.notice()?.text).toMatch(/María Quispe/);
  });

  it.each([
    ['documento_duplicado', 'numero_documento'],
    ['nombres_invalido', 'nombres'],
    ['fechas_incoherentes', 'fecha_cese'],
    ['usuario_ya_vinculado', 'usuario_id'],
  ])('el rechazo %s se muestra bajo el campo %s', async (code, field) => {
    api.crear.mockReturnValue(problem(409, code, 'detalle del servidor'));
    const store = setup();
    await store.load();
    expect(await store.guardar(undefined, valores(), 'k1')).toEqual({
      field,
      message: 'detalle del servidor',
    });
  });

  it('al editar envía solo lo que cambió', async () => {
    api.editar.mockReturnValue(of(empleado('1')));
    const store = setup();
    await store.load();
    await store.guardar(empleado('1'), valores({ cargo: 'Jefa de sala' }), 'k1');
    expect(api.editar).toHaveBeenCalledWith('1', { cargo: 'Jefa de sala' });
  });

  it('vaciar la fecha de cese o el acceso envía null; la fecha de ingreso nunca va vacía', async () => {
    api.editar.mockReturnValue(of(empleado('1')));
    const store = setup();
    await store.load();
    const actual = empleado('1', { usuario_id: '4', username: 'u4', fecha_cese: '2026-09-30' });
    await store.guardar(
      actual,
      valores({ fecha_ingreso: '', fecha_cese: '', usuario_id: '' }),
      'k1',
    );
    expect(api.editar).toHaveBeenCalledWith('1', { fecha_cese: null, usuario_id: null });
  });

  it('sin selector de acceso (usuario_id ausente) no toca el vínculo', async () => {
    api.editar.mockReturnValue(of(empleado('1')));
    const store = setup();
    await store.load();
    const actual = empleado('1', { usuario_id: '4', username: 'u4' });
    const v = valores({ cargo: 'Cajero' });
    delete v.usuario_id;
    await store.guardar(actual, v, 'k1');
    expect(api.editar).toHaveBeenCalledWith('1', { cargo: 'Cajero' });
  });

  it('sin cambios no llama al servidor', async () => {
    const store = setup();
    await store.load();
    expect(await store.guardar(empleado('1'), valores(), 'k1')).toBeUndefined();
    expect(api.editar).not.toHaveBeenCalled();
  });

  it('la baja con un usuario activo se rechaza con el aviso del servidor', async () => {
    api.baja.mockReturnValue(
      problem(409, 'empleado_con_usuario_activo', 'dé de baja primero su usuario'),
    );
    const store = setup();
    await store.load();
    expect(await store.baja(empleado('1'), 'Renunció', 'kb')).toBe('dé de baja primero su usuario');
  });

  it('la baja recorta el motivo y refresca la lista', async () => {
    api.baja.mockReturnValue(of(empleado('1', { es_activo: false })));
    const store = setup();
    await store.load();
    api.empleados.mockReturnValue(
      of({ items: [empleado('1', { es_activo: false })], nextCursor: null }),
    );
    expect(await store.baja(empleado('1'), '  Renunció ', 'kb')).toBeUndefined();
    expect(api.baja).toHaveBeenCalledWith('1', 'Renunció', 'kb');
    expect(store.empleados()[0].es_activo).toBe(false);
  });

  it('restaurar informa el resultado y libera el botón; un fallo se avisa', async () => {
    api.restaurar.mockReturnValueOnce(of(empleado('1')));
    const store = setup();
    await store.load();
    await store.restaurar(empleado('1', { es_activo: false }), 'kr');
    expect(store.notice()?.kind).toBe('ok');
    expect(store.busyId()).toBeNull();

    api.restaurar.mockReturnValueOnce(problem(422, 'usuario_invalido', 'su usuario está de baja'));
    await store.restaurar(empleado('1', { es_activo: false }), 'kr2');
    expect(store.notice()).toEqual({ kind: 'error', text: 'su usuario está de baja' });
    expect(store.busyId()).toBeNull();
  });
});
