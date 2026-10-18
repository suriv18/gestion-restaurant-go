import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  viewChild,
} from '@angular/core';
import {
  CdkMenu,
  CdkMenuGroup,
  CdkMenuItem,
  CdkMenuItemRadio,
  CdkMenuTrigger,
} from '@angular/cdk/menu';
import { ConnectedPosition } from '@angular/cdk/overlay';
import { Dialog } from '@angular/cdk/dialog';
import { Router } from '@angular/router';
import { AuthSession } from '../../core/auth/auth-session';
import { DemoMode } from '../../core/auth/demo-mode';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { ThemeService, ThemePreference } from '../../core/theme/theme-service';
import { DetailDialog } from './form-dialog';
import { Icon } from './icon';

@Component({
  selector: 'app-user-menu',
  imports: [CdkMenu, CdkMenuGroup, CdkMenuItem, CdkMenuItemRadio, CdkMenuTrigger, Icon],
  templateUrl: './user-menu.html',
  styleUrl: './user-menu.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UserMenu {
  readonly theme = inject(ThemeService);
  private readonly store = inject(RestaurantStore);
  private readonly auth = inject(AuthSession);
  private readonly demo = inject(DemoMode);
  private readonly router = inject(Router);
  private readonly dialog = inject(Dialog);
  private readonly trigger = viewChild.required<CdkMenuTrigger>('accountTrigger');
  private readonly button = viewChild.required<ElementRef<HTMLButtonElement>>('accountButton');
  /** Persona de ejemplo; solo se usa en modo demostración. */
  private readonly demoUser = computed(() =>
    this.store.state().users.find((user) => user.id === 1),
  );
  /** Usuario autenticado en el servidor, o null en modo demostración. */
  readonly session = this.auth.user;
  readonly displayName = computed(
    () => this.session()?.username ?? this.demoUser()?.name ?? 'Usuario',
  );
  readonly subtitle = computed(() => {
    const s = this.session();
    return s
      ? `Sesión activa · ${s.permissions.length} permisos`
      : `${this.demoUser()?.role ?? ''} · Demo`;
  });
  readonly initials = computed(() => {
    const s = this.session();
    if (s) return s.username.slice(0, 2).toUpperCase();
    return (
      this.demoUser()
        ?.name.split(' ')
        .slice(0, 2)
        .map((part) => part[0])
        .join('') ?? 'VT'
    );
  });
  readonly themes: { value: ThemePreference; label: string; icon: string }[] = [
    { value: 'light', label: 'Claro', icon: 'sun' },
    { value: 'dark', label: 'Oscuro', icon: 'moon' },
    { value: 'system', label: 'Sistema', icon: 'monitor' },
  ];
  readonly positions: ConnectedPosition[] = [
    { originX: 'end', originY: 'bottom', overlayX: 'end', overlayY: 'top', offsetY: 10 },
    { originX: 'end', originY: 'top', overlayX: 'end', overlayY: 'bottom', offsetY: -10 },
  ];

  profile(): void {
    this.trigger().close();
    const s = this.session();
    this.dialog.open(DetailDialog, {
      panelClass: 'app-dialog',
      ariaModal: true,
      ariaLabelledBy: 'dialog-title',
      ariaDescribedBy: 'dialog-description',
      restoreFocus: this.button().nativeElement,
      data: {
        title: 'Mi perfil',
        description: s ? 'Datos de tu cuenta.' : 'Datos del usuario de demostración.',
        rows: s
          ? [
              { label: 'Usuario', value: s.username },
              { label: 'Identificador', value: s.id },
              { label: 'Sucursales', value: s.branches.join(', ') || 'Sin sucursales asignadas' },
              { label: 'Permisos', value: s.permissions.join(' · ') || 'Sin permisos asignados' },
              { label: 'Sesión', value: 'Autenticada en el servidor' },
            ]
          : [
              { label: 'Nombre', value: this.demoUser()?.name ?? 'Valeria Torres' },
              { label: 'Usuario', value: this.demoUser()?.username ?? 'valeria' },
              { label: 'Rol', value: this.demoUser()?.role ?? 'Administradora' },
              { label: 'Restaurante', value: this.store.state().settings.name },
              { label: 'Sucursal', value: this.store.state().settings.branch },
              { label: 'Sesión', value: 'Demostración · sin autenticación real' },
            ],
      },
    });
  }

  /** Cierra la sesión (revoca en el servidor) o sale de la demo, y vuelve al login. */
  async logout(): Promise<void> {
    this.trigger().close();
    this.store.notice.set('');
    await this.auth.logout();
    this.demo.leave();
    await this.router.navigateByUrl('/login');
  }
}
