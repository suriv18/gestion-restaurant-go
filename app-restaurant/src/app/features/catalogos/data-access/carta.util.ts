/**
 * Utilidades de decimales y fechas de la carta. El dinero y las tasas se manipulan siempre como
 * texto: nunca se convierten a `number`.
 */

const PRECIO = /^(\d{1,8})(?:\.(\d{1,2}))?$/;
const FACTOR = /^(\d{1,6})(?:\.(\d{1,6}))?$/;
const TASA = /^(\d{1,3})(?:\.(\d{1,4}))?$/;
const FECHA = /^(\d{4})-(\d{2})-(\d{2})$/;

/** Precio >= 0 con hasta 2 decimales (el servidor es la fuente de verdad). */
export function precioValido(valor: string): boolean {
  return PRECIO.test(valor.trim());
}

/** Factor de consumo > 0 con hasta 6 decimales. */
export function factorValido(valor: string): boolean {
  const m = FACTOR.exec(valor.trim());
  return !!m && /[1-9]/.test(m[1] + (m[2] ?? ''));
}

/** Tasa entre 0 y 100 con hasta 4 decimales. */
export function tasaValida(valor: string): boolean {
  const m = TASA.exec(valor.trim());
  if (!m) return false;
  const entera = m[1].replace(/^0+(?=\d)/, '');
  // Mismo largo: la comparación de texto equivale a la numérica.
  if (entera.length > 3 || (entera.length === 3 && entera > '100')) return false;
  return entera !== '100' || !/[1-9]/.test(m[2] ?? '');
}

/** Fecha real AAAA-MM-DD (años 1900 a 2999, como el servidor). */
export function fechaValida(valor: string): boolean {
  const m = FECHA.exec(valor.trim());
  if (!m) return false;
  const [anio, mes, dia] = [Number(m[1]), Number(m[2]), Number(m[3])];
  if (anio < 1900 || anio > 2999 || mes < 1 || mes > 12 || dia < 1) return false;
  return dia <= new Date(Date.UTC(anio, mes, 0)).getUTCDate();
}

/** `S/ 25.50` a partir de «25.5», «25» o «25.50», sin pasar por `number`. Lo no numérico se devuelve igual. */
export function formatearSoles(valor: string | null | undefined): string {
  if (valor === null || valor === undefined) return '';
  const m = /^(\d+)(?:\.(\d*))?$/.exec(valor.trim());
  if (!m) return valor;
  const entera = m[1].replace(/^0+(?=\d)/, '');
  const decimales = ((m[2] ?? '') + '00').slice(0, 2);
  return `S/ ${entera}.${decimales}`;
}

/**
 * Forma canónica de un decimal escrito como texto («25.50» y «25.5» -> «25.5»; «1.000000» -> «1»),
 * para comparar sin `number` lo que escribió la persona con lo que devuelve el servidor.
 */
export function normalizarDecimal(valor: string): string {
  const m = /^(\d+)(?:\.(\d*))?$/.exec(valor.trim());
  if (!m) return valor.trim();
  const entera = m[1].replace(/^0+(?=\d)/, '');
  const decimales = (m[2] ?? '').replace(/0+$/, '');
  return decimales ? `${entera}.${decimales}` : entera;
}

/** Tasa para mostrar: «18.0000» -> «18 %», «10.5000» -> «10.5 %». */
export function formatearTasa(valor: string): string {
  const m = /^(\d+)(?:\.(\d*))?$/.exec(valor.trim());
  if (!m) return valor;
  const decimales = (m[2] ?? '').replace(/0+$/, '');
  return `${m[1].replace(/^0+(?=\d)/, '')}${decimales ? '.' + decimales : ''} %`;
}

/** Compara fechas ISO como texto: negativo si a < b, 0 si iguales, positivo si a > b. */
export function compararFechas(a: string, b: string): number {
  return a < b ? -1 : a > b ? 1 : 0;
}

/** La vigencia es [desde, hasta): `hasta` null = abierta. `hoy` en AAAA-MM-DD. */
export function vigenteEn(desde: string, hasta: string | null, hoy: string): boolean {
  return compararFechas(desde, hoy) <= 0 && (hasta === null || compararFechas(hoy, hasta) < 0);
}

/** `2026-01-05` -> `05/01/2026`, por texto. Lo que no es una fecha ISO se devuelve igual. */
export function formatearFecha(iso: string): string {
  const m = FECHA.exec(iso.trim());
  return m ? `${m[3]}/${m[2]}/${m[1]}` : iso;
}

/** «desde 01/01/2026 – hasta 01/01/2027» o «desde 01/01/2026 (vigente)» si no tiene fin. */
export function formatearVigencia(desde: string, hasta: string | null): string {
  return hasta === null
    ? `desde ${formatearFecha(desde)} (vigente)`
    : `desde ${formatearFecha(desde)} – hasta ${formatearFecha(hasta)}`;
}

/** Fecha de hoy en Lima (AAAA-MM-DD), la zona horaria del negocio; no depende del reloj del navegador. */
export function hoyLima(ahora: Date = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'America/Lima',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(ahora);
}
