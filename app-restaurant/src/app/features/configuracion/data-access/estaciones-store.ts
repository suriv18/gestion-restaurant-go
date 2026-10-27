import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem } from '../../../core/http/problem';
import { rechazo } from '../../../shared/forms/rechazo';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import type { Estacion } from '../../catalogos/data-access/carta.models';
import { EstacionesApi } from '../../catalogos/data-access/estaciones-api';

export const ESTACION_VER = 'ESTACION_VER';
export const ESTACION_ADMINISTRAR = 'ESTACION_ADMINISTRAR';

export interface EstacionesNotice {
  kind: 'ok' | 'error';
  text: string;
}

/** Código de rechazo del servidor -> campo del formulario. */
export const CAMPO_ESTACION: Record<string, string> = {
  estacion_duplicada: 'codigo',
  codigo_invalido: 'codigo',
  nombre_invalido: 'nombre',
};

const CODIGO = /^[A-Za-z0-9][A-Za-z0-9_-]{0,19}$/;
const MAX_NOMBRE = 80;

type Resultado = Promise<string | FieldProblem | undefined>;

/**
 * Estaciones de preparación de una sede (sección de Configuración). Almacén propio, provisto por
 * `ConfiguracionReal`: la sede la decide el selector de Salón y llega por `cargar(sedeId)`.
 * Las bajas se listan siempre (el servidor devuelve activas y bajas); «Incluir bajas» es de la vista.
 */
@Injectable()
export class EstacionesStore {
  private readonly api = inject(EstacionesApi);
  private readonly auth = inject(AuthSession);

  readonly status = signal<'loading' | 'ready' | 'error'>('loading');
  readonly error = signal('');
  readonly notice = signal<EstacionesNotice | null>(null);
  readonly busyId = signal<string | null>(null);
  readonly estaciones = signal<Estacion[]>([]);
  private readonly sede = signal<string | null>(null);
  private secuencia = 0;

  readonly puedeAdministrar = computed(() => this.auth.hasPermission(ESTACION_ADMINISTRAR));

  /** Ordenadas por código (comparación natural). */
  readonly ordenadas = computed(() =>
    [...this.estaciones()].sort((a, b) =>
      a.codigo.localeCompare(b.codigo, 'es', { numeric: true }),
    ),
  );

  /** Carga las estaciones de la sede (null = no hay sede: lista vacía sin llamar). */
  async cargar(sedeId: string | null): Promise<void> {
    this.sede.set(sedeId);
    const id = ++this.secuencia;
    this.error.set('');
    if (sedeId === null) {
      this.estaciones.set([]);
      this.status.set('ready');
      return;
    }
    this.status.set('loading');
    try {
      const lista = await this.listar(sedeId);
      if (id !== this.secuencia) return;
      this.estaciones.set(lista);
      this.status.set('ready');
    } catch (error) {
      if (id !== this.secuencia) return;
      this.error.set(describeProblem(error));
      this.status.set('error');
    }
  }

  async crear(v: { codigo: string; nombre: string }, key: string): Resultado {
    const sede = this.sede();
    if (!sede) return 'Elige una sede para crear la estación.';
    const codigo = v.codigo.trim().toUpperCase();
    const nombre = v.nombre.trim();
    const invalido = this.validar(codigo, nombre);
    if (invalido) return invalido;
    try {
      await firstValueFrom(this.api.crear({ sucursal_id: sede, codigo, nombre }, key));
    } catch (error) {
      return rechazo(error, CAMPO_ESTACION);
    }
    await this.avisarTrasRefresco(`Se creó la estación ${nombre}.`);
    return undefined;
  }

  /** Solo el nombre es editable; sin cambios no llama al servidor. */
  async editar(actual: Estacion, v: { nombre: string }): Resultado {
    const nombre = v.nombre.trim();
    const invalido = this.validar(null, nombre);
    if (invalido) return invalido;
    if (nombre === actual.nombre) return undefined;
    try {
      await firstValueFrom(this.api.editar(actual.id, { nombre }));
    } catch (error) {
      return rechazo(error, CAMPO_ESTACION);
    }
    await this.avisarTrasRefresco(`Guardamos los cambios de la estación ${nombre}.`);
    return undefined;
  }

  /** Baja sin motivo. El 409 `estacion_en_uso` vuelve como texto con su `detail`. */
  async baja(e: Estacion, key: string): Resultado {
    try {
      await firstValueFrom(this.api.baja(e.id, key));
    } catch (error) {
      return rechazo(error, {});
    }
    await this.avisarTrasRefresco(`La estación ${e.nombre} quedó de baja.`);
    return undefined;
  }

  async restaurar(e: Estacion, key: string): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(e.id);
    try {
      await firstValueFrom(this.api.restaurar(e.id, key));
      await this.avisarTrasRefresco(`La estación ${e.nombre} volvió a estar activa.`);
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  private validar(codigo: string | null, nombre: string): FieldProblem | undefined {
    if (codigo !== null && !CODIGO.test(codigo)) {
      return {
        field: 'codigo',
        message: 'Usa letras, números, guion o guion bajo (hasta 20 caracteres).',
      };
    }
    if (nombre === '' || nombre.length > MAX_NOMBRE) {
      return { field: 'nombre', message: `Escribe un nombre de hasta ${MAX_NOMBRE} caracteres.` };
    }
    return undefined;
  }

  private listar(sedeId: string): Promise<Estacion[]> {
    return collect((c) => firstValueFrom(this.api.listar({ sucursal_id: sedeId }, c)));
  }

  private say(kind: EstacionesNotice['kind'], text: string): void {
    this.notice.set({ kind, text });
  }

  /** La acción ya se hizo: se vuelve a pedir la lista y se avisa (con advertencia si no se pudo). */
  private async avisarTrasRefresco(ok: string): Promise<void> {
    const sede = this.sede();
    try {
      if (sede) this.estaciones.set(await this.listar(sede));
      this.say('ok', ok);
    } catch {
      this.say(
        'error',
        `${ok} Pero no pudimos actualizar la lista de estaciones: recarga la página para verla al día.`,
      );
    }
  }
}
