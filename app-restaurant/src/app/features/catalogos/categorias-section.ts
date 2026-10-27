import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs, type FieldProblem } from '../../shared/ui/form-dialog';
import { CartaStore, type ValoresCategoria } from './data-access/carta-store';
import { Categoria } from './data-access/carta.models';

type Resultado = Promise<string | FieldProblem | undefined>;

/** Categorías de la carta: tabla ordenada, filtro «Incluir bajas» (de cliente) y diálogos. */
@Component({
  selector: 'app-categorias-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="section-toolbar">
      <label class="flex gap-2 text-xs items-center">
        <input
          type="checkbox"
          [checked]="incluirBajas()"
          (change)="incluirBajas.set($any($event.target).checked)"
        />Incluir bajas
      </label>
    </div>
    @if (filas().length === 0) {
      <div class="empty-state" role="status">
        <h2>Aún no hay categorías</h2>
        <p>Las categorías agrupan los productos de tu carta.</p>
        @if (store.puedeAdministrarCarta()) {
          <button type="button" class="btn primary" (click)="nueva()">+ Nueva categoría</button>
        }
      </div>
    } @else {
      <div class="panel table-scroll">
        <table>
          <caption class="sr-only">
            Categorías de la carta
          </caption>
          <thead>
            <tr>
              <th scope="col">Nombre</th>
              <th scope="col">Descripción</th>
              <th scope="col">Orden</th>
              <th scope="col">Estado</th>
              <th scope="col">Acciones</th>
            </tr>
          </thead>
          <tbody>
            @for (c of filas(); track c.id) {
              <tr>
                <td>
                  <strong>{{ c.nombre }}</strong>
                </td>
                <td>{{ c.descripcion || '—' }}</td>
                <td>{{ c.orden_visualizacion }}</td>
                <td>
                  <span class="pill" [class.green]="c.es_activo">{{
                    c.es_activo ? 'Activa' : 'De baja'
                  }}</span>
                </td>
                <td>
                  @if (store.puedeAdministrarCarta()) {
                    <div class="inline-actions">
                      @if (c.es_activo) {
                        <button
                          type="button"
                          class="btn"
                          [attr.aria-label]="'Editar categoría ' + c.nombre"
                          (click)="editar(c)"
                        >
                          Editar
                        </button>
                        <button
                          type="button"
                          class="btn danger"
                          [attr.aria-label]="'Desactivar categoría ' + c.nombre"
                          (click)="desactivar(c)"
                        >
                          Desactivar
                        </button>
                      } @else {
                        <button
                          type="button"
                          class="btn"
                          [attr.aria-label]="'Restaurar categoría ' + c.nombre"
                          [disabled]="store.busyId() !== null"
                          (click)="restaurar(c)"
                        >
                          Restaurar
                        </button>
                      }
                    </div>
                  }
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>
    }
  `,
})
export class CategoriasSection {
  readonly store = inject(CartaStore);
  private readonly dialogs = inject(Dialogs);

  readonly incluirBajas = signal(false);

  readonly filas = computed(() =>
    this.store.categoriasOrdenadas().filter((c) => this.incluirBajas() || c.es_activo),
  );

  nueva(): void {
    this.formCategoria();
  }

  editar(c: Categoria): void {
    this.formCategoria(c);
  }

  desactivar(c: Categoria): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Desactivar categoría: ' + c.nombre,
      description:
        'Dejará de ofrecerse al crear productos. Si todavía tiene productos activos, primero muévelos o dalos de baja. Podrás restaurarla cuando quieras.',
      danger: true,
      submit: 'Desactivar',
      fields: [],
      commit: async (): Resultado => this.store.desactivarCategoria(c, key),
    });
  }

  restaurar(c: Categoria): void {
    void this.store.restaurarCategoria(c, newIdempotencyKey());
  }

  private formCategoria(c?: Categoria): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: c ? 'Editar categoría' : 'Nueva categoría',
      description: 'Las categorías se muestran en la carta según su orden, de menor a mayor.',
      submit: 'Guardar categoría',
      fields: [
        { key: 'nombre', label: 'Nombre', value: c?.nombre },
        {
          key: 'descripcion',
          label: 'Descripción',
          type: 'textarea',
          required: false,
          value: c?.descripcion ?? '',
        },
        {
          key: 'orden_visualizacion',
          label: 'Orden',
          type: 'number',
          min: 0,
          max: 9999,
          step: '1',
          value: c?.orden_visualizacion ?? 0,
          hint: 'Un número entre 0 y 9999. Menor se muestra primero.',
        },
      ],
      commit: async (v): Resultado => {
        if (!/^\d{1,4}$/.test(v['orden_visualizacion'].trim())) {
          return {
            field: 'orden_visualizacion',
            message: 'Escribe un número entero entre 0 y 9999.',
          };
        }
        const valores = v as unknown as ValoresCategoria;
        return c ? this.store.editarCategoria(c, valores) : this.store.crearCategoria(valores, key);
      },
    });
  }
}
