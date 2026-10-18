import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';
import { AuthSession } from './auth-session';
import { API_BASE } from './auth.models';

/** Solo se adjuntan credenciales al propio API (mismo origen); nunca a otros destinos. */
const isApiUrl = (url: string): boolean => url === API_BASE || url.startsWith(`${API_BASE}/`);
const isLogin = (url: string): boolean => url === `${API_BASE}/auth/login`;

/** Añade `Authorization: Bearer` a las llamadas al API cuando hay sesión. */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const token = inject(AuthSession).accessToken();
  if (!token || !isApiUrl(req.url) || isLogin(req.url)) return next(req);
  return next(req.clone({ setHeaders: { Authorization: `Bearer ${token}` } }));
};

/**
 * Si el API responde 401 a una llamada autenticada, la sesión ya no es válida (vencida,
 * revocada o usuario dado de baja): se cierra localmente y se vuelve al login. El 401 del
 * propio login significa credenciales incorrectas y no cierra nada.
 */
export const sessionExpiredInterceptor: HttpInterceptorFn = (req, next) => {
  const session = inject(AuthSession);
  return next(req).pipe(
    catchError((error: unknown) => {
      if (
        error instanceof HttpErrorResponse &&
        error.status === 401 &&
        isApiUrl(req.url) &&
        !isLogin(req.url)
      ) {
        session.expire();
      }
      return throwError(() => error);
    }),
  );
};
