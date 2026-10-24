import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { AuthSession } from '../../core/auth/auth-session';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs } from '../../shared/ui/form-dialog';
import { PageHeader } from '../../shared/ui/page-header';
import { Empleado } from './data-access/empleados.models';
import { EmpleadosStore, nombre, ValoresEmpleado } from './data-access/empleados-store';

const fecha = new Intl.DateTimeFormat('es-PE', { dateStyle: 'medium', timeZone: 'UTC' });

/** Las fechas llegan como AAAA-MM-DD: se leen como UTC para que la zona horaria no las corra un día. */
const verFecha = (iso: string): string => fecha.format(new Date(`${iso}T00:00:00Z`));

const TIPOS_DOCUMENTO = [
  { value: 'DNI', label: 'DNI' },
  { value: 'CE', label: 'Carné de extranjería' },
  { value: 'PASAPORTE', label: 'Pasaporte' },
  { value: 'RUC', label: 'RUC' },
];

/** «Empleados»: el personal del restaurante y su acceso al sistema. */
@Component({
  selector: 'app-empleados-page',
  imports: [PageHeader],
  providers: [EmpleadosStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="Quienes hacen que todo funcione"
      description="El personal del restaurante y su acceso al sistema."
      eyebrow="EMPLEADOS"
    >
      @if (store.puedeAdministrar() && store.status() === 'ready') {
        <button type="button" class="btn primary" (click)="form()">+ Nuevo empleado</button>
      }
    </app-page-header>

    @if (!auth.isAuthenticated()) {
      <p class="note-box orange" role="status">
        Los empleados se administran con la sesión del restaurante. Inicia sesión para verlos; la
        demostración no incluye personal.
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
          <p class="empty" role="status">Cargando los empleados…</p>
        }
        @case ('error') {
          <div class="empty" role="alert">
            <h3>No pudimos cargar los empleados</h3>
            <p>{{ store.loadError() }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        }
        @case ('ready') {
          @if (!store.puedeVer()) {
            <p class="note-box orange" role="status">
              No tienes permiso para ver a los empleados. Pídele acceso a quien administra tu
              equipo.
            </p>
          } @else {
            <div class="section-toolbar">
              <label class="search"
                ><input
                  aria-label="Buscar empleado"
                  placeholder="Buscar nombre o documento..."
                  (input)="query.set($any($event.target).value)"
              /></label>
              <label class="flex gap-2 text-xs items-center"
                ><input
                  type="checkbox"
                  (change)="inactive.set($any($event.target).checked)"
                />Incluir bajas</label
              >
            </div>
            <div class="panel table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Nombre</th>
                    <th>Documento</th>
                    <th>Acceso</th>
                    <th>Ingreso</th>
                    <th>Estado</th>
                    <th>Acciones</th>
                  </tr>
                </thead>
                <tbody>
                  @for (e of filas(); track e.id) {
                    <tr>
                      <td>
                        <strong>{{ nombre(e) }}</strong>
                        @if (e.cargo) {
                          <small>{{ e.cargo }}</small>
                        }
                      </td>
                      <td>{{ e.tipo_documento }} {{ e.numero_documento }}</td>
                      <td>{{ e.username ?? 'Sin acceso' }}</td>
                      <td>
                        {{ verFecha(e.fecha_ingreso) }}
                        @if (e.fecha_cese) {
                          <small>Cese: {{ verFecha(e.fecha_cese) }}</small>
                        }
                      </td>
                      <td>
                        <span class="pill" [class.green]="e.es_activo">{{
                          e.es_activo ? 'Activo' : 'De baja'
                        }}</span>
                        @if (!e.es_activo && e.motivo_eliminacion) {
                          <small>Motivo: {{ e.motivo_eliminacion }}</small>
                        }
                      </td>
                      <td>
                        @if (store.puedeAdministrar()) {
                          <div class="inline-actions">
                            @if (e.es_activo) {
                              <button type="button" class="btn" (click)="form(e)">Editar</button>
                              <button type="button" class="btn danger" (click)="baja(e)">
                                Dar de baja
                              </button>
                            } @else {
                              <button
                                type="button"
                                class="btn"
                                [disabled]="store.busyId() !== null"
                                (click)="restaurar(e)"
                              >
                                Restaurar
                              </button>
                            }
                          </div>
                        }
                      </td>
                    </tr>
                  } @empty {
                    <tr>
                      <td colspan="6">
                        <div class="empty-state">No hay empleados que coincidan.</div>
                      </td>
                    </tr>
                  }
                </tbody>
              </table>
            </div>
          }
        }
      }
    }`,
})
export class EmpleadosPage implements OnInit {
  readonly store = inject(EmpleadosStore);
  readonly auth = inject(AuthSession);
  private readonly dialogs = inject(Dialogs);

  readonly nombre = nombre;
  readonly verFecha = verFecha;
  readonly query = signal('');
  readonly inactive = signal(false);

  readonly filas = computed(() => {
    const buscado = this.query().trim().toLowerCase();
    return this.store
      .empleados()
      .filter(
        (e) =>
          (this.inactive() || e.es_activo) &&
          (nombre(e) + ' ' + e.numero_documento).toLowerCase().includes(buscado),
      );
  });

  ngOnInit(): void {
    if (this.auth.isAuthenticated()) void this.store.load();
  }

  form(e?: Empleado): void {
    // Una clave por formulario abierto: si el envío se reintenta, el servidor no lo duplica.
    const key = newIdempotencyKey();
    const vinculables = this.store.puedeVincular() ? this.store.usuariosVinculables(e) : null;
    this.dialogs.form({
      title: e ? 'Editar empleado' : 'Nuevo empleado',
      description: 'Los datos personales del empleado; su acceso al sistema es opcional.',
      fields: [
        { key: 'nombres', label: 'Nombres', value: e?.nombres, autocomplete: 'off' },
        { key: 'apellidos', label: 'Apellidos', value: e?.apellidos, autocomplete: 'off' },
        {
          key: 'tipo_documento',
          label: 'Tipo de documento',
          type: 'select',
          value: e?.tipo_documento ?? 'DNI',
          options: TIPOS_DOCUMENTO,
        },
        {
          key: 'numero_documento',
          label: 'Número de documento',
          value: e?.numero_documento,
          hint: 'DNI de 8 dígitos.',
          autocomplete: 'off',
        },
        { key: 'cargo', label: 'Cargo', value: e?.cargo ?? '', required: false },
        {
          key: 'telefono',
          label: 'Teléfono',
          type: 'tel',
          value: e?.telefono ?? '',
          required: false,
        },
        {
          key: 'fecha_ingreso',
          label: 'Fecha de ingreso',
          type: 'date',
          value: e?.fecha_ingreso ?? '',
          required: false,
          hint: e ? undefined : 'Si la dejas vacía, se usa la de hoy.',
        },
        ...(e
          ? [
              {
                key: 'fecha_cese',
                label: 'Fecha de cese',
                type: 'date' as const,
                value: e.fecha_cese ?? '',
                required: false,
              },
            ]
          : []),
        ...(vinculables
          ? [
              {
                key: 'usuario_id',
                label: 'Acceso al sistema',
                type: 'select' as const,
                required: false,
                value: e?.usuario_id ?? '',
                options: [
                  { value: '', label: 'Sin acceso' },
                  ...vinculables.map((u) => ({ value: u.id, label: u.username })),
                ],
              },
            ]
          : []),
      ],
      commit: (v) => this.store.guardar(e, v as unknown as ValoresEmpleado, key),
    });
  }

  baja(e: Empleado): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja: ' + nombre(e),
      description:
        'Se conserva su historial y podrás restaurarlo. Si tiene acceso al sistema, primero hay que dar de baja su usuario en Equipo y accesos.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'motivo', label: 'Motivo', type: 'textarea' }],
      commit: (v) => this.store.baja(e, v['motivo'], key),
    });
  }

  restaurar(e: Empleado): void {
    void this.store.restaurar(e, newIdempotencyKey());
  }
}
