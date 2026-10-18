import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { PageHeader } from '../../shared/ui/page-header';
import { Metric } from '../../shared/ui/metric';
import { MoneyPipe } from '../../shared/ui/money';
import { StatusBadge } from '../../shared/ui/status-badge';
@Component({
  imports: [RouterLink, PageHeader, Metric, MoneyPipe, StatusBadge],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: ` <app-page-header
      title="Un buen día empieza aquí"
      description="Una mirada a tu restaurante. Todo lo que necesitas para el próximo servicio."
      eyebrow="RESUMEN"
      ><a class="btn primary" routerLink="/salon">Ir al salón →</a></app-page-header
    >
    <div class="dashboard-greeting mb-7">
      <div>
        <span class="eyebrow">BIENVENIDA, VALERIA</span>
        <h2>Hagamos que cada mesa cuente.</h2>
        <p class="subtle mt-3">
          {{ store.state().settings.name }} · {{ store.state().settings.branch }}
        </p>
      </div>
    </div>
    <div class="metrics">
      <app-metric
        label="Cobros registrados"
        [value]="collected() | money"
        caption="Historial de demostración"
        icon="wallet"
      /><app-metric
        label="Pedidos abiertos"
        [value]="store.orders().length"
        caption="En todos los canales"
        icon="receipt"
      /><app-metric
        label="Mesas ocupadas"
        [value]="store.occupied()"
        caption="Servicio en curso"
        icon="table"
      /><app-metric
        label="Insumos por reponer"
        [value]="store.lowStock().length"
        caption="Revisa tu inventario"
        icon="box"
      />
    </div>
    <div class="grid xl:grid-cols-[1.5fr_1fr] gap-6">
      <section class="panel">
        <div class="panel-header">
          <h2 class="font-bold">La operación, en marcha</h2>
          <a class="text-link" routerLink="/cocina">Ver cocina</a>
        </div>
        <div class="panel-body">
          @for (o of store.orders(); track o.id) {
            <a
              class="flex items-center justify-between gap-3 py-4 border-b border-stone-100"
              routerLink="/salon"
              [queryParams]="{ pedido: o.id }"
              ><div>
                <strong class="text-sm">{{ store.tableName(o) }}</strong>
                <p class="subtle">#{{ o.id }} · {{ o.customer }}</p>
              </div>
              <app-status [value]="o.state" /><strong class="text-sm">{{
                store.total(o) | money
              }}</strong></a
            >
          } @empty {
            <p class="empty-state">Todo listo para el próximo servicio.</p>
          }
        </div>
      </section>
      <section class="panel">
        <div class="panel-header">
          <h2 class="font-bold">Una mirada al almacén</h2>
          <a class="text-link" routerLink="/inventario">Ver inventario</a>
        </div>
        <div class="panel-body">
          @for (i of store.lowStock(); track i.id) {
            <div class="info-row">
              <div>
                <strong>{{ i.name }}</strong>
                <p class="subtle">Mínimo {{ i.min }} {{ i.unit }}</p>
              </div>
              <span class="pill orange">{{ store.stock(i.id) }} {{ i.unit }}</span>
            </div>
          } @empty {
            <p class="subtle">Todos tus insumos están por encima del mínimo.</p>
          }
          <div class="note-box orange mt-5">
            Planifica la reposición antes del próximo servicio.
          </div>
        </div>
      </section>
    </div>
    <section class="panel mt-6">
      <div class="panel-header">
        <h2 class="font-bold">Últimos movimientos</h2>
        <a class="text-link" routerLink="/auditoria">Ver historial</a>
      </div>
      <div class="panel-body">
        @for (a of recent(); track a.id) {
          <div class="info-row">
            <div>
              <strong>{{ a.entity }} · {{ a.action }}</strong>
              <p class="subtle">{{ a.detail }}</p>
            </div>
            <span class="subtle">{{ a.actor }}</span>
          </div>
        }
      </div>
    </section>`,
})
export class DashboardPage {
  readonly store = inject(RestaurantStore);
  readonly collected = computed(() =>
    this.store.state().payments.reduce((s, p) => s + p.amount, 0),
  );
  readonly recent = computed(() => this.store.state().audit.slice(-4).reverse());
}
