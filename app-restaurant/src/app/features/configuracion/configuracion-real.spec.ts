import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ConfiguracionReal } from './configuracion-real';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { NegocioSection } from './negocio-section';
import { SalonSection } from './salon-section';
import { SedesSection } from './sedes-section';

function setup(over: Record<string, unknown> = {}) {
  const store = {
    status: signal<'loading' | 'ready'>('ready'),
    notice: signal<{ kind: 'ok' | 'error'; text: string } | null>(null),
    sinAcceso: signal(false),
    puedeVerNegocio: signal(true),
    puedeVerSedes: signal(true),
    puedeVerSalon: signal(true),
    negocio: signal(null), // NegocioSection lo lee en su constructor aunque su plantilla esté vacía
    load: vi.fn(),
    dismissNotice: vi.fn(),
    ...over,
  };
  TestBed.configureTestingModule({ imports: [ConfiguracionReal] });
  // Las secciones se sustituyen por plantillas vacías: aquí solo se prueba qué secciones aparecen.
  for (const seccion of [NegocioSection, SedesSection, SalonSection]) {
    TestBed.overrideComponent(seccion, { set: { template: '', imports: [], providers: [] } });
  }
  TestBed.overrideComponent(ConfiguracionReal, {
    set: { providers: [{ provide: ConfiguracionStore, useValue: store }] },
  });
  const fixture = TestBed.createComponent(ConfiguracionReal);
  fixture.detectChanges();
  return { store, el: fixture.nativeElement as HTMLElement };
}

describe('Pantalla real de Configuración', () => {
  it('carga al iniciar y muestra las tres secciones con todos los permisos', () => {
    const { store, el } = setup();
    expect(store.load).toHaveBeenCalled();
    expect(el.querySelector('app-negocio-section')).not.toBeNull();
    expect(el.querySelector('app-sedes-section')).not.toBeNull();
    expect(el.querySelector('app-salon-section')).not.toBeNull();
  });

  it('oculta las secciones sin permiso de lectura', () => {
    const { el } = setup({ puedeVerNegocio: signal(false), puedeVerSedes: signal(false) });
    expect(el.querySelector('app-negocio-section')).toBeNull();
    expect(el.querySelector('app-sedes-section')).toBeNull();
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
