/**
 * Filter Context Types
 *
 * Type definitions for filter state management.
 */

// Date range type
export interface DateRange {
  start: Date | null;
  end: Date | null;
}

// Pagination state
export interface PaginationState {
  page: number;
  pageSize: number;
}

// Sort state
export interface SortState {
  field: string;
  direction: 'asc' | 'desc';
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
  bidirectional: boolean | null;
  sort: SortState;
  pagination: PaginationState;
}

// Saved view for persistence
export interface SavedView {
  id: string;
  name: string;
  scope: FilterScope;
  filters: unknown;
  createdAt: string;
}

// Filter scopes
export type FilterScope = 'baseStations' | 'endpoints';

// Complete filters state
export interface FiltersState {
  baseStations: BaseStationFiltersState;
  endpoints: EndpointFiltersState;
  savedViews: SavedView[];
}

// Filter actions
export type FiltersAction =
  | { type: 'SET_FILTER'; scope: FilterScope; key: string; value: unknown }
  | { type: 'SET_SEARCH'; scope: FilterScope; search: string }
  | { type: 'SET_PAGINATION'; scope: FilterScope; pagination: PaginationState }
  | { type: 'SET_SORT'; scope: FilterScope; sort: SortState }
  | { type: 'SET_DATE_RANGE'; scope: FilterScope; dateRange: DateRange }
  | { type: 'RESET_SCOPE'; scope: FilterScope }
  | { type: 'RESET_ALL' }
  | { type: 'SAVE_VIEW'; name: string; scope: FilterScope }
  | { type: 'LOAD_VIEW'; viewId: string }
  | { type: 'DELETE_VIEW'; viewId: string }
  | { type: 'LOAD_STATE'; state: FiltersState };

// Context value type
export interface FiltersContextValue {
  state: FiltersState;
  dispatch: React.Dispatch<FiltersAction>;
  // Convenience methods
  setFilter: (scope: FilterScope, key: string, value: unknown) => void;
  setSearch: (scope: FilterScope, search: string) => void;
  setPagination: (scope: FilterScope, pagination: PaginationState) => void;
  setSort: (scope: FilterScope, sort: SortState) => void;
  setDateRange: (scope: FilterScope, dateRange: DateRange) => void;
  resetScope: (scope: FilterScope) => void;
  resetAll: () => void;
  saveView: (name: string, scope: FilterScope) => void;
  loadView: (viewId: string) => void;
  deleteView: (viewId: string) => void;
}
