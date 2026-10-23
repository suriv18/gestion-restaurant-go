import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs } from '../../shared/ui/form-dialog';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { SedesSection } from './sedes-section';
import type { Sucursal } from './configuracion.models';

const sede = (id: string, activa: boolean): Sucursal => ({
  id,
  negocio_id: '1',
  codigo: 'S' + id,
  nombre: 'Sede ' + id,
  direccion: 'Calle ' + id,
  ubigeo: null,
  codigo_establecimiento_sunat: null,
  telefono: null,
  es_activo: activa,
  motivo_eliminacion: activa ? null : 'Cierre',
  eliminado_en: null,
  eliminado_por: null,
  creado_en: '',
  creado_por: '1',
  actualizado_en: '',
  actualizado_por: '1',
});

function setup(editar: boolean) {
  const store = {
    sedes: signal([sede('1', true), sede('2', false)]),
    errors: signal({}),
    busyId: signal<string | null>(null),
    puedeEditarSedes: signal(editar),
    restaurarSede: vi.fn(),
    load: vi.fn(),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [SedesSection],
    providers: [
      { provide: ConfiguracionStore, useValue: store },
      { provide: Dialogs, useValue: dialogs },
    ],
  });
  const fixture = TestBed.createComponent(SedesSection);
  fixture.detectChanges();
  return { store, dialogs, el: fixture.nativeElement as HTMLElement };
}

describe('Sección Sedes', () => {
  it('lista las sedes y su estado', () => {
    const { el } = setup(true);
    expect(el.textContent).toContain('Sede 1');
    expect(el.textContent).toContain('De baja');
  });

  it('con permiso ofrece alta, edición, baja y restauración', () => {
    const { el } = setup(true);
    const textos = [...el.querySelectorAll('button')].map((b) => b.textContent?.trim());
    expect(textos).toEqual(
      expect.arrayContaining(['Nueva sede', 'Editar', 'Dar de baja', 'Restaurar']),
    );
  });

  it('sin permiso de escritura no hay botones de acción', () => {
    const { el } = setup(false);
    expect(el.querySelectorAll('button')).toHaveLength(0);
  });

  it('dar de baja abre un diálogo que exige motivo', () => {
    const { el, dialogs } = setup(true);
    const baja = [...el.querySelectorAll('button')].find(
      (b) => b.textContent?.trim() === 'Dar de baja',
    );
    baja?.click();
    const data = dialogs.form.mock.calls[0][0];
    expect(data.danger).toBe(true);
    expect(data.fields.map((f: { key: string }) => f.key)).toEqual(['motivo']);
  });

  it('restaurar no abre diálogo y usa una clave de idempotencia', () => {
    const { el, dialogs, store } = setup(true);
    const restaurar = [...el.querySelectorAll('button')].find(
      (b) => b.textContent?.trim() === 'Restaurar',
    );
    restaurar?.click();
    expect(dialogs.form).not.toHaveBeenCalled();
    expect(store.restaurarSede).toHaveBeenCalledWith(
      expect.objectContaining({ id: '2' }),
      expect.any(String),
    );
  });
});
