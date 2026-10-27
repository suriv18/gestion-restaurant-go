import { expect, test, type Page } from '@playwright/test';

/**
 * Integración REAL de «Clientes y proveedores» y «Empleados» con el backend (sin simulación).
 * Opt-in: requiere el stack de Docker con la migración 00009, el front con el proxy y un
 * administrador existente (con CLIENTE_, PROVEEDOR_ y EMPLEADO_ ADMINISTRAR, que ADMIN recibe).
 * Los datos de cada corrida son únicos y no se limpian (solo existen bajas lógicas); los clientes,
 * proveedores y empleados terminan restaurados (activos).
 *
 *   E2E_API_USER=admin E2E_API_PASSWORD=... npx playwright test directorio.real-api
 */
const admin = process.env['E2E_API_USER'];
const adminPass = process.env['E2E_API_PASSWORD'];

test.skip(
  !admin || !adminPass,
  'Defina E2E_API_USER y E2E_API_PASSWORD para correr contra el API real.',
);
test.describe.configure({ mode: 'serial' });

const unico = String(Date.now()).slice(-8);
const dniCliente = unico;
const rucProveedor = `20${unico}0`.slice(0, 11);
const dniEmpleado = String(Number(unico) + 1).padStart(8, '0');

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

test('cliente: alta, documento repetido, edición, baja con motivo y restauración', async ({
  page,
}) => {
  const nombre = `Cliente E2E ${unico}`;
  await entrar(page, '/contactos');
  await expect(page.getByRole('button', { name: '+ Nuevo cliente' })).toBeVisible();

  await page.getByRole('button', { name: '+ Nuevo cliente' }).click();
  await dialogo(page).locator('#field-nombre').fill(nombre);
  await dialogo(page).locator('#field-tipo_documento').selectOption('DNI');
  await dialogo(page).locator('#field-numero_documento').fill(dniCliente);
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialogo(page)).toHaveCount(0);
  await expect(fila(page, nombre)).toContainText(`DNI ${dniCliente}`);

  // El mismo documento: el servidor lo rechaza dentro del diálogo, bajo el campo.
  await page.getByRole('button', { name: '+ Nuevo cliente' }).click();
  await dialogo(page).locator('#field-nombre').fill(`${nombre} bis`);
  await dialogo(page).locator('#field-tipo_documento').selectOption('DNI');
  await dialogo(page).locator('#field-numero_documento').fill(dniCliente);
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialogo(page)).toContainText('ya existe un cliente con ese documento');
  await page.keyboard.press('Escape');

  await fila(page, nombre).getByRole('button', { name: 'Editar' }).click();
  await dialogo(page).locator('#field-telefono').fill('999 000 111');
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(fila(page, nombre)).toContainText('999 000 111');

  await fila(page, nombre).getByRole('button', { name: 'Dar de baja' }).click();
  await dialogo(page).locator('#field-motivo').fill('prueba e2e');
  await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await expect(page.getByText(new RegExp(`${nombre} quedó de baja`))).toBeVisible();
  await page.getByLabel('Incluir bajas').check();
  await expect(fila(page, nombre)).toContainText('Motivo: prueba e2e');

  // Un documento dado de baja sigue reservado.
  await page.getByRole('button', { name: '+ Nuevo cliente' }).click();
  await dialogo(page).locator('#field-nombre').fill(`${nombre} otra vez`);
  await dialogo(page).locator('#field-tipo_documento').selectOption('DNI');
  await dialogo(page).locator('#field-numero_documento').fill(dniCliente);
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialogo(page)).toContainText('ya existe un cliente con ese documento');
  await page.keyboard.press('Escape');

  await fila(page, nombre).getByRole('button', { name: 'Restaurar' }).click();
  await expect(fila(page, nombre).getByRole('button', { name: 'Dar de baja' })).toBeVisible();
});

test('proveedor: alta con RUC, RUC inválido rechazado bajo su campo, baja y restauración', async ({
  page,
}) => {
  const razon = `Proveedor E2E ${unico}`;
  await entrar(page, '/contactos');
  await page.getByRole('button', { name: 'Proveedores' }).click();

  await page.getByRole('button', { name: '+ Nuevo proveedor' }).click();
  await dialogo(page).locator('#field-razon_social').fill(razon);
  await dialogo(page).locator('#field-numero_documento').fill('123');
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialogo(page)).toContainText('el documento no es válido');
  await dialogo(page).locator('#field-numero_documento').fill(rucProveedor);
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialogo(page)).toHaveCount(0);
  await expect(fila(page, razon)).toContainText(`RUC ${rucProveedor}`);

  await fila(page, razon).getByRole('button', { name: 'Dar de baja' }).click();
  await dialogo(page).locator('#field-motivo').fill('prueba e2e');
  await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await page.getByLabel('Incluir bajas').check();
  await expect(fila(page, razon)).toContainText('De baja');
  await fila(page, razon).getByRole('button', { name: 'Restaurar' }).click();
  await expect(fila(page, razon).getByRole('button', { name: 'Dar de baja' })).toBeVisible();
});

test('empleado: alta, documento repetido, edición, baja con motivo y restauración', async ({
  page,
}) => {
  const nombres = 'Persona';
  const apellidos = `E2E ${unico}`;
  await entrar(page, '/empleados');

  await page.getByRole('button', { name: '+ Nuevo empleado' }).click();
  await dialogo(page).locator('#field-nombres').fill(nombres);
  await dialogo(page).locator('#field-apellidos').fill(apellidos);
  await dialogo(page).locator('#field-numero_documento').fill(dniEmpleado);
  await dialogo(page).locator('#field-cargo').fill('Mesera');
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialogo(page)).toHaveCount(0);
  await expect(fila(page, apellidos)).toContainText('Mesera');
  await expect(fila(page, apellidos)).toContainText('Sin acceso');

  // Documento repetido: error bajo el campo.
  await page.getByRole('button', { name: '+ Nuevo empleado' }).click();
  await dialogo(page).locator('#field-nombres').fill('Otra');
  await dialogo(page).locator('#field-apellidos').fill('Persona');
  await dialogo(page).locator('#field-numero_documento').fill(dniEmpleado);
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(dialogo(page)).toContainText('ya existe un empleado con ese documento');
  await page.keyboard.press('Escape');

  await fila(page, apellidos).getByRole('button', { name: 'Editar' }).click();
  await dialogo(page).locator('#field-cargo').fill('Jefa de sala');
  await dialogo(page).getByRole('button', { name: 'Guardar' }).click();
  await expect(fila(page, apellidos)).toContainText('Jefa de sala');

  await fila(page, apellidos).getByRole('button', { name: 'Dar de baja' }).click();
  await dialogo(page).locator('#field-motivo').fill('prueba e2e');
  await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
  await page.getByLabel('Incluir bajas').check();
  await expect(fila(page, apellidos)).toContainText('Motivo: prueba e2e');
  await fila(page, apellidos).getByRole('button', { name: 'Restaurar' }).click();
  await expect(fila(page, apellidos).getByRole('button', { name: 'Dar de baja' })).toBeVisible();
});
