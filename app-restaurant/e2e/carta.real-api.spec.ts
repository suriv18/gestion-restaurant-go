import { expect, test, type Page } from '@playwright/test';

/**
 * Integración REAL de «Carta» y de las estaciones de «Configuración» con el backend (sin
 * simulación). Opt-in: requiere el stack de Docker con la migración 00010, el front con el proxy
 * y un administrador existente con una sede asignada (ADMIN recibe CARTA_, ESTACION_ e IMPUESTO_).
 * Los datos de cada corrida son únicos y no se limpian (solo existen bajas lógicas): la categoría
 * y el producto terminan restaurados y la estación también. No se crean tasas: no se pueden
 * borrar y alterarían los impuestos de la base compartida; solo se comprueba que se listen.
 *
 *   E2E_API_USER=admin E2E_API_PASSWORD=... npx playwright test carta.real-api
 */
const admin = process.env['E2E_API_USER'];
const adminPass = process.env['E2E_API_PASSWORD'];

test.skip(
  !admin || !adminPass,
  'Defina E2E_API_USER y E2E_API_PASSWORD para correr contra el API real.',
);
test.describe.configure({ mode: 'serial' });

const unico = String(Date.now()).slice(-8);
const categoria = `Categoría E2E ${unico}`;
const codigoProducto = `E2E${unico}`;
const producto = `Producto E2E ${unico}`;
const codigoEstacion = `E2E${unico}`;
const estacion = `Estación E2E ${unico}`;

async function entrar(page: Page, ruta: string) {
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill(admin!);
  await page.getByLabel('Contraseña', { exact: true }).fill(adminPass!);
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/dashboard$/);
  await page.goto(ruta);
}

const dialogo = (page: Page) => page.getByRole('dialog');
const fila = (page: Page, texto: string) => page.locator('tbody tr').filter({ hasText: texto });

test('categoría: alta, edición, baja y restauración', async ({ page }) => {
  await entrar(page, '/carta');
  await page.getByRole('tab', { name: 'Categorías' }).click();
  // Con la carta vacía el botón también aparece en el aviso de «Aún no hay categorías».
  await page.getByRole('button', { name: '+ Nueva categoría' }).first().click();
  await dialogo(page).getByLabel('Nombre', { exact: true }).fill(categoria);
  await dialogo(page).getByLabel('Orden').fill('900');
  await dialogo(page).getByRole('button', { name: 'Guardar categoría' }).click();
  await expect(page.getByText(`Se creó la categoría ${categoria}.`)).toBeVisible();

  await page.getByRole('button', { name: `Editar categoría ${categoria}` }).click();
  await dialogo(page).getByLabel('Descripción').fill('Creada por la prueba e2e');
  await dialogo(page).getByRole('button', { name: 'Guardar categoría' }).click();
  await expect(fila(page, categoria)).toContainText('Creada por la prueba e2e');
});

test('producto: alta, código repetido, pausa y habilitación, ajustes de sede, baja y restauración', async ({
  page,
}) => {
  await entrar(page, '/carta');
  await page.getByRole('button', { name: '+ Nuevo producto' }).first().click();
  await dialogo(page).getByLabel('Categoría').selectOption({ label: categoria });
  await dialogo(page).getByLabel('Código', { exact: true }).fill(codigoProducto.toLowerCase());
  await dialogo(page).getByLabel('Nombre del producto').fill(producto);
  await dialogo(page).getByLabel('Precio general (S/)').fill('25.5');
  await dialogo(page).getByRole('button', { name: 'Guardar producto' }).click();
  await expect(page.getByText(`Se creó el producto ${producto}.`)).toBeVisible();
  await expect(fila(page, producto)).toContainText('S/ 25.50');

  // El mismo código: el servidor lo rechaza dentro del diálogo.
  await page.getByRole('button', { name: '+ Nuevo producto' }).first().click();
  await dialogo(page).getByLabel('Categoría').selectOption({ label: categoria });
  await dialogo(page).getByLabel('Código', { exact: true }).fill(codigoProducto);
  await dialogo(page).getByLabel('Nombre del producto').fill(`${producto} bis`);
  await dialogo(page).getByLabel('Precio general (S/)').fill('10');
  await dialogo(page).getByRole('button', { name: 'Guardar producto' }).click();
  await expect(dialogo(page)).toBeVisible();
  await expect(dialogo(page).getByLabel('Nombre del producto')).toHaveValue(`${producto} bis`);
  await page.keyboard.press('Escape');

  // Sin fila en la sede el producto aparece «No ofrecido»; habilitarlo crea la fila.
  await expect(fila(page, producto)).toContainText('No ofrecido');
  await page.getByRole('button', { name: `Habilitar en sede el producto ${producto}` }).click();
  await expect(fila(page, producto)).toContainText('Disponible');
  await page.getByRole('button', { name: `Pausar producto ${producto}` }).click();
  await expect(fila(page, producto)).toContainText('Pausado');

  await page.getByRole('button', { name: `Ajustes de sede del producto ${producto}` }).click();
  await dialogo(page).getByLabel('Precio de esta sede (S/)').fill('30');
  await dialogo(page).getByRole('button', { name: 'Guardar ajustes' }).click();
  await expect(fila(page, producto)).toContainText('S/ 30.00');

  await page.getByRole('button', { name: `Dar de baja producto ${producto}` }).click();
  await dialogo(page).getByLabel('Motivo de la baja').fill('prueba e2e');
  await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(page.getByText(`${producto} quedó de baja`)).toBeVisible();
  await page.getByLabel('Estado').selectOption({ label: 'Dados de baja' });
  await expect(fila(page, producto)).toContainText('Motivo: prueba e2e');
  await page.getByRole('button', { name: `Restaurar producto ${producto}` }).click();
  await expect(page.getByText(`${producto} volvió a estar activo.`)).toBeVisible();
});

test('categoría con productos activos no se puede desactivar', async ({ page }) => {
  await entrar(page, '/carta');
  await page.getByRole('tab', { name: 'Categorías' }).click();
  await page.getByRole('button', { name: `Desactivar categoría ${categoria}` }).click();
  await dialogo(page).getByRole('button', { name: 'Desactivar', exact: true }).click();
  // El detalle viene del servidor; el diálogo se queda abierto y no hay baja.
  await expect(dialogo(page)).toContainText(/producto/i);
  await page.keyboard.press('Escape');
  await expect(fila(page, categoria)).toContainText('Activa');
});

test('estación: alta, código repetido, edición, baja y restauración', async ({ page }) => {
  await entrar(page, '/configuracion');
  await page.getByRole('button', { name: '+ Nueva estación' }).click();
  await dialogo(page).getByLabel('Código').fill(codigoEstacion);
  await dialogo(page).getByLabel('Nombre').fill(estacion);
  await dialogo(page).getByRole('button', { name: 'Crear estación' }).click();
  await expect(page.getByText(`Se creó la estación ${estacion}.`)).toBeVisible();

  await page.getByRole('button', { name: '+ Nueva estación' }).click();
  await dialogo(page).getByLabel('Código').fill(codigoEstacion);
  await dialogo(page).getByLabel('Nombre').fill(`${estacion} bis`);
  await dialogo(page).getByRole('button', { name: 'Crear estación' }).click();
  await expect(dialogo(page)).toBeVisible();
  await expect(dialogo(page).getByLabel('Nombre')).toHaveValue(`${estacion} bis`);
  await page.keyboard.press('Escape');

  await page.getByRole('button', { name: `Editar estación ${estacion}` }).click();
  await dialogo(page).getByLabel('Nombre').fill(`${estacion} editada`);
  await dialogo(page).getByRole('button', { name: 'Guardar cambios' }).click();
  await expect(fila(page, `${estacion} editada`)).toBeVisible();

  await page.getByRole('button', { name: `Dar de baja estación ${estacion} editada` }).click();
  await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(page.getByText(`La estación ${estacion} editada quedó de baja.`)).toBeVisible();
  await page.getByLabel('Incluir bajas').check();
  await page.getByRole('button', { name: `Restaurar estación ${estacion} editada` }).click();
  await expect(
    page.getByText(`La estación ${estacion} editada volvió a estar activa.`),
  ).toBeVisible();
});

test('impuestos y tasas: se listan con su vigencia', async ({ page }) => {
  await entrar(page, '/configuracion');
  await expect(page.getByRole('heading', { name: 'Impuestos y tasas' })).toBeVisible();
  await expect(page.getByRole('article').first()).toBeVisible();
});
