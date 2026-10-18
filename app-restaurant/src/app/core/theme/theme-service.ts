import { DOCUMENT } from '@angular/common';
import { computed, DestroyRef, effect, inject, Injectable, signal } from '@angular/core';

export type ThemePreference = 'light' | 'dark' | 'system';
export const THEME_KEY = 'app-restaurant.theme';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  private readonly document = inject(DOCUMENT);
  private readonly window = this.document.defaultView;
  private readonly media = this.window?.matchMedia?.('(prefers-color-scheme: dark)');
  private readonly systemDark = signal(this.media?.matches ?? false);
  readonly preference = signal<ThemePreference>(this.read());
  readonly resolved = computed(() =>
    this.preference() === 'system' ? (this.systemDark() ? 'dark' : 'light') : this.preference(),
  );
  readonly storageWarning = signal('');

  constructor() {
    const listener = (event: MediaQueryListEvent) => this.systemDark.set(event.matches);
    this.media?.addEventListener('change', listener);
    inject(DestroyRef).onDestroy(() => this.media?.removeEventListener('change', listener));
    effect(() => {
      this.document.documentElement.dataset['theme'] = this.resolved();
      this.document.documentElement.style.colorScheme = this.resolved();
    });
  }

  select(theme: ThemePreference): void {
    this.preference.set(theme);
    this.storageWarning.set('');
    try {
      this.window?.localStorage.setItem(THEME_KEY, theme);
    } catch {
      this.storageWarning.set('El tema se aplicó, pero este navegador no permite guardarlo.');
    }
  }

  private read(): ThemePreference {
    try {
      const value = this.window?.localStorage.getItem(THEME_KEY);
      return value === 'light' || value === 'dark' || value === 'system' ? value : 'system';
    } catch {
      return 'system';
    }
  }
}
