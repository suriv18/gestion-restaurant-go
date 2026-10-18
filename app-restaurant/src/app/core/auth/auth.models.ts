/** Base de la API: siempre relativa (mismo origen). En desarrollo la reenvía `proxy.conf.mjs`. */
export const API_BASE = '/api/v1';

/** Respuesta de `POST /auth/login` (contrato OpenAPI `Token`). */
export interface TokenResponse {
  access_token: string;
  token_type: 'Bearer';
  expires_in: number;
}

/** Respuesta de `GET /sesion` (contrato OpenAPI `SesionActual`). Los ids viajan como cadenas. */
export interface SessionResponse {
  usuario_id: string;
  username: string;
  sucursales: string[];
  permisos: string[];
}

/** Proyección de UI del usuario autenticado. */
export interface SessionUser {
  readonly id: string;
  readonly username: string;
  readonly branches: readonly string[];
  readonly permissions: readonly string[];
}

export type AuthErrorKind =
  /** 401: usuario o contraseña incorrectos, cuenta inactiva o bloqueada (el servidor no distingue). */
  | 'credentials'
  /** 400/422: la solicitud no es válida. */
  | 'invalid'
  /** Sin respuesta del servidor. */
  | 'network'
  /** 5xx u otro fallo inesperado. */
  | 'server';

/** Error de autenticación ya clasificado, sin datos del servidor que filtrar a la UI. */
export class AuthError extends Error {
  constructor(readonly kind: AuthErrorKind) {
    super(kind);
    this.name = 'AuthError';
  }
}
