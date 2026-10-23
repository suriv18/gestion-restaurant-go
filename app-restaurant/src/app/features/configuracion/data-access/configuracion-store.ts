import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem, problemCode } from '../../../core/http/problem';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import { cambiosDe } from '../configuracion.forms';
import type {
  Mesa,
  Negocio,
  NegocioCambios,
  SedeRef,
  Sucursal,
  SucursalCambios,
  SucursalNueva,
  Zona,
} from '../configuracion.models';
import { ConfiguracionApi } from './configuracion-api';

export const NEGOCIO_VER = 'NEGOCIO_VER';
export const NEGOCIO_ADMINISTRAR = 'NEGOCIO_ADMINISTRAR';
export const SUCURSAL_VER = 'SUCURSAL_VER';
export const SUCURSAL_ADMINISTRAR = 'SUCURSAL_ADMINISTRAR';
export const MESA_VER = 'MESA_VER';
export const MESA_ADMINISTRAR = 'MESA_ADMINISTRAR';

export interface ConfiguracionNotice {
  kind: 'ok' | 'error';
  text: string;
}

export interface SectionErrors {
  negocio?: string;
  sedes?: string;
  salon?: string;
}

/** Resultado de una acción de diálogo: undefined = éxito; texto = aviso; FieldProblem = campo. */
export type ActionError = string | FieldProblem;

/** Código de rechazo del servidor -> campo del formulario que lo causó. */
const FIELD_BY_CODE: Record<string, string> = {
  razon_social_invalida: 'razon_social',
  nombre_comercial_invalido: 'nombre_comercial',
  direccion_fiscal_invalida: 'direccion_fiscal',
  moneda_invalida: 'moneda',
  zona_horaria_invalida: 'zona_horaria',
  email_invalido: 'email',
  codigo_duplicado: 'codigo',
  codigo_invalido: 'codigo',
  direccion_invalida: 'direccion',
  ubigeo_invalido: 'ubigeo',
  telefono_invalido: 'telefono',
  codigo_establecimiento_sunat_invalido: 'codigo_establecimiento_sunat',
  nombre_duplicado: 'nombre',
  nombre_invalido: 'nombre',
  orden_invalido: 'orden_visualizacion',
  numero_mesa_duplicado: 'numero_mesa',
  numero_mesa_invalido: 'numero_mesa',
  capacidad_invalida: 'capacidad',
  posicion_x_invalida: 'posicion_x',
  posicion_y_invalida: 'posicion_y',
  zona_invalida: 'zona_id',
  motivo_requerido: 'motivo',
};

@Injectable()
export class ConfiguracionStore {
  protected readonly api = inject(ConfiguracionApi);
  private readonly auth = inject(AuthSession);

  readonly status = signal<'loading' | 'ready'>('loading');
  readonly errors = signal<SectionErrors>({});
  readonly notice = signal<ConfiguracionNotice | null>(null);
  readonly busyId = signal<string | null>(null);
  readonly negocio = signal<Negocio | null>(null);
  readonly sedes = signal<Sucursal[]>([]);
  readonly zonas = signal<Zona[]>([]);
  readonly mesas = signal<Mesa[]>([]);
  private readonly sedeElegida = signal<string | null>(null);

  readonly puedeVerNegocio = computed(
    () => this.tiene(NEGOCIO_VER) || this.tiene(NEGOCIO_ADMINISTRAR),
  );
  readonly puedeEditarNegocio = computed(() => this.tiene(NEGOCIO_ADMINISTRAR));
  readonly puedeVerSedes = computed(
    () => this.tiene(SUCURSAL_VER) || this.tiene(SUCURSAL_ADMINISTRAR),
  );
  readonly puedeEditarSedes = computed(() => this.tiene(SUCURSAL_ADMINISTRAR));
  readonly puedeVerSalon = computed(() => this.tiene(MESA_VER) || this.tiene(MESA_ADMINISTRAR));
  readonly puedeEditarSalon = computed(() => this.tiene(MESA_ADMINISTRAR));
  readonly sinAcceso = computed(
    () => !this.puedeVerNegocio() && !this.puedeVerSedes() && !this.puedeVerSalon(),
  );

  /** Sedes elegibles: con permiso de sedes, las activas con su nombre; si no, las que aparecen en los datos. */
  readonly sedesSalon = computed<SedeRef[]>(() => {
    if (this.puedeVerSedes()) {
      return this.sedes()
        .filter((s) => s.es_activo)
        .map((s) => ({ id: s.id, nombre: s.nombre }));
    }
    const ids = new Set([
      ...this.zonas().map((z) => z.sucursal_id),
      ...this.mesas().map((m) => m.sucursal_id),
    ]);
    return [...ids]
      .sort((a, b) => Number(a) - Number(b))
      .map((id) => ({ id, nombre: `Sede ${id}` }));
  });

  readonly sedeActual = computed<string | null>(() => {
    const lista = this.sedesSalon();
    return lista.find((s) => s.id === this.sedeElegida())?.id ?? lista[0]?.id ?? null;
  });

  readonly zonasSede = computed(() =>
    this.zonas()
      .filter((z) => z.sucursal_id === this.sedeActual())
      .sort(
        (a, b) =>
          a.orden_visualizacion - b.orden_visualizacion || a.nombre.localeCompare(b.nombre, 'es'),
      ),
  );
  readonly zonasActivasSede = computed(() => this.zonasSede().filter((z) => z.es_activo));

  elegirSede(id: string): void {
    this.sedeElegida.set(id);
  }

  mesasDeZona(zonaId: string): Mesa[] {
    return this.mesas()
      .filter((m) => m.zona_id === zonaId)
      .sort((a, b) => a.numero_mesa.localeCompare(b.numero_mesa, 'es', { numeric: true }));
  }

  async load(): Promise<void> {
    this.status.set('loading');
    this.errors.set({});
    if (this.sinAcceso()) {
      this.status.set('ready');
      return;
    }
    await Promise.all([this.cargarNegocio(), this.cargarSedes(), this.cargarSalon()]);
    this.status.set('ready');
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  // --- Negocio ---------------------------------------------------------------------------

  async editarNegocio(cambios: NegocioCambios): Promise<ActionError | undefined> {
    if (Object.keys(cambios).length === 0) return undefined;
    try {
      this.negocio.set(await firstValueFrom(this.api.editarNegocio(cambios)));
    } catch (error) {
      return this.comoError(error);
    }
    this.say('ok', 'Guardamos los datos de tu restaurante.');
    return undefined;
  }

  // --- Sedes -----------------------------------------------------------------------------

  async crearSede(body: SucursalNueva, key: string): Promise<ActionError | undefined> {
    return this.mutarSedes(
      () => firstValueFrom(this.api.crearSucursal(body, key)),
      `Se creó la sede ${body.nombre}. Asígnala a tu equipo en Equipo y accesos para que alguien pueda operarla.`,
    );
  }

  async editarSede(
    sede: Sucursal,
    nuevo: Record<string, string | null>,
  ): Promise<ActionError | undefined> {
    const cambios = cambiosDe({ ...sede }, nuevo) as SucursalCambios;
    if (Object.keys(cambios).length === 0) return undefined;
    return this.mutarSedes(
      () => firstValueFrom(this.api.editarSucursal(sede.id, cambios)),
      `Guardamos los cambios de la sede ${sede.nombre}.`,
    );
  }

  async bajaSede(sede: Sucursal, motivo: string, key: string): Promise<ActionError | undefined> {
    return this.mutarSedes(
      () => firstValueFrom(this.api.bajaSucursal(sede.id, motivo, key)),
      `La sede ${sede.nombre} quedó de baja.`,
    );
  }

  async restaurarSede(sede: Sucursal, key: string): Promise<void> {
    await this.conBloqueo(sede.id, async () => {
      await firstValueFrom(this.api.restaurarSucursal(sede.id, key));
      await this.refrescarSedes();
      this.say('ok', `La sede ${sede.nombre} volvió a estar activa.`);
    });
  }

  // --- Internos --------------------------------------------------------------------------

  protected tiene(permiso: string): boolean {
    return this.auth.hasPermission(permiso);
  }

  protected say(kind: 'ok' | 'error', text: string): void {
    this.notice.set({ kind, text });
  }

  protected fallo(seccion: keyof SectionErrors, error: unknown): void {
    this.errors.update((actual) => ({ ...actual, [seccion]: describeProblem(error) }));
  }

  /** Error del servidor -> texto general o error de un campo, según su código. */
  protected comoError(error: unknown): ActionError {
    const message = describeProblem(error);
    const code = problemCode(error);
    const field = code ? FIELD_BY_CODE[code] : undefined;
    return field ? { field, message } : message;
  }

  /** Ejecuta una acción sin diálogo (interruptor, restaurar): una a la vez y con aviso de error. */
  protected async conBloqueo(id: string, accion: () => Promise<void>): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(id);
    try {
      await accion();
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  private async mutarSedes(
    accion: () => Promise<unknown>,
    ok: string,
  ): Promise<ActionError | undefined> {
    try {
      await accion();
    } catch (error) {
      return this.comoError(error);
    }
    await this.refrescarSedes();
    this.say('ok', ok);
    return undefined;
  }

  private async cargarNegocio(): Promise<void> {
    if (!this.puedeVerNegocio()) return;
    try {
      this.negocio.set(await firstValueFrom(this.api.negocio()));
    } catch (error) {
      this.fallo('negocio', error);
    }
  }

  private async cargarSedes(): Promise<void> {
    if (!this.puedeVerSedes()) return;
    try {
      this.sedes.set(await collect((c) => firstValueFrom(this.api.sucursales(c))));
    } catch (error) {
      this.fallo('sedes', error);
    }
  }

  protected async cargarSalon(): Promise<void> {
    if (!this.puedeVerSalon()) return;
    try {
      const [zonas, mesas] = await Promise.all([
        collect((c) => firstValueFrom(this.api.zonas(c))),
        collect((c) => firstValueFrom(this.api.mesas(c))),
      ]);
      this.zonas.set(zonas);
      this.mesas.set(mesas);
    } catch (error) {
      this.fallo('salon', error);
    }
  }

  private async refrescarSedes(): Promise<void> {
    try {
      this.sedes.set(await collect((c) => firstValueFrom(this.api.sucursales(c))));
    } catch (error) {
      this.say('error', `No se pudo actualizar la lista de sedes: ${describeProblem(error)}`);
    }
  }
}
