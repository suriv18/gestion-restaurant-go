import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs, type FormDialogData } from '../../shared/ui/form-dialog';
import { CartaStore, type FiltrosCarta } from './data-access/carta-store';
import type { Categoria, Estacion, Impuesto, Producto } from './data-access/carta.models';
import { ProductosSection } from './productos-section';

const categoria = (id: string, nombre: string, es_activo = true): Categoria => ({
  id,
  nombre,
  descripcion: null,
  orden_visualizacion: 0,
  es_activo,
});

const producto = (id: string, over: Partial<Producto> = {}): Producto => ({
  id,
  categoria_id: '1',
  codigo: 'LOMO' + id,
  nombre: 'Lomo ' + id,
  descripcion: null,
  tipo: 'PREPARADO',
  articulo_id: null,
  factor_consumo: '1.000000',
  precio_venta: '25.00',
  afectacion_igv: '10',
  impuesto_id: '7',
  imagen_url: null,
  es_activo: true,
  motivo_eliminacion: null,
  ...over,
});

const FILTROS: FiltrosCarta = {
  q: '',
  categoriaId: null,
  estado: 'activos',
  sucursalId: null,
  soloDisponibles: null,
};

interface Opciones {
  admin?: boolean;
  pausar?: boolean;
  productos?: Producto[];
  filtros?: Partial<FiltrosCarta>;
  hayMas?: boolean;
  listaStatus?: 'loading' | 'ready' | 'error';
}

function setup(o: Opciones = {}) {
  const admin = o.admin ?? true;
  const store = {
    filtros: signal<FiltrosCarta>({ ...FILTROS, ...o.filtros }),
    productos: signal<Producto[]>(o.productos ?? [producto('1'), producto('2')]),
    categorias: signal([categoria('1', 'Fondos'), categoria('2', 'Postres')]),
    categoriasActivas: signal([categoria('1', 'Fondos'), categoria('2', 'Postres')]),
    impuestos: signal<Impuesto[]>([
      { id: '7', codigo: 'IGV', nombre: 'IGV', descripcion: null, es_activo: true },
    ]),
    estaciones: signal<Estacion[]>([
      { id: '5', sucursal_id: '1', codigo: 'COC', nombre: 'Cocina caliente', es_activo: true },
    ]),
    sedes: signal(['1', '2']),
    nombreSede: (id: string) => (id === '2' ? 'Miraflores' : `Sede ${id}`),
    listaStatus: signal(o.listaStatus ?? 'ready'),
    listaError: signal('Sin conexión'),
    cargandoMas: signal(false),
    hayMas: signal(o.hayMas ?? false),
    busyId: signal<string | null>(null),
    puedeAdministrarCarta: signal(admin),
    puedePausar: signal(o.pausar ?? admin),
    cambiarFiltros: vi.fn(),
    recargarProductos: vi.fn(),
    cargarMas: vi.fn(),
    pausarEnSede: vi.fn(),
    habilitarEnSede: vi.fn(),
    restaurarProducto: vi.fn(),
    darDeBajaProducto: vi.fn().mockResolvedValue(undefined),
    crearProducto: vi.fn().mockResolvedValue(undefined),
    editarProducto: vi.fn().mockResolvedValue(undefined),
    ajustarSede: vi.fn().mockResolvedValue(undefined),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [ProductosSection],
    providers: [
      { provide: CartaStore, useValue: store },
      { provide: Dialogs, useValue: dialogs },
    ],
  });
  const fixture = TestBed.createComponent(ProductosSection);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const boton = (texto: string) =>
    [...el.querySelectorAll('button')].find(
      (b) => b.getAttribute('aria-label') === texto || b.textContent?.trim() === texto,
    ) as HTMLButtonElement | undefined;
  const select = (etiqueta: string) =>
    el.querySelector(`select[aria-label="${etiqueta}"]`) as HTMLSelectElement | null;
  const elegir = (etiqueta: string, valor: string) => {
    const s = select(etiqueta)!;
    s.value = valor;
    s.dispatchEvent(new Event('change'));
  };
  const ultimoDialogo = () => dialogs.form.mock.calls.at(-1)![0] as FormDialogData;
  return { store, dialogs, fixture, el, boton, select, elegir, ultimoDialogo };
}

const textos = (el: HTMLElement) =>
  [...el.querySelectorAll('button')].map(
    (b) => b.getAttribute('aria-label') ?? b.textContent?.trim(),
  );

describe('Sección Productos: tabla', () => {
  it('muestra producto, código, categoría, tipo, precio e IGV legibles', () => {
    const { el } = setup();
    const fila = el.querySelector('tbody tr')!.textContent!;
    expect(fila).toContain('Lomo 1');
    expect(fila).toContain('LOMO1');
    expect(fila).toContain('Fondos');
    expect(fila).toContain('Preparado');
    expect(fila).toContain('S/ 25.00');
    expect(fila).toContain('Gravado');
    expect(fila).toContain('IGV');
    expect(fila).toContain('Activo');
    expect(el.querySelector('caption')?.textContent).toContain('Productos de la carta');
  });

  it('usa «—» cuando la categoría no se conoce y no muestra códigos crudos', () => {
    const { el } = setup({ productos: [producto('1', { categoria_id: '99', tipo: 'REVENTA' })] });
    const fila = el.querySelector('tbody tr')!.textContent!;
    expect(fila).toContain('—');
    expect(fila).toContain('Reventa');
    expect(el.textContent).not.toContain('es_activo');
  });

  it('sin sede no hay columna de disponibilidad', () => {
    const { el } = setup();
    expect(el.textContent).not.toContain('Disponibilidad');
  });

  it('con sede muestra disponibilidad, estación y el precio de sede con el general tachado', () => {
    const { el } = setup({
      filtros: { sucursalId: '1' },
      productos: [
        producto('1', {
          en_sucursal: {
            disponible: true,
            precio_sucursal: '30.00',
            precio_efectivo: '30.00',
            estacion_id: '5',
          },
        }),
        producto('2', {
          en_sucursal: {
            disponible: false,
            precio_sucursal: null,
            precio_efectivo: '25.00',
            estacion_id: null,
          },
        }),
        producto('3', { en_sucursal: null }),
      ],
    });
    const filas = [...el.querySelectorAll('tbody tr')].map((f) => f.textContent!);
    expect(el.textContent).toContain('Disponibilidad');
    expect(filas[0]).toContain('S/ 30.00');
    expect(el.querySelector('tbody tr s')?.textContent).toBe('S/ 25.00');
    expect(filas[0]).toContain('precio de sede');
    expect(filas[0]).toContain('Disponible');
    expect(filas[0]).toContain('Cocina caliente');
    expect(filas[1]).toContain('Pausado');
    expect(filas[1]).not.toContain('precio de sede');
    expect(filas[2]).toContain('No ofrecido');
  });

  it('un precio de sede igual al general no se marca como precio de sede', () => {
    const { el } = setup({
      filtros: { sucursalId: '1' },
      productos: [
        producto('1', {
          en_sucursal: {
            disponible: true,
            precio_sucursal: '25',
            precio_efectivo: '25.00',
            estacion_id: null,
          },
        }),
      ],
    });
    expect(el.textContent).not.toContain('precio de sede');
  });

  it('un producto de baja muestra su motivo', () => {
    const { el } = setup({
      productos: [producto('1', { es_activo: false, motivo_eliminacion: 'Ya no se vende' })],
    });
    expect(el.textContent).toContain('De baja');
    expect(el.textContent).toContain('Motivo: Ya no se vende');
  });
});

describe('Sección Productos: permisos por fila', () => {
  it('ADMIN ve editar, baja y restaurar (en productos de baja)', () => {
    const { el } = setup({
      productos: [
        producto('1', { nombre: 'Lomo' }),
        producto('2', { nombre: 'Ají', es_activo: false }),
      ],
    });
    expect(textos(el)).toEqual(
      expect.arrayContaining([
        'Editar producto Lomo',
        'Dar de baja producto Lomo',
        'Restaurar producto Ají',
      ]),
    );
    expect(textos(el)).not.toContain('Editar producto Ají');
  });

  it('MESERO (solo lectura) no ve ninguna acción', () => {
    const { el } = setup({ admin: false, pausar: false });
    expect(el.querySelectorAll('tbody .inline-actions button').length).toBe(0);
    expect(el.textContent).not.toContain('Elige una sede para pausar');
  });

  it('COCINA solo ve Pausar o Habilitar en sede', () => {
    const { el } = setup({
      admin: false,
      pausar: true,
      filtros: { sucursalId: '1' },
      productos: [
        producto('1', {
          nombre: 'Lomo',
          en_sucursal: {
            disponible: true,
            precio_sucursal: null,
            precio_efectivo: '25.00',
            estacion_id: null,
          },
        }),
        producto('2', { nombre: 'Ají', en_sucursal: null }),
      ],
    });
    expect(textos(el).filter((t) => t !== 'Buscar')).toEqual([
      'Pausar producto Lomo',
      'Habilitar en sede el producto Ají',
    ]);
  });

  it('sin sede, Pausar queda deshabilitado con la explicación', () => {
    const { el } = setup();
    const pausar = el.querySelector('tbody .inline-actions button[title]') as HTMLButtonElement;
    expect(pausar.disabled).toBe(true);
    expect(pausar.title).toContain('Elige una sede');
    expect(el.textContent).toContain('Elige una sede para pausar o habilitar');
    expect(textos(el).some((t) => t?.startsWith('Ajustes de sede'))).toBe(false);
  });

  it('con sede, Pausar llama al store y Habilitar también; Ajustes de sede aparece para ADMIN', () => {
    const lomo = producto('1', {
      nombre: 'Lomo',
      en_sucursal: {
        disponible: true,
        precio_sucursal: null,
        precio_efectivo: '25.00',
        estacion_id: null,
      },
    });
    const aji = producto('2', {
      nombre: 'Ají',
      en_sucursal: {
        disponible: false,
        precio_sucursal: null,
        precio_efectivo: '25.00',
        estacion_id: null,
      },
    });
    const { store, boton } = setup({ filtros: { sucursalId: '1' }, productos: [lomo, aji] });
    boton('Pausar producto Lomo')!.click();
    boton('Habilitar en sede el producto Ají')!.click();
    expect(store.pausarEnSede).toHaveBeenCalledWith(lomo);
    expect(store.habilitarEnSede).toHaveBeenCalledWith(aji);
    expect(boton('Ajustes de sede del producto Lomo')).toBeDefined();
  });

  it('Restaurar usa una clave de idempotencia', () => {
    const p = producto('1', { nombre: 'Ají', es_activo: false });
    const { store, boton } = setup({ productos: [p] });
    boton('Restaurar producto Ají')!.click();
    expect(store.restaurarProducto).toHaveBeenCalledWith(p, expect.any(String));
  });
});

describe('Sección Productos: filtros', () => {
  it('cada cambio de filtro se envía al servidor', () => {
    const { store, elegir } = setup();
    elegir('Categoría', '2');
    elegir('Estado', 'bajas');
    elegir('Sede', '2');
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(1, { categoriaId: '2' });
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(2, { estado: 'bajas' });
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(3, { sucursalId: '2' });
  });

  it('el selector muestra el nombre de la sede que da el store', () => {
    const { el } = setup();
    const opciones = [...el.querySelectorAll('select[aria-label="Sede"] option')].map((o) =>
      o.textContent?.trim(),
    );
    expect(opciones).toEqual(['Todas las sedes', 'Sede 1', 'Miraflores']);
  });

  it('«Todas las categorías» y «Todas las sedes» envían null', () => {
    const { store, elegir } = setup({ filtros: { categoriaId: '2', sucursalId: '1' } });
    elegir('Categoría', '');
    elegir('Sede', '');
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(1, { categoriaId: null });
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(2, { sucursalId: null });
  });

  it('la búsqueda se envía al pulsar Buscar, sin recortar en el cliente', () => {
    const { store, el, boton } = setup();
    const input = el.querySelector('input[aria-label="Buscar producto"]') as HTMLInputElement;
    input.value = '  lomo ';
    input.dispatchEvent(new Event('input'));
    expect(store.cambiarFiltros).not.toHaveBeenCalled();
    boton('Buscar')!.click();
    expect(store.cambiarFiltros).toHaveBeenCalledWith({ q: 'lomo' });
  });

  it('el filtro de disponibilidad solo existe con sede y explica «No ofrecidos»', () => {
    expect(setup().select('Disponibilidad')).toBeNull();
    TestBed.resetTestingModule();
    const { select, elegir, store, el } = setup({ filtros: { sucursalId: '1' } });
    expect(select('Disponibilidad')).not.toBeNull();
    expect(el.textContent).toContain('los «No ofrecidos» no aparecen ahí');
    elegir('Disponibilidad', 'no');
    elegir('Disponibilidad', 'si');
    elegir('Disponibilidad', '');
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(1, { soloDisponibles: false });
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(2, { soloDisponibles: true });
    expect(store.cambiarFiltros).toHaveBeenNthCalledWith(3, { soloDisponibles: null });
  });

  it('«Limpiar filtros» solo aparece con filtros activos y los reinicia', () => {
    expect(setup().boton('Limpiar filtros')).toBeUndefined();
    TestBed.resetTestingModule();
    const { boton, store } = setup({ filtros: { estado: 'todos', q: 'x' } });
    boton('Limpiar filtros')!.click();
    expect(store.cambiarFiltros).toHaveBeenCalledWith({
      q: '',
      categoriaId: null,
      estado: 'activos',
      soloDisponibles: null,
    });
  });

  it('la sede por sí sola no cuenta como filtro', () => {
    expect(setup({ filtros: { sucursalId: '1' } }).boton('Limpiar filtros')).toBeUndefined();
  });
});

describe('Sección Productos: estados y paginación', () => {
  it('vacío con filtros ofrece limpiar', () => {
    const { el, store } = setup({ productos: [], filtros: { q: 'zzz' } });
    expect(el.textContent).toContain('No encontramos productos con esos filtros');
    (el.querySelector('.empty-state button') as HTMLButtonElement).click();
    expect(store.cambiarFiltros).toHaveBeenCalled();
  });

  it('vacío sin datos invita a crear solo a quien puede', () => {
    const admin = setup({ productos: [] });
    expect(admin.el.textContent).toContain('Aún no hay productos');
    expect(admin.boton('+ Nuevo producto')).toBeDefined();
    TestBed.resetTestingModule();
    const mesero = setup({ productos: [], admin: false, pausar: false });
    expect(mesero.el.textContent).toContain('Aún no hay productos');
    expect(mesero.boton('+ Nuevo producto')).toBeUndefined();
  });

  it('error de recarga muestra el motivo y Reintentar', () => {
    const { el, boton, store } = setup({ listaStatus: 'error' });
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Sin conexión');
    boton('Reintentar')!.click();
    expect(store.recargarProductos).toHaveBeenCalled();
  });

  it('cargando con la lista vacía avisa; con filas las conserva', () => {
    const vacio = setup({ listaStatus: 'loading', productos: [] });
    expect(vacio.el.textContent).toContain('Buscando productos');
    TestBed.resetTestingModule();
    const con = setup({ listaStatus: 'loading' });
    expect(con.el.textContent).toContain('Actualizando la lista');
    expect(con.el.querySelectorAll('tbody tr').length).toBe(2);
  });

  it('«Cargar más» solo con siguiente página, con contador; se deshabilita mientras carga', () => {
    expect(setup().boton('Cargar más')).toBeUndefined();
    TestBed.resetTestingModule();
    const { el, boton, store, fixture } = setup({ hayMas: true });
    expect(el.textContent).toContain('Mostrando 2');
    boton('Cargar más')!.click();
    expect(store.cargarMas).toHaveBeenCalled();
    store.cargandoMas.set(true);
    fixture.detectChanges();
    expect(boton('Cargando…')!.disabled).toBe(true);
  });
});

describe('Sección Productos: diálogos', () => {
  const valores = (over: Record<string, string> = {}): Record<string, string> => ({
    categoria_id: '1',
    codigo: ' lomo ',
    nombre: 'Lomo saltado',
    descripcion: '',
    tipo: 'PREPARADO',
    precio_venta: '25.50',
    factor_consumo: '1',
    afectacion_igv: '10',
    impuesto_id: '7',
    imagen_url: '',
    ...over,
  });

  it('Nuevo producto: campos, sin REVENTA, categorías activas, impuesto único preseleccionado', () => {
    // El alta se abre desde el estado vacío para no depender del encabezado.
    const { boton, ultimoDialogo } = setup({ productos: [] });
    boton('+ Nuevo producto')!.click();
    const d = ultimoDialogo();
    expect(d.title).toBe('Nuevo producto');
    const claves = d.fields.map((f) => f.key);
    expect(claves).toEqual([
      'categoria_id',
      'codigo',
      'nombre',
      'descripcion',
      'tipo',
      'precio_venta',
      'factor_consumo',
      'afectacion_igv',
      'impuesto_id',
      'imagen_url',
    ]);
    const tipo = d.fields.find((f) => f.key === 'tipo')!;
    expect(tipo.options!.map((o) => o.value)).toEqual(['PREPARADO', 'ADICIONAL', 'SERVICIO']);
    expect(d.fields.find((f) => f.key === 'categoria_id')!.options!.map((o) => o.label)).toEqual([
      'Fondos',
      'Postres',
    ]);
    expect(d.fields.find((f) => f.key === 'factor_consumo')!.value).toBe('1');
    expect(d.fields.find((f) => f.key === 'impuesto_id')!.value).toBe('7');
    expect(d.fields.find((f) => f.key === 'afectacion_igv')!.options!.map((o) => o.value)).toEqual([
      '10',
      '20',
      '30',
    ]);
  });

  it('el alta pone el código en mayúsculas y llama al store con una clave', async () => {
    const { boton, store, ultimoDialogo } = setup({ productos: [] });
    boton('+ Nuevo producto')!.click();
    const r = await ultimoDialogo().commit(valores());
    expect(r).toBeUndefined();
    expect(store.crearProducto).toHaveBeenCalledWith(
      expect.objectContaining({ codigo: 'LOMO', precio_venta: '25.50' }),
      expect.any(String),
    );
  });

  it('afectación gravada exige impuesto y exonerada lo rechaza', async () => {
    const { boton, store, ultimoDialogo } = setup({ productos: [] });
    boton('+ Nuevo producto')!.click();
    const d = ultimoDialogo();
    expect(await d.commit(valores({ impuesto_id: '' }))).toEqual({
      field: 'impuesto_id',
      message: expect.stringContaining('obligatorio'),
    });
    expect(await d.commit(valores({ afectacion_igv: '20', impuesto_id: '7' }))).toEqual({
      field: 'impuesto_id',
      message: expect.stringContaining('no aplica'),
    });
    expect(store.crearProducto).not.toHaveBeenCalled();
    await d.commit(valores({ afectacion_igv: '20', impuesto_id: '' }));
    expect(store.crearProducto).toHaveBeenCalledTimes(1);
  });

  it('precio y factor inválidos se devuelven junto a su campo sin llamar al servidor', async () => {
    const { boton, store, ultimoDialogo } = setup({ productos: [] });
    boton('+ Nuevo producto')!.click();
    const d = ultimoDialogo();
    expect(await d.commit(valores({ precio_venta: '-3' }))).toMatchObject({
      field: 'precio_venta',
    });
    expect(await d.commit(valores({ precio_venta: '1.234' }))).toMatchObject({
      field: 'precio_venta',
    });
    expect(await d.commit(valores({ factor_consumo: '0' }))).toMatchObject({
      field: 'factor_consumo',
    });
    expect(store.crearProducto).not.toHaveBeenCalled();
  });

  it('el rechazo del servidor se propaga al diálogo', async () => {
    const { boton, store, ultimoDialogo } = setup({ productos: [] });
    store.crearProducto.mockResolvedValue({ field: 'codigo', message: 'Código repetido' });
    boton('+ Nuevo producto')!.click();
    expect(await ultimoDialogo().commit(valores())).toEqual({
      field: 'codigo',
      message: 'Código repetido',
    });
  });

  it('Editar: no incluye el código, precarga y conserva el código original', async () => {
    const p = producto('1', { nombre: 'Lomo', factor_consumo: '1.500000' });
    const { boton, store, ultimoDialogo } = setup({ productos: [p] });
    boton('Editar producto Lomo')!.click();
    const d = ultimoDialogo();
    expect(d.fields.map((f) => f.key)).not.toContain('codigo');
    expect(d.fields.find((f) => f.key === 'factor_consumo')!.value).toBe('1.5');
    expect(d.fields.find((f) => f.key === 'impuesto_id')!.value).toBe('7');
    await d.commit(valores());
    expect(store.editarProducto).toHaveBeenCalledWith(
      p,
      expect.objectContaining({ codigo: 'LOMO1' }),
    );
  });

  it('Editar un producto REVENTA no ofrece cambiar el tipo y lo conserva', async () => {
    const p = producto('1', { nombre: 'Gaseosa', tipo: 'REVENTA' });
    const { boton, store, ultimoDialogo } = setup({ productos: [p] });
    boton('Editar producto Gaseosa')!.click();
    const d = ultimoDialogo();
    expect(d.fields.map((f) => f.key)).not.toContain('tipo');
    await d.commit(valores({ tipo: '' }));
    expect(store.editarProducto).toHaveBeenCalledWith(
      p,
      expect.objectContaining({ tipo: 'REVENTA' }),
    );
  });

  it('Dar de baja: motivo obligatorio, explica el efecto y es peligroso', async () => {
    const p = producto('1', { nombre: 'Lomo' });
    const { boton, store, ultimoDialogo } = setup({ productos: [p] });
    boton('Dar de baja producto Lomo')!.click();
    const d = ultimoDialogo();
    expect(d.danger).toBe(true);
    expect(d.description).toContain('todas las sedes');
    expect(d.description).toContain('consumos ya registrados');
    const motivo = d.fields.find((f) => f.key === 'motivo')!;
    expect(motivo.required).not.toBe(false);
    expect(await d.commit({ motivo: 'x'.repeat(2001) })).toMatchObject({ field: 'motivo' });
    expect(store.darDeBajaProducto).not.toHaveBeenCalled();
    await d.commit({ motivo: 'Descontinuado' });
    expect(store.darDeBajaProducto).toHaveBeenCalledWith(p, 'Descontinuado', expect.any(String));
  });

  it('Ajustes de sede: muestra la sede, ofrece las estaciones y valida el precio', async () => {
    const p = producto('1', {
      nombre: 'Lomo',
      en_sucursal: {
        disponible: true,
        precio_sucursal: '30.00',
        precio_efectivo: '30.00',
        estacion_id: '5',
      },
    });
    const { boton, store, ultimoDialogo } = setup({ filtros: { sucursalId: '1' }, productos: [p] });
    boton('Ajustes de sede del producto Lomo')!.click();
    const d = ultimoDialogo();
    expect(d.title).toContain('Lomo');
    expect(d.description).toContain('Sede 1');
    const precio = d.fields.find((f) => f.key === 'precio')!;
    expect(precio.required).toBe(false);
    expect(precio.value).toBe('30.00');
    const est = d.fields.find((f) => f.key === 'estacionId')!;
    expect(est.options!.map((o) => o.label)).toEqual(['Sin estación', 'Cocina caliente']);
    expect(est.value).toBe('5');
    expect(await d.commit({ precio: 'abc', estacionId: '' })).toMatchObject({ field: 'precio' });
    await d.commit({ precio: '', estacionId: '' });
    expect(store.ajustarSede).toHaveBeenCalledWith(p, { precio: '', estacionId: '' });
  });
});
