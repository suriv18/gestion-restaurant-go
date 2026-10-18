import { test, expect } from '@playwright/test';

/**
 * Integración REAL con el backend (sin simulación). Solo corre si se indican las credenciales de
 * un usuario existente y el front tiene el proxy al API:
 *
 *   docker compose up -d --build            # en service-restaurant (API + PostgreSQL)
 *   npm start                                # front en :4200 con proxy a /api
 *   E2E_API_USER=admin E2E_API_PASSWORD=... npx playwright test login.real-api
 */
const user = process.env['E2E_API_USER'];
const pass = process.env['E2E_API_PASSWORD'];

test.skip(!user || !pass, 'Defina E2E_API_USER y E2E_API_PASSWORD para probar contra el API real');

test.describe.configure({ mode: 'serial' });

test('login real: entra, muestra la persona y permisos del servidor y cierra revocando el token', async ({
  page,
  request,
}) => {
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill(user!);
  await page.getByLabel('Contraseña', { exact: true }).fill(pass!);
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/\/dashboard$/);

  const token = await page.evaluate(() => sessionStorage.getItem('app-restaurant.session'));
  expect(token).toMatch(/^[\w-]+\.[\w-]+\.[\w-]+$/);
  const authorized = await request.get('/api/v1/sesion', {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(authorized.status()).toBe(200);

  await page.getByRole('button', { name: 'Menú de usuario' }).click();
  await expect(page.getByRole('menu', { name: 'Cuenta de usuario' })).toContainText(user!);
  await page.getByRole('menuitem', { name: 'Mi perfil' }).click();
  await expect(page.getByRole('dialog', { name: 'Mi perfil' })).toContainText(
    'Autenticada en el servidor',
  );
  await page.keyboard.press('Escape');

  await page.getByRole('button', { name: 'Menú de usuario' }).click();
  await page.getByRole('menuitem', { name: 'Cerrar sesión' }).click();
  await expect(page).toHaveURL(/\/login$/);
  // El servidor revocó la sesión: el mismo token ya no sirve.
  const revoked = await request.get('/api/v1/sesion', {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(revoked.status()).toBe(401);
});

test('login real: una contraseña incorrecta no entra ni deja sesión', async ({ page }) => {
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill(user!);
  await page.getByLabel('Contraseña', { exact: true }).fill(`${pass}-incorrecta`);
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Usuario o contraseña incorrectos');
  await expect(page).toHaveURL(/\/login$/);
  expect(await page.evaluate(() => sessionStorage.getItem('app-restaurant.session'))).toBeNull();
});
