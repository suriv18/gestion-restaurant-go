import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Invoice } from '../../shared/models/restaurant';
import { PageHeader } from '../../shared/ui/page-header';
import { MoneyPipe } from '../../shared/ui/money';
import { money } from '../../core/demo/engine/model.js';
import { Dialogs, options } from '../../shared/ui/form-dialog';
@Component({
  imports: [PageHeader, MoneyPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="Cada venta, bien documentada"
      description="Consulta las vistas previas de tus pedidos cobrados."
      eyebrow="COMPROBANTES"
      ><button class="btn primary" (click)="create()">
        + Simular comprobante
      </button></app-page-header
    >
    <p class="note-box orange mb-6">
      Documentos de ejemplo, sin validez tributaria. No se envían a SUNAT ni incluyen un desglose
      fiscal validado.
    </p>
    <div class="panel table-scroll">
      <table>
        <thead>
          <tr>
            <th>Número</th>
            <th>Cliente</th>
            <th>Pedido</th>
            <th>Importe</th>
            <th>Estado</th>
            <th>Acción</th>
          </tr>
        </thead>
        <tbody>
          @for (i of store.state().invoices; track i.id) {
            <tr>
              <td>
                <strong>{{ i.number }}</strong
                ><small>{{ i.type }}</small>
              </td>
              <td>
                {{ i.customer }}<small>{{ i.document || 'Sin documento' }}</small>
              </td>
              <td>#{{ i.order }}</td>
              <td>{{ i.amount | money }}</td>
              <td><span class="pill orange">Simulado</span></td>
              <td><button class="btn" (click)="detail(i)">Ver documento</button></td>
            </tr>
          } @empty {
            <tr>
              <td colspan="6">
                <div class="empty-state">Cobra un pedido para generar su vista previa.</div>
              </td>
            </tr>
          }
        </tbody>
      </table>
    </div>`,
})
export class ComprobantesPage {
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  create(): void {
    const orders = this.store
      .state()
      .orders.filter(
        (o) =>
          o.state !== 'CANCELADO' &&
          this.store.total(o) > 0 &&
          this.store.balance(o) === 0 &&
          !this.store.state().invoices.some((i) => i.order === o.id),
      );
    if (!orders.length) {
      this.store.notify('No hay pedidos cobrados sin comprobante.');
      return;
    }
    this.dialogs.form({
      title: 'Simular comprobante',
      description:
        'Vista previa sin validez tributaria. La factura de ejemplo requiere RUC y razón social.',
      submit: 'Crear documento de ejemplo',
      fields: [
        {
          key: 'order',
          label: 'Pedido cobrado',
          type: 'select',
          options: orders.map((o) => ({
            value: o.id,
            label: '#' + o.id + ' · ' + o.customer + ' · ' + money(this.store.total(o)),
          })),
        },
        { key: 'type', label: 'Tipo', type: 'select', options: options(['Boleta', 'Factura']) },
        { key: 'document', label: 'Documento / RUC', required: false },
        { key: 'customer', label: 'Cliente / razón social', required: false },
      ],
      commit: (v) =>
        this.store.execute('invoice', {
          order: Number(v['order']),
          type: v['type'],
          document: v['document'],
          customer: v['customer'],
        }),
    });
  }
  detail(i: Invoice): void {
    const lines = i.lines ?? this.store.state().orders.find((o) => o.id === i.order)?.lines ?? [];
    this.dialogs.detail({
      title: i.number,
      description: 'DOCUMENTO DE EJEMPLO · Sin validez tributaria · No enviado a SUNAT',
      rows: [
        { label: 'Restaurante', value: this.store.state().settings.name },
        { label: 'Cliente', value: i.customer },
        { label: 'Documento', value: i.document || '—' },
        ...lines
          .filter((l) => l.state !== 'CANCELADO')
          .map((l) => ({ label: l.qty + ' × ' + l.name, value: money(l.price * l.qty) })),
        { label: 'Total', value: money(i.amount) },
      ],
    });
  }
}
