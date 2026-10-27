import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { AuthSession } from '../../core/auth/auth-session';
import { CartaDemo } from './carta-demo';
import { CartaReal } from './carta-real';

@Component({
  imports: [CartaReal, CartaDemo],
  changeDetection: ChangeDetectionStrategy.OnPush,
  // Con sesión real se muestra la carta del servidor; en modo demostración, los datos de ejemplo.
  template: `@if (auth.isAuthenticated()) {
      <app-carta-real />
    } @else {
      <app-carta-demo />
    }`,
})
export class CartaPage {
  readonly auth = inject(AuthSession);
}
