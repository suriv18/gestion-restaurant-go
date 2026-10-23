import type { Pagina } from '../../../core/http/collect';

/** Contratos del API de administración de accesos. Los ids viajan como cadenas. */

export interface RolRef {
  id: string;
  codigo: string;
  nombre: string;
  es_activo: boolean;
}

export interface SucursalRef {
  id: string;
  codigo: string;
  nombre: string;
}

/** Una persona del equipo con sus roles y sedes vigentes (`GET /usuarios`). */
export interface UsuarioEquipo {
  id: string;
  username: string;
  email: string | null;
  es_activo: boolean;
  ultimo_acceso_en: string | null;
  bloqueado_hasta: string | null;
  motivo_baja: string | null;
  roles: RolRef[];
  sucursales: SucursalRef[];
}

export interface Rol {
  id: string;
  codigo: string;
  nombre: string;
  descripcion: string | null;
  es_activo: boolean;
}

export type { Pagina };

export interface UsuarioCreado {
  id: string;
  username: string;
}

export interface NuevoUsuario {
  username: string;
  email?: string;
  password: string;
}
