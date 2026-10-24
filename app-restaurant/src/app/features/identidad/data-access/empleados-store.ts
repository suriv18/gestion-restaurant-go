import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem } from '../../../core/http/problem';
import { rechazo } from '../../../shared/forms/rechazo';
import { cambiosDe, opcional, texto } from '../../../shared/forms/valores';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import { AccesosApi } from './accesos-api';
import { UsuarioEquipo } from './accesos.models';
import { EmpleadosApi } from './empleados-api';
import { EdicionEmpleado, Empleado, NuevoEmpleado } from './empleados.models';

export const PERMISO_VER = 'EMPLEADO_VER';
export const PERMISO_ADMINISTRAR = 'EMPLEADO_ADMINISTRAR';
const PERMISO_USUARIOS = 'USUARIO_ADMINISTRAR';

export interface Notice {
  kind: 'ok' | 'error';
  text: string;
}

/** Valores del formulario (el diálogo entrega siempre texto). */
export interface ValoresEmpleado {
  nombres: string;
  apellidos: string;
  tipo_documento: string;
  numero_documento: string;
  cargo: string;
  telefono: string;
  fecha_ingreso: string;
  fecha_cese?: string;
  usuario_id?: string;
}

/** Campo del formulario al que pertenece cada rechazo del servidor. */
const CAMPO_POR_CODIGO: Record<string, string> = {
  nombres_invalido: 'nombres',
  apellidos_invalido: 'apellidos',
  documento_invalido: 'numero_documento',
  documento_duplicado: 'numero_documento',
  telefono_invalido: 'telefono',
  cargo_invalido: 'cargo',
  fecha_ingreso_invalida: 'fecha_ingreso',
  fecha_cese_invalida: 'fecha_cese',
  fechas_incoherentes: 'fecha_cese',
  usuario_invalido: 'usuario_id',
  usuario_ya_vinculado: 'usuario_id',
};

type Resultado = Promise<string | FieldProblem | undefined>;

/**
 * Estado de «Empleados» con datos reales. Se provee en la página (no en root). El servidor decide
 * qué se puede hacer; aquí solo se oculta lo que el permiso no permite.
 */
@Injectable()
export class EmpleadosStore {
  private readonly api = inject(EmpleadosApi);
  private readonly accesos = inject(AccesosApi);
  private readonly auth = inject(AuthSession);

  readonly status = signal<'loading' | 'ready' | 'error'>('loading');
  readonly loadError = signal('');
  readonly empleados = signal<Empleado[]>([]);
  readonly usuarios = signal<UsuarioEquipo[]>([]);
  readonly notice = signal<Notice | null>(null);
  readonly busyId = signal<string | null>(null);

  readonly puedeVer = computed(
    () => this.auth.hasPermission(PERMISO_VER) || this.auth.hasPermission(PERMISO_ADMINISTRAR),
  );
  readonly puedeAdministrar = computed(() => this.auth.hasPermission(PERMISO_ADMINISTRAR));
  /** Vincular a un usuario existente exige poder consultar el equipo. */
  readonly puedeVincular = computed(
    () => this.puedeAdministrar() && this.auth.hasPermission(PERMISO_USUARIOS),
  );

  /** Carga los empleados y, si puede vincularlos, el equipo para ofrecer usuarios sin empleado. */
  async load(): Promise<void> {
    if (!this.puedeVer()) {
      this.status.set('ready');
      return;
    }
    this.status.set('loading');
    this.loadError.set('');
    try {
      const [empleados, usuarios] = await Promise.all([
        collect((c) => firstValueFrom(this.api.empleados(c))),
        this.puedeVincular() ? collect((c) => firstValueFrom(this.accesos.usuarios(c))) : [],
      ]);
      this.empleados.set(empleados);
      this.usuarios.set(usuarios);
      this.status.set('ready');
    } catch (error) {
      this.loadError.set(describeProblem(error));
      this.status.set('error');
    }
  }

  /** Usuarios activos sin empleado, más el que ya tiene este empleado (para poder conservarlo). */
  usuariosVinculables(actual?: Empleado): UsuarioEquipo[] {
    return this.usuarios().filter(
      (u) => u.id === actual?.usuario_id || (u.es_activo && u.empleado === null),
    );
  }

  /** Alta (`actual` indefinido) o edición parcial. Reintentable con la misma clave. */
  async guardar(actual: Empleado | undefined, v: ValoresEmpleado, key: string): Resultado {
    try {
      if (!actual) {
        const nuevo = this.cuerpoNuevo(v);
        await firstValueFrom(this.api.crear(nuevo, key));
        this.say('ok', `Se registró a ${nuevo.nombres} ${nuevo.apellidos}.`);
      } else {
        const cambios = this.cambios(actual, v);
        if (Object.keys(cambios).length === 0) return undefined;
        await firstValueFrom(this.api.editar(actual.id, cambios));
        this.say('ok', `Se actualizaron los datos de ${texto(v.nombres)} ${texto(v.apellidos)}.`);
      }
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.refrescar();
    return undefined;
  }

  async baja(e: Empleado, motivo: string, key: string): Resultado {
    try {
      await firstValueFrom(this.api.baja(e.id, motivo.trim(), key));
    } catch (error) {
      return rechazo(error, {});
    }
    await this.refrescar();
    this.say('ok', `${nombre(e)} quedó de baja. Puedes restaurarlo cuando quieras.`);
    return undefined;
  }

  /** Restaura desde la lista (sin diálogo): informa el resultado con un aviso. */
  async restaurar(e: Empleado, key: string): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(e.id);
    try {
      await firstValueFrom(this.api.restaurar(e.id, key));
      await this.refrescar();
      this.say('ok', `${nombre(e)} volvió a estar activo.`);
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  private cuerpoNuevo(v: ValoresEmpleado): NuevoEmpleado {
    return {
      tipo_documento: texto(v.tipo_documento),
      numero_documento: texto(v.numero_documento),
      nombres: texto(v.nombres),
      apellidos: texto(v.apellidos),
      telefono: opcional(v.telefono),
      cargo: opcional(v.cargo),
      fecha_ingreso: opcional(v.fecha_ingreso),
      usuario_id: opcional(v.usuario_id ?? ''),
    };
  }

  /** Solo viajan los campos que cambian. La fecha de ingreso nunca se envía vacía. */
  private cambios(actual: Empleado, v: ValoresEmpleado): EdicionEmpleado {
    const nuevo: Record<string, string | null> = {
      tipo_documento: texto(v.tipo_documento),
      numero_documento: texto(v.numero_documento),
      nombres: texto(v.nombres),
      apellidos: texto(v.apellidos),
      telefono: opcional(v.telefono),
      cargo: opcional(v.cargo),
      fecha_ingreso: opcional(v.fecha_ingreso),
      fecha_cese: opcional(v.fecha_cese ?? ''),
    };
    // Sin el selector de usuarios (falta permiso) no se toca el vínculo.
    if (v.usuario_id !== undefined) nuevo['usuario_id'] = opcional(v.usuario_id);
    const cambios = cambiosDe(
      {
        tipo_documento: actual.tipo_documento,
        numero_documento: actual.numero_documento,
        nombres: actual.nombres,
        apellidos: actual.apellidos,
        telefono: actual.telefono,
        cargo: actual.cargo,
        fecha_ingreso: actual.fecha_ingreso,
        fecha_cese: actual.fecha_cese,
        usuario_id: actual.usuario_id,
      },
      nuevo,
    ) as EdicionEmpleado;
    if (cambios.fecha_ingreso === null) delete cambios.fecha_ingreso;
    return cambios;
  }

  private async refrescar(): Promise<void> {
    try {
      const [empleados, usuarios] = await Promise.all([
        collect((c) => firstValueFrom(this.api.empleados(c))),
        this.puedeVincular() ? collect((c) => firstValueFrom(this.accesos.usuarios(c))) : [],
      ]);
      this.empleados.set(empleados);
      this.usuarios.set(usuarios);
    } catch (error) {
      this.say('error', `No se pudo actualizar la lista: ${describeProblem(error)}`);
    }
  }

  private say(kind: Notice['kind'], text: string): void {
    this.notice.set({ kind, text });
  }
}

export const nombre = (e: Pick<Empleado, 'nombres' | 'apellidos'>): string =>
  `${e.nombres} ${e.apellidos}`;
