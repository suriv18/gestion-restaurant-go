import { test, expect, Page } from '@playwright/test';
import { enterDemo } from './support';

// Las pantallas de operación exigen sesión; estas pruebas recorren la demo explícita.
test.beforeEach(({ page }) => enterDemo(page));

async function account(page: Page) {
  await page.getByRole('button', { name: 'Menú de usuario' }).click();
  await expect(page.getByRole('menu', { name: 'Cuenta de usuario' })).toBeVisible();
}

async function selectTheme(page: Page, name: string) {
  await account(page);
  await page.getByRole('menuitemradio', { name, exact: true }).click();
}

test('el perfil del header se abre con teclado y restaura el foco', async ({ page }, testInfo) => {
  await page.goto('/salon');
  const trigger = page.getByRole('button', { name: 'Menú de usuario' });
  await expect(trigger).toBeVisible();
  await trigger.focus();
  await page.keyboard.press('ArrowDown');
  await expect(page.getByRole('menuitem', { name: 'Mi perfil' })).toBeFocused();
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({ path: testInfo.outputPath('menu-usuario.png') });
  await page.keyboard.press('Enter');
  await expect(page.getByRole('dialog', { name: 'Mi perfil' })).toContainText('Valeria Torres');
  await expect(page.getByRole('dialog')).toContainText('Administradora');
  await page.keyboard.press('Escape');
  await expect(trigger).toBeFocused();
  await account(page);
  await page.keyboard.press('Escape');
  await expect(trigger).toBeFocused();
  await account(page);
  await page.locator('.breadcrumb strong').click();
  await expect(page.getByRole('menu', { name: 'Cuenta de usuario' })).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});

test('persiste el tema y adapta las pantallas y diálogos al modo oscuro', async ({
  page,
}, testInfo) => {
  const errors: string[] = [];
  page.on('pageerror', (error) => errors.push(error.message));
  await page.emulateMedia({ colorScheme: 'light' });
  await page.goto('/salon');
  await selectTheme(page, 'Oscuro');
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await account(page);
  await expect(page.getByRole('menuitemradio', { name: 'Oscuro' })).toHaveAttribute(
    'aria-checked',
    'true',
  );
  const bounds = await page.getByRole('group', { name: 'Tema de la aplicación' }).boundingBox();
  expect(bounds?.x).toBeGreaterThanOrEqual(0);
  expect((bounds?.x ?? 0) + (bounds?.width ?? 0)).toBeLessThanOrEqual(page.viewportSize()!.width);
  await page.screenshot({ path: testInfo.outputPath('menu-tema-oscuro.png') });
  await page.keyboard.press('Escape');
  await page.keyboard.press('Escape');
  for (const route of [
    'salon',
    'cocina',
    'caja',
    'carta',
    'dashboard',
    'reservas',
    'delivery',
    'compras',
    'inventario',
    'comprobantes',
    'contactos',
    'usuarios',
    'auditoria',
    'configuracion',
  ]) {
    await page.goto('/' + route);
    await expect(page.locator('main h1')).toBeVisible();
    await expect(page.locator('.topbar')).toHaveCSS('background-color', 'rgb(34, 50, 41)');
    if (route === 'caja') {
      await expect(page.locator('tbody td').first()).toHaveCSS('color', 'rgb(168, 184, 172)');
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
      true,
    );
    if (['salon', 'caja', 'carta'].includes(route)) {
      await page.screenshot({ path: testInfo.outputPath(`oscuro-${route}.png`), fullPage: true });
    }
  }
  await account(page);
  await page.getByRole('menuitem', { name: 'Mi perfil' }).click();
  await expect(page.locator('.modal-header')).toHaveCSS('background-color', 'rgb(30, 44, 36)');
  await page.keyboard.press('Escape');
  await selectTheme(page, 'Claro');
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await selectTheme(page, 'Sistema');
  await page.emulateMedia({ colorScheme: 'dark' });
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.emulateMedia({ colorScheme: 'light' });
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  expect(errors).toEqual([]);
});

test('cerrar sesión vuelve al login y conserva los datos de la demo y el tema', async ({
  page,
}, testInfo) => {
  await page.goto('/salon');
  await selectTheme(page, 'Oscuro');
  const state = await page.evaluate(() => localStorage.getItem('app-restaurant.demo.v1'));
  await account(page);
  await page.getByRole('menuitem', { name: 'Cerrar sesión' }).click();
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('button', { name: 'Menú de usuario' })).toHaveCount(0);
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await expect(page.locator('label[for="login-username"]')).toHaveCSS(
    'color',
    'rgb(228, 236, 229)',
  );
  await expect(page.locator('#login-username')).toHaveCSS('color', 'rgb(228, 236, 229)');
  await expect(page.locator('.demo-link small')).toHaveCSS('color', 'rgb(168, 184, 172)');
  await page.screenshot({ path: testInfo.outputPath('login-oscuro.png'), fullPage: true });
  expect(await page.evaluate(() => localStorage.getItem('app-restaurant.demo.v1'))).toBe(state);
  await page.getByRole('link', { name: /Explorar la demo/ }).click();
  await expect(page).toHaveURL(/\/salon$/);
  await expect(page.getByRole('button', { name: 'Menú de usuario' })).toBeVisible();
});
