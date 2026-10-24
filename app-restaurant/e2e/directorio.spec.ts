import { test, expect, Page } from '@playwright/test';
import { enterDemo, mockAuthApi } from './support';
import { DirectorioMock, mockDirectorioApi } from './directorio-support';

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

const TODO = [
  'CLIENTE_VER',
  'CLIENTE_ADMINISTRAR',
  'PROVEEDOR_VER',
  'PROVEEDOR_ADMINISTRAR',
  'EMPLEADO_VER',
  'EMPLEADO_ADMINISTRAR',
  'USUARIO_ADMINISTRAR',
];

/** Inicia sesión con la interfaz (con los permisos dados) y abre la ruta. */
async function abrir(page: Page, ruta: string, permisos = TODO): Promise<DirectorioMock> {
  const auth = await mockAuthApi(page);
  auth.session = { ...auth.session, permisos };
  const mock = await mockDirectorioApi(page);
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill('ana');
  await page.getByLabel('Contraseña', { exact: true }).fill('clave-de-prueba');
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await page.goto(ruta);
  return mock;
}

const fila = (page: Page, texto: string) => page.locator('tbody tr').filter({ hasText: texto });
const dialogo = (page: Page) => page.getByRole('dialog');

test.describe('Clientes y proveedores con la API simulada', () => {
  test('lista los clientes del servidor y oculta las bajas hasta pedirlas', async ({ page }) => {
    await abrir(page, '/contactos');
    await expect(fila(page, 'Ana Ríos')).toContainText('DNI 12345678');
    await expect(fila(page, 'Mesa Doble SAC')).toContainText('RUC 20111111111');
    await expect(page.getByText('Luis Antiguo')).toHaveCount(0);
    await page.getByLabel('Incluir bajas').check();
    await expect(fila(page, 'Luis Antiguo')).toContainText('Sin documento');
    await expect(fila(page, 'Luis Antiguo')).toContainText('Motivo: se mudó');
    // No es la demostración.
    await expect(page.getByText('Valeria')).toHaveCount(0);
    await page.getByLabel('Buscar contacto').fill('mesa doble');
    await expect(page.locator('tbody tr')).toHaveCount(1);
  });

  test('crea un cliente con clave de idempotencia y documento normalizado', async ({ page }) => {
    const mock = await abrir(page, '/contactos');
    await page.getByRole('button', { name: '+ Nuevo cliente' }).click();
    await dialogo(page).locator('#field-nombre').fill('  Rosa Vega  ');
    await dialogo(page).locator('#field-tipo_documento').selectOption('DNI');
    await dialogo(page).locator('#field-numero_documento').fill('87654321');
    await dialogo(page).locator('#field-telefono').fill('999 888 777');
    await dialogo(page).getByRole('button', { name: 'Guardar' }).click();

    await expect(dialogo(page)).toHaveCount(0);
    await expect(fila(page, 'Rosa Vega')).toContainText('DNI 87654321');
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/clientes');
    expect(alta?.idempotencyKey).toMatch(UUID);
    expect(alta?.body).toMatchObject({
      nombre_razon_social: 'Rosa Vega',
      tipo_documento: 'DNI',
      numero_documento: '87654321',
      telefono: '999 888 777',
      email: null,
    });
  });

  test('un documento repetido aparece bajo el campo y el diálogo conserva lo escrito', async ({
    page,
  }) => {
    await abrir(page, '/contactos');
    await page.getByRole('button', { name: '+ Nuevo cliente' }).click();
    await dialogo(page).locator('#field-nombre').fill('Otra Ana');
    await dialogo(page).locator('#field-tipo_documento').selectOption('DNI');
    await dialogo(page).locator('#field-numero_documento').fill('12345678');
    await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
    await expect(dialogo(page)).toContainText('ya existe un registro con ese documento');
    await expect(dialogo(page).locator('#field-nombre')).toHaveValue('Otra Ana');
    await expect(page.locator('tbody tr').filter({ hasText: 'Otra Ana' })).toHaveCount(0);
  });

  test('edita enviando solo lo que cambia, da de baja con motivo y restaura', async ({ page }) => {
    const mock = await abrir(page, '/contactos');
    await fila(page, 'Ana Ríos').getByRole('button', { name: 'Editar' }).click();
    await dialogo(page).locator('#field-telefono').fill('999000111');
    await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
    await expect(fila(page, 'Ana Ríos')).toContainText('999000111');
    const patch = mock.llamadas.find((c) => c.method === 'PATCH');
    expect(patch?.path).toBe('/clientes/1');
    expect(patch?.body).toEqual({ telefono: '999000111' });

    await fila(page, 'Ana Ríos').getByRole('button', { name: 'Dar de baja' }).click();
    await dialogo(page).locator('#field-motivo').fill('Pidió que lo borren');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText(/Ana Ríos quedó de baja/)).toBeVisible();
    await expect(page.getByText('Ana Ríos', { exact: true })).toHaveCount(0);
    const baja = mock.llamadas.find((c) => c.path === '/clientes/1/baja');
    expect(baja?.idempotencyKey).toMatch(UUID);
    expect(baja?.body).toEqual({ motivo: 'Pidió que lo borren' });

    await page.getByLabel('Incluir bajas').check();
    await fila(page, 'Ana Ríos').getByRole('button', { name: 'Restaurar' }).click();
    await expect(page.getByText(/Ana Ríos volvió a estar activo/)).toBeVisible();
    await expect(fila(page, 'Ana Ríos').getByRole('button', { name: 'Dar de baja' })).toBeVisible();
  });

  test('crea un proveedor y el RUC inválido se rechaza bajo su campo', async ({ page }) => {
    const mock = await abrir(page, '/contactos');
    await page.getByRole('button', { name: 'Proveedores' }).click();
    await expect(fila(page, 'Distribuidora Andina SAC')).toContainText('RUC 20123456789');
    mock.rechazos.set('POST /proveedores', {
      status: 422,
      code: 'documento_invalido',
      detail: 'el documento no es válido: RUC exige 11 dígitos',
    });
    await page.getByRole('button', { name: '+ Nuevo proveedor' }).click();
    await dialogo(page).locator('#field-razon_social').fill('Carnes del Sur SAC');
    await dialogo(page).locator('#field-numero_documento').fill('123');
    await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
    await expect(dialogo(page)).toContainText('RUC exige 11 dígitos');
    await dialogo(page).locator('#field-numero_documento').fill('20999999999');
    await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
    await expect(dialogo(page)).toHaveCount(0);
    await expect(fila(page, 'Carnes del Sur SAC')).toContainText('RUC 20999999999');
    expect(
      mock.llamadas.filter((c) => c.method === 'POST' && c.path === '/proveedores'),
    ).toHaveLength(2);
  });

  test('con solo CLIENTE_VER lista sin ofrecer acciones ni pedir proveedores', async ({ page }) => {
    const mock = await abrir(page, '/contactos', ['CLIENTE_VER']);
    await expect(fila(page, 'Ana Ríos')).toBeVisible();
    await expect(page.getByRole('button', { name: '+ Nuevo cliente' })).toHaveCount(0);
    await expect(fila(page, 'Ana Ríos').getByRole('button')).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Proveedores' })).toHaveCount(0);
    expect(mock.llamadas.some((c) => c.path.startsWith('/proveedores'))).toBe(false);
  });

  test('sin permisos del directorio muestra el aviso y no llama al servidor', async ({ page }) => {
    const mock = await abrir(page, '/contactos', ['CAJA_OPERAR']);
    await expect(
      page.getByText(/No tienes permiso para ver clientes ni proveedores/),
    ).toBeVisible();
    expect(mock.llamadas).toHaveLength(0);
  });
});

test.describe('Empleados con la API simulada', () => {
  test('lista a los empleados y ofrece solo usuarios sin empleado para vincular', async ({
    page,
  }) => {
    await abrir(page, '/empleados');
    await expect(fila(page, 'María Quispe')).toContainText('DNI 40000001');
    await expect(fila(page, 'María Quispe')).toContainText('Sin acceso');
    await fila(page, 'María Quispe').getByRole('button', { name: 'Editar' }).click();
    const opciones = await dialogo(page).locator('#field-usuario_id option').allTextContents();
    // «luis» ya tiene empleado: no se ofrece.
    expect(opciones).toEqual(['Sin acceso', 'ana', 'marta']);
  });

  test('crea un empleado vinculado a un usuario y lo muestra con su acceso', async ({ page }) => {
    const mock = await abrir(page, '/empleados');
    await page.getByRole('button', { name: '+ Nuevo empleado' }).click();
    await dialogo(page).locator('#field-nombres').fill('Pedro');
    await dialogo(page).locator('#field-apellidos').fill('Alva');
    await dialogo(page).locator('#field-numero_documento').fill('40000002');
    await dialogo(page).locator('#field-cargo').fill('Cajero');
    await dialogo(page).locator('#field-usuario_id').selectOption({ label: 'ana' });
    await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
    await expect(dialogo(page)).toHaveCount(0);
    await expect(fila(page, 'Pedro Alva')).toContainText('Cajero');
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/empleados');
    expect(alta?.idempotencyKey).toMatch(UUID);
    expect(alta?.body).toMatchObject({
      tipo_documento: 'DNI',
      numero_documento: '40000002',
      nombres: 'Pedro',
      apellidos: 'Alva',
      cargo: 'Cajero',
      fecha_ingreso: null,
      usuario_id: '2',
    });
  });

  test('la baja con usuario activo se rechaza dentro del diálogo, sin cerrarlo', async ({
    page,
  }) => {
    const mock = await abrir(page, '/empleados');
    mock.rechazos.set('POST /empleados/11/baja', {
      status: 409,
      code: 'empleado_con_usuario_activo',
      detail: 'el empleado tiene un usuario activo: dé de baja primero su usuario',
    });
    await fila(page, 'María Quispe').getByRole('button', { name: 'Dar de baja' }).click();
    await dialogo(page).locator('#field-motivo').fill('Renunció');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(dialogo(page)).toContainText('dé de baja primero su usuario');
    await expect(dialogo(page).locator('#field-motivo')).toHaveValue('Renunció');
    // Con el mismo formulario, el reintento ya pasa.
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(dialogo(page)).toHaveCount(0);
    await expect(page.getByText(/María Quispe quedó de baja/)).toBeVisible();
  });

  test('sin EMPLEADO_ADMINISTRAR consulta sin ofrecer acciones', async ({ page }) => {
    await abrir(page, '/empleados', ['EMPLEADO_VER']);
    await expect(fila(page, 'María Quispe')).toBeVisible();
    await expect(page.getByRole('button', { name: '+ Nuevo empleado' })).toHaveCount(0);
    await expect(fila(page, 'María Quispe').getByRole('button')).toHaveCount(0);
  });

  test('en modo demostración explica que se necesita iniciar sesión', async ({ page }) => {
    await enterDemo(page);
    await page.goto('/empleados');
    await expect(page.getByText(/Inicia sesión para verlos/)).toBeVisible();
  });
});
