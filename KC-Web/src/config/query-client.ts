/**
 * React Query Client Configuration
 *
 * Centralized QueryClient with defaults for caching, retries, and error handling.
 */

import { QueryClient } from '@tanstack/react-query';

import { TIMING } from '@constants/app';

/**
 * Exponential backoff delay for retries
 * Base delay of 1 second with exponential increase
 */
function exponentialBackoff(attemptIndex: number): number {
  return Math.min(1000 * 2 ** attemptIndex, 30000);
}

/**
 * Default QueryClient with shared configuration
 *
 * Defaults:
 * - staleTime: 30s (matches LIST_REFRESH from constants)
 * - gcTime: 5 min garbage collection
 * - retry: 2 attempts with exponential backoff
 * - refetchOnWindowFocus: true for fresh data when user returns
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: TIMING.LIST_REFRESH * 1000,
      gcTime: 5 * 60 * 1000, // 5 minute garbage collection
      retry: 2,
      retryDelay: exponentialBackoff,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
    },
    mutations: {
      retry: 1,
      retryDelay: exponentialBackoff,
    },
  },
});

/**
 * Reset query client state (useful for logout/testing)
 */
export function resetQueryClient(): void {
  queryClient.clear();
}
