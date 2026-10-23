import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { DIALOG_DATA, Dialog, DialogRef } from '@angular/cdk/dialog';
import { FormControl, FormRecord, ReactiveFormsModule, Validators } from '@angular/forms';
import { Injectable } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Icon } from './icon';
export interface Field {
  key: string;
  label: string;
  type?: 'text' | 'number' | 'date' | 'time' | 'email' | 'tel' | 'password' | 'textarea' | 'select';
  autocomplete?: string;
  value?: string | number;
  required?: boolean;
  min?: number | string;
  max?: number;
  step?: string;
  options?: { value: string | number; label: string }[];
  hint?: string;
}

/** Rechazo del servidor asociado a un campo: el diálogo lo pinta bajo ese campo. */
export interface FieldProblem {
  field: string;
  message: string;
}

function isFieldProblem(value: unknown): value is FieldProblem {
  if (typeof value !== 'object' || value === null) return false;
  const { field, message } = value as Record<string, unknown>;
  return typeof field === 'string' && typeof message === 'string';
}

export interface FormDialogData {
  title: string;
  description: string;
  fields: Field[];
  submit?: string;
  danger?: boolean;
  /**
   * Ejecuta la acción. Si devuelve una promesa (idealmente Promise<string | FieldProblem | undefined>), el
   * diálogo espera: un texto resuelto (o un error lanzado) se muestra dentro del diálogo, que
   * se mantiene abierto con los datos escritos. Un `FieldProblem` resuelto se muestra bajo su
   * campo (o en el aviso general si el campo no existe). Un resultado vacío lo cierra. Cualquier otro
   * valor de retorno se ignora y el diálogo cierra al instante (acciones síncronas).
   */
  commit: (values: Record<string, string>) => unknown;
}
@Component({
  selector: 'app-form-dialog',
  imports: [ReactiveFormsModule, Icon],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: ` <div class="modal-header">
      <div>
        <h2 id="dialog-title">{{ data.title }}</h2>
        <p id="dialog-description">{{ data.description }}</p>
      </div>
      <button class="icon-btn" aria-label="Cerrar diálogo" (click)="ref.close()">
        <app-icon name="close" />
      </button>
    </div>
    <form class="modal-body" [formGroup]="form" (ngSubmit)="submit()">
      <div class="form-grid">
        @for (field of data.fields; track field.key) {
          <label
            class="field"
            [class.full]="field.type === 'textarea'"
            [for]="'field-' + field.key"
          >
            <span
              >{{ field.label }}
              @if (field.required !== false) {
                <span aria-hidden="true">*</span>
              }
            </span>
            @switch (field.type) {
              @case ('select') {
                <select
                  [id]="'field-' + field.key"
                  [formControlName]="field.key"
                  [attr.aria-label]="field.label"
                  [attr.aria-invalid]="flagged(field.key)"
                >
                  @for (option of field.options; track option.value) {
                    <option [value]="option.value">{{ option.label }}</option>
                  }
                </select>
              }
              @case ('textarea') {
                <textarea
                  [id]="'field-' + field.key"
                  [formControlName]="field.key"
                  [attr.aria-label]="field.label"
                  [attr.aria-invalid]="flagged(field.key)"
                  rows="3"
                ></textarea>
              }
              @default {
                <input
                  [id]="'field-' + field.key"
                  [type]="field.type || 'text'"
                  [formControlName]="field.key"
                  [attr.aria-label]="field.label"
                  [attr.min]="field.min ?? null"
                  [attr.max]="field.max ?? null"
                  [attr.step]="field.step ?? null"
                  [attr.autocomplete]="field.autocomplete ?? null"
                  [attr.aria-invalid]="flagged(field.key)"
                />
              }
            }
            @if (field.hint) {
              <small>{{ field.hint }}</small>
            }
            @if (fieldErrors()[field.key]; as message) {
              <small class="text-red-700" role="alert">{{ message }}</small>
            } @else if (invalid(field.key)) {
              <small class="text-red-700">Revisa este campo.</small>
            }
          </label>
        }
      </div>
      @if (error()) {
        <p class="form-error" role="alert">{{ error() }}</p>
      }
      <div class="modal-actions">
        <button type="button" class="btn" (click)="ref.close()">Cancelar</button
        ><button
          class="btn"
          [class.primary]="!data.danger"
          [class.danger]="data.danger"
          [disabled]="busy()"
        >
          {{ data.submit || 'Guardar' }}
        </button>
      </div>
    </form>`,
})
export class FormDialog {
  readonly data = inject<FormDialogData>(DIALOG_DATA);
  readonly ref = inject(DialogRef);
  readonly error = signal('');
  readonly busy = signal(false);
  readonly form = new FormRecord<FormControl<string>>(
    Object.fromEntries(
      this.data.fields.map((field) => {
        const validators =
          field.required !== false ? [Validators.required, Validators.pattern(/\S/)] : [];
        if (typeof field.min === 'number') validators.push(Validators.min(field.min));
        if (field.max !== undefined) validators.push(Validators.max(field.max));
        if (field.type === 'email') validators.push(Validators.email);
        return [
          field.key,
          new FormControl(String(field.value ?? field.options?.[0]?.value ?? ''), {
            nonNullable: true,
            validators,
          }),
        ];
      }),
    ),
  );
  readonly fieldErrors = signal<Record<string, string>>({});
  constructor() {
    for (const [key, control] of Object.entries(this.form.controls)) {
      control.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => {
        if (!this.fieldErrors()[key]) return;
        this.fieldErrors.update((current) => {
          const next = { ...current };
          delete next[key];
          return next;
        });
      });
    }
  }
  invalid(key: string): boolean {
    const c = this.form.controls[key];
    return c.touched && c.invalid;
  }
  flagged(key: string): boolean {
    return this.invalid(key) || !!this.fieldErrors()[key];
  }
  async submit(): Promise<void> {
    if (this.busy()) return;
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.error.set('Completa los campos indicados antes de continuar.');
      return;
    }
    this.busy.set(true);
    this.error.set('');
    this.fieldErrors.set({});
    try {
      const result = this.data.commit(this.form.getRawValue());
      // Solo se espera si la acción es asíncrona: la síncrona cierra sin ceder el turno.
      const outcome = result instanceof Promise ? await result : undefined;
      if (typeof outcome === 'string' && outcome) {
        this.error.set(outcome);
        return;
      }
      if (isFieldProblem(outcome)) {
        if (Object.hasOwn(this.form.controls, outcome.field)) {
          this.fieldErrors.set({ [outcome.field]: outcome.message });
          this.form.controls[outcome.field].markAsTouched();
        } else {
          this.error.set(outcome.message);
        }
        return;
      }
      this.ref.close(true);
    } catch (error) {
      this.error.set(error instanceof Error ? error.message : 'No se pudo guardar.');
    } finally {
      this.busy.set(false);
    }
  }
}
export interface DetailData {
  title: string;
  description: string;
  rows: { label: string; value: string }[];
}
@Component({
  selector: 'app-detail-dialog',
  imports: [Icon],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<div class="modal-header">
      <div>
        <h2 id="dialog-title">{{ data.title }}</h2>
        <p id="dialog-description">{{ data.description }}</p>
      </div>
      <button class="icon-btn" (click)="ref.close()" aria-label="Cerrar diálogo">
        <app-icon name="close" />
      </button>
    </div>
    <div class="modal-body">
      @for (row of data.rows; track $index) {
        <div class="info-row">
          <span>{{ row.label }}</span
          ><strong>{{ row.value }}</strong>
        </div>
      }
      <div class="modal-actions">
        <button class="btn primary" (click)="ref.close()">Cerrar</button>
      </div>
    </div>`,
})
export class DetailDialog {
  readonly data = inject<DetailData>(DIALOG_DATA);
  readonly ref = inject(DialogRef);
}
@Injectable({ providedIn: 'root' })
export class Dialogs {
  private readonly dialog = inject(Dialog);
  form(data: FormDialogData): void {
    this.dialog.open(FormDialog, {
      data,
      panelClass: 'app-dialog',
      ariaModal: true,
      ariaLabelledBy: 'dialog-title',
      ariaDescribedBy: 'dialog-description',
      autoFocus: 'first-tabbable',
    });
  }
  detail(data: DetailData): void {
    this.dialog.open(DetailDialog, {
      data,
      panelClass: 'app-dialog',
      ariaModal: true,
      ariaLabelledBy: 'dialog-title',
      ariaDescribedBy: 'dialog-description',
    });
  }
}
export const options = (values: string[]): { value: string; label: string }[] =>
  values.map((value) => ({ value, label: value }));
