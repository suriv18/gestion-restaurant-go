import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { DirectorioApi } from './directorio-api';
import { Cliente, Proveedor } from './directorio.models';
import { DirectorioStore, ValoresCliente, ValoresProveedor } from './directorio-store';

const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

const cliente = (id: string, over: Partial<Cliente> = {}): Cliente => ({
  id,
  tipo_documento: 'DNI',
  numero_documento: '12345678',
  nombre_razon_social: `Cliente ${id}`,
  telefono: null,
  email: null,
  direccion: null,
  es_activo: true,
  motivo_eliminacion: null,
  ...over,
});

const proveedor = (id: string, over: Partial<Proveedor> = {}): Proveedor => ({
  id,
  tipo_documento: 'RUC',
  numero_documento: '20123456789',
  razon_social: `Proveedor ${id}`,
  contacto_nombre: null,
  telefono: null,
  email: null,
  direccion: null,
  es_activo: true,
  motivo_eliminacion: null,
  ...over,
});

const valoresCliente = (over: Partial<ValoresCliente> = {}): ValoresCliente => ({
  nombre: 'Ana Ríos',
  tipo_documento: 'DNI',
  numero_documento: '12345678',
  telefono: '',
  email: '',
  direccion: '',
  ...over,
});

const valoresProveedor = (over: Partial<ValoresProveedor> = {}): ValoresProveedor => ({
  razon_social: 'Distribuidora SAC',
  tipo_documento: 'RUC',
  numero_documento: '20123456789',
  contacto_nombre: '',
  telefono: '',
  email: '',
  direccion: '',
  ...over,
});

describe('Estado de Clientes y proveedores', () => {
  const api = {
    clientes: vi.fn(),
    crearCliente: vi.fn(),
    editarCliente: vi.fn(),
    bajaCliente: vi.fn(),
    restaurarCliente: vi.fn(),
    proveedores: vi.fn(),
    crearProveedor: vi.fn(),
    editarProveedor: vi.fn(),
    bajaProveedor: vi.fn(),
    restaurarProveedor: vi.fn(),
  };
  let permisos: string[];

  function setup(): DirectorioStore {
    TestBed.configureTestingModule({
      providers: [
        DirectorioStore,
        { provide: DirectorioApi, useValue: api },
        { provide: AuthSession, useValue: { hasPermission: (p: string) => permisos.includes(p) } },
      ],
    });
    return TestBed.inject(DirectorioStore);
  }

  beforeEach(() => {
    permisos = ['CLIENTE_VER', 'CLIENTE_ADMINISTRAR', 'PROVEEDOR_VER', 'PROVEEDOR_ADMINISTRAR'];
    Object.values(api).forEach((f) => f.mockReset());
    api.clientes.mockReturnValue(of({ items: [cliente('1'), cliente('2')], nextCursor: null }));
    api.proveedores.mockReturnValue(of({ items: [proveedor('7')], nextCursor: null }));
  });

  describe('carga', () => {
    it('trae clientes y proveedores cuando el permiso lo deja ver', async () => {
      const store = setup();
      await store.load();
      expect(store.status()).toBe('ready');
      expect(store.clientes()).toHaveLength(2);
      expect(store.proveedores()).toHaveLength(1);
    });

    it('solo pide la lista que puede ver', async () => {
      permisos = ['PROVEEDOR_VER'];
      const store = setup();
      await store.load();
      expect(api.clientes).not.toHaveBeenCalled();
      expect(api.proveedores).toHaveBeenCalled();
      expect(store.puedeVerClientes()).toBe(false);
      expect(store.puedeAdministrarProveedores()).toBe(false);
    });

    it('sin ningún permiso no llama al servidor', async () => {
      permisos = [];
      const store = setup();
      await store.load();
      expect(store.sinAcceso()).toBe(true);
      expect(api.clientes).not.toHaveBeenCalled();
      expect(api.proveedores).not.toHaveBeenCalled();
    });

    it('recorre todas las páginas usando el cursor', async () => {
      api.clientes
        .mockReturnValueOnce(of({ items: [cliente('1')], nextCursor: 'c2' }))
        .mockReturnValueOnce(of({ items: [cliente('2')], nextCursor: null }));
      const store = setup();
      await store.load();
      expect(api.clientes).toHaveBeenNthCalledWith(2, 'c2');
      expect(store.clientes().map((c) => c.id)).toEqual(['1', '2']);
    });

    it('explica el fallo y deja reintentar', async () => {
      api.clientes.mockReturnValue(problem(503, 'x', 'interno'));
      const store = setup();
      await store.load();
      expect(store.status()).toBe('error');
      expect(store.loadError()).toMatch(/servicio no está disponible/);
    });
  });

  describe('clientes', () => {
    it('crea con el documento normalizado y vacíos como null', async () => {
      api.crearCliente.mockReturnValue(of(cliente('9')));
      const store = setup();
      await store.load();
      expect(
        await store.guardarCliente(undefined, valoresCliente({ nombre: '  Ana Ríos ' }), 'k1'),
      ).toBeUndefined();
      expect(api.crearCliente).toHaveBeenCalledWith(
        {
          nombre_razon_social: 'Ana Ríos',
          tipo_documento: 'DNI',
          numero_documento: '12345678',
          telefono: null,
          email: null,
          direccion: null,
        },
        'k1',
      );
      expect(store.notice()?.kind).toBe('ok');
    });

    it('sin número de documento no envía tipo (van siempre juntos)', async () => {
      api.crearCliente.mockReturnValue(of(cliente('9')));
      const store = setup();
      await store.load();
      await store.guardarCliente(undefined, valoresCliente({ numero_documento: '' }), 'k1');
      expect(api.crearCliente.mock.calls[0][0]).toMatchObject({
        tipo_documento: null,
        numero_documento: null,
      });
    });

    it('un documento repetido se muestra bajo el campo del número', async () => {
      api.crearCliente.mockReturnValue(
        problem(409, 'documento_duplicado', 'ya existe ese documento'),
      );
      const store = setup();
      await store.load();
      expect(await store.guardarCliente(undefined, valoresCliente(), 'k1')).toEqual({
        field: 'numero_documento',
        message: 'ya existe ese documento',
      });
    });

    it('un rechazo sin campo queda como aviso general', async () => {
      api.crearCliente.mockReturnValue(problem(403, 'sin_permiso', 'x'));
      const store = setup();
      await store.load();
      expect(await store.guardarCliente(undefined, valoresCliente(), 'k1')).toBe(
        'No tienes permiso para realizar esta acción.',
      );
    });

    it('al editar envía solo lo que cambió', async () => {
      api.editarCliente.mockReturnValue(of(cliente('1')));
      const store = setup();
      await store.load();
      const actual = cliente('1', { nombre_razon_social: 'Ana Ríos' });
      await store.guardarCliente(actual, valoresCliente({ telefono: '999 888 777' }), 'k1');
      expect(api.editarCliente).toHaveBeenCalledWith('1', { telefono: '999 888 777' });
    });

    it('si cambia el tipo o el número, envía el documento completo', async () => {
      api.editarCliente.mockReturnValue(of(cliente('1')));
      const store = setup();
      await store.load();
      const actual = cliente('1', { nombre_razon_social: 'Ana Ríos' });
      await store.guardarCliente(actual, valoresCliente({ numero_documento: '87654321' }), 'k1');
      expect(api.editarCliente).toHaveBeenCalledWith('1', {
        tipo_documento: 'DNI',
        numero_documento: '87654321',
      });
    });

    it('sin cambios no llama al servidor', async () => {
      const store = setup();
      await store.load();
      const actual = cliente('1', { nombre_razon_social: 'Ana Ríos' });
      expect(await store.guardarCliente(actual, valoresCliente(), 'k1')).toBeUndefined();
      expect(api.editarCliente).not.toHaveBeenCalled();
    });

    it('da de baja con motivo recortado y actualiza la lista', async () => {
      api.bajaCliente.mockReturnValue(of(cliente('1', { es_activo: false })));
      const store = setup();
      await store.load();
      api.clientes.mockReturnValue(
        of({ items: [cliente('1', { es_activo: false }), cliente('2')], nextCursor: null }),
      );
      expect(await store.bajaCliente(cliente('1'), '  Se mudó  ', 'k2')).toBeUndefined();
      expect(api.bajaCliente).toHaveBeenCalledWith('1', 'Se mudó', 'k2');
      expect(store.clientes()[0].es_activo).toBe(false);
    });

    it('restaurar informa el resultado y libera el botón', async () => {
      api.restaurarCliente.mockReturnValue(of(cliente('1')));
      const store = setup();
      await store.load();
      await store.restaurarCliente(cliente('1', { es_activo: false }), 'k3');
      expect(api.restaurarCliente).toHaveBeenCalledWith('1', 'k3');
      expect(store.busyId()).toBeNull();
      expect(store.notice()?.kind).toBe('ok');
    });

    it('un fallo al restaurar se avisa y libera el botón', async () => {
      api.restaurarCliente.mockReturnValue(problem(409, 'ya_activa', 'el cliente ya está activo'));
      const store = setup();
      await store.load();
      await store.restaurarCliente(cliente('1'), 'k3');
      expect(store.notice()).toEqual({ kind: 'error', text: 'el cliente ya está activo' });
      expect(store.busyId()).toBeNull();
    });
  });

  describe('proveedores', () => {
    it('crea con el documento y los opcionales vacíos como null', async () => {
      api.crearProveedor.mockReturnValue(of(proveedor('8')));
      const store = setup();
      await store.load();
      await store.guardarProveedor(undefined, valoresProveedor({ contacto_nombre: 'Luis' }), 'k1');
      expect(api.crearProveedor).toHaveBeenCalledWith(
        {
          razon_social: 'Distribuidora SAC',
          tipo_documento: 'RUC',
          numero_documento: '20123456789',
          contacto_nombre: 'Luis',
          telefono: null,
          email: null,
          direccion: null,
        },
        'k1',
      );
    });

    it('un RUC inválido se muestra bajo el campo del número', async () => {
      api.crearProveedor.mockReturnValue(problem(422, 'documento_invalido', 'RUC de 11 dígitos'));
      const store = setup();
      await store.load();
      expect(await store.guardarProveedor(undefined, valoresProveedor(), 'k1')).toEqual({
        field: 'numero_documento',
        message: 'RUC de 11 dígitos',
      });
    });

    it('al editar envía solo lo que cambió y vacía los opcionales con null', async () => {
      api.editarProveedor.mockReturnValue(of(proveedor('7')));
      const store = setup();
      await store.load();
      const actual = proveedor('7', { razon_social: 'Distribuidora SAC', contacto_nombre: 'Luis' });
      await store.guardarProveedor(actual, valoresProveedor({ contacto_nombre: '' }), 'k1');
      expect(api.editarProveedor).toHaveBeenCalledWith('7', { contacto_nombre: null });
    });

    it('la baja con motivo y la restauración usan su propia clave', async () => {
      api.bajaProveedor.mockReturnValue(of(proveedor('7', { es_activo: false })));
      api.restaurarProveedor.mockReturnValue(of(proveedor('7')));
      const store = setup();
      await store.load();
      await store.bajaProveedor(proveedor('7'), 'Dejó de abastecer', 'kb');
      await store.restaurarProveedor(proveedor('7', { es_activo: false }), 'kr');
      expect(api.bajaProveedor).toHaveBeenCalledWith('7', 'Dejó de abastecer', 'kb');
      expect(api.restaurarProveedor).toHaveBeenCalledWith('7', 'kr');
    });
  });
});
