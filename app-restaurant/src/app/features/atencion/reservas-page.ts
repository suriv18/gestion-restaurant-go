import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { today } from '../../core/demo/engine/data.js';
import { PageHeader } from '../../shared/ui/page-header';
import { StatusBadge } from '../../shared/ui/status-badge';
import { Dialogs } from '../../shared/ui/form-dialog';
@Component({
  imports: [PageHeader, StatusBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: ` <app-page-header
      title="Un lugar para cada encuentro"
      description="Organiza las próximas visitas y recibe a tus clientes sin esperas."
      eyebrow="RESERVAS"
      ><button class="btn primary" (click)="create()">+ Nueva reserva</button></app-page-header
    >
    <div class="section-toolbar">
      <label class="field !mb-0"
        >Fecha de reserva<input
          type="date"
          [value]="date()"
          (input)="date.set($any($event.target).value)" /></label
      ><button class="btn" (click)="date.set('')">Ver todas</button>
    </div>
    <div class="panel table-scroll">
      <table>
        <caption>
          Agenda de reservas ·
          {{
            filtered().length
          }}
          registros
        </caption>
        <thead>
          <tr>
            <th>Cliente</th>
            <th>Fecha y horario</th>
            <th>Mesa</th>
            <th>Personas</th>
            <th>Estado</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          @for (r of filtered(); track r.id) {
            <tr>
              <td>
                <strong>{{ r.name }}</strong
                ><small>{{ r.phone }}</small>
                @if (r.note) {
                  <small>{{ r.note }}</small>
                }
              </td>
              <td>
                {{ r.date }}<small>{{ r.start }} – {{ r.end }}</small>
              </td>
              <td>{{ tableName(r.table) }}</td>
              <td>{{ r.guests }}</td>
              <td><app-status [value]="r.state" /></td>
              <td>
                <div class="inline-actions">
                  @if (r.state === 'PENDIENTE') {
                    <button
                      class="btn"
                      (click)="store.run('reservationState', { id: r.id, state: 'CONFIRMADA' })"
                    >
                      Confirmar
                    </button>
                  }
                  @if (r.state === 'PENDIENTE' || r.state === 'CONFIRMADA') {
                    <button class="btn primary" (click)="receive(r.id)">Recibir</button
                    ><button
                      class="btn"
                      (click)="store.run('reservationState', { id: r.id, state: 'NO_ASISTIO' })"
                    >
                      No asistió</button
                    ><button class="btn danger" (click)="cancel(r.id, r.name)">Cancelar</button>
                  }
                </div>
              </td>
            </tr>
          } @empty {
            <tr>
              <td colspan="6"><div class="empty-state">No hay reservas para esta fecha.</div></td>
            </tr>
          }
        </tbody>
      </table>
    </div>`,
})
export class ReservasPage {
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  private readonly router = inject(Router);
  readonly date = signal(today());
  readonly filtered = computed(() =>
    this.store
      .state()
      .reservations.filter((r) => !this.date() || r.date === this.date())
      .slice()
      .sort((a, b) => (a.date + a.start).localeCompare(b.date + b.start)),
  );
  tableName(id: number): string {
    return this.store.state().tables.find((t) => t.id === id)?.name ?? 'Mesa';
  }
  create(): void {
    this.dialogs.form({
      title: 'Nueva reserva',
      description: 'Los horarios de una misma mesa no pueden superponerse.',
      submit: 'Confirmar reserva',
      fields: [
        { key: 'name', label: 'Nombre del cliente' },
        { key: 'phone', label: 'Teléfono', type: 'tel' },
        {
          key: 'table',
          label: 'Mesa',
          type: 'select',
          options: this.store
            .enabledTables()
            .map((t) => ({ value: t.id, label: t.name + ' · ' + t.capacity + ' personas' })),
        },
        { key: 'guests', label: 'Personas', type: 'number', min: 1, value: 2, step: '1' },
        { key: 'date', label: 'Fecha', type: 'date', value: today(), min: today() },
        { key: 'start', label: 'Hora de llegada', type: 'time', value: '19:00' },
        { key: 'end', label: 'Hora de salida', type: 'time', value: '20:30' },
        { key: 'note', label: 'Observaciones', type: 'textarea', required: false },
      ],
      commit: (v) =>
        this.store.execute('reservation', {
          name: v['name'],
          phone: v['phone'],
          table: Number(v['table']),
          guests: Number(v['guests']),
          date: v['date'],
          start: v['start'],
          end: v['end'],
          note: v['note'],
        }),
    });
  }
  receive(id: number): void {
    try {
      const order = this.store.execute('reservationState', { id, state: 'ATENDIDA' });
      void this.router.navigate(['/salon'], { queryParams: { pedido: order } });
    } catch (e) {
      this.store.notify(e instanceof Error ? e.message : 'No se pudo recibir.');
    }
  }
  cancel(id: number, name: string): void {
    this.dialogs.form({
      title: 'Cancelar reserva',
      description: 'La reserva de ' + name + ' conservará su historial.',
      fields: [],
      danger: true,
      submit: 'Cancelar reserva',
      commit: () => this.store.execute('reservationState', { id, state: 'CANCELADA' }),
    });
  }
}
