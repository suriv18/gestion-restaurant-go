import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { CartaReal } from './carta-real';
import { CategoriasSection } from './categorias-section';
import { CartaStore } from './data-access/carta-store';
import { ProductosSection } from './productos-section';

function setup(over: Record<string, unknown> = {}) {
  const store = {
    status: signal<'loading' | 'ready' | 'error'>('ready'),
    loadError: signal('Sin conexión'),
    notice: signal<{ kind: 'ok' | 'error'; text: string } | null>(null),
    sinAcceso: signal(false),
    puedeAdministrarCarta: signal(true),
    filtros: signal({ q: '' }), // ProductosSection lo lee en su constructor aunque su plantilla esté vacía
    load: vi.fn(),
    dismissNotice: vi.fn(),
    ...over,
  };
  TestBed.configureTestingModule({ imports: [CartaReal] });
  // Las secciones se sustituyen por plantillas vacías: aquí solo se prueba el contenedor.
  TestBed.overrideComponent(ProductosSection, {
    set: { template: '<p id="productos"></p>', imports: [], providers: [] },
  });
  TestBed.overrideComponent(CategoriasSection, {
    set: { template: '<p id="categorias"></p>', imports: [], providers: [] },
  });
  TestBed.overrideComponent(CartaReal, {
    set: { providers: [{ provide: CartaStore, useValue: store }] },
  });
  const fixture = TestBed.createComponent(CartaReal);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const boton = (texto: string) =>
    [...el.querySelectorAll('button')].find((b) => b.textContent?.trim() === texto) as
      HTMLButtonElement | undefined;
  return { store, fixture, el, boton };
}

describe('Pantalla real de Carta y recetas', () => {
  it('carga al iniciar y muestra las pestañas con Productos activa', () => {
    const { store, el } = setup();
    expect(store.load).toHaveBeenCalled();
    expect(el.textContent).toContain('CARTA Y RECETAS');
    const tabs = [...el.querySelectorAll('[role="tab"]')];
    expect(tabs.map((t) => t.textContent?.trim())).toEqual(['Productos', 'Categorías']);
    expect(tabs[0].getAttribute('aria-selected')).toBe('true');
    expect(tabs[1].getAttribute('aria-selected')).toBe('false');
    expect(el.querySelector('#productos')).not.toBeNull();
  });

  it('cambia de pestaña y la acción primaria sigue a la pestaña', () => {
    const { el, boton, fixture } = setup();
    expect(boton('+ Nuevo producto')).toBeDefined();
    boton('Categorías')!.click();
    fixture.detectChanges();
    expect(el.querySelector('#categorias')).not.toBeNull();
    expect(el.querySelector('#productos')).toBeNull();
    expect(boton('+ Nueva categoría')).toBeDefined();
    expect(boton('+ Nuevo producto')).toBeUndefined();
    expect(el.querySelectorAll('[aria-selected="true"]')[0].textContent).toContain('Categorías');
  });

  it('la acción primaria abre el alta de la sección activa', () => {
    const { fixture, boton } = setup();
    const productos = vi.fn();
    const categorias = vi.fn();
    Object.assign(fixture.componentInstance as object, {
      productos: () => ({ nuevo: productos }),
      categorias: () => ({ nueva: categorias }),
    });
    boton('+ Nuevo producto')!.click();
    expect(productos).toHaveBeenCalled();
    boton('Categorías')!.click();
    fixture.detectChanges();
    boton('+ Nueva categoría')!.click();
    expect(categorias).toHaveBeenCalled();
  });

  it('sin permiso de administrar no hay acción de alta', () => {
    const { boton } = setup({ puedeAdministrarCarta: signal(false) });
    expect(boton('+ Nuevo producto')).toBeUndefined();
  });

  it('cargando muestra el estado y no las pestañas', () => {
    const { el, boton } = setup({ status: signal('loading') });
    expect(el.querySelector('[role="status"]')?.textContent).toContain('Cargando la carta');
    expect(el.querySelector('[role="tab"]')).toBeNull();
    expect(boton('+ Nuevo producto')).toBeUndefined();
  });

  it('con error muestra el motivo y Reintentar vuelve a cargar', () => {
    const { el, store, boton } = setup({ status: signal('error') });
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Sin conexión');
    boton('Reintentar')!.click();
    expect(store.load).toHaveBeenCalledTimes(2);
  });

  it('sin acceso explica qué hacer y no muestra pestañas ni alta', () => {
    const { el, boton } = setup({ sinAcceso: signal(true) });
    expect(el.textContent).toContain('No tienes permiso para ver la carta');
    expect(el.querySelector('[role="tab"]')).toBeNull();
    expect(boton('+ Nuevo producto')).toBeUndefined();
  });

  it('el aviso usa role status (ok) o alert (error) y se descarta con Entendido', () => {
    const ok = setup({ notice: signal({ kind: 'ok', text: 'Listo' }) });
    expect(ok.el.querySelector('.note-box')?.getAttribute('role')).toBe('status');
    ok.boton('Entendido')!.click();
    expect(ok.store.dismissNotice).toHaveBeenCalled();
    TestBed.resetTestingModule();
    const error = setup({ notice: signal({ kind: 'error', text: 'Falló' }) });
    const aviso = error.el.querySelector('.note-box')!;
    expect(aviso.getAttribute('role')).toBe('alert');
    expect(aviso.classList).toContain('orange');
  });
});
