/**
 * Filters Context
 *
 * Centralized filter state management with org-scoped persistence.
 * Persists filter state per organization to localStorage.
 */

import React, {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useReducer,
} from "react";

import { useOrganization } from "@contexts/OrganizationContext";
import { logger } from "@utils/logger";
import { storageService } from "@utils/storage";
import { PAGINATION, STORAGE_KEYS } from "@constants/app";

import type {
  BaseStationFiltersState,
  DateRange,
  EndpointFiltersState,
  FiltersAction,
  FiltersContextValue,
  FilterScope,
  FiltersState,
  PaginationState,
  SavedView,
  SortState,
} from "./types";

// Default filter states
const defaultBaseStationFilters: BaseStationFiltersState = {
  search: "",
  status: [],
  sort: { field: "lastSeen", direction: "desc" },
  pagination: { page: 0, pageSize: PAGINATION.DEFAULT_PAGE_SIZE },
};

const defaultEndpointFilters: EndpointFiltersState = {
  search: "",
  attachState: [],
  bidirectional: null,
  sort: { field: "lastSeen", direction: "desc" },
  pagination: { page: 0, pageSize: PAGINATION.DEFAULT_PAGE_SIZE },
};

// Create initial state
function createInitialState(): FiltersState {
  return {
    baseStations: { ...defaultBaseStationFilters },
    endpoints: { ...defaultEndpointFilters },
    savedViews: [],
  };
}

// Generate unique ID for saved views
function generateViewId(): string {
  return `view_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

// Reducer function
function filtersReducer(
  state: FiltersState,
  action: FiltersAction,
): FiltersState {
  switch (action.type) {
    case "SET_FILTER":
      // Reset pagination page to 0 when filters change
      return {
        ...state,
        [action.scope]: {
          ...state[action.scope],
          [action.key]: action.value,
          pagination: { ...state[action.scope].pagination, page: 0 },
        },
      };

    case "SET_SEARCH":
      // Reset pagination page to 0 when search changes
      return {
        ...state,
        [action.scope]: {
          ...state[action.scope],
          search: action.search,
          pagination: { ...state[action.scope].pagination, page: 0 },
        },
      };

    case "SET_PAGINATION":
      return {
        ...state,
        [action.scope]: {
          ...state[action.scope],
          pagination: action.pagination,
        },
      };

    case "SET_SORT":
      // Reset pagination page to 0 when sort changes
      return {
        ...state,
        [action.scope]: {
          ...state[action.scope],
          sort: action.sort,
          pagination: { ...state[action.scope].pagination, page: 0 },
        },
      };

    case "SET_DATE_RANGE":
      return {
        ...state,
        [action.scope]: {
          ...state[action.scope],
          dateRange: action.dateRange,
        },
      };

    case "RESET_SCOPE": {
      const defaults: Record<FilterScope, object> = {
        baseStations: defaultBaseStationFilters,
        endpoints: defaultEndpointFilters,
      };
      return {
        ...state,
        [action.scope]: { ...defaults[action.scope] },
      };
    }

    case "RESET_ALL":
      return {
        ...createInitialState(),
        savedViews: state.savedViews, // Preserve saved views
      };

    case "SAVE_VIEW": {
      const view: SavedView = {
        id: generateViewId(),
        name: action.name,
        scope: action.scope,
        filters: state[action.scope],
        createdAt: new Date().toISOString(),
      };
      return {
        ...state,
        savedViews: [...state.savedViews, view],
      };
    }

    case "LOAD_VIEW": {
      const view = state.savedViews.find((v) => v.id === action.viewId);
      if (!view) return state;
      return {
        ...state,
        [view.scope]: view.filters as FiltersState[FilterScope],
      };
    }

    case "DELETE_VIEW":
      return {
        ...state,
        savedViews: state.savedViews.filter((v) => v.id !== action.viewId),
      };

    case "LOAD_STATE":
      return action.state;

    default:
      return state;
  }
}

// Context
const FiltersContext = createContext<FiltersContextValue | undefined>(
  undefined,
);

// Storage key builder with org scope
function buildStorageKey(orgId: string | null): string {
  const namespace = orgId || "default";
  return `${STORAGE_KEYS.FILTERS}-${namespace}`;
}

// Load state from storage
function loadFromStorage(storageKey: string): FiltersState | null {
  try {
    const stored = storageService.getItem(storageKey);
    if (stored) {
      const parsed = JSON.parse(stored) as Partial<FiltersState>;
      return {
        baseStations: {
          ...defaultBaseStationFilters,
          ...parsed.baseStations,
          pagination:
            parsed.baseStations?.pagination ??
            defaultBaseStationFilters.pagination,
        },
        endpoints: {
          ...defaultEndpointFilters,
          ...parsed.endpoints,
          pagination:
            parsed.endpoints?.pagination ?? defaultEndpointFilters.pagination,
        },
        savedViews: Array.isArray(parsed.savedViews) ? parsed.savedViews : [],
      };
    }
  } catch (error) {
    logger.error("Failed to load filters from storage:", error);
  }
  return null;
}

// Save state to storage
function saveToStorage(storageKey: string, state: FiltersState): void {
  try {
    storageService.setItem(storageKey, JSON.stringify(state));
  } catch (error) {
    logger.error("Failed to save filters to storage:", error);
  }
}

// Provider props
interface FiltersProviderProps {
  children: React.ReactNode;
}

/**
 * Filters Provider
 *
 * Wraps the application to provide filter state context.
 * Persists state per organization to localStorage.
 */
export function FiltersProvider({ children }: FiltersProviderProps) {
  const { organizationId } = useOrganization();

  // Build org-scoped storage key
  const storageKey = useMemo(
    () => buildStorageKey(organizationId),
    [organizationId],
  );

  // Initialize state from storage or defaults
  const [state, dispatch] = useReducer(filtersReducer, storageKey, (key) => {
    const loaded = loadFromStorage(key);
    return loaded ?? createInitialState();
  });

  // Reload state when organization changes
  useEffect(() => {
    const loaded = loadFromStorage(storageKey);
    if (loaded) {
      dispatch({ type: "LOAD_STATE", state: loaded });
    } else {
      dispatch({ type: "RESET_ALL" });
    }
  }, [storageKey]);

  // Persist state changes to storage
  useEffect(() => {
    saveToStorage(storageKey, state);
  }, [storageKey, state]);

  // Convenience methods
  const contextValue = useMemo<FiltersContextValue>(
    () => ({
      state,
      dispatch,
      setFilter: (scope: FilterScope, key: string, value: unknown) =>
        dispatch({ type: "SET_FILTER", scope, key, value }),
      setSearch: (scope: FilterScope, search: string) =>
        dispatch({ type: "SET_SEARCH", scope, search }),
      setPagination: (scope: FilterScope, pagination: PaginationState) =>
        dispatch({ type: "SET_PAGINATION", scope, pagination }),
      setSort: (scope: FilterScope, sort: SortState) =>
        dispatch({ type: "SET_SORT", scope, sort }),
      setDateRange: (scope: FilterScope, dateRange: DateRange) =>
        dispatch({ type: "SET_DATE_RANGE", scope, dateRange }),
      resetScope: (scope: FilterScope) =>
        dispatch({ type: "RESET_SCOPE", scope }),
      resetAll: () => dispatch({ type: "RESET_ALL" }),
      saveView: (name: string, scope: FilterScope) =>
        dispatch({ type: "SAVE_VIEW", name, scope }),
      loadView: (viewId: string) => dispatch({ type: "LOAD_VIEW", viewId }),
      deleteView: (viewId: string) => dispatch({ type: "DELETE_VIEW", viewId }),
    }),
    [state],
  );

  return (
    <FiltersContext.Provider value={contextValue}>
      {children}
    </FiltersContext.Provider>
  );
}

/**
 * Hook to access filters context
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useFilters(): FiltersContextValue {
  const context = useContext(FiltersContext);
  if (!context) {
    throw new Error("useFilters must be used within a FiltersProvider");
  }
  return context;
}
