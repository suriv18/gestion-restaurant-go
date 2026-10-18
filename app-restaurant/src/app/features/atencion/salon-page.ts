import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Table } from '../../shared/models/restaurant';
import { Icon } from '../../shared/ui/icon';
import { Metric } from '../../shared/ui/metric';
import { MoneyPipe } from '../../shared/ui/money';
import { PageHeader } from '../../shared/ui/page-header';
import { StatusBadge } from '../../shared/ui/status-badge';
import { OrderActions } from './order-actions';
@Component({
  imports: [RouterLink, Icon, Metric, MoneyPipe, PageHeader, StatusBadge],
  templateUrl: './salon-page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SalonPage {
  readonly store = inject(RestaurantStore);
  readonly actions = inject(OrderActions);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly params = toSignal(this.route.queryParamMap, {
    initialValue: this.route.snapshot.queryParamMap,
  });
  readonly selectedId = signal(1042);
  readonly zone = signal('Salón principal');
  readonly account = signal(0);
  readonly selected = computed(() =>
    this.store
      .orders()
      .find((o) => o.id === (Number(this.params().get('pedido')) || this.selectedId())),
  );
  readonly tables = computed(() =>
    this.store.state().tables.filter((t) => t.zone === this.zone() && t.es_activo),
  );
  readonly visibleLines = computed(
    () =>
      this.selected()?.lines.filter(
        (l) => l.state !== 'CANCELADO' && (!this.account() || l.account === this.account()),
      ) ?? [],
  );
  readonly hasNew = computed(
    () => this.selected()?.lines.some((l) => l.state === 'NUEVO') ?? false,
  );
  readonly canEdit = computed(() => !!this.selected() && this.store.paid(this.selected()!) === 0);
  readonly nextReservations = computed(() =>
    this.store
      .state()
      .reservations.filter((r) => ['PENDIENTE', 'CONFIRMADA'].includes(r.state))
      .slice(0, 2),
  );
  orderFor(table: Table) {
    return this.store.orders().find((o) => o.table === table.id);
  }
  select(table: Table): void {
    const order = this.orderFor(table);
    if (order) this.selectOrder(order.id);
    else this.actions.newOrder('SALON', table.id);
  }
  selectOrder(id: number): void {
    this.account.set(0);
    this.selectedId.set(id);
    void this.router.navigate([], { relativeTo: this.route, queryParams: { pedido: id } });
  }
  minutes(at: string): number {
    return Math.max(1, Math.floor((Date.now() - new Date(at).getTime()) / 60000));
  }
}
