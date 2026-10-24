import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs } from '../../shared/ui/form-dialog';
import { PageHeader } from '../../shared/ui/page-header';
import { Cliente, Proveedor, TIPOS_DOCUMENTO } from './data-access/directorio.models';
import { DirectorioStore, ValoresCliente, ValoresProveedor } from './data-access/directorio-store';

type Tipo = 'clientes' | 'proveedores';

/** Una fila de la tabla, común a clientes y proveedores. */
interface Fila {
  id: string;
  nombre: string;
  documento: string;
  telefono: string | null;
  email: string | null;
  es_activo: boolean;
  motivo: string | null;
}

const documentoDe = (tipo: string | null, numero: string | null): string =>
  tipo && numero ? `${tipo} ${numero}` : 'Sin documento';

/** «Clientes y proveedores» con datos reales del servidor. */
@Component({
  selector: 'app-contactos-real',
  imports: [PageHeader],
  providers: [DirectorioStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="Las personas detrás de cada día"
      description="Tus clientes y proveedores, siempre a mano."
      eyebrow="DIRECTORIO"
    >
      @if (puedeAdministrar() && store.status() === 'ready') {
        <button type="button" class="btn primary" (click)="nuevo()">
          {{ tipo() === 'clientes' ? '+ Nuevo cliente' : '+ Nuevo proveedor' }}
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
        <p class="empty" role="status">Cargando el directorio…</p>
      }
      @case ('error') {
        <div class="empty" role="alert">
          <h3>No pudimos cargar el directorio</h3>
          <p>{{ store.loadError() }}</p>
          <button type="button" class="btn primary" (click)="recargar()">Reintentar</button>
        </div>
      }
      @case ('ready') {
        @if (store.sinAcceso()) {
          <p class="note-box orange" role="status">
            No tienes permiso para ver clientes ni proveedores. Pídele acceso a quien administra tu
            equipo.
          </p>
        } @else {
          <div class="section-toolbar">
            <div class="tabs">
              @if (store.puedeVerClientes()) {
                <button
                  type="button"
                  class="tab"
                  [class.active]="tipo() === 'clientes'"
                  (click)="tipo.set('clientes')"
                >
                  Clientes
                </button>
              }
              @if (store.puedeVerProveedores()) {
                <button
                  type="button"
                  class="tab"
                  [class.active]="tipo() === 'proveedores'"
                  (click)="tipo.set('proveedores')"
                >
                  Proveedores
                </button>
              }
            </div>
            <label class="search"
              ><input
                aria-label="Buscar contacto"
                placeholder="Buscar nombre o documento..."
                (input)="query.set($any($event.target).value)"
            /></label>
            <label class="flex gap-2 text-xs items-center"
              ><input type="checkbox" (change)="inactive.set($any($event.target).checked)" />Incluir
              bajas</label
            >
          </div>
          <div class="panel table-scroll">
            <table>
              <thead>
                <tr>
                  <th>Nombre</th>
                  <th>Documento</th>
                  <th>Contacto</th>
                  <th>Estado</th>
                  <th>Acciones</th>
                </tr>
              </thead>
              <tbody>
                @for (f of filas(); track f.id) {
                  <tr>
                    <td>
                      <strong>{{ f.nombre }}</strong>
                    </td>
                    <td>{{ f.documento }}</td>
                    <td>
                      {{ f.telefono }}<small>{{ f.email }}</small>
                    </td>
                    <td>
                      <span class="pill" [class.green]="f.es_activo">{{
                        f.es_activo ? 'Activo' : 'De baja'
                      }}</span>
                      @if (!f.es_activo && f.motivo) {
                        <small>Motivo: {{ f.motivo }}</small>
                      }
                    </td>
                    <td>
                      @if (puedeAdministrar()) {
                        <div class="inline-actions">
                          @if (f.es_activo) {
                            <button type="button" class="btn" (click)="editar(f)">Editar</button>
                            <button type="button" class="btn danger" (click)="baja(f)">
                              Dar de baja
                            </button>
                          } @else {
                            <button
                              type="button"
                              class="btn"
                              [disabled]="store.busyId() !== null"
                              (click)="restaurar(f)"
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
                    <td colspan="5">
                      <div class="empty-state">No hay contactos que coincidan.</div>
                    </td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        }
      }
    }`,
})
export class ContactosReal implements OnInit {
  readonly store = inject(DirectorioStore);
  private readonly dialogs = inject(Dialogs);

  readonly tipo = signal<Tipo>('clientes');
  readonly query = signal('');
  readonly inactive = signal(false);

  readonly puedeAdministrar = computed(() =>
    this.tipo() === 'clientes'
      ? this.store.puedeAdministrarClientes()
      : this.store.puedeAdministrarProveedores(),
  );

  readonly filas = computed<Fila[]>(() => {
    const todas =
      this.tipo() === 'clientes'
        ? this.store.clientes().map(filaDeCliente)
        : this.store.proveedores().map(filaDeProveedor);
    const buscado = this.query().trim().toLowerCase();
    return todas.filter(
      (f) =>
        (this.inactive() || f.es_activo) &&
        (f.nombre + ' ' + f.documento).toLowerCase().includes(buscado),
    );
  });

  async ngOnInit(): Promise<void> {
    await this.store.load();
    // Si no puede ver clientes pero sí proveedores, la pantalla abre en Proveedores.
    if (!this.store.puedeVerClientes() && this.store.puedeVerProveedores()) {
      this.tipo.set('proveedores');
    }
  }

  recargar(): void {
    void this.store.load();
  }

  nuevo(): void {
    if (this.tipo() === 'clientes') this.formCliente();
    else this.formProveedor();
  }

  editar(f: Fila): void {
    if (this.tipo() === 'clientes') {
      this.formCliente(this.store.clientes().find((c) => c.id === f.id));
    } else {
      this.formProveedor(this.store.proveedores().find((p) => p.id === f.id));
    }
  }

  baja(f: Fila): void {
    // Una clave por formulario abierto: si el envío se reintenta, el servidor no lo duplica.
    const key = newIdempotencyKey();
    const cliente =
      this.tipo() === 'clientes' ? this.store.clientes().find((c) => c.id === f.id) : undefined;
    const proveedor =
      this.tipo() === 'proveedores'
        ? this.store.proveedores().find((p) => p.id === f.id)
        : undefined;
    this.dialogs.form({
      title: 'Dar de baja: ' + f.nombre,
      description: 'Podrás restaurarlo sin perder su historial. El documento sigue reservado.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'motivo', label: 'Motivo', type: 'textarea' }],
      commit: (v) =>
        cliente
          ? this.store.bajaCliente(cliente, v['motivo'], key)
          : proveedor
            ? this.store.bajaProveedor(proveedor, v['motivo'], key)
            : undefined,
    });
  }

  restaurar(f: Fila): void {
    const key = newIdempotencyKey();
    if (this.tipo() === 'clientes') {
      const c = this.store.clientes().find((x) => x.id === f.id);
      if (c) void this.store.restaurarCliente(c, key);
    } else {
      const p = this.store.proveedores().find((x) => x.id === f.id);
      if (p) void this.store.restaurarProveedor(p, key);
    }
  }

  private formCliente(c?: Cliente): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: c ? 'Editar cliente' : 'Nuevo cliente',
      description: 'El documento es opcional; si lo indicas, identifica al cliente y no se repite.',
      fields: [
        { key: 'nombre', label: 'Nombre / razón social', value: c?.nombre_razon_social },
        {
          key: 'tipo_documento',
          label: 'Tipo de documento',
          type: 'select',
          required: false,
          value: c?.tipo_documento ?? '',
          options: [{ value: '', label: 'Sin documento' }, ...TIPOS_DOCUMENTO],
        },
        {
          key: 'numero_documento',
          label: 'Número de documento',
          required: false,
          value: c?.numero_documento ?? '',
          hint: 'DNI de 8 dígitos o RUC de 11.',
        },
        {
          key: 'telefono',
          label: 'Teléfono',
          type: 'tel',
          value: c?.telefono ?? '',
          required: false,
        },
        {
          key: 'email',
          label: 'Correo electrónico',
          type: 'email',
          value: c?.email ?? '',
          required: false,
        },
        {
          key: 'direccion',
          label: 'Dirección',
          type: 'textarea',
          value: c?.direccion ?? '',
          required: false,
        },
      ],
      commit: (v) => this.store.guardarCliente(c, v as unknown as ValoresCliente, key),
    });
  }

  private formProveedor(p?: Proveedor): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: p ? 'Editar proveedor' : 'Nuevo proveedor',
      description: 'El documento identifica al proveedor y se conserva con su historial.',
      fields: [
        { key: 'razon_social', label: 'Razón social', value: p?.razon_social },
        {
          key: 'tipo_documento',
          label: 'Tipo de documento',
          type: 'select',
          value: p?.tipo_documento ?? 'RUC',
          options: TIPOS_DOCUMENTO,
        },
        {
          key: 'numero_documento',
          label: 'Número de documento',
          value: p?.numero_documento,
          hint: 'RUC de 11 dígitos.',
        },
        {
          key: 'contacto_nombre',
          label: 'Persona de contacto',
          value: p?.contacto_nombre ?? '',
          required: false,
        },
        {
          key: 'telefono',
          label: 'Teléfono',
          type: 'tel',
          value: p?.telefono ?? '',
          required: false,
        },
        {
          key: 'email',
          label: 'Correo electrónico',
          type: 'email',
          value: p?.email ?? '',
          required: false,
        },
        {
          key: 'direccion',
          label: 'Dirección',
          type: 'textarea',
          value: p?.direccion ?? '',
          required: false,
        },
      ],
      commit: (v) => this.store.guardarProveedor(p, v as unknown as ValoresProveedor, key),
    });
  }
}

function filaDeCliente(c: Cliente): Fila {
  return {
    id: c.id,
    nombre: c.nombre_razon_social,
    documento: documentoDe(c.tipo_documento, c.numero_documento),
    telefono: c.telefono,
    email: c.email,
    es_activo: c.es_activo,
    motivo: c.motivo_eliminacion,
  };
}

function filaDeProveedor(p: Proveedor): Fila {
  return {
    id: p.id,
    nombre: p.razon_social,
    documento: documentoDe(p.tipo_documento, p.numero_documento),
    telefono: p.telefono,
    email: p.email,
    es_activo: p.es_activo,
    motivo: p.motivo_eliminacion,
  };
}
