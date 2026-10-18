import { ChangeDetectionStrategy, Component, ElementRef, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { safeReturnUrl } from '../../../core/auth/auth-guards';
import { AuthSession } from '../../../core/auth/auth-session';
import { AuthError, AuthErrorKind } from '../../../core/auth/auth.models';
import { DemoMode } from '../../../core/auth/demo-mode';
import { Icon } from '../../../shared/ui/icon';
import { Dialogs } from '../../../shared/ui/form-dialog';
import { LoginStory } from './login-story';

type Field = 'username' | 'password';

// El servidor responde igual ante un usuario inexistente, una contraseña errónea, una cuenta
// inactiva o bloqueada; la interfaz tampoco las distingue.
const MESSAGES: Record<AuthErrorKind, string> = {
  credentials:
    'Usuario o contraseña incorrectos. Si tu cuenta está bloqueada o inactiva, consulta con la persona que administra tu restaurante.',
  invalid: 'Revisa tus datos e inténtalo de nuevo.',
  network: 'No pudimos conectar con el servidor. Revisa tu conexión e inténtalo de nuevo.',
  server: 'El servicio no está disponible en este momento. Inténtalo de nuevo en unos minutos.',
};

@Component({
  selector: 'app-login-page',
  imports: [ReactiveFormsModule, RouterLink, Icon, LoginStory],
  templateUrl: './login-page.html',
  styleUrl: './login-page.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoginPage {
  readonly year = new Date().getFullYear();
  private readonly fb = inject(FormBuilder);
  private readonly element = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly dialogs = inject(Dialogs);
  private readonly auth = inject(AuthSession);
  private readonly demo = inject(DemoMode);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  readonly showPassword = signal(false);
  readonly busy = signal(false);
  readonly message = signal('');
  readonly form = this.fb.nonNullable.group({
    username: ['', [Validators.required, Validators.maxLength(80)]],
    password: ['', Validators.required],
  });

  constructor() {
    if (this.route.snapshot.queryParamMap.get('motivo') === 'expirada') {
      this.message.set('Tu sesión venció. Inicia sesión de nuevo.');
    }
  }

  invalid(field: Field): boolean {
    const control = this.form.controls[field];
    return control.touched && control.invalid;
  }

  async submit(): Promise<void> {
    if (this.busy()) return;
    this.message.set('');
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.focus(this.form.controls.username.invalid ? 'username' : 'password');
      return;
    }
    this.busy.set(true);
    const { username, password } = this.form.getRawValue();
    try {
      await this.auth.login(username.trim(), password);
      const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl');
      await this.router.navigateByUrl(safeReturnUrl(returnUrl));
    } catch (error) {
      // Nunca se conserva la contraseña tras un intento fallido.
      this.form.controls.password.reset();
      this.showPassword.set(false);
      this.message.set(MESSAGES[error instanceof AuthError ? error.kind : 'server']);
      this.focus('password');
    } finally {
      this.busy.set(false);
    }
  }

  /** «Explorar la demo»: datos de ejemplo locales, sin cuenta. No es una sesión. */
  enterDemo(): void {
    this.demo.enter();
  }

  help(recovery = false): void {
    this.dialogs.detail({
      title: recovery ? 'Recuperar tu acceso' : 'Estamos para ayudarte',
      description: 'Esta versión de Mesa es una demostración.',
      rows: [
        {
          label: 'Acceso del equipo',
          value:
            'Tu cuenta la crea la persona que administra tu restaurante. Si no puedes entrar, ella puede ayudarte.',
        },
        {
          label: recovery ? 'Recuperación de contraseña' : '¿Quieres conocer Mesa?',
          value: recovery
            ? 'La recuperación automática aún no está disponible. Pídele a quien administra tu restaurante que te ayude. No se ha enviado ningún correo.'
            : 'Selecciona «Explorar la demo» para recorrer el salón, la cocina y la caja con datos de ejemplo.',
        },
      ],
    });
  }

  private focus(field: Field): void {
    this.element.nativeElement.querySelector<HTMLInputElement>(`#login-${field}`)?.focus();
  }
}
