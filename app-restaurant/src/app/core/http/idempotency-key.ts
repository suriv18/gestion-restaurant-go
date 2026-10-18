/**
 * Clave de idempotencia (UUID v4) para una intención de escritura. Se genera una vez al abrir el
 * formulario y se reutiliza si se reintenta el mismo envío: el servidor devuelve el efecto ya
 * confirmado en lugar de repetirlo. `randomUUID` solo existe en contextos seguros, por eso el
 * respaldo con `getRandomValues`.
 */
export function newIdempotencyKey(): string {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID();
  const b = crypto.getRandomValues(new Uint8Array(16));
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('');
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}
