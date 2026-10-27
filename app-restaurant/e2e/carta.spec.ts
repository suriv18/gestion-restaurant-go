import { expect, test, type Page } from '@playwright/test';
import { mockCartaApi, type CartaMock } from './carta-support';
import { mockAuthApi } from './support';

const TODOS = [
  'CARTA_VER',
  'CARTA_ADMINISTRAR',
  'CARTA_DISPONIBILIDAD',
  'ESTACION_VER',
  'IMPUESTO_VER',
  'SUCURSAL_VER',
];

async function abrir(page: Page, permisos: string[] = TODOS) {
  const auth = await mockAuthApi(page);
  auth.session = { ...auth.session, permisos };
  const mock = await mockCartaApi(page);
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Usuario' }).fill('admin');
  await page.getByLabel('Contraseña', { exact: true }).fill('UnaClaveSegura2026');
  await page.getByRole('button', { name: 'Iniciar sesión', exact: true }).click();
  await expect(page).toHaveURL(/dashboard$/);
  await page.goto('/carta');
  return mock;
}

const dialogo = (page: Page) => page.getByRole('dialog');
const fila = (page: Page, nombre: string) =>
  page.getByRole('row').filter({ has: page.getByText(nombre, { exact: true }) });

function productosPedidos(mock: CartaMock) {
  return mock.llamadas.filter((c) => c.method === 'GET' && c.path === '/productos');
}

/** Contrato: POST lleva Idempotency-Key; PATCH y PUT no; ningún cuerpo trae auditoría. */
function contratoDeEscrituras(mock: CartaMock) {
  const escrituras = mock.llamadas.filter((c) => ['POST', 'PATCH', 'PUT'].includes(c.method));
  expect(escrituras.length).toBeGreaterThan(0);
  for (const c of escrituras) {
    const claves = Object.keys((c.body ?? {}) as Record<string, unknown>);
    expect(claves.filter((k) => k === 'creado_por' || k.startsWith('eliminado_'))).toEqual([]);
    if (c.method === 'POST') expect(c.idempotencyKey, 'POST ' + c.path).toMatch(/^[0-9a-f-]{36}$/);
    else expect(c.idempotencyKey, c.method + ' ' + c.path).toBeUndefined();
  }
}

test.describe('Carta con la API simulada', () => {
  test('lista los productos de la sede con su disponibilidad y precio efectivo', async ({
    page,
  }) => {
    const mock = await abrir(page);
    await expect(page.getByRole('heading', { name: 'El sabor de tu restaurante' })).toBeVisible();
    await expect(fila(page, 'Ceviche clásico')).toContainText('Disponible');
    await expect(fila(page, 'Ceviche clásico')).toContainText('S/ 32.00');
    await expect(fila(page, 'Lomo saltado')).toContainText('Pausado');
    await expect(fila(page, 'Lomo saltado')).toContainText('S/ 40.00');
    await expect(fila(page, 'Lomo saltado')).toContainText('Estación: Cocina');
    await expect(fila(page, 'Chicha morada')).toContainText('No ofrecido');
    const pedido = productosPedidos(mock)[0];
    expect(pedido.query).toMatchObject({ es_activo: 'true', sucursal_id: '1' });
  });

  test('«Todas las sedes» oculta la disponibilidad y pide la lista sin sede', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByLabel('Sede', { exact: true }).selectOption({ label: 'Sede Principal' });
    await page.getByLabel('Sede', { exact: true }).selectOption({ label: 'Todas las sedes' });
    await expect(page.getByRole('columnheader', { name: 'Disponibilidad' })).toHaveCount(0);
    const ultimo = productosPedidos(mock).at(-1)!;
    expect(ultimo.query['sucursal_id']).toBeUndefined();
    await expect(
      page.getByText('Elige una sede para pausar o habilitar productos en ella.'),
    ).toBeVisible();
  });

  test('la búsqueda y la categoría se envían al servidor', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByLabel('Buscar producto').fill('lomo');
    await page.getByRole('button', { name: 'Buscar', exact: true }).click();
    await expect(fila(page, 'Lomo saltado')).toBeVisible();
    await expect(page.getByText('Ceviche clásico')).toHaveCount(0);
    expect(productosPedidos(mock).at(-1)!.query['q']).toBe('lomo');

    await page.getByRole('button', { name: 'Limpiar filtros' }).click();
    await page.getByLabel('Categoría', { exact: true }).selectOption({ label: 'Bebidas' });
    await expect(fila(page, 'Chicha morada')).toBeVisible();
    expect(productosPedidos(mock).at(-1)!.query['categoria_id']).toBe('3');
  });

  test('«Cargar más» sigue el cursor del servidor', async ({ page }) => {
    const mock = await abrir(page);
    mock.paginaDe = 2;
    await page.getByLabel('Estado').selectOption({ label: 'Todos' });
    await expect(page.getByText('Mostrando 2')).toBeVisible();
    await page.getByRole('button', { name: 'Cargar más' }).click();
    await expect(page.getByText('Mostrando 3')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Cargar más' })).toHaveCount(0);
    expect(productosPedidos(mock).at(-1)!.query['cursor']).toBe('2');
  });

  test('pausa y vuelve a habilitar un producto sin Idempotency-Key', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: 'Pausar producto Ceviche clásico' }).click();
    await expect(page.getByText('Ceviche clásico quedó pausado en esta sede.')).toBeVisible();
    await expect(fila(page, 'Ceviche clásico')).toContainText('Pausado');
    await page.getByRole('button', { name: 'Habilitar en sede el producto Chicha morada' }).click();
    await expect(page.getByText('Chicha morada está disponible en esta sede.')).toBeVisible();
    await expect(fila(page, 'Chicha morada')).toContainText('Disponible');
    const puts = mock.llamadas.filter((c) => c.method === 'PUT');
    expect(puts.map((c) => c.body)).toEqual([
      { sucursal_id: '1', disponible: false },
      { sucursal_id: '1', disponible: true },
    ]);
    contratoDeEscrituras(mock);
  });

  test('ajustes de sede: vaciar el precio vuelve al general y solo viaja lo que cambió', async ({
    page,
  }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: 'Ajustes de sede del producto Lomo saltado' }).click();
    await dialogo(page).getByLabel('Precio de esta sede (S/)').fill('');
    await dialogo(page).getByRole('button', { name: 'Guardar ajustes' }).click();
    await expect(
      page.getByText('Guardamos los ajustes de Lomo saltado en esta sede.'),
    ).toBeVisible();
    await expect(fila(page, 'Lomo saltado')).toContainText('S/ 38.50');
    const put = mock.llamadas.find((c) => c.method === 'PUT')!;
    expect(put.body).toEqual({ sucursal_id: '1', disponible: false, precio_venta: null });
  });

  test('un precio de sede inválido aparece bajo el campo y conserva lo escrito', async ({
    page,
  }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: 'Ajustes de sede del producto Lomo saltado' }).click();
    await dialogo(page).getByLabel('Precio de esta sede (S/)').fill('12.345');
    await dialogo(page).getByRole('button', { name: 'Guardar ajustes' }).click();
    await expect(
      dialogo(page).getByText('Escribe un precio en soles, por ejemplo 25.50.'),
    ).toBeVisible();
    await expect(dialogo(page).getByLabel('Precio de esta sede (S/)')).toHaveValue('12.345');
    expect(mock.llamadas.some((c) => c.method === 'PUT')).toBe(false);
  });

  test('crea un producto gravado: código en mayúsculas, precio como texto y una sola alta', async ({
    page,
  }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: '+ Nuevo producto' }).click();
    await dialogo(page).getByLabel('Categoría').selectOption({ label: 'Fondos' });
    await dialogo(page).getByLabel('Código', { exact: true }).fill('arroz');
    await dialogo(page).getByLabel('Nombre del producto').fill('Arroz chaufa');
    await dialogo(page).getByLabel('Precio general (S/)').fill('27.5');
    await dialogo(page).getByRole('button', { name: 'Guardar producto' }).click();
    await expect(page.getByText('Se creó el producto Arroz chaufa.')).toBeVisible();
    await expect(fila(page, 'Arroz chaufa')).toContainText('S/ 27.50');
    const altas = mock.llamadas.filter((c) => c.method === 'POST' && c.path === '/productos');
    expect(altas).toHaveLength(1);
    expect(altas[0].body).toMatchObject({
      categoria_id: '2',
      codigo: 'ARROZ',
      nombre: 'Arroz chaufa',
      tipo: 'PREPARADO',
      afectacion_igv: '10',
      impuesto_id: '1',
    });
    expect(typeof (altas[0].body as { precio_venta: unknown }).precio_venta).toBe('string');
    expect((await dialogo(page).count()) === 0, 'el diálogo se cierra tras guardar').toBe(true);
    contratoDeEscrituras(mock);
  });

  test('un producto gravado sin impuesto no se envía y marca el campo', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: '+ Nuevo producto' }).click();
    await dialogo(page).getByLabel('Código', { exact: true }).fill('flan');
    await dialogo(page).getByLabel('Nombre del producto').fill('Flan');
    await dialogo(page).getByLabel('Precio general (S/)').fill('8');
    await dialogo(page).getByLabel('Impuesto').selectOption({ label: 'Sin impuesto' });
    await dialogo(page).getByRole('button', { name: 'Guardar producto' }).click();
    await expect(
      dialogo(page).getByText('Elige el impuesto: es obligatorio en un producto gravado.'),
    ).toBeVisible();
    expect(mock.llamadas.some((c) => c.method === 'POST' && c.path === '/productos')).toBe(false);
  });

  test('un código duplicado del servidor aparece en el campo y el diálogo sigue abierto', async ({
    page,
  }) => {
    await abrir(page);
    await page.getByRole('button', { name: '+ Nuevo producto' }).click();
    await dialogo(page).getByLabel('Código', { exact: true }).fill('CEV');
    await dialogo(page).getByLabel('Nombre del producto').fill('Otro ceviche');
    await dialogo(page).getByLabel('Precio general (S/)').fill('30');
    await dialogo(page).getByRole('button', { name: 'Guardar producto' }).click();
    await expect(dialogo(page).getByText('Ya existe un producto con ese código.')).toBeVisible();
    await expect(dialogo(page).getByLabel('Nombre del producto')).toHaveValue('Otro ceviche');
  });

  test('edita enviando solo el campo que cambió', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: 'Editar producto Lomo saltado' }).click();
    await dialogo(page).getByLabel('Precio general (S/)').fill('39.90');
    await dialogo(page).getByRole('button', { name: 'Guardar producto' }).click();
    await expect(page.getByText('Guardamos los cambios de Lomo saltado.')).toBeVisible();
    const patch = mock.llamadas.find((c) => c.method === 'PATCH' && c.path === '/productos/11')!;
    expect(patch.body).toEqual({ precio_venta: '39.90' });
    contratoDeEscrituras(mock);
  });

  test('la baja pide motivo, se ve en «Dados de baja» y se restaura con el mismo id', async ({
    page,
  }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: 'Dar de baja producto Chicha morada' }).click();
    await dialogo(page).getByLabel('Motivo de la baja').fill('Sin stock de maíz morado');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText('Chicha morada quedó de baja en todas las sedes.')).toBeVisible();
    await expect(page.getByText('Chicha morada', { exact: true })).toHaveCount(0);
    expect(mock.llamadas.find((c) => c.path === '/productos/12/baja')?.body).toEqual({
      motivo: 'Sin stock de maíz morado',
    });

    await page.getByLabel('Estado').selectOption({ label: 'Dados de baja' });
    await expect(fila(page, 'Chicha morada')).toContainText('Motivo: Sin stock de maíz morado');
    await page.getByRole('button', { name: 'Restaurar producto Chicha morada' }).click();
    await expect(page.getByText('Chicha morada volvió a estar activo.')).toBeVisible();
    expect(
      mock.llamadas.filter((c) => c.method === 'POST' && c.path === '/productos/12/restauracion'),
    ).toHaveLength(1);
    contratoDeEscrituras(mock);
  });
});

test.describe('Categorías con la API simulada', () => {
  test('crea, edita y ordena por orden de visualización', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByRole('tab', { name: 'Categorías' }).click();
    await page.getByRole('button', { name: '+ Nueva categoría' }).click();
    await dialogo(page).getByLabel('Nombre', { exact: true }).fill('Postres');
    await dialogo(page).getByLabel('Orden').fill('0');
    await dialogo(page).getByRole('button', { name: 'Guardar categoría' }).click();
    await expect(page.getByText('Se creó la categoría Postres.')).toBeVisible();
    await expect(page.getByRole('row').nth(1)).toContainText('Postres');
    const alta = mock.llamadas.find((c) => c.method === 'POST' && c.path === '/categorias')!;
    expect(alta.body).toMatchObject({ nombre: 'Postres', orden_visualizacion: 0 });

    await page.getByRole('button', { name: 'Editar categoría Entradas' }).click();
    await dialogo(page).getByLabel('Nombre', { exact: true }).fill('Entradas frías');
    await dialogo(page).getByRole('button', { name: 'Guardar categoría' }).click();
    await expect(
      page.getByText('Guardamos los cambios de la categoría Entradas frías.'),
    ).toBeVisible();
    expect(
      mock.llamadas.find((c) => c.method === 'PATCH' && c.path === '/categorias/1')!.body,
    ).toEqual({
      nombre: 'Entradas frías',
    });
    contratoDeEscrituras(mock);
  });

  test('desactivar una categoría con productos muestra el motivo del servidor y no la da de baja', async ({
    page,
  }) => {
    await abrir(page);
    await page.getByRole('tab', { name: 'Categorías' }).click();
    await page.getByRole('button', { name: 'Desactivar categoría Fondos' }).click();
    await dialogo(page).getByRole('button', { name: 'Desactivar', exact: true }).click();
    await expect(
      dialogo(page).getByText(
        'La categoría tiene productos activos: muévelos o dalos de baja primero.',
      ),
    ).toBeVisible();
  });

  test('desactiva una categoría vacía y la restaura desde «Incluir bajas»', async ({ page }) => {
    const mock = await abrir(page);
    await page.getByRole('button', { name: 'Dar de baja producto Chicha morada' }).click();
    await dialogo(page).getByLabel('Motivo de la baja').fill('Fuera de temporada');
    await dialogo(page).getByRole('button', { name: 'Confirmar baja' }).click();
    await expect(page.getByText('Chicha morada quedó de baja')).toBeVisible();

    await page.getByRole('tab', { name: 'Categorías' }).click();
    await page.getByRole('button', { name: 'Desactivar categoría Bebidas' }).click();
    await dialogo(page).getByRole('button', { name: 'Desactivar', exact: true }).click();
    await expect(page.getByText('La categoría Bebidas quedó de baja.')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Restaurar categoría Bebidas' })).toHaveCount(0);

    await page.getByLabel('Incluir bajas').check();
    await page.getByRole('button', { name: 'Restaurar categoría Bebidas' }).click();
    await expect(page.getByText('La categoría Bebidas volvió a estar activa.')).toBeVisible();
    expect(
      mock.llamadas.filter((c) => c.method === 'POST' && c.path.startsWith('/categorias/3/')),
    ).toHaveLength(2);
  });
});

test.describe('Permisos en la carta', () => {
  test('con solo CARTA_VER no hay botones de escritura ni de disponibilidad', async ({ page }) => {
    await abrir(page, ['CARTA_VER']);
    await expect(fila(page, 'Ceviche clásico')).toBeVisible();
    await expect(page.getByRole('button', { name: '+ Nuevo producto' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /^Editar producto/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /^Pausar producto/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /^Dar de baja producto/ })).toHaveCount(0);
  });

  test('con CARTA_DISPONIBILIDAD puede pausar pero no editar', async ({ page }) => {
    await abrir(page, ['CARTA_DISPONIBILIDAD']);
    await expect(
      page.getByRole('button', { name: 'Pausar producto Ceviche clásico' }),
    ).toBeVisible();
    await expect(page.getByRole('button', { name: /^Editar producto/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /^Ajustes de sede/ })).toHaveCount(0);
  });

  test('sin permisos de carta explica que no hay acceso', async ({ page }) => {
    await abrir(page, ['CAJA_OPERAR']);
    await expect(page.getByText(/No tienes permiso para ver la carta/)).toBeVisible();
  });
});
