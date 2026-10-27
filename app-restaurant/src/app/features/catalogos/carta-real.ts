import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
  viewChild,
} from '@angular/core';
import { PageHeader } from '../../shared/ui/page-header';
import { CartaStore } from './data-access/carta-store';
import { CategoriasSection } from './categorias-section';
import { ProductosSection } from './productos-section';

type Pestana = 'productos' | 'categorias';

/** «Carta y recetas» con datos reales del servidor (las recetas llegan con inventario, I5). */
@Component({
  selector: 'app-carta-real',
  imports: [PageHeader, ProductosSection, CategoriasSection],
  providers: [CartaStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="El sabor de tu restaurante"
      description="Tu carta, con sus categorías y precios, siempre al día en cada sede."
      eyebrow="CARTA Y RECETAS"
    >
      @if (store.puedeAdministrarCarta() && store.status() === 'ready' && !store.sinAcceso()) {
        <button type="button" class="btn primary" (click)="nuevo()">
          {{ pestana() === 'productos' ? '+ Nuevo producto' : '+ Nueva categoría' }}
        </button>
      }
    </app-page-header>

    @if (store.notice(); as notice) {
      <p
        class="note-box mb-4"
        [class.orange]="notice.kind === 'error'"
        [attr.role]="notice.kind === 'error' ? 'alert' : 'status'"
      >
        <span>{{ notice.text }}</span>
        <button type="button" class="btn" (click)="store.dismissNotice()">Entendido</button>
      </p>
    }

    @switch (store.status()) {
      @case ('loading') {
        <p class="empty" role="status">Cargando la carta…</p>
      }
      @case ('error') {
        <div class="empty" role="alert">
          <h3>No pudimos cargar la carta</h3>
          <p>{{ store.loadError() }}</p>
          <button type="button" class="btn primary" (click)="recargar()">Reintentar</button>
        </div>
      }
      @case ('ready') {
        @if (store.sinAcceso()) {
          <p class="note-box orange" role="status">
            No tienes permiso para ver la carta. Pídele acceso a quien administra tu equipo.
          </p>
        } @else {
          <div class="tabs mb-4" role="tablist" aria-label="Secciones de la carta">
            <button
              type="button"
              role="tab"
              id="tab-productos"
              class="tab"
              aria-controls="panel-carta"
              [class.active]="pestana() === 'productos'"
              [attr.aria-selected]="pestana() === 'productos'"
              (click)="pestana.set('productos')"
            >
              Productos
            </button>
            <button
              type="button"
              role="tab"
              id="tab-categorias"
              class="tab"
              aria-controls="panel-carta"
              [class.active]="pestana() === 'categorias'"
              [attr.aria-selected]="pestana() === 'categorias'"
              (click)="pestana.set('categorias')"
            >
              Categorías
            </button>
          </div>
          <div id="panel-carta" role="tabpanel" [attr.aria-labelledby]="'tab-' + pestana()">
            @if (pestana() === 'productos') {
              <app-productos-section />
            } @else {
              <app-categorias-section />
            }
          </div>
        }
      }
    }`,
})
export class CartaReal implements OnInit {
  readonly store = inject(CartaStore);
  /** Pestaña activa: se recuerda mientras la pantalla está abierta. */
  readonly pestana = signal<Pestana>('productos');

  private readonly productos = viewChild(ProductosSection);
  private readonly categorias = viewChild(CategoriasSection);

  ngOnInit(): void {
    void this.store.load();
  }

  recargar(): void {
    void this.store.load();
  }

  nuevo(): void {
    if (this.pestana() === 'productos') this.productos()?.nuevo();
    else this.categorias()?.nueva();
  }
}
