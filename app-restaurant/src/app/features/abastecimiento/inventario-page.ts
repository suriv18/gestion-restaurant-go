import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Ingredient } from '../../shared/models/restaurant';
import { PageHeader } from '../../shared/ui/page-header';
import { Metric } from '../../shared/ui/metric';
import { Dialogs } from '../../shared/ui/form-dialog';
@Component({
  imports: [PageHeader, Metric],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: ` <app-page-header
      title="Todo en su punto"
      description="Cuida tus insumos. Cada ingreso y cada consumo tienen una historia."
      eyebrow="INVENTARIO"
      ><button class="btn primary" (click)="movement()">
        + Registrar movimiento
      </button></app-page-header
    >
    <div class="metrics !grid-cols-2">
      <app-metric
        label="Insumos registrados"
        [value]="store.state().ingredients.length"
        caption="Almacén principal"
        icon="box"
      /><app-metric
        label="Por reponer"
        [value]="store.lowStock().length"
        caption="Por debajo del mínimo"
        icon="bell"
      />
    </div>
    <div class="section-toolbar">
      <label class="search"
        ><input
          placeholder="Buscar insumo..."
          aria-label="Buscar insumo"
          [value]="query()"
          (input)="query.set($any($event.target).value)" /></label
      ><label class="flex gap-2 items-center text-xs"
        ><input
          type="checkbox"
          [checked]="lowOnly()"
          (change)="lowOnly.set($any($event.target).checked)"
        />Solo por reponer</label
      >
    </div>
    <div class="panel table-scroll">
      <table>
        <thead>
          <tr>
            <th>Insumo</th>
            <th>Existencia</th>
            <th>Mínimo</th>
            <th>Estado</th>
            <th>Historial</th>
          </tr>
        </thead>
        <tbody>
          @for (i of ingredients(); track i.id) {
            <tr>
              <td>
                <strong>{{ i.name }}</strong
                ><small>{{ i.code }}</small>
              </td>
              <td class="amount">{{ store.stock(i.id) }} {{ i.unit }}</td>
              <td>{{ i.min }} {{ i.unit }}</td>
              <td>
                <span
                  class="pill"
                  [class.orange]="store.stock(i.id) < i.min"
                  [class.green]="store.stock(i.id) >= i.min"
                  >{{ store.stock(i.id) < i.min ? 'Por reponer' : 'Suficiente' }}</span
                >
              </td>
              <td><button class="btn" (click)="kardex(i)">Ver kardex</button></td>
            </tr>
          } @empty {
            <tr>
              <td colspan="5"><div class="empty-state">No hay insumos que coincidan.</div></td>
            </tr>
          }
        </tbody>
      </table>
    </div>`,
})
export class InventarioPage {
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  readonly query = signal('');
  readonly lowOnly = signal(false);
  readonly ingredients = computed(() =>
    this.store
      .state()
      .ingredients.filter(
        (i) =>
          i.name.toLowerCase().includes(this.query().toLowerCase()) &&
          (!this.lowOnly() || this.store.stock(i.id) < i.min),
      ),
  );
  movement(): void {
    this.dialogs.form({
      title: 'Registrar movimiento',
      description:
        'El stock se calcula desde el kardex. No se permiten salidas con stock negativo.',
      submit: 'Registrar movimiento',
      fields: [
        {
          key: 'item',
          label: 'Insumo',
          type: 'select',
          options: this.store
            .state()
            .ingredients.filter((i) => i.es_activo)
            .map((i) => ({ value: i.id, label: i.name + ' (' + i.unit + ')' })),
        },
        {
          key: 'type',
          label: 'Tipo',
          type: 'select',
          options: [
            { value: 'AJUSTE_ENTRADA', label: 'Ajuste de entrada' },
            { value: 'AJUSTE_SALIDA', label: 'Ajuste de salida' },
            { value: 'MERMA', label: 'Merma' },
          ],
        },
        { key: 'qty', label: 'Cantidad en unidad base', type: 'number', min: 0.001, step: '0.001' },
        { key: 'reason', label: 'Motivo', type: 'textarea' },
      ],
      commit: (v) =>
        this.store.execute('movement', {
          item: Number(v['item']),
          type: v['type'],
          qty: Number(v['qty']),
          reason: v['reason'],
        }),
    });
  }
  kardex(i: Ingredient): void {
    this.dialogs.detail({
      title: 'Kardex · ' + i.name,
      description: 'Existencia actual: ' + this.store.stock(i.id) + ' ' + i.unit,
      rows: this.store
        .state()
        .movements.filter((m) => m.item === i.id)
        .slice()
        .reverse()
        .map((m) => ({
          label: new Date(m.at).toLocaleString('es-PE') + ' · ' + m.type,
          value: (m.qty > 0 ? '+' : '') + m.qty + ' ' + i.unit + ' · ' + m.note,
        })),
    });
  }
}
