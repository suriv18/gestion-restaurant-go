import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import { Categoria, CategoriaAlta, CategoriaCambios, Pagina } from './carta.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

/** Acceso HTTP a las categorías de la carta. */
@Injectable({ providedIn: 'root' })
export class CategoriasApi {
  private readonly http = inject(HttpClient);

  /** Sin `es_activo` el servidor devuelve activas y de baja. */
  listar(cursor?: string): Observable<Pagina<Categoria>> {
    let params = new HttpParams().set('limit', '100');
    if (cursor) params = params.set('cursor', cursor);
    return this.http.get<Pagina<Categoria>>(`${API_BASE}/categorias`, { params });
  }

  crear(body: CategoriaAlta, key: string): Observable<Categoria> {
    return this.http.post<Categoria>(`${API_BASE}/categorias`, body, idem(key));
  }

  editar(id: string, cambios: CategoriaCambios): Observable<Categoria> {
    return this.http.patch<Categoria>(`${API_BASE}/categorias/${id}`, cambios);
  }

  baja(id: string, key: string): Observable<Categoria> {
    return this.http.post<Categoria>(`${API_BASE}/categorias/${id}/baja`, {}, idem(key));
  }

  restaurar(id: string, key: string): Observable<Categoria> {
    return this.http.post<Categoria>(`${API_BASE}/categorias/${id}/restauracion`, {}, idem(key));
  }
}
