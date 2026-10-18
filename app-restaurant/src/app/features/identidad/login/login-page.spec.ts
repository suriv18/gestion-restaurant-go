import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';
import { AuthSession } from '../../../core/auth/auth-session';
import { AuthError } from '../../../core/auth/auth.models';
import { DemoMode } from '../../../core/auth/demo-mode';
import { Dialogs } from '../../../shared/ui/form-dialog';
import { LoginPage } from './login-page';

describe('Acceso a Mesa', () => {
  const detail = vi.fn();
  const login = vi.fn();
  const enter = vi.fn();
  let navigateByUrl: ReturnType<typeof vi.spyOn>;

  function setup(query: Record<string, string> = {}) {
    TestBed.configureTestingModule({
      imports: [LoginPage],
      providers: [
        provideRouter([]),
        { provide: Dialogs, useValue: { detail } },
        { provide: AuthSession, useValue: { login } },
        { provide: DemoMode, useValue: { enter } },
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { queryParamMap: convertToParamMap(query) } },
        },
      ],
    });
    navigateByUrl = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
    return TestBed.createComponent(LoginPage);
  }

  beforeEach(() => {
    detail.mockReset();
    login.mockReset();
    enter.mockReset();
  });

  it('valida campos requeridos y relaciona los errores con sus controles', async () => {
    const fixture = setup();
    await fixture.whenStable();
    await fixture.componentInstance.submit();
    await fixture.whenStable();
    const root = fixture.nativeElement as HTMLElement;
    expect(root.querySelector('#login-username')?.getAttribute('aria-invalid')).toBe('true');
    expect(root.querySelector('#login-username')?.getAttribute('aria-describedby')).toBe(
      'username-error',
    );
    expect(root.querySelector('#username-error')?.textContent).toContain('Ingresa tu usuario');
    expect(root.querySelector('#password-error')?.textContent).toContain('Ingresa tu contraseña');
    expect(login).not.toHaveBeenCalled();
    expect(fixture.componentInstance.message()).toBe('');
  });

  it('acepta cualquier usuario (no solo correos) y no aplica reglas de registro a la contraseña', async () => {
    login.mockResolvedValue(undefined);
    const page = setup().componentInstance;
    page.form.setValue({ username: 'ana.perez', password: 'a' });
    await page.submit();
    expect(page.invalid('username')).toBe(false);
    expect(page.form.controls.password.valid).toBe(true);
    expect(login).toHaveBeenCalledWith('ana.perez', 'a');
  });

  it('permite mostrar y ocultar la contraseña sin cambiar su valor', async () => {
    const fixture = setup();
    fixture.componentInstance.form.controls.password.setValue('ejemplo');
    await fixture.whenStable();
    const root = fixture.nativeElement as HTMLElement;
    (root.querySelector('[aria-label="Mostrar contraseña"]') as HTMLButtonElement).click();
    await fixture.whenStable();
    expect(root.querySelector('#login-password')?.getAttribute('type')).toBe('text');
    (root.querySelector('[aria-label="Ocultar contraseña"]') as HTMLButtonElement).click();
    await fixture.whenStable();
    expect(root.querySelector('#login-password')?.getAttribute('type')).toBe('password');
    expect(fixture.componentInstance.form.controls.password.value).toBe('ejemplo');
  });

  it('inicia sesión con el usuario recortado y va al destino interno pedido', async () => {
    login.mockResolvedValue(undefined);
    const page = setup({ returnUrl: '/caja' }).componentInstance;
    page.form.setValue({ username: '  ana  ', password: 'clave-de-prueba' });
    await page.submit();
    expect(login).toHaveBeenCalledWith('ana', 'clave-de-prueba');
    expect(navigateByUrl).toHaveBeenCalledWith('/caja');
    expect(page.busy()).toBe(false);
  });

  it('sin destino va al resumen y descarta destinos externos', async () => {
    login.mockResolvedValue(undefined);
    const sin = setup().componentInstance;
    sin.form.setValue({ username: 'ana', password: 'x' });
    await sin.submit();
    expect(navigateByUrl).toHaveBeenLastCalledWith('/dashboard');
    TestBed.resetTestingModule();
    const externo = setup({ returnUrl: '//evil.example.com' }).componentInstance;
    externo.form.setValue({ username: 'ana', password: 'x' });
    await externo.submit();
    expect(navigateByUrl).toHaveBeenLastCalledWith('/dashboard');
  });

  it('ante credenciales incorrectas descarta la contraseña, avisa sin revelar el motivo y no navega', async () => {
    login.mockRejectedValue(new AuthError('credentials'));
    const storage = vi.spyOn(Storage.prototype, 'setItem');
    const page = setup().componentInstance;
    page.form.setValue({ username: 'ana', password: 'incorrecta' });
    page.showPassword.set(true);
    await page.submit();
    expect(page.message()).toContain('Usuario o contraseña incorrectos');
    expect(page.form.controls.password.value).toBe('');
    expect(page.form.controls.username.value).toBe('ana');
    expect(page.showPassword()).toBe(false);
    expect(page.busy()).toBe(false);
    expect(navigateByUrl).not.toHaveBeenCalled();
    expect(storage).not.toHaveBeenCalled();
    storage.mockRestore();
  });

  it('distingue la falta de conexión de un fallo del servicio', async () => {
    const page = setup().componentInstance;
    login.mockRejectedValueOnce(new AuthError('network'));
    page.form.setValue({ username: 'ana', password: 'x' });
    await page.submit();
    expect(page.message()).toContain('No pudimos conectar');
    login.mockRejectedValueOnce(new AuthError('server'));
    page.form.setValue({ username: 'ana', password: 'x' });
    await page.submit();
    expect(page.message()).toContain('no está disponible');
    login.mockRejectedValueOnce(new Error('inesperado'));
    page.form.setValue({ username: 'ana', password: 'x' });
    await page.submit();
    expect(page.message()).toContain('no está disponible');
  });

  it('ignora un segundo envío mientras el primero sigue en curso', async () => {
    let resolve!: () => void;
    login.mockReturnValue(new Promise<void>((r) => (resolve = r)));
    const page = setup().componentInstance;
    page.form.setValue({ username: 'ana', password: 'x' });
    const first = page.submit();
    expect(page.busy()).toBe(true);
    await page.submit();
    expect(login).toHaveBeenCalledTimes(1);
    resolve();
    await first;
    expect(page.busy()).toBe(false);
  });

  it('avisa cuando se llega por una sesión vencida', () => {
    const page = setup({ motivo: 'expirada' }).componentInstance;
    expect(page.message()).toBe('Tu sesión venció. Inicia sesión de nuevo.');
  });

  it('«Explorar la demo» activa el modo demostración explícito', () => {
    setup().componentInstance.enterDemo();
    expect(enter).toHaveBeenCalledOnce();
  });

  it('explica que la recuperación aún no envía correos', () => {
    const page = setup().componentInstance;
    page.help(true);
    expect(detail).toHaveBeenCalledWith(expect.objectContaining({ title: 'Recuperar tu acceso' }));
    expect(detail.mock.calls[0][0].rows[1].value).toContain('No se ha enviado ningún correo');
  });
});
