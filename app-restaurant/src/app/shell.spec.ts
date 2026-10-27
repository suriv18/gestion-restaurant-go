import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Shell } from './shell';
describe('Shell de Mesa', () => {
  beforeEach(async () => {
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [Shell],
      providers: [provideRouter([])],
    }).compileComponents();
  });
  it('expone las 15 áreas y el modo demostración', async () => {
    const fixture = TestBed.createComponent(Shell);
    await fixture.whenStable();
    const root = fixture.nativeElement as HTMLElement;
    expect(root.querySelectorAll('nav a')).toHaveLength(15);
    expect(root.textContent).toContain('Modo demostración');
    expect(root.querySelector('a.skip-link')?.getAttribute('href')).toBe('#main');
  });
  it('abre y cierra el menú adaptable', async () => {
    const fixture = TestBed.createComponent(Shell);
    await fixture.whenStable();
    const root = fixture.nativeElement as HTMLElement;
    (root.querySelector('[aria-label="Abrir menú"]') as HTMLButtonElement).click();
    await fixture.whenStable();
    expect(root.querySelector('.sidebar')?.classList.contains('open')).toBe(true);
    (root.querySelector('[aria-label="Cerrar menú"]') as HTMLButtonElement).click();
    await fixture.whenStable();
    expect(root.querySelector('.sidebar')?.classList.contains('open')).toBe(false);
  });
});
