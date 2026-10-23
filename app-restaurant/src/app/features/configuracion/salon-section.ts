import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs } from '../../shared/ui/form-dialog';
import { entero, opcional, texto } from './configuracion.forms';
import type { Mesa, MesaNueva, Zona } from './configuracion.models';
import { ConfiguracionStore } from './data-access/configuracion-store';

@Component({
  selector: 'app-salon-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel mt-6" aria-labelledby="salon-title">
      <div class="panel-header">
        <h2 id="salon-title" class="font-bold">Zonas y mesas</h2>
        @if (store.sedesSalon().length > 1) {
          <select aria-label="Sede" (change)="elegirSede($event)">
            @for (sede of store.sedesSalon(); track sede.id) {
              <option [value]="sede.id" [selected]="sede.id === store.sedeActual()">
                {{ sede.nombre }}
              </option>
            }
          </select>
        }
      </div>
      <div class="panel-body">
        @if (store.errors().salon; as error) {
          <div class="empty" role="alert">
            <h3>No pudimos cargar las zonas y mesas</h3>
            <p>{{ error }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        } @else {
          @if (store.puedeEditarSalon() && store.sedeActual()) {
            <div class="inline-actions mb-4">
              <button type="button" class="btn" (click)="nuevaZona()">Nueva zona</button>
              <button
                type="button"
                class="btn primary"
                [disabled]="store.zonasActivasSede().length === 0"
                (click)="nuevaMesa()"
              >
                Nueva mesa
              </button>
            </div>
          }
          @for (zona of store.zonasSede(); track zona.id) {
            <article class="setting-block" [class.inactive]="!zona.es_activo">
              <div>
                <h3>{{ zona.nombre }}</h3>
                <span class="pill" [class.green]="zona.es_activo">
                  {{ zona.es_activo ? 'Activa' : 'De baja' }}
                </span>
              </div>
              @if (store.puedeEditarSalon()) {
                <div class="inline-actions">
                  @if (zona.es_activo) {
                    <button
                      type="button"
                      class="btn"
                      [attr.aria-label]="'Editar zona ' + zona.nombre"
                      (click)="editarZona(zona)"
                    >
                      Editar
                    </button>
                    <button
                      type="button"
                      class="btn danger"
                      [attr.aria-label]="'Dar de baja zona ' + zona.nombre"
                      (click)="bajaZona(zona)"
                    >
                      Dar de baja
                    </button>
                  } @else {
                    <button
                      type="button"
                      class="btn"
                      [attr.aria-label]="'Restaurar zona ' + zona.nombre"
                      [disabled]="store.busyId() !== null"
                      (click)="restaurarZona(zona)"
                    >
                      Restaurar
                    </button>
                  }
                </div>
              }
            </article>
            @for (mesa of store.mesasDeZona(zona.id); track mesa.id) {
              <div class="setting-block ml-6" [class.inactive]="!mesa.es_activo">
                <div>
                  <h4>Mesa {{ mesa.numero_mesa }} · {{ mesa.capacidad }} personas</h4>
                  @if (mesa.posicion_x !== null && mesa.posicion_y !== null) {
                    <p class="subtle">Posición ({{ mesa.posicion_x }}, {{ mesa.posicion_y }})</p>
                  }
                  @if (!mesa.es_activo && mesa.motivo_eliminacion) {
                    <p class="subtle">Motivo de la baja: {{ mesa.motivo_eliminacion }}</p>
                  }
                </div>
                <div class="inline-actions">
                  <button
                    type="button"
                    class="toggle"
                    role="switch"
                    [class.on]="mesa.habilitada"
                    [attr.aria-checked]="mesa.habilitada"
                    [attr.aria-label]="'Habilitada: mesa ' + mesa.numero_mesa"
                    [disabled]="
                      !store.puedeEditarSalon() || !mesa.es_activo || store.busyId() !== null
                    "
                    (click)="store.habilitarMesa(mesa)"
                  ></button>
                  @if (store.puedeEditarSalon()) {
                    @if (mesa.es_activo) {
                      <button
                        type="button"
                        class="btn"
                        [attr.aria-label]="'Editar mesa ' + mesa.numero_mesa"
                        (click)="editarMesa(mesa)"
                      >
                        Editar
                      </button>
                      <button
                        type="button"
                        class="btn danger"
                        [attr.aria-label]="'Dar de baja mesa ' + mesa.numero_mesa"
                        (click)="bajaMesa(mesa)"
                      >
                        Dar de baja
                      </button>
                    } @else {
                      <button
                        type="button"
                        class="btn"
                        [attr.aria-label]="'Restaurar mesa ' + mesa.numero_mesa"
                        [disabled]="store.busyId() !== null"
                        (click)="restaurarMesa(mesa)"
                      >
                        Restaurar
                      </button>
                    }
                  }
                </div>
              </div>
            }
          } @empty {
            <p class="empty" role="status">Esta sede todavía no tiene zonas.</p>
          }
        }
      </div>
    </section>
  `,
})
export class SalonSection {
  readonly store = inject(ConfiguracionStore);
  private readonly dialogs = inject(Dialogs);

  elegirSede(evento: Event): void {
    this.store.elegirSede((evento.target as HTMLSelectElement).value);
  }

  nuevaZona(): void {
    const sede = this.store.sedeActual();
    if (!sede) return;
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva zona',
      description: 'Una zona agrupa mesas del salón (por ejemplo, Terraza).',
      submit: 'Crear zona',
      fields: [
        { key: 'nombre', label: 'Nombre' },
        {
          key: 'orden_visualizacion',
          label: 'Orden',
          type: 'number',
          min: 0,
          max: 9999,
          required: false,
          value: 0,
        },
      ],
      commit: (v) =>
        this.store.crearZona(
          {
            sucursal_id: sede,
            nombre: texto(v['nombre']),
            orden_visualizacion:
              v['orden_visualizacion'] === '' ? 0 : entero(v['orden_visualizacion']),
          },
          key,
        ),
    });
  }

  editarZona(zona: Zona): void {
    this.dialogs.form({
      title: 'Editar zona: ' + zona.nombre,
      description: 'Lo que dejes igual no se envía.',
      submit: 'Guardar cambios',
      fields: [
        { key: 'nombre', label: 'Nombre', value: zona.nombre },
        {
          key: 'orden_visualizacion',
          label: 'Orden',
          type: 'number',
          min: 0,
          max: 9999,
          required: false,
          value: zona.orden_visualizacion,
        },
      ],
      commit: (v) =>
        this.store.editarZona(zona, {
          nombre: texto(v['nombre']),
          orden_visualizacion:
            v['orden_visualizacion'] === '' ? 0 : entero(v['orden_visualizacion']),
        }),
    });
  }

  bajaZona(zona: Zona): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja la zona ' + zona.nombre,
      description: 'No puede tener mesas activas. Podrás restaurarla después.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [],
      commit: () => this.store.bajaZona(zona, key),
    });
  }

  restaurarZona(zona: Zona): void {
    void this.store.restaurarZona(zona, newIdempotencyKey());
  }

  nuevaMesa(): void {
    const sede = this.store.sedeActual();
    const zonas = this.store.zonasActivasSede();
    if (!sede || zonas.length === 0) return;
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Nueva mesa',
      description: 'El número debe ser único en la sede.',
      submit: 'Crear mesa',
      fields: [
        {
          key: 'zona_id',
          label: 'Zona',
          type: 'select',
          options: zonas.map((z) => ({ value: z.id, label: z.nombre })),
        },
        { key: 'numero_mesa', label: 'Número de mesa', hint: 'Hasta 20 caracteres.' },
        { key: 'capacidad', label: 'Capacidad', type: 'number', min: 1, max: 500, value: 4 },
        { key: 'posicion_x', label: 'Posición X', required: false, hint: 'Opcional. Ej.: 12.50' },
        { key: 'posicion_y', label: 'Posición Y', required: false, hint: 'Opcional. Ej.: 8.00' },
      ],
      commit: (v) => {
        const cuerpo: MesaNueva = {
          sucursal_id: sede,
          zona_id: v['zona_id'],
          numero_mesa: texto(v['numero_mesa']),
          capacidad: entero(v['capacidad']),
        };
        const x = opcional(v['posicion_x']);
        const y = opcional(v['posicion_y']);
        if (x) cuerpo.posicion_x = x;
        if (y) cuerpo.posicion_y = y;
        return this.store.crearMesa(cuerpo, key);
      },
    });
  }

  editarMesa(mesa: Mesa): void {
    const zonas = this.store.zonasActivasSede();
    this.dialogs.form({
      title: 'Editar mesa ' + mesa.numero_mesa,
      description: 'Lo que dejes igual no se envía. Deja la posición en blanco para quitarla.',
      submit: 'Guardar cambios',
      fields: [
        {
          key: 'zona_id',
          label: 'Zona',
          type: 'select',
          value: mesa.zona_id,
          options: zonas.map((z) => ({ value: z.id, label: z.nombre })),
        },
        { key: 'numero_mesa', label: 'Número de mesa', value: mesa.numero_mesa },
        {
          key: 'capacidad',
          label: 'Capacidad',
          type: 'number',
          min: 1,
          max: 500,
          value: mesa.capacidad,
        },
        { key: 'posicion_x', label: 'Posición X', required: false, value: mesa.posicion_x ?? '' },
        { key: 'posicion_y', label: 'Posición Y', required: false, value: mesa.posicion_y ?? '' },
      ],
      commit: (v) =>
        this.store.editarMesa(mesa, {
          zona_id: v['zona_id'],
          numero_mesa: texto(v['numero_mesa']),
          capacidad: entero(v['capacidad']),
          posicion_x: opcional(v['posicion_x']),
          posicion_y: opcional(v['posicion_y']),
        }),
    });
  }

  bajaMesa(mesa: Mesa): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja la mesa ' + mesa.numero_mesa,
      description: 'No puede tener un pedido abierto. Podrás restaurarla después.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'motivo', label: 'Motivo', type: 'textarea' }],
      commit: (v) => this.store.bajaMesa(mesa, texto(v['motivo']), key),
    });
  }

  restaurarMesa(mesa: Mesa): void {
    void this.store.restaurarMesa(mesa, newIdempotencyKey());
  }
}
