import type { Pagina } from '../../../core/http/collect';

/** Contratos del API de clientes y proveedores. Los ids viajan como cadenas. */
export type { Pagina };

/** Datos de baja lógica que comparten los dos maestros. */
interface Baja {
  es_activo: boolean;
  motivo_eliminacion: string | null;
}

export interface Cliente extends Baja {
  id: string;
  tipo_documento: string | null;
  numero_documento: string | null;
  nombre_razon_social: string;
  telefono: string | null;
  email: string | null;
  direccion: string | null;
}

export interface Proveedor extends Baja {
  id: string;
  tipo_documento: string;
  numero_documento: string;
  razon_social: string;
  contacto_nombre: string | null;
  telefono: string | null;
  email: string | null;
  direccion: string | null;
}

/** El documento es opcional, pero tipo y número van siempre juntos. */
export interface NuevoCliente {
  tipo_documento: string | null;
  numero_documento: string | null;
  nombre_razon_social: string;
  telefono: string | null;
  email: string | null;
  direccion: string | null;
}

export interface NuevoProveedor {
  tipo_documento: string;
  numero_documento: string;
  razon_social: string;
  contacto_nombre: string | null;
  telefono: string | null;
  email: string | null;
  direccion: string | null;
}

/** PATCH parcial: solo los campos que cambian. */
export type EdicionCliente = Partial<NuevoCliente>;
export type EdicionProveedor = Partial<NuevoProveedor>;

/** Tipos de documento que ofrece el formulario; el servidor valida el formato de DNI y RUC. */
export const TIPOS_DOCUMENTO = [
  { value: 'DNI', label: 'DNI' },
  { value: 'RUC', label: 'RUC' },
  { value: 'CE', label: 'Carné de extranjería' },
  { value: 'PASAPORTE', label: 'Pasaporte' },
];
