import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { AuthSession } from '../../core/auth/auth-session';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { User } from '../../shared/models/restaurant';
import { PageHeader } from '../../shared/ui/page-header';
import { Dialogs, options } from '../../shared/ui/form-dialog';
import { TeamAccess } from './team-access';
@Component({
  imports: [PageHeader, TeamAccess],
  changeDetection: ChangeDetectionStrategy.OnPush,
  // Con sesión real se muestra el equipo del servidor; en modo demostración, los datos de ejemplo.
  template: `@if (auth.isAuthenticated()) {
      <app-team-access />
    } @else {
      <app-page-header
        title="Un gran equipo, un buen servicio"
        description="Conoce a las personas que hacen que todo funcione."
        eyebrow="EQUIPO Y ACCESOS"
      />
      <p class="note-box orange mb-6">
        Configuración de ejemplo. Los roles no aplican autorización real y la sesión de demostración
        permanece fija.
      </p>
      <div class="user-list">
        @for (u of store.state().users; track u.id) {
          <article class="user-card" [class.inactive]="!u.es_activo">
            <div class="user-card-top">
              <span class="avatar">{{ initials(u.name) }}</span>
              <div>
                <h2 class="text-sm font-bold">{{ u.name }}</h2>
                <p>{{ u.username }} · {{ u.role }}</p>
              </div>
            </div>
            <div class="card-actions">
              <span class="pill" [class.green]="u.es_activo">{{
                u.es_activo ? 'Activo' : 'De baja'
              }}</span>
              <div class="inline-actions">
                @if (u.id !== 1) {
                  @if (u.es_activo) {
                    <button class="btn" (click)="role(u)">Cambiar rol</button
                    ><button class="btn danger" (click)="remove(u)">Dar de baja</button>
                  } @else {
                    <button
                      class="btn"
                      (click)="store.run('restore', { entity: 'users', id: u.id })"
                    >
                      Restaurar
                    </button>
                  }
                } @else {
                  <small class="subtle">Tu sesión</small>
                }
              </div>
            </div>
          </article>
        }
      </div>
    }`,
})
export class UsuariosPage {
  readonly store = inject(RestaurantStore);
  readonly auth = inject(AuthSession);
  private readonly dialogs = inject(Dialogs);
  initials(name: string): string {
    return name
      .split(' ')
      .map((n) => n[0])
      .slice(0, 2)
      .join('');
  }
  role(u: User): void {
    this.dialogs.form({
      title: 'Cambiar rol de ' + u.name,
      description: 'Configuración local de ejemplo.',
      fields: [
        {
          key: 'role',
          label: 'Rol',
          type: 'select',
          value: u.role,
          options: options(['Administradora', 'Mesero', 'Mesera', 'Cocinero', 'Cajero']),
        },
      ],
      commit: (v) => this.store.execute('userRole', { id: u.id, role: v['role'] }),
    });
  }
  remove(u: User): void {
    this.dialogs.form({
      title: 'Dar de baja: ' + u.name,
      description: 'Se conserva el usuario y su historial.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'reason', label: 'Motivo', type: 'textarea' }],
      commit: (v) =>
        this.store.execute('softDelete', { entity: 'users', id: u.id, reason: v['reason'] }),
    });
  }
}
