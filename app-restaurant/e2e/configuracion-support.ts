import type { Page, Route } from '@playwright/test';

export interface Llamada {
  method: string;
  path: string;
  body: unknown;
  idempotencyKey: string | undefined;
}

const AUDIT = {
  creado_en: '2026-10-21T10:00:00Z',
  creado_por: '1',
  actualizado_en: '2026-10-21T10:00:00Z',
  actualizado_por: '1',
};

export interface ConfiguracionMock {
  llamadas: Llamada[];
  /** Próximo rechazo programado para `METHOD /ruta` (una sola vez). */
  rechazos: Map<string, { status: number; code: string; detail: string }>;
}

const problem = (status: number, code: string, detail: string) => ({
  status,
  contentType: 'application/problem+json',
  body: JSON.stringify({ type: 'about:blank', title: 'Error', status, code, detail }),
});

export async function mockConfiguracionApi(page: Page): Promise<ConfiguracionMock> {
  const mock: ConfiguracionMock = { llamadas: [], rechazos: new Map() };
  let seq = 100;
  const negocio = {
    id: '1',
    singleton: true,
    ruc: '20123456789',
    razon_social: 'Mi Restaurante SAC',
    nombre_comercial: 'Mi Restaurante',
    direccion_fiscal: 'Av. Principal 123',
    ubigeo: null as string | null,
    telefono: null as string | null,
    email: null as string | null,
    moneda: 'PEN',
    zona_horaria: 'America/Lima',
    es_activo: true,
    ...AUDIT,
  };
  const sedes = [
    {
      id: '1',
      negocio_id: '1',
      codigo: 'PRINCIPAL',
      nombre: 'Sede Principal',
      direccion: 'Av. Principal 123',
      ubigeo: null,
      codigo_establecimiento_sunat: null,
      telefono: null,
      es_activo: true,
      motivo_eliminacion: null as string | null,
      eliminado_en: null,
      eliminado_por: null,
      ...AUDIT,
    },
  ];
  const zonas = [
    {
      id: '10',
      sucursal_id: '1',
      nombre: 'Salón',
      orden_visualizacion: 0,
      es_activo: true,
      ...AUDIT,
    },
  ];
  const mesas = [
    {
      id: '100',
      sucursal_id: '1',
      zona_id: '10',
      numero_mesa: '1',
      capacidad: 4,
      habilitada: true,
      posicion_x: null,
      posicion_y: null,
      es_activo: true,
      motivo_eliminacion: null as string | null,
      eliminado_en: null,
      eliminado_por: null,
      ...AUDIT,
    },
  ];

  await page.route('**/api/v1/**', async (route: Route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace('/api/v1', '');
    if (path === '/sesion' || path.startsWith('/auth/')) return route.fallback();
    const method = request.method();
    const body = request.postData() ? JSON.parse(request.postData() as string) : undefined;
    mock.llamadas.push({
      method,
      path,
      body,
      idempotencyKey: request.headers()['idempotency-key'],
    });
    const clave = `${method} ${path}`;
    const rechazo = mock.rechazos.get(clave);
    if (rechazo) {
      mock.rechazos.delete(clave);
      return route.fulfill(problem(rechazo.status, rechazo.code, rechazo.detail));
    }
    const json = (data: unknown, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    const lista = (items: unknown[]) => json({ items, nextCursor: null });
    const id = path.split('/')[2];

    if (path === '/negocio' && method === 'GET') return json(negocio);
    if (path === '/negocio' && method === 'PATCH') return json(Object.assign(negocio, body));
    if (path === '/sucursales' && method === 'GET') return lista(sedes);
    if (path === '/sucursales' && method === 'POST') {
      // Como el backend: quien crea la sede no queda asignado a ella, así que GET /sucursales
      // (solo sedes visibles para el actor) no la devuelve después.
      const nueva = { ...sedes[0], id: String(++seq), es_activo: true, ...body };
      return json(nueva, 201);
    }
    if (path.startsWith('/sucursales/') && path.endsWith('/baja')) {
      const s = sedes.find((x) => x.id === id)!;
      Object.assign(s, { es_activo: false, motivo_eliminacion: body.motivo });
      return json(s);
    }
    if (path.startsWith('/sucursales/') && path.endsWith('/restauracion')) {
      const s = sedes.find((x) => x.id === id)!;
      Object.assign(s, { es_activo: true, motivo_eliminacion: null });
      return json(s);
    }
    if (path === '/zonas' && method === 'GET') return lista(zonas);
    if (path === '/zonas' && method === 'POST') {
      const nueva = {
        id: String(++seq),
        es_activo: true,
        orden_visualizacion: 0,
        ...AUDIT,
        ...body,
      };
      zonas.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/zonas/') && path.endsWith('/baja')) {
      const z = zonas.find((x) => x.id === id)!;
      z.es_activo = false;
      return json(z);
    }
    if (path.startsWith('/zonas/') && path.endsWith('/restauracion')) {
      const z = zonas.find((x) => x.id === id)!;
      z.es_activo = true;
      return json(z);
    }
    if (path === '/mesas' && method === 'GET') return lista(mesas);
    if (path === '/mesas' && method === 'POST') {
      const nueva = {
        id: String(++seq),
        habilitada: true,
        es_activo: true,
        posicion_x: null,
        posicion_y: null,
        motivo_eliminacion: null,
        eliminado_en: null,
        eliminado_por: null,
        ...AUDIT,
        ...body,
      };
      mesas.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/mesas/') && method === 'PATCH') {
      const m = mesas.find((x) => x.id === id)!;
      return json(Object.assign(m, body));
    }
    if (path.startsWith('/mesas/') && path.endsWith('/baja')) {
      const m = mesas.find((x) => x.id === id)!;
      Object.assign(m, { es_activo: false, motivo_eliminacion: body.motivo });
      return json(m);
    }
    if (path.startsWith('/mesas/') && path.endsWith('/restauracion')) {
      const m = mesas.find((x) => x.id === id)!;
      Object.assign(m, { es_activo: true, motivo_eliminacion: null });
      return json(m);
    }
    return route.fulfill(problem(404, 'no_encontrado', 'Ruta no simulada: ' + clave));
  });
  return mock;
}
