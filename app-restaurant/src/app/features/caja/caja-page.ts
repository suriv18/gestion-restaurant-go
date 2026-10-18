import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Payment } from '../../shared/models/restaurant';
import { OrderActions } from '../atencion/order-actions';
import { PageHeader } from '../../shared/ui/page-header';
import { Metric } from '../../shared/ui/metric';
import { MoneyPipe, toCents } from '../../shared/ui/money';
import { Dialogs, options } from '../../shared/ui/form-dialog';
@Component({
  imports: [RouterLink, PageHeader, Metric, MoneyPipe],
  templateUrl: './caja-page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CajaPage {
  readonly store = inject(RestaurantStore);
  readonly actions = inject(OrderActions);
  private readonly dialogs = inject(Dialogs);
  collected(): number {
    return this.store
      .state()
      .payments.filter((p) => p.session === this.store.state().session.id)
      .reduce((s, p) => s + p.amount, 0);
  }
  cashAction(): void {
    const open = this.store.state().session.state === 'ABIERTA';
    this.dialogs.form({
      title: open ? 'Cerrar turno de caja' : 'Abrir caja',
      description: open
        ? 'Cuenta el efectivo físico. El cierre conserva el esperado, contado y diferencia.'
        : 'El fondo inicial será la base del efectivo esperado.',
      submit: open ? 'Confirmar cierre' : 'Abrir caja',
      fields: [
        {
          key: 'amount',
          label: open ? 'Efectivo contado (S/)' : 'Fondo inicial (S/)',
          value: open ? '' : '200.00',
        },
      ],
      commit: (v) =>
        open
          ? this.store.execute('closeCash', { counted: toCents(v['amount']) })
          : this.store.execute('openCash', { opening: toCents(v['amount']) }),
    });
  }
  movement(): void {
    this.dialogs.form({
      title: 'Movimiento de efectivo',
      description: 'Entradas o salidas distintas de un cobro.',
      fields: [
        {
          key: 'type',
          label: 'Tipo de movimiento',
          type: 'select',
          options: options(['ENTRADA', 'SALIDA']),
        },
        { key: 'amount', label: 'Importe (S/)' },
        { key: 'reason', label: 'Motivo', type: 'textarea' },
      ],
      commit: (v) =>
        this.store.execute('cashMovement', {
          type: v['type'] as 'ENTRADA' | 'SALIDA',
          amount: toCents(v['amount']),
          reason: v['reason'],
        }),
    });
  }
  refund(p: Payment): void {
    const refunded = this.store
      .state()
      .refunds.filter((r) => r.payment === p.id)
      .reduce((s, r) => s + r.amount, 0);
    this.dialogs.form({
      title: 'Registrar devolución',
      description:
        'Pedido #' +
        p.order +
        ' · Disponible S/ ' +
        ((p.amount - refunded) / 100).toFixed(2) +
        '. El cobro original se conserva.',
      submit: 'Registrar devolución',
      fields: [
        { key: 'amount', label: 'Importe a devolver (S/)' },
        { key: 'reason', label: 'Motivo', type: 'textarea' },
      ],
      commit: (v) =>
        this.store.execute('refund', {
          payment: p.id,
          amount: toCents(v['amount']),
          reason: v['reason'],
        }),
    });
  }
}
