/** Envoltorio de listas paginadas por cursor (`GET` de la API). */
export interface Pagina<T> {
  items: T[];
  nextCursor: string | null;
}

const MAX_PAGES = 20;

/** Recorre las páginas por cursor (con tope) y devuelve todos los elementos. */
export async function collect<T>(fetchPage: (cursor?: string) => Promise<Pagina<T>>): Promise<T[]> {
  const all: T[] = [];
  let cursor: string | undefined;
  for (let i = 0; i < MAX_PAGES; i++) {
    const page = await fetchPage(cursor);
    all.push(...page.items);
    if (!page.nextCursor) return all;
    cursor = page.nextCursor;
  }
  return all;
}
