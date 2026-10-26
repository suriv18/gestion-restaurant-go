import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { CartaStore, ValoresCategoria, ValoresProducto } from './carta-store';
import { Categoria, Estacion, Impuesto, Producto } from './carta.models';
import { CategoriasApi } from './categorias-api';
import { EstacionesApi } from './estaciones-api';
import { ImpuestosApi } from './impuestos-api';
import { ProductosApi } from './productos-api';

const problem = (status: number, code: string, detail: string) =>
  throwError(() => new HttpErrorResponse({ status, error: { status, code, detail } }));

const categoria = (id: string, over: Partial<Categoria> = {}): Categoria => ({
  id,
  nombre: `Categoria ${id}`,
  descripcion: null,
  orden_visualizacion: 0,
  es_activo: true,
  ...over,
});

const producto = (id: string, over: Partial<Producto> = {}): Producto => ({
  id,
  categoria_id: '1',
  codigo: `P${id}`,
  nombre: `Producto ${id}`,
  descripcion: null,
  tipo: 'PREPARADO',
  articulo_id: null,
  factor_consumo: '1.000000',
  precio_venta: '25.00',
  afectacion_igv: '10',
  impuesto_id: '1',
  imagen_url: null,
  es_activo: true,
  motivo_eliminacion: null,
  ...over,
});

const impuesto = (id: string, es_activo = true): Impuesto => ({
  id,
  codigo: `I${id}`,
  nombre: `Impuesto ${id}`,
  descripcion: null,
  es_activo,
});

const estacion = (id: string): Estacion => ({
  id,
  sucursal_id: '7',
  codigo: `E${id}`,
  nombre: `Estacion ${id}`,
  es_activo: true,
});

const valoresProducto = (over: Partial<ValoresProducto> = {}): ValoresProducto => ({
  categoria_id: '1',
  codigo: 'P1',
  nombre: 'Producto 1',
  descripcion: '',
  tipo: 'PREPARADO',
  factor_consumo: '1',
  precio_venta: '25',
  afectacion_igv: '10',
  impuesto_id: '1',
  imagen_url: '',
  ...over,
});

const valoresCategoria = (over: Partial<ValoresCategoria> = {}): ValoresCategoria => ({
  nombre: 'Categoria 1',
  descripcion: '',
  orden_visualizacion: '0',
  ...over,
});

const TODOS = [
  'CARTA_VER',
  'CARTA_ADMINISTRAR',
  'CARTA_DISPONIBILIDAD',
  'ESTACION_VER',
  'ESTACION_ADMINISTRAR',
  'IMPUESTO_VER',
  'IMPUESTO_ADMINISTRAR',
];

describe('Estado de la Carta', () => {
  const categorias = {
    listar: vi.fn(),
    crear: vi.fn(),
    editar: vi.fn(),
    baja: vi.fn(),
    restaurar: vi.fn(),
  };
  const productos = {
    listar: vi.fn(),
    obtener: vi.fn(),
    crear: vi.fn(),
    editar: vi.fn(),
    baja: vi.fn(),
    restaurar: vi.fn(),
    ponerDisponibilidad: vi.fn(),
  };
  const estaciones = { listar: vi.fn() };
  const impuestos = { impuestos: vi.fn() };
  let permisos: string[];
  let sedes: string[];

  function setup(): CartaStore {
    TestBed.configureTestingModule({
      providers: [
        CartaStore,
        { provide: CategoriasApi, useValue: categorias },
        { provide: ProductosApi, useValue: productos },
        { provide: EstacionesApi, useValue: estaciones },
        { provide: ImpuestosApi, useValue: impuestos },
        {
          provide: AuthSession,
          useValue: {
            hasPermission: (p: string) => permisos.includes(p),
            user: () => ({ branches: sedes }),
          },
        },
      ],
    });
    return TestBed.inject(CartaStore);
  }

  beforeEach(() => {
    permisos = [...TODOS];
    sedes = ['7', '8'];
    for (const api of [categorias, productos, estaciones, impuestos]) {
      Object.values(api).forEach((f) => f.mockReset());
    }
    categorias.listar.mockReturnValue(
      of({ items: [categoria('2', { orden_visualizacion: 2 }), categoria('1')], nextCursor: null }),
    );
    productos.listar.mockReturnValue(
      of({ items: [producto('1'), producto('2')], nextCursor: null }),
    );
    estaciones.listar.mockReturnValue(of({ items: [estacion('4')], nextCursor: null }));
    impuestos.impuestos.mockReturnValue(
      of({ items: [impuesto('1'), impuesto('2', false)], nextCursor: null }),
    );
  });

  describe('permisos y carga', () => {
    it('ADMIN lo ve y lo hace todo, y carga apoyo de formularios', async () => {
      const store = setup();
      await store.load();
      expect(store.status()).toBe('ready');
      expect(store.puedeVerCarta()).toBe(true);
      expect(store.puedeAdministrarCarta()).toBe(true);
      expect(store.puedePausar()).toBe(true);
      expect(store.puedeVerEstaciones()).toBe(true);
      expect(store.puedeVerImpuestos()).toBe(true);
      expect(store.sinAcceso()).toBe(false);
      expect(store.productos().map((p) => p.id)).toEqual(['1', '2']);
      expect(store.categoriasOrdenadas().map((c) => c.id)).toEqual(['1', '2']);
      expect(store.impuestos().map((i) => i.id)).toEqual(['1']);
      expect(store.estaciones().map((e) => e.id)).toEqual(['4']);
      expect(estaciones.listar).toHaveBeenCalledWith(
        { sucursal_id: '7', es_activo: true },
        undefined,
      );
    });

    it('COCINA solo pausa: ve la carta y no administra ni pide impuestos ni estaciones', async () => {
      permisos = ['CARTA_DISPONIBILIDAD'];
      const store = setup();
      await store.load();
      expect(store.puedeVerCarta()).toBe(true);
      expect(store.puedePausar()).toBe(true);
      expect(store.puedeAdministrarCarta()).toBe(false);
      expect(store.puedeVerEstaciones()).toBe(false);
      expect(store.puedeVerImpuestos()).toBe(false);
      expect(impuestos.impuestos).not.toHaveBeenCalled();
      expect(estaciones.listar).not.toHaveBeenCalled();
    });

    it('MESERO solo ve', async () => {
      permisos = ['CARTA_VER'];
      const store = setup();
      await store.load();
      expect(store.puedeVerCarta()).toBe(true);
      expect(store.puedePausar()).toBe(false);
      expect(store.puedeAdministrarCarta()).toBe(false);
      expect(store.productos()).toHaveLength(2);
    });

    it('sin permisos de carta no pide nada y queda sin acceso', async () => {
      permisos = ['IMPUESTO_VER'];
      const store = setup();
      await store.load();
      expect(store.sinAcceso()).toBe(true);
      expect(store.status()).toBe('ready');
      expect(categorias.listar).not.toHaveBeenCalled();
      expect(productos.listar).not.toHaveBeenCalled();
    });

    it('CARTA_ADMINISTRAR basta para ver impuestos, sin permiso de estaciones no pide estaciones', async () => {
      permisos = ['CARTA_ADMINISTRAR'];
      const store = setup();
      await store.load();
      expect(store.puedeVerImpuestos()).toBe(true);
      expect(impuestos.impuestos).toHaveBeenCalled();
      expect(estaciones.listar).not.toHaveBeenCalled();
    });

    it('si falla la carga principal queda en error con el detalle y puede reintentar', async () => {
      productos.listar.mockReturnValueOnce(problem(500, 'x', 'interno'));
      const store = setup();
      await store.load();
      expect(store.status()).toBe('error');
      expect(store.loadError()).toBeTruthy();
      await store.load();
      expect(store.status()).toBe('ready');
    });

    it('la sede inicial es la primera de la sesión y envía es_activo explícito', async () => {
      const store = setup();
      await store.load();
      expect(productos.listar).toHaveBeenCalledWith({ es_activo: 'true', sucursal_id: '7' });
      expect(store.filtros().sucursalId).toBe('7');
    });

    it('sin sedes en la sesión lista todas las sedes', async () => {
      sedes = [];
      const store = setup();
      await store.load();
      expect(productos.listar).toHaveBeenCalledWith({ es_activo: 'true' });
    });
  });

  describe('filtros y cursor', () => {
    it('cambiar filtros reinicia la lista con los parámetros del servidor', async () => {
      const store = setup();
      await store.load();
      productos.listar.mockClear();
      await store.cambiarFiltros({
        q: '  lomo ',
        categoriaId: '3',
        estado: 'bajas',
        soloDisponibles: false,
      });
      expect(productos.listar).toHaveBeenCalledWith({
        es_activo: 'false',
        q: 'lomo',
        categoria_id: '3',
        sucursal_id: '7',
        disponible: false,
      });
      await store.cambiarFiltros({ estado: 'todos' });
      expect(productos.listar).toHaveBeenLastCalledWith(
        expect.objectContaining({ es_activo: 'todos' }),
      );
    });

    it('todas las sedes quita sucursal_id y disponible, y vacía las estaciones', async () => {
      const store = setup();
      await store.load();
      await store.cambiarFiltros({ soloDisponibles: true });
      productos.listar.mockClear();
      await store.cambiarFiltros({ sucursalId: null });
      expect(productos.listar).toHaveBeenCalledWith({ es_activo: 'true' });
      expect(store.filtros().soloDisponibles).toBeNull();
      expect(store.estaciones()).toEqual([]);
    });

    it('cambiar de sede vuelve a pedir las estaciones de esa sede', async () => {
      const store = setup();
      await store.load();
      estaciones.listar.mockClear();
      await store.cambiarFiltros({ sucursalId: '8' });
      expect(estaciones.listar).toHaveBeenCalledWith(
        { sucursal_id: '8', es_activo: true },
        undefined,
      );
    });

    it('«cargar más» acumula con nextCursor hasta agotarlo', async () => {
      productos.listar
        .mockReturnValueOnce(of({ items: [producto('1')], nextCursor: 'c1' }))
        .mockReturnValueOnce(of({ items: [producto('2')], nextCursor: null }));
      const store = setup();
      await store.load();
      expect(store.hayMas()).toBe(true);
      await store.cargarMas();
      expect(productos.listar).toHaveBeenLastCalledWith(expect.any(Object), 'c1');
      expect(store.productos().map((p) => p.id)).toEqual(['1', '2']);
      expect(store.hayMas()).toBe(false);
      productos.listar.mockClear();
      await store.cargarMas();
      expect(productos.listar).not.toHaveBeenCalled();
    });

    it('un error al cargar más conserva lo acumulado y avisa', async () => {
      productos.listar
        .mockReturnValueOnce(of({ items: [producto('1')], nextCursor: 'c1' }))
        .mockReturnValueOnce(problem(503, 'x', 'caído'));
      const store = setup();
      await store.load();
      await store.cargarMas();
      expect(store.productos()).toHaveLength(1);
      expect(store.hayMas()).toBe(true);
      expect(store.notice()?.kind).toBe('error');
    });

    it('un fallo al filtrar deja la lista en error recuperable', async () => {
      const store = setup();
      await store.load();
      productos.listar.mockReturnValueOnce(problem(500, 'x', 'interno'));
      await store.cambiarFiltros({ q: 'a' });
      expect(store.listaStatus()).toBe('error');
      expect(await store.recargarProductos()).toBe(true);
      expect(store.listaStatus()).toBe('ready');
    });
  });

  describe('categorías', () => {
    it('crea con clave, refresca y avisa', async () => {
      categorias.crear.mockReturnValue(of(categoria('3')));
      const store = setup();
      await store.load();
      categorias.listar.mockClear();
      const r = await store.crearCategoria(
        valoresCategoria({ nombre: ' Postres ', orden_visualizacion: '4' }),
        'k1',
      );
      expect(r).toBeUndefined();
      expect(categorias.crear).toHaveBeenCalledWith(
        { nombre: 'Postres', descripcion: null, orden_visualizacion: 4 },
        'k1',
      );
      expect(categorias.listar).toHaveBeenCalled();
      expect(store.notice()).toEqual({ kind: 'ok', text: 'Se creó la categoría Postres.' });
    });

    it('un duplicado se devuelve como error del campo nombre y no refresca', async () => {
      categorias.crear.mockReturnValue(problem(409, 'categoria_duplicada', 'Ya existe'));
      const store = setup();
      await store.load();
      categorias.listar.mockClear();
      expect(await store.crearCategoria(valoresCategoria(), 'k')).toEqual({
        field: 'nombre',
        message: 'Ya existe',
      });
      expect(categorias.listar).not.toHaveBeenCalled();
    });

    it('editar sin cambios no llama; con cambios envía solo lo cambiado', async () => {
      categorias.editar.mockReturnValue(of(categoria('1')));
      const store = setup();
      await store.load();
      const actual = categoria('1', { descripcion: 'Antes' });
      expect(
        await store.editarCategoria(
          actual,
          valoresCategoria({ nombre: 'Categoria 1', descripcion: 'Antes' }),
        ),
      ).toBeUndefined();
      expect(categorias.editar).not.toHaveBeenCalled();
      await store.editarCategoria(
        actual,
        valoresCategoria({ descripcion: '', orden_visualizacion: '5' }),
      );
      expect(categorias.editar).toHaveBeenCalledWith('1', {
        descripcion: null,
        orden_visualizacion: 5,
      });
    });

    it('desactivar sin motivo: el 409 con productos activos se muestra con su detalle', async () => {
      categorias.baja.mockReturnValue(
        problem(409, 'categoria_con_productos_activos', 'Tiene productos activos'),
      );
      const store = setup();
      await store.load();
      expect(await store.desactivarCategoria(categoria('1'), 'k')).toBe('Tiene productos activos');
      expect(categorias.baja).toHaveBeenCalledWith('1', 'k');
    });

    it('desactivar y restaurar refrescan la lista', async () => {
      categorias.baja.mockReturnValue(of(categoria('1', { es_activo: false })));
      categorias.restaurar.mockReturnValue(of(categoria('1')));
      const store = setup();
      await store.load();
      categorias.listar.mockClear();
      expect(await store.desactivarCategoria(categoria('1'), 'k')).toBeUndefined();
      await store.restaurarCategoria(categoria('1', { es_activo: false }), 'k2');
      expect(categorias.restaurar).toHaveBeenCalledWith('1', 'k2');
      expect(categorias.listar).toHaveBeenCalledTimes(2);
      expect(store.busyId()).toBeNull();
    });

    it('categoriasActivas excluye las bajas', async () => {
      categorias.listar.mockReturnValue(
        of({ items: [categoria('1'), categoria('2', { es_activo: false })], nextCursor: null }),
      );
      const store = setup();
      await store.load();
      expect(store.categoriasActivas().map((c) => c.id)).toEqual(['1']);
    });
  });

  describe('productos', () => {
    it('crea con clave, normaliza el cuerpo y recarga la primera página', async () => {
      productos.crear.mockReturnValue(of(producto('9')));
      const store = setup();
      await store.load();
      productos.listar.mockClear();
      const r = await store.crearProducto(
        valoresProducto({ nombre: ' Lomo ', precio_venta: ' 25.5 ' }),
        'k1',
      );
      expect(r).toBeUndefined();
      expect(productos.crear).toHaveBeenCalledWith(
        {
          categoria_id: '1',
          codigo: 'P1',
          nombre: 'Lomo',
          descripcion: null,
          tipo: 'PREPARADO',
          factor_consumo: '1',
          precio_venta: '25.5',
          afectacion_igv: '10',
          impuesto_id: '1',
          imagen_url: null,
        },
        'k1',
      );
      expect(productos.listar).toHaveBeenCalledTimes(1);
      expect(store.notice()?.kind).toBe('ok');
    });

    it('un servicio no usa factor: viaja «1»', async () => {
      productos.crear.mockReturnValue(of(producto('9')));
      const store = setup();
      await store.load();
      await store.crearProducto(valoresProducto({ tipo: 'SERVICIO', factor_consumo: '3' }), 'k');
      expect(productos.crear.mock.calls[0][0].factor_consumo).toBe('1');
    });

    it.each([
      ['codigo_duplicado', 'codigo'],
      ['categoria_id', 'categoria_id'],
      ['referencia_invalida', 'categoria_id'],
      ['impuesto_requerido', 'impuesto_id'],
      ['articulo_requerido', 'tipo'],
      ['nombre_invalido', 'nombre'],
      ['precio_invalido', 'precio_venta'],
      ['factor_consumo_invalido', 'factor_consumo'],
      ['afectacion_invalida', 'afectacion_igv'],
    ])('el código %s va al campo %s y conserva el diálogo', async (code, field) => {
      productos.crear.mockReturnValue(problem(422, code, 'detalle'));
      const store = setup();
      await store.load();
      expect(await store.crearProducto(valoresProducto(), 'k')).toEqual({
        field,
        message: 'detalle',
      });
    });

    it('un rechazo sin campo vuelve como texto general', async () => {
      productos.crear.mockReturnValue(problem(500, 'raro', 'x'));
      const store = setup();
      await store.load();
      expect(typeof (await store.crearProducto(valoresProducto(), 'k'))).toBe('string');
    });

    it('editar sin cambios no llama al servidor (25 = 25.00, 1 = 1.000000)', async () => {
      const store = setup();
      await store.load();
      const r = await store.editarProducto(producto('1'), valoresProducto());
      expect(r).toBeUndefined();
      expect(productos.editar).not.toHaveBeenCalled();
    });

    it('editar envía solo los campos cambiados y null para vaciar', async () => {
      productos.editar.mockReturnValue(of(producto('1')));
      const store = setup();
      await store.load();
      productos.listar.mockClear();
      const actual = producto('1', { descripcion: 'Rico', imagen_url: 'http://x/y.png' });
      await store.editarProducto(
        actual,
        valoresProducto({
          nombre: 'Nuevo nombre',
          precio_venta: '30',
          descripcion: '',
          imagen_url: 'http://x/y.png',
        }),
      );
      expect(productos.editar).toHaveBeenCalledWith('1', {
        nombre: 'Nuevo nombre',
        precio_venta: '30',
        descripcion: null,
      });
      expect(productos.listar).toHaveBeenCalledTimes(1);
    });

    it('un producto REVENTA conserva su tipo: no se envía', async () => {
      productos.editar.mockReturnValue(of(producto('1')));
      const store = setup();
      await store.load();
      await store.editarProducto(
        producto('1', {
          tipo: 'REVENTA',
          articulo_id: '5',
          afectacion_igv: '20',
          impuesto_id: null,
        }),
        valoresProducto({ tipo: 'REVENTA', afectacion_igv: '20', impuesto_id: '', nombre: 'Otro' }),
      );
      expect(productos.editar).toHaveBeenCalledWith('1', { nombre: 'Otro' });
    });

    it('baja con motivo y clave, y recarga; el motivo vacío del servidor se ve en su campo', async () => {
      productos.baja.mockReturnValueOnce(problem(422, 'motivo_requerido', 'Indica el motivo'));
      const store = setup();
      await store.load();
      expect(await store.darDeBajaProducto(producto('1'), '  ', 'k')).toEqual({
        field: 'motivo',
        message: 'Indica el motivo',
      });
      productos.baja.mockReturnValue(of(producto('1', { es_activo: false })));
      productos.listar.mockClear();
      expect(await store.darDeBajaProducto(producto('1'), ' Ya no ', 'k2')).toBeUndefined();
      expect(productos.baja).toHaveBeenLastCalledWith('1', 'Ya no', 'k2');
      expect(productos.listar).toHaveBeenCalledTimes(1);
    });

    it('restaurar usa la clave, bloquea la fila y avisa del error del servidor', async () => {
      productos.restaurar.mockReturnValueOnce(
        problem(409, 'referencia_inactiva', 'Categoría de baja'),
      );
      const store = setup();
      await store.load();
      await store.restaurarProducto(producto('1', { es_activo: false }), 'k');
      expect(store.notice()).toEqual({ kind: 'error', text: 'Categoría de baja' });
      expect(store.busyId()).toBeNull();
      productos.restaurar.mockReturnValue(of(producto('1')));
      await store.restaurarProducto(producto('1', { es_activo: false }), 'k2');
      expect(store.notice()?.kind).toBe('ok');
    });
  });

  describe('disponibilidad por sede', () => {
    it('pausar envía PUT sin precio ni estación y refresca solo esa fila', async () => {
      productos.ponerDisponibilidad.mockReturnValue(of({}));
      productos.obtener.mockReturnValue(
        of(
          producto('1', {
            en_sucursal: {
              disponible: false,
              precio_sucursal: null,
              precio_efectivo: '25.00',
              estacion_id: null,
            },
          }),
        ),
      );
      const store = setup();
      await store.load();
      productos.listar.mockClear();
      await store.pausarEnSede(producto('1'));
      expect(productos.ponerDisponibilidad).toHaveBeenCalledWith('1', {
        sucursal_id: '7',
        disponible: false,
      });
      expect(productos.obtener).toHaveBeenCalledWith('1', '7');
      expect(productos.listar).not.toHaveBeenCalled();
      expect(store.productos()[0].en_sucursal?.disponible).toBe(false);
      expect(store.productos()[1].id).toBe('2');
      expect(store.notice()?.kind).toBe('ok');
    });

    it('habilitar envía disponible true', async () => {
      productos.ponerDisponibilidad.mockReturnValue(of({}));
      productos.obtener.mockReturnValue(of(producto('1')));
      const store = setup();
      await store.load();
      await store.habilitarEnSede(producto('1'));
      expect(productos.ponerDisponibilidad).toHaveBeenCalledWith('1', {
        sucursal_id: '7',
        disponible: true,
      });
    });

    it('sin sede elegida no llama y explica', async () => {
      const store = setup();
      await store.load();
      await store.cambiarFiltros({ sucursalId: null });
      await store.pausarEnSede(producto('1'));
      expect(productos.ponerDisponibilidad).not.toHaveBeenCalled();
      expect(store.notice()?.kind).toBe('error');
    });

    it('un rechazo del servidor se avisa y libera la fila', async () => {
      productos.ponerDisponibilidad.mockReturnValue(
        problem(409, 'producto_inactivo', 'Está de baja'),
      );
      const store = setup();
      await store.load();
      await store.pausarEnSede(producto('1'));
      expect(store.notice()).toEqual({ kind: 'error', text: 'Está de baja' });
      expect(store.busyId()).toBeNull();
    });

    it('ajustar sede envía solo lo cambiado: precio vacío = null, estación elegida', async () => {
      productos.ponerDisponibilidad.mockReturnValue(of({}));
      productos.obtener.mockReturnValue(of(producto('1')));
      const store = setup();
      await store.load();
      const p = producto('1', {
        en_sucursal: {
          disponible: false,
          precio_sucursal: '30.00',
          precio_efectivo: '30.00',
          estacion_id: null,
        },
      });
      expect(await store.ajustarSede(p, { precio: '', estacionId: '4' })).toBeUndefined();
      expect(productos.ponerDisponibilidad).toHaveBeenCalledWith('1', {
        sucursal_id: '7',
        disponible: false,
        precio_venta: null,
        estacion_id: '4',
      });
    });

    it('ajustar sede sin cambios (30 = 30.00) no llama', async () => {
      const store = setup();
      await store.load();
      const p = producto('1', {
        en_sucursal: {
          disponible: true,
          precio_sucursal: '30.00',
          precio_efectivo: '30.00',
          estacion_id: '4',
        },
      });
      expect(await store.ajustarSede(p, { precio: '30', estacionId: '4' })).toBeUndefined();
      expect(productos.ponerDisponibilidad).not.toHaveBeenCalled();
    });

    it('un producto no ofrecido queda disponible al darle ajustes, y el precio inválido va a su campo', async () => {
      productos.ponerDisponibilidad
        .mockReturnValueOnce(problem(422, 'precio_invalido', 'Precio inválido'))
        .mockReturnValueOnce(of({}));
      productos.obtener.mockReturnValue(of(producto('1')));
      const store = setup();
      await store.load();
      const p = producto('1', { en_sucursal: null });
      expect(await store.ajustarSede(p, { precio: '1.234', estacionId: '' })).toEqual({
        field: 'precio',
        message: 'Precio inválido',
      });
      await store.ajustarSede(p, { precio: '12', estacionId: '' });
      expect(productos.ponerDisponibilidad).toHaveBeenLastCalledWith('1', {
        sucursal_id: '7',
        disponible: true,
        precio_venta: '12',
      });
    });
  });
});
