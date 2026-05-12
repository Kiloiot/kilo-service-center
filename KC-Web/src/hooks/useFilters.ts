/**
 * Scope-Specific Filter Hooks
 *
 * Convenience hooks for accessing filter state per domain.
 * Wraps useFilters() with scope-specific typing and actions.
 */

import { useMemo } from "react";

import type {
  BaseStationFiltersState,
  EndpointFiltersState,
  SortState,
} from "@contexts/filters";
import { useFilters } from "@contexts/filters";

/**
 * Hook for base station filters
 */
export function useBaseStationFilters() {
  const { state, setFilter, setSearch, setSort, setPagination, resetScope } =
    useFilters();

  return useMemo(
    () => ({
      filters: state.baseStations,
      setSearch: (search: string) => setSearch("baseStations", search),
      setStatus: (status: string[]) =>
        setFilter("baseStations", "status", status),
      setSort: (sort: SortState) => setSort("baseStations", sort),
      reset: () => resetScope("baseStations"),
      // Pagination
      pagination: state.baseStations.pagination,
      setPage: (page: number) =>
        setPagination("baseStations", {
          ...state.baseStations.pagination,
          page,
        }),
      setPageSize: (pageSize: number) =>
        setPagination("baseStations", {
          page: 0, // Reset page on size change
          pageSize,
        }),
      // Individual field setters
      updateFilter: <K extends keyof BaseStationFiltersState>(
        key: K,
        value: BaseStationFiltersState[K],
      ) => setFilter("baseStations", key, value),
    }),
    [
      state.baseStations,
      setFilter,
      setSearch,
      setSort,
      setPagination,
      resetScope,
    ],
  );
}

/**
 * Hook for endpoint filters
 */
export function useEndpointFilters() {
  const { state, setFilter, setSearch, setSort, setPagination, resetScope } =
    useFilters();

  return useMemo(
    () => ({
      filters: state.endpoints,
      setSearch: (search: string) => setSearch("endpoints", search),
      setAttachState: (attachState: string[]) =>
        setFilter("endpoints", "attachState", attachState),
      setBidirectional: (bidi: boolean | null) =>
        setFilter("endpoints", "bidirectional", bidi),
      setSort: (sort: SortState) => setSort("endpoints", sort),
      reset: () => resetScope("endpoints"),
      // Pagination
      pagination: state.endpoints.pagination,
      setPage: (page: number) =>
        setPagination("endpoints", {
          ...state.endpoints.pagination,
          page,
        }),
      setPageSize: (pageSize: number) =>
        setPagination("endpoints", {
          page: 0, // Reset page on size change
          pageSize,
        }),
      // Individual field setters
      updateFilter: <K extends keyof EndpointFiltersState>(
        key: K,
        value: EndpointFiltersState[K],
      ) => setFilter("endpoints", key, value),
    }),
    [state.endpoints, setFilter, setSearch, setSort, setPagination, resetScope],
  );
}

/**
 * Hook for saved views management
 */
export function useSavedViews() {
  const { state, saveView, loadView, deleteView } = useFilters();

  return useMemo(
    () => ({
      views: state.savedViews,
      save: saveView,
      load: loadView,
      delete: deleteView,
    }),
    [state.savedViews, saveView, loadView, deleteView],
  );
}
