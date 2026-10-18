import { Pipe, PipeTransform } from '@angular/core';
const pen = new Intl.NumberFormat('es-PE', { style: 'currency', currency: 'PEN' });
@Pipe({ name: 'money' })
export class MoneyPipe implements PipeTransform {
  transform(cents: number | undefined): string {
    return pen.format((cents ?? 0) / 100);
  }
}
/** Entrada decimal validada; evita aceptar notación exponencial y decimales silenciosamente truncados. */
export function toCents(value: string): number {
  if (!/^\d+(\.\d{1,2})?$/.test(value.trim()))
    throw new Error('Usa un importe positivo con hasta dos decimales.');
  const [integer, fraction = ''] = value.trim().split('.');
  const cents = Number(integer) * 100 + Number(fraction.padEnd(2, '0'));
  if (!Number.isSafeInteger(cents)) throw new Error('El importe es demasiado grande.');
  return cents;
}
