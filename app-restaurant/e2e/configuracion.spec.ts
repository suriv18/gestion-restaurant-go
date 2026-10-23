import { expect, test, type Page } from '@playwright/test';
import { mockConfiguracionApi, type ConfiguracionMock } from './configuracion-support';
import { enterDemo, mockAuthApi } from './support';

const TODOS = [
  'NEGOCIO_VER',
  'NEGOCIO_ADMINISTRAR',
  'SUCURSAL_VER',
  'SUCURSAL_ADMINISTRAR',
  'MESA_VER',
  'MESA_ADMINISTRAR',
];

async function abrir(
  page: Page,
  permisos: string[],
  antesDeConfiguracion?: (mock: ConfiguracionMock) => void,
) {
  const auth = await mockAuthApi(page);
  auth.session = { ...auth.session, permisos };
  const mock = await mockConfiguracionApi(page);
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill('admin');
  await page.getByLabel('Contraseña', { exact: true }).fill('UnaClaveSegura2026');
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/dashboard$/);
  antesDeConfiguracion?.(mock);
  await page.goto('/configuracion');
  return mock;
}

const dialogo = (page: Page) => page.getByRole('dialog');
const seccionSedes = (page: Page) => page.locator('section[aria-labelledby="sedes-title"]');

function sinCamposDeAuditoria(body: unknown) {
  const claves = Object.keys((body ?? {}) as Record<string, unknown>);
  expect(
    claves.filter(
      (k) => k === 'creado_por' || k === 'actualizado_por' || k.startsWith('eliminado_'),
    ),
  ).toEqual([]);
}

/** Contrato: POST lleva Idempotency-Key, PATCH no; ningún cuerpo trae campos de auditoría. */
function contratoDeEscrituras(mock: ConfiguracionMock) {
  const escrituras = mock.llamadas.filter((c) => c.method === 'POST' || c.method === 'PATCH');
  expect(escrituras.length).toBeGreaterThan(0);
  for (const c of escrituras) {
    sinCamposDeAuditoria(c.body);
    if (c.method === 'POST') expect(c.idempotencyKey, 'POST ' + c.path).toMatch(/^[0-9a-f-]{36}$/);
    else expect(c.idempotencyKey, 'PATCH ' + c.path).toBeUndefined();
  }
}

test.describe('Configuración con la API simulada', () => {
  test('muestra negocio, sedes, zonas y mesas', async ({ page }) => {
    await abrir(page, TODOS);
    await expect(page.getByRole('heading', { name: 'Tu restaurante' })).toBeVisible();
    await expect(page.locator('#negocio-ruc')).toHaveValue('20123456789');
    await expect(page.getByText('Sede Principal')).toBeVisible();
    await expect(page.getByText('Mesa 1 · 4 personas')).toBeVisible();
  });

  test('edita el negocio enviando solo lo que cambió', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await page.locator('#negocio-nombre_comercial').fill('La Casa');
    await page.getByRole('button', { name: 'Guardar cambios' }).click();
    await expect(page.getByText('Guardamos los datos de tu restaurante.')).toBeVisible();
    const patch = mock.llamadas.find((c) => c.method === 'PATCH' && c.path === '/negocio');
    expect(patch?.body).toEqual({ nombre_comercial: 'La Casa' });
    contratoDeEscrituras(mock);
  });

  test('un rechazo de campo del negocio aparece bajo el campo y conserva lo escrito', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('PATCH /negocio', {
      status: 422,
      code: 'moneda_invalida',
      detail: 'La moneda debe tener 3 letras mayúsculas.',
    });
    await page.locator('#negocio-moneda').fill('xx');
    await page.getByRole('button', { name: 'Guardar cambios' }).click();
    await expect(page.getByText('La moneda debe tener 3 letras mayúsculas.')).toBeVisible();
    await expect(page.locator('#negocio-moneda')).toHaveValue('xx');
  });

  test('crea una sede, avisa que hay que asignarla y no duplica con el mismo formulario', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    await page.getByRole('button', { name: 'Nueva sede' }).click();
    await dialogo(page).getByLabel('Código', { exact: true }).fill('norte');
    await dialogo(page).getByLabel('Nombre').fill('Sede Norte');
    await dialogo(page).getByLabel('Dirección').fill('Calle Norte 1');
    await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
    await expect(page.getByText(/Se creó la sede/)).toBeVisible();
    await expect(page.getByText(/Asígnala a tu equipo en Equipo y accesos/)).toBeVisible();
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/sucursales');
    expect(alta?.idempotencyKey).toMatch(/^[0-9a-f-]{36}$/);
    expect(alta?.body).toMatchObject({ codigo: 'NORTE', nombre: 'Sede Norte' });
    expect(
      mock.llamadas.filter((c) => c.method === 'POST' && c.path === '/sucursales'),
    ).toHaveLength(1);
    contratoDeEscrituras(mock);
  });

  test('código de sede duplicado se muestra en el campo y el diálogo sigue abierto', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('POST /sucursales', {
      status: 409,
      code: 'codigo_duplicado',
      detail: 'Ya existe una sede con ese código.',
    });
    await page.getByRole('button', { name: 'Nueva sede' }).click();
    await dialogo(page).getByLabel('Código', { exact: true }).fill('PRINCIPAL');
    await dialogo(page).getByLabel('Nombre').fill('Otra');
    await dialogo(page).getByLabel('Dirección').fill('Calle 1');
    await dialogo(page).getByRole('button', { name: 'Crear sede' }).click();
    await expect(dialogo(page).getByText('Ya existe una sede con ese código.')).toBeVisible();
    await expect(dialogo(page).getByLabel('Nombre')).toHaveValue('Otra');
  });

  test('baja de sede pide motivo y la sede puede restaurarse', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await seccionSedes(page)
      .getByRole('button', { name: 'Dar de baja sede Sede Principal', exact: true })
      .click();
    await dialogo(page).getByLabel('Motivo').fill('Cierre temporal');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText('Motivo de la baja: Cierre temporal')).toBeVisible();
    expect(
      mock.llamadas.find((c) => c.path.endsWith('/baja') && c.path.startsWith('/sucursales'))?.body,
    ).toEqual({ motivo: 'Cierre temporal' });
    await seccionSedes(page).getByRole('button', { name: 'Restaurar', exact: true }).click();
    await expect(page.getByText(/volvió a estar activa/)).toBeVisible();
    expect(
      mock.llamadas.filter((c) => c.method === 'POST' && c.path === '/sucursales/1/restauracion'),
    ).toHaveLength(1);
    contratoDeEscrituras(mock);
  });

  test('crea una zona y una mesa en esa zona', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await page.getByRole('button', { name: 'Nueva zona' }).click();
    await dialogo(page).getByLabel('Nombre').fill('Terraza');
    await dialogo(page).getByRole('button', { name: 'Crear zona' }).click();
    await expect(page.getByRole('heading', { name: 'Terraza' })).toBeVisible();

    await page.getByRole('button', { name: 'Nueva mesa' }).click();
    await dialogo(page).getByLabel('Zona').selectOption({ label: 'Terraza' });
    await dialogo(page).getByLabel('Número de mesa').fill('T1');
    await dialogo(page).getByLabel('Capacidad').fill('6');
    await dialogo(page).getByRole('button', { name: 'Crear mesa' }).click();
    await expect(page.getByText('Mesa T1 · 6 personas')).toBeVisible();
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/mesas');
    expect(alta?.body).toMatchObject({ numero_mesa: 'T1', capacidad: 6, sucursal_id: '1' });
    expect(mock.llamadas.some((c) => c.method === 'POST' && c.path === '/zonas')).toBe(true);
    contratoDeEscrituras(mock);
  });

  test('número de mesa duplicado aparece en el campo', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('POST /mesas', {
      status: 409,
      code: 'numero_mesa_duplicado',
      detail: 'Ya existe una mesa con ese número en la sucursal.',
    });
    await page.getByRole('button', { name: 'Nueva mesa' }).click();
    await dialogo(page).getByLabel('Número de mesa').fill('1');
    await dialogo(page).getByRole('button', { name: 'Crear mesa' }).click();
    await expect(
      dialogo(page).getByText('Ya existe una mesa con ese número en la sucursal.'),
    ).toBeVisible();
  });

  test('el interruptor habilita o deshabilita la mesa sin motivo', async ({ page }) => {
    const mock = await abrir(page, TODOS);
    await page.getByRole('switch', { name: 'Habilitada: mesa 1' }).click();
    await expect
      .poll(() => mock.llamadas.find((c) => c.method === 'PATCH' && c.path === '/mesas/100')?.body)
      .toEqual({ habilitada: false });
    contratoDeEscrituras(mock);
  });

  test('baja de mesa pide motivo; baja de zona con mesas muestra el 409 del servidor', async ({
    page,
  }) => {
    const mock = await abrir(page, TODOS);
    mock.rechazos.set('POST /zonas/10/baja', {
      status: 409,
      code: 'zona_con_mesas',
      detail: 'La zona tiene mesas activas.',
    });
    await page.getByRole('button', { name: 'Dar de baja zona Salón' }).click();
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(dialogo(page).getByText('La zona tiene mesas activas.')).toBeVisible();
    await dialogo(page).getByRole('button', { name: 'Cancelar' }).click();

    await page.getByRole('button', { name: 'Dar de baja mesa 1' }).click();
    await dialogo(page).getByLabel('Motivo').fill('Mesa rota');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText('Motivo de la baja: Mesa rota')).toBeVisible();
    expect(mock.llamadas.some((c) => c.method === 'POST' && c.path === '/mesas/100/baja')).toBe(
      true,
    );
    contratoDeEscrituras(mock);
  });

  test('si la lista de sedes falla (403) muestra el error con Reintentar y el negocio sigue visible', async ({
    page,
  }) => {
    await abrir(page, TODOS, (m) =>
      m.rechazos.set('GET /sucursales', {
        status: 403,
        code: 'prohibido',
        detail: 'No tienes permiso para ver las sedes.',
      }),
    );
    await expect(page.locator('#negocio-ruc')).toHaveValue('20123456789');
    const sedes = seccionSedes(page);
    await expect(sedes.getByText('No pudimos cargar las sedes')).toBeVisible();
    await expect(sedes.getByRole('alert')).toContainText('No tienes permiso para ver las sedes.');
    // Las demás secciones siguen funcionando: el salón conserva sus zonas y sus acciones.
    await expect(page.getByRole('heading', { name: 'Salón' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Nueva zona' })).toBeVisible();
    await expect(page.getByText('Esta sede todavía no tiene zonas.')).toHaveCount(0);
    await sedes.getByRole('button', { name: 'Reintentar', exact: true }).click();
    await expect(sedes.getByText('Sede Principal')).toBeVisible();
    await expect(sedes.getByText('No pudimos cargar las sedes')).toHaveCount(0);
  });

  test('un mesero (solo MESA_VER) ve el salón en solo lectura y no pide negocio ni sedes', async ({
    page,
  }) => {
    const mock = await abrir(page, ['MESA_VER']);
    await expect(page.getByText('Mesa 1 · 4 personas')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Tu restaurante' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Nueva mesa' })).toHaveCount(0);
    expect(mock.llamadas.some((c) => c.path === '/negocio' || c.path === '/sucursales')).toBe(
      false,
    );
  });

  test('sin ningún permiso de configuración muestra el aviso', async ({ page }) => {
    const mock = await abrir(page, []);
    await expect(page.getByText(/No tienes permiso para ver la configuración/)).toBeVisible();
    expect(mock.llamadas).toHaveLength(0);
  });

  test('en modo demostración conserva la pantalla de ejemplo', async ({ page }) => {
    await enterDemo(page);
    await page.goto('/configuracion');
    await expect(page.getByRole('button', { name: 'Restablecer demostración' })).toBeVisible();
  });
});
