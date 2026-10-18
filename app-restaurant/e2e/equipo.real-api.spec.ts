import { test, expect, Page } from '@playwright/test';

/**
 * Integración REAL de «Equipo y accesos» con el backend (sin simulación). Opt-in: requiere el
 * stack de Docker, el front con el proxy y un administrador existente. Cada ejecución crea una
 * persona nueva (las personas no se borran: solo se dan de baja).
 *
 *   E2E_API_USER=admin E2E_API_PASSWORD=... npx playwright test equipo.real-api
 */
const admin = process.env['E2E_API_USER'];
const adminPass = process.env['E2E_API_PASSWORD'];

test.skip(
  !admin || !adminPass,
  'Defina E2E_API_USER y E2E_API_PASSWORD para probar contra el API real',
);
test.describe.configure({ mode: 'serial' });

const card = (page: Page, username: string) =>
  page.locator('article.user-card').filter({ has: page.getByRole('heading', { name: username }) });
const dialog = (page: Page) => page.getByRole('dialog');

test('administra una persona de extremo a extremo contra el servidor real', async ({
  page,
  request,
}) => {
  const username = `e2e.${Date.now()}`;
  const password = 'E2eClaveSegura2026';

  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill(admin!);
  await page.getByLabel('Contraseña', { exact: true }).fill(adminPass!);
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.goto('/usuarios');
  await expect(card(page, admin!)).toContainText('Tu sesión');
  await expect(card(page, admin!)).toContainText('Administrador');

  // Alta.
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await dialog(page).locator('#field-username').fill(username);
  await dialog(page).locator('#field-password').fill(password);
  await dialog(page).locator('#field-role').selectOption({ label: 'Cajero' });
  await dialog(page).getByRole('button', { name: 'Crear acceso' }).click();
  await expect(dialog(page)).toHaveCount(0);
  await expect(card(page, username)).toContainText('Cajero');
  await expect(card(page, username)).toContainText('Activo');

  // La nueva persona ya puede entrar (lo comprueba el servidor real).
  const login = () => request.post('/api/v1/auth/login', { data: { username, password } });
  expect((await login()).status()).toBe(200);

  // Un usuario repetido se rechaza dentro del diálogo, con el mensaje del servidor.
  await page.getByRole('button', { name: 'Nueva persona' }).click();
  await dialog(page).locator('#field-username').fill(username.toUpperCase());
  await dialog(page).locator('#field-password').fill(password);
  await dialog(page).getByRole('button', { name: 'Crear acceso' }).click();
  await expect(dialog(page).getByRole('alert')).toContainText('ya existe');
  await page.keyboard.press('Escape');

  // Cambio de rol.
  await card(page, username).getByRole('button', { name: 'Cambiar rol' }).click();
  await dialog(page).locator('#field-role').selectOption({ label: 'Cocina' });
  await dialog(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(card(page, username)).toContainText('Cocina');

  // Baja: queda registrada con su motivo y la persona deja de poder entrar.
  await card(page, username).getByRole('button', { name: 'Dar de baja' }).click();
  await dialog(page).locator('#field-reason').fill('prueba e2e');
  await dialog(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(card(page, username)).toContainText('De baja');
  await expect(card(page, username)).toContainText('Motivo de la baja: prueba e2e');
  expect((await login()).status()).toBe(401);

  // Restauración: vuelve a poder entrar.
  await card(page, username).getByRole('button', { name: 'Restaurar' }).click();
  await expect(card(page, username)).toContainText('Activo');
  expect((await login()).status()).toBe(200);
});

test('una persona sin permiso de administración no ve el equipo', async ({ page, request }) => {
  // Crea (por API, como administrador) una persona sin permisos de administración.
  const tk = (
    await (
      await request.post('/api/v1/auth/login', { data: { username: admin, password: adminPass } })
    ).json()
  ).access_token as string;
  const headers = { Authorization: `Bearer ${tk}` };
  const username = `e2e.sinpoder.${Date.now()}`;
  const password = 'E2eSinPoder2026Ok';
  const created = await request.post('/api/v1/usuarios', {
    headers: { ...headers, 'Idempotency-Key': crypto.randomUUID() },
    data: { username, password },
  });
  expect(created.status()).toBe(201);
  const { id } = await created.json();
  const roles = await (await request.get('/api/v1/roles?limit=100', { headers })).json();
  const sedes = await (await request.get('/api/v1/sucursales-asignables', { headers })).json();
  const cajero = roles.items.find((r: { codigo: string }) => r.codigo === 'CAJERO').id;
  const assigned = await request.put(`/api/v1/usuarios/${id}/asignaciones`, {
    headers,
    data: { rol_ids: [cajero], sucursal_ids: [sedes.items[0].id] },
  });
  expect(assigned.status()).toBe(200);

  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill(username);
  await page.getByLabel('Contraseña', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.goto('/usuarios');
  await expect(page.getByText('No tienes permiso para administrar el equipo')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Nueva persona' })).toHaveCount(0);
  // Y el servidor lo confirma: aunque la interfaz lo permitiera, respondería 403.
  const loginSinPoder = await (
    await request.post('/api/v1/auth/login', { data: { username, password } })
  ).json();
  const prohibido = await request.get('/api/v1/usuarios', {
    headers: { Authorization: `Bearer ${loginSinPoder.access_token}` },
  });
  expect(prohibido.status()).toBe(403);
});
