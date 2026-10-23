import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { NegocioSection } from './negocio-section';
import type { Negocio } from './configuracion.models';

const negocio: Negocio = {
  id: '1',
  singleton: true,
  ruc: '20123456789',
  razon_social: 'Mi Restaurante SAC',
  nombre_comercial: 'Mi Restaurante',
  direccion_fiscal: 'Av. Principal 123',
  ubigeo: null,
  telefono: null,
  email: null,
  moneda: 'PEN',
  zona_horaria: 'America/Lima',
  es_activo: true,
  creado_en: '',
  creado_por: '1',
  actualizado_en: '',
  actualizado_por: '1',
};

function setup(editar: boolean, editarNegocio = vi.fn().mockResolvedValue(undefined)) {
  const store = {
    negocio: signal<Negocio | null>(negocio),
    errors: signal({}),
    puedeEditarNegocio: signal(editar),
    editarNegocio,
    load: vi.fn(),
  };
  TestBed.configureTestingModule({
    imports: [NegocioSection],
    providers: [{ provide: ConfiguracionStore, useValue: store }],
  });
  const fixture = TestBed.createComponent(NegocioSection);
  fixture.detectChanges();
  return { fixture, store, el: fixture.nativeElement as HTMLElement };
}

describe('Sección Tu restaurante', () => {
  it('muestra el RUC sin poder editarlo', () => {
    const { el } = setup(true);
    const ruc = el.querySelector('#negocio-ruc') as HTMLInputElement;
    expect(ruc.value).toBe('20123456789');
    expect(ruc.readOnly).toBe(true);
  });

  it('sin permiso de edición queda en solo lectura y sin botón de guardar', () => {
    const { el } = setup(false);
    expect((el.querySelector('#negocio-nombre_comercial') as HTMLInputElement).readOnly).toBe(true);
    expect(el.querySelector('button[type="submit"]')).toBeNull();
  });

  it('guarda solo lo que cambió', async () => {
    const { fixture, store, el } = setup(true);
    const input = el.querySelector('#negocio-nombre_comercial') as HTMLInputElement;
    input.value = 'Nuevo nombre';
    input.dispatchEvent(new Event('input'));
    (el.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    expect(store.editarNegocio).toHaveBeenCalledWith({ nombre_comercial: 'Nuevo nombre' });
  });

  it('pinta el error de campo que devuelve el store y conserva lo escrito', async () => {
    const editar = vi
      .fn()
      .mockResolvedValue({ field: 'moneda', message: 'La moneda no es válida.' });
    const { fixture, el } = setup(true, editar);
    const moneda = el.querySelector('#negocio-moneda') as HTMLInputElement;
    moneda.value = 'xx';
    moneda.dispatchEvent(new Event('input'));
    (el.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(el.textContent).toContain('La moneda no es válida.');
    expect((el.querySelector('#negocio-moneda') as HTMLInputElement).value).toBe('xx');
  });

  it('un error de un campo desconocido cae al aviso general', async () => {
    const editar = vi.fn().mockResolvedValue({ field: 'otro', message: 'Dato inválido.' });
    const { fixture, el } = setup(true, editar);
    const moneda = el.querySelector('#negocio-moneda') as HTMLInputElement;
    moneda.value = 'xx';
    moneda.dispatchEvent(new Event('input'));
    (el.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(el.querySelector('[role="alert"]')?.textContent).toContain('Dato inválido.');
  });
});
