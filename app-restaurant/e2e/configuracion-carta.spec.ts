import { expect, test, type Page } from '@playwright/test';
import { mockCartaApi, type CartaMock } from './carta-support';
import { mockAuthApi } from './support';

const TODOS = ['ESTACION_VER', 'ESTACION_ADMINISTRAR', 'IMPUESTO_VER', 'IMPUESTO_ADMINISTRAR'];

async function abrir(page: Page, permisos: string[] = TODOS) {
  const auth = await mockAuthApi(page);
  auth.session = { ...auth.session, permisos };
  const mock = await mockCartaApi(page);
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill('admin');
  await page.getByLabel('Contraseña', { exact: true }).fill('UnaClaveSegura2026');
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/dashboard$/);
  await page.goto('/configuracion');
  return mock;
}

const dialogo = (page: Page) => page.getByRole('dialog');
const seccionEstaciones = (page: Page) =>
  page.locator('section[aria-labelledby="estaciones-title"]');
const bloqueIgv = (page: Page) => page.getByRole('article', { name: 'Impuesto IGV' });

function contratoDeEscrituras(mock: CartaMock) {
  const escrituras = mock.llamadas.filter((c) => ['POST', 'PATCH'].includes(c.method));
  expect(escrituras.length).toBeGreaterThan(0);
  for (const c of escrituras) {
    if (c.method === 'POST') expect(c.idempotencyKey, 'POST ' + c.path).toMatch(/^[0-9a-f-]{36}$/);
    else expect(c.idempotencyKey, 'PATCH ' + c.path).toBeUndefined();
  }
}

test.describe('Estaciones de preparación', () => {
  test('lista las estaciones de la sede y crea una con el código en mayúsculas', async ({
    page,
  }) => {
    const mock = await abrir(page);
    await expect(seccionEstaciones(page).getByText('Cocina', { exact: true })).toBeVisible();
    expect(
      mock.llamadas.find((c) => c.method === 'GET' && c.path === '/estaciones')!.query,
    ).toMatchObject({ sucursal_id: '1' });

    await page.getByRole('button', { name: '+ Nueva estación' }).click();
    await dialogo(page).getByLabel('Código').fill('barra');
    await dialogo(page).getByLabel('Nombre').fill('Barra');
    await dialogo(page).getByRole('button', { name: 'Crear estación' }).click();
    await expect(page.getByText('Se creó la estación Barra.')).toBeVisible();
    await expect(seccionEstaciones(page).getByText('BARRA', { exact: true })).toBeVisible();
    const altas = mock.llamadas.filter((c) => c.method === 'POST' && c.path === '/estaciones');
    expect(altas).toHaveLength(1);
    expect(altas[0].body).toMatchObject({ sucursal_id: '1', codigo: 'BARRA', nombre: 'Barra' });
    contratoDeEscrituras(mock);
  });

  test('un código repetido aparece en el campo y conserva lo escrito', async ({ page }) => {
    await abrir(page);
    await page.getByRole('button', { name: '+ Nueva estación' }).click();
    await dialogo(page).getByLabel('Código').fill('COCINA');
    await dialogo(page).getByLabel('Nombre').fill('Cocina 2');
    await dialogo(page).getByRole('button', { name: 'Crear estación' }).click();
    await expect(
      dialogo(page).getByText('Ya existe una estación con ese código en la sede.'),
    ).toBeVisible();
    await expect(dialogo(page).getByLabel('Nombre')).toHaveValue('Cocina 2');
  });

  test('edita el nombre, da de baja y restaura', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: 'Editar estación Cocina' }).click();
    await dialogo(page).getByLabel('Nombre').fill('Cocina caliente');
    await dialogo(page).getByRole('button', { name: 'Guardar cambios' }).click();
    await expect(
      page.getByText('Guardamos los cambios de la estación Cocina caliente.'),
    ).toBeVisible();
    expect(mock.llamadas.find((c) => c.method === 'PATCH')!.body).toEqual({
      nombre: 'Cocina caliente',
    });

    await page.getByRole('button', { name: 'Dar de baja estación Cocina caliente' }).click();
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText('La estación Cocina caliente quedó de baja.')).toBeVisible();

    await page.getByLabel('Incluir bajas').check();
    await page.getByRole('button', { name: 'Restaurar estación Cocina caliente' }).click();
    await expect(
      page.getByText('La estación Cocina caliente volvió a estar activa.'),
    ).toBeVisible();
    contratoDeEscrituras(mock);
  });

  test('«estación en uso» se muestra con el detalle del servidor', async ({ page }) => {
    const mock = await abrir(page);
    mock.rechazos.set('POST /estaciones/50/baja', {
      status: 409,
      code: 'estacion_en_uso',
      detail: 'La estación la usan productos activos de la sede.',
    });
    await page.getByRole('button', { name: 'Dar de baja estación Cocina' }).click();
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(
      dialogo(page).getByText('La estación la usan productos activos de la sede.'),
    ).toBeVisible();
  });

  test('solo con ESTACION_VER no hay acciones de escritura', async ({ page }) => {
    await abrir(page, ['ESTACION_VER']);
    await expect(seccionEstaciones(page).getByText('Cocina', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '+ Nueva estación' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /^Editar estación/ })).toHaveCount(0);
  });
});

test.describe('Impuestos y tasas', () => {
  test('muestra la tasa vigente hoy con su vigencia abierta y el fundamento', async ({ page }) => {
    await abrir(page);
    await expect(bloqueIgv(page)).toContainText(/18\s*%/);
    await expect(bloqueIgv(page)).toContainText('Vigente hoy');
    await expect(bloqueIgv(page)).toContainText('Fundamento: Ley del IGV');
    await expect(page.getByText(/No hay una tasa por defecto/)).toBeVisible();
  });

  test('registra una tasa futura sin tocar la vigente', async ({ page }) => {
    const mock = await abrir(page);
    // La tasa actual no tiene fin; primero se le pone para que la nueva no se cruce.
    await page.getByRole('button', { name: /^Editar tasa 18/ }).click();
    await dialogo(page).getByLabel('Vigente hasta (no incluida)').fill('2030-01-01');
    await dialogo(page).getByRole('button', { name: 'Guardar cambios' }).click();
    await expect(page.getByText('Guardamos los cambios de la tasa de IGV.')).toBeVisible();
    expect(mock.llamadas.find((c) => c.method === 'PATCH')!.body).toEqual({
      vigente_hasta: '2030-01-01',
    });

    await page.getByRole('button', { name: 'Nueva tasa de IGV' }).click();
    await dialogo(page).getByLabel('Tasa (%)').fill('17.5');
    await dialogo(page).getByLabel('Vigente desde').fill('2030-01-01');
    await dialogo(page).getByLabel('Fundamento').fill('Reforma tributaria');
    await dialogo(page).getByRole('button', { name: 'Registrar tasa' }).click();
    await expect(page.getByText('Se registró la tasa de IGV.')).toBeVisible();
    await expect(bloqueIgv(page)).toContainText('17.5');
    await expect(bloqueIgv(page).getByText('Vigente hoy')).toHaveCount(1);
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/tasas-impuesto')!;
    expect(alta.body).toMatchObject({
      impuesto_id: '1',
      tasa: '17.5',
      vigente_desde: '2030-01-01',
      vigente_hasta: null,
    });
    contratoDeEscrituras(mock);
  });

  test('una vigencia que se cruza se rechaza junto al campo y conserva lo escrito', async ({
    page,
  }) => {
    await abrir(page);
    await page.getByRole('button', { name: 'Nueva tasa de IGV' }).click();
    await dialogo(page).getByLabel('Tasa (%)').fill('10');
    await dialogo(page).getByLabel('Vigente desde').fill('2026-01-01');
    await dialogo(page).getByRole('button', { name: 'Registrar tasa' }).click();
    await expect(
      dialogo(page).getByText('La vigencia se cruza con otra tasa del impuesto.'),
    ).toBeVisible();
    await expect(dialogo(page).getByLabel('Tasa (%)')).toHaveValue('10');
  });

  test('sin IMPUESTO_ADMINISTRAR solo se consulta', async ({ page }) => {
    await abrir(page, ['IMPUESTO_VER']);
    await expect(bloqueIgv(page)).toContainText(/18\s*%/);
    await expect(page.getByRole('button', { name: 'Nueva tasa de IGV' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /^Editar tasa/ })).toHaveCount(0);
  });

  test('sin permisos de estaciones ni impuestos esas secciones no aparecen', async ({ page }) => {
    await abrir(page, ['NEGOCIO_VER']);
    await expect(page.locator('#estaciones-title')).toHaveCount(0);
    await expect(page.locator('#tasas-title')).toHaveCount(0);
  });
});
