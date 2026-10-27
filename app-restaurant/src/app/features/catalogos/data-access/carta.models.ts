import type { Pagina } from '../../../core/http/collect';

/**
 * Contratos del API de la carta (categorías, productos, estaciones, impuestos y tasas).
 * Igual que el directorio, se conserva el snake_case del servidor. Los ids y los decimales
 * (precios, factor de consumo, tasas) viajan siempre como cadenas; las fechas, como AAAA-MM-DD.
 */
export type { Pagina };

export interface Categoria {
  id: string;
  nombre: string;
  descripcion: string | null;
  orden_visualizacion: number;
  es_activo: boolean;
}

export interface CategoriaAlta {
  nombre: string;
  descripcion: string | null;
  orden_visualizacion: number;
}

/** PATCH parcial: solo los campos que cambian (descripcion admite null para vaciarla). */
export type CategoriaCambios = Partial<CategoriaAlta>;

export type TipoProducto = 'PREPARADO' | 'REVENTA' | 'ADICIONAL' | 'SERVICIO';
export type AfectacionIgv = '10' | '20' | '30';

/** Estado del producto en la sede consultada (solo si se pidió `sucursal_id`). */
export interface ProductoEnSucursal {
  disponible: boolean;
  precio_sucursal: string | null;
  precio_efectivo: string;
  estacion_id: string | null;
}

export interface Producto {
  id: string;
  categoria_id: string;
  codigo: string;
  nombre: string;
  descripcion: string | null;
  tipo: TipoProducto;
  articulo_id: string | null;
  factor_consumo: string;
  precio_venta: string;
  afectacion_igv: AfectacionIgv;
  impuesto_id: string | null;
  imagen_url: string | null;
  es_activo: boolean;
  motivo_eliminacion: string | null;
  /** Ausente: no se pidió sede. null: el producto no tiene fila en esa sede («No ofrecido»). */
  en_sucursal?: ProductoEnSucursal | null;
}

export interface ProductoAlta {
  categoria_id: string;
  codigo: string;
  nombre: string;
  descripcion: string | null;
  tipo: TipoProducto;
  articulo_id?: string | null;
  factor_consumo?: string | null;
  precio_venta: string;
  afectacion_igv: AfectacionIgv;
  impuesto_id: string | null;
  imagen_url: string | null;
}

/** PATCH parcial: descripcion, articulo_id, impuesto_id e imagen_url admiten null para vaciarse. */
export type ProductoCambios = Partial<ProductoAlta>;

/** Fila de disponibilidad de un producto en una sede (`productos_sucursal`). */
export interface ProductoDisponibilidad {
  id: string;
  producto_id: string;
  sucursal_id: string;
  estacion_id: string | null;
  precio_venta: string | null;
  disponible: boolean;
  es_activo: boolean;
}

/** Cuerpo del PUT de disponibilidad: precio_venta/estacion_id ausentes = sin cambio; null = vaciar. */
export interface DisponibilidadCuerpo {
  sucursal_id: string;
  disponible: boolean;
  precio_venta?: string | null;
  estacion_id?: string | null;
}

export interface Estacion {
  id: string;
  sucursal_id: string;
  codigo: string;
  nombre: string;
  es_activo: boolean;
}

export interface EstacionAlta {
  sucursal_id: string;
  codigo: string;
  nombre: string;
}

/** Una estación no cambia de sede. */
export type EstacionCambios = Partial<Pick<EstacionAlta, 'codigo' | 'nombre'>>;

export interface Impuesto {
  id: string;
  codigo: string;
  nombre: string;
  descripcion: string | null;
  es_activo: boolean;
}

/** Tasa con 4 decimales ("18.0000"); la vigencia es [desde, hasta) con fechas AAAA-MM-DD. */
export interface TasaImpuesto {
  id: string;
  impuesto_id: string;
  tasa: string;
  vigente_desde: string;
  vigente_hasta: string | null;
  fundamento: string | null;
}

export interface TasaAlta {
  impuesto_id: string;
  tasa: string;
  vigente_desde: string;
  vigente_hasta: string | null;
  fundamento: string | null;
}

/** Una tasa no cambia de impuesto; vigente_hasta null deja la vigencia abierta. */
export type TasaCambios = Partial<Omit<TasaAlta, 'impuesto_id'>>;

export type EstadoFiltro = 'activos' | 'bajas' | 'todos';

/** Filtros de la lista de productos tal como se envían al servidor. */
export interface ProductoFiltros {
  q?: string;
  categoria_id?: string;
  es_activo: 'true' | 'false' | 'todos';
  sucursal_id?: string;
  /** Solo se envía junto con `sucursal_id`. */
  disponible?: boolean;
  limit?: number;
}

export interface EstacionFiltros {
  sucursal_id?: string;
  es_activo?: boolean;
  q?: string;
}

export const TIPOS_PRODUCTO: readonly {
  value: TipoProducto;
  label: string;
  /** REVENTA necesita un artículo de inventario (I5): no se ofrece en el alta. */
  ofrecido: boolean;
  usaFactor: boolean;
}[] = [
  { value: 'PREPARADO', label: 'Preparado', ofrecido: true, usaFactor: true },
  { value: 'REVENTA', label: 'Reventa', ofrecido: false, usaFactor: true },
  { value: 'ADICIONAL', label: 'Adicional', ofrecido: true, usaFactor: true },
  { value: 'SERVICIO', label: 'Servicio', ofrecido: true, usaFactor: false },
];

export const AFECTACIONES_IGV: readonly { value: AfectacionIgv; label: string }[] = [
  { value: '10', label: 'Gravado (10)' },
  { value: '20', label: 'Exonerado (20)' },
  { value: '30', label: 'Inafecto (30)' },
];
