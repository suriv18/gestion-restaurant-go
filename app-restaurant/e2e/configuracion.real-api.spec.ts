import { expect, test, type Page } from '@playwright/test';

/**
 * Integración REAL de «Configuración» con el backend (sin simulación). Opt-in: requiere el stack
 * de Docker, el front con el proxy y un administrador existente. Los datos de cada corrida son
 * únicos y no se limpian (solo existen bajas lógicas).
 *
 * Requisito: el administrador de prueba debe tener al menos una sede asignada. La sede nueva no
 * se le asigna sola, así que la zona y la mesa se crean en la sede que el selector muestre
 * primero (normalmente una sede real preexistente). Esa zona y esa mesa solo se agregan: la mesa
 * termina restaurada (activa) y ambas quedan en la base compartida.
 *
 *   E2E_API_USER=admin E2E_API_PASSWORD=... npx playwright test configuracion.real-api
 */
const admin = process.env['E2E_API_USER'];
const adminPass = process.env['E2E_API_PASSWORD'];

test.skip(
  !admin || !adminPass,
  'Defina E2E_API_USER y E2E_API_PASSWORD para correr contra el API real.',
);
test.describe.configure({ mode: 'serial' });

const unico = String(Date.now()).slice(-8);
const codigo = `E2E${unico}`;
const sede = `Sede E2E ${unico}`;
const zona = `Zona E2E ${unico}`;
const numero = `E${unico.slice(-4)}`;

async function entrar(page: Page) {
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill(admin!);
  await page.getByLabel('Contraseña', { exact: true }).fill(adminPass!);
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/dashboard$/);
  await page.goto('/configuracion');
}

const dialogo = (page: Page) => page.getByRole('dialog');

test('crea una sede, una zona y una mesa, da de baja la mesa con motivo y la restaura', async ({
  page,
}) => {
  await entrar(page);
  await expect(page.getByRole('heading', { name: 'Tu restaurante' })).toBeVisible();

  // Sede: el administrador crea la sede pero no la ve en «Zonas y mesas» hasta que se la asignen.
  await page.getByRole('button', { name: 'Nueva sede' }).click();
  await dialogo(page).getByLabel('Código', { exact: true }).fill(codigo);
  await dialogo(page).getByLabel('Nombre').fill(sede);
  await dialogo(page).getByLabel('Dirección').fill('Av. de la Prueba 1');
  await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
  await expect(page.getByText(/Asígnala a tu equipo en Equipo y accesos/)).toBeVisible();
  await expect(
    page
      .locator('section[aria-labelledby="sedes-title"]')
      .getByRole('heading', { name: sede, level: 3 }),
  ).toBeVisible();

  // El mismo código otra vez: error en el campo, el diálogo sigue abierto.
  await page.getByRole('button', { name: 'Nueva sede' }).click();
  await dialogo(page).getByLabel('Código', { exact: true }).fill(codigo);
  await dialogo(page).getByLabel('Nombre').fill('Repetida');
  await dialogo(page).getByLabel('Dirección').fill('Calle 1');
  await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
  await expect(dialogo(page)).toBeVisible();
  await expect(dialogo(page).getByLabel('Nombre')).toHaveValue('Repetida');
  await dialogo(page).getByRole('button', { name: 'Cancelar' }).click();

  // Zona y mesa en una sede donde el administrador sí opera (la primera del selector).
  await expect(
    page.getByRole('button', { name: 'Nueva zona' }),
    'El administrador de prueba debe tener al menos una sede asignada para operar zonas y mesas.',
  ).toBeVisible();
  await page.getByRole('button', { name: 'Nueva zona' }).click();
  await dialogo(page).getByLabel('Nombre').fill(zona);
  await dialogo(page).getByRole('button', { name: 'Crear zona' }).click();
  await expect(page.getByRole('heading', { name: zona })).toBeVisible();

  await page.getByRole('button', { name: 'Nueva mesa' }).click();
  await dialogo(page).getByLabel('Zona').selectOption({ label: zona });
  await dialogo(page).getByLabel('Número de mesa').fill(numero);
  await dialogo(page).getByLabel('Capacidad').fill('4');
  await dialogo(page).getByRole('button', { name: 'Crear mesa' }).click();
  await expect(page.getByText(`Mesa ${numero} · 4 personas`)).toBeVisible();

  // Baja con motivo y restauración (conserva el ID: la mesa vuelve a aparecer activa).
  await page.getByRole('button', { name: `Dar de baja mesa ${numero}` }).click();
  await dialogo(page).getByLabel('Motivo').fill('Prueba E2E');
  await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(page.getByText('Motivo de la baja: Prueba E2E')).toBeVisible();
  await page.getByRole('button', { name: `Restaurar mesa ${numero}` }).click();
  await expect(page.getByText(`La mesa ${numero} volvió a estar activa.`)).toBeVisible();
});
