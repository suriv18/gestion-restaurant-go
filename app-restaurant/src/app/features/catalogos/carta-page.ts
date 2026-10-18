import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Product } from '../../shared/models/restaurant';
import { PageHeader } from '../../shared/ui/page-header';
import { Icon } from '../../shared/ui/icon';
import { MoneyPipe, toCents } from '../../shared/ui/money';
import { Dialogs, options } from '../../shared/ui/form-dialog';
@Component({
  imports: [PageHeader, Icon, MoneyPipe],
  templateUrl: './carta-page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CartaPage {
  readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  readonly query = signal('');
  readonly category = signal('Todos');
  readonly inactive = signal(false);
  readonly categories = ['Todos', 'Fondos', 'Entradas', 'Bebidas', 'Postres', 'Adicionales'];
  readonly products = computed(() =>
    this.store
      .state()
      .products.filter(
        (p) =>
          (this.inactive() || p.es_activo) &&
          (this.category() === 'Todos' || p.category === this.category()) &&
          (p.name + ' ' + p.code)
            .toLocaleLowerCase('es')
            .includes(this.query().toLocaleLowerCase('es')),
      ),
  );
  icon(category: string): string {
    return (
      (
        { Fondos: 'plate', Entradas: 'leaf', Bebidas: 'drink', Postres: 'cake' } as Record<
          string,
          string
        >
      )[category] ?? 'plate'
    );
  }
  edit(p?: Product): void {
    this.dialogs.form({
      title: p ? 'Editar producto' : 'Un nuevo sabor en la carta',
      description: 'Los cambios conservan los nombres y precios de ventas anteriores.',
      submit: 'Guardar producto',
      fields: [
        { key: 'name', label: 'Nombre del producto', value: p?.name },
        { key: 'code', label: 'Código único', value: p?.code },
        { key: 'price', label: 'Precio de venta (S/)', value: p ? (p.price / 100).toFixed(2) : '' },
        {
          key: 'category',
          label: 'Categoría',
          type: 'select',
          value: p?.category,
          options: options(this.categories.slice(1)),
        },
        {
          key: 'station',
          label: 'Estación',
          type: 'select',
          value: p?.station,
          options: options(['Cocina', 'Barra']),
        },
        {
          key: 'type',
          label: 'Tipo',
          type: 'select',
          value: p?.type,
          options: options(['PREPARADO', 'REVENTA', 'ADICIONAL', 'SERVICIO']),
        },
      ],
      commit: (v) =>
        this.store.execute('saveProduct', {
          id: p?.id,
          name: v['name'],
          code: v['code'],
          price: toCents(v['price']),
          category: v['category'],
          station: v['station'],
          type: v['type'],
        }),
    });
  }
  detail(p: Product): void {
    const recipe = this.store.state().recipes[p.id] ?? [];
    this.dialogs.detail({
      title: p.name,
      description: p.code + ' · Receta de referencia para una porción',
      rows: [
        ...recipe.map((r) => {
          const i = this.store.state().ingredients.find((i) => i.id === r.item);
          return { label: i?.name ?? 'Insumo', value: r.qty + ' ' + i?.unit };
        }),
        ...(!recipe.length
          ? [{ label: 'Receta', value: 'Sin receta de referencia. No genera consumo automático.' }]
          : []),
        { label: 'Creado por', value: p.creado_por },
        { label: 'Actualizado', value: new Date(p.actualizado_en).toLocaleString('es-PE') },
        ...(p.motivo_eliminacion ? [{ label: 'Motivo de baja', value: p.motivo_eliminacion }] : []),
      ],
    });
  }
  remove(p: Product): void {
    this.dialogs.form({
      title: 'Dar de baja: ' + p.name,
      description: 'El código y el historial se conservan. Podrás restaurar el producto.',
      submit: 'Confirmar baja',
      danger: true,
      fields: [{ key: 'reason', label: 'Motivo de la baja', type: 'textarea' }],
      commit: (v) =>
        this.store.execute('softDelete', { entity: 'products', id: p.id, reason: v['reason'] }),
    });
  }
}
