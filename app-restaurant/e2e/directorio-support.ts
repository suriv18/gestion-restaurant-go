import type { Page, Route } from '@playwright/test';

export interface Llamada {
  method: string;
  path: string;
  body: Record<string, unknown> | null;
  idempotencyKey: string | undefined;
}

export interface DirectorioMock {
  clientes: Registro[];
  proveedores: Registro[];
  empleados: Registro[];
  llamadas: Llamada[];
  /** Próximo rechazo programado para `METHOD /ruta` (una sola vez). */
  rechazos: Map<string, { status: number; code: string; detail: string }>;
}

export type Registro = Record<string, unknown> & {
  id: string;
  es_activo: boolean;
  motivo_eliminacion: string | null;
};

const base = { es_activo: true, motivo_eliminacion: null };

const cliente = (id: string, nombre: string, doc: [string, string] | null): Registro => ({
  id,
  tipo_documento: doc?.[0] ?? null,
  numero_documento: doc?.[1] ?? null,
  nombre_razon_social: nombre,
  telefono: null,
  email: null,
  direccion: null,
  ...base,
});

const proveedor = (id: string, razon: string, ruc: string): Registro => ({
  id,
  tipo_documento: 'RUC',
  numero_documento: ruc,
  razon_social: razon,
  contacto_nombre: null,
  telefono: null,
  email: null,
  direccion: null,
  ...base,
});

const empleado = (id: string, nombres: string, apellidos: string, dni: string): Registro => ({
  id,
  usuario_id: null,
  username: null,
  tipo_documento: 'DNI',
  numero_documento: dni,
  nombres,
  apellidos,
  telefono: null,
  cargo: null,
  fecha_ingreso: '2026-03-15',
  fecha_cese: null,
  ...base,
});

const json = (route: Route, status: number, body: unknown) =>
  route.fulfill({
    status,
    contentType: status >= 400 ? 'application/problem+json' : 'application/json',
    body: JSON.stringify(body),
  });

/**
 * Simula `/clientes`, `/proveedores`, `/empleados` y `/usuarios` con estado en memoria, con las
 * mismas formas y códigos que el backend. La integración real se comprueba aparte.
 */
export async function mockDirectorioApi(page: Page): Promise<DirectorioMock> {
  const mock: DirectorioMock = {
    clientes: [
      cliente('1', 'Ana Ríos', ['DNI', '12345678']),
      cliente('2', 'Mesa Doble SAC', ['RUC', '20111111111']),
      { ...cliente('3', 'Luis Antiguo', null), es_activo: false, motivo_eliminacion: 'se mudó' },
    ],
    proveedores: [proveedor('7', 'Distribuidora Andina SAC', '20123456789')],
    empleados: [empleado('11', 'María', 'Quispe', '40000001')],
    llamadas: [],
    rechazos: new Map(),
  };
  let seq = 100;

  const recurso = async (
    nombre: 'clientes' | 'proveedores' | 'empleados',
    nuevo: (body: Record<string, unknown>, id: string) => Registro,
    repetido: (body: Record<string, unknown>, existente: Registro) => boolean,
  ) => {
    await page.route(`**/api/v1/${nombre}**`, async (route) => {
      const req = route.request();
      const path = new URL(req.url()).pathname.replace('/api/v1', '');
      const method = req.method();
      let body: Record<string, unknown> | null = null;
      try {
        body = req.postDataJSON();
      } catch {
        body = null;
      }
      mock.llamadas.push({ method, path, body, idempotencyKey: req.headers()['idempotency-key'] });
      const rechazo = mock.rechazos.get(`${method} ${path}`);
      if (rechazo) {
        mock.rechazos.delete(`${method} ${path}`);
        return json(route, rechazo.status, { type: 'about:blank', ...rechazo });
      }
      const items = mock[nombre];
      if (method === 'GET' && path === `/${nombre}`)
        return json(route, 200, { items, nextCursor: null });
      if (method === 'POST' && path === `/${nombre}`) {
        if (items.some((x) => repetido(body ?? {}, x))) {
          return json(route, 409, {
            type: 'about:blank',
            status: 409,
            code: 'documento_duplicado',
            detail: 'ya existe un registro con ese documento (aunque esté dado de baja)',
          });
        }
        const creado = nuevo(body ?? {}, String(seq++));
        items.push(creado);
        return json(route, 201, creado);
      }
      const m = new RegExp(`^/${nombre}/(\\d+)(?:/(baja|restauracion))?$`).exec(path);
      const objetivo = items.find((x) => x.id === m?.[1]);
      if (!m || !objetivo) {
        return json(route, 404, {
          type: 'about:blank',
          status: 404,
          code: 'no_encontrado',
          detail: 'no encontrado',
        });
      }
      if (m[2] === 'baja') {
        objetivo.es_activo = false;
        objetivo.motivo_eliminacion = String(body?.['motivo']);
      } else if (m[2] === 'restauracion') {
        objetivo.es_activo = true;
        objetivo.motivo_eliminacion = null;
      } else {
        Object.assign(objetivo, body);
      }
      return json(route, 200, objetivo);
    });
  };

  const mismoDocumento = (b: Record<string, unknown>, x: Registro) =>
    b['numero_documento'] != null &&
    b['tipo_documento'] === x['tipo_documento'] &&
    b['numero_documento'] === x['numero_documento'];

  await recurso(
    'clientes',
    (b, id) => ({ ...cliente(id, String(b['nombre_razon_social']), null), ...b, id }) as Registro,
    mismoDocumento,
  );
  await recurso(
    'proveedores',
    (b, id) => ({ ...proveedor(id, '', ''), ...b, id }) as Registro,
    mismoDocumento,
  );
  await recurso(
    'empleados',
    (b, id) =>
      ({
        ...empleado(id, '', '', ''),
        ...b,
        id,
        fecha_ingreso: String(b['fecha_ingreso'] ?? '2026-10-24'),
      }) as Registro,
    mismoDocumento,
  );

  // Equipo: lo mínimo para ofrecer usuarios sin empleado al editar un empleado.
  await page.route('**/api/v1/usuarios**', (route) =>
    json(route, 200, {
      items: [usuario('2', 'ana', false), usuario('3', 'luis', true), usuario('4', 'marta', false)],
      nextCursor: null,
    }),
  );
  return mock;
}

const usuario = (id: string, username: string, conEmpleado: boolean) => ({
  id,
  username,
  email: null,
  es_activo: true,
  ultimo_acceso_en: null,
  bloqueado_hasta: null,
  motivo_baja: null,
  empleado: conEmpleado ? { id: '99', nombres: 'Luis', apellidos: 'Soto', cargo: null } : null,
  roles: [],
  sucursales: [],
});
