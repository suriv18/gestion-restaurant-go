import { csvCell } from './auditoria-page';
describe('Exportación de auditoría', () => {
  it.each(['=SUM(A1:A2)', '+cmd', '-cmd', '@SUM(A1)', '  =SUM(A1)'])(
    'neutraliza fórmulas CSV: %s',
    (value) => expect(csvCell(value)).toMatch(/^"'/),
  );
  it('escapa comillas y conserva el contenido', () =>
    expect(csvCell('Dijo "hola"')).toBe('"Dijo ""hola"""'));
});
