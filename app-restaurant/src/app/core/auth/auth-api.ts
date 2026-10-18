import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE, SessionResponse, TokenResponse } from './auth.models';

/** Acceso HTTP a los endpoints de autenticación. No guarda estado. */
@Injectable({ providedIn: 'root' })
export class AuthApi {
  private readonly http = inject(HttpClient);

  login(username: string, password: string): Observable<TokenResponse> {
    return this.http.post<TokenResponse>(`${API_BASE}/auth/login`, { username, password });
  }

  /** Requiere el encabezado Authorization (lo añade el interceptor). */
  session(): Observable<SessionResponse> {
    return this.http.get<SessionResponse>(`${API_BASE}/sesion`);
  }

  /** Revoca la sesión en el servidor. */
  logout(): Observable<void> {
    return this.http.post<void>(`${API_BASE}/auth/logout`, null);
  }
}
