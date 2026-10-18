import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { AccesosApi } from './accesos-api';
import { Rol, UsuarioEquipo } from './accesos.models';
import { EquipoStore } from './equipo-store';

const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

const user = (id: string, over: Partial<UsuarioEquipo> = {}): UsuarioEquipo => ({
  id,
  username: `u${id}`,
  email: null,
  es_activo: true,
  ultimo_acceso_en: null,
  bloqueado_hasta: null,
  motivo_baja: null,
  roles: [{ id: '3', codigo: 'MESERO', nombre: 'Mesero', es_activo: true }],
  sucursales: [{ id: '1', codigo: 'PRINCIPAL', nombre: 'Local Principal' }],
  ...over,
});
const rol = (id: string, es_activo = true): Rol => ({
  id,
  codigo: `R${id}`,
  nombre: `Rol ${id}`,
  descripcion: null,
  es_activo,
});

describe('Estado de Equipo y accesos', () => {
  const api = {
    usuarios: vi.fn(),
    roles: vi.fn(),
    sucursalesAsignables: vi.fn(),
    crearUsuario: vi.fn(),
    asignar: vi.fn(),
    baja: vi.fn(),
    restaurar: vi.fn(),
  };
  let canAdminister: boolean;

  function setup(): EquipoStore {
    TestBed.configureTestingModule({
      providers: [
        EquipoStore,
        { provide: AccesosApi, useValue: api },
        {
          provide: AuthSession,
          useValue: { hasPermission: () => canAdminister, user: signal({ id: '2' }) },
        },
      ],
    });
    return TestBed.inject(EquipoStore);
  }

  beforeEach(() => {
    canAdminister = true;
    Object.values(api).forEach((f) => f.mockReset());
    api.usuarios.mockReturnValue(of({ items: [user('2'), user('3')], nextCursor: null }));
    api.roles.mockReturnValue(of({ items: [rol('1'), rol('2', false)], nextCursor: null }));
    api.sucursalesAsignables.mockReturnValue(
      of({ items: [{ id: '1', codigo: 'PRINCIPAL', nombre: 'Local Principal' }] }),
    );
  });

  it('carga el equipo, los roles y las sedes; solo ofrece roles activos', async () => {
    const store = setup();
    await store.load();
    expect(store.status()).toBe('ready');
    expect(store.users().map((u) => u.id)).toEqual(['2', '3']);
    expect(store.activeRoles().map((r) => r.id)).toEqual(['1']);
    expect(store.branches()).toHaveLength(1);
    expect(store.currentUserId()).toBe('2');
  });

  it('recorre todas las páginas usando el cursor', async () => {
    api.usuarios
      .mockReturnValueOnce(of({ items: [user('2')], nextCursor: 'c1' }))
      .mockReturnValueOnce(of({ items: [user('3')], nextCursor: null }));
    const store = setup();
    await store.load();
    expect(api.usuarios).toHaveBeenNthCalledWith(1, undefined);
    expect(api.usuarios).toHaveBeenNthCalledWith(2, 'c1');
    expect(store.users().map((u) => u.id)).toEqual(['2', '3']);
  });

  it('sin permiso de administración no consulta nada', async () => {
    canAdminister = false;
    const store = setup();
    await store.load();
    expect(store.canAdminister()).toBe(false);
    expect(store.status()).toBe('ready');
    expect(api.usuarios).not.toHaveBeenCalled();
    expect(api.roles).not.toHaveBeenCalled();
  });

  it('si la carga falla, informa sin exponer detalles técnicos', async () => {
    api.usuarios.mockReturnValue(problem(500, 'x', 'pq: relation does not exist'));
    const store = setup();
    await store.load();
    expect(store.status()).toBe('error');
    expect(store.loadError()).not.toContain('pq:');
    expect(store.loadError()).toContain('no está disponible');
  });

  describe('alta de una persona', () => {
    const values = {
      username: ' ana ',
      email: '',
      password: 'ClaveSegura2026',
      roleId: '1',
      branchId: '1',
    };

    it('crea la persona y le asigna rol y sede con la clave del formulario', async () => {
      api.crearUsuario.mockReturnValue(of({ id: '9', username: 'ana' }));
      api.asignar.mockReturnValue(of({ id: '9' }));
      const store = setup();
      expect(await store.invite(values, 'key-1')).toBeUndefined();
      expect(api.crearUsuario).toHaveBeenCalledWith(
        { username: 'ana', email: undefined, password: 'ClaveSegura2026' },
        'key-1',
      );
      expect(api.asignar).toHaveBeenCalledWith('9', ['1'], ['1']);
      expect(store.notice()).toEqual({ kind: 'ok', text: 'Se creó el acceso de ana.' });
    });

    it('si el alta se rechaza, devuelve el motivo del servidor y no asigna nada', async () => {
      api.crearUsuario.mockReturnValue(problem(409, 'registro_duplicado', 'ya existe ese usuario'));
      const store = setup();
      expect(await store.invite(values, 'key-1')).toBe('ya existe ese usuario');
      expect(api.asignar).not.toHaveBeenCalled();
      expect(store.notice()).toBeNull();
    });

    it('si falla la asignación, actualiza la lista y explica que se puede reintentar', async () => {
      api.crearUsuario.mockReturnValue(of({ id: '9', username: 'ana' }));
      api.asignar.mockReturnValue(problem(403, 'sede_no_autorizada', 'sede ajena'));
      const store = setup();
      const message = await store.invite(values, 'key-1');
      expect(message).toContain('La persona se creó, pero falta asignarle su acceso');
      expect(message).toContain('sede ajena');
      expect(message).toContain('reintentar');
      expect(api.usuarios).toHaveBeenCalled();
    });
  });

  it('cambia el rol conservando las sedes que se indican', async () => {
    api.asignar.mockReturnValue(of({ id: '3' }));
    const store = setup();
    expect(await store.changeAccess(user('3'), '1', ['1', '5'])).toBeUndefined();
    expect(api.asignar).toHaveBeenCalledWith('3', ['1'], ['1', '5']);
    expect(store.notice()?.kind).toBe('ok');
  });

  it('muestra por qué se rechaza un cambio (p. ej. el último administrador)', async () => {
    api.asignar.mockReturnValue(
      problem(422, 'ultimo_administrador', 'la operación dejaría al sistema sin administrador'),
    );
    const store = setup();
    expect(await store.changeAccess(user('2'), '3', ['1'])).toContain('sin administrador');
  });

  it('da de baja con el motivo recortado y la misma clave', async () => {
    api.baja.mockReturnValue(of({ id: '3' }));
    const store = setup();
    expect(await store.deactivate(user('3'), '  cambio de turno ', 'k')).toBeUndefined();
    expect(api.baja).toHaveBeenCalledWith('3', 'cambio de turno', 'k');
    expect(store.notice()?.text).toContain('sus sesiones se cerraron');
  });

  it('restaura avisando el resultado y bloquea acciones simultáneas', async () => {
    let finish!: () => void;
    api.restaurar.mockReturnValue(
      new (class {
        subscribe(o: { next: (v: unknown) => void; complete: () => void }) {
          finish = () => {
            o.next({ id: '3' });
            o.complete();
          };
          return { unsubscribe() {} };
        }
      })(),
    );
    const store = setup();
    const first = store.restore(user('3', { es_activo: false }), 'k1');
    expect(store.busyId()).toBe('3');
    await store.restore(user('4', { es_activo: false }), 'k2');
    expect(api.restaurar).toHaveBeenCalledTimes(1);
    finish();
    await first;
    expect(store.busyId()).toBeNull();
    expect(store.notice()?.text).toContain('Deberá iniciar sesión de nuevo');
  });

  it('si restaurar falla, deja un aviso de error y libera el bloqueo', async () => {
    api.restaurar.mockReturnValue(problem(403, 'sin_permiso', 'x'));
    const store = setup();
    await store.restore(user('3', { es_activo: false }), 'k');
    expect(store.notice()?.kind).toBe('error');
    expect(store.busyId()).toBeNull();
  });
});
