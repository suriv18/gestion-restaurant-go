import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { API_BASE } from '../../../core/auth/auth.models';
import type { Pagina } from '../../../core/http/collect';
import { EdicionEmpleado, Empleado, NuevoEmpleado } from './empleados.models';

const idem = (key: string) => ({ headers: new HttpHeaders({ 'Idempotency-Key': key }) });

/** Acceso HTTP a los empleados del restaurante. */
@Injectable({ providedIn: 'root' })
export class EmpleadosApi {
  private readonly http = inject(HttpClient);

  empleados(cursor?: string): Observable<Pagina<Empleado>> {
    let params = new HttpParams().set('limit', '100');
    if (cursor) params = params.set('cursor', cursor);
    return this.http.get<Pagina<Empleado>>(`${API_BASE}/empleados`, { params });
  }

  crear(body: NuevoEmpleado, key: string): Observable<Empleado> {
    return this.http.post<Empleado>(`${API_BASE}/empleados`, body, idem(key));
  }

  editar(id: string, body: EdicionEmpleado): Observable<Empleado> {
    return this.http.patch<Empleado>(`${API_BASE}/empleados/${id}`, body);
  }

  baja(id: string, motivo: string, key: string): Observable<Empleado> {
    return this.http.post<Empleado>(`${API_BASE}/empleados/${id}/baja`, { motivo }, idem(key));
  }

  restaurar(id: string, key: string): Observable<Empleado> {
    return this.http.post<Empleado>(`${API_BASE}/empleados/${id}/restauracion`, {}, idem(key));
  }
}
