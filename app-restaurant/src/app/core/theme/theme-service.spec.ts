import { TestBed } from '@angular/core/testing';
import { THEME_KEY, ThemeService } from './theme-service';

describe('Preferencia de apariencia', () => {
  let change: (event: { matches: boolean }) => void;
  const remove = vi.fn();
  beforeEach(() => {
    localStorage.clear();
    remove.mockClear();
    vi.stubGlobal('matchMedia', () => ({
      matches: false,
      addEventListener: (_name: string, listener: typeof change) => {
        change = listener;
      },
      removeEventListener: remove,
    }));
  });
  afterEach(() => {
    TestBed.resetTestingModule();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    document.documentElement.removeAttribute('data-theme');
    document.documentElement.style.colorScheme = '';
  });

  it('recupera una preferencia guardada y la aplica al documento', () => {
    localStorage.setItem(THEME_KEY, 'dark');
    const theme = TestBed.inject(ThemeService);
    TestBed.tick();
    expect(theme.preference()).toBe('dark');
    expect(document.documentElement.dataset['theme']).toBe('dark');
    expect(document.documentElement.style.colorScheme).toBe('dark');
  });

  it('sigue los cambios del sistema solo mientras esa opción está seleccionada', () => {
    const theme = TestBed.inject(ThemeService);
    expect(theme.resolved()).toBe('light');
    change({ matches: true });
    expect(theme.resolved()).toBe('dark');
    theme.select('light');
    change({ matches: false });
    change({ matches: true });
    expect(theme.resolved()).toBe('light');
    theme.select('system');
    expect(theme.resolved()).toBe('dark');
    expect(localStorage.getItem(THEME_KEY)).toBe('system');
  });

  it('descarta valores de almacenamiento desconocidos', () => {
    localStorage.setItem(THEME_KEY, 'unexpected');
    expect(TestBed.inject(ThemeService).preference()).toBe('system');
  });

  it('aplica el tema en memoria e informa si no puede persistirlo', () => {
    const theme = TestBed.inject(ThemeService);
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('Storage disabled');
    });
    theme.select('dark');
    TestBed.tick();
    expect(document.documentElement.dataset['theme']).toBe('dark');
    expect(theme.storageWarning()).toContain('no permite guardarlo');
  });

  it('admite navegadores con lectura de almacenamiento bloqueada', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('Storage disabled');
    });
    expect(TestBed.inject(ThemeService).preference()).toBe('system');
  });

  it('elimina el listener del sistema al destruir el servicio', () => {
    TestBed.inject(ThemeService);
    TestBed.resetTestingModule();
    expect(remove).toHaveBeenCalledWith('change', change);
  });
});
