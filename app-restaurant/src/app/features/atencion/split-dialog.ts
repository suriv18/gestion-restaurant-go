import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { DIALOG_DATA, DialogRef } from '@angular/cdk/dialog';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Order } from '../../shared/models/restaurant';
import { Icon } from '../../shared/ui/icon';
import { MoneyPipe } from '../../shared/ui/money';
@Component({
  selector: 'app-split-dialog',
  imports: [Icon, MoneyPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<div class="modal-header">
      <div>
        <h2 id="split-title">Dividir cuenta</h2>
        <p>Selecciona líneas completas antes del primer pago.</p>
      </div>
      <button class="icon-btn" aria-label="Cerrar diálogo" (click)="ref.close()">
        <app-icon name="close" />
      </button>
    </div>
    <div class="modal-body">
      @for (line of order.lines; track line.id) {
        @if (line.state !== 'CANCELADO') {
          <label class="check-row"
            ><input
              type="checkbox"
              [checked]="selected().includes(line.id)"
              (change)="toggle(line.id)"
            /><span
              >{{ line.qty }} × {{ line.name
              }}<small class="block">Cuenta {{ line.account }}</small></span
            ><strong>{{ line.qty * line.price | money }}</strong></label
          >
        }
      }
      @if (error()) {
        <p class="form-error" role="alert">{{ error() }}</p>
      }
      <div class="modal-actions">
        <button class="btn" (click)="ref.close()">Cancelar</button
        ><button class="btn primary" [disabled]="!selected().length" (click)="save()">
          Crear cuenta
        </button>
      </div>
    </div>`,
})
export class SplitDialog {
  readonly order = inject<Order>(DIALOG_DATA);
  readonly ref = inject(DialogRef);
  private readonly store = inject(RestaurantStore);
  readonly selected = signal<number[]>([]);
  readonly error = signal('');
  toggle(id: number): void {
    this.selected.update((ids) => (ids.includes(id) ? ids.filter((i) => i !== id) : [...ids, id]));
  }
  save(): void {
    try {
      this.store.execute('split', { order: this.order.id, lines: this.selected() });
      this.ref.close(true);
    } catch (e) {
      this.error.set(e instanceof Error ? e.message : 'No se pudo dividir.');
    }
  }
}
