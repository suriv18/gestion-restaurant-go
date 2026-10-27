import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Dialogs, type FormDialogData } from '../../shared/ui/form-dialog';
import { CategoriasSection } from './categorias-section';
import { CartaStore } from './data-access/carta-store';
import type { Categoria } from './data-access/carta.models';

const categoria = (id: string, nombre: string, orden: number, es_activo = true): Categoria => ({
  id,
  nombre,
  descripcion: id === '1' ? 'Platos de fondo' : null,
  orden_visualizacion: orden,
  es_activo,
});

function setup(admin: boolean, lista: Categoria[] | null = null) {
  const todas = lista ?? [
    categoria('3', 'Postres', 2),
    categoria('2', 'Bebidas', 1, false),
    categoria('1', 'Fondos', 1),
  ];
  const store = {
    categoriasOrdenadas: signal(
      [...todas].sort(
        (a, b) => a.orden_visualizacion - b.orden_visualizacion || a.nombre.localeCompare(b.nombre),
      ),
    ),
    busyId: signal<string | null>(null),
    puedeAdministrarCarta: signal(admin),
    crearCategoria: vi.fn().mockResolvedValue(undefined),
    editarCategoria: vi.fn().mockResolvedValue(undefined),
    desactivarCategoria: vi.fn().mockResolvedValue(undefined),
    restaurarCategoria: vi.fn(),
  };
  const dialogs = { form: vi.fn() };
  TestBed.configureTestingModule({
    imports: [CategoriasSection],
    providers: [
      { provide: CartaStore, useValue: store },
      { provide: Dialogs, useValue: dialogs },
    ],
  });
  const fixture = TestBed.createComponent(CategoriasSection);
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const boton = (texto: string) =>
    [...el.querySelectorAll('button')].find(
      (b) => b.getAttribute('aria-label') === texto || b.textContent?.trim() === texto,
    ) as HTMLButtonElement | undefined;
  const incluirBajas = () => {
    const check = el.querySelector('input[type="checkbox"]') as HTMLInputElement;
    check.checked = true;
    check.dispatchEvent(new Event('change'));
    fixture.detectChanges();
  };
  const ultimoDialogo = () => dialogs.form.mock.calls.at(-1)![0] as FormDialogData;
  return { store, dialogs, fixture, el, boton, incluirBajas, ultimoDialogo };
}

describe('Sección Categorías', () => {
  it('lista solo las activas, en orden, con descripción y estado', () => {
    const { el } = setup(true);
    const filas = [...el.querySelectorAll('tbody tr')].map((f) => f.textContent!);
    expect(filas.length).toBe(2);
    expect(filas[0]).toContain('Fondos');
    expect(filas[0]).toContain('Platos de fondo');
    expect(filas[0]).toContain('Activa');
    expect(filas[1]).toContain('Postres');
    expect(filas[1]).toContain('—');
    expect(el.querySelector('caption')?.textContent).toContain('Categorías de la carta');
  });

  it('«Incluir bajas» muestra las categorías de baja con Restaurar por fila', () => {
    const { el, boton, incluirBajas, store } = setup(true);
    incluirBajas();
    expect(el.querySelectorAll('tbody tr').length).toBe(3);
    expect(el.textContent).toContain('De baja');
    const etiquetas = [...el.querySelectorAll('tbody button')].map((b) =>
      b.getAttribute('aria-label'),
    );
    expect(etiquetas).toEqual(
      expect.arrayContaining([
        'Editar categoría Fondos',
        'Desactivar categoría Fondos',
        'Restaurar categoría Bebidas',
      ]),
    );
    boton('Restaurar categoría Bebidas')!.click();
    expect(store.restaurarCategoria).toHaveBeenCalledWith(
      expect.objectContaining({ id: '2' }),
      expect.any(String),
    );
  });

  it('sin permiso es de solo lectura', () => {
    const { el } = setup(false);
    expect(el.querySelectorAll('tbody button').length).toBe(0);
  });

  it('sin categorías ofrece crear solo a quien puede', () => {
    const admin = setup(true, []);
    expect(admin.el.textContent).toContain('Aún no hay categorías');
    expect(admin.boton('+ Nueva categoría')).toBeDefined();
    TestBed.resetTestingModule();
    expect(setup(false, []).boton('+ Nueva categoría')).toBeUndefined();
  });

  it('Editar: campos, orden 0–9999 y validación del orden', async () => {
    const { boton, store, ultimoDialogo } = setup(true);
    boton('Editar categoría Fondos')!.click();
    const d = ultimoDialogo();
    expect(d.fields.map((f) => f.key)).toEqual(['nombre', 'descripcion', 'orden_visualizacion']);
    expect([d.fields[2].min, d.fields[2].max]).toEqual([0, 9999]);
    expect(
      await d.commit({ nombre: 'F', descripcion: '', orden_visualizacion: '10000' }),
    ).toMatchObject({ field: 'orden_visualizacion' });
    expect(
      await d.commit({ nombre: 'F', descripcion: '', orden_visualizacion: '1.5' }),
    ).toMatchObject({ field: 'orden_visualizacion' });
    expect(store.editarCategoria).not.toHaveBeenCalled();
    await d.commit({ nombre: 'Fondos', descripcion: '', orden_visualizacion: '5' });
    expect(store.editarCategoria).toHaveBeenCalledWith(
      expect.objectContaining({ id: '1' }),
      expect.objectContaining({ orden_visualizacion: '5' }),
    );
  });

  it('Nueva categoría usa una clave de idempotencia', async () => {
    const { store, fixture, ultimoDialogo } = setup(true);
    fixture.componentInstance.nueva();
    await ultimoDialogo().commit({ nombre: 'Sopas', descripcion: '', orden_visualizacion: '0' });
    expect(store.crearCategoria).toHaveBeenCalledWith(
      expect.objectContaining({ nombre: 'Sopas' }),
      expect.any(String),
    );
  });

  it('Desactivar pide confirmación y muestra el detalle del servidor si la rechaza', async () => {
    const { boton, store, ultimoDialogo } = setup(true);
    store.desactivarCategoria.mockResolvedValue('Tiene 3 productos activos.');
    boton('Desactivar categoría Fondos')!.click();
    const d = ultimoDialogo();
    expect(d.danger).toBe(true);
    expect(d.fields).toEqual([]);
    expect(await d.commit({})).toBe('Tiene 3 productos activos.');
    expect(store.desactivarCategoria).toHaveBeenCalledWith(
      expect.objectContaining({ id: '1' }),
      expect.any(String),
    );
  });
});
