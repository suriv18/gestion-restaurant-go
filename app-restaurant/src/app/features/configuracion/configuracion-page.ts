import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { AuthSession } from '../../core/auth/auth-session';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { PageHeader } from '../../shared/ui/page-header';
import { Dialogs } from '../../shared/ui/form-dialog';
import { ConfiguracionReal } from './configuracion-real';
@Component({
  imports: [PageHeader, ReactiveFormsModule, ConfiguracionReal],
  templateUrl: './configuracion-page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConfiguracionPage {
  readonly auth = inject(AuthSession);
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  readonly error = signal('');
  readonly form = new FormGroup({
    name: new FormControl(this.store.state().settings.name, {
      nonNullable: true,
      validators: [Validators.required, Validators.pattern(/\S/)],
    }),
    branch: new FormControl(this.store.state().settings.branch, {
      nonNullable: true,
      validators: [Validators.required, Validators.pattern(/\S/)],
    }),
    address: new FormControl(this.store.state().settings.address, { nonNullable: true }),
  });
  save(): void {
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.error.set('El nombre y la sede son obligatorios.');
      return;
    }
    try {
      this.store.execute('settings', this.form.getRawValue());
      this.error.set('');
    } catch (e) {
      this.error.set(e instanceof Error ? e.message : 'No se pudo guardar.');
    }
  }
  reset(): void {
    this.dialogs.form({
      title: 'Restablecer demostración',
      description:
        'Se reemplazarán los ejemplos guardados de este navegador. Escribe RESTABLECER para confirmar.',
      danger: true,
      submit: 'Restablecer datos',
      fields: [{ key: 'confirm', label: 'Confirmación' }],
      commit: (v) => {
        if (v['confirm'] !== 'RESTABLECER') throw new Error('Escribe RESTABLECER para continuar.');
        this.store.reset();
        this.form.setValue({
          name: this.store.state().settings.name,
          branch: this.store.state().settings.branch,
          address: this.store.state().settings.address,
        });
      },
    });
  }
}
