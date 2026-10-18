import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { AuthSession } from './auth-session';
import { AuthError } from './auth.models';
import { DemoMode } from './demo-mode';

const TOKEN = 'aaa.bbb.ccc';
const KEY = 'app-restaurant.session';
const SESSION = { usuario_id: '2', username: 'ana', sucursales: ['1'], permisos: ['PEDIDO_CREAR'] };

describe('Sesión de autenticación', () => {
  let http: HttpTestingController;

  function setup(): AuthSession {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    return TestBed.inject(AuthSession);
  }

  /** Espera a que el siguiente paso asíncrono de la sesión emita su petición. */
  const next = (url: string) => vi.waitFor(() => http.expectOne(url));

  beforeEach(() => sessionStorage.clear());
  afterEach(() => http.verify());

  it('inicia sesión: pide el token, lo valida con /sesion y guarda solo el token', async () => {
    const session = setup();
    const pending = session.login('ana', 'clave-de-prueba');
    const login = http.expectOne('/api/v1/auth/login');
    expect(login.request.method).toBe('POST');
    expect(login.request.body).toEqual({ username: 'ana', password: 'clave-de-prueba' });
    login.flush({ access_token: TOKEN, token_type: 'Bearer', expires_in: 7200 });
    (await next('/api/v1/sesion')).flush(SESSION);
    await pending;

    expect(session.isAuthenticated()).toBe(true);
    expect(session.user()).toEqual({
      id: '2',
      username: 'ana',
      branches: ['1'],
      permissions: ['PEDIDO_CREAR'],
    });
    expect(session.accessToken()).toBe(TOKEN);
    expect(sessionStorage.getItem(KEY)).toBe(TOKEN);
    // Nunca se persiste la contraseña ni datos del usuario.
    expect(JSON.stringify({ ...sessionStorage })).not.toContain('clave-de-prueba');
    expect(JSON.stringify({ ...localStorage })).not.toContain('clave-de-prueba');
  });

  it('sale de la demo al iniciar sesión', async () => {
    const session = setup();
    TestBed.inject(DemoMode).enter();
    const pending = session.login('ana', 'x');
    http.expectOne('/api/v1/auth/login').flush({ access_token: TOKEN });
    (await next('/api/v1/sesion')).flush(SESSION);
    await pending;
    expect(TestBed.inject(DemoMode).active()).toBe(false);
  });

  it.each([
    [401, 'credentials'],
    [400, 'invalid'],
    [422, 'invalid'],
    [500, 'server'],
    [503, 'server'],
  ])('un %i al iniciar sesión se clasifica como «%s» y no deja sesión', async (status, kind) => {
    const session = setup();
    const pending = session.login('ana', 'mala');
    http.expectOne('/api/v1/auth/login').flush({ code: 'x' }, { status, statusText: 'x' });
    await expect(pending).rejects.toMatchObject({ kind });
    await expect(pending).rejects.toBeInstanceOf(AuthError);
    expect(session.isAuthenticated()).toBe(false);
    expect(session.accessToken()).toBeNull();
    expect(sessionStorage.getItem(KEY)).toBeNull();
  });

  it('sin conexión se clasifica como «network»', async () => {
    const session = setup();
    const pending = session.login('ana', 'x');
    http.expectOne('/api/v1/auth/login').error(new ProgressEvent('error'));
    await expect(pending).rejects.toMatchObject({ kind: 'network' });
  });

  it('si /sesion falla tras obtener el token, no deja una sesión a medias', async () => {
    const session = setup();
    const pending = session.login('ana', 'x');
    http.expectOne('/api/v1/auth/login').flush({ access_token: TOKEN });
    (await next('/api/v1/sesion')).flush({}, { status: 500, statusText: 'x' });
    await expect(pending).rejects.toMatchObject({ kind: 'server' });
    expect(session.isAuthenticated()).toBe(false);
    expect(sessionStorage.getItem(KEY)).toBeNull();
  });

  it('restaura una recarga validando el token guardado contra el servidor', async () => {
    sessionStorage.setItem(KEY, TOKEN);
    const session = setup();
    const pending = session.restore();
    http.expectOne('/api/v1/sesion').flush(SESSION);
    await pending;
    expect(session.isAuthenticated()).toBe(true);
    expect(session.user()?.username).toBe('ana');
  });

  it('descarta el token guardado si el servidor ya no lo acepta', async () => {
    sessionStorage.setItem(KEY, TOKEN);
    const session = setup();
    const pending = session.restore();
    http.expectOne('/api/v1/sesion').flush({}, { status: 401, statusText: 'x' });
    await pending;
    expect(session.isAuthenticated()).toBe(false);
    expect(sessionStorage.getItem(KEY)).toBeNull();
    expect(session.accessToken()).toBeNull();
  });

  it('sin token guardado no hace ninguna petición al restaurar', async () => {
    const session = setup();
    await session.restore();
    http.expectNone('/api/v1/sesion');
    expect(session.isAuthenticated()).toBe(false);
  });

  it('cierra sesión: revoca en el servidor y limpia lo local', async () => {
    sessionStorage.setItem(KEY, TOKEN);
    const session = setup();
    const restoring = session.restore();
    http.expectOne('/api/v1/sesion').flush(SESSION);
    await restoring;
    const pending = session.logout();
    const req = http.expectOne('/api/v1/auth/logout');
    expect(req.request.method).toBe('POST');
    req.flush(null, { status: 204, statusText: 'No Content' });
    await pending;
    expect(session.isAuthenticated()).toBe(false);
    expect(sessionStorage.getItem(KEY)).toBeNull();
  });

  it('cierra la sesión local aunque el servidor no responda', async () => {
    sessionStorage.setItem(KEY, TOKEN);
    const session = setup();
    const restoring = session.restore();
    http.expectOne('/api/v1/sesion').flush(SESSION);
    await restoring;
    const pending = session.logout();
    http.expectOne('/api/v1/auth/logout').error(new ProgressEvent('error'));
    await pending;
    expect(session.isAuthenticated()).toBe(false);
    expect(session.accessToken()).toBeNull();
    expect(sessionStorage.getItem(KEY)).toBeNull();
  });

  it('al vencer la sesión limpia y manda al login con el motivo y el destino', async () => {
    sessionStorage.setItem(KEY, TOKEN);
    const session = setup();
    const restoring = session.restore();
    http.expectOne('/api/v1/sesion').flush(SESSION);
    await restoring;
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    session.expire();
    expect(session.isAuthenticated()).toBe(false);
    expect(sessionStorage.getItem(KEY)).toBeNull();
    expect(navigate).toHaveBeenCalledWith(['/login'], {
      queryParams: { motivo: 'expirada', returnUrl: expect.any(String) },
    });
  });

  it('expirar sin sesión no navega (evita bucles en el arranque)', () => {
    const session = setup();
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    session.expire();
    expect(navigate).not.toHaveBeenCalled();
  });
});
