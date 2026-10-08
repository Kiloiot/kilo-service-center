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
import {
  FILTERS_STORAGE_DEFAULT_SCOPE,
  PAGINATION,
  SORT_DIRECTION,
  STORAGE_KEYS,
} from "@constants/app";
import { APP_ERRORS, LOG_MESSAGES } from "@constants/messages";

import type {
  BaseStationFiltersState,
  EndpointFiltersState,
  FiltersAction,
  FiltersContextValue,
  FilterScope,
  FiltersState,
  PaginationState,
  SortState,
} from "./types";

// Default filter states
const defaultBaseStationFilters: BaseStationFiltersState = {
  search: "",
  status: [],
  sort: { field: "lastSeen", direction: SORT_DIRECTION.DESC },
  pagination: { page: 0, pageSize: PAGINATION.DEFAULT_PAGE_SIZE },
};

const defaultEndpointFilters: EndpointFiltersState = {
  search: "",
  attachState: [],
  activity: [],
  sort: { field: "lastSeen", direction: SORT_DIRECTION.DESC },
  pagination: { page: 0, pageSize: PAGINATION.DEFAULT_PAGE_SIZE },
};

// Create initial state
function createInitialState(): FiltersState {
  return {
    baseStations: { ...defaultBaseStationFilters },
    endpoints: { ...defaultEndpointFilters },
  };
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
  const namespace = orgId || FILTERS_STORAGE_DEFAULT_SCOPE;
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
      };
    }
  } catch (error) {
    logger.error(LOG_MESSAGES.FILTERS_LOAD_FAILED, error);
  }
  return null;
}

// Save state to storage
function saveToStorage(storageKey: string, state: FiltersState): void {
  try {
    storageService.setItem(storageKey, JSON.stringify(state));
  } catch (error) {
    logger.error(LOG_MESSAGES.FILTERS_SAVE_FAILED, error);
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
      setFilter: (scope: FilterScope, key: string, value: unknown) =>
        dispatch({ type: "SET_FILTER", scope, key, value }),
      setSearch: (scope: FilterScope, search: string) =>
        dispatch({ type: "SET_SEARCH", scope, search }),
      setPagination: (scope: FilterScope, pagination: PaginationState) =>
        dispatch({ type: "SET_PAGINATION", scope, pagination }),
      setSort: (scope: FilterScope, sort: SortState) =>
        dispatch({ type: "SET_SORT", scope, sort }),
      resetScope: (scope: FilterScope) =>
        dispatch({ type: "RESET_SCOPE", scope }),
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
    throw new Error(APP_ERRORS.FILTERS_CONTEXT_REQUIRED);
  }
  return context;
}
