import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs } from '../../shared/ui/form-dialog';
import { opcional, texto } from './configuracion.forms';
import type { Sucursal } from './configuracion.models';
import { ConfiguracionStore } from './data-access/configuracion-store';

@Component({
  selector: 'app-sedes-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel mt-6" aria-labelledby="sedes-title">
      <div class="panel-header">
        <h2 id="sedes-title" class="font-bold">Sedes</h2>
        @if (store.puedeEditarSedes()) {
          <button type="button" class="btn primary" (click)="crear()">Nueva sede</button>
        }
      </div>
      <div class="panel-body">
        @if (store.errors().sedes; as error) {
          <div class="empty" role="alert">
            <h3>No pudimos cargar las sedes</h3>
            <p>{{ error }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        } @else {
          @for (sede of store.sedes(); track sede.id) {
            <article class="setting-block" [class.inactive]="!sede.es_activo">
              <div>
                <h3>
                  {{ sede.nombre }} <span class="subtle">({{ sede.codigo }})</span>
                </h3>
                <p>{{ sede.direccion }}</p>
                @if (!sede.es_activo && sede.motivo_eliminacion) {
                  <p class="subtle">Motivo de la baja: {{ sede.motivo_eliminacion }}</p>
                }
              </div>
              <div class="inline-actions">
                <span class="pill" [class.green]="sede.es_activo">
                  {{ sede.es_activo ? 'Activa' : 'De baja' }}
                </span>
                @if (store.puedeEditarSedes()) {
                  @if (sede.es_activo) {
                    <button
                      type="button"
                      class="btn"
                      [attr.aria-label]="'Editar sede ' + sede.nombre"
                      (click)="editar(sede)"
                    >
                      Editar
                    </button>
                    <button
                      type="button"
                      class="btn danger"
                      [attr.aria-label]="'Dar de baja sede ' + sede.nombre"
                      (click)="baja(sede)"
                    >
                      Dar de baja
                    </button>
                  } @else {
                    <button
                      type="button"
                      class="btn"
                      [disabled]="store.busyId() !== null"
                      (click)="restaurar(sede)"
                    >
                      Restaurar
                    </button>
                  }
                }
              </div>
            </article>
          } @empty {
            <p class="empty" role="status">Todavía no hay sedes registradas.</p>
          }
        }
      </div>
    </section>
  `,
})
export class SedesSection {
  readonly store = inject(ConfiguracionStore);
  private readonly dialogs = inject(Dialogs);

  crear(): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva sede',
      description: 'Después de crearla, asígnala a tu equipo en Equipo y accesos.',
      submit: 'Crear sede',
      fields: [
        {
          key: 'codigo',
          label: 'Código',
          hint: 'De 2 a 20 caracteres: mayúsculas, números, _ o -.',
        },
        { key: 'nombre', label: 'Nombre' },
        { key: 'direccion', label: 'Dirección' },
        { key: 'ubigeo', label: 'Ubigeo (6 dígitos)', required: false },
        {
          key: 'codigo_establecimiento_sunat',
          label: 'Código de establecimiento SUNAT',
          required: false,
        },
        { key: 'telefono', label: 'Teléfono', type: 'tel', required: false },
      ],
      commit: (v) =>
        this.store.crearSede(
          {
            codigo: texto(v['codigo']).toUpperCase(),
            nombre: texto(v['nombre']),
            direccion: texto(v['direccion']),
            ...this.opcionales(v),
          },
          key,
        ),
    });
  }

  editar(sede: Sucursal): void {
    this.dialogs.form({
      title: 'Editar sede: ' + sede.nombre,
      description: 'Cambia solo lo necesario; lo que dejes igual no se envía.',
      submit: 'Guardar cambios',
      fields: [
        { key: 'codigo', label: 'Código', value: sede.codigo },
        { key: 'nombre', label: 'Nombre', value: sede.nombre },
        { key: 'direccion', label: 'Dirección', value: sede.direccion },
        { key: 'ubigeo', label: 'Ubigeo (6 dígitos)', required: false, value: sede.ubigeo ?? '' },
        {
          key: 'codigo_establecimiento_sunat',
          label: 'Código de establecimiento SUNAT',
          required: false,
          value: sede.codigo_establecimiento_sunat ?? '',
        },
        {
          key: 'telefono',
          label: 'Teléfono',
          type: 'tel',
          required: false,
          value: sede.telefono ?? '',
        },
      ],
      commit: (v) =>
        this.store.editarSede(sede, {
          codigo: texto(v['codigo']).toUpperCase(),
          nombre: texto(v['nombre']),
          direccion: texto(v['direccion']),
          ubigeo: opcional(v['ubigeo']),
          codigo_establecimiento_sunat: opcional(v['codigo_establecimiento_sunat']),
          telefono: opcional(v['telefono']),
        }),
    });
  }

  baja(sede: Sucursal): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja: ' + sede.nombre,
      description:
        'La sede deja de operar; sus datos y su historial se conservan y podrás restaurarla.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'motivo', label: 'Motivo', type: 'textarea' }],
      commit: (v) => this.store.bajaSede(sede, texto(v['motivo']), key),
    });
  }

  restaurar(sede: Sucursal): void {
    void this.store.restaurarSede(sede, newIdempotencyKey());
  }

  private opcionales(v: Record<string, string>) {
    const out: { ubigeo?: string; codigo_establecimiento_sunat?: string; telefono?: string } = {};
    const ubigeo = opcional(v['ubigeo']);
    const sunat = opcional(v['codigo_establecimiento_sunat']);
    const telefono = opcional(v['telefono']);
    if (ubigeo) out.ubigeo = ubigeo;
    if (sunat) out.codigo_establecimiento_sunat = sunat;
    if (telefono) out.telefono = telefono;
    return out;
  }
}
