import { test, expect, Page } from '@playwright/test';
import { enterDemo } from './support';

// Las pantallas de operación exigen sesión; estas pruebas recorren la demo explícita.
test.beforeEach(({ page }) => enterDemo(page));

test('divide varias líneas en una cuenta y mantiene el saldo total', async ({ page }) => {
  await open(page);
  const panel = page.getByRole('complementary', { name: 'Pedido seleccionado' });
  await panel.getByRole('button', { name: 'Dividir cuenta' }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('checkbox', { name: /Lomo saltado/ }).check();
  await dialog.getByRole('checkbox', { name: /Causa limeña/ }).check();
  await dialogSubmit(page, 'Crear cuenta');
  await panel.getByLabel('Cuenta del pedido').selectOption('2');
  await expect(panel.locator('.order-summary .total')).toContainText('82.00');
  await panel.getByLabel('Cuenta del pedido').selectOption('0');
  await expect(panel.locator('.order-summary .total')).toContainText('106.00');
});

test('crea una compra con varias líneas y revisa su detalle', async ({ page }) => {
  await open(page, '/compras');
  await page.getByRole('button', { name: 'Nueva compra' }).click();
  await page.getByLabel('Referencia de compra').fill('E2E-COMPRA-002');
  await page.getByRole('combobox', { name: 'Insumo 1', exact: true }).selectOption('1');
  await page.getByLabel('Cantidad 1').fill('2');
  await page.getByLabel('Costo unitario 1').fill('28.00');
  await page.getByRole('button', { name: 'Agregar insumo' }).click();
  await page.getByRole('combobox', { name: 'Insumo 2', exact: true }).selectOption('6');
  await page.getByLabel('Cantidad 2').fill('3');
  await page.getByLabel('Costo unitario 2').fill('6.00');
  await dialogSubmit(page, 'Guardar borrador');
  const row = page.getByRole('row').filter({ hasText: 'E2E-COMPRA-002' });
  await expect(row).toContainText('74.00');
  await row.getByRole('button', { name: 'Detalle', exact: true }).click();
  await expect(page.getByRole('dialog')).toContainText('Lomo de res');
  await expect(page.getByRole('dialog')).toContainText('Limón');
});

async function open(page: Page, route = '/salon') {
  await page.goto(route);
  await expect(page.locator('main h1')).toBeVisible();
}
async function dialogSubmit(page: Page, name: string) {
  await page.getByRole('dialog').getByRole('button', { name, exact: true }).click();
}
async function noOverflow(page: Page) {
  const widths = await page.evaluate(() => ({
    scroll: document.documentElement.scrollWidth,
    viewport: window.innerWidth,
  }));
  expect(widths.scroll).toBeLessThanOrEqual(widths.viewport + 1);
}
test('las 14 rutas funcionan sin errores y se adaptan a la pantalla', async ({
  page,
}, testInfo) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  const routes = [
    'salon',
    'dashboard',
    'cocina',
    'reservas',
    'delivery',
    'caja',
    'compras',
    'inventario',
    'carta',
    'comprobantes',
    'contactos',
    'usuarios',
    'auditoria',
    'configuracion',
  ];
  for (const route of routes) {
    await open(page, '/' + route);
    await noOverflow(page);
    if (['salon', 'cocina', 'carta'].includes(route))
      await page.screenshot({ path: testInfo.outputPath(route + '.png'), fullPage: true });
  }
  expect(errors).toEqual([]);
});

test('atiende una mesa, consume receta, cobra con efectivo y Yape, y libera la mesa', async ({
  page,
}) => {
  await open(page);
  await page.getByRole('button', { name: 'Mesa 01, disponible', exact: true }).click();
  await page.getByLabel('Cliente', { exact: false }).fill('Cliente E2E');
  await page.getByLabel('Comensales').fill('2');
  await dialogSubmit(page, 'Abrir pedido');
  await expect(page.getByRole('dialog')).toHaveCount(0);
  const panel = page.getByRole('complementary', { name: 'Pedido seleccionado' });
  await expect(panel).toContainText('Mesa 01');
  await panel.getByRole('button', { name: 'Agregar productos' }).click();
  await page.getByRole('combobox', { name: 'Producto', exact: true }).selectOption('1');
  await page.getByLabel('Indicaciones para cocina').fill('Sin cebolla');
  await dialogSubmit(page, 'Agregar al pedido');
  await panel.getByRole('button', { name: 'Enviar comanda' }).click();
  await open(page, '/cocina');
  let ticket = page
    .locator('.ticket')
    .filter({ has: page.getByRole('heading', { name: 'Mesa 01', exact: true }) });
  await ticket.getByRole('button', { name: 'Comenzar preparación' }).click();
  ticket = page
    .locator('.ticket')
    .filter({ has: page.getByRole('heading', { name: 'Mesa 01', exact: true }) });
  await ticket.getByRole('button', { name: 'Marcar como listo' }).click();
  ticket = page
    .locator('.ticket')
    .filter({ has: page.getByRole('heading', { name: 'Mesa 01', exact: true }) });
  await ticket.getByRole('button', { name: 'Confirmar entrega' }).click();
  await open(page);
  await page.getByRole('button', { name: 'Mesa 01, ocupada', exact: true }).click();
  await panel.getByRole('button', { name: /^Cobrar/ }).click();
  await page.getByLabel('Importe a aplicar').fill('12.00');
  await page.getByLabel('Efectivo recibido').fill('20.00');
  await dialogSubmit(page, 'Registrar cobro');
  await expect(page.getByRole('status')).toContainText('8.00');
  await panel.getByRole('button', { name: /^Cobrar/ }).click();
  await page.getByLabel('Método de pago').selectOption('YAPE');
  await page.getByLabel('Referencia del pago digital').fill('E2E-YAPE-001');
  await dialogSubmit(page, 'Registrar cobro');
  await panel.getByRole('button', { name: 'Cerrar pedido y liberar mesa' }).click();
  await expect(
    page.getByRole('button', { name: 'Mesa 01, disponible', exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(
    page.getByRole('button', { name: 'Mesa 01, disponible', exact: true }),
  ).toBeVisible();
  await open(page, '/inventario');
  await expect(page.getByRole('row').filter({ hasText: 'Lomo de res' })).toContainText('2.2 KG');
});

test('un sobrepago muestra el error y conserva los campos del formulario', async ({ page }) => {
  await open(page);
  await page.getByRole('button', { name: 'Mesa 07, ocupada', exact: true }).click();
  await page
    .getByRole('complementary', { name: 'Pedido seleccionado' })
    .getByRole('button', { name: /^Cobrar/ })
    .click();
  await page.getByLabel('Importe a aplicar').fill('999.00');
  await dialogSubmit(page, 'Registrar cobro');
  await expect(page.getByRole('alert')).toContainText('saldo');
  await expect(page.getByLabel('Importe a aplicar')).toHaveValue('999.00');
  await page.getByLabel('Importe a aplicar').fill('128.00');
  await dialogSubmit(page, 'Registrar cobro');
  await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('crea un producto, filtra, pausa, da de baja y restaura conservando identidad', async ({
  page,
}) => {
  await open(page, '/carta');
  await page.getByRole('button', { name: 'Nuevo producto' }).click();
  await page.getByLabel('Nombre del producto').fill('Plato E2E');
  await page.getByLabel('Código único').fill('E2E-001');
  await page.getByLabel('Precio de venta').fill('27.50');
  await dialogSubmit(page, 'Guardar producto');
  await page.getByLabel('Buscar producto', { exact: true }).fill('E2E-001');
  const card = page.locator('.product-card').filter({ hasText: 'Plato E2E' });
  await expect(card).toContainText('27.50');
  await card.getByRole('button', { name: 'Pausar', exact: true }).click();
  await expect(card).toContainText('Pausado');
  await card.getByRole('button', { name: 'Dar de baja Plato E2E', exact: true }).click();
  await page.getByLabel('Motivo de la baja').fill('Prueba de restauración');
  await dialogSubmit(page, 'Confirmar baja');
  await expect(card).toHaveCount(0);
  await page.getByLabel('Incluir dados de baja').check();
  await card.getByRole('button', { name: 'Restaurar', exact: true }).click();
  await expect(card).toContainText('Pausado');
  await page.reload();
  await page.getByLabel('Buscar producto', { exact: true }).fill('E2E-001');
  await expect(card).toHaveCount(1);
});

test('recibe compra una sola vez y rechaza stock negativo', async ({ page }) => {
  await open(page, '/compras');
  const row = page.getByRole('row').filter({ hasText: 'F001-002458' });
  await row.getByRole('button', { name: 'Recibir', exact: true }).click();
  await dialogSubmit(page, 'Confirmar recepción');
  await expect(row).toContainText('Confirmada');
  await expect(row.getByRole('button', { name: 'Recibir', exact: true })).toHaveCount(0);
  await open(page, '/inventario');
  await expect(page.getByRole('row').filter({ hasText: 'Lomo de res' })).toContainText('7.4 KG');
  await page.getByRole('button', { name: 'Registrar movimiento' }).click();
  await page.getByRole('combobox', { name: 'Insumo', exact: true }).selectOption('1');
  await page.getByLabel('Tipo', { exact: false }).selectOption('MERMA');
  await page.getByLabel('Cantidad en unidad base').fill('999');
  await page.getByLabel('Motivo').fill('Prueba E2E');
  await dialogSubmit(page, 'Registrar movimiento');
  await expect(page.getByRole('alert')).toContainText('stock negativo');
});

test('rechaza reservas superpuestas y permite corregir el horario', async ({ page }) => {
  await open(page, '/reservas');
  await page.getByRole('button', { name: 'Nueva reserva' }).click();
  await page.getByLabel('Nombre del cliente').fill('Reserva E2E');
  await page.getByLabel('Teléfono').fill('999111222');
  await page.getByRole('combobox', { name: 'Mesa', exact: true }).selectOption('5');
  await page.getByLabel('Personas').fill('2');
  await dialogSubmit(page, 'Confirmar reserva');
  await expect(page.getByRole('alert')).toContainText('horario');
  await page.getByLabel('Hora de llegada').fill('20:30');
  await page.getByLabel('Hora de salida').fill('21:30');
  await dialogSubmit(page, 'Confirmar reserva');
  await expect(page.getByRole('row').filter({ hasText: 'Reserva E2E' })).toBeVisible();
});

test('el diálogo atrapa foco, cierra con Escape y devuelve el foco', async ({ page }) => {
  await open(page);
  const trigger = page.getByRole('button', { name: 'Nuevo pedido', exact: true });
  await trigger.click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  for (let index = 0; index < 9; index++) {
    await page.keyboard.press('Tab');
    expect(await dialog.evaluate((el) => el.contains(document.activeElement))).toBe(true);
  }
  await page.keyboard.press('Escape');
  await expect(dialog).toHaveCount(0);
  await expect(trigger).toBeFocused();
});

test('navegación por menú, atrás del navegador y ruta inexistente', async ({ page }, testInfo) => {
  await open(page);
  if (testInfo.project.name === 'mobile')
    await page.getByRole('button', { name: 'Abrir menú' }).click();
  await page
    .getByRole('navigation', { name: 'Menú principal' })
    .getByRole('link', { name: 'Carta y recetas' })
    .click();
  await expect(page.locator('main h1')).toHaveText('El sabor de tu restaurante');
  if (testInfo.project.name === 'mobile')
    await expect(page.getByRole('button', { name: 'Cerrar menú' })).toHaveCount(0);
  await page.goBack();
  await expect(page.locator('main h1')).toHaveText('El salón, bajo control');
  await open(page, '/no-existe');
  await expect(page.locator('main h1')).toHaveText('Esta página no está en la carta');
});
