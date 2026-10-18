import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { DialogRef } from '@angular/cdk/dialog';
import { FormArray, FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Icon } from '../../shared/ui/icon';
import { toCents } from '../../shared/ui/money';
@Component({
  selector: 'app-purchase-dialog',
  imports: [ReactiveFormsModule, Icon],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: ` <div class="modal-header">
      <div>
        <h2 id="purchase-title">Nueva compra</h2>
        <p>Registra los insumos y confirma la recepción cuando lleguen.</p>
      </div>
      <button class="icon-btn" aria-label="Cerrar diálogo" (click)="ref.close()">
        <app-icon name="close" />
      </button>
    </div>
    <form class="modal-body" [formGroup]="form" (ngSubmit)="save()">
      <div class="form-grid">
        <label class="field"
          >Proveedor<select formControlName="supplier">
            @for (s of suppliers; track s.id) {
              <option [value]="s.id">{{ s.name }}</option>
            }
          </select></label
        ><label class="field">Referencia de compra<input formControlName="reference" /></label>
      </div>
      <div formArrayName="lines" class="mt-6">
        @for (line of form.controls.lines.controls; track line; let index = $index) {
          <fieldset
            [formGroupName]="index"
            class="rounded-xl border border-stone-200 p-4 mb-4 bg-white"
          >
            <legend class="px-2 text-xs font-bold">Insumo {{ index + 1 }}</legend>
            <label class="field"
              >Insumo {{ index + 1
              }}<select formControlName="item" [attr.aria-label]="'Insumo ' + (index + 1)">
                @for (i of ingredients; track i.id) {
                  <option [value]="i.id">{{ i.name }} · {{ i.unit }}</option>
                }
              </select></label
            >
            <div class="form-grid">
              <label class="field"
                >Cantidad {{ index + 1
                }}<input type="number" min="0.001" step="0.001" formControlName="qty" /><small
                  >En la unidad base del insumo.</small
                ></label
              ><label class="field"
                >Costo unitario {{ index + 1 }} (S/)<input
                  inputmode="decimal"
                  formControlName="cost"
              /></label>
            </div>
            <button
              type="button"
              class="text-link mt-3"
              [disabled]="form.controls.lines.length === 1"
              (click)="form.controls.lines.removeAt(index)"
            >
              Retirar insumo {{ index + 1 }}
            </button>
          </fieldset>
        }
      </div>
      <button type="button" class="btn" (click)="addLine()">+ Agregar insumo</button>
      @if (error()) {
        <p class="form-error" role="alert">{{ error() }}</p>
      }
      <div class="modal-actions">
        <button type="button" class="btn" (click)="ref.close()">Cancelar</button
        ><button class="btn primary">Guardar borrador</button>
      </div>
    </form>`,
})
export class PurchaseDialog {
  readonly store = inject(RestaurantStore);
  readonly ref = inject(DialogRef);
  readonly error = signal('');
  readonly suppliers = this.store.state().suppliers.filter((s) => s.es_activo);
  readonly ingredients = this.store.state().ingredients.filter((i) => i.es_activo);
  readonly form = new FormGroup({
    supplier: new FormControl(String(this.suppliers[0]?.id ?? ''), {
      nonNullable: true,
      validators: [Validators.required],
    }),
    reference: new FormControl('', {
      nonNullable: true,
      validators: [Validators.required, Validators.pattern(/\S/)],
    }),
    lines: new FormArray([this.createLine()]),
  });
  private createLine() {
    return new FormGroup({
      item: new FormControl(String(this.ingredients[0]?.id ?? ''), {
        nonNullable: true,
        validators: [Validators.required],
      }),
      qty: new FormControl(1, {
        nonNullable: true,
        validators: [Validators.required, Validators.min(0.001)],
      }),
      cost: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
    });
  }
  addLine(): void {
    this.form.controls.lines.push(this.createLine());
  }
  save(): void {
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.error.set('Completa proveedor, referencia, cantidad y costo de cada insumo.');
      return;
    }
    try {
      const v = this.form.getRawValue();
      this.store.execute('purchase', {
        supplier: Number(v.supplier),
        reference: v.reference,
        lines: v.lines.map((l) => ({
          item: Number(l.item),
          qty: Number(l.qty),
          cost: toCents(l.cost),
        })),
      });
      this.ref.close(true);
    } catch (e) {
      this.error.set(e instanceof Error ? e.message : 'No se pudo guardar.');
    }
  }
}
