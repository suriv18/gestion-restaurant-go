import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem } from '../../../core/http/problem';
import { rechazo } from '../../../shared/forms/rechazo';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import type {
  Impuesto,
  TasaAlta,
  TasaCambios,
  TasaImpuesto,
} from '../../catalogos/data-access/carta.models';
import {
  compararFechas,
  fechaValida,
  hoyLima,
  normalizarDecimal,
  tasaValida,
} from '../../catalogos/data-access/carta.util';
import { ImpuestosApi } from '../../catalogos/data-access/impuestos-api';

export const IMPUESTO_VER = 'IMPUESTO_VER';
export const IMPUESTO_ADMINISTRAR = 'IMPUESTO_ADMINISTRAR';
/** Quien administra la carta también puede consultar impuestos y tasas (spec F1). */
export const CARTA_ADMINISTRAR = 'CARTA_ADMINISTRAR';

export interface TasasNotice {
  kind: 'ok' | 'error';
  text: string;
}

/** Valores del formulario de tasa (el diálogo entrega siempre texto). */
export interface ValoresTasa {
  tasa: string;
  vigente_desde: string;
  vigente_hasta: string;
  fundamento: string;
}

/** Código de rechazo del servidor -> campo del formulario. */
export const CAMPO_TASA: Record<string, string> = {
  tasa_invalida: 'tasa',
  vigente_desde_invalido: 'vigente_desde',
  vigente_hasta_invalido: 'vigente_hasta',
  vigencia_invalida: 'vigente_hasta',
  vigencia_solapada: 'vigente_desde',
  fundamento_invalido: 'fundamento',
};

const MAX_FUNDAMENTO = 2000;

type Resultado = Promise<string | FieldProblem | undefined>;

/**
 * Impuestos (solo lectura) y su historial de tasas (sección de Configuración). Los impuestos se
 * cargan primero; las tasas se piden por impuesto, de modo que el fallo de uno no tumba a los demás.
 */
@Injectable()
export class TasasStore {
  private readonly api = inject(ImpuestosApi);
  private readonly auth = inject(AuthSession);

  readonly status = signal<'loading' | 'ready' | 'error'>('loading');
  readonly error = signal('');
  readonly notice = signal<TasasNotice | null>(null);
  readonly impuestos = signal<Impuesto[]>([]);
  /** Tasas por id de impuesto, de la vigencia más reciente a la más antigua. */
  readonly tasas = signal<Record<string, TasaImpuesto[]>>({});
  /** Error al cargar las tasas de un impuesto concreto. */
  readonly errores = signal<Record<string, string>>({});
  /** Fecha de hoy en Lima (AAAA-MM-DD), tomada al cargar. */
  readonly hoy = signal(hoyLima());

  readonly puedeAdministrar = computed(() => this.auth.hasPermission(IMPUESTO_ADMINISTRAR));

  async cargar(): Promise<void> {
    this.status.set('loading');
    this.error.set('');
    this.errores.set({});
    this.hoy.set(hoyLima());
    try {
      this.impuestos.set(await collect((c) => firstValueFrom(this.api.impuestos(c))));
    } catch (error) {
      this.error.set(describeProblem(error));
      this.status.set('error');
      return;
    }
    await Promise.all(this.impuestos().map((i) => this.cargarTasas(i.id)));
    this.status.set('ready');
  }

  /** Vuelve a pedir las tasas de un impuesto (tras un error parcial o una escritura). */
  async reintentar(impuestoId: string): Promise<void> {
    await this.cargarTasas(impuestoId);
  }

  async crear(impuesto: Impuesto, v: ValoresTasa, key: string): Resultado {
    const invalido = this.validar(v);
    if (invalido) return invalido;
    const cuerpo: TasaAlta = {
      impuesto_id: impuesto.id,
      tasa: v.tasa.trim(),
      vigente_desde: v.vigente_desde.trim(),
      vigente_hasta: v.vigente_hasta.trim() || null,
      fundamento: v.fundamento.trim() || null,
    };
    try {
      await firstValueFrom(this.api.crearTasa(cuerpo, key));
    } catch (error) {
      return rechazo(error, CAMPO_TASA);
    }
    await this.avisarTrasRefresco(impuesto.id, `Se registró la tasa de ${impuesto.nombre}.`);
    return undefined;
  }

  /** PATCH solo con lo que cambió; sin cambios no llama al servidor. */
  async editar(actual: TasaImpuesto, v: ValoresTasa): Resultado {
    const invalido = this.validar(v);
    if (invalido) return invalido;
    const tasa = v.tasa.trim();
    const desde = v.vigente_desde.trim();
    const hasta = v.vigente_hasta.trim() || null;
    const fundamento = v.fundamento.trim() || null;
    const cambios: TasaCambios = {};
    if (normalizarDecimal(tasa) !== normalizarDecimal(actual.tasa)) cambios.tasa = tasa;
    if (desde !== actual.vigente_desde) cambios.vigente_desde = desde;
    if (hasta !== actual.vigente_hasta) cambios.vigente_hasta = hasta;
    if (fundamento !== actual.fundamento) cambios.fundamento = fundamento;
    if (Object.keys(cambios).length === 0) return undefined;
    try {
      await firstValueFrom(this.api.editarTasa(actual.id, cambios));
    } catch (error) {
      return rechazo(error, CAMPO_TASA);
    }
    const nombre = this.impuestos().find((i) => i.id === actual.impuesto_id)?.nombre ?? 'impuesto';
    await this.avisarTrasRefresco(
      actual.impuesto_id,
      `Guardamos los cambios de la tasa de ${nombre}.`,
    );
    return undefined;
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  private validar(v: ValoresTasa): FieldProblem | undefined {
    if (!tasaValida(v.tasa)) {
      return {
        field: 'tasa',
        message: 'Escribe un porcentaje entre 0 y 100 con hasta 4 decimales, por ejemplo 18.',
      };
    }
    const desde = v.vigente_desde.trim();
    if (!fechaValida(desde)) {
      return { field: 'vigente_desde', message: 'Elige una fecha de inicio válida.' };
    }
    const hasta = v.vigente_hasta.trim();
    if (hasta !== '') {
      if (!fechaValida(hasta)) {
        return { field: 'vigente_hasta', message: 'Elige una fecha de fin válida o déjala vacía.' };
      }
      if (compararFechas(hasta, desde) <= 0) {
        return {
          field: 'vigente_hasta',
          message: 'La fecha de fin debe ser posterior a la de inicio.',
        };
      }
    }
    if (v.fundamento.trim().length > MAX_FUNDAMENTO) {
      return {
        field: 'fundamento',
        message: `El fundamento admite hasta ${MAX_FUNDAMENTO} caracteres.`,
      };
    }
    return undefined;
  }

  private async cargarTasas(impuestoId: string): Promise<void> {
    try {
      const lista = await collect((c) => firstValueFrom(this.api.tasas(impuestoId, c)));
      this.tasas.update((actual) => ({
        ...actual,
        [impuestoId]: [...lista].sort((a, b) => compararFechas(b.vigente_desde, a.vigente_desde)),
      }));
      this.errores.update((actual) => {
        const resto = { ...actual };
        delete resto[impuestoId];
        return resto;
      });
    } catch (error) {
      this.errores.update((actual) => ({ ...actual, [impuestoId]: describeProblem(error) }));
    }
  }

  private async avisarTrasRefresco(impuestoId: string, ok: string): Promise<void> {
    this.hoy.set(hoyLima());
    await this.cargarTasas(impuestoId);
    if (this.errores()[impuestoId]) {
      this.notice.set({
        kind: 'error',
        text: `${ok} Pero no pudimos actualizar el historial: recarga la página para verlo al día.`,
      });
    } else {
      this.notice.set({ kind: 'ok', text: ok });
    }
  }
}
