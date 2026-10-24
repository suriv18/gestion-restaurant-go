import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import {
  Cliente,
  EdicionCliente,
  EdicionProveedor,
  NuevoCliente,
  NuevoProveedor,
  Pagina,
  Proveedor,
} from './directorio.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

/** Acceso HTTP al directorio: clientes y proveedores. */
@Injectable({ providedIn: 'root' })
export class DirectorioApi {
  private readonly http = inject(HttpClient);

  clientes(cursor?: string): Observable<Pagina<Cliente>> {
    return this.http.get<Pagina<Cliente>>(`${API_BASE}/clientes`, { params: page(cursor) });
  }

  crearCliente(body: NuevoCliente, key: string): Observable<Cliente> {
    return this.http.post<Cliente>(`${API_BASE}/clientes`, body, idem(key));
  }

  editarCliente(id: string, body: EdicionCliente): Observable<Cliente> {
    return this.http.patch<Cliente>(`${API_BASE}/clientes/${id}`, body);
  }

  bajaCliente(id: string, motivo: string, key: string): Observable<Cliente> {
    return this.http.post<Cliente>(`${API_BASE}/clientes/${id}/baja`, { motivo }, idem(key));
  }

  restaurarCliente(id: string, key: string): Observable<Cliente> {
    return this.http.post<Cliente>(`${API_BASE}/clientes/${id}/restauracion`, {}, idem(key));
  }

  proveedores(cursor?: string): Observable<Pagina<Proveedor>> {
    return this.http.get<Pagina<Proveedor>>(`${API_BASE}/proveedores`, { params: page(cursor) });
  }

  crearProveedor(body: NuevoProveedor, key: string): Observable<Proveedor> {
    return this.http.post<Proveedor>(`${API_BASE}/proveedores`, body, idem(key));
  }

  editarProveedor(id: string, body: EdicionProveedor): Observable<Proveedor> {
    return this.http.patch<Proveedor>(`${API_BASE}/proveedores/${id}`, body);
  }

  bajaProveedor(id: string, motivo: string, key: string): Observable<Proveedor> {
    return this.http.post<Proveedor>(`${API_BASE}/proveedores/${id}/baja`, { motivo }, idem(key));
  }

  restaurarProveedor(id: string, key: string): Observable<Proveedor> {
    return this.http.post<Proveedor>(`${API_BASE}/proveedores/${id}/restauracion`, {}, idem(key));
  }
}

function page(cursor?: string): HttpParams {
  let params = new HttpParams().set('limit', '100');
  if (cursor) params = params.set('cursor', cursor);
  return params;
}
