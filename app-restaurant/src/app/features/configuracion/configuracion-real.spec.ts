import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ConfiguracionReal } from './configuracion-real';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { EstacionesSection } from './estaciones-section';
import { EstacionesStore } from './data-access/estaciones-store';
import { TasasStore } from './data-access/tasas-store';
import { NegocioSection } from './negocio-section';
import { SalonSection } from './salon-section';
import { SedesSection } from './sedes-section';
import { TasasSection } from './tasas-section';

function setup(over: Record<string, unknown> = {}) {
  const store = {
    status: signal<'loading' | 'ready'>('ready'),
    notice: signal<{ kind: 'ok' | 'error'; text: string } | null>(null),
    sinAcceso: signal(false),
    puedeVerNegocio: signal(true),
    puedeVerSedes: signal(true),
    puedeVerSalon: signal(true),
    puedeVerEstaciones: signal(true),
    puedeVerTasas: signal(true),
    sedeActual: signal(null), // EstacionesSection lo lee en su constructor
    negocio: signal(null), // NegocioSection lo lee en su constructor aunque su plantilla esté vacía
    load: vi.fn(),
    dismissNotice: vi.fn(),
    ...over,
  };
  TestBed.configureTestingModule({ imports: [ConfiguracionReal] });
  // Las secciones se sustituyen por plantillas vacías: aquí solo se prueba qué secciones aparecen.
  for (const seccion of [
    NegocioSection,
    SedesSection,
    SalonSection,
    EstacionesSection,
    TasasSection,
  ]) {
    TestBed.overrideComponent(seccion, { set: { template: '', imports: [], providers: [] } });
  }
  TestBed.overrideComponent(ConfiguracionReal, {
    set: {
      providers: [
        { provide: ConfiguracionStore, useValue: store },
        { provide: EstacionesStore, useValue: { cargar: vi.fn() } },
        { provide: TasasStore, useValue: { cargar: vi.fn() } },
      ],
    },
  });
  const fixture = TestBed.createComponent(ConfiguracionReal);
  fixture.detectChanges();
  return { store, el: fixture.nativeElement as HTMLElement };
}

describe('Pantalla real de Configuración', () => {
  it('carga al iniciar y muestra las cinco secciones con todos los permisos', () => {
    const { store, el } = setup();
    expect(store.load).toHaveBeenCalled();
    expect(el.querySelector('app-negocio-section')).not.toBeNull();
    expect(el.querySelector('app-sedes-section')).not.toBeNull();
    expect(el.querySelector('app-salon-section')).not.toBeNull();
    expect(el.querySelector('app-estaciones-section')).not.toBeNull();
    expect(el.querySelector('app-tasas-section')).not.toBeNull();
  });

  it('oculta las secciones sin permiso de lectura', () => {
    const { el } = setup({ puedeVerNegocio: signal(false), puedeVerSedes: signal(false) });
    expect(el.querySelector('app-negocio-section')).toBeNull();
    expect(el.querySelector('app-sedes-section')).toBeNull();
    expect(el.querySelector('app-salon-section')).not.toBeNull();
  });

  it('con permisos solo de estaciones y tasas muestra esas secciones y no el aviso de sin acceso', () => {
    const { el } = setup({
      puedeVerNegocio: signal(false),
      puedeVerSedes: signal(false),
      puedeVerSalon: signal(false),
    });
    expect(el.textContent).not.toContain('No tienes permiso');
    expect(el.querySelector('app-estaciones-section')).not.toBeNull();
    expect(el.querySelector('app-tasas-section')).not.toBeNull();
    expect(el.querySelector('app-salon-section')).toBeNull();
  });

  it('oculta estaciones y tasas sin su permiso de lectura', () => {
    const { el } = setup({ puedeVerEstaciones: signal(false), puedeVerTasas: signal(false) });
    expect(el.querySelector('app-estaciones-section')).toBeNull();
    expect(el.querySelector('app-tasas-section')).toBeNull();
    expect(el.querySelector('app-salon-section')).not.toBeNull();
  });

  it('sin ningún permiso muestra el aviso y ninguna sección', () => {
    const { el } = setup({ sinAcceso: signal(true) });
    expect(el.textContent).toContain('No tienes permiso');
    expect(el.querySelector('app-salon-section')).toBeNull();
  });

  it('muestra el aviso del store y permite cerrarlo', () => {
    const { el, store } = setup({ notice: signal({ kind: 'ok', text: 'Se creó la sede X.' }) });
    expect(el.textContent).toContain('Se creó la sede X.');
    (el.querySelector('.note-box button') as HTMLButtonElement).click();
    expect(store.dismissNotice).toHaveBeenCalled();
  });
});
