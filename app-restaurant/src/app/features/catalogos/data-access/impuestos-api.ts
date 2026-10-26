import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import { Impuesto, Pagina, TasaAlta, TasaCambios, TasaImpuesto } from './carta.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

function page(cursor?: string): HttpParams {
  let params = new HttpParams().set('limit', '100');
  if (cursor) params = params.set('cursor', cursor);
  return params;
}

/** Acceso HTTP a los impuestos (solo lectura) y a sus tasas con vigencia. */
@Injectable({ providedIn: 'root' })
export class ImpuestosApi {
  private readonly http = inject(HttpClient);

  impuestos(cursor?: string): Observable<Pagina<Impuesto>> {
    return this.http.get<Pagina<Impuesto>>(`${API_BASE}/impuestos`, { params: page(cursor) });
  }

  /** El servidor exige `impuesto_id` (400 si falta): pásalo siempre. */
  tasas(impuestoId: string, cursor?: string): Observable<Pagina<TasaImpuesto>> {
    const params = page(cursor).set('impuesto_id', impuestoId);
    return this.http.get<Pagina<TasaImpuesto>>(`${API_BASE}/tasas-impuesto`, { params });
  }

  crearTasa(body: TasaAlta, key: string): Observable<TasaImpuesto> {
    return this.http.post<TasaImpuesto>(`${API_BASE}/tasas-impuesto`, body, idem(key));
  }

  editarTasa(id: string, cambios: TasaCambios): Observable<TasaImpuesto> {
    return this.http.patch<TasaImpuesto>(`${API_BASE}/tasas-impuesto/${id}`, cambios);
  }
}
