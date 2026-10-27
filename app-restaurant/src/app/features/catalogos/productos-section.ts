import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { newIdempotencyKey } from '../../core/http/idempotency-key';
import { Dialogs, type Field, type FieldProblem } from '../../shared/ui/form-dialog';
import { CartaStore, type FiltrosCarta, type ValoresProducto } from './data-access/carta-store';
import { Producto, TIPOS_PRODUCTO } from './data-access/carta.models';
import {
  factorValido,
  formatearSoles,
  normalizarDecimal,
  precioValido,
} from './data-access/carta.util';
import { etiquetaAfectacion, etiquetaTipo, OPCIONES_AFECTACION } from './carta.textos';

type Disponibilidad = 'disponible' | 'pausado' | 'no_ofrecido';

/** Una fila de la tabla, ya con textos legibles. */
interface Fila {
  producto: Producto;
  categoria: string;
  tipo: string;
  precio: string;
  /** Precio general tachado cuando la sede cobra otro. */
  precioGeneral: string | null;
  igv: string;
  impuesto: string | null;
  disponibilidad: Disponibilidad | null;
  estacion: string | null;
}

const ETIQUETA_DISPONIBILIDAD: Record<Disponibilidad, string> = {
  disponible: 'Disponible',
  pausado: 'Pausado',
  no_ofrecido: 'No ofrecido',
};

type Resultado = Promise<string | FieldProblem | undefined>;

/** Productos de la carta: filtros que consulta el servidor, tabla, acciones por fila y diálogos. */
@Component({
  selector: 'app-productos-section',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <form
      class="section-toolbar filtros-carta"
      role="search"
      aria-label="Filtrar productos"
      (submit)="buscar($event)"
    >
      <label class="search">
        <input
          type="search"
          aria-label="Buscar producto"
          placeholder="Buscar por nombre o código..."
          [value]="borrador()"
          (input)="borrador.set($any($event.target).value)"
        />
      </label>
      <button type="submit" class="btn">Buscar</button>
      <select aria-label="Categoría" (change)="cambiar({ categoriaId: valor($event) || null })">
        <option value="" [selected]="!store.filtros().categoriaId">Todas las categorías</option>
        @for (c of store.categoriasActivas(); track c.id) {
          <option [value]="c.id" [selected]="c.id === store.filtros().categoriaId">
            {{ c.nombre }}
          </option>
        }
      </select>
      <select aria-label="Estado" (change)="cambiar({ estado: $any(valor($event)) })">
        <option value="activos" [selected]="store.filtros().estado === 'activos'">Activos</option>
        <option value="bajas" [selected]="store.filtros().estado === 'bajas'">Dados de baja</option>
        <option value="todos" [selected]="store.filtros().estado === 'todos'">Todos</option>
      </select>
      <select aria-label="Sede" (change)="cambiar({ sucursalId: valor($event) || null })">
        <option value="" [selected]="!store.filtros().sucursalId">Todas las sedes</option>
        @for (id of store.sedes(); track id) {
          <option [value]="id" [selected]="id === store.filtros().sucursalId">
            {{ nombreSede(id) }}
          </option>
        }
      </select>
      @if (store.filtros().sucursalId) {
        <select
          aria-label="Disponibilidad"
          aria-describedby="nota-disponibilidad"
          (change)="cambiarDisponibilidad($event)"
        >
          <option value="" [selected]="store.filtros().soloDisponibles === null">
            Disponibilidad: todos
          </option>
          <option value="si" [selected]="store.filtros().soloDisponibles === true">
            Disponibles
          </option>
          <option value="no" [selected]="store.filtros().soloDisponibles === false">
            Pausados
          </option>
        </select>
      }
      @if (hayFiltros()) {
        <button type="button" class="btn" (click)="limpiar()">Limpiar filtros</button>
      }
    </form>
    @if (store.filtros().sucursalId) {
      <p id="nota-disponibilidad" class="subtle mb-3">
        «Pausados» muestra solo los productos que la sede pausó; los «No ofrecidos» no aparecen ahí.
      </p>
    } @else if (store.puedePausar()) {
      <p class="subtle mb-3">Elige una sede para pausar o habilitar productos en ella.</p>
    }

    @if (store.listaStatus() === 'error') {
      <div class="empty" role="alert">
        <h3>No pudimos actualizar los productos</h3>
        <p>{{ store.listaError() }}</p>
        <button type="button" class="btn primary" (click)="reintentar()">Reintentar</button>
      </div>
    } @else if (store.listaStatus() === 'loading' && filas().length === 0) {
      <p class="empty" role="status">Buscando productos…</p>
    } @else if (filas().length === 0) {
      <div class="empty-state" role="status">
        @if (hayFiltros()) {
          <h2>No encontramos productos con esos filtros</h2>
          <p>Prueba con otra búsqueda o quita algún filtro.</p>
          <button type="button" class="btn" (click)="limpiar()">Limpiar filtros</button>
        } @else {
          <h2>Aún no hay productos</h2>
          <p>Cuando agregues productos a la carta, aparecerán aquí.</p>
          @if (store.puedeAdministrarCarta()) {
            <button type="button" class="btn primary" (click)="nuevo()">+ Nuevo producto</button>
          }
        }
      </div>
    } @else {
      @if (store.listaStatus() === 'loading') {
        <p class="subtle" role="status">Actualizando la lista…</p>
      }
      <div
        class="panel table-scroll tabla-carta"
        [attr.aria-busy]="store.listaStatus() === 'loading'"
      >
        <table>
          <caption class="sr-only">
            Productos de la carta
          </caption>
          <thead>
            <tr>
              <th scope="col">Producto</th>
              <th scope="col">Categoría</th>
              <th scope="col">Tipo</th>
              <th scope="col">Precio</th>
              <th scope="col">IGV</th>
              <th scope="col">Estado</th>
              @if (store.filtros().sucursalId) {
                <th scope="col">Disponibilidad</th>
              }
              <th scope="col">Acciones</th>
            </tr>
          </thead>
          <tbody>
            @for (f of filas(); track f.producto.id) {
              <tr>
                <td>
                  <strong>{{ f.producto.nombre }}</strong>
                  <small>{{ f.producto.codigo }}</small>
                </td>
                <td>{{ f.categoria }}</td>
                <td>{{ f.tipo }}</td>
                <td>
                  {{ f.precio }}
                  @if (f.precioGeneral) {
                    <small>
                      <s>{{ f.precioGeneral }}</s> · precio de sede
                    </small>
                  }
                </td>
                <td>
                  {{ f.igv }}
                  @if (f.impuesto) {
                    <small>{{ f.impuesto }}</small>
                  }
                </td>
                <td>
                  <span class="pill" [class.green]="f.producto.es_activo">{{
                    f.producto.es_activo ? 'Activo' : 'De baja'
                  }}</span>
                  @if (!f.producto.es_activo && f.producto.motivo_eliminacion) {
                    <small>Motivo: {{ f.producto.motivo_eliminacion }}</small>
                  }
                </td>
                @if (store.filtros().sucursalId) {
                  <td>
                    @if (f.disponibilidad; as d) {
                      <span
                        class="pill"
                        [class.green]="d === 'disponible'"
                        [class.orange]="d === 'pausado'"
                        >{{ etiqueta(d) }}</span
                      >
                    }
                    @if (f.estacion) {
                      <small>Estación: {{ f.estacion }}</small>
                    }
                  </td>
                }
                <td>
                  <div class="inline-actions">
                    @if (f.producto.es_activo) {
                      @if (store.puedeAdministrarCarta()) {
                        <button
                          type="button"
                          class="btn"
                          [attr.aria-label]="'Editar producto ' + f.producto.nombre"
                          (click)="editar(f.producto)"
                        >
                          Editar
                        </button>
                      }
                      @if (store.puedePausar()) {
                        <button
                          type="button"
                          class="btn"
                          [attr.aria-label]="textoAlternar(f) + ' ' + f.producto.nombre"
                          [disabled]="!store.filtros().sucursalId || store.busyId() !== null"
                          [attr.title]="
                            store.filtros().sucursalId
                              ? null
                              : 'Elige una sede para usar esta acción'
                          "
                          (click)="alternar(f)"
                        >
                          {{ f.disponibilidad === 'disponible' ? 'Pausar' : 'Habilitar en sede' }}
                        </button>
                      }
                      @if (store.puedeAdministrarCarta() && store.filtros().sucursalId) {
                        <button
                          type="button"
                          class="btn"
                          [attr.aria-label]="'Ajustes de sede del producto ' + f.producto.nombre"
                          (click)="ajustes(f.producto)"
                        >
                          Ajustes de sede
                        </button>
                      }
                      @if (store.puedeAdministrarCarta()) {
                        <button
                          type="button"
                          class="btn danger"
                          [attr.aria-label]="'Dar de baja producto ' + f.producto.nombre"
                          (click)="baja(f.producto)"
                        >
                          Dar de baja
                        </button>
                      }
                    } @else if (store.puedeAdministrarCarta()) {
                      <button
                        type="button"
                        class="btn"
                        [attr.aria-label]="'Restaurar producto ' + f.producto.nombre"
                        [disabled]="store.busyId() !== null"
                        (click)="restaurar(f.producto)"
                      >
                        Restaurar
                      </button>
                    }
                  </div>
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>
      <div class="section-toolbar mt-4">
        <span class="subtle" role="status">Mostrando {{ filas().length }}</span>
        @if (store.hayMas()) {
          <button
            type="button"
            class="btn"
            [disabled]="store.cargandoMas()"
            (click)="store.cargarMas()"
          >
            {{ store.cargandoMas() ? 'Cargando…' : 'Cargar más' }}
          </button>
        }
      </div>
    }
  `,
})
export class ProductosSection {
  readonly store = inject(CartaStore);
  private readonly dialogs = inject(Dialogs);

  /**
   * La búsqueda de texto se envía al pulsar Enter o «Buscar»: el repo no tiene un mecanismo de
   * debounce y así se evita una petición por tecla (y temporizadores en las pruebas).
   */
  readonly borrador = signal(this.store.filtros().q);

  readonly hayFiltros = computed(() => {
    const f = this.store.filtros();
    return (
      f.q.trim() !== '' ||
      f.categoriaId !== null ||
      f.estado !== 'activos' ||
      f.soloDisponibles !== null
    );
  });

  readonly filas = computed<Fila[]>(() => {
    const categorias = new Map(this.store.categorias().map((c) => [c.id, c.nombre]));
    const impuestos = new Map(this.store.impuestos().map((i) => [i.id, i.nombre]));
    const estaciones = new Map(this.store.estaciones().map((e) => [e.id, e.nombre]));
    return this.store.productos().map((p): Fila => {
      const sede = p.en_sucursal;
      const diferente =
        !!sede?.precio_sucursal &&
        normalizarDecimal(sede.precio_sucursal) !== normalizarDecimal(p.precio_venta);
      return {
        producto: p,
        categoria: categorias.get(p.categoria_id) ?? '—',
        tipo: etiquetaTipo(p.tipo),
        precio: formatearSoles(sede ? sede.precio_efectivo : p.precio_venta),
        precioGeneral: diferente ? formatearSoles(p.precio_venta) : null,
        igv: etiquetaAfectacion(p.afectacion_igv),
        impuesto: p.impuesto_id ? (impuestos.get(p.impuesto_id) ?? null) : null,
        disponibilidad:
          sede === undefined
            ? null
            : sede === null
              ? 'no_ofrecido'
              : sede.disponible
                ? 'disponible'
                : 'pausado',
        estacion: sede?.estacion_id ? (estaciones.get(sede.estacion_id) ?? null) : null,
      };
    });
  });

  etiqueta(d: Disponibilidad): string {
    return ETIQUETA_DISPONIBILIDAD[d];
  }

  textoAlternar(f: Fila): string {
    return f.disponibilidad === 'disponible' ? 'Pausar producto' : 'Habilitar en sede el producto';
  }

  nombreSede(id: string): string {
    return this.store.nombreSede(id);
  }

  valor(evento: Event): string {
    return (evento.target as HTMLSelectElement).value;
  }

  // --- Filtros: cada cambio vuelve a pedir la lista al servidor -----------------------------

  cambiar(cambios: Partial<FiltrosCarta>): void {
    void this.store.cambiarFiltros(cambios);
  }

  cambiarDisponibilidad(evento: Event): void {
    const v = this.valor(evento);
    this.cambiar({ soloDisponibles: v === 'si' ? true : v === 'no' ? false : null });
  }

  buscar(evento: Event): void {
    evento.preventDefault();
    this.cambiar({ q: this.borrador().trim() });
  }

  limpiar(): void {
    this.borrador.set('');
    this.cambiar({ q: '', categoriaId: null, estado: 'activos', soloDisponibles: null });
  }

  reintentar(): void {
    void this.store.recargarProductos();
  }

  // --- Acciones por fila --------------------------------------------------------------------

  alternar(f: Fila): void {
    if (f.disponibilidad === 'disponible') void this.store.pausarEnSede(f.producto);
    else void this.store.habilitarEnSede(f.producto);
  }

  restaurar(p: Producto): void {
    void this.store.restaurarProducto(p, newIdempotencyKey());
  }

  nuevo(): void {
    this.formProducto();
  }

  editar(p: Producto): void {
    this.formProducto(p);
  }

  baja(p: Producto): void {
    const key = newIdempotencyKey();
    this.dialogs.form({
      title: 'Dar de baja: ' + p.nombre,
      description:
        'Saldrá de la carta de todas las sedes. No afecta los consumos ya registrados y podrás restaurarlo cuando quieras.',
      danger: true,
      submit: 'Confirmar baja',
      fields: [{ key: 'motivo', label: 'Motivo de la baja', type: 'textarea' }],
      commit: async (v): Resultado => {
        if (v['motivo'].trim().length > 2000) {
          return { field: 'motivo', message: 'El motivo admite hasta 2000 caracteres.' };
        }
        return this.store.darDeBajaProducto(p, v['motivo'], key);
      },
    });
  }

  ajustes(p: Producto): void {
    const sede = this.store.filtros().sucursalId;
    if (!sede) return;
    const actual = p.en_sucursal ?? null;
    const estaciones = this.store.estaciones().map((e) => ({ value: e.id, label: e.nombre }));
    if (actual?.estacion_id && !estaciones.some((e) => e.value === actual.estacion_id)) {
      estaciones.push({ value: actual.estacion_id, label: 'Estación actual' });
    }
    this.dialogs.form({
      title: 'Ajustes de sede: ' + p.nombre,
      description: `Precio y estación de preparación en ${this.nombreSede(sede)}. Precio general: ${formatearSoles(p.precio_venta)}.`,
      submit: 'Guardar ajustes',
      fields: [
        {
          key: 'precio',
          label: 'Precio de esta sede (S/)',
          required: false,
          value: actual?.precio_sucursal ?? '',
          hint: 'Déjalo vacío para usar el precio general.',
        },
        {
          key: 'estacionId',
          label: 'Estación de preparación',
          type: 'select',
          required: false,
          value: actual?.estacion_id ?? '',
          options: [{ value: '', label: 'Sin estación' }, ...estaciones],
        },
      ],
      commit: async (v): Resultado => {
        const precio = v['precio'].trim();
        if (precio !== '' && !precioValido(precio)) {
          return { field: 'precio', message: 'Escribe un precio en soles, por ejemplo 25.50.' };
        }
        return this.store.ajustarSede(p, { precio, estacionId: v['estacionId'] });
      },
    });
  }

  private formProducto(p?: Producto): void {
    const key = newIdempotencyKey();
    const tipoFijo = p?.tipo === 'REVENTA';
    const categorias = this.store
      .categoriasActivas()
      .map((c) => ({ value: c.id, label: c.nombre }));
    if (p && !categorias.some((c) => c.value === p.categoria_id)) {
      const actual = this.store.categorias().find((c) => c.id === p.categoria_id);
      categorias.push({
        value: p.categoria_id,
        label: `${actual?.nombre ?? 'Categoría actual'} (de baja)`,
      });
    }
    const impuestos = this.store.impuestos().map((i) => ({ value: i.id, label: i.nombre }));
    if (p?.impuesto_id && !impuestos.some((i) => i.value === p.impuesto_id)) {
      impuestos.push({ value: p.impuesto_id, label: 'Impuesto actual' });
    }
    const tipos = TIPOS_PRODUCTO.filter((t) => t.ofrecido).map((t) => ({
      value: t.value,
      label: t.label,
    }));

    const campos: Field[] = [
      {
        key: 'categoria_id',
        label: 'Categoría',
        type: 'select',
        value: p?.categoria_id,
        options: categorias,
      },
    ];
    // El código no se edita después del alta: identifica al producto en su historial.
    if (!p) {
      campos.push({
        key: 'codigo',
        label: 'Código',
        hint: 'Único en el restaurante. Se guarda en mayúsculas.',
      });
    }
    campos.push(
      { key: 'nombre', label: 'Nombre del producto', value: p?.nombre },
      {
        key: 'descripcion',
        label: 'Descripción',
        type: 'textarea',
        required: false,
        value: p?.descripcion ?? '',
      },
    );
    // Reventa necesita un artículo de inventario (llega con inventario): su tipo no se cambia.
    if (!tipoFijo) {
      campos.push({ key: 'tipo', label: 'Tipo', type: 'select', value: p?.tipo, options: tipos });
    }
    campos.push(
      {
        key: 'precio_venta',
        label: 'Precio general (S/)',
        value: p?.precio_venta ?? '',
        hint: 'En soles, hasta 2 decimales. Ejemplo: 25.50.',
      },
      {
        key: 'factor_consumo',
        label: 'Factor de consumo',
        value: p ? normalizarDecimal(p.factor_consumo) : '1',
        hint: 'Unidades de insumo que consume cada venta. Déjalo en 1 si no lo sabes; los servicios lo ignoran.',
      },
      {
        key: 'afectacion_igv',
        label: 'Afectación al IGV',
        type: 'select',
        value: p?.afectacion_igv ?? '10',
        options: [...OPCIONES_AFECTACION],
      },
      {
        key: 'impuesto_id',
        label: 'Impuesto',
        type: 'select',
        required: false,
        value: p ? (p.impuesto_id ?? '') : impuestos.length === 1 ? impuestos[0].value : '',
        options: [{ value: '', label: 'Sin impuesto' }, ...impuestos],
        hint: 'Obligatorio si el producto es gravado; no aplica a exonerados ni inafectos.',
      },
      {
        key: 'imagen_url',
        label: 'Dirección de la imagen',
        required: false,
        value: p?.imagen_url ?? '',
        hint: 'Opcional. Pega el enlace de una imagen.',
      },
    );

    this.dialogs.form({
      title: p ? 'Editar producto' : 'Nuevo producto',
      description: p
        ? `Código ${p.codigo}. Los cambios no alteran las ventas ya registradas.`
        : categorias.length === 0
          ? 'Primero crea una categoría activa en la pestaña Categorías.'
          : 'Se agrega a la carta de todas las sedes; luego puedes ajustar precio y estación por sede.',
      submit: 'Guardar producto',
      fields: campos,
      commit: async (v): Resultado => {
        const precio = v['precio_venta'].trim();
        if (!precioValido(precio)) {
          return {
            field: 'precio_venta',
            message: 'Escribe un precio en soles, por ejemplo 25.50.',
          };
        }
        if (!factorValido(v['factor_consumo'])) {
          return {
            field: 'factor_consumo',
            message: 'Escribe un número mayor que 0 (hasta 6 decimales), por ejemplo 1 o 0.5.',
          };
        }
        const impuesto = v['impuesto_id'].trim();
        if (v['afectacion_igv'] === '10' && impuesto === '') {
          return {
            field: 'impuesto_id',
            message: 'Elige el impuesto: es obligatorio en un producto gravado.',
          };
        }
        if (v['afectacion_igv'] !== '10' && impuesto !== '') {
          return {
            field: 'impuesto_id',
            message: 'Déjalo en «Sin impuesto»: no aplica a un producto exonerado o inafecto.',
          };
        }
        const valores: ValoresProducto = {
          categoria_id: v['categoria_id'],
          codigo: p ? p.codigo : v['codigo'].trim().toUpperCase(),
          nombre: v['nombre'],
          descripcion: v['descripcion'],
          tipo: tipoFijo ? 'REVENTA' : v['tipo'],
          factor_consumo: v['factor_consumo'],
          precio_venta: precio,
          afectacion_igv: v['afectacion_igv'],
          impuesto_id: impuesto,
          imagen_url: v['imagen_url'],
        };
        return p ? this.store.editarProducto(p, valores) : this.store.crearProducto(valores, key);
      },
    });
  }
}
