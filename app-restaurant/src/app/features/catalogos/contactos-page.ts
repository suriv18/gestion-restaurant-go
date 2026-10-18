import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Contact } from '../../shared/models/restaurant';
import { PageHeader } from '../../shared/ui/page-header';
import { Dialogs } from '../../shared/ui/form-dialog';
@Component({
  imports: [PageHeader],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="Las personas detrás de cada día"
      description="Tus clientes y proveedores, siempre a mano."
      eyebrow="DIRECTORIO"
      ><button class="btn primary" (click)="edit()">+ Nuevo contacto</button></app-page-header
    >
    <div class="section-toolbar">
      <div class="tabs">
        <button class="tab" [class.active]="type() === 'customers'" (click)="type.set('customers')">
          Clientes</button
        ><button
          class="tab"
          [class.active]="type() === 'suppliers'"
          (click)="type.set('suppliers')"
        >
          Proveedores
        </button>
      </div>
      <label class="search"
        ><input
          aria-label="Buscar contacto"
          placeholder="Buscar nombre o documento..."
          (input)="query.set($any($event.target).value)" /></label
      ><label class="flex gap-2 text-xs items-center"
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
          @for (c of contacts(); track c.id) {
            <tr>
              <td>
                <strong>{{ c.name }}</strong>
              </td>
              <td>{{ c.document }}</td>
              <td>
                {{ c.phone }}<small>{{ c.email }}</small>
              </td>
              <td>
                <span class="pill" [class.green]="c.es_activo">{{
                  c.es_activo ? 'Activo' : 'De baja'
                }}</span>
              </td>
              <td>
                <div class="inline-actions">
                  @if (c.es_activo) {
                    <button class="btn" (click)="edit(c)">Editar</button
                    ><button class="btn danger" (click)="remove(c)">Dar de baja</button>
                  } @else {
                    <button
                      class="btn"
                      (click)="store.run('restore', { entity: type(), id: c.id })"
                    >
                      Restaurar
                    </button>
                  }
                </div>
              </td>
            </tr>
          } @empty {
            <tr>
              <td colspan="5"><div class="empty-state">No hay contactos que coincidan.</div></td>
            </tr>
          }
        </tbody>
      </table>
    </div>`,
})
export class ContactosPage {
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  readonly type = signal<'customers' | 'suppliers'>('customers');
  readonly query = signal('');
  readonly inactive = signal(false);
  readonly contacts = computed(() => {
    const contacts = this.store.state()[this.type()];
    return contacts.filter(
      (c) =>
        (this.inactive() || c.es_activo) &&
        (c.name + ' ' + c.document).toLowerCase().includes(this.query().toLowerCase()),
    );
  });
  edit(c?: Contact): void {
    const entity = this.type();
    this.dialogs.form({
      title: c ? 'Editar contacto' : entity === 'customers' ? 'Nuevo cliente' : 'Nuevo proveedor',
      description: 'El documento identifica al contacto y se conserva con su historial.',
      fields: [
        { key: 'name', label: 'Nombre / razón social', value: c?.name },
        {
          key: 'document',
          label: entity === 'suppliers' ? 'RUC (11 dígitos)' : 'Documento',
          value: c?.document,
        },
        { key: 'phone', label: 'Teléfono', type: 'tel', value: c?.phone, required: false },
        {
          key: 'email',
          label: 'Correo electrónico',
          type: 'email',
          value: c?.email,
          required: false,
        },
      ],
      commit: (v) =>
        this.store.execute('saveContact', {
          entity,
          id: c?.id,
          name: v['name'],
          document: v['document'],
          phone: v['phone'],
          email: v['email'],
        }),
    });
  }
  remove(c: Contact): void {
    const entity = this.type();
    this.dialogs.form({
      title: 'Dar de baja: ' + c.name,
      description: 'Podrás restaurar este contacto sin perder su historial.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'reason', label: 'Motivo', type: 'textarea' }],
      commit: (v) => this.store.execute('softDelete', { entity, id: c.id, reason: v['reason'] }),
    });
  }
}
