import { describeProblem, problemCode } from '../../core/http/problem';
import type { FieldProblem } from '../ui/form-dialog';

/**
 * Traduce el error del servidor para un diálogo de formulario: si su código pertenece a un campo
 * (`porCodigo`), el mensaje se pinta bajo ese campo; si no, queda como aviso general del diálogo.
 */
export function rechazo(error: unknown, porCodigo: Record<string, string>): string | FieldProblem {
  const message = describeProblem(error);
  const code = problemCode(error);
  const field = code ? porCodigo[code] : undefined;
  return field ? { field, message } : message;
}
