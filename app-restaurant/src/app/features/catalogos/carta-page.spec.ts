import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { AuthSession } from '../../core/auth/auth-session';
import { CartaPage } from './carta-page';
import { CartaReal } from './carta-real';
import { CartaStore } from './data-access/carta-store';

function setup(autenticado: boolean) {
  TestBed.configureTestingModule({
    imports: [CartaPage],
    providers: [{ provide: AuthSession, useValue: { isAuthenticated: signal(autenticado) } }],
  });
  TestBed.overrideComponent(CartaReal, {
    set: {
      template: '<p id="real"></p>',
      imports: [],
      providers: [{ provide: CartaStore, useValue: { load: vi.fn() } }],
    },
  });
  const fixture = TestBed.createComponent(CartaPage);
  fixture.detectChanges();
  return fixture.nativeElement as HTMLElement;
}

describe('Contenedor de Carta', () => {
  it('con sesión real muestra la pantalla real y no la demostración', () => {
    const el = setup(true);
    expect(el.querySelector('#real')).not.toBeNull();
    expect(el.querySelector('app-carta-demo')).toBeNull();
  });

  it('sin sesión conserva la pantalla de demostración', () => {
    const el = setup(false);
    expect(el.querySelector('#real')).toBeNull();
    expect(el.querySelector('app-carta-demo')).not.toBeNull();
    expect(el.textContent).toContain('El sabor de tu restaurante');
  });
});
