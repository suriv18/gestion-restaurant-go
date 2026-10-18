import { TestBed } from '@angular/core/testing';
import { DialogRef } from '@angular/cdk/dialog';
import { PurchaseDialog } from './purchase-dialog';
import { RestaurantStore } from '../../core/demo/restaurant-store';
describe('Compra con líneas dinámicas', () => {
  const close = vi.fn();
  beforeEach(async () => {
    close.mockReset();
    localStorage.clear();
    await TestBed.configureTestingModule({
      imports: [PurchaseDialog],
      providers: [{ provide: DialogRef, useValue: { close } }],
    }).compileComponents();
  });
  it('crea un borrador con varias líneas sin alterar el stock hasta recibir', () => {
    const fixture = TestBed.createComponent(PurchaseDialog);
    const c = fixture.componentInstance;
    const store = TestBed.inject(RestaurantStore);
    const initial = store.stock(1);
    c.form.controls.reference.setValue('TEST-COMPRA');
    c.form.controls.lines.at(0).setValue({ item: '1', qty: 2, cost: '28.00' });
    c.addLine();
    c.form.controls.lines.at(1).setValue({ item: '6', qty: 3, cost: '6.00' });
    c.save();
    expect(close).toHaveBeenCalledWith(true);
    const purchase = store.state().purchases.at(-1)!;
    expect(purchase.lines).toHaveLength(2);
    expect(store.stock(1)).toBe(initial);
    store.execute('receivePurchase', { id: purchase.id });
    expect(store.stock(1)).toBe(initial + 2);
  });
  it('rechaza un costo inválido sin perder las líneas', () => {
    const fixture = TestBed.createComponent(PurchaseDialog);
    const c = fixture.componentInstance;
    c.form.controls.reference.setValue('TEST-COMPRA');
    c.form.controls.lines.at(0).setValue({ item: '1', qty: 2, cost: '1.999' });
    c.save();
    expect(close).not.toHaveBeenCalled();
    expect(c.error()).toContain('decimales');
    expect(c.form.controls.lines.length).toBe(1);
  });
});
