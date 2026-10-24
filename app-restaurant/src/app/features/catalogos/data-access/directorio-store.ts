import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem } from '../../../core/http/problem';
import { cambiosDe, opcional, texto } from '../../../shared/forms/valores';
import { rechazo } from '../../../shared/forms/rechazo';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import { DirectorioApi } from './directorio-api';
import {
  Cliente,
  EdicionCliente,
  EdicionProveedor,
  NuevoCliente,
  NuevoProveedor,
  Proveedor,
} from './directorio.models';

export const PERMISOS = {
  clienteVer: 'CLIENTE_VER',
  clienteAdministrar: 'CLIENTE_ADMINISTRAR',
  proveedorVer: 'PROVEEDOR_VER',
  proveedorAdministrar: 'PROVEEDOR_ADMINISTRAR',
} as const;

export interface Notice {
  kind: 'ok' | 'error';
  text: string;
}

/** Valores de los formularios (el diálogo entrega siempre texto). */
export interface ValoresCliente {
  nombre: string;
  tipo_documento: string;
  numero_documento: string;
  telefono: string;
  email: string;
  direccion: string;
}

export interface ValoresProveedor {
  razon_social: string;
  tipo_documento: string;
  numero_documento: string;
  contacto_nombre: string;
  telefono: string;
  email: string;
  direccion: string;
}

/** Campo del formulario al que pertenece cada rechazo del servidor. */
const CAMPO_POR_CODIGO: Record<string, string> = {
  nombre_invalido: 'nombre',
  razon_social_invalido: 'razon_social',
  documento_invalido: 'numero_documento',
  documento_incompleto: 'numero_documento',
  documento_duplicado: 'numero_documento',
  telefono_invalido: 'telefono',
  email_invalido: 'email',
  direccion_invalido: 'direccion',
  contacto_nombre_invalido: 'contacto_nombre',
};

type Resultado = Promise<string | FieldProblem | undefined>;

/**
 * Estado de «Clientes y proveedores» con datos reales. Se provee en la página (no en root).
 * Todo lo que muestra viene del servidor; el navegador solo oculta lo que el permiso no permite
 * (el servidor vuelve a validarlo). Las acciones de diálogo devuelven el rechazo (o nada si salió
 * bien) para que el diálogo lo muestre sin cerrarse.
 */
@Injectable()
export class DirectorioStore {
  private readonly api = inject(DirectorioApi);
  private readonly auth = inject(AuthSession);

  readonly status = signal<'loading' | 'ready' | 'error'>('loading');
  readonly loadError = signal('');
  readonly clientes = signal<Cliente[]>([]);
  readonly proveedores = signal<Proveedor[]>([]);
  readonly notice = signal<Notice | null>(null);
  readonly busyId = signal<string | null>(null);

  readonly puedeVerClientes = computed(
    () => this.tiene(PERMISOS.clienteVer) || this.tiene(PERMISOS.clienteAdministrar),
  );
  readonly puedeAdministrarClientes = computed(() => this.tiene(PERMISOS.clienteAdministrar));
  readonly puedeVerProveedores = computed(
    () => this.tiene(PERMISOS.proveedorVer) || this.tiene(PERMISOS.proveedorAdministrar),
  );
  readonly puedeAdministrarProveedores = computed(() => this.tiene(PERMISOS.proveedorAdministrar));
  readonly sinAcceso = computed(() => !this.puedeVerClientes() && !this.puedeVerProveedores());

  /** Carga las listas que el permiso de la persona deja ver. */
  async load(): Promise<void> {
    if (this.sinAcceso()) {
      this.status.set('ready');
      return;
    }
    this.status.set('loading');
    this.loadError.set('');
    try {
      const [clientes, proveedores] = await Promise.all([
        this.puedeVerClientes() ? collect((c) => firstValueFrom(this.api.clientes(c))) : [],
        this.puedeVerProveedores() ? collect((c) => firstValueFrom(this.api.proveedores(c))) : [],
      ]);
      this.clientes.set(clientes);
      this.proveedores.set(proveedores);
      this.status.set('ready');
    } catch (error) {
      this.loadError.set(describeProblem(error));
      this.status.set('error');
    }
  }

  /** Alta (`actual` indefinido) o edición parcial de un cliente. */
  async guardarCliente(actual: Cliente | undefined, v: ValoresCliente, key: string): Resultado {
    const nuevo = this.cuerpoCliente(v);
    try {
      if (!actual) {
        await firstValueFrom(this.api.crearCliente(nuevo, key));
        this.say('ok', `Se registró a ${nuevo.nombre_razon_social}.`);
      } else {
        const cambios = this.cambiosCliente(actual, nuevo);
        if (Object.keys(cambios).length === 0) return undefined;
        await firstValueFrom(this.api.editarCliente(actual.id, cambios));
        this.say('ok', `Se actualizaron los datos de ${nuevo.nombre_razon_social}.`);
      }
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.refrescarClientes();
    return undefined;
  }

  async bajaCliente(c: Cliente, motivo: string, key: string): Resultado {
    try {
      await firstValueFrom(this.api.bajaCliente(c.id, motivo.trim(), key));
    } catch (error) {
      return rechazo(error, {});
    }
    await this.refrescarClientes();
    this.say('ok', `${c.nombre_razon_social} quedó de baja. Puedes restaurarlo cuando quieras.`);
    return undefined;
  }

  /** Restaura desde la lista (sin diálogo): informa el resultado con un aviso. */
  async restaurarCliente(c: Cliente, key: string): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(c.id);
    try {
      await firstValueFrom(this.api.restaurarCliente(c.id, key));
      await this.refrescarClientes();
      this.say('ok', `${c.nombre_razon_social} volvió a estar activo.`);
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  /** Alta (`actual` indefinido) o edición parcial de un proveedor. */
  async guardarProveedor(
    actual: Proveedor | undefined,
    v: ValoresProveedor,
    key: string,
  ): Resultado {
    const nuevo = this.cuerpoProveedor(v);
    try {
      if (!actual) {
        await firstValueFrom(this.api.crearProveedor(nuevo, key));
        this.say('ok', `Se registró a ${nuevo.razon_social}.`);
      } else {
        const cambios = cambiosDe(
          this.comoRegistro(actual),
          this.comoRegistro(nuevo),
        ) as EdicionProveedor;
        if (Object.keys(cambios).length === 0) return undefined;
        await firstValueFrom(this.api.editarProveedor(actual.id, cambios));
        this.say('ok', `Se actualizaron los datos de ${nuevo.razon_social}.`);
      }
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.refrescarProveedores();
    return undefined;
  }

  async bajaProveedor(p: Proveedor, motivo: string, key: string): Resultado {
    try {
      await firstValueFrom(this.api.bajaProveedor(p.id, motivo.trim(), key));
    } catch (error) {
      return rechazo(error, {});
    }
    await this.refrescarProveedores();
    this.say('ok', `${p.razon_social} quedó de baja. Puedes restaurarlo cuando quieras.`);
    return undefined;
  }

  async restaurarProveedor(p: Proveedor, key: string): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(p.id);
    try {
      await firstValueFrom(this.api.restaurarProveedor(p.id, key));
      await this.refrescarProveedores();
      this.say('ok', `${p.razon_social} volvió a estar activo.`);
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  private tiene(permiso: string): boolean {
    return this.auth.hasPermission(permiso);
  }

  private cuerpoCliente(v: ValoresCliente): NuevoCliente {
    const numero = opcional(v.numero_documento);
    return {
      nombre_razon_social: texto(v.nombre),
      // Tipo y número van juntos: sin número, no se envía tipo.
      tipo_documento: numero === null ? null : opcional(v.tipo_documento),
      numero_documento: numero,
      telefono: opcional(v.telefono),
      email: opcional(v.email),
      direccion: opcional(v.direccion),
    };
  }

  /** El documento se envía completo (tipo y número) si cambia cualquiera de los dos. */
  private cambiosCliente(actual: Cliente, nuevo: NuevoCliente): EdicionCliente {
    const cambios = cambiosDe(
      this.comoRegistro(actual),
      this.comoRegistro(nuevo),
    ) as EdicionCliente;
    if ('tipo_documento' in cambios || 'numero_documento' in cambios) {
      cambios.tipo_documento = nuevo.tipo_documento;
      cambios.numero_documento = nuevo.numero_documento;
    }
    return cambios;
  }

  private cuerpoProveedor(v: ValoresProveedor): NuevoProveedor {
    return {
      razon_social: texto(v.razon_social),
      tipo_documento: texto(v.tipo_documento),
      numero_documento: texto(v.numero_documento),
      contacto_nombre: opcional(v.contacto_nombre),
      telefono: opcional(v.telefono),
      email: opcional(v.email),
      direccion: opcional(v.direccion),
    };
  }

  /** Solo las claves del contrato de alta: el registro del servidor trae además id, baja y auditoría. */
  private comoRegistro(origen: object): Record<string, string | null> {
    const claves = [
      'nombre_razon_social',
      'razon_social',
      'tipo_documento',
      'numero_documento',
      'contacto_nombre',
      'telefono',
      'email',
      'direccion',
    ];
    const fuente = origen as Record<string, string | null>;
    return Object.fromEntries(claves.filter((k) => k in fuente).map((k) => [k, fuente[k]]));
  }

  private async refrescarClientes(): Promise<void> {
    try {
      this.clientes.set(await collect((c) => firstValueFrom(this.api.clientes(c))));
    } catch (error) {
      this.say('error', `No se pudo actualizar la lista: ${describeProblem(error)}`);
    }
  }

  private async refrescarProveedores(): Promise<void> {
    try {
      this.proveedores.set(await collect((c) => firstValueFrom(this.api.proveedores(c))));
    } catch (error) {
      this.say('error', `No se pudo actualizar la lista: ${describeProblem(error)}`);
    }
  }

  private say(kind: Notice['kind'], text: string): void {
    this.notice.set({ kind, text });
  }
}
