import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem, problemCode } from '../../../core/http/problem';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import { cambiosDe } from '../configuracion.forms';
import type {
  Mesa,
  MesaCambios,
  MesaNueva,
  Negocio,
  NegocioCambios,
  SedeRef,
  Sucursal,
  SucursalCambios,
  SucursalNueva,
  Zona,
  ZonaCambios,
  ZonaNueva,
} from '../configuracion.models';
import { ConfiguracionApi } from './configuracion-api';
import { ESTACION_ADMINISTRAR, ESTACION_VER } from './estaciones-store';
import { CARTA_ADMINISTRAR, IMPUESTO_ADMINISTRAR, IMPUESTO_VER } from './tasas-store';

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
  // Estaciones y tasas tienen su propio store; aquí solo se decide si la sección aparece.
  readonly puedeVerEstaciones = computed(
    () => this.tiene(ESTACION_VER) || this.tiene(ESTACION_ADMINISTRAR),
  );
  readonly puedeVerTasas = computed(
    () =>
      this.tiene(IMPUESTO_VER) || this.tiene(IMPUESTO_ADMINISTRAR) || this.tiene(CARTA_ADMINISTRAR),
  );
  readonly sinAcceso = computed(
    () =>
      !this.puedeVerNegocio() &&
      !this.puedeVerSedes() &&
      !this.puedeVerSalon() &&
      !this.puedeVerEstaciones() &&
      !this.puedeVerTasas(),
  );

  /** Sedes elegibles: con permiso de sedes, las activas con su nombre; si no, las que aparecen en los datos. */
  readonly sedesSalon = computed<SedeRef[]>(() => {
    // Si la lista de sedes falló, el salón sigue operando con las sedes que salen de zonas y mesas.
    if (this.puedeVerSedes() && !this.errors().sedes) {
      return this.sedes()
        .filter((s) => s.es_activo)
        .map((s) => ({ id: s.id, nombre: s.nombre }));
    }
    const ids = new Set([
      ...this.zonas().map((z) => z.sucursal_id),
      ...this.mesas().map((m) => m.sucursal_id),
    ]);
    // Sin zonas ni mesas (sede nueva, o solo permisos de estaciones) valen las sedes de la sesión.
    const conocidas = ids.size > 0 ? [...ids] : (this.auth.user()?.branches ?? []);
    return [...conocidas]
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
      this.avisarTrasRefresco(
        await this.refrescarSedes(),
        `La sede ${sede.nombre} volvió a estar activa.`,
      );
    });
  }

  // --- Zonas -----------------------------------------------------------------------------

  async crearZona(body: ZonaNueva, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.crearZona(body, key)),
      `Se creó la zona ${body.nombre}.`,
    );
  }

  async editarZona(
    zona: Zona,
    nuevo: Record<string, string | number>,
  ): Promise<ActionError | undefined> {
    const cambios = cambiosDe({ ...zona }, nuevo) as ZonaCambios;
    if (Object.keys(cambios).length === 0) return undefined;
    return this.mutarSalon(
      () => firstValueFrom(this.api.editarZona(zona.id, cambios)),
      `Guardamos los cambios de la zona ${zona.nombre}.`,
    );
  }

  async bajaZona(zona: Zona, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.bajaZona(zona.id, key)),
      `La zona ${zona.nombre} quedó de baja.`,
    );
  }

  async restaurarZona(zona: Zona, key: string): Promise<void> {
    await this.conBloqueo(zona.id, async () => {
      await firstValueFrom(this.api.restaurarZona(zona.id, key));
      this.avisarTrasRefresco(
        await this.refrescarSalon(),
        `La zona ${zona.nombre} volvió a estar activa.`,
        'zonas y mesas',
      );
    });
  }

  // --- Mesas -----------------------------------------------------------------------------

  async crearMesa(body: MesaNueva, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.crearMesa(body, key)),
      `Se creó la mesa ${body.numero_mesa}.`,
    );
  }

  async editarMesa(
    mesa: Mesa,
    nuevo: Record<string, string | number | null>,
  ): Promise<ActionError | undefined> {
    const cambios = cambiosDe({ ...mesa }, nuevo) as MesaCambios;
    if (Object.keys(cambios).length === 0) return undefined;
    return this.mutarSalon(
      () => firstValueFrom(this.api.editarMesa(mesa.id, cambios)),
      `Guardamos los cambios de la mesa ${mesa.numero_mesa}.`,
    );
  }

  /** Interruptor «habilitada»: sin diálogo ni motivo; actualiza solo esa fila. */
  async habilitarMesa(mesa: Mesa): Promise<void> {
    await this.conBloqueo(mesa.id, async () => {
      const actualizada = await firstValueFrom(
        this.api.editarMesa(mesa.id, { habilitada: !mesa.habilitada }),
      );
      this.mesas.update((lista) => lista.map((m) => (m.id === actualizada.id ? actualizada : m)));
    });
  }

  async bajaMesa(mesa: Mesa, motivo: string, key: string): Promise<ActionError | undefined> {
    return this.mutarSalon(
      () => firstValueFrom(this.api.bajaMesa(mesa.id, motivo, key)),
      `La mesa ${mesa.numero_mesa} quedó de baja.`,
    );
  }

  async restaurarMesa(mesa: Mesa, key: string): Promise<void> {
    await this.conBloqueo(mesa.id, async () => {
      await firstValueFrom(this.api.restaurarMesa(mesa.id, key));
      this.avisarTrasRefresco(
        await this.refrescarSalon(),
        `La mesa ${mesa.numero_mesa} volvió a estar activa.`,
        'zonas y mesas',
      );
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
    this.avisarTrasRefresco(await this.refrescarSedes(), ok);
    return undefined;
  }

  private async mutarSalon(
    accion: () => Promise<unknown>,
    ok: string,
  ): Promise<ActionError | undefined> {
    try {
      await accion();
    } catch (error) {
      return this.comoError(error);
    }
    this.avisarTrasRefresco(await this.refrescarSalon(), ok, 'zonas y mesas');
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

  /** Vuelve a pedir las sedes; devuelve si lo logró (si no, la lista queda desactualizada). */
  private async refrescarSedes(): Promise<boolean> {
    try {
      this.sedes.set(await collect((c) => firstValueFrom(this.api.sucursales(c))));
      return true;
    } catch {
      return false;
    }
  }

  /** Vuelve a pedir zonas y mesas; devuelve si lo logró (si no, las listas quedan desactualizadas). */
  private async refrescarSalon(): Promise<boolean> {
    try {
      const [zonas, mesas] = await Promise.all([
        collect((c) => firstValueFrom(this.api.zonas(c))),
        collect((c) => firstValueFrom(this.api.mesas(c))),
      ]);
      this.zonas.set(zonas);
      this.mesas.set(mesas);
      return true;
    } catch {
      return false;
    }
  }

  /** Aviso tras una acción ya hecha: éxito, o éxito con advertencia si la lista no se pudo actualizar. */
  private avisarTrasRefresco(refrescada: boolean, ok: string, lista = 'sedes'): void {
    if (refrescada) {
      this.say('ok', ok);
    } else {
      this.say(
        'error',
        `${ok} Pero no pudimos actualizar la lista de ${lista}: recarga la página para verla al día.`,
      );
    }
  }
}
