import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs } from '../../shared/ui/form-dialog';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { SalonSection } from './salon-section';
import type { Mesa, Zona } from './configuracion.models';

const audit = { creado_en: '', creado_por: '1', actualizado_en: '', actualizado_por: '1' };
const zona = (id: string, activa = true): Zona => ({
  id,
  sucursal_id: '1',
  nombre: 'Zona ' + id,
  orden_visualizacion: 0,
  es_activo: activa,
  ...audit,
});
const mesa = (id: string, over: Partial<Mesa> = {}): Mesa => ({
  id,
  sucursal_id: '1',
  zona_id: '10',
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

function setup(
  editar: boolean,
  sedes = [{ id: '1', nombre: 'Principal' }],
  errors: Record<string, string> = {},
) {
  const zonas = [zona('10'), zona('11', false)];
  const store = {
    errors: signal(errors),
    busyId: signal<string | null>(null),
    puedeEditarSalon: signal(editar),
    sedesSalon: signal(sedes),
    sedeActual: signal<string | null>('1'),
    zonasSede: signal(zonas),
    zonasActivasSede: signal([zonas[0]]),
    mesasDeZona: (id: string) =>
      id === '10' ? [mesa('1'), mesa('2', { es_activo: false, motivo_eliminacion: 'Rota' })] : [],
    elegirSede: vi.fn(),
    habilitarMesa: vi.fn(),
    crearZona: vi.fn(),
    editarZona: vi.fn(),
    bajaZona: vi.fn(),
    restaurarZona: vi.fn(),
    crearMesa: vi.fn(),
    editarMesa: vi.fn(),
    bajaMesa: vi.fn(),
    restaurarMesa: vi.fn(),
    load: vi.fn(),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [SalonSection],
    providers: [
      { provide: ConfiguracionStore, useValue: store },
      { provide: Dialogs, useValue: dialogs },
    ],
  });
  const fixture = TestBed.createComponent(SalonSection);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const boton = (texto: string) =>
    [...el.querySelectorAll('button')].find(
      (b) => b.getAttribute('aria-label') === texto || b.textContent?.trim() === texto,
    );
  return { store, dialogs, el, boton };
}

describe('Sección Zonas y mesas', () => {
  it('agrupa las mesas bajo su zona', () => {
    const { el } = setup(true);
    expect(el.textContent).toContain('Zona 10');
    expect(el.textContent).toContain('Mesa 1');
    expect(el.textContent).toContain('Motivo de la baja: Rota');
  });

  it('con una sola sede no muestra el selector; con varias sí', () => {
    expect(setup(true).el.querySelector('select[aria-label="Sede"]')).toBeNull();
    TestBed.resetTestingModule();
    const varias = setup(true, [
      { id: '1', nombre: 'Principal' },
      { id: '2', nombre: 'Norte' },
    ]);
    expect(varias.el.querySelector('select[aria-label="Sede"]')).not.toBeNull();
  });

  it('sin permiso de escritura no hay botones de acción ni interruptor habilitado', () => {
    const { el } = setup(false);
    expect(
      [...el.querySelectorAll('button')].filter((b) => !b.classList.contains('toggle')),
    ).toHaveLength(0);
    expect((el.querySelector('button.toggle') as HTMLButtonElement).disabled).toBe(true);
  });

  it('el interruptor de habilitación llama al store sin diálogo', () => {
    const { el, store, dialogs } = setup(true);
    (el.querySelector('button.toggle') as HTMLButtonElement).click();
    expect(store.habilitarMesa).toHaveBeenCalledWith(expect.objectContaining({ id: '1' }));
    expect(dialogs.form).not.toHaveBeenCalled();
  });

  it('dar de baja una mesa exige motivo; dar de baja una zona no tiene campos', () => {
    const { boton, dialogs } = setup(true);
    boton('Dar de baja mesa 1')?.click();
    expect(dialogs.form.mock.calls[0][0].fields.map((f: { key: string }) => f.key)).toEqual([
      'motivo',
    ]);
    boton('Dar de baja zona Zona 10')?.click();
    expect(dialogs.form.mock.calls[1][0].fields).toEqual([]);
  });

  it('nueva mesa ofrece solo zonas activas y los campos del spec', () => {
    const { boton, dialogs } = setup(true);
    boton('Nueva mesa')?.click();
    const campos = dialogs.form.mock.calls[0][0].fields;
    expect(campos.map((f: { key: string }) => f.key)).toEqual([
      'zona_id',
      'numero_mesa',
      'capacidad',
      'posicion_x',
      'posicion_y',
    ]);
    expect(campos[0].options).toEqual([{ value: '10', label: 'Zona 10' }]);
  });

  it('restaurar zona o mesa no abre diálogo', () => {
    const { boton, dialogs, store } = setup(true);
    boton('Restaurar zona Zona 11')?.click();
    boton('Restaurar mesa 2')?.click();
    expect(dialogs.form).not.toHaveBeenCalled();
    expect(store.restaurarZona).toHaveBeenCalledWith(
      expect.objectContaining({ id: '11' }),
      expect.any(String),
    );
    expect(store.restaurarMesa).toHaveBeenCalledWith(
      expect.objectContaining({ id: '2' }),
      expect.any(String),
    );
  });

  it('si la carga falla muestra el mensaje y Reintentar vuelve a cargar', () => {
    const { el, boton, store } = setup(true, undefined, { salon: 'No tienes permiso.' });
    expect(el.textContent).toContain('No tienes permiso.');
    expect(el.textContent).not.toContain('Zona 10');
    boton('Reintentar')?.click();
    expect(store.load).toHaveBeenCalled();
  });

  it('nueva zona: la clave de idempotencia es estable entre reintentos', () => {
    const { boton, dialogs, store } = setup(true);
    boton('Nueva zona')?.click();
    const commit = dialogs.form.mock.calls[0][0].commit;
    const valores = { nombre: ' Terraza ', orden_visualizacion: '' };
    commit(valores);
    commit(valores);
    const [cuerpo, key] = store.crearZona.mock.calls[0];
    expect(cuerpo).toEqual({ sucursal_id: '1', nombre: 'Terraza', orden_visualizacion: 0 });
    expect(store.crearZona.mock.calls[1][1]).toBe(key);
  });

  it('nueva mesa: arma el cuerpo, omite posiciones vacías y reutiliza la clave', () => {
    const { boton, dialogs, store } = setup(true);
    boton('Nueva mesa')?.click();
    const commit = dialogs.form.mock.calls[0][0].commit;
    commit({
      zona_id: '10',
      numero_mesa: ' 7 ',
      capacidad: '6',
      posicion_x: '12.50',
      posicion_y: '',
    });
    commit({ zona_id: '10', numero_mesa: '7', capacidad: '6', posicion_x: '', posicion_y: '' });
    const [cuerpo, key] = store.crearMesa.mock.calls[0];
    expect(cuerpo).toEqual({
      sucursal_id: '1',
      zona_id: '10',
      numero_mesa: '7',
      capacidad: 6,
      posicion_x: '12.50',
    });
    expect(store.crearMesa.mock.calls[1][0]).not.toHaveProperty('posicion_x');
    expect(store.crearMesa.mock.calls[1][1]).toBe(key);
  });

  it('editar zona y mesa envían los valores normalizados sin clave', () => {
    const { boton, dialogs, store } = setup(true);
    boton('Editar zona Zona 10')?.click();
    dialogs.form.mock.calls[0][0].commit({ nombre: ' Salon ', orden_visualizacion: '3' });
    expect(store.editarZona).toHaveBeenCalledWith(expect.objectContaining({ id: '10' }), {
      nombre: 'Salon',
      orden_visualizacion: 3,
    });
    boton('Editar mesa 1')?.click();
    dialogs.form.mock.calls[1][0].commit({
      zona_id: '10',
      numero_mesa: '1',
      capacidad: '5',
      posicion_x: '',
      posicion_y: '8.00',
    });
    expect(store.editarMesa).toHaveBeenCalledWith(expect.objectContaining({ id: '1' }), {
      zona_id: '10',
      numero_mesa: '1',
      capacidad: 5,
      posicion_x: null,
      posicion_y: '8.00',
    });
  });

  it('baja de zona y de mesa reutilizan su clave entre reintentos', () => {
    const { boton, dialogs, store } = setup(true);
    boton('Dar de baja zona Zona 10')?.click();
    const bajaZona = dialogs.form.mock.calls[0][0].commit;
    bajaZona({});
    bajaZona({});
    expect(store.bajaZona.mock.calls[1][1]).toBe(store.bajaZona.mock.calls[0][1]);
    boton('Dar de baja mesa 1')?.click();
    const bajaMesa = dialogs.form.mock.calls[1][0].commit;
    bajaMesa({ motivo: ' Rota ' });
    bajaMesa({ motivo: ' Rota ' });
    expect(store.bajaMesa.mock.calls[0][1]).toBe('Rota');
    expect(store.bajaMesa.mock.calls[1][2]).toBe(store.bajaMesa.mock.calls[0][2]);
  });
});
