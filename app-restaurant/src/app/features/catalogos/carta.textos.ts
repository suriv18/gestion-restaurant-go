import { AfectacionIgv, TIPOS_PRODUCTO } from './data-access/carta.models';

/** Textos legibles de la carta: la pantalla nunca muestra códigos crudos del servidor. */
export const etiquetaTipo = (tipo: string): string =>
  TIPOS_PRODUCTO.find((t) => t.value === tipo)?.label ?? 'Otro';

const AFECTACION: Record<AfectacionIgv, string> = {
  '10': 'Gravado',
  '20': 'Exonerado',
  '30': 'Inafecto',
};

export const etiquetaAfectacion = (afectacion: string): string =>
  AFECTACION[afectacion as AfectacionIgv] ?? 'Sin definir';

/** Opciones del formulario: se explica cada una en lenguaje del negocio. */
export const OPCIONES_AFECTACION: readonly { value: AfectacionIgv; label: string }[] = [
  { value: '10', label: 'Gravado (paga IGV)' },
  { value: '20', label: 'Exonerado (sin IGV por ley)' },
  { value: '30', label: 'Inafecto (fuera del IGV)' },
];
