import { TestBed } from '@angular/core/testing';
import { DIALOG_DATA, DialogRef } from '@angular/cdk/dialog';
import { FormDialog, FormDialogData } from './form-dialog';
describe('Formulario de operación', () => {
  const close = vi.fn();
  const commit = vi.fn();
  beforeEach(async () => {
    close.mockReset();
    commit.mockReset();
    const data: FormDialogData = {
      title: 'Nuevo cliente',
      description: 'Prueba',
      fields: [
        { key: 'name', label: 'Nombre' },
        { key: 'email', label: 'Correo', type: 'email', required: false },
      ],
      commit,
    };
    await TestBed.configureTestingModule({
      imports: [FormDialog],
      providers: [
        { provide: DIALOG_DATA, useValue: data },
        { provide: DialogRef, useValue: { close } },
      ],
    }).compileComponents();
  });
  it('rechaza campos vacíos y correos inválidos sin ejecutar la acción', async () => {
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.submit();
    await fixture.whenStable();
    expect(commit).not.toHaveBeenCalled();
    expect(fixture.nativeElement.querySelector('[role=alert]').textContent).toContain('Completa');
    fixture.componentInstance.form.setValue({ name: 'Ana', email: 'incorrecto' });
    fixture.componentInstance.submit();
    expect(commit).not.toHaveBeenCalled();
  });
  it('conserva el formulario y muestra el rechazo del dominio', async () => {
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.setValue({ name: 'Ana', email: '' });
    commit.mockImplementation(() => {
      throw new Error('El documento ya existe.');
    });
    fixture.componentInstance.submit();
    await fixture.whenStable();
    expect(close).not.toHaveBeenCalled();
    expect(fixture.componentInstance.form.controls['name'].value).toBe('Ana');
    expect(fixture.nativeElement.querySelector('[role=alert]').textContent).toContain('ya existe');
  });
  it('cierra únicamente después de una operación exitosa', () => {
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.setValue({ name: 'Ana', email: 'ana@example.com' });
    fixture.componentInstance.submit();
    expect(commit).toHaveBeenCalledWith({ name: 'Ana', email: 'ana@example.com' });
    expect(close).toHaveBeenCalledWith(true);
  });

  it('muestra el error de campo bajo el campo y conserva lo escrito', async () => {
    commit.mockResolvedValue({ field: 'name', message: 'Ese nombre ya existe.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    fixture.detectChanges();
    const el = fixture.nativeElement as HTMLElement;
    expect(el.textContent).toContain('Ese nombre ya existe.');
    expect(el.querySelector('#field-name')?.getAttribute('aria-invalid')).toBe('true');
    expect(fixture.componentInstance.form.controls['name'].value).toBe('Terraza');
    expect(close).not.toHaveBeenCalled();
  });

  it('borra el error del campo al editarlo', async () => {
    commit.mockResolvedValue({ field: 'name', message: 'Ese nombre ya existe.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    fixture.detectChanges();
    const el = fixture.nativeElement as HTMLElement;
    expect(el.textContent).toContain('Ese nombre ya existe.');
    expect(el.querySelector('#field-name')?.getAttribute('aria-invalid')).toBe('true');
    fixture.componentInstance.form.controls['name'].setValue('Terraza 2');
    fixture.detectChanges();
    expect(el.textContent).not.toContain('Ese nombre ya existe.');
    expect(el.querySelector('#field-name')?.getAttribute('aria-invalid')).toBe('false');
  });

  it('un error de un campo que no existe cae al aviso general', async () => {
    commit.mockResolvedValue({ field: 'otro', message: 'Dato inválido.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    fixture.detectChanges();
    expect(fixture.componentInstance.error()).toBe('Dato inválido.');
    expect(close).not.toHaveBeenCalled();
  });

  it('un campo heredado como «toString» no cuenta como campo del formulario', async () => {
    commit.mockResolvedValue({ field: 'toString', message: 'Dato inválido.' });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    expect(fixture.componentInstance.error()).toBe('Dato inválido.');
  });

  it('un objeto con field o message que no son texto no se toma como error de campo', async () => {
    commit.mockResolvedValue({ field: 1, message: 2 });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.controls['name'].setValue('Terraza');
    await fixture.componentInstance.submit();
    expect(fixture.componentInstance.error()).toBe('');
    expect(close).toHaveBeenCalledWith(true);
  });
});

describe('Formulario con acción asíncrona', () => {
  const close = vi.fn();
  const commit = vi.fn();

  function create() {
    TestBed.configureTestingModule({
      imports: [FormDialog],
      providers: [
        {
          provide: DIALOG_DATA,
          useValue: {
            title: 'Nueva persona',
            description: 'Prueba',
            fields: [
              { key: 'user', label: 'Usuario' },
              {
                key: 'secret',
                label: 'Contraseña',
                type: 'password',
                autocomplete: 'new-password',
              },
            ],
            commit,
          } satisfies FormDialogData,
        },
        { provide: DialogRef, useValue: { close } },
      ],
    });
    const fixture = TestBed.createComponent(FormDialog);
    fixture.componentInstance.form.setValue({ user: 'ana', secret: 'clave' });
    return fixture;
  }

  beforeEach(() => {
    close.mockReset();
    commit.mockReset();
  });

  it('espera la respuesta y cierra solo si salió bien', async () => {
    commit.mockResolvedValue(undefined);
    const fixture = create();
    const pending = fixture.componentInstance.submit();
    expect(fixture.componentInstance.busy()).toBe(true);
    expect(close).not.toHaveBeenCalled();
    await pending;
    expect(close).toHaveBeenCalledWith(true);
    expect(fixture.componentInstance.busy()).toBe(false);
  });

  it('si el servidor rechaza, muestra el motivo y conserva los datos escritos', async () => {
    commit.mockResolvedValue('Ya existe una persona con ese usuario.');
    const fixture = create();
    await fixture.componentInstance.submit();
    await fixture.whenStable();
    expect(close).not.toHaveBeenCalled();
    expect(fixture.componentInstance.error()).toBe('Ya existe una persona con ese usuario.');
    expect(fixture.componentInstance.form.controls['user'].value).toBe('ana');
    expect(fixture.nativeElement.querySelector('[role=alert]').textContent).toContain('Ya existe');
    expect(fixture.componentInstance.busy()).toBe(false);
  });

  it('si la promesa falla, también lo muestra y no cierra', async () => {
    commit.mockRejectedValue(new Error('Sin conexión.'));
    const fixture = create();
    await fixture.componentInstance.submit();
    expect(close).not.toHaveBeenCalled();
    expect(fixture.componentInstance.error()).toBe('Sin conexión.');
  });

  it('ignora un segundo envío mientras el primero sigue en curso', async () => {
    let finish!: () => void;
    commit.mockReturnValue(new Promise<void>((resolve) => (finish = resolve)));
    const fixture = create();
    const first = fixture.componentInstance.submit();
    await fixture.componentInstance.submit();
    expect(commit).toHaveBeenCalledTimes(1);
    finish();
    await first;
  });

  it('limpia el error anterior al reintentar y el campo de contraseña es de tipo password', async () => {
    commit.mockResolvedValueOnce('Falló.').mockResolvedValueOnce(undefined);
    const fixture = create();
    await fixture.componentInstance.submit();
    expect(fixture.componentInstance.error()).toBe('Falló.');
    await fixture.componentInstance.submit();
    expect(fixture.componentInstance.error()).toBe('');
    expect(close).toHaveBeenCalledTimes(1);
    await fixture.whenStable();
    const secret = fixture.nativeElement.querySelector('#field-secret') as HTMLInputElement;
    expect(secret.type).toBe('password');
    expect(secret.getAttribute('autocomplete')).toBe('new-password');
  });

  it('una acción síncrona con cualquier valor de retorno cierra al instante', () => {
    commit.mockReturnValue(42);
    const fixture = create();
    void fixture.componentInstance.submit();
    expect(close).toHaveBeenCalledWith(true);
  });
});
