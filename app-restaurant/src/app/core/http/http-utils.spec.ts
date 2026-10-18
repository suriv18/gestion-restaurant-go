import { HttpErrorResponse } from '@angular/common/http';
import { newIdempotencyKey } from './idempotency-key';
import { describeProblem } from './problem';

const http = (status: number, error?: unknown) => new HttpErrorResponse({ status, error });

describe('describeProblem', () => {
  it('usa el detalle redactado por el servidor en los rechazos esperados (4xx)', () => {
    const detail = 'la operación dejaría al sistema sin ningún administrador activo';
    expect(describeProblem(http(422, { code: 'ultimo_administrador', detail }))).toBe(detail);
    expect(describeProblem(http(409, { code: 'registro_duplicado', detail: 'ya existe' }))).toBe(
      'ya existe',
    );
  });

  it('traduce los códigos que necesitan otro texto para la pantalla', () => {
    expect(describeProblem(http(403, { code: 'sin_permiso', detail: 'x' }))).toContain(
      'No tienes permiso',
    );
    expect(describeProblem(http(409, { code: 'idempotency_key_reutilizada' }))).toContain(
      'ya se registró',
    );
  });

  it('nunca muestra detalles de un fallo técnico (5xx)', () => {
    const text = describeProblem(http(500, { detail: 'pq: relation "x" does not exist' }));
    expect(text).not.toContain('pq:');
    expect(text).toContain('no está disponible');
  });

  it('distingue la falta de conexión y lo inesperado', () => {
    expect(describeProblem(http(0))).toContain('No pudimos conectar');
    expect(describeProblem(new Error('boom'))).toContain('no está disponible');
    expect(describeProblem(http(404, undefined))).toContain('no está disponible');
  });
});

describe('newIdempotencyKey', () => {
  const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

  it('genera UUID v4 distintos', () => {
    const keys = new Set(Array.from({ length: 50 }, () => newIdempotencyKey()));
    expect(keys.size).toBe(50);
    for (const k of keys) expect(k).toMatch(UUID);
  });

  it('también funciona sin randomUUID (contextos no seguros)', () => {
    const original = crypto.randomUUID;
    Object.defineProperty(crypto, 'randomUUID', { value: undefined, configurable: true });
    try {
      expect(newIdempotencyKey()).toMatch(UUID);
    } finally {
      Object.defineProperty(crypto, 'randomUUID', { value: original, configurable: true });
    }
  });
});
