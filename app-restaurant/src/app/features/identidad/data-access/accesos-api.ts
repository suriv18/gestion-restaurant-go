import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import {
  NuevoUsuario,
  Pagina,
  Rol,
  SucursalRef,
  UsuarioCreado,
  UsuarioEquipo,
} from './accesos.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

/** Acceso HTTP a la administración de accesos (usuarios, roles y sedes asignables). */
@Injectable({ providedIn: 'root' })
export class AccesosApi {
  private readonly http = inject(HttpClient);

  usuarios(cursor?: string): Observable<Pagina<UsuarioEquipo>> {
    return this.http.get<Pagina<UsuarioEquipo>>(`${API_BASE}/usuarios`, { params: page(cursor) });
  }

  roles(cursor?: string): Observable<Pagina<Rol>> {
    return this.http.get<Pagina<Rol>>(`${API_BASE}/roles`, { params: page(cursor) });
  }

  /** Solo las sedes donde la persona conectada puede administrar accesos. */
  sucursalesAsignables(): Observable<{ items: SucursalRef[] }> {
    return this.http.get<{ items: SucursalRef[] }>(`${API_BASE}/sucursales-asignables`);
  }

  crearUsuario(body: NuevoUsuario, key: string): Observable<UsuarioCreado> {
    return this.http.post<UsuarioCreado>(`${API_BASE}/usuarios`, body, idem(key));
  }

  /** Reemplaza los roles y las sedes de la persona. */
  asignar(id: string, rolIds: string[], sucursalIds: string[]): Observable<UsuarioCreado> {
    return this.http.put<UsuarioCreado>(`${API_BASE}/usuarios/${id}/asignaciones`, {
      rol_ids: rolIds,
      sucursal_ids: sucursalIds,
    });
  }

  baja(id: string, motivo: string, key: string): Observable<UsuarioCreado> {
    return this.http.post<UsuarioCreado>(`${API_BASE}/usuarios/${id}/baja`, { motivo }, idem(key));
  }

  restaurar(id: string, key: string): Observable<UsuarioCreado> {
    return this.http.post<UsuarioCreado>(`${API_BASE}/usuarios/${id}/restauracion`, {}, idem(key));
  }
}

function page(cursor?: string): HttpParams {
  let params = new HttpParams().set('limit', '100');
  if (cursor) params = params.set('cursor', cursor);
  return params;
}
