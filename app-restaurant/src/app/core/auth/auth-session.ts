import { HttpErrorResponse } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AuthApi } from './auth-api';
import { AuthError, SessionResponse, SessionUser } from './auth.models';
import { DemoMode } from './demo-mode';

const TOKEN_KEY = 'app-restaurant.session';

/**
 * Sesión de la persona conectada. Único lugar que conoce el token.
 *
 * El token (JWT) vive en memoria y se copia a sessionStorage solo para sobrevivir a una
 * recarga de la pestaña; se borra al cerrar la pestaña, al cerrar sesión y ante cualquier 401.
 * Nunca se guardan la contraseña ni datos del usuario. Al arrancar, el token se valida contra
 * `GET /sesion`: el servidor es la autoridad (revocación, permisos y sedes vigentes).
 */
@Injectable({ providedIn: 'root' })
export class AuthSession {
  private readonly api = inject(AuthApi);
  private readonly router = inject(Router);
  private readonly demo = inject(DemoMode);

  private token: string | null = this.readToken();
  private readonly userState = signal<SessionUser | null>(null);

  readonly user = this.userState.asReadonly();
  readonly isAuthenticated = computed(() => this.userState() !== null);

  /**
   * Para decidir qué mostrar en la interfaz. NO es seguridad: el servidor vuelve a comprobar
   * el permiso en cada llamada.
   */
  hasPermission(permission: string): boolean {
    return this.userState()?.permissions.includes(permission) ?? false;
  }

  /** Lo usa únicamente el interceptor. */
  accessToken(): string | null {
    return this.token;
  }

  /** Autentica y deja la sesión lista. Lanza AuthError si falla. */
  async login(username: string, password: string): Promise<void> {
    let access: string;
    try {
      access = (await firstValueFrom(this.api.login(username, password))).access_token;
    } catch (error) {
      throw toAuthError(error);
    }
    this.setToken(access);
    try {
      this.userState.set(toUser(await firstValueFrom(this.api.session())));
    } catch (error) {
      this.clear();
      throw toAuthError(error);
    }
    this.demo.leave();
  }

  /** Recupera la sesión de una recarga: valida el token guardado. Nunca lanza. */
  async restore(): Promise<void> {
    if (!this.token) return;
    try {
      this.userState.set(toUser(await firstValueFrom(this.api.session())));
      this.demo.leave();
    } catch {
      this.clear();
    }
  }

  /** Cierra la sesión: revoca en el servidor (mejor esfuerzo) y limpia siempre lo local. */
  async logout(): Promise<void> {
    if (this.token) {
      try {
        await firstValueFrom(this.api.logout());
      } catch {
        // Aunque el servidor no responda, la sesión local se cierra igual.
      }
    }
    this.clear();
  }

  /** El servidor rechazó el token (401): limpia y vuelve al login. */
  expire(): void {
    const wasAuthenticated = this.isAuthenticated();
    this.clear();
    if (wasAuthenticated) {
      void this.router.navigate(['/login'], {
        queryParams: { motivo: 'expirada', returnUrl: this.router.url },
      });
    }
  }

  private setToken(value: string): void {
    this.token = value;
    try {
      sessionStorage.setItem(TOKEN_KEY, value);
    } catch {
      // Sin almacenamiento la sesión dura lo que dure la pestaña en memoria.
    }
  }

  private clear(): void {
    this.token = null;
    this.userState.set(null);
    try {
      sessionStorage.removeItem(TOKEN_KEY);
    } catch {
      // Nada que limpiar.
    }
  }

  private readToken(): string | null {
    try {
      return sessionStorage.getItem(TOKEN_KEY);
    } catch {
      return null;
    }
  }
}

function toUser(s: SessionResponse): SessionUser {
  return {
    id: s.usuario_id,
    username: s.username,
    branches: s.sucursales,
    permissions: s.permisos,
  };
}

function toAuthError(error: unknown): AuthError {
  if (error instanceof AuthError) return error;
  if (error instanceof HttpErrorResponse) {
    if (error.status === 0) return new AuthError('network');
    if (error.status === 401) return new AuthError('credentials');
    if (error.status === 400 || error.status === 422) return new AuthError('invalid');
  }
  return new AuthError('server');
}
