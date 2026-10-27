import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import { Estacion, EstacionAlta, EstacionCambios, EstacionFiltros, Pagina } from './carta.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

/** Acceso HTTP a las estaciones de preparación. */
@Injectable({ providedIn: 'root' })
export class EstacionesApi {
  private readonly http = inject(HttpClient);

  listar(filtros: EstacionFiltros = {}, cursor?: string): Observable<Pagina<Estacion>> {
    let params = new HttpParams().set('limit', '100');
    if (filtros.sucursal_id) params = params.set('sucursal_id', filtros.sucursal_id);
    if (filtros.es_activo !== undefined)
      params = params.set('es_activo', String(filtros.es_activo));
    if (filtros.q) params = params.set('q', filtros.q);
    if (cursor) params = params.set('cursor', cursor);
    return this.http.get<Pagina<Estacion>>(`${API_BASE}/estaciones`, { params });
  }

  crear(body: EstacionAlta, key: string): Observable<Estacion> {
    return this.http.post<Estacion>(`${API_BASE}/estaciones`, body, idem(key));
  }

  editar(id: string, cambios: EstacionCambios): Observable<Estacion> {
    return this.http.patch<Estacion>(`${API_BASE}/estaciones/${id}`, cambios);
  }

  baja(id: string, key: string): Observable<Estacion> {
    return this.http.post<Estacion>(`${API_BASE}/estaciones/${id}/baja`, {}, idem(key));
  }

  restaurar(id: string, key: string): Observable<Estacion> {
    return this.http.post<Estacion>(`${API_BASE}/estaciones/${id}/restauracion`, {}, idem(key));
  }
}
