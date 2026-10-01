/**
 * React Query Client Configuration
 *
 * Centralized QueryClient with defaults for caching, retries, and error handling.
 */

import { isApiError } from "@api-types/api";
import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";

import { exponentialBackoffMs } from "@utils/backoff";
import { MS_PER_SECOND, QUERY_DEFAULTS, TIMING } from "@constants/app";
import { queryKeys } from "@config/query-keys";

function queryRetryDelay(attemptIndex: number): number {
  return exponentialBackoffMs(
    attemptIndex,
    QUERY_DEFAULTS.RETRY_BASE_DELAY_MS,
    QUERY_DEFAULTS.RETRY_MAX_DELAY_MS,
  );
}

/**
 * Default QueryClient with shared configuration
 *
 * Defaults:
 * - staleTime: 30s (matches LIST_REFRESH from constants)
 * - gcTime: 5 min garbage collection
 * - retry: 2 attempts with exponential backoff for queries, none for mutations
 * - refetchOnWindowFocus: true for fresh data when user returns
 */
/**
 * A refused call means the roles the UI holds are stale: re-read them so
 * navigation and actions follow the change without a new sign-in.
 */
function refreshRolesWhenRefused(error: unknown): void {
  if (isApiError(error) && error.isForbidden()) {
    void queryClient.invalidateQueries({ queryKey: queryKeys.auth.rolesAll() });
  }
}

export const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: refreshRolesWhenRefused }),
  mutationCache: new MutationCache({ onError: refreshRolesWhenRefused }),
  defaultOptions: {
    queries: {
      staleTime: TIMING.LIST_REFRESH * MS_PER_SECOND,
      gcTime: QUERY_DEFAULTS.GC_TIME_MS,
      retry: QUERY_DEFAULTS.RETRY_COUNT,
      retryDelay: queryRetryDelay,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
    },
    // A replayed write (login, single-use code, certificate issuance) is worse than a visible error.
    mutations: {
      retry: 0,
    },
  },
});

/**
 * Reset query client state (useful for logout/testing)
 */
export function resetQueryClient(): void {
  queryClient.clear();
}
