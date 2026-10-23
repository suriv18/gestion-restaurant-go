import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import type { Pagina } from '../../../core/http/collect';
import type {
  Mesa,
  MesaCambios,
  MesaNueva,
  Negocio,
  NegocioCambios,
  Sucursal,
  SucursalCambios,
  SucursalNueva,
  Zona,
  ZonaCambios,
  ZonaNueva,
} from '../configuracion.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

function page(cursor?: string): HttpParams {
  let params = new HttpParams().set('limit', '100');
  if (cursor) params = params.set('cursor', cursor);
  return params;
}

@Injectable({ providedIn: 'root' })
export class ConfiguracionApi {
  private readonly http = inject(HttpClient);

  negocio(): Observable<Negocio> {
    return this.http.get<Negocio>(`${API_BASE}/negocio`);
  }
  editarNegocio(cambios: NegocioCambios): Observable<Negocio> {
    return this.http.patch<Negocio>(`${API_BASE}/negocio`, cambios);
  }

  sucursales(cursor?: string): Observable<Pagina<Sucursal>> {
    return this.http.get<Pagina<Sucursal>>(`${API_BASE}/sucursales`, { params: page(cursor) });
  }
  crearSucursal(body: SucursalNueva, key: string): Observable<Sucursal> {
    return this.http.post<Sucursal>(`${API_BASE}/sucursales`, body, idem(key));
  }
  editarSucursal(id: string, cambios: SucursalCambios): Observable<Sucursal> {
    return this.http.patch<Sucursal>(`${API_BASE}/sucursales/${id}`, cambios);
  }
  bajaSucursal(id: string, motivo: string, key: string): Observable<Sucursal> {
    return this.http.post<Sucursal>(`${API_BASE}/sucursales/${id}/baja`, { motivo }, idem(key));
  }
  restaurarSucursal(id: string, key: string): Observable<Sucursal> {
    return this.http.post<Sucursal>(`${API_BASE}/sucursales/${id}/restauracion`, {}, idem(key));
  }

  zonas(cursor?: string): Observable<Pagina<Zona>> {
    return this.http.get<Pagina<Zona>>(`${API_BASE}/zonas`, { params: page(cursor) });
  }
  crearZona(body: ZonaNueva, key: string): Observable<Zona> {
    return this.http.post<Zona>(`${API_BASE}/zonas`, body, idem(key));
  }
  editarZona(id: string, cambios: ZonaCambios): Observable<Zona> {
    return this.http.patch<Zona>(`${API_BASE}/zonas/${id}`, cambios);
  }
  bajaZona(id: string, key: string): Observable<Zona> {
    return this.http.post<Zona>(`${API_BASE}/zonas/${id}/baja`, {}, idem(key));
  }
  restaurarZona(id: string, key: string): Observable<Zona> {
    return this.http.post<Zona>(`${API_BASE}/zonas/${id}/restauracion`, {}, idem(key));
  }

  mesas(cursor?: string): Observable<Pagina<Mesa>> {
    return this.http.get<Pagina<Mesa>>(`${API_BASE}/mesas`, { params: page(cursor) });
  }
  crearMesa(body: MesaNueva, key: string): Observable<Mesa> {
    return this.http.post<Mesa>(`${API_BASE}/mesas`, body, idem(key));
  }
  editarMesa(id: string, cambios: MesaCambios): Observable<Mesa> {
    return this.http.patch<Mesa>(`${API_BASE}/mesas/${id}`, cambios);
  }
  bajaMesa(id: string, motivo: string, key: string): Observable<Mesa> {
    return this.http.post<Mesa>(`${API_BASE}/mesas/${id}/baja`, { motivo }, idem(key));
  }
  restaurarMesa(id: string, key: string): Observable<Mesa> {
    return this.http.post<Mesa>(`${API_BASE}/mesas/${id}/restauracion`, {}, idem(key));
  }
}
