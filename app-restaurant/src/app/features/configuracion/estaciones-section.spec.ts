import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs, type FormDialogData } from '../../shared/ui/form-dialog';
import type { Estacion } from '../catalogos/data-access/carta.models';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { EstacionesStore } from './data-access/estaciones-store';
import { EstacionesSection } from './estaciones-section';

const estacion = (id: string, over: Partial<Estacion> = {}): Estacion => ({
  id,
  sucursal_id: '1',
  codigo: `E${id}`,
  nombre: `Estación ${id}`,
  es_activo: true,
  ...over,
});

interface Opciones {
  administrar?: boolean;
  estaciones?: Estacion[];
  status?: 'loading' | 'ready' | 'error';
  sede?: string | null;
  sedes?: { id: string; nombre: string }[];
}

function setup(o: Opciones = {}) {
  const lista = o.estaciones ?? [estacion('1'), estacion('2', { es_activo: false })];
  const sede = signal<string | null>(o.sede === undefined ? '1' : o.sede);
  const config = {
    sedesSalon: signal(o.sedes ?? [{ id: '1', nombre: 'Principal' }]),
    sedeActual: sede,
    elegirSede: vi.fn(),
  };
  const store = {
    status: signal(o.status ?? 'ready'),
    error: signal('Falló.'),
    notice: signal<{ kind: 'ok' | 'error'; text: string } | null>(null),
    busyId: signal<string | null>(null),
    puedeAdministrar: signal(o.administrar ?? true),
    ordenadas: signal(lista),
    cargar: vi.fn(),
    crear: vi.fn(),
    editar: vi.fn(),
    baja: vi.fn(),
    restaurar: vi.fn(),
    dismissNotice: vi.fn(),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [EstacionesSection],
    providers: [
      { provide: ConfiguracionStore, useValue: config },
      { provide: Dialogs, useValue: dialogs },
    ],
  });
  TestBed.overrideComponent(EstacionesSection, {
    set: { providers: [{ provide: EstacionesStore, useValue: store }] },
  });
  const fixture = TestBed.createComponent(EstacionesSection);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const boton = (texto: string) =>
    [...el.querySelectorAll('button')].find(
      (b) => b.getAttribute('aria-label') === texto || b.textContent?.trim() === texto,
    );
  const ultimo = () => dialogs.form.mock.calls.at(-1)![0] as FormDialogData;
  return { config, store, dialogs, fixture, el, boton, ultimo, sede };
}

describe('Sección Estaciones de preparación', () => {
  it('carga las estaciones de la sede actual y recarga si cambia', async () => {
    const { store, sede, fixture } = setup();
    expect(store.cargar).toHaveBeenCalledWith('1');
    sede.set('2');
    fixture.detectChanges();
    await fixture.whenStable();
    expect(store.cargar).toHaveBeenLastCalledWith('2');
  });

  it('muestra código, nombre y estado; las bajas solo con «Incluir bajas»', () => {
    const { el, fixture } = setup();
    expect(el.textContent).toContain('E1');
    expect(el.textContent).toContain('Estación 1');
    expect(el.querySelector('.pill.green')?.textContent).toContain('Activa');
    expect(el.textContent).not.toContain('Estación 2');
    const incluir = el.querySelector('input[type="checkbox"]') as HTMLInputElement;
    incluir.checked = true;
    incluir.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    expect(el.textContent).toContain('Estación 2');
    expect(el.textContent).toContain('De baja');
  });

  it('con permiso de administrar: botones con aria-label por fila; restaurar en las bajas', () => {
    const { el, boton, fixture } = setup();
    expect(boton('+ Nueva estación')).toBeDefined();
    expect(boton('Editar estación Estación 1')).toBeDefined();
    expect(boton('Dar de baja estación Estación 1')).toBeDefined();
    const incluir = el.querySelector('input[type="checkbox"]') as HTMLInputElement;
    incluir.checked = true;
    incluir.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    expect(boton('Restaurar estación Estación 2')).toBeDefined();
  });

  it('sin ESTACION_ADMINISTRAR es de solo lectura: ni botones ni columna de acciones', () => {
    const { el, boton } = setup({ administrar: false });
    expect(boton('+ Nueva estación')).toBeUndefined();
    expect(boton('Editar estación Estación 1')).toBeUndefined();
    expect(el.textContent).not.toContain('Acciones');
    expect(el.textContent).toContain('Estación 1');
  });

  it('estado vacío, de carga y de error con Reintentar', () => {
    expect(setup({ estaciones: [] }).el.textContent).toContain('Aún no hay estaciones');
    TestBed.resetTestingModule();
    expect(setup({ status: 'loading' }).el.textContent).toContain('Cargando las estaciones');
    TestBed.resetTestingModule();
    const { el, store, boton } = setup({ status: 'error' });
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Falló.');
    boton('Reintentar')!.click();
    expect(store.cargar).toHaveBeenLastCalledWith('1');
  });

  it('sin sede explica por qué y no ofrece crear', () => {
    const { el, boton, store } = setup({ sede: null, sedes: [] });
    expect(el.textContent).toContain('No hay una sede disponible');
    expect(boton('+ Nueva estación')).toBeUndefined();
    expect(store.cargar).toHaveBeenCalledWith(null);
  });

  it('el selector de sede solo aparece con varias sedes y usa la sede de Salón', () => {
    expect(setup().el.querySelector('select')).toBeNull();
    TestBed.resetTestingModule();
    const { el, config } = setup({
      sedes: [
        { id: '1', nombre: 'Principal' },
        { id: '2', nombre: 'Norte' },
      ],
    });
    const select = el.querySelector(
      'select[aria-label="Sede de las estaciones"]',
    ) as HTMLSelectElement;
    select.value = '2';
    select.dispatchEvent(new Event('change'));
    expect(config.elegirSede).toHaveBeenCalledWith('2');
  });

  it('alta: diálogo con código y nombre, una clave por diálogo', async () => {
    const { boton, ultimo, store } = setup();
    boton('+ Nueva estación')!.click();
    const d = ultimo();
    expect(d.fields.map((f) => f.key)).toEqual(['codigo', 'nombre']);
    await d.commit({ codigo: 'coc', nombre: 'Cocina' });
    await d.commit({ codigo: 'coc', nombre: 'Cocina' });
    const claves = store.crear.mock.calls.map((c) => c[1]);
    expect(store.crear).toHaveBeenCalledWith(
      { codigo: 'coc', nombre: 'Cocina' },
      expect.any(String),
    );
    expect(claves[0]).toBe(claves[1]);
  });

  it('edición: solo el nombre, el código no es un campo', async () => {
    const { boton, ultimo, store } = setup();
    const e = estacion('1');
    boton('Editar estación Estación 1')!.click();
    const d = ultimo();
    expect(d.fields.map((f) => f.key)).toEqual(['nombre']);
    expect(d.fields[0].value).toBe('Estación 1');
    await d.commit({ nombre: 'Parrilla' });
    expect(store.editar).toHaveBeenCalledWith(e, { nombre: 'Parrilla' });
  });

  it('baja: confirmación sin motivo y con la clave del diálogo', async () => {
    const { boton, ultimo, store } = setup();
    boton('Dar de baja estación Estación 1')!.click();
    const d = ultimo();
    expect(d.danger).toBe(true);
    expect(d.fields).toEqual([]);
    await d.commit({});
    expect(store.baja).toHaveBeenCalledWith(estacion('1'), expect.any(String));
  });

  it('restaurar llama al store con una clave nueva', () => {
    const { boton, store, el, fixture } = setup();
    const incluir = el.querySelector('input[type="checkbox"]') as HTMLInputElement;
    incluir.checked = true;
    incluir.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    boton('Restaurar estación Estación 2')!.click();
    expect(store.restaurar).toHaveBeenCalledWith(
      estacion('2', { es_activo: false }),
      expect.any(String),
    );
  });

  it('muestra el aviso del store y permite cerrarlo', () => {
    const { store, fixture, el } = setup();
    store.notice.set({ kind: 'ok', text: 'Se creó la estación X.' });
    fixture.detectChanges();
    expect(el.textContent).toContain('Se creó la estación X.');
    (el.querySelector('.note-box button') as HTMLButtonElement).click();
    expect(store.dismissNotice).toHaveBeenCalled();
  });
});
