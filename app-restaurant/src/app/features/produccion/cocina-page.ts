import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { PageHeader } from '../../shared/ui/page-header';
import { StatusBadge } from '../../shared/ui/status-badge';
import { Order, LineState } from '../../shared/models/restaurant';
@Component({
  imports: [PageHeader, StatusBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: ` <app-page-header
      title="Una cocina en sintonía"
      description="Cada comanda a tiempo. Del primer fuego a la mesa."
      eyebrow="PRODUCCIÓN"
    />
    <div class="section-toolbar">
      <div class="tabs" aria-label="Estación de preparación">
        @for (s of ['Todas', 'Cocina', 'Barra']; track s) {
          <button
            class="tab"
            [class.active]="station() === s"
            [attr.aria-pressed]="station() === s"
            (click)="station.set(s)"
          >
            {{ s }}
          </button>
        }
      </div>
      <p class="subtle">El consumo de receta se registra al comenzar la preparación.</p>
    </div>
    <div class="kanban">
      @for (column of columns; track column.state) {
        <section class="kanban-col">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-sm font-bold">{{ column.title }}</h2>
            <span class="pill">{{ count(column.state) }}</span>
          </div>
          @for (order of store.orders(); track order.id) {
            @if (lines(order, column.state).length) {
              <article class="ticket">
                <div class="ticket-top">
                  <div>
                    <h3>{{ store.tableName(order) }}</h3>
                    <p class="subtle mt-1">Pedido #{{ order.id }} · {{ order.waiter }}</p>
                  </div>
                  <app-status [value]="column.state" />
                </div>
                @for (line of lines(order, column.state); track line.id) {
                  <div class="ticket-line">
                    <div class="flex justify-between gap-3">
                      <strong>{{ line.qty }} × {{ line.name }}</strong
                      ><small class="subtle">{{ line.station }}</small>
                    </div>
                    @if (line.note) {
                      <p class="note-box orange mt-3">{{ line.note }}</p>
                    }
                    <button
                      class="btn w-full mt-3"
                      [class.primary]="column.state !== 'LISTO'"
                      (click)="
                        store.run('lineState', {
                          order: order.id,
                          line: line.id,
                          state: column.next,
                        })
                      "
                    >
                      {{ column.action }}
                    </button>
                  </div>
                }
              </article>
            }
          }
          @if (!count(column.state)) {
            <div class="py-12 text-center subtle">{{ column.empty }}</div>
          }
        </section>
      }
    </div>`,
})
export class CocinaPage {
  readonly store = inject(RestaurantStore);
  readonly station = signal('Todas');
  readonly columns: {
    state: LineState;
    title: string;
    next: LineState;
    action: string;
    empty: string;
  }[] = [
    {
      state: 'PENDIENTE',
      title: 'Por preparar',
      next: 'PREPARACION',
      action: 'Comenzar preparación',
      empty: 'No hay comandas pendientes.',
    },
    {
      state: 'PREPARACION',
      title: 'En preparación',
      next: 'LISTO',
      action: 'Marcar como listo',
      empty: 'Todo al día en la cocina.',
    },
    {
      state: 'LISTO',
      title: 'Listo para servir',
      next: 'ENTREGADO',
      action: 'Confirmar entrega',
      empty: 'Los próximos platos aparecerán aquí.',
    },
  ];
  lines(order: Order, state: LineState) {
    return order.lines.filter(
      (l) => l.state === state && (this.station() === 'Todas' || l.station === this.station()),
    );
  }
  count(state: LineState): number {
    return this.store.orders().reduce((n, o) => n + this.lines(o, state).length, 0);
  }
}
