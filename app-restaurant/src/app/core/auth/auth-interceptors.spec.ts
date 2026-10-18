import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { authInterceptor, sessionExpiredInterceptor } from './auth-interceptors';
import { AuthSession } from './auth-session';

describe('Interceptores de autenticación', () => {
  const expire = vi.fn();
  let token: string | null;
  let http: HttpClient;
  let ctrl: HttpTestingController;

  beforeEach(() => {
    expire.mockReset();
    token = 'aaa.bbb.ccc';
    TestBed.configureTestingModule({
      providers: [
        { provide: AuthSession, useValue: { accessToken: () => token, expire } },
        provideHttpClient(withInterceptors([authInterceptor, sessionExpiredInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpClient);
    ctrl = TestBed.inject(HttpTestingController);
  });
  afterEach(() => ctrl.verify());

  it('adjunta el token Bearer a las llamadas al API', () => {
    http.get('/api/v1/roles').subscribe();
    const req = ctrl.expectOne('/api/v1/roles');
    expect(req.request.headers.get('Authorization')).toBe('Bearer aaa.bbb.ccc');
    req.flush({});
  });

  it('no adjunta credenciales a otros destinos ni al propio login', () => {
    for (const url of ['https://otro.example.com/api/v1/roles', '/api/v10/x', '/assets/a.json']) {
      http.get(url).subscribe();
      expect(ctrl.expectOne(url).request.headers.has('Authorization')).toBe(false);
    }
    http.post('/api/v1/auth/login', {}).subscribe();
    const login = ctrl.expectOne('/api/v1/auth/login');
    expect(login.request.headers.has('Authorization')).toBe(false);
    login.flush({});
  });

  it('sin sesión no añade el encabezado', () => {
    token = null;
    http.get('/api/v1/sesion').subscribe({ error: () => undefined });
    const req = ctrl.expectOne('/api/v1/sesion');
    expect(req.request.headers.has('Authorization')).toBe(false);
    req.flush({}, { status: 401, statusText: 'x' });
  });

  it('un 401 del API cierra la sesión local y propaga el error', () => {
    const onError = vi.fn();
    http.get('/api/v1/roles').subscribe({ error: onError });
    ctrl.expectOne('/api/v1/roles').flush({}, { status: 401, statusText: 'x' });
    expect(expire).toHaveBeenCalledOnce();
    expect(onError).toHaveBeenCalledOnce();
  });

  it('el 401 del login (credenciales incorrectas), un 403, un 500 o un 401 ajeno no cierran sesión', () => {
    const casos: [string, number][] = [
      ['/api/v1/auth/login', 401],
      ['/api/v1/roles', 403],
      ['/api/v1/roles', 500],
      ['https://otro.example.com/x', 401],
    ];
    for (const [url, status] of casos) {
      http.get(url).subscribe({ error: () => undefined });
      ctrl.expectOne(url).flush({}, { status, statusText: 'x' });
    }
    expect(expire).not.toHaveBeenCalled();
  });
});
