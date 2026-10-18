import { test, expect, Page } from '@playwright/test';
import { mockAuthApi, TOKEN } from './support';

const submit = (page: Page) => page.getByRole('button', { name: /Iniciar sesión|Ingresando/ });
const username = (page: Page) => page.getByRole('textbox', { name: 'Usuario' });
const password = (page: Page) => page.getByLabel('Contraseña', { exact: true });

async function fill(page: Page, user: string, pass: string) {
  await username(page).fill(user);
  await password(page).fill(pass);
}

test('el acceso se adapta y permite entrar y volver de la demo', async ({ page }, testInfo) => {
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.goto('/');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Qué bueno verte de nuevo.');
  await expect(page.getByRole('navigation', { name: 'Menú principal' })).toHaveCount(0);
  await page.evaluate(() => document.fonts.ready);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.screenshot({ path: testInfo.outputPath('login.png'), fullPage: true });
  await page.getByRole('link', { name: /Explorar la demo/ }).click();
  await expect(page).toHaveURL(/\/salon$/);
  await expect(page.locator('main h1')).toBeVisible();
  await page.getByRole('button', { name: 'Menú de usuario' }).click();
  await page.getByRole('menuitem', { name: 'Cerrar sesión' }).click();
  await expect(page).toHaveURL(/\/login$/);
  // Salir de la demo la desactiva: las pantallas vuelven a exigir cuenta.
  await page.goto('/salon');
  await expect(page).toHaveURL(/\/login\?returnUrl=%2Fsalon$/);
  expect(errors).toEqual([]);
});

test('sin sesión ni demo las pantallas de operación redirigen al login', async ({ page }) => {
  for (const route of ['/salon', '/caja', '/usuarios']) {
    await page.goto(route);
    await expect(page).toHaveURL(new RegExp(`/login\\?returnUrl=${encodeURIComponent(route)}$`));
    await expect(page.getByRole('navigation', { name: 'Menú principal' })).toHaveCount(0);
  }
});

test('valida y enfoca el primer error antes de llamar al servidor', async ({ page }) => {
  const api = await mockAuthApi(page);
  await page.goto('/login');
  await submit(page).click();
  await expect(username(page)).toBeFocused();
  await expect(username(page)).toHaveAttribute('aria-invalid', 'true');
  await expect(page.getByText('Ingresa tu usuario.', { exact: true })).toBeVisible();
  await expect(page.getByText('Ingresa tu contraseña.', { exact: true })).toBeVisible();
  await username(page).fill('ana');
  await page.getByRole('button', { name: 'Mostrar contraseña', exact: true }).click();
  await expect(password(page)).toHaveAttribute('type', 'text');
  await page.getByRole('button', { name: 'Ocultar contraseña', exact: true }).click();
  await expect(password(page)).toHaveAttribute('type', 'password');
  await submit(page).click();
  await expect(password(page)).toBeFocused();
  expect(api.logins).toEqual([]);
});

test('inicia sesión, entra al resumen y envía el token solo en llamadas autenticadas', async ({
  page,
}) => {
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  const api = await mockAuthApi(page);
  await page.goto('/login');
  await fill(page, '  ana  ', 'clave-de-prueba');
  await password(page).press('Enter');
  await expect(page).toHaveURL(/\/dashboard$/);
  // En móvil el menú lateral está oculto hasta abrirlo: se comprueba la pantalla de destino.
  await expect(page.locator('main h1')).toBeVisible();
  expect(api.logins).toEqual([{ username: 'ana', password: 'clave-de-prueba' }]);
  expect(api.sessionAuth).toEqual([`Bearer ${TOKEN}`]);
  // El token vive en sessionStorage; la contraseña no se guarda en ningún almacenamiento.
  expect(await page.evaluate(() => sessionStorage.getItem('app-restaurant.session'))).toBe(TOKEN);
  const stored = await page.evaluate(() => JSON.stringify({ ...localStorage, ...sessionStorage }));
  expect(stored).not.toContain('clave-de-prueba');
  // El menú muestra la persona real y no la de demostración.
  await page.getByRole('button', { name: 'Menú de usuario' }).click();
  await expect(page.getByRole('menu', { name: 'Cuenta de usuario' })).toContainText('ana');
  await expect(page.getByRole('menu', { name: 'Cuenta de usuario' })).not.toContainText('Valeria');
  await page.getByRole('menuitem', { name: 'Mi perfil' }).click();
  await expect(page.getByRole('dialog', { name: 'Mi perfil' })).toContainText(
    'USUARIO_ADMINISTRAR',
  );
  await page.keyboard.press('Escape');
  expect(errors).toEqual([]);
});

test('vuelve al destino pedido tras iniciar sesión y descarta destinos externos', async ({
  page,
}) => {
  await mockAuthApi(page);
  await page.goto('/caja');
  await expect(page).toHaveURL(/\/login\?returnUrl=%2Fcaja$/);
  await fill(page, 'ana', 'x');
  await submit(page).click();
  await expect(page).toHaveURL(/\/caja$/);

  await page.evaluate(() => sessionStorage.clear());
  await page.goto('/login?returnUrl=//evil.example.com');
  await fill(page, 'ana', 'x');
  await submit(page).click();
  await expect(page).toHaveURL(/\/dashboard$/);
});

test('credenciales incorrectas: avisa sin revelar el motivo, vacía la contraseña y no crea sesión', async ({
  page,
}) => {
  const api = await mockAuthApi(page);
  api.loginMode = 401;
  await page.goto('/login');
  await fill(page, 'ana', 'incorrecta');
  await submit(page).click();
  await expect(page.getByRole('alert')).toContainText('Usuario o contraseña incorrectos');
  await expect(password(page)).toHaveValue('');
  await expect(password(page)).toBeFocused();
  await expect(username(page)).toHaveValue('ana');
  await expect(page).toHaveURL(/\/login$/);
  expect(await page.evaluate(() => sessionStorage.getItem('app-restaurant.session'))).toBeNull();
  expect(api.sessionAuth).toEqual([]);
  // Reintentar con credenciales válidas funciona.
  api.loginMode = 'ok';
  await password(page).fill('correcta');
  await submit(page).click();
  await expect(page).toHaveURL(/\/dashboard$/);
});

test('distingue la falta de conexión y un fallo del servicio', async ({ page }) => {
  const api = await mockAuthApi(page);
  await page.goto('/login');
  api.loginMode = 'network';
  await fill(page, 'ana', 'x');
  await submit(page).click();
  await expect(page.getByRole('alert')).toContainText('No pudimos conectar');
  api.loginMode = 500;
  await password(page).fill('x');
  await submit(page).click();
  await expect(page.getByRole('alert')).toContainText('no está disponible');
  await expect(page).toHaveURL(/\/login$/);
});

test('cerrar sesión revoca en el servidor y las pantallas vuelven a exigir cuenta', async ({
  page,
}) => {
  const api = await mockAuthApi(page);
  await page.goto('/login');
  await fill(page, 'ana', 'x');
  await submit(page).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.getByRole('button', { name: 'Menú de usuario' }).click();
  await page.getByRole('menuitem', { name: 'Cerrar sesión' }).click();
  await expect(page).toHaveURL(/\/login$/);
  expect(api.logouts).toBe(1);
  expect(await page.evaluate(() => sessionStorage.getItem('app-restaurant.session'))).toBeNull();
  await page.goto('/dashboard');
  await expect(page).toHaveURL(/\/login\?returnUrl=%2Fdashboard$/);
});

test('una recarga conserva la sesión validándola; si fue revocada, vuelve al login', async ({
  page,
}) => {
  const api = await mockAuthApi(page);
  await page.goto('/login');
  await fill(page, 'ana', 'x');
  await submit(page).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.reload();
  await expect(page).toHaveURL(/\/dashboard$/);
  expect(api.sessionAuth.length).toBeGreaterThanOrEqual(2);
  // El servidor revoca la sesión (baja del usuario, vencimiento...): la recarga ya no entra.
  api.sessionRevoked = true;
  await page.reload();
  await expect(page).toHaveURL(/\/login\?returnUrl=%2Fdashboard$/);
  expect(await page.evaluate(() => sessionStorage.getItem('app-restaurant.session'))).toBeNull();
});

test('recuperación y ayuda funcionan con teclado y restauran el foco', async ({ page }) => {
  await page.goto('/login');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  await page.keyboard.press('Tab');
  await expect(page.getByRole('link', { name: 'Ir al acceso' })).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.locator('#login-main')).toBeFocused();
  const recovery = page.getByRole('button', { name: '¿La olvidaste?' });
  await recovery.click();
  await expect(page.getByRole('dialog')).toContainText('No se ha enviado ningún correo');
  await page.keyboard.press('Escape');
  await expect(recovery).toBeFocused();
  await page.getByRole('button', { name: '¿Necesitas ayuda?' }).click();
  await expect(page.getByRole('dialog')).toContainText('Selecciona «Explorar la demo»');
  await page.keyboard.press('Escape');
  await expect(page.getByRole('button', { name: '¿Necesitas ayuda?' })).toBeFocused();
});
