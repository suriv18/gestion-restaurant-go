import { ChangeDetectionStrategy, Component, inject, OnInit } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs, type Field, type FieldProblem } from '../../shared/ui/form-dialog';
import type { Impuesto, TasaImpuesto } from '../catalogos/data-access/carta.models';
import {
  formatearFecha,
  formatearTasa,
  formatearVigencia,
  vigenteEn,
} from '../catalogos/data-access/carta.util';
import { TasasStore, type ValoresTasa } from './data-access/tasas-store';

type Resultado = Promise<string | FieldProblem | undefined>;

/** Impuestos (solo lectura) con su historial de tasas y vigencias. */
@Component({
  selector: 'app-tasas-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel mt-6" aria-labelledby="tasas-title">
      <div class="panel-header">
        <h2 id="tasas-title" class="font-bold">Impuestos y tasas</h2>
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
        @switch (store.status()) {
          @case ('loading') {
            <p class="empty" role="status">Cargando los impuestos…</p>
          }
          @case ('error') {
            <div class="empty" role="alert">
              <h3>No pudimos cargar los impuestos</h3>
              <p>{{ store.error() }}</p>
              <button type="button" class="btn primary" (click)="store.cargar()">Reintentar</button>
            </div>
          }
          @default {
            <p class="subtle mb-4">
              No hay una tasa por defecto: sin una tasa vigente no se puede vender con el impuesto
              correspondiente. Las vigencias van desde una fecha (incluida) hasta otra (no
              incluida).
            </p>
            @for (imp of store.impuestos(); track imp.id) {
              <article class="impuesto-bloque" [attr.aria-label]="'Impuesto ' + imp.nombre">
                <div class="flex flex-wrap justify-between gap-2 items-center mb-2">
                  <div>
                    <h3>
                      {{ imp.nombre }} <small class="subtle">{{ imp.codigo }}</small>
                    </h3>
                    @if (imp.descripcion) {
                      <p class="subtle">{{ imp.descripcion }}</p>
                    }
                    @if (!imp.es_activo) {
                      <span class="pill">Inactivo</span>
                    }
                  </div>
                  @if (store.puedeAdministrar() && imp.es_activo) {
                    <button
                      type="button"
                      class="btn"
                      [attr.aria-label]="'Nueva tasa de ' + imp.nombre"
                      (click)="nueva(imp)"
                    >
                      + Nueva tasa
                    </button>
                  }
                </div>
                @if (store.errores()[imp.id]; as error) {
                  <div class="empty" role="alert">
                    <p>No pudimos cargar las tasas de {{ imp.nombre }}. {{ error }}</p>
                    <button type="button" class="btn primary" (click)="store.reintentar(imp.id)">
                      Reintentar
                    </button>
                  </div>
                } @else {
                  @for (t of tasasDe(imp.id); track t.id) {
                    <div class="tasa-fila">
                      <div>
                        <strong>{{ formatearTasa(t.tasa) }}</strong>
                        @if (vigente(t)) {
                          <span class="pill green">Vigente hoy</span>
                        }
                        <p class="subtle">
                          {{ formatearVigencia(t.vigente_desde, t.vigente_hasta) }}
                        </p>
                        @if (t.fundamento) {
                          <p class="subtle">Fundamento: {{ t.fundamento }}</p>
                        }
                      </div>
                      @if (store.puedeAdministrar()) {
                        <button
                          type="button"
                          class="btn"
                          [attr.aria-label]="etiquetaEditar(imp, t)"
                          (click)="editar(imp, t)"
                        >
                          Editar
                        </button>
                      }
                    </div>
                  } @empty {
                    <div class="empty-state" role="status">
                      <h4>Aún no hay tasa configurada</h4>
                      <p>
                        {{ imp.nombre }} no tiene una tasa por defecto: hasta que registres una
                        vigente, no se podrán vender productos con este impuesto.
                      </p>
                    </div>
                  }
                }
              </article>
            } @empty {
              <p class="empty" role="status">No hay impuestos configurados.</p>
            }
          }
        }
      </div>
    </section>
  `,
})
export class TasasSection implements OnInit {
  readonly store = inject(TasasStore);
  private readonly dialogs = inject(Dialogs);

  readonly formatearTasa = formatearTasa;
  readonly formatearVigencia = formatearVigencia;

  ngOnInit(): void {
    void this.store.cargar();
  }

  tasasDe(impuestoId: string): TasaImpuesto[] {
    return this.store.tasas()[impuestoId] ?? [];
  }

  vigente(t: TasaImpuesto): boolean {
    return vigenteEn(t.vigente_desde, t.vigente_hasta, this.store.hoy());
  }

  etiquetaEditar(imp: Impuesto, t: TasaImpuesto): string {
    return `Editar tasa ${formatearTasa(t.tasa)} de ${imp.nombre} desde ${formatearFecha(t.vigente_desde)}`;
  }

  nueva(imp: Impuesto): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva tasa de ' + imp.nombre,
      description:
        'La vigencia no puede cruzarse con otra tasa del mismo impuesto. Deja el fin vacío si no tiene fecha de término.',
      submit: 'Registrar tasa',
      fields: this.campos(),
      commit: async (v): Resultado => this.store.crear(imp, v as unknown as ValoresTasa, key),
    });
  }

  editar(imp: Impuesto, t: TasaImpuesto): void {
    this.dialogs.form({
      title: 'Editar tasa de ' + imp.nombre,
      description: 'Lo que dejes igual no se envía. El impuesto no se puede cambiar.',
      submit: 'Guardar cambios',
      fields: this.campos(t),
      commit: async (v): Resultado => this.store.editar(t, v as unknown as ValoresTasa),
    });
  }

  private campos(t?: TasaImpuesto): Field[] {
    return [
      {
        key: 'tasa',
        label: 'Tasa (%)',
        value: t ? t.tasa : '',
        hint: 'Entre 0 y 100, hasta 4 decimales. Ej.: 18 o 10.5',
      },
      { key: 'vigente_desde', label: 'Vigente desde', type: 'date', value: t?.vigente_desde ?? '' },
      {
        key: 'vigente_hasta',
        label: 'Vigente hasta (no incluida)',
        type: 'date',
        required: false,
        value: t?.vigente_hasta ?? '',
        hint: 'Opcional. Vacío = sin fecha de término.',
      },
      {
        key: 'fundamento',
        label: 'Fundamento',
        type: 'textarea',
        required: false,
        value: t?.fundamento ?? '',
        hint: 'Opcional. Norma o motivo del cambio (hasta 2000 caracteres).',
      },
    ];
  }
}
