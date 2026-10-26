import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import {
  DisponibilidadCuerpo,
  Pagina,
  Producto,
  ProductoAlta,
  ProductoCambios,
  ProductoDisponibilidad,
  ProductoFiltros,
} from './carta.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

const LIMITE = 50;

/** Acceso HTTP a los productos de la carta y su disponibilidad por sede. */
@Injectable({ providedIn: 'root' })
export class ProductosApi {
  private readonly http = inject(HttpClient);

  listar(filtros: ProductoFiltros, cursor?: string): Observable<Pagina<Producto>> {
    let params = new HttpParams()
      .set('es_activo', filtros.es_activo)
      .set('limit', String(filtros.limit ?? LIMITE));
    if (filtros.q) params = params.set('q', filtros.q);
    if (filtros.categoria_id) params = params.set('categoria_id', filtros.categoria_id);
    if (filtros.sucursal_id) {
      params = params.set('sucursal_id', filtros.sucursal_id);
      // `disponible` solo tiene sentido con sede.
      if (filtros.disponible !== undefined) {
        params = params.set('disponible', String(filtros.disponible));
      }
    }
    if (cursor) params = params.set('cursor', cursor);
    return this.http.get<Pagina<Producto>>(`${API_BASE}/productos`, { params });
  }

  obtener(id: string, sucursalId?: string): Observable<Producto> {
    let params = new HttpParams();
    if (sucursalId) params = params.set('sucursal_id', sucursalId);
    return this.http.get<Producto>(`${API_BASE}/productos/${id}`, { params });
  }

  crear(body: ProductoAlta, key: string): Observable<Producto> {
    return this.http.post<Producto>(`${API_BASE}/productos`, body, idem(key));
  }

  editar(id: string, cambios: ProductoCambios): Observable<Producto> {
    return this.http.patch<Producto>(`${API_BASE}/productos/${id}`, cambios);
  }

  baja(id: string, motivo: string, key: string): Observable<Producto> {
    return this.http.post<Producto>(`${API_BASE}/productos/${id}/baja`, { motivo }, idem(key));
  }

  restaurar(id: string, key: string): Observable<Producto> {
    return this.http.post<Producto>(`${API_BASE}/productos/${id}/restauracion`, {}, idem(key));
  }

  /**
   * PUT idempotente por naturaleza (sin Idempotency-Key). En el cuerpo, `precio_venta` y
   * `estacion_id` ausentes = sin cambio; null = precio general / sin estación.
   */
  ponerDisponibilidad(
    id: string,
    cuerpo: DisponibilidadCuerpo,
  ): Observable<ProductoDisponibilidad> {
    return this.http.put<ProductoDisponibilidad>(
      `${API_BASE}/productos/${id}/disponibilidad`,
      cuerpo,
    );
  }
}
