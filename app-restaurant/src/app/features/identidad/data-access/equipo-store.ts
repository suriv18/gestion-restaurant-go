import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem } from '../../../core/http/problem';
import { AccesosApi } from './accesos-api';
import { Rol, SucursalRef, UsuarioEquipo } from './accesos.models';

export const ADMIN_PERMISSION = 'USUARIO_ADMINISTRAR';

export interface Notice {
  kind: 'ok' | 'error';
  text: string;
}

/**
 * Estado de la pantalla «Equipo y accesos». Se provee en la página (no en root): vive y muere
 * con ella. Todo lo que muestra viene del servidor; el navegador no decide permisos.
 *
 * Las acciones devuelven un texto de error (o nada si salió bien) para que el diálogo que las
 * lanzó lo muestre sin cerrarse.
 */
@Injectable()
export class EquipoStore {
  private readonly api = inject(AccesosApi);
  private readonly auth = inject(AuthSession);

  readonly status = signal<'loading' | 'ready' | 'error'>('loading');
  readonly loadError = signal('');
  readonly users = signal<UsuarioEquipo[]>([]);
  readonly roles = signal<Rol[]>([]);
  readonly branches = signal<SucursalRef[]>([]);
  readonly notice = signal<Notice | null>(null);
  readonly busyId = signal<string | null>(null);

  readonly canAdminister = computed(() => this.auth.hasPermission(ADMIN_PERMISSION));
  readonly activeRoles = computed(() => this.roles().filter((r) => r.es_activo));
  readonly currentUserId = computed(() => this.auth.user()?.id ?? null);

  /** Carga el equipo, los roles y las sedes asignables. */
  async load(): Promise<void> {
    if (!this.canAdminister()) {
      this.status.set('ready');
      return;
    }
    this.status.set('loading');
    this.loadError.set('');
    try {
      const [users, roles, branches] = await Promise.all([
        collect((c) => firstValueFrom(this.api.usuarios(c))),
        collect((c) => firstValueFrom(this.api.roles(c))),
        firstValueFrom(this.api.sucursalesAsignables()),
      ]);
      this.users.set(users);
      this.roles.set(roles);
      this.branches.set(branches.items);
      this.status.set('ready');
    } catch (error) {
      this.loadError.set(describeProblem(error));
      this.status.set('error');
    }
  }

  /** Alta de una persona y asignación de su rol y sede. Reintentable con la misma clave. */
  async invite(
    v: { username: string; email: string; password: string; roleId: string; branchId: string },
    key: string,
  ): Promise<string | undefined> {
    let created: { id: string };
    try {
      created = await firstValueFrom(
        this.api.crearUsuario(
          { username: v.username.trim(), email: v.email.trim() || undefined, password: v.password },
          key,
        ),
      );
    } catch (error) {
      return describeProblem(error);
    }
    try {
      await firstValueFrom(this.api.asignar(created.id, [v.roleId], [v.branchId]));
    } catch (error) {
      await this.refreshUsers();
      // Con la misma clave, reintentar repite el alta ya confirmada y vuelve a asignar.
      return `La persona se creó, pero falta asignarle su acceso: ${describeProblem(error)} Puedes reintentar.`;
    }
    await this.refreshUsers();
    this.say('ok', `Se creó el acceso de ${v.username.trim()}.`);
    return undefined;
  }

  /** Cambia el rol conservando (o fijando) las sedes. */
  async changeAccess(
    user: UsuarioEquipo,
    roleId: string,
    branchIds: string[],
  ): Promise<string | undefined> {
    try {
      await firstValueFrom(this.api.asignar(user.id, [roleId], branchIds));
    } catch (error) {
      return describeProblem(error);
    }
    await this.refreshUsers();
    this.say('ok', `Se actualizó el acceso de ${user.username}.`);
    return undefined;
  }

  async deactivate(user: UsuarioEquipo, reason: string, key: string): Promise<string | undefined> {
    try {
      await firstValueFrom(this.api.baja(user.id, reason.trim(), key));
    } catch (error) {
      return describeProblem(error);
    }
    await this.refreshUsers();
    this.say('ok', `${user.username} quedó de baja y sus sesiones se cerraron.`);
    return undefined;
  }

  /** Restaura desde la lista (sin diálogo): informa el resultado con un aviso. */
  async restore(user: UsuarioEquipo, key: string): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(user.id);
    try {
      await firstValueFrom(this.api.restaurar(user.id, key));
      await this.refreshUsers();
      this.say('ok', `${user.username} volvió a estar activo. Deberá iniciar sesión de nuevo.`);
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  private async refreshUsers(): Promise<void> {
    try {
      this.users.set(await collect((c) => firstValueFrom(this.api.usuarios(c))));
    } catch (error) {
      this.say('error', `No se pudo actualizar la lista: ${describeProblem(error)}`);
    }
  }

  private say(kind: Notice['kind'], text: string): void {
    this.notice.set({ kind, text });
  }
}
