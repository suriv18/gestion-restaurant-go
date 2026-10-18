import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import {
  ActivatedRouteSnapshot,
  provideRouter,
  Router,
  RouterStateSnapshot,
  UrlTree,
} from '@angular/router';
import { authGuard, guestGuard, safeReturnUrl } from './auth-guards';
import { AuthSession } from './auth-session';
import { DemoMode } from './demo-mode';

describe('Guards de acceso', () => {
  const authenticated = signal(false);
  const demo = signal(false);

  beforeEach(() => {
    authenticated.set(false);
    demo.set(false);
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: AuthSession, useValue: { isAuthenticated: authenticated } },
        { provide: DemoMode, useValue: { active: demo } },
      ],
    });
  });

  const run = (guard: typeof authGuard, url = '/salon') =>
    TestBed.runInInjectionContext(() =>
      guard({} as ActivatedRouteSnapshot, { url } as RouterStateSnapshot),
    );
  const tree = (value: unknown) => TestBed.inject(Router).serializeUrl(value as UrlTree);

  it('sin sesión ni demo redirige al login conservando el destino', () => {
    expect(tree(run(authGuard, '/caja?mesa=3'))).toBe('/login?returnUrl=%2Fcaja%3Fmesa%3D3');
  });

  it('con sesión o con demo explícita deja pasar', () => {
    authenticated.set(true);
    expect(run(authGuard)).toBe(true);
    authenticated.set(false);
    demo.set(true);
    expect(run(authGuard)).toBe(true);
  });

  it('la raíz no añade returnUrl', () => {
    expect(tree(run(authGuard, '/'))).toBe('/login');
  });

  it('con sesión iniciada el login redirige al resumen', () => {
    expect(run(guestGuard)).toBe(true);
    authenticated.set(true);
    expect(tree(run(guestGuard))).toBe('/dashboard');
  });
});

describe('safeReturnUrl', () => {
  it.each([
    ['/caja', '/caja'],
    ['/caja?mesa=3', '/caja?mesa=3'],
    ['/salon#detalle', '/salon#detalle'],
  ])('acepta rutas internas: %s', (input, expected) => {
    expect(safeReturnUrl(input)).toBe(expected);
  });

  it.each([
    [null],
    [undefined],
    [''],
    ['//evil.example.com'],
    ['https://evil.example.com'],
    ['javascript:alert(1)'],
    ['caja'],
    ['/\\evil.example.com'],
    ['/login'],
    ['/login?returnUrl=%2Fcaja'],
  ])('descarta destinos peligrosos o inútiles: %s', (input) => {
    expect(safeReturnUrl(input)).toBe('/dashboard');
  });
});
