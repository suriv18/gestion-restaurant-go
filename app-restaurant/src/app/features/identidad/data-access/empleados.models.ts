/** Contratos del API de empleados. Los ids viajan como cadenas y las fechas como AAAA-MM-DD. */

export interface Empleado {
  id: string;
  usuario_id: string | null;
  /** Usuario vinculado (acceso al sistema), si lo hay. */
  username: string | null;
  tipo_documento: string;
  numero_documento: string;
  nombres: string;
  apellidos: string;
  telefono: string | null;
  cargo: string | null;
  fecha_ingreso: string;
  fecha_cese: string | null;
  es_activo: boolean;
  motivo_eliminacion: string | null;
}

export interface NuevoEmpleado {
  tipo_documento: string;
  numero_documento: string;
  nombres: string;
  apellidos: string;
  telefono: string | null;
  cargo: string | null;
  /** Sin fecha, el servidor toma la de hoy. */
  fecha_ingreso: string | null;
  /** Sin usuario, la persona no tiene acceso al sistema. */
  usuario_id: string | null;
}

/** PATCH parcial: solo los campos que cambian. */
export interface EdicionEmpleado {
  tipo_documento?: string;
  numero_documento?: string;
  nombres?: string;
  apellidos?: string;
  telefono?: string | null;
  cargo?: string | null;
  fecha_ingreso?: string;
  fecha_cese?: string | null;
  usuario_id?: string | null;
}
