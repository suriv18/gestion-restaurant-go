import { Dialog } from '@angular/cdk/dialog';
import { PurchaseDialog } from './purchase-dialog';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Purchase } from '../../shared/models/restaurant';
import { PageHeader } from '../../shared/ui/page-header';
import { StatusBadge } from '../../shared/ui/status-badge';
import { MoneyPipe } from '../../shared/ui/money';
import { Dialogs } from '../../shared/ui/form-dialog';
@Component({
  imports: [PageHeader, StatusBadge, MoneyPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: ` <app-page-header
      title="La calidad empieza aquí"
      description="Organiza tus compras y recibe los insumos que tu cocina necesita."
      eyebrow="COMPRAS"
      ><button class="btn primary" (click)="create()">+ Nueva compra</button></app-page-header
    >
    <div class="panel table-scroll">
      <table>
        <thead>
          <tr>
            <th>Referencia</th>
            <th>Proveedor</th>
            <th>Importe</th>
            <th>Estado</th>
            <th>Acciones</th>
          </tr>
        </thead>
        <tbody>
          @for (p of store.state().purchases; track p.id) {
            <tr>
              <td>
                <strong>{{ p.reference }}</strong
                ><small>{{ p.at.slice(0, 10) }}</small>
              </td>
              <td>{{ supplier(p.supplier) }}</td>
              <td>{{ total(p) | money }}</td>
              <td><app-status [value]="p.state" /></td>
              <td>
                <div class="inline-actions">
                  <button class="btn" (click)="detail(p)">Detalle</button>
                  @if (p.state === 'BORRADOR') {
                    <button class="btn primary" (click)="receive(p)">Recibir</button
                    ><button class="btn danger" (click)="cancel(p)">Cancelar</button>
                  }
                </div>
              </td>
            </tr>
          } @empty {
            <tr>
              <td colspan="5"><div class="empty-state">Aún no hay compras registradas.</div></td>
            </tr>
          }
        </tbody>
      </table>
    </div>`,
})
export class ComprasPage {
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  private readonly dialog = inject(Dialog);
  supplier(id: number): string {
    return this.store.state().suppliers.find((s) => s.id === id)?.name ?? 'Proveedor';
  }
  total(p: Purchase): number {
    return Math.round(p.lines.reduce((sum, l) => sum + l.qty * l.cost, 0));
  }
  create(): void {
    this.dialog.open(PurchaseDialog, {
      panelClass: 'app-dialog',
      ariaLabelledBy: 'purchase-title',
      ariaModal: true,
    });
  }
  detail(p: Purchase): void {
    this.dialogs.detail({
      title: p.reference,
      description: this.supplier(p.supplier),
      rows: p.lines.map((l) => ({
        label: this.store.state().ingredients.find((i) => i.id === l.item)?.name ?? 'Insumo',
        value: l.qty + ' × S/ ' + (l.cost / 100).toFixed(2),
      })),
    });
  }
  receive(p: Purchase): void {
    this.dialogs.form({
      title: 'Recibir mercadería',
      description:
        p.reference +
        ' · Confirma que las cantidades llegaron al almacén. Solo se puede recibir una vez.',
      fields: [],
      submit: 'Confirmar recepción',
      commit: () => this.store.execute('receivePurchase', { id: p.id }),
    });
  }
  cancel(p: Purchase): void {
    this.dialogs.form({
      title: 'Cancelar compra',
      description: p.reference + ' · El borrador se conservará como cancelado.',
      fields: [],
      submit: 'Cancelar compra',
      danger: true,
      commit: () => this.store.execute('cancelPurchase', { id: p.id }),
    });
  }
}
