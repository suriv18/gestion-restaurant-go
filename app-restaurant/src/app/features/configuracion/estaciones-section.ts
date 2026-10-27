import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  signal,
  untracked,
} from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs, type FieldProblem } from '../../shared/ui/form-dialog';
import type { Estacion } from '../catalogos/data-access/carta.models';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { EstacionesStore } from './data-access/estaciones-store';

type Resultado = Promise<string | FieldProblem | undefined>;

/** Estaciones de preparación de la sede actual (la misma que elige «Zonas y mesas»). */
@Component({
  selector: 'app-estaciones-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel mt-6" aria-labelledby="estaciones-title">
      <div class="panel-header">
        <h2 id="estaciones-title" class="font-bold">Estaciones de preparación</h2>
        <div class="inline-actions">
          @if (config.sedesSalon().length > 1) {
            <select aria-label="Sede de las estaciones" (change)="elegirSede($event)">
              @for (sede of config.sedesSalon(); track sede.id) {
                <option [value]="sede.id" [selected]="sede.id === config.sedeActual()">
                  {{ sede.nombre }}
                </option>
              }
            </select>
          }
          @if (config.sedeActual()) {
            <label class="flex gap-2 text-xs items-center">
              <input
                type="checkbox"
                [checked]="incluirBajas()"
                (change)="incluirBajas.set($any($event.target).checked)"
              />Incluir bajas
            </label>
          }
          @if (store.puedeAdministrar() && config.sedeActual()) {
            <button type="button" class="btn primary" (click)="nueva()">+ Nueva estación</button>
          }
        </div>
      </div>
      <div class="panel-body">
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
        @if (!config.sedeActual()) {
          <p class="empty" role="status">
            No hay una sede disponible para mostrar estaciones. Cuando tengas una sede asignada,
            aparecerá aquí.
          </p>
        } @else if (store.status() === 'loading') {
          <p class="empty" role="status">Cargando las estaciones…</p>
        } @else if (store.status() === 'error') {
          <div class="empty" role="alert">
            <h3>No pudimos cargar las estaciones</h3>
            <p>{{ store.error() }}</p>
            <button type="button" class="btn primary" (click)="recargar()">Reintentar</button>
          </div>
        } @else if (filas().length === 0) {
          <div class="empty-state" role="status">
            <h3>
              {{ incluirBajas() ? 'Esta sede no tiene estaciones' : 'Aún no hay estaciones' }}
            </h3>
            <p>
              Una estación es el lugar donde se prepara un pedido (por ejemplo, Cocina o Barra).
              Después puedes asignarla a cada producto desde la Carta.
            </p>
          </div>
        } @else {
          <div class="table-scroll">
            <table>
              <caption class="sr-only">
                Estaciones de preparación de la sede
              </caption>
              <thead>
                <tr>
                  <th scope="col">Código</th>
                  <th scope="col">Nombre</th>
                  <th scope="col">Estado</th>
                  @if (store.puedeAdministrar()) {
                    <th scope="col">Acciones</th>
                  }
                </tr>
              </thead>
              <tbody>
                @for (e of filas(); track e.id) {
                  <tr>
                    <td>
                      <strong>{{ e.codigo }}</strong>
                    </td>
                    <td>{{ e.nombre }}</td>
                    <td>
                      <span class="pill" [class.green]="e.es_activo">{{
                        e.es_activo ? 'Activa' : 'De baja'
                      }}</span>
                    </td>
                    @if (store.puedeAdministrar()) {
                      <td>
                        <div class="inline-actions">
                          @if (e.es_activo) {
                            <button
                              type="button"
                              class="btn"
                              [attr.aria-label]="'Editar estación ' + e.nombre"
                              (click)="editar(e)"
                            >
                              Editar
                            </button>
                            <button
                              type="button"
                              class="btn danger"
                              [attr.aria-label]="'Dar de baja estación ' + e.nombre"
                              (click)="baja(e)"
                            >
                              Dar de baja
                            </button>
                          } @else {
                            <button
                              type="button"
                              class="btn"
                              [attr.aria-label]="'Restaurar estación ' + e.nombre"
                              [disabled]="store.busyId() !== null"
                              (click)="restaurar(e)"
                            >
                              Restaurar
                            </button>
                          }
                        </div>
                      </td>
                    }
                  </tr>
                }
              </tbody>
            </table>
          </div>
        }
      </div>
    </section>
  `,
})
export class EstacionesSection {
  readonly config = inject(ConfiguracionStore);
  readonly store = inject(EstacionesStore);
  private readonly dialogs = inject(Dialogs);

  readonly incluirBajas = signal(false);
  readonly filas = computed(() =>
    this.store.ordenadas().filter((e) => this.incluirBajas() || e.es_activo),
  );

  constructor() {
    // Una carga por cada sede que se muestre (la inicial y cada cambio de selector).
    effect(() => {
      const sede = this.config.sedeActual();
      untracked(() => void this.store.cargar(sede));
    });
  }

  elegirSede(evento: Event): void {
    this.config.elegirSede((evento.target as HTMLSelectElement).value);
  }

  recargar(): void {
    void this.store.cargar(this.config.sedeActual());
  }

  nueva(): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva estación',
      description: 'El código identifica la estación y no se puede cambiar después.',
      submit: 'Crear estación',
      fields: [
        {
          key: 'codigo',
          label: 'Código',
          hint: 'Hasta 20 letras, números, guion o guion bajo. Se guarda en mayúsculas.',
        },
        { key: 'nombre', label: 'Nombre', hint: 'Hasta 80 caracteres.' },
      ],
      commit: async (v): Resultado =>
        this.store.crear({ codigo: v['codigo'], nombre: v['nombre'] }, key),
    });
  }

  editar(e: Estacion): void {
    this.dialogs.form({
      title: 'Editar estación ' + e.codigo,
      description: 'Solo se puede cambiar el nombre; el código queda igual.',
      submit: 'Guardar cambios',
      fields: [{ key: 'nombre', label: 'Nombre', value: e.nombre, hint: 'Hasta 80 caracteres.' }],
      commit: async (v): Resultado => this.store.editar(e, { nombre: v['nombre'] }),
    });
  }

  baja(e: Estacion): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja la estación ' + e.nombre,
      description:
        'Si algún producto activo de la sede la usa, primero cámbiale la estación. Podrás restaurarla cuando quieras.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [],
      commit: async (): Resultado => this.store.baja(e, key),
    });
  }

  restaurar(e: Estacion): void {
    void this.store.restaurar(e, newIdempotencyKey());
  }
}
