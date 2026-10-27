import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs, type FormDialogData } from '../../shared/ui/form-dialog';
import type { Impuesto, TasaImpuesto } from '../catalogos/data-access/carta.models';
import { TasasStore } from './data-access/tasas-store';
import { TasasSection } from './tasas-section';

const igv: Impuesto = {
  id: '1',
  codigo: 'IGV',
  nombre: 'IGV',
  descripcion: 'Impuesto general a las ventas',
  es_activo: true,
};
const icbper: Impuesto = {
  id: '2',
  codigo: 'ICBPER',
  nombre: 'ICBPER',
  descripcion: null,
  es_activo: true,
};
const viejo: Impuesto = {
  id: '3',
  codigo: 'OLD',
  nombre: 'Antiguo',
  descripcion: null,
  es_activo: false,
};

const tasa = (id: string, over: Partial<TasaImpuesto> = {}): TasaImpuesto => ({
  id,
  impuesto_id: '1',
  tasa: '18.0000',
  vigente_desde: '2026-01-01',
  vigente_hasta: null,
  fundamento: null,
  ...over,
});

const vigenteAhora = tasa('b', { fundamento: 'Ley 123' });
const pasada = tasa('a', {
  tasa: '17.5000',
  vigente_desde: '2020-01-01',
  vigente_hasta: '2026-01-01',
});

function setup(
  o: {
    administrar?: boolean;
    status?: 'loading' | 'ready' | 'error';
    errores?: Record<string, string>;
    hoy?: string;
  } = {},
) {
  const store = {
    status: signal(o.status ?? 'ready'),
    error: signal('Falló.'),
    notice: signal<{ kind: 'ok' | 'error'; text: string } | null>(null),
    puedeAdministrar: signal(o.administrar ?? true),
    impuestos: signal([igv, icbper, viejo]),
    tasas: signal<Record<string, TasaImpuesto[]>>({
      '1': [vigenteAhora, pasada],
      '2': [],
      '3': [],
    }),
    errores: signal(o.errores ?? {}),
    hoy: signal(o.hoy ?? '2026-10-25'),
    cargar: vi.fn(),
    reintentar: vi.fn(),
    crear: vi.fn(),
    editar: vi.fn(),
    dismissNotice: vi.fn(),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [TasasSection],
    providers: [{ provide: Dialogs, useValue: dialogs }],
  });
  TestBed.overrideComponent(TasasSection, {
    set: { providers: [{ provide: TasasStore, useValue: store }] },
  });
  const fixture = TestBed.createComponent(TasasSection);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const boton = (texto: string) =>
    [...el.querySelectorAll('button')].find(
      (b) => b.getAttribute('aria-label') === texto || b.textContent?.trim() === texto,
    );
  const ultimo = () => dialogs.form.mock.calls.at(-1)![0] as FormDialogData;
  return { store, dialogs, fixture, el, boton, ultimo };
}

describe('Sección Impuestos y tasas', () => {
  it('carga al iniciar', () => {
    expect(setup().store.cargar).toHaveBeenCalled();
  });

  it('muestra cada impuesto con su historial, tasa formateada y vigencia', () => {
    const { el } = setup();
    expect(el.textContent).toContain('Impuesto general a las ventas');
    expect(el.textContent).toContain('18 %');
    expect(el.textContent).toContain('17.5 %');
    expect(el.textContent).toContain('desde 01/01/2026 (vigente)');
    expect(el.textContent).toContain('desde 01/01/2020 – hasta 01/01/2026');
    expect(el.textContent).toContain('Fundamento: Ley 123');
  });

  it('marca «Vigente hoy» solo la tasa cuya vigencia [desde, hasta) cubre hoy', () => {
    const { el } = setup();
    const chips = [...el.querySelectorAll('.pill.green')].map((c) => c.textContent?.trim());
    expect(chips).toEqual(['Vigente hoy']);
    const fila = el.querySelector('.pill.green')!.closest('.tasa-fila')!;
    expect(fila.textContent).toContain('18 %');
  });

  it('el día de fin la tasa anterior ya no está vigente (límite exclusivo)', () => {
    const { el } = setup({ hoy: '2026-01-01' });
    const fila = el.querySelector('.pill.green')!.closest('.tasa-fila')!;
    expect(fila.textContent).toContain('18 %');
    expect(el.querySelectorAll('.pill.green')).toHaveLength(1);
  });

  it('un impuesto sin tasas explica que no hay tasa por defecto', () => {
    const { el } = setup();
    expect(el.textContent).toContain('Aún no hay tasa configurada');
    expect(el.textContent).toContain('no se podrán vender productos con este impuesto');
  });

  it('con IMPUESTO_ADMINISTRAR: nueva tasa por impuesto activo y editar con aria-label por fila', () => {
    const { boton } = setup();
    expect(boton('Nueva tasa de IGV')).toBeDefined();
    expect(boton('Nueva tasa de ICBPER')).toBeDefined();
    expect(boton('Nueva tasa de Antiguo')).toBeUndefined();
    expect(boton('Editar tasa 18 % de IGV desde 01/01/2026')).toBeDefined();
    expect(boton('Editar tasa 17.5 % de IGV desde 01/01/2020')).toBeDefined();
  });

  it('sin IMPUESTO_ADMINISTRAR es de solo lectura', () => {
    const { el } = setup({ administrar: false });
    expect([...el.querySelectorAll('button')]).toEqual([]);
    expect(el.textContent).toContain('18 %');
  });

  it('alta: impuesto preseleccionado, campos y una clave por diálogo', async () => {
    const { boton, ultimo, store } = setup();
    boton('Nueva tasa de ICBPER')!.click();
    const d = ultimo();
    expect(d.title).toContain('ICBPER');
    expect(d.fields.map((f) => f.key)).toEqual([
      'tasa',
      'vigente_desde',
      'vigente_hasta',
      'fundamento',
    ]);
    expect(d.fields.find((f) => f.key === 'vigente_hasta')?.required).toBe(false);
    const v = { tasa: '5', vigente_desde: '2026-01-01', vigente_hasta: '', fundamento: '' };
    await d.commit(v);
    await d.commit(v);
    expect(store.crear).toHaveBeenCalledWith(icbper, v, expect.any(String));
    expect(store.crear.mock.calls[0][2]).toBe(store.crear.mock.calls[1][2]);
  });

  it('edición: parte de los valores actuales y no recibe clave ni impuesto', async () => {
    const { boton, ultimo, store } = setup();
    boton('Editar tasa 18 % de IGV desde 01/01/2026')!.click();
    const d = ultimo();
    expect(d.fields.map((f) => f.value)).toEqual(['18.0000', '2026-01-01', '', 'Ley 123']);
    const v = { tasa: '19', vigente_desde: '2026-01-01', vigente_hasta: '', fundamento: 'Ley 123' };
    await d.commit(v);
    expect(store.editar).toHaveBeenCalledWith(vigenteAhora, v);
  });

  it('estados de carga y error general', () => {
    expect(setup({ status: 'loading' }).el.textContent).toContain('Cargando los impuestos');
    TestBed.resetTestingModule();
    const { el, boton, store } = setup({ status: 'error' });
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Falló.');
    boton('Reintentar')!.click();
    expect(store.cargar).toHaveBeenCalledTimes(2);
  });

  it('carga parcial: el impuesto con error ofrece reintentar solo el suyo', () => {
    const { el, store } = setup({ errores: { '2': 'Sin conexión.' } });
    expect(el.textContent).toContain('No pudimos cargar las tasas de ICBPER');
    expect(el.textContent).toContain('18 %');
    const reintentar = [...el.querySelectorAll('button')].find(
      (b) => b.textContent?.trim() === 'Reintentar',
    )!;
    reintentar.click();
    expect(store.reintentar).toHaveBeenCalledWith('2');
  });

  it('muestra el aviso del store y permite cerrarlo', () => {
    const { store, fixture, el } = setup();
    store.notice.set({ kind: 'error', text: 'Algo pasó.' });
    fixture.detectChanges();
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Algo pasó.');
    (el.querySelector('.note-box button') as HTMLButtonElement).click();
    expect(store.dismissNotice).toHaveBeenCalled();
  });
});
