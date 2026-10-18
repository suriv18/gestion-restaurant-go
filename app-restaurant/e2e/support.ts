import { Page, Route } from '@playwright/test';

/** Activa el modo demostración explícito antes de cada carga (equivale a «Explorar la demo»). */
export async function enterDemo(page: Page): Promise<void> {
  await page.addInitScript(() => sessionStorage.setItem('app-restaurant.demo-mode', '1'));
}

export const SESSION: SessionBody = {
  usuario_id: '2',
  username: 'ana',
  sucursales: ['1'],
  permisos: ['CAJA_OPERAR', 'PEDIDO_CREAR', 'USUARIO_ADMINISTRAR'],
};
export const TOKEN = 'eyJhbGciOiJIUzI1NiJ9.e2e.firma';

interface SessionBody {
  usuario_id: string;
  username: string;
  sucursales: string[];
  permisos: string[];
}

export interface ApiMock {
  /** Cuerpo de GET /sesion (cámbielo para simular otros permisos). */
  session: SessionBody;
  /** Cuerpos recibidos en POST /auth/login. */
  logins: { username: string; password: string }[];
  /** Encabezados Authorization recibidos en GET /sesion. */
  sessionAuth: (string | undefined)[];
  /** Veces que se llamó a POST /auth/logout. */
  logouts: number;
  /** Cambia la respuesta del login: 'ok' | 401 | 'network' | 500. */
  loginMode: 'ok' | 401 | 'network' | 500;
  /** Si es true, GET /sesion responde 401 (token revocado). */
  sessionRevoked: boolean;
}

/**
 * Simula el backend para pruebas deterministas del front. La integración real con el API se
 * comprueba aparte, contra el stack de Docker (ver README).
 */
export async function mockAuthApi(page: Page): Promise<ApiMock> {
  const mock: ApiMock = {
    session: SESSION,
    logins: [],
    sessionAuth: [],
    logouts: 0,
    loginMode: 'ok',
    sessionRevoked: false,
  };
  const json = (route: Route, status: number, body: unknown) =>
    route.fulfill({
      status,
      contentType: status >= 400 ? 'application/problem+json' : 'application/json',
      body: JSON.stringify(body),
    });
  await page.route('**/api/v1/auth/login', async (route) => {
    mock.logins.push(route.request().postDataJSON());
    if (mock.loginMode === 'network') return route.abort('connectionrefused');
    if (mock.loginMode === 401) {
      return json(route, 401, { status: 401, code: 'credenciales_invalidas' });
    }
    if (mock.loginMode === 500) return json(route, 500, { status: 500 });
    return json(route, 200, { access_token: TOKEN, token_type: 'Bearer', expires_in: 7200 });
  });
  await page.route('**/api/v1/sesion', async (route) => {
    mock.sessionAuth.push((await route.request().headerValue('authorization')) ?? undefined);
    if (mock.sessionRevoked) return json(route, 401, { status: 401, code: 'no_autenticado' });
    return json(route, 200, mock.session);
  });
  await page.route('**/api/v1/auth/logout', (route) => {
    mock.logouts += 1;
    return route.fulfill({ status: 204 });
  });
  return mock;
}
