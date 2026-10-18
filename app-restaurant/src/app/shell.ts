import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet, NavigationEnd } from '@angular/router';
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop';
import { filter, map } from 'rxjs';
import { RestaurantStore } from './core/demo/restaurant-store';
import { Icon } from './shared/ui/icon';
import { Dialogs } from './shared/ui/form-dialog';
import { UserMenu } from './shared/ui/user-menu';
export const navigation = [
  {
    label: 'OPERACIÓN',
    items: [
      { path: '/dashboard', label: 'Resumen', icon: 'grid' },
      { path: '/salon', label: 'Salón y pedidos', icon: 'table' },
      { path: '/cocina', label: 'Cocina', icon: 'chef' },
      { path: '/reservas', label: 'Reservas', icon: 'calendar' },
      { path: '/delivery', label: 'Delivery', icon: 'bike' },
    ],
  },
  {
    label: 'GESTIÓN',
    items: [
      { path: '/caja', label: 'Caja y cobros', icon: 'wallet' },
      { path: '/compras', label: 'Compras', icon: 'cart' },
      { path: '/inventario', label: 'Inventario', icon: 'box' },
      { path: '/carta', label: 'Carta y recetas', icon: 'book' },
      { path: '/comprobantes', label: 'Comprobantes', icon: 'receipt' },
    ],
  },
  {
    label: 'ADMINISTRACIÓN',
    items: [
      { path: '/contactos', label: 'Clientes y proveedores', icon: 'users' },
      { path: '/usuarios', label: 'Equipo y accesos', icon: 'shield' },
      { path: '/auditoria', label: 'Auditoría', icon: 'clock' },
      { path: '/configuracion', label: 'Configuración', icon: 'settings' },
    ],
  },
];
@Component({
  selector: 'app-shell',
  imports: [RouterOutlet, RouterLink, RouterLinkActive, Icon, UserMenu],
  templateUrl: './shell.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Shell {
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  private readonly router = inject(Router);
  readonly menuOpen = signal(false);
  readonly groups = navigation;
  private readonly path = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map((e) => e.urlAfterRedirects.split('?')[0]),
    ),
    { initialValue: this.router.url.split('?')[0] },
  );
  readonly currentPage = computed(
    () =>
      navigation.flatMap((g) => g.items).find((i) => i.path === this.path())?.label ??
      'Página no encontrada',
  );
  readonly date = new Intl.DateTimeFormat('es-PE', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  }).format(new Date());
  constructor() {
    this.router.events
      .pipe(
        filter((e) => e instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(() => {
        this.menuOpen.set(false);
      });
  }
  help(): void {
    this.dialogs.detail({
      title: 'Tu restaurante, en acción',
      description: 'La experiencia Mesa, ahora en Angular.',
      rows: [
        {
          label: '01 · Abre una mesa',
          value: 'Agrega productos y envía la comanda desde Salón y pedidos.',
        },
        {
          label: '02 · Prepara y sirve',
          value: 'Avanza los productos en Cocina. Los insumos se descuentan una sola vez.',
        },
        {
          label: '03 · Cobra y cierra',
          value: 'Combina métodos de pago y libera la mesa al completar la atención.',
        },
        {
          label: 'Modo demostración',
          value:
            'Sin autenticación, cobros bancarios ni emisión tributaria. Los datos permanecen en este navegador.',
        },
      ],
    });
  }
  alerts(): void {
    this.dialogs.detail({
      title: 'Lo que necesita tu atención',
      description: 'Un vistazo a la operación actual.',
      rows: [
        { label: 'Insumos por reponer', value: String(this.store.lowStock().length) },
        { label: 'Productos listos para servir', value: String(this.store.ready()) },
        { label: 'Pedidos abiertos', value: String(this.store.orders().length) },
      ],
    });
  }
}
