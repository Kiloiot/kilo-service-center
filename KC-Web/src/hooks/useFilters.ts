/**
 * Scoped view over the filters context: one hook per list scope with the
 * setters a page needs, pagination included.
 */

import { useMemo } from "react";

import type { FilterScope, FiltersState, SortState } from "@contexts/filters";
import { useFilters } from "@contexts/filters";

export function useScopedFilters<S extends FilterScope>(scope: S) {
  const { state, setFilter, setSearch, setSort, setPagination, resetScope } =
    useFilters();
  const filters = state[scope] as FiltersState[S];

  return useMemo(
    () => ({
      filters,
      setSearch: (search: string) => setSearch(scope, search),
      setSort: (sort: SortState) => setSort(scope, sort),
      setPage: (page: number) =>
        setPagination(scope, { ...filters.pagination, page }),
      setPageSize: (pageSize: number) =>
        setPagination(scope, { page: 0, pageSize }),
      updateFilter: <K extends keyof FiltersState[S]>(
        key: K,
        value: FiltersState[S][K],
      ) => setFilter(scope, key as string, value),
      reset: () => resetScope(scope),
    }),
    [scope, filters, setFilter, setSearch, setSort, setPagination, resetScope],
  );
}
