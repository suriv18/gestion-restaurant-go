import { test, expect, Page } from '@playwright/test';
import { enterDemo, mockAuthApi, ApiMock } from './support';
import { mockTeamApi, TeamMock } from './team-support';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

interface World {
  auth: ApiMock;
  team: TeamMock;
}

/** Inicia sesión con la interfaz y abre «Equipo y accesos». */
async function openTeam(page: Page, setup?: (w: World) => void): Promise<World> {
  const world: World = { auth: await mockAuthApi(page), team: await mockTeamApi(page) };
  setup?.(world);
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill('ana');
  await page.getByLabel('Contraseña', { exact: true }).fill('clave-de-prueba');
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.goto('/usuarios');
  return world;
}

const card = (page: Page, username: string) =>
  page.locator('article.user-card').filter({ has: page.getByRole('heading', { name: username }) });
const dialog = (page: Page) => page.getByRole('dialog');

test('muestra el equipo del servidor con sus roles, sedes y estado', async ({ page }) => {
  await openTeam(page);
  await expect(card(page, 'ana')).toContainText('Tu sesión');
  await expect(card(page, 'ana')).toContainText('Administrador · Local Principal');
  await expect(card(page, 'luis')).toContainText('Mesero · Local Principal');
  await expect(card(page, 'luis')).toContainText('luis@example.com');
  await expect(card(page, 'luis')).toContainText('Activo');
  await expect(card(page, 'marta')).toContainText('De baja');
  await expect(card(page, 'marta')).toContainText('Motivo de la baja: fin de contrato');
  await expect(card(page, 'marta').getByRole('button', { name: 'Restaurar' })).toBeVisible();
  // No es la demostración: ni personas de ejemplo ni el aviso de configuración ficticia.
  await expect(page.getByText('Valeria')).toHaveCount(0);
  await expect(page.getByText(/Configuración de ejemplo/)).toHaveCount(0);
  // Una persona no puede darse de baja ni cambiarse el rol a sí misma desde aquí.
  await expect(card(page, 'ana').getByRole('button', { name: 'Dar de baja' })).toHaveCount(0);
});

test('crea una persona: alta con clave de idempotencia y asignación de rol y sede', async ({
  page,
}) => {
  const { team } = await openTeam(page);
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await expect(dialog(page)).toContainText('Nueva persona del equipo');
  await expect(dialog(page).locator('#field-password')).toHaveAttribute('type', 'password');
  await dialog(page).locator('#field-username').fill('  julia  ');
  await dialog(page).locator('#field-email').fill('julia@example.com');
  await dialog(page).locator('#field-password').fill('ClaveSegura2026');
  await dialog(page).locator('#field-role').selectOption({ label: 'Cocina' });
  await dialog(page).getByRole('button', { name: 'Crear acceso' }).click();

  await expect(dialog(page)).toHaveCount(0);
  await expect(card(page, 'julia')).toContainText('Cocina · Local Principal');
  await expect(page.getByRole('status')).toContainText('Se creó el acceso de julia');
  const post = team.calls.find((c) => c.method === 'POST' && c.path === '/usuarios')!;
  expect(post.idempotencyKey).toMatch(UUID);
  expect(post.body).toEqual({
    username: 'julia',
    email: 'julia@example.com',
    password: 'ClaveSegura2026',
  });
  const put = team.calls.find((c) => c.method === 'PUT')!;
  expect(put.body).toEqual({ rol_ids: ['4'], sucursal_ids: ['1'] });
  // Solo se ofrecen roles activos.
  // (El rol archivado del servidor no aparece entre las opciones.)
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await expect(dialog(page).locator('#field-role option')).toHaveText([
    'Administrador',
    'Mesero',
    'Cocina',
  ]);
  await page.keyboard.press('Escape');
  // La contraseña nunca queda en el almacenamiento del navegador.
  const stored = await page.evaluate(() => JSON.stringify({ ...localStorage, ...sessionStorage }));
  expect(stored).not.toContain('ClaveSegura2026');
});

test('con permiso de empleados, el alta guarda también los datos personales y muestra el nombre real', async ({
  page,
}) => {
  const { team } = await openTeam(page, ({ auth }) => {
    auth.session = {
      ...auth.session,
      permisos: [...auth.session.permisos, 'EMPLEADO_ADMINISTRAR'],
    };
  });
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await dialog(page).locator('#field-nombres').fill(' Julia ');
  await dialog(page).locator('#field-apellidos').fill('Paredes Soto');
  await dialog(page).locator('#field-numero_documento').fill('40000003');
  await dialog(page).locator('#field-cargo').fill('Cocinera');
  await dialog(page).locator('#field-username').fill('julia');
  await dialog(page).locator('#field-password').fill('ClaveSegura2026');
  await dialog(page).getByRole('button', { name: 'Crear acceso' }).click();

  await expect(dialog(page)).toHaveCount(0);
  // La tarjeta muestra el nombre real, con el cargo y el usuario debajo.
  await expect(card(page, 'Julia Paredes Soto')).toContainText('Cocinera · julia');
  const post = team.calls.find((c) => c.method === 'POST' && c.path === '/usuarios')!;
  expect(post.body).toEqual({
    username: 'julia',
    password: 'ClaveSegura2026',
    empleado: {
      nombres: 'Julia',
      apellidos: 'Paredes Soto',
      tipo_documento: 'DNI',
      numero_documento: '40000003',
      cargo: 'Cocinera',
    },
  });
});

test('sin permiso de empleados el formulario no pide datos personales', async ({ page }) => {
  await openTeam(page);
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await expect(dialog(page).locator('#field-nombres')).toHaveCount(0);
  await expect(dialog(page).locator('#field-username')).toBeVisible();
});

test('un rechazo del servidor aparece dentro del diálogo, conserva los datos y se puede corregir', async ({
  page,
}) => {
  const { team } = await openTeam(page);
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await dialog(page).locator('#field-username').fill('LUIS');
  await dialog(page).locator('#field-password').fill('ClaveSegura2026');
  await dialog(page).getByRole('button', { name: 'Crear acceso' }).click();
  await expect(dialog(page).getByRole('alert')).toContainText('ya existe un registro');
  await expect(dialog(page)).toBeVisible();
  await expect(dialog(page).locator('#field-username')).toHaveValue('LUIS');
  await expect(dialog(page).locator('#field-password')).toHaveValue('ClaveSegura2026');
  // Corrige y reintenta con el mismo formulario.
  await dialog(page).locator('#field-username').fill('pedro');
  await dialog(page).getByRole('button', { name: 'Crear acceso' }).click();
  await expect(dialog(page)).toHaveCount(0);
  await expect(card(page, 'pedro')).toBeVisible();
  // Los dos intentos comparten la clave del formulario abierto.
  const keys = team.calls.filter((c) => c.method === 'POST').map((c) => c.idempotencyKey);
  expect(keys).toHaveLength(2);
  expect(keys[0]).toBe(keys[1]);
});

test('si el alta se creó pero la asignación falla, lo explica y permite reintentar con la misma clave', async ({
  page,
}) => {
  const { team } = await openTeam(page, ({ team }) =>
    team.rechazos.push({
      method: 'PUT',
      path: /asignaciones$/,
      status: 403,
      code: 'sede_no_autorizada',
      detail: 'solo puede agregar o quitar sucursales donde usted administra accesos',
    }),
  );
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await dialog(page).locator('#field-username').fill('rosa');
  await dialog(page).locator('#field-password').fill('ClaveSegura2026');
  await dialog(page).getByRole('button', { name: 'Crear acceso' }).click();
  await expect(dialog(page).getByRole('alert')).toContainText('La persona se creó, pero falta');
  await expect(dialog(page).getByRole('alert')).toContainText('Puedes reintentar');
  // La persona ya aparece en la lista (sin acceso asignado).
  // (Con el diálogo modal abierto el fondo sale del árbol de accesibilidad: se busca por texto.)
  await expect(page.locator('article.user-card', { hasText: 'rosa' })).toContainText('Sin rol');
  expect(team.calls.filter((c) => c.method === 'POST')).toHaveLength(1);
});

test('cambia el rol conservando la sucursal', async ({ page }) => {
  const { team } = await openTeam(page);
  await card(page, 'luis').getByRole('button', { name: 'Cambiar rol' }).click();
  await expect(dialog(page)).toContainText('Cambiar rol de luis');
  await dialog(page).locator('#field-role').selectOption({ label: 'Cocina' });
  await dialog(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialog(page)).toHaveCount(0);
  await expect(card(page, 'luis')).toContainText('Cocina · Local Principal');
  const put = team.calls.find((c) => c.method === 'PUT')!;
  expect(put.path).toBe('/usuarios/3/asignaciones');
  expect(put.body).toEqual({ rol_ids: ['4'], sucursal_ids: ['1'] });
});

test('una regla del servidor (último administrador) se muestra sin cerrar el diálogo', async ({
  page,
}) => {
  await openTeam(page, ({ team }) =>
    team.rechazos.push({
      method: 'PUT',
      path: /asignaciones$/,
      status: 422,
      code: 'ultimo_administrador',
      detail: 'la operación dejaría al sistema sin ningún administrador activo',
    }),
  );
  await card(page, 'luis').getByRole('button', { name: 'Cambiar rol' }).click();
  await dialog(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialog(page).getByRole('alert')).toContainText('sin ningún administrador activo');
  await expect(dialog(page)).toBeVisible();
});

test('da de baja con motivo obligatorio y restaura', async ({ page }) => {
  const { team } = await openTeam(page);
  await card(page, 'luis').getByRole('button', { name: 'Dar de baja' }).click();
  await expect(dialog(page)).toContainText('Se cierran sus sesiones');
  await dialog(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(dialog(page).getByRole('alert')).toContainText('Completa los campos');
  expect(team.calls.some((c) => c.path.endsWith('/baja'))).toBe(false);
  await dialog(page).locator('#field-reason').fill('  cambio de turno ');
  await dialog(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(dialog(page)).toHaveCount(0);
  await expect(card(page, 'luis')).toContainText('De baja');
  await expect(card(page, 'luis')).toContainText('Motivo de la baja: cambio de turno');
  await expect(page.getByRole('status')).toContainText('sus sesiones se cerraron');
  const baja = team.calls.find((c) => c.path === '/usuarios/3/baja')!;
  expect(baja.body).toEqual({ motivo: 'cambio de turno' });
  expect(baja.idempotencyKey).toMatch(UUID);

  await card(page, 'luis').getByRole('button', { name: 'Restaurar' }).click();
  await expect(card(page, 'luis')).toContainText('Activo');
  await expect(page.getByRole('status')).toContainText('Deberá iniciar sesión de nuevo');
  expect(team.calls.find((c) => c.path === '/usuarios/3/restauracion')!.idempotencyKey).toMatch(
    UUID,
  );
});

test('sin permiso de administración no consulta ni ofrece acciones', async ({ page }) => {
  const { team } = await openTeam(page, ({ auth }) => {
    auth.session = { ...auth.session, permisos: ['PEDIDO_CREAR'] };
  });
  await expect(page.getByRole('status').filter({ hasText: 'No tienes permiso' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Nueva persona' })).toHaveCount(0);
  await expect(page.locator('article.user-card')).toHaveCount(0);
  expect(team.calls).toEqual([]);
});

test('si la carga falla informa sin detalles internos y permite reintentar', async ({ page }) => {
  const { team } = await openTeam(page, ({ team }) => (team.listFails = true));
  await expect(page.getByRole('alert')).toContainText('No pudimos cargar el equipo');
  await expect(page.getByText('pq:')).toHaveCount(0);
  team.listFails = false;
  await page.getByRole('button', { name: 'Reintentar' }).click();
  await expect(card(page, 'luis')).toBeVisible();
  await expect(page.getByRole('alert')).toHaveCount(0);
});

test('en modo demostración la pantalla sigue usando los datos de ejemplo', async ({ page }) => {
  await enterDemo(page);
  await page.goto('/usuarios');
  await expect(page.getByText(/Configuración de ejemplo/)).toBeVisible();
  await expect(page.locator('article.user-card').first()).toContainText('Valeria');
  await expect(page.getByRole('button', { name: 'Nueva persona' })).toHaveCount(0);
});
