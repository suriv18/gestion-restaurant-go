import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { OrderActions } from './order-actions';
import { PageHeader } from '../../shared/ui/page-header';
import { StatusBadge } from '../../shared/ui/status-badge';
import { MoneyPipe } from '../../shared/ui/money';
import { Dialogs } from '../../shared/ui/form-dialog';
@Component({
  imports: [RouterLink, PageHeader, StatusBadge, MoneyPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="Tu cocina llega más lejos"
      description="Acompaña cada pedido, desde la preparación hasta la puerta."
      eyebrow="DELIVERY"
      ><button class="btn primary" (click)="actions.newOrder('DELIVERY')">
        + Nuevo delivery
      </button></app-page-header
    >
    <div class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
      @for (o of orders(); track o.id) {
        <article class="panel">
          <div class="panel-header">
            <div>
              <h2 class="font-bold">Pedido #{{ o.id }}</h2>
              <p class="subtle">{{ o.customer }}</p>
            </div>
            <app-status [value]="o.deliveryState || 'PENDIENTE'" />
          </div>
          <div class="panel-body">
            <p class="mb-3">{{ o.address }}</p>
            <p class="subtle">{{ o.phone }}</p>
            <div class="info-row"><span>Preparación</span><app-status [value]="o.state" /></div>
            <div class="info-row">
              <span>Importe</span><strong>{{ store.total(o) | money }}</strong>
            </div>
            @if (o.courier) {
              <div class="info-row">
                <span>Repartidor</span><strong>{{ o.courier }}</strong>
              </div>
            }
            <div class="flex flex-wrap gap-2 mt-5">
              <a class="btn" routerLink="/salon" [queryParams]="{ pedido: o.id }">Ver pedido</a>
              @if (o.deliveryState === 'PENDIENTE') {
                <button class="btn primary" (click)="dispatch(o.id)">Despachar</button>
              }
              @if (o.deliveryState === 'EN_RUTA') {
                <button
                  class="btn primary"
                  (click)="store.run('deliveryState', { order: o.id, state: 'ENTREGADO' })"
                >
                  Confirmar entrega
                </button>
              }
            </div>
          </div>
        </article>
      } @empty {
        <div class="empty-state">No hay pedidos de delivery abiertos.</div>
      }
    </div>`,
})
export class DeliveryPage {
  readonly store = inject(RestaurantStore);
  readonly actions = inject(OrderActions);
  private readonly dialogs = inject(Dialogs);
  readonly orders = computed(() => this.store.orders().filter((o) => o.type === 'DELIVERY'));
  dispatch(id: number): void {
    this.dialogs.form({
      title: 'Despachar pedido #' + id,
      description: 'Todos los productos deben estar listos antes de salir.',
      submit: 'Enviar a ruta',
      fields: [{ key: 'courier', label: 'Nombre del repartidor' }],
      commit: (v) =>
        this.store.execute('deliveryState', { order: id, state: 'EN_RUTA', courier: v['courier'] }),
    });
  }
}
