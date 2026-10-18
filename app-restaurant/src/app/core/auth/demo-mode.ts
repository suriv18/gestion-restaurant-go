import { Injectable, signal } from '@angular/core';

const KEY = 'app-restaurant.demo-mode';

/**
 * Modo demostración explícito: permite recorrer las pantallas con datos de ejemplo sin cuenta.
 * Se recuerda solo en la pestaña actual (sessionStorage) y NO es una sesión ni una credencial.
 */
@Injectable({ providedIn: 'root' })
export class DemoMode {
  readonly active = signal(this.read());

  enter(): void {
    this.active.set(true);
    this.write('1');
  }

  leave(): void {
    this.active.set(false);
    this.write(null);
  }

  private read(): boolean {
    try {
      return sessionStorage.getItem(KEY) === '1';
    } catch {
      return false;
    }
  }

  private write(value: string | null): void {
    try {
      if (value === null) sessionStorage.removeItem(KEY);
      else sessionStorage.setItem(KEY, value);
    } catch {
      // Sin almacenamiento disponible el modo sigue vigente solo en memoria.
    }
  }
}
