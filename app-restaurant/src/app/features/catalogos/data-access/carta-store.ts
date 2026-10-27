import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { AuthSession } from '../../../core/auth/auth-session';
import { collect } from '../../../core/http/collect';
import { describeProblem } from '../../../core/http/problem';
import { rechazo } from '../../../shared/forms/rechazo';
import { cambiosDe, opcional, texto } from '../../../shared/forms/valores';
import type { FieldProblem } from '../../../shared/ui/form-dialog';
import { ConfiguracionApi } from '../../configuracion/data-access/configuracion-api';
import { CategoriasApi } from './categorias-api';
import {
  Categoria,
  CategoriaAlta,
  CategoriaCambios,
  EstadoFiltro,
  Estacion,
  Impuesto,
  Producto,
  ProductoAlta,
  ProductoCambios,
  ProductoFiltros,
  TIPOS_PRODUCTO,
} from './carta.models';
import { normalizarDecimal } from './carta.util';
import { EstacionesApi } from './estaciones-api';
import { ImpuestosApi } from './impuestos-api';
import { ProductosApi } from './productos-api';

export const PERMISOS_CARTA = {
  cartaVer: 'CARTA_VER',
  cartaAdministrar: 'CARTA_ADMINISTRAR',
  cartaDisponibilidad: 'CARTA_DISPONIBILIDAD',
  estacionVer: 'ESTACION_VER',
  estacionAdministrar: 'ESTACION_ADMINISTRAR',
  impuestoVer: 'IMPUESTO_VER',
  impuestoAdministrar: 'IMPUESTO_ADMINISTRAR',
  sucursalVer: 'SUCURSAL_VER',
  sucursalAdministrar: 'SUCURSAL_ADMINISTRAR',
} as const;

export interface CartaNotice {
  kind: 'ok' | 'error';
  text: string;
}

/** Valores del formulario de categoría (el diálogo entrega siempre texto). */
export interface ValoresCategoria {
  nombre: string;
  descripcion: string;
  orden_visualizacion: string;
}

/** Valores del formulario de producto. */
export interface ValoresProducto {
  categoria_id: string;
  codigo: string;
  nombre: string;
  descripcion: string;
  tipo: string;
  factor_consumo: string;
  precio_venta: string;
  afectacion_igv: string;
  impuesto_id: string;
  imagen_url: string;
}

/** Valores de «Ajustes de sede»: precio vacío = precio general; estación vacía = sin estación. */
export interface ValoresSede {
  precio: string;
  estacionId: string;
}

export interface FiltrosCarta {
  q: string;
  categoriaId: string | null;
  estado: EstadoFiltro;
  /** null = todas las sedes (sin `sucursal_id`). */
  sucursalId: string | null;
  /** Solo con sede: true = disponibles, false = pausados, null = sin filtro. */
  soloDisponibles: boolean | null;
}

/** Campo del formulario al que pertenece cada rechazo del servidor. */
export const CAMPO_POR_CODIGO: Record<string, string> = {
  codigo_duplicado: 'codigo',
  codigo_invalido: 'codigo',
  nombre_invalido: 'nombre',
  categoria_duplicada: 'nombre',
  descripcion_invalido: 'descripcion',
  imagen_url_invalido: 'imagen_url',
  tipo_invalido: 'tipo',
  articulo_requerido: 'tipo',
  articulo_no_permitido: 'tipo',
  precio_invalido: 'precio_venta',
  precio_venta_invalido: 'precio_venta',
  factor_consumo_invalido: 'factor_consumo',
  afectacion_invalida: 'afectacion_igv',
  impuesto_requerido: 'impuesto_id',
  impuesto_id_invalido: 'impuesto_id',
  categoria_requerida: 'categoria_id',
  categoria_id: 'categoria_id',
  categoria_id_invalido: 'categoria_id',
  referencia_invalida: 'categoria_id',
  orden_invalido: 'orden_visualizacion',
  motivo_requerido: 'motivo',
};

/** Rechazos propios del diálogo «Ajustes de sede». */
const CAMPO_AJUSTE_SEDE: Record<string, string> = {
  precio_invalido: 'precio',
  precio_venta_invalido: 'precio',
  estacion_invalida: 'estacionId',
  estacion_inactiva: 'estacionId',
};

type Resultado = Promise<string | FieldProblem | undefined>;

const FILTROS_INICIALES = (sucursalId: string | null): FiltrosCarta => ({
  q: '',
  categoriaId: null,
  estado: 'activos',
  sucursalId,
  soloDisponibles: null,
});

/**
 * Estado de «Carta» con datos reales. Se provee en la página (no en root). Todo lo que muestra
 * viene del servidor; el navegador solo oculta lo que el permiso no permite (el servidor vuelve a
 * validarlo). Las acciones de diálogo devuelven el rechazo (o nada si salió bien) para que el
 * diálogo lo muestre sin cerrarse.
 */
@Injectable()
export class CartaStore {
  private readonly categoriasApi = inject(CategoriasApi);
  private readonly productosApi = inject(ProductosApi);
  private readonly estacionesApi = inject(EstacionesApi);
  private readonly impuestosApi = inject(ImpuestosApi);
  private readonly sucursalesApi = inject(ConfiguracionApi);
  private readonly auth = inject(AuthSession);

  /** Carga inicial (categorías + primera página de productos). */
  readonly status = signal<'loading' | 'ready' | 'error'>('loading');
  readonly loadError = signal('');
  readonly notice = signal<CartaNotice | null>(null);
  readonly busyId = signal<string | null>(null);

  readonly categorias = signal<Categoria[]>([]);
  readonly productos = signal<Producto[]>([]);
  /** Estado de la lista de productos al cambiar filtros o recargar. */
  readonly listaStatus = signal<'loading' | 'ready' | 'error'>('ready');
  readonly listaError = signal('');
  readonly cargandoMas = signal(false);
  private readonly nextCursor = signal<string | null>(null);
  readonly hayMas = computed(() => this.nextCursor() !== null);

  /** Impuestos activos y estaciones activas de la sede, para los formularios. */
  readonly impuestos = signal<Impuesto[]>([]);
  readonly estaciones = signal<Estacion[]>([]);

  readonly filtros = signal<FiltrosCarta>(FILTROS_INICIALES(null));

  readonly puedeVerCarta = computed(
    () =>
      this.tiene(PERMISOS_CARTA.cartaVer) ||
      this.tiene(PERMISOS_CARTA.cartaAdministrar) ||
      this.tiene(PERMISOS_CARTA.cartaDisponibilidad),
  );
  readonly puedeAdministrarCarta = computed(() => this.tiene(PERMISOS_CARTA.cartaAdministrar));
  readonly puedePausar = computed(
    () =>
      this.tiene(PERMISOS_CARTA.cartaDisponibilidad) || this.tiene(PERMISOS_CARTA.cartaAdministrar),
  );
  readonly puedeVerEstaciones = computed(
    () => this.tiene(PERMISOS_CARTA.estacionVer) || this.tiene(PERMISOS_CARTA.estacionAdministrar),
  );
  readonly puedeVerImpuestos = computed(
    () =>
      this.tiene(PERMISOS_CARTA.impuestoVer) ||
      this.tiene(PERMISOS_CARTA.impuestoAdministrar) ||
      this.tiene(PERMISOS_CARTA.cartaAdministrar),
  );
  readonly puedeVerSedes = computed(
    () => this.tiene(PERMISOS_CARTA.sucursalVer) || this.tiene(PERMISOS_CARTA.sucursalAdministrar),
  );
  readonly sinAcceso = computed(() => !this.puedeVerCarta());

  /** Sedes de la sesión (ids); el servidor responde 404 si una no es accesible. */
  readonly sedes = computed<readonly string[]>(() => this.auth.user()?.branches ?? []);
  /** Nombres de las sedes (id -> nombre), solo si el actor puede leer `/sucursales`. */
  private readonly nombresSede = signal<Record<string, string>>({});

  /** Nombre de la sede para mostrar; sin permiso de sedes (o si falló la lectura) queda «Sede {id}». */
  nombreSede(id: string): string {
    return this.nombresSede()[id] ?? `Sede ${id}`;
  }

  /** Categorías ordenadas por `orden_visualizacion` y nombre. */
  readonly categoriasOrdenadas = computed(() =>
    [...this.categorias()].sort(
      (a, b) =>
        a.orden_visualizacion - b.orden_visualizacion || a.nombre.localeCompare(b.nombre, 'es'),
    ),
  );
  readonly categoriasActivas = computed(() =>
    this.categoriasOrdenadas().filter((c) => c.es_activo),
  );

  /** Se descarta la respuesta de una lista que ya quedó obsoleta por otra petición. */
  private secuencia = 0;

  /** Carga lo que el permiso deja ver; los datos de apoyo de los formularios fallan en silencio relativo. */
  async load(): Promise<void> {
    if (this.sinAcceso()) {
      this.status.set('ready');
      return;
    }
    this.status.set('loading');
    this.loadError.set('');
    this.filtros.set(FILTROS_INICIALES(this.sedes()[0] ?? null));
    this.nextCursor.set(null);
    try {
      const [categorias, pagina] = await Promise.all([
        collect((c) => firstValueFrom(this.categoriasApi.listar(c))),
        firstValueFrom(this.productosApi.listar(this.filtrosApi())),
      ]);
      this.categorias.set(categorias);
      this.productos.set(pagina.items);
      this.nextCursor.set(pagina.nextCursor);
    } catch (error) {
      this.loadError.set(describeProblem(error));
      this.status.set('error');
      return;
    }
    await Promise.all([this.cargarImpuestos(), this.cargarEstaciones(), this.cargarNombresSedes()]);
    this.status.set('ready');
  }

  // --- Productos: filtros y paginación ---------------------------------------------------

  /** Cambia filtros y vuelve a pedir la primera página al servidor. */
  async cambiarFiltros(cambios: Partial<FiltrosCarta>): Promise<void> {
    const anterior = this.filtros();
    const siguiente = { ...anterior, ...cambios };
    // Sin sede no hay disponibilidad que filtrar.
    if (siguiente.sucursalId === null) siguiente.soloDisponibles = null;
    this.filtros.set(siguiente);
    const reinicia = this.recargarProductos();
    if (siguiente.sucursalId !== anterior.sucursalId) {
      await Promise.all([reinicia, this.cargarEstaciones()]);
    } else {
      await reinicia;
    }
  }

  /** Primera página con los filtros actuales; devuelve si lo logró. */
  async recargarProductos(): Promise<boolean> {
    const id = ++this.secuencia;
    this.listaStatus.set('loading');
    this.listaError.set('');
    this.cargandoMas.set(false);
    try {
      const pagina = await firstValueFrom(this.productosApi.listar(this.filtrosApi()));
      if (id !== this.secuencia) return false;
      this.productos.set(pagina.items);
      this.nextCursor.set(pagina.nextCursor);
      this.listaStatus.set('ready');
      return true;
    } catch (error) {
      if (id !== this.secuencia) return false;
      this.listaError.set(describeProblem(error));
      this.listaStatus.set('error');
      return false;
    }
  }

  /** «Cargar más»: la página siguiente se acumula tras la actual. */
  async cargarMas(): Promise<void> {
    const cursor = this.nextCursor();
    if (!cursor || this.cargandoMas()) return;
    const id = this.secuencia;
    this.cargandoMas.set(true);
    try {
      const pagina = await firstValueFrom(this.productosApi.listar(this.filtrosApi(), cursor));
      if (id !== this.secuencia) return;
      this.productos.update((lista) => [...lista, ...pagina.items]);
      this.nextCursor.set(pagina.nextCursor);
    } catch (error) {
      if (id === this.secuencia) this.say('error', describeProblem(error));
    } finally {
      this.cargandoMas.set(false);
    }
  }

  // --- Categorías ------------------------------------------------------------------------

  async crearCategoria(v: ValoresCategoria, key: string): Resultado {
    const nueva = this.cuerpoCategoria(v);
    try {
      await firstValueFrom(this.categoriasApi.crear(nueva, key));
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.refrescarCategorias(`Se creó la categoría ${nueva.nombre}.`);
    return undefined;
  }

  async editarCategoria(actual: Categoria, v: ValoresCategoria): Resultado {
    const nueva = this.cuerpoCategoria(v);
    const cambios = cambiosDe(
      {
        nombre: actual.nombre,
        descripcion: actual.descripcion,
        orden_visualizacion: actual.orden_visualizacion,
      },
      { ...nueva },
    ) as CategoriaCambios;
    if (Object.keys(cambios).length === 0) return undefined;
    try {
      await firstValueFrom(this.categoriasApi.editar(actual.id, cambios));
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.refrescarCategorias(`Guardamos los cambios de la categoría ${nueva.nombre}.`);
    return undefined;
  }

  /** Baja sin motivo (la confirmación la pide la vista); el 409 con productos activos se muestra tal cual. */
  async desactivarCategoria(c: Categoria, key: string): Resultado {
    try {
      await firstValueFrom(this.categoriasApi.baja(c.id, key));
    } catch (error) {
      return rechazo(error, {});
    }
    await this.refrescarCategorias(`La categoría ${c.nombre} quedó de baja.`);
    return undefined;
  }

  async restaurarCategoria(c: Categoria, key: string): Promise<void> {
    await this.conBloqueo(c.id, async () => {
      await firstValueFrom(this.categoriasApi.restaurar(c.id, key));
      await this.refrescarCategorias(`La categoría ${c.nombre} volvió a estar activa.`);
    });
  }

  // --- Productos: escrituras -------------------------------------------------------------

  async crearProducto(v: ValoresProducto, key: string): Resultado {
    const nuevo = this.cuerpoProducto(v);
    try {
      await firstValueFrom(this.productosApi.crear(nuevo, key));
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.avisarTrasRecarga(`Se creó el producto ${nuevo.nombre}.`);
    return undefined;
  }

  /** PATCH solo con lo que cambió; sin cambios no llama al servidor. */
  async editarProducto(actual: Producto, v: ValoresProducto): Resultado {
    const nuevo = this.cuerpoProducto(v);
    const cambios = this.cambiosProducto(actual, nuevo);
    if (Object.keys(cambios).length === 0) return undefined;
    try {
      await firstValueFrom(this.productosApi.editar(actual.id, cambios));
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.avisarTrasRecarga(`Guardamos los cambios de ${nuevo.nombre}.`);
    return undefined;
  }

  async darDeBajaProducto(p: Producto, motivo: string, key: string): Resultado {
    try {
      await firstValueFrom(this.productosApi.baja(p.id, texto(motivo), key));
    } catch (error) {
      return rechazo(error, CAMPO_POR_CODIGO);
    }
    await this.avisarTrasRecarga(
      `${p.nombre} quedó de baja en todas las sedes. Puedes restaurarlo cuando quieras.`,
    );
    return undefined;
  }

  async restaurarProducto(p: Producto, key: string): Promise<void> {
    await this.conBloqueo(p.id, async () => {
      await firstValueFrom(this.productosApi.restaurar(p.id, key));
      await this.avisarTrasRecarga(`${p.nombre} volvió a estar activo.`);
    });
  }

  // --- Disponibilidad por sede -----------------------------------------------------------

  pausarEnSede(p: Producto): Promise<void> {
    return this.cambiarDisponibilidad(p, false);
  }

  habilitarEnSede(p: Producto): Promise<void> {
    return this.cambiarDisponibilidad(p, true);
  }

  /**
   * «Ajustes de sede»: precio propio (vacío = precio general) y estación (vacía = sin estación).
   * Solo envía lo que cambió; la disponibilidad se conserva (un producto «no ofrecido» queda
   * disponible al darle ajustes, porque la fila de la sede se crea con ellos).
   */
  async ajustarSede(p: Producto, v: ValoresSede): Resultado {
    const sede = this.filtros().sucursalId;
    if (!sede) return 'Elige una sede para ajustar el producto.';
    const precio = opcional(v.precio);
    const estacion = opcional(v.estacionId);
    const actual = p.en_sucursal ?? null;
    const cuerpo: { precio_venta?: string | null; estacion_id?: string | null } = {};
    const precioActual = actual?.precio_sucursal ?? null;
    if (this.decimal(precio) !== this.decimal(precioActual)) cuerpo.precio_venta = precio;
    if (estacion !== (actual?.estacion_id ?? null)) cuerpo.estacion_id = estacion;
    if (Object.keys(cuerpo).length === 0) return undefined;
    try {
      await firstValueFrom(
        this.productosApi.ponerDisponibilidad(p.id, {
          sucursal_id: sede,
          disponible: actual?.disponible ?? true,
          ...cuerpo,
        }),
      );
    } catch (error) {
      return rechazo(error, CAMPO_AJUSTE_SEDE);
    }
    await this.refrescarFila(p, `Guardamos los ajustes de ${p.nombre} en esta sede.`);
    return undefined;
  }

  dismissNotice(): void {
    this.notice.set(null);
  }

  // --- Internos --------------------------------------------------------------------------

  private tiene(permiso: string): boolean {
    return this.auth.hasPermission(permiso);
  }

  private say(kind: CartaNotice['kind'], text: string): void {
    this.notice.set({ kind, text });
  }

  private filtrosApi(): ProductoFiltros {
    const f = this.filtros();
    const filtros: ProductoFiltros = {
      es_activo: f.estado === 'activos' ? 'true' : f.estado === 'bajas' ? 'false' : 'todos',
    };
    const q = f.q.trim();
    if (q) filtros.q = q;
    if (f.categoriaId) filtros.categoria_id = f.categoriaId;
    if (f.sucursalId) {
      filtros.sucursal_id = f.sucursalId;
      if (f.soloDisponibles !== null) filtros.disponible = f.soloDisponibles;
    }
    return filtros;
  }

  /** Ejecuta una acción sin diálogo: una a la vez y con aviso de error. */
  private async conBloqueo(id: string, accion: () => Promise<void>): Promise<void> {
    if (this.busyId()) return;
    this.busyId.set(id);
    try {
      await accion();
    } catch (error) {
      this.say('error', describeProblem(error));
    } finally {
      this.busyId.set(null);
    }
  }

  private async cambiarDisponibilidad(p: Producto, disponible: boolean): Promise<void> {
    const sede = this.filtros().sucursalId;
    if (!sede) {
      this.say('error', 'Elige una sede para pausar o habilitar el producto.');
      return;
    }
    await this.conBloqueo(p.id, async () => {
      await firstValueFrom(
        this.productosApi.ponerDisponibilidad(p.id, { sucursal_id: sede, disponible }),
      );
      await this.refrescarFila(
        p,
        disponible
          ? `${p.nombre} está disponible en esta sede.`
          : `${p.nombre} quedó pausado en esta sede.`,
      );
    });
  }

  /** Vuelve a pedir un producto con la sede actual y reemplaza su fila (conserva «cargar más»). */
  private async refrescarFila(p: Producto, ok: string): Promise<void> {
    try {
      const sede = this.filtros().sucursalId ?? undefined;
      const fresco = await firstValueFrom(this.productosApi.obtener(p.id, sede));
      this.productos.update((lista) => lista.map((x) => (x.id === fresco.id ? fresco : x)));
      this.say('ok', ok);
    } catch {
      this.say(
        'error',
        `${ok} Pero no pudimos actualizar la fila: recarga la lista para verla al día.`,
      );
    }
  }

  private async avisarTrasRecarga(ok: string): Promise<void> {
    if (await this.recargarProductos()) {
      this.say('ok', ok);
    } else {
      this.say('error', `${ok} Pero no pudimos actualizar la lista: recárgala para verla al día.`);
    }
  }

  private async refrescarCategorias(ok: string): Promise<void> {
    try {
      this.categorias.set(await collect((c) => firstValueFrom(this.categoriasApi.listar(c))));
      this.say('ok', ok);
    } catch {
      this.say(
        'error',
        `${ok} Pero no pudimos actualizar la lista de categorías: recarga la página para verla al día.`,
      );
    }
  }

  /** Mejora cosmética: si no se puede leer, el selector sigue mostrando «Sede {id}». */
  private async cargarNombresSedes(): Promise<void> {
    if (!this.puedeVerSedes()) {
      this.nombresSede.set({});
      return;
    }
    try {
      const sedes = await collect((c) => firstValueFrom(this.sucursalesApi.sucursales(c)));
      this.nombresSede.set(Object.fromEntries(sedes.map((s) => [s.id, s.nombre])));
    } catch {
      this.nombresSede.set({});
    }
  }

  private async cargarImpuestos(): Promise<void> {
    if (!this.puedeAdministrarCarta() || !this.puedeVerImpuestos()) return;
    try {
      const todos = await collect((c) => firstValueFrom(this.impuestosApi.impuestos(c)));
      this.impuestos.set(todos.filter((i) => i.es_activo));
    } catch (error) {
      this.say('error', `No pudimos cargar los impuestos: ${describeProblem(error)}`);
    }
  }

  /** Estaciones activas de la sede elegida (para el diálogo de ajustes); sin sede, ninguna. */
  private async cargarEstaciones(): Promise<void> {
    const sede = this.filtros().sucursalId;
    if (!sede || !this.puedeAdministrarCarta() || !this.puedeVerEstaciones()) {
      this.estaciones.set([]);
      return;
    }
    try {
      this.estaciones.set(
        await collect((c) =>
          firstValueFrom(this.estacionesApi.listar({ sucursal_id: sede, es_activo: true }, c)),
        ),
      );
    } catch (error) {
      this.estaciones.set([]);
      this.say('error', `No pudimos cargar las estaciones: ${describeProblem(error)}`);
    }
  }

  private cuerpoCategoria(v: ValoresCategoria): CategoriaAlta {
    return {
      nombre: texto(v.nombre),
      descripcion: opcional(v.descripcion),
      orden_visualizacion: Number(texto(v.orden_visualizacion) || '0'),
    };
  }

  private cuerpoProducto(v: ValoresProducto): ProductoAlta {
    const tipo = TIPOS_PRODUCTO.find((t) => t.value === v.tipo);
    return {
      categoria_id: texto(v.categoria_id),
      codigo: texto(v.codigo),
      nombre: texto(v.nombre),
      descripcion: opcional(v.descripcion),
      tipo: (tipo?.value ?? v.tipo) as ProductoAlta['tipo'],
      // Los tipos sin factor (servicio) no lo envían: el servidor usa 1.
      factor_consumo: tipo?.usaFactor === false ? '1' : texto(v.factor_consumo) || '1',
      precio_venta: texto(v.precio_venta),
      afectacion_igv: texto(v.afectacion_igv) as ProductoAlta['afectacion_igv'],
      impuesto_id: opcional(v.impuesto_id),
      imagen_url: opcional(v.imagen_url),
    };
  }

  /** Compara precio y factor como decimales canónicos: «25.5» y «25.50» no son un cambio. */
  private cambiosProducto(actual: Producto, nuevo: ProductoAlta): ProductoCambios {
    const comparable = (o: ProductoAlta | Producto) => ({
      categoria_id: o.categoria_id,
      codigo: o.codigo,
      nombre: o.nombre,
      descripcion: o.descripcion,
      tipo: o.tipo,
      factor_consumo: this.decimal(o.factor_consumo ?? null),
      precio_venta: this.decimal(o.precio_venta),
      afectacion_igv: o.afectacion_igv,
      impuesto_id: o.impuesto_id,
      imagen_url: o.imagen_url,
    });
    const claves = Object.keys(cambiosDe(comparable(actual), comparable(nuevo)));
    const cuerpo = nuevo as unknown as Record<string, string | null>;
    return Object.fromEntries(claves.map((k) => [k, cuerpo[k]])) as ProductoCambios;
  }

  private decimal(valor: string | null): string | null {
    return valor === null ? null : normalizarDecimal(valor);
  }
}
