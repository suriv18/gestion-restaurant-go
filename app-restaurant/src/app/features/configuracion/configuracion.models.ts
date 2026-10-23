/** Contratos de la API de Configuración. Todos los IDs viajan como cadenas. */
export interface Auditoria {
  creado_en: string;
  creado_por: string;
  actualizado_en: string;
  actualizado_por: string;
}

export interface Negocio extends Auditoria {
  id: string;
  singleton: boolean;
  ruc: string;
  razon_social: string;
  nombre_comercial: string;
  direccion_fiscal: string;
  ubigeo: string | null;
  telefono: string | null;
  email: string | null;
  moneda: string;
  zona_horaria: string;
  es_activo: boolean;
}

export interface NegocioCambios {
  razon_social?: string;
  nombre_comercial?: string;
  direccion_fiscal?: string;
  ubigeo?: string | null;
  telefono?: string | null;
  email?: string | null;
  moneda?: string;
  zona_horaria?: string;
}

export interface Sucursal extends Auditoria {
  id: string;
  negocio_id: string;
  codigo: string;
  nombre: string;
  direccion: string;
  ubigeo: string | null;
  codigo_establecimiento_sunat: string | null;
  telefono: string | null;
  es_activo: boolean;
  motivo_eliminacion: string | null;
  eliminado_en: string | null;
  eliminado_por: string | null;
}

export interface SucursalNueva {
  codigo: string;
  nombre: string;
  direccion: string;
  ubigeo?: string;
  codigo_establecimiento_sunat?: string;
  telefono?: string;
}

export interface SucursalCambios {
  codigo?: string;
  nombre?: string;
  direccion?: string;
  ubigeo?: string | null;
  codigo_establecimiento_sunat?: string | null;
  telefono?: string | null;
}

export interface Zona extends Auditoria {
  id: string;
  sucursal_id: string;
  nombre: string;
  orden_visualizacion: number;
  es_activo: boolean;
}

export interface ZonaNueva {
  sucursal_id: string;
  nombre: string;
  orden_visualizacion?: number;
}

export interface ZonaCambios {
  nombre?: string;
  orden_visualizacion?: number;
}

export interface Mesa extends Auditoria {
  id: string;
  sucursal_id: string;
  zona_id: string;
  numero_mesa: string;
  capacidad: number;
  habilitada: boolean;
  posicion_x: string | null;
  posicion_y: string | null;
  es_activo: boolean;
  motivo_eliminacion: string | null;
  eliminado_en: string | null;
  eliminado_por: string | null;
}

export interface MesaNueva {
  sucursal_id: string;
  zona_id: string;
  numero_mesa: string;
  capacidad: number;
  habilitada?: boolean;
  posicion_x?: string;
  posicion_y?: string;
}

export interface MesaCambios {
  zona_id?: string;
  numero_mesa?: string;
  capacidad?: number;
  habilitada?: boolean;
  posicion_x?: string | null;
  posicion_y?: string | null;
}

/** Sede elegible en «Zonas y mesas». */
export interface SedeRef {
  id: string;
  nombre: string;
}
