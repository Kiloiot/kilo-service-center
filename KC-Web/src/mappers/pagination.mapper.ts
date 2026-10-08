import type { ListPage } from "@api-types/pagination";

/** Maps the items of a page, keeping its counts and token. */
export function mapListPage<T, U>(
  page: ListPage<T>,
  map: (item: T) => U,
): ListPage<U> {
  return { ...page, items: page.items.map(map) };
}
