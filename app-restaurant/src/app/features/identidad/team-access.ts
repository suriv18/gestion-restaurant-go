import { ChangeDetectionStrategy, Component, inject, OnInit } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs } from '../../shared/ui/form-dialog';
import { PageHeader } from '../../shared/ui/page-header';
import { UsuarioEquipo } from './data-access/accesos.models';
import { EquipoStore } from './data-access/equipo-store';

const dateTime = new Intl.DateTimeFormat('es-PE', { dateStyle: 'medium', timeStyle: 'short' });

/** «Equipo y accesos» con datos reales del servidor (personas, roles y sedes). */
@Component({
  selector: 'app-team-access',
  imports: [PageHeader],
  providers: [EquipoStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="Un gran equipo, un buen servicio"
      description="Conoce a las personas que hacen que todo funcione."
      eyebrow="EQUIPO Y ACCESOS"
    >
      @if (store.canAdminister() && store.status() === 'ready') {
        <button class="btn primary" (click)="invite()">Nueva persona</button>
      }
    </app-page-header>

    @if (!store.canAdminister()) {
      <p class="note-box orange" role="status">
        No tienes permiso para administrar el equipo. Pídele acceso a la persona que administra tu
        restaurante.
      </p>
    } @else {
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
          <p class="empty" role="status">Cargando el equipo…</p>
        }
        @case ('error') {
          <div class="empty" role="alert">
            <h3>No pudimos cargar el equipo</h3>
            <p>{{ store.loadError() }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        }
        @case ('ready') {
          <div class="user-list">
            @for (u of store.users(); track u.id) {
              <article class="user-card" [class.inactive]="!u.es_activo">
                <div class="user-card-top">
                  <span class="avatar" aria-hidden="true">{{ initials(u) }}</span>
                  <div>
                    <h2 class="text-sm font-bold">{{ displayName(u) }}</h2>
                    @if (u.empleado) {
                      <p>{{ u.empleado.cargo ? u.empleado.cargo + ' · ' : '' }}{{ u.username }}</p>
                    }
                    <p>{{ roleNames(u) }} · {{ branchNames(u) }}</p>
                    @if (u.email) {
                      <p>{{ u.email }}</p>
                    }
                  </div>
                </div>
                <p class="subtle mt-3">{{ lastAccess(u) }}</p>
                @if (!u.es_activo && u.motivo_baja) {
                  <p class="subtle">Motivo de la baja: {{ u.motivo_baja }}</p>
                }
                <div class="card-actions">
                  <span class="pill" [class.green]="u.es_activo">{{
                    u.es_activo ? 'Activo' : 'De baja'
                  }}</span>
                  <div class="inline-actions">
                    @if (u.id === store.currentUserId()) {
                      <small class="subtle">Tu sesión</small>
                    } @else if (u.es_activo) {
                      <button type="button" class="btn" (click)="changeAccess(u)">
                        Cambiar rol
                      </button>
                      <button type="button" class="btn danger" (click)="remove(u)">
                        Dar de baja
                      </button>
                    } @else {
                      <button
                        type="button"
                        class="btn"
                        [disabled]="store.busyId() !== null"
                        (click)="restore(u)"
                      >
                        Restaurar
                      </button>
                    }
                  </div>
                </div>
              </article>
            }
          </div>
        }
      }
    }`,
})
export class TeamAccess implements OnInit {
  readonly store = inject(EquipoStore);
  private readonly dialogs = inject(Dialogs);

  ngOnInit(): void {
    void this.store.load();
  }

  /** El nombre real si la persona tiene empleado vinculado; si no, su usuario. */
  displayName(u: UsuarioEquipo): string {
    return u.empleado ? `${u.empleado.nombres} ${u.empleado.apellidos}` : u.username;
  }

  initials(u: UsuarioEquipo): string {
    const e = u.empleado;
    return (e ? e.nombres[0] + e.apellidos[0] : u.username.slice(0, 2)).toUpperCase();
  }

  roleNames(u: UsuarioEquipo): string {
    return u.roles.map((r) => r.nombre).join(', ') || 'Sin rol';
  }

  branchNames(u: UsuarioEquipo): string {
    return u.sucursales.map((s) => s.nombre).join(', ') || 'Sin sucursal';
  }

  lastAccess(u: UsuarioEquipo): string {
    return u.ultimo_acceso_en
      ? `Último acceso: ${dateTime.format(new Date(u.ultimo_acceso_en))}`
      : 'Aún no ha ingresado';
  }

  invite(): void {
    const roles = this.store.activeRoles();
    const branches = this.store.branches();
    if (!roles.length || !branches.length) {
      this.store.notice.set({
        kind: 'error',
        text: 'Para crear una persona hace falta al menos un rol activo y una sucursal que administres.',
      });
      return;
    }
    // Una clave por formulario abierto: si el envío se reintenta, el servidor no lo duplica.
    const key = newIdempotencyKey();
    const conEmpleado = this.store.canAdministerEmployees();
    this.dialogs.form({
      title: 'Nueva persona del equipo',
      description: 'Crea su acceso. Podrá iniciar sesión de inmediato con esta contraseña.',
      submit: 'Crear acceso',
      fields: [
        // Con permiso para administrar empleados, el alta guarda también los datos de la persona.
        ...(conEmpleado
          ? [
              { key: 'nombres', label: 'Nombres', autocomplete: 'off' },
              { key: 'apellidos', label: 'Apellidos', autocomplete: 'off' },
              {
                key: 'tipo_documento',
                label: 'Tipo de documento',
                type: 'select' as const,
                options: [
                  { value: 'DNI', label: 'DNI' },
                  { value: 'CE', label: 'Carné de extranjería' },
                  { value: 'PASAPORTE', label: 'Pasaporte' },
                ],
              },
              {
                key: 'numero_documento',
                label: 'Número de documento',
                autocomplete: 'off',
                hint: 'DNI de 8 dígitos.',
              },
              { key: 'cargo', label: 'Cargo', required: false, autocomplete: 'off' },
            ]
          : []),
        { key: 'username', label: 'Usuario', autocomplete: 'off' },
        { key: 'email', label: 'Correo', type: 'email', required: false, autocomplete: 'off' },
        {
          key: 'password',
          label: 'Contraseña inicial',
          type: 'password',
          autocomplete: 'new-password',
          hint: 'Al menos 12 caracteres, con letras y números.',
        },
        {
          key: 'role',
          label: 'Rol',
          type: 'select',
          options: roles.map((r) => ({ value: r.id, label: r.nombre })),
        },
        {
          key: 'branch',
          label: 'Sucursal',
          type: 'select',
          options: branches.map((b) => ({ value: b.id, label: b.nombre })),
        },
      ],
      commit: (v) =>
        this.store.invite(
          {
            username: v['username'],
            email: v['email'],
            password: v['password'],
            roleId: v['role'],
            branchId: v['branch'],
            empleado: conEmpleado
              ? {
                  nombres: v['nombres'],
                  apellidos: v['apellidos'],
                  tipo_documento: v['tipo_documento'],
                  numero_documento: v['numero_documento'],
                  cargo: v['cargo'],
                }
              : undefined,
          },
          key,
        ),
    });
  }

  changeAccess(u: UsuarioEquipo): void {
    const roles = this.store.activeRoles();
    if (!roles.length) {
      this.store.notice.set({ kind: 'error', text: 'No hay roles activos para asignar.' });
      return;
    }
    const current = u.roles.find((r) => roles.some((x) => x.id === r.id));
    const branches = this.store.branches();
    // Con varias sedes se conservan tal cual; con una o ninguna se puede elegir.
    const keepBranches = u.sucursales.length > 1;
    const fields = [
      {
        key: 'role',
        label: 'Rol',
        type: 'select' as const,
        value: current?.id ?? roles[0].id,
        options: roles.map((r) => ({ value: r.id, label: r.nombre })),
        hint:
          u.roles.length > 1
            ? 'Esta persona tiene varios roles: al guardar quedará solo con el elegido.'
            : undefined,
      },
      ...(keepBranches
        ? []
        : [
            {
              key: 'branch',
              label: 'Sucursal',
              type: 'select' as const,
              value: u.sucursales[0]?.id ?? branches[0]?.id,
              options: branches.map((b) => ({ value: b.id, label: b.nombre })),
            },
          ]),
    ];
    this.dialogs.form({
      title: 'Cambiar rol de ' + this.displayName(u),
      description: 'El cambio se aplica de inmediato a su próxima acción.',
      fields,
      commit: (v) =>
        this.store.changeAccess(
          u,
          v['role'],
          keepBranches ? u.sucursales.map((s) => s.id) : [v['branch']],
        ),
    });
  }

  remove(u: UsuarioEquipo): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja: ' + this.displayName(u),
      description: 'Se cierran sus sesiones. Se conserva la persona y su historial.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'reason', label: 'Motivo', type: 'textarea' }],
      commit: (v) => this.store.deactivate(u, v['reason'], key),
    });
  }

  restore(u: UsuarioEquipo): void {
    void this.store.restore(u, newIdempotencyKey());
  }
}
