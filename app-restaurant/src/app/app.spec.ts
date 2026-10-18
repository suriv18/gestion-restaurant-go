import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';
import { routes } from './app.routes';
import { LoginPage } from './features/identidad/login/login-page';
describe('Rutas públicas', () => {
  it('abre el login al visitar la raíz, sin navegación de operación', async () => {
    TestBed.configureTestingModule({ providers: [provideRouter(routes)] });
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/', LoginPage);
    expect(harness.routeNativeElement?.querySelector('h1')?.textContent).toContain(
      'Qué bueno verte',
    );
    expect(harness.routeNativeElement?.querySelector('.sidebar')).toBeNull();
  });
});
