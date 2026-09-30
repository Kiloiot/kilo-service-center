import type { SortDirection } from "@constants/app";
/**
 * Filter Context Types
 *
 * Type definitions for filter state management.
 */

// Pagination state
export interface PaginationState {
  page: number;
  pageSize: number;
}

// Sort state
export interface SortState {
  field: string;
  direction: SortDirection;
}

// Base station filters
export interface BaseStationFiltersState {
  search: string;
  status: string[];
  sort: SortState;
  pagination: PaginationState;
}

// Endpoint filters
export interface EndpointFiltersState {
  search: string;
  attachState: string[];
  activity: string[];
  sort: SortState;
  pagination: PaginationState;
}

// Filter scopes
export type FilterScope = "baseStations" | "endpoints";

// Complete filters state
export interface FiltersState {
  baseStations: BaseStationFiltersState;
  endpoints: EndpointFiltersState;
}

// Filter actions
export type FiltersAction =
  | { type: "SET_FILTER"; scope: FilterScope; key: string; value: unknown }
  | { type: "SET_SEARCH"; scope: FilterScope; search: string }
  | { type: "SET_PAGINATION"; scope: FilterScope; pagination: PaginationState }
  | { type: "SET_SORT"; scope: FilterScope; sort: SortState }
  | { type: "RESET_SCOPE"; scope: FilterScope }
  | { type: "RESET_ALL" }
  | { type: "LOAD_STATE"; state: FiltersState };

// Context value type
export interface FiltersContextValue {
  state: FiltersState;
  // Convenience methods
  setFilter: (scope: FilterScope, key: string, value: unknown) => void;
  setSearch: (scope: FilterScope, search: string) => void;
  setPagination: (scope: FilterScope, pagination: PaginationState) => void;
  setSort: (scope: FilterScope, sort: SortState) => void;
  resetScope: (scope: FilterScope) => void;
}
