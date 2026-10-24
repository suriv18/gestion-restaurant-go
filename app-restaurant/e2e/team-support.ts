import { Page, Route } from '@playwright/test';

interface Rol {
  id: string;
  codigo: string;
  nombre: string;
  es_activo: boolean;
}
interface Sede {
  id: string;
  codigo: string;
  nombre: string;
}
export interface Persona {
  id: string;
  username: string;
  email: string | null;
  es_activo: boolean;
  ultimo_acceso_en: string | null;
  bloqueado_hasta: string | null;
  motivo_baja: string | null;
  empleado: { id: string; nombres: string; apellidos: string; cargo: string | null } | null;
  roles: Rol[];
  sucursales: Sede[];
}
export interface Call {
  method: string;
  path: string;
  body: Record<string, unknown> | null;
  idempotencyKey: string | undefined;
}
export interface Rechazo {
  /** Se aplica a la próxima llamada que coincida con este método y ruta (regex sobre el path). */
  method: string;
  path: RegExp;
  status: number;
  code: string;
  detail: string;
}
export interface TeamMock {
  users: Persona[];
  calls: Call[];
  /** Rechazos programados; cada uno se consume al usarse. */
  rechazos: Rechazo[];
  /** Si es true, GET /usuarios responde 500 hasta que se ponga en false. */
  listFails: boolean;
}

const ADMIN: Rol = { id: '1', codigo: 'ADMIN', nombre: 'Administrador', es_activo: true };
const MESERO: Rol = { id: '3', codigo: 'MESERO', nombre: 'Mesero', es_activo: true };
const COCINA: Rol = { id: '4', codigo: 'COCINA', nombre: 'Cocina', es_activo: true };
const ARCHIVADO: Rol = { id: '9', codigo: 'ARCHIVADO', nombre: 'Rol archivado', es_activo: false };
export const PRINCIPAL: Sede = { id: '1', codigo: 'PRINCIPAL', nombre: 'Local Principal' };

const persona = (id: string, username: string, over: Partial<Persona> = {}): Persona => ({
  id,
  username,
  email: null,
  es_activo: true,
  ultimo_acceso_en: '2026-10-17T20:15:00Z',
  bloqueado_hasta: null,
  motivo_baja: null,
  empleado: null,
  roles: [MESERO],
  sucursales: [PRINCIPAL],
  ...over,
});

/**
 * Simula los endpoints de administración de accesos con un estado en memoria que responde como
 * el backend real (mismos códigos y formas). La integración real se comprueba aparte.
 */
export async function mockTeamApi(page: Page): Promise<TeamMock> {
  const mock: TeamMock = {
    users: [
      persona('2', 'ana', { roles: [ADMIN] }),
      persona('3', 'luis', { email: 'luis@example.com' }),
      persona('4', 'marta', { es_activo: false, motivo_baja: 'fin de contrato' }),
    ],
    calls: [],
    rechazos: [],
    listFails: false,
  };
  let nextId = 10;
  const json = (route: Route, status: number, body: unknown) =>
    route.fulfill({
      status,
      contentType: status >= 400 ? 'application/problem+json' : 'application/json',
      body: JSON.stringify(body),
    });
  const problem = (route: Route, status: number, code: string, detail: string) =>
    json(route, status, { type: 'about:blank', status, code, detail });

  await page.route('**/api/v1/roles**', (route) =>
    json(route, 200, { items: [ADMIN, MESERO, COCINA, ARCHIVADO], nextCursor: null }),
  );
  await page.route('**/api/v1/sucursales-asignables', (route) =>
    json(route, 200, { items: [PRINCIPAL] }),
  );
  await page.route('**/api/v1/usuarios**', async (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname.replace('/api/v1', '');
    const method = req.method();
    let body: Record<string, unknown> | null = null;
    try {
      body = req.postDataJSON();
    } catch {
      body = null;
    }
    mock.calls.push({ method, path, body, idempotencyKey: req.headers()['idempotency-key'] });

    const i = mock.rechazos.findIndex((r) => r.method === method && r.path.test(path));
    if (i >= 0) {
      const [r] = mock.rechazos.splice(i, 1);
      return problem(route, r.status, r.code, r.detail);
    }
    if (method === 'GET' && path === '/usuarios') {
      if (mock.listFails) return problem(route, 500, 'error', 'pq: detalle interno');
      return json(route, 200, { items: mock.users, nextCursor: null });
    }
    if (method === 'POST' && path === '/usuarios') {
      const username = String(body?.['username']);
      if (mock.users.some((u) => u.username.toLowerCase() === username.toLowerCase())) {
        return problem(
          route,
          409,
          'registro_duplicado',
          'ya existe un registro con ese valor único',
        );
      }
      const datos = body?.['empleado'] as
        { nombres: string; apellidos: string; cargo?: string } | undefined;
      const created = persona(String(nextId++), username, {
        empleado: datos
          ? {
              id: `e${nextId}`,
              nombres: datos.nombres,
              apellidos: datos.apellidos,
              cargo: datos.cargo ?? null,
            }
          : null,
        roles: [],
        sucursales: [],
        email: (body?.['email'] as string) ?? null,
        ultimo_acceso_en: null,
      });
      mock.users.push(created);
      return json(route, 201, { id: created.id, username });
    }
    const m = /^\/usuarios\/(\d+)\/(asignaciones|baja|restauracion)$/.exec(path);
    const target = mock.users.find((u) => u.id === m?.[1]);
    if (!m || !target) return problem(route, 404, 'no_encontrado', 'usuario no encontrado');
    if (m[2] === 'asignaciones') {
      const roleIds = (body?.['rol_ids'] as string[]) ?? [];
      const siteIds = (body?.['sucursal_ids'] as string[]) ?? [];
      target.roles = [ADMIN, MESERO, COCINA].filter((r) => roleIds.includes(r.id));
      target.sucursales = [PRINCIPAL].filter((s) => siteIds.includes(s.id));
    } else if (m[2] === 'baja') {
      target.es_activo = false;
      target.motivo_baja = String(body?.['motivo']);
    } else {
      target.es_activo = true;
      target.motivo_baja = null;
    }
    return json(route, 200, { id: target.id, username: target.username });
  });
  return mock;
}
