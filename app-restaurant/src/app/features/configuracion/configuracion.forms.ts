/** Texto de un campo obligatorio, sin espacios sobrantes. */
export const texto = (valor: string): string => valor.trim();

/** Texto de un campo opcional: vacío = null (el servidor lo limpia). */
export const opcional = (valor: string): string | null => {
  const limpio = valor.trim();
  return limpio === '' ? null : limpio;
};

export const entero = (valor: string): number => Number(valor.trim());

type Valor = string | number | boolean | null;

/** Campos de `nuevo` cuyo valor difiere del `actual`: base de los PATCH parciales. */
export function cambiosDe(
  actual: Record<string, Valor>,
  nuevo: Record<string, Valor>,
): Record<string, Valor> {
  const cambios: Record<string, Valor> = {};
  for (const [clave, valor] of Object.entries(nuevo)) {
    if (actual[clave] !== valor) cambios[clave] = valor;
  }
  return cambios;
}
