import { collect, Pagina } from './collect';

describe('Carga de páginas', () => {
  it('sigue el cursor hasta que no hay más páginas', async () => {
    const fetchPage = vi
      .fn<(cursor?: string) => Promise<Pagina<number>>>()
      .mockResolvedValueOnce({ items: [1, 2], nextCursor: 'c1' })
      .mockResolvedValueOnce({ items: [3], nextCursor: null });
    expect(await collect(fetchPage)).toEqual([1, 2, 3]);
    expect(fetchPage).toHaveBeenNthCalledWith(1, undefined);
    expect(fetchPage).toHaveBeenNthCalledWith(2, 'c1');
  });

  it('se detiene a las 20 páginas aunque el servidor siga devolviendo cursor', async () => {
    const fetchPage = vi.fn(async () => ({ items: [1], nextCursor: 'siempre' }));
    expect(await collect(fetchPage)).toHaveLength(20);
    expect(fetchPage).toHaveBeenCalledTimes(20);
  });
});
