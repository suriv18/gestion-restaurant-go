import { Dialog } from '@angular/cdk/dialog';
import { SplitDialog } from './split-dialog';
import { Injectable, inject } from '@angular/core';
import { Router } from '@angular/router';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { Dialogs, Field, options } from '../../shared/ui/form-dialog';
import { toCents } from '../../shared/ui/money';
import { money } from '../../core/demo/engine/model.js';
import { Order, PaymentMethod } from '../../shared/models/restaurant';
@Injectable({ providedIn: 'root' })
export class OrderActions {
  private readonly store = inject(RestaurantStore);
  private readonly dialogs = inject(Dialogs);
  private readonly dialog = inject(Dialog);
  private readonly router = inject(Router);
  newOrder(type: Order['type'] = 'SALON', table?: number): void {
    const tables = this.store
      .enabledTables()
      .filter((t) => !this.store.orders().some((o) => o.table === t.id));
    if (type === 'SALON' && !tables.length) {
      this.store.notify('Todas las mesas están ocupadas.');
      return;
    }
    const fields: Field[] = [
      { key: 'customer', label: 'Cliente', required: type === 'DELIVERY' },
      { key: 'guests', label: 'Comensales', type: 'number', value: 1, min: 1, max: 99, step: '1' },
    ];
    if (type === 'SALON')
      fields.push({
        key: 'table',
        label: 'Mesa',
        type: 'select',
        value: table,
        options: tables.map((t) => ({ value: t.id, label: `${t.name} · ${t.capacity} personas` })),
      });
    if (type === 'DELIVERY')
      fields.push(
        { key: 'phone', label: 'Teléfono', type: 'tel' },
        { key: 'address', label: 'Dirección de entrega', type: 'textarea' },
      );
    this.dialogs.form({
      title: 'Nuevo pedido',
      description:
        type === 'SALON'
          ? 'Recibe a tus clientes y comienza una buena experiencia.'
          : type === 'DELIVERY'
            ? 'Un nuevo pedido a domicilio.'
            : 'Prepara un pedido para recoger.',
      fields,
      submit: 'Abrir pedido',
      commit: (v) => {
        const id = this.store.execute('newOrder', {
          type,
          table: Number(v['table']),
          customer: v['customer'],
          guests: Number(v['guests']),
          phone: v['phone'],
          address: v['address'],
        });
        void this.router.navigate(['/salon'], { queryParams: { pedido: id } });
      },
    });
  }
  add(order: Order): void {
    const products = this.store.state().products.filter((p) => p.es_activo && p.disponible);
    if (!products.length) {
      this.store.notify('No hay productos disponibles en la carta.');
      return;
    }
    this.dialogs.form({
      title: 'Agregar productos',
      description: `${this.store.tableName(order)} · Pedido #${order.id}`,
      submit: 'Agregar al pedido',
      fields: [
        {
          key: 'product',
          label: 'Producto',
          type: 'select',
          options: products.map((p) => ({ value: p.id, label: `${p.name} · ${money(p.price)}` })),
        },
        { key: 'qty', label: 'Cantidad', type: 'number', value: 1, min: 1, max: 99, step: '1' },
        {
          key: 'account',
          label: 'Cuenta',
          type: 'select',
          options: order.accounts.map((a) => ({ value: a, label: `Cuenta ${a}` })),
        },
        {
          key: 'note',
          label: 'Indicaciones para cocina',
          type: 'textarea',
          required: false,
          hint: 'Por ejemplo: sin cebolla, término medio.',
        },
      ],
      commit: (v) =>
        this.store.execute('addLine', {
          order: order.id,
          product: Number(v['product']),
          qty: Number(v['qty']),
          account: Number(v['account']),
          note: v['note'],
        }),
    });
  }
  pay(order: Order, account = 1): void {
    this.dialogs.form({
      title: 'Cobrar pedido',
      description: `Pedido #${order.id} · Pendiente ${money(this.store.balance(order))}. Los pagos pueden ser parciales o mixtos.`,
      submit: 'Registrar cobro',
      fields: [
        {
          key: 'account',
          label: 'Cuenta',
          type: 'select',
          value: account,
          options: order.accounts.map((a) => ({
            value: a,
            label: `Cuenta ${a} · ${money(this.store.balance(order, a))}`,
          })),
        },
        {
          key: 'method',
          label: 'Método de pago',
          type: 'select',
          options: options(['EFECTIVO', 'YAPE', 'PLIN', 'TARJETA', 'TRANSFERENCIA']),
        },
        {
          key: 'amount',
          label: 'Importe a aplicar (S/)',
          value: (this.store.balance(order, account) / 100).toFixed(2),
          hint: 'El importe no debe superar el saldo de la cuenta seleccionada.',
        },
        {
          key: 'received',
          label: 'Efectivo recibido (S/)',
          required: false,
          hint: 'Solo efectivo. El excedente se devuelve como vuelto.',
        },
        { key: 'reference', label: 'Referencia del pago digital', required: false },
      ],
      commit: (v) => {
        const amount = toCents(v['amount']);
        const received = v['received'] ? toCents(v['received']) : amount;
        this.store.execute('pay', {
          order: order.id,
          account: Number(v['account']),
          method: v['method'] as PaymentMethod,
          amount,
          received,
          reference: v['reference'],
        });
        this.store.notify(
          v['method'] === 'EFECTIVO'
            ? `Cobro registrado. Vuelto: ${money(received - amount)}.`
            : 'Cobro de demostración registrado.',
        );
      },
    });
  }
  split(order: Order): void {
    this.dialog.open(SplitDialog, {
      data: order,
      panelClass: 'app-dialog',
      ariaLabelledBy: 'split-title',
      ariaModal: true,
    });
  }
  cancel(order: Order): void {
    this.dialogs.form({
      title: `Cancelar pedido #${order.id}`,
      description: 'Los consumos de inventario ya registrados se conservan en el kardex.',
      danger: true,
      submit: 'Cancelar pedido',
      fields: [{ key: 'reason', label: 'Motivo de cancelación', type: 'textarea' }],
      commit: (v) => this.store.execute('cancelOrder', { order: order.id, reason: v['reason'] }),
    });
  }
}
