import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { AuthSession } from '../../core/auth/auth-session';
import { ConfiguracionPage } from './configuracion-page';
import { ConfiguracionReal } from './configuracion-real';
import { ConfiguracionStore } from './data-access/configuracion-store';

function setup(autenticado: boolean) {
  TestBed.configureTestingModule({
    imports: [ConfiguracionPage],
    providers: [{ provide: AuthSession, useValue: { isAuthenticated: signal(autenticado) } }],
  });
  TestBed.overrideComponent(ConfiguracionReal, {
    set: {
      template: '<p id="real"></p>',
      imports: [],
      providers: [{ provide: ConfiguracionStore, useValue: { load: vi.fn() } }],
    },
  });
  const fixture = TestBed.createComponent(ConfiguracionPage);
  fixture.detectChanges();
  return fixture.nativeElement as HTMLElement;
}

describe('Contenedor de Configuración', () => {
  it('con sesión real muestra la pantalla real y no la demostración', () => {
    const el = setup(true);
    expect(el.querySelector('#real')).not.toBeNull();
    expect(el.textContent).not.toContain('Restablecer demostración');
  });

  it('sin sesión conserva la pantalla de demostración', () => {
    const el = setup(false);
    expect(el.querySelector('#real')).toBeNull();
    expect(el.textContent).toContain('Restablecer demostración');
  });
});
