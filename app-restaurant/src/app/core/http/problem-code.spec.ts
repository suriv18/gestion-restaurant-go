import { HttpErrorResponse } from '@angular/common/http';
import { problemCode } from './problem';

describe('Código del problema', () => {
  it('devuelve el código del cuerpo problem+json', () => {
    const error = new HttpErrorResponse({ status: 409, error: { code: 'nombre_duplicado' } });
    expect(problemCode(error)).toBe('nombre_duplicado');
  });

  it('devuelve undefined si no hay código o el error no es HTTP', () => {
    expect(problemCode(new HttpErrorResponse({ status: 500 }))).toBeUndefined();
    expect(problemCode(new Error('x'))).toBeUndefined();
  });
});
