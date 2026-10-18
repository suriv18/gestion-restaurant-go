import { HttpErrorResponse } from '@angular/common/http';

/** Problem Details (RFC 9457) tal como los devuelve el backend. */
export interface Problem {
  status?: number;
  code?: string;
  detail?: string;
}

const GENERIC_NETWORK =
  'No pudimos conectar con el servidor. Revisa tu conexión e inténtalo de nuevo.';
const GENERIC_SERVER =
  'El servicio no está disponible en este momento. Inténtalo de nuevo en unos minutos.';

// Casos en que el mensaje del servidor no es el más útil para quien usa la pantalla.
const BY_CODE: Record<string, string> = {
  sin_permiso: 'No tienes permiso para realizar esta acción.',
  no_autenticado: 'Tu sesión venció. Inicia sesión de nuevo.',
  idempotency_key_reutilizada:
    'Esta operación ya se registró con otros datos. Cierra el formulario y revisa la lista antes de intentarlo de nuevo.',
  idempotency_en_curso: 'La operación sigue en curso. Espera un momento y revisa la lista.',
};

/**
 * Convierte un error HTTP en un mensaje para la persona usuaria. Los rechazos esperados (4xx)
 * traen un `detail` ya redactado para ella; los fallos técnicos nunca muestran detalles internos.
 */
export function describeProblem(error: unknown): string {
  if (!(error instanceof HttpErrorResponse)) return GENERIC_SERVER;
  if (error.status === 0) return GENERIC_NETWORK;
  const problem = (error.error ?? {}) as Problem;
  if (problem.code && BY_CODE[problem.code]) return BY_CODE[problem.code];
  if (error.status >= 400 && error.status < 500 && problem.detail) return problem.detail;
  return GENERIC_SERVER;
}
