import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthSession } from './auth-session';
import { DemoMode } from './demo-mode';

/** Pantallas de operación: exigen sesión, o el modo demostración explícito. */
export const authGuard: CanActivateFn = (_route, state) => {
  if (inject(AuthSession).isAuthenticated() || inject(DemoMode).active()) return true;
  const returnUrl = state.url === '/' ? undefined : state.url;
  return inject(Router).createUrlTree(['/login'], { queryParams: returnUrl ? { returnUrl } : {} });
};

/** Con sesión iniciada no tiene sentido ver el login. */
export const guestGuard: CanActivateFn = () =>
  inject(AuthSession).isAuthenticated() ? inject(Router).createUrlTree(['/dashboard']) : true;

/**
 * Acepta solo rutas internas como destino tras el login. Rechaza URLs absolutas o con
 * esquema (//host, https://...), que permitirían una redirección abierta.
 */
export function safeReturnUrl(value: string | null | undefined, fallback = '/dashboard'): string {
  if (!value || !value.startsWith('/') || value.startsWith('//') || value.includes('\\')) {
    return fallback;
  }
  if (value === '/login' || value.startsWith('/login?')) return fallback;
  return value;
}
