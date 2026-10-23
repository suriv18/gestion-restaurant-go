import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  signal,
} from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { cambiosDe, opcional, texto } from './configuracion.forms';
import { ConfiguracionStore } from './data-access/configuracion-store';

const CAMPOS = [
  { key: 'razon_social', label: 'Razón social', opcional: false },
  { key: 'nombre_comercial', label: 'Nombre comercial', opcional: false },
  { key: 'direccion_fiscal', label: 'Dirección fiscal', opcional: false },
  { key: 'ubigeo', label: 'Ubigeo (6 dígitos)', opcional: true },
  { key: 'telefono', label: 'Teléfono', opcional: true },
  { key: 'email', label: 'Correo', opcional: true },
  { key: 'moneda', label: 'Moneda (código de 3 letras)', opcional: false },
  { key: 'zona_horaria', label: 'Zona horaria', opcional: false },
] as const;

@Component({
  selector: 'app-negocio-section',
  imports: [ReactiveFormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="negocio-title">
      <div class="panel-header">
        <h2 id="negocio-title" class="font-bold">Tu restaurante</h2>
      </div>
      <div class="panel-body">
        @if (store.errors().negocio; as error) {
          <div class="empty" role="alert">
            <h3>No pudimos cargar los datos de tu restaurante</h3>
            <p>{{ error }}</p>
            <button type="button" class="btn primary" (click)="store.load()">Reintentar</button>
          </div>
        } @else if (store.negocio()) {
          <form class="settings-form" [formGroup]="form" (ngSubmit)="guardar()">
            <div class="form-grid">
              <label class="field" for="negocio-ruc">
                <span>RUC</span>
                <input id="negocio-ruc" [value]="store.negocio()?.ruc" readonly />
                <small>El RUC no se puede cambiar.</small>
              </label>
              @for (campo of campos; track campo.key) {
                <label class="field" [for]="'negocio-' + campo.key">
                  <span>{{ campo.label }}</span>
                  <input
                    [id]="'negocio-' + campo.key"
                    [formControlName]="campo.key"
                    [readonly]="!store.puedeEditarNegocio()"
                    [attr.aria-invalid]="!!errores()[campo.key]"
                  />
                  @if (errores()[campo.key]; as message) {
                    <small class="text-red-700" role="alert">{{ message }}</small>
                  }
                </label>
              }
            </div>
            @if (aviso()) {
              <p class="form-error" role="alert">{{ aviso() }}</p>
            }
            @if (store.puedeEditarNegocio()) {
              <button type="submit" class="btn primary" [disabled]="guardando()">
                Guardar cambios
              </button>
            } @else {
              <p class="note-box" role="status">Solo puedes consultar estos datos.</p>
            }
          </form>
        }
      </div>
    </section>
  `,
})
export class NegocioSection {
  readonly store = inject(ConfiguracionStore);
  readonly campos = CAMPOS;
  readonly errores = signal<Record<string, string>>({});
  readonly aviso = signal('');
  readonly guardando = signal(false);

  readonly form = new FormGroup(
    Object.fromEntries(CAMPOS.map((c) => [c.key, new FormControl('', { nonNullable: true })])),
  );

  private readonly actual = computed(() => this.store.negocio());

  constructor() {
    effect(() => {
      const negocio = this.actual();
      if (!negocio) return;
      this.form.reset(
        Object.fromEntries(
          CAMPOS.map((c) => [
            c.key,
            (negocio as unknown as Record<string, string | null>)[c.key] ?? '',
          ]),
        ),
      );
    });
    this.form.valueChanges.subscribe(() => {
      if (Object.keys(this.errores()).length) this.errores.set({});
    });
  }

  async guardar(): Promise<void> {
    const negocio = this.actual();
    if (!negocio || this.guardando()) return;
    const valores = this.form.getRawValue();
    const nuevo = Object.fromEntries(
      CAMPOS.map((c) => [c.key, c.opcional ? opcional(valores[c.key]) : texto(valores[c.key])]),
    );
    const cambios = cambiosDe({ ...negocio }, nuevo);
    this.aviso.set('');
    this.errores.set({});
    this.guardando.set(true);
    try {
      const resultado = await this.store.editarNegocio(cambios);
      if (typeof resultado === 'string') this.aviso.set(resultado);
      else if (resultado) {
        if (CAMPOS.some((c) => c.key === resultado.field)) {
          this.errores.set({ [resultado.field]: resultado.message });
        } else {
          this.aviso.set(resultado.message);
        }
      }
    } finally {
      this.guardando.set(false);
    }
  }
}
