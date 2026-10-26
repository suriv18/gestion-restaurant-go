import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { API_BASE } from '../../../core/auth/auth.models';
import { CategoriasApi } from './categorias-api';
import { EstacionesApi } from './estaciones-api';
import { ImpuestosApi } from './impuestos-api';
import { ProductosApi } from './productos-api';

describe('APIs de la carta', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
  });
  afterEach(() => http.verify());

  it('lista productos con es_activo explícito y sin disponible si no hay sede', () => {
    TestBed.inject(ProductosApi)
      .listar({ es_activo: 'true', q: 'lomo', categoria_id: '3', disponible: false }, 'abc')
      .subscribe();
    const req = http.expectOne((r) => r.url === `${API_BASE}/productos`);
    expect(req.request.method).toBe('GET');
    expect(req.request.params.get('es_activo')).toBe('true');
    expect(req.request.params.get('q')).toBe('lomo');
    expect(req.request.params.get('categoria_id')).toBe('3');
    expect(req.request.params.get('cursor')).toBe('abc');
    expect(req.request.params.get('limit')).toBe('50');
    expect(req.request.params.has('sucursal_id')).toBe(false);
    expect(req.request.params.has('disponible')).toBe(false);
    req.flush({ items: [], nextCursor: null });
  });

  it('envía disponible junto con la sede', () => {
    TestBed.inject(ProductosApi)
      .listar({ es_activo: 'todos', sucursal_id: '7', disponible: false })
      .subscribe();
    const req = http.expectOne((r) => r.url === `${API_BASE}/productos`);
    expect(req.request.params.get('es_activo')).toBe('todos');
    expect(req.request.params.get('sucursal_id')).toBe('7');
    expect(req.request.params.get('disponible')).toBe('false');
    req.flush({ items: [], nextCursor: null });
  });

  it('crea, da de baja y restaura con Idempotency-Key; edita y pone disponibilidad sin ella', () => {
    const api = TestBed.inject(ProductosApi);
    const alta = {
      categoria_id: '1',
      codigo: 'P1',
      nombre: 'Lomo',
      descripcion: null,
      tipo: 'PREPARADO' as const,
      precio_venta: '25.00',
      afectacion_igv: '10' as const,
      impuesto_id: '1',
      imagen_url: null,
    };

    api.crear(alta, 'k1').subscribe();
    let req = http.expectOne(`${API_BASE}/productos`);
    expect(req.request.method).toBe('POST');
    expect(req.request.headers.get('Idempotency-Key')).toBe('k1');
    expect(req.request.body).toEqual(alta);
    req.flush({});

    api.editar('9', { nombre: 'Lomo saltado', descripcion: null }).subscribe();
    req = http.expectOne(`${API_BASE}/productos/9`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.headers.has('Idempotency-Key')).toBe(false);
    expect(req.request.body).toEqual({ nombre: 'Lomo saltado', descripcion: null });
    req.flush({});

    api.baja('9', 'Ya no se vende', 'k2').subscribe();
    req = http.expectOne(`${API_BASE}/productos/9/baja`);
    expect(req.request.method).toBe('POST');
    expect(req.request.headers.get('Idempotency-Key')).toBe('k2');
    expect(req.request.body).toEqual({ motivo: 'Ya no se vende' });
    req.flush({});

    api.restaurar('9', 'k3').subscribe();
    req = http.expectOne(`${API_BASE}/productos/9/restauracion`);
    expect(req.request.headers.get('Idempotency-Key')).toBe('k3');
    req.flush({});

    api
      .ponerDisponibilidad('9', { sucursal_id: '7', disponible: false, estacion_id: null })
      .subscribe();
    req = http.expectOne(`${API_BASE}/productos/9/disponibilidad`);
    expect(req.request.method).toBe('PUT');
    expect(req.request.headers.has('Idempotency-Key')).toBe(false);
    // precio_venta ausente = sin cambio; estacion_id null = sin estación.
    expect(req.request.body).toEqual({ sucursal_id: '7', disponible: false, estacion_id: null });
    expect('precio_venta' in req.request.body).toBe(false);
    req.flush({});
  });

  it('categorías: baja y restauración con clave, edición sin ella', () => {
    const api = TestBed.inject(CategoriasApi);
    api.crear({ nombre: 'Fondos', descripcion: null, orden_visualizacion: 0 }, 'c1').subscribe();
    let req = http.expectOne(`${API_BASE}/categorias`);
    expect(req.request.headers.get('Idempotency-Key')).toBe('c1');
    req.flush({});

    api.editar('2', { orden_visualizacion: 3 }).subscribe();
    req = http.expectOne(`${API_BASE}/categorias/2`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.headers.has('Idempotency-Key')).toBe(false);
    req.flush({});

    api.baja('2', 'c2').subscribe();
    req = http.expectOne(`${API_BASE}/categorias/2/baja`);
    expect(req.request.headers.get('Idempotency-Key')).toBe('c2');
    expect(req.request.body).toEqual({});
    req.flush({});

    api.restaurar('2', 'c3').subscribe();
    req = http.expectOne(`${API_BASE}/categorias/2/restauracion`);
    expect(req.request.headers.get('Idempotency-Key')).toBe('c3');
    req.flush({});

    api.listar().subscribe();
    req = http.expectOne((r) => r.url === `${API_BASE}/categorias`);
    expect(req.request.params.has('es_activo')).toBe(false);
    req.flush({ items: [], nextCursor: null });
  });

  it('estaciones: filtros y escrituras', () => {
    const api = TestBed.inject(EstacionesApi);
    api.listar({ sucursal_id: '7', es_activo: true, q: 'par' }).subscribe();
    let req = http.expectOne((r) => r.url === `${API_BASE}/estaciones`);
    expect(req.request.params.get('sucursal_id')).toBe('7');
    expect(req.request.params.get('es_activo')).toBe('true');
    expect(req.request.params.get('q')).toBe('par');
    req.flush({ items: [], nextCursor: null });

    api.crear({ sucursal_id: '7', codigo: 'PAR', nombre: 'Parrilla' }, 'e1').subscribe();
    req = http.expectOne(`${API_BASE}/estaciones`);
    expect(req.request.headers.get('Idempotency-Key')).toBe('e1');
    expect(req.request.body).toEqual({ sucursal_id: '7', codigo: 'PAR', nombre: 'Parrilla' });
    req.flush({});

    api.editar('4', { nombre: 'Grill' }).subscribe();
    req = http.expectOne(`${API_BASE}/estaciones/4`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.headers.has('Idempotency-Key')).toBe(false);
    req.flush({});

    api.baja('4', 'e2').subscribe();
    req = http.expectOne(`${API_BASE}/estaciones/4/baja`);
    expect(req.request.headers.get('Idempotency-Key')).toBe('e2');
    req.flush({});

    api.restaurar('4', 'e3').subscribe();
    req = http.expectOne(`${API_BASE}/estaciones/4/restauracion`);
    expect(req.request.headers.get('Idempotency-Key')).toBe('e3');
    req.flush({});
  });

  it('impuestos y tasas', () => {
    const api = TestBed.inject(ImpuestosApi);
    api.impuestos().subscribe();
    let req = http.expectOne((r) => r.url === `${API_BASE}/impuestos`);
    expect(req.request.method).toBe('GET');
    req.flush({ items: [], nextCursor: null });

    api.tasas('1', 'cur').subscribe();
    req = http.expectOne((r) => r.url === `${API_BASE}/tasas-impuesto`);
    expect(req.request.params.get('impuesto_id')).toBe('1');
    expect(req.request.params.get('cursor')).toBe('cur');
    req.flush({ items: [], nextCursor: null });

    const alta = {
      impuesto_id: '1',
      tasa: '18',
      vigente_desde: '2026-01-01',
      vigente_hasta: null,
      fundamento: null,
    };
    api.crearTasa(alta, 't1').subscribe();
    req = http.expectOne(`${API_BASE}/tasas-impuesto`);
    expect(req.request.method).toBe('POST');
    expect(req.request.headers.get('Idempotency-Key')).toBe('t1');
    expect(req.request.body).toEqual(alta);
    req.flush({});

    api.editarTasa('5', { vigente_hasta: '2027-01-01' }).subscribe();
    req = http.expectOne(`${API_BASE}/tasas-impuesto/5`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.headers.has('Idempotency-Key')).toBe(false);
    expect(req.request.body).toEqual({ vigente_hasta: '2027-01-01' });
    req.flush({});
  });
});
