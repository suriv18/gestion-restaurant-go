import type { Page, Route } from '@playwright/test';

export interface Llamada {
  method: string;
  path: string;
  query: Record<string, string>;
  body: unknown;
  idempotencyKey: string | undefined;
}

export interface CartaMock {
  llamadas: Llamada[];
  /** Próximo rechazo programado para `METHOD /ruta` (una sola vez). */
  rechazos: Map<string, { status: number; code: string; detail: string }>;
  /** Tamaño de página del listado de productos (para probar «Cargar más»); por defecto el pedido. */
  paginaDe: number | null;
}

const problem = (status: number, code: string, detail: string) => ({
  status,
  contentType: 'application/problem+json',
  body: JSON.stringify({ type: 'about:blank', title: 'Error', status, code, detail }),
});

interface ProductoMock {
  id: string;
  categoria_id: string;
  codigo: string;
  nombre: string;
  descripcion: string | null;
  tipo: string;
  articulo_id: string | null;
  factor_consumo: string;
  precio_venta: string;
  afectacion_igv: string;
  impuesto_id: string | null;
  imagen_url: string | null;
  es_activo: boolean;
  motivo_eliminacion: string | null;
}

interface FilaSede {
  disponible: boolean;
  precio_sucursal: string | null;
  estacion_id: string | null;
}

/**
 * API en memoria de la carta, las estaciones, los impuestos y sus tasas. Aplica las reglas que
 * la pantalla debe respetar (baja de categoría con productos, vigencias que se cruzan) y deja
 * que `rechazos` fuerce cualquier otro error del servidor.
 */
export async function mockCartaApi(page: Page): Promise<CartaMock> {
  const mock: CartaMock = { llamadas: [], rechazos: new Map(), paginaDe: null };
  let seq = 100;

  const categorias = [
    { id: '1', nombre: 'Entradas', descripcion: null as string | null, orden_visualizacion: 1 },
    { id: '2', nombre: 'Fondos', descripcion: 'Platos de fondo', orden_visualizacion: 2 },
    { id: '3', nombre: 'Bebidas', descripcion: null as string | null, orden_visualizacion: 3 },
  ].map((c) => ({ ...c, es_activo: true }));

  const producto = (
    id: string,
    categoria: string,
    codigo: string,
    nombre: string,
    precio: string,
  ) =>
    ({
      id,
      categoria_id: categoria,
      codigo,
      nombre,
      descripcion: null,
      tipo: 'PREPARADO',
      articulo_id: null,
      factor_consumo: '1.000000',
      precio_venta: precio,
      afectacion_igv: '10',
      impuesto_id: '1',
      imagen_url: null,
      es_activo: true,
      motivo_eliminacion: null,
    }) as ProductoMock;
  const productos: ProductoMock[] = [
    producto('10', '1', 'CEV', 'Ceviche clásico', '32.00'),
    producto('11', '2', 'LOM', 'Lomo saltado', '38.50'),
    producto('12', '3', 'CHI', 'Chicha morada', '9.00'),
  ];
  // Disponibilidad por sede: «Ceviche» disponible, «Lomo» pausado, «Chicha» sin fila (no ofrecido).
  const sedes: Record<string, Record<string, FilaSede>> = {
    '1': {
      '10': { disponible: true, precio_sucursal: null, estacion_id: null },
      '11': { disponible: false, precio_sucursal: '40.00', estacion_id: '50' },
    },
  };

  const estaciones = [
    { id: '50', sucursal_id: '1', codigo: 'COCINA', nombre: 'Cocina', es_activo: true },
  ];
  const impuestos = [
    {
      id: '1',
      codigo: 'IGV',
      nombre: 'IGV',
      descripcion: 'Impuesto general a las ventas',
      es_activo: true,
    },
  ];
  const tasas = [
    {
      id: '70',
      impuesto_id: '1',
      tasa: '18.0000',
      vigente_desde: '2020-01-01',
      vigente_hasta: null as string | null,
      fundamento: 'Ley del IGV' as string | null,
    },
  ];

  const enSucursal = (p: ProductoMock, sede: string | undefined) => {
    if (!sede) return {};
    const fila = sedes[sede]?.[p.id];
    if (!fila) return { en_sucursal: null };
    return {
      en_sucursal: {
        ...fila,
        precio_efectivo: fila.precio_sucursal ?? p.precio_venta,
      },
    };
  };

  await page.route('**/api/v1/**', async (route: Route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace('/api/v1', '');
    if (path === '/sesion' || path.startsWith('/auth/')) return route.fallback();
    const method = request.method();
    const query = Object.fromEntries(url.searchParams.entries());
    const body = request.postData() ? JSON.parse(request.postData() as string) : undefined;
    mock.llamadas.push({
      method,
      path,
      query,
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

    if (path === '/sucursales' && method === 'GET') {
      return lista([{ id: '1', codigo: 'PRINCIPAL', nombre: 'Sede Principal', es_activo: true }]);
    }

    // --- Categorías ---
    if (path === '/categorias' && method === 'GET') return lista(categorias);
    if (path === '/categorias' && method === 'POST') {
      const nueva = { id: String(++seq), es_activo: true, ...body };
      categorias.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/categorias/') && method === 'PATCH') {
      return json(
        Object.assign(
          categorias.find((c) => c.id === id)!,
          body,
        ),
      );
    }
    if (path.startsWith('/categorias/') && path.endsWith('/baja')) {
      if (productos.some((p) => p.categoria_id === id && p.es_activo)) {
        return route.fulfill(
          problem(
            409,
            'categoria_con_productos_activos',
            'La categoría tiene productos activos: muévelos o dalos de baja primero.',
          ),
        );
      }
      const c = categorias.find((x) => x.id === id)!;
      c.es_activo = false;
      return json(c);
    }
    if (path.startsWith('/categorias/') && path.endsWith('/restauracion')) {
      const c = categorias.find((x) => x.id === id)!;
      c.es_activo = true;
      return json(c);
    }

    // --- Productos ---
    if (path === '/productos' && method === 'GET') {
      const activo = query['es_activo'];
      let items = productos.filter(
        (p) => activo === 'todos' || String(p.es_activo) === (activo ?? 'true'),
      );
      if (query['q']) {
        const q = query['q'].toLowerCase();
        items = items.filter(
          (p) => p.nombre.toLowerCase().includes(q) || p.codigo.toLowerCase().includes(q),
        );
      }
      if (query['categoria_id'])
        items = items.filter((p) => p.categoria_id === query['categoria_id']);
      const sede = query['sucursal_id'];
      if (sede && query['disponible'] !== undefined) {
        const quiere = query['disponible'] === 'true';
        items = items.filter((p) => sedes[sede]?.[p.id]?.disponible === quiere);
      }
      const tam = mock.paginaDe ?? Number(query['limit'] ?? 50);
      const desde = Number(query['cursor'] ?? 0);
      const trozo = items.slice(desde, desde + tam);
      const siguiente = desde + tam < items.length ? String(desde + tam) : null;
      return json({
        items: trozo.map((p) => ({ ...p, ...enSucursal(p, sede) })),
        nextCursor: siguiente,
      });
    }
    if (path === '/productos' && method === 'POST') {
      if (productos.some((p) => p.codigo === body.codigo)) {
        return route.fulfill(
          problem(409, 'codigo_duplicado', 'Ya existe un producto con ese código.'),
        );
      }
      const nuevo = {
        id: String(++seq),
        articulo_id: null,
        factor_consumo: '1.000000',
        es_activo: true,
        motivo_eliminacion: null,
        ...body,
      };
      productos.push(nuevo);
      return json(nuevo, 201);
    }
    if (/^\/productos\/[^/]+$/.test(path) && method === 'GET') {
      const p = productos.find((x) => x.id === id)!;
      return json({ ...p, ...enSucursal(p, query['sucursal_id']) });
    }
    if (/^\/productos\/[^/]+$/.test(path) && method === 'PATCH') {
      return json(
        Object.assign(
          productos.find((x) => x.id === id)!,
          body,
        ),
      );
    }
    if (path.startsWith('/productos/') && path.endsWith('/baja')) {
      const p = productos.find((x) => x.id === id)!;
      Object.assign(p, { es_activo: false, motivo_eliminacion: body.motivo });
      return json(p);
    }
    if (path.startsWith('/productos/') && path.endsWith('/restauracion')) {
      const p = productos.find((x) => x.id === id)!;
      Object.assign(p, { es_activo: true, motivo_eliminacion: null });
      return json(p);
    }
    if (path.startsWith('/productos/') && path.endsWith('/disponibilidad') && method === 'PUT') {
      const { sucursal_id, disponible, precio_venta, estacion_id } = body;
      const fila = (sedes[sucursal_id] ??= {})[id] ?? {
        disponible,
        precio_sucursal: null,
        estacion_id: null,
      };
      fila.disponible = disponible;
      if (precio_venta !== undefined) fila.precio_sucursal = precio_venta;
      if (estacion_id !== undefined) fila.estacion_id = estacion_id;
      sedes[sucursal_id][id] = fila;
      return json({
        id: String(++seq),
        producto_id: id,
        sucursal_id,
        estacion_id: fila.estacion_id,
        precio_venta: fila.precio_sucursal,
        disponible,
        es_activo: true,
      });
    }

    // --- Estaciones ---
    if (path === '/estaciones' && method === 'GET') {
      return lista(
        estaciones.filter(
          (e) =>
            (!query['sucursal_id'] || e.sucursal_id === query['sucursal_id']) &&
            (query['es_activo'] === undefined || String(e.es_activo) === query['es_activo']),
        ),
      );
    }
    if (path === '/estaciones' && method === 'POST') {
      if (estaciones.some((e) => e.sucursal_id === body.sucursal_id && e.codigo === body.codigo)) {
        return route.fulfill(
          problem(409, 'codigo_duplicado', 'Ya existe una estación con ese código en la sede.'),
        );
      }
      const nueva = { id: String(++seq), es_activo: true, ...body };
      estaciones.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/estaciones/') && method === 'PATCH') {
      return json(
        Object.assign(
          estaciones.find((e) => e.id === id)!,
          body,
        ),
      );
    }
    if (path.startsWith('/estaciones/') && path.endsWith('/baja')) {
      const e = estaciones.find((x) => x.id === id)!;
      e.es_activo = false;
      return json(e);
    }
    if (path.startsWith('/estaciones/') && path.endsWith('/restauracion')) {
      const e = estaciones.find((x) => x.id === id)!;
      e.es_activo = true;
      return json(e);
    }

    // --- Impuestos y tasas ---
    if (path === '/impuestos' && method === 'GET') return lista(impuestos);
    if (path === '/tasas-impuesto' && method === 'GET') {
      return lista(tasas.filter((t) => t.impuesto_id === query['impuesto_id']));
    }
    const cruza = (desde: string, hasta: string | null, excepto?: string) =>
      tasas.some(
        (t) =>
          t.id !== excepto &&
          desde < (t.vigente_hasta ?? '9999-12-31') &&
          t.vigente_desde < (hasta ?? '9999-12-31'),
      );
    if (path === '/tasas-impuesto' && method === 'POST') {
      if (cruza(body.vigente_desde, body.vigente_hasta ?? null)) {
        return route.fulfill(
          problem(409, 'vigencia_solapada', 'La vigencia se cruza con otra tasa del impuesto.'),
        );
      }
      const nueva = { id: String(++seq), vigente_hasta: null, fundamento: null, ...body };
      tasas.push(nueva);
      return json(nueva, 201);
    }
    if (path.startsWith('/tasas-impuesto/') && method === 'PATCH') {
      const t = tasas.find((x) => x.id === id)!;
      const desde = body.vigente_desde ?? t.vigente_desde;
      const hasta = 'vigente_hasta' in body ? body.vigente_hasta : t.vigente_hasta;
      if (cruza(desde, hasta, t.id)) {
        return route.fulfill(
          problem(409, 'vigencia_solapada', 'La vigencia se cruza con otra tasa del impuesto.'),
        );
      }
      return json(Object.assign(t, body));
    }

    return route.fulfill(problem(404, 'no_encontrado', 'Ruta no simulada: ' + clave));
  });
  return mock;
}
