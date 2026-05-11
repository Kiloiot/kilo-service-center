/**
 * Dashboard Hooks
 *
 * React Query hooks for dashboard data fetching.
 */

import { useQuery } from "@tanstack/react-query";

import { apiService } from "@services/api";
import { SERVER_ACTIVITY_EVENT_TYPES,TIMING } from "@constants/app";
import { queryKeys } from "@config/query-keys";

/**
 * Fetch analytics overview for dashboard
 *
 * Uses DASHBOARD_REFRESH interval for periodic updates
 */
export function useDashboardAnalytics() {
  return useQuery({
    queryKey: queryKeys.dashboard.analytics(),
    queryFn: () => apiService.getAnalyticsOverview(),
    refetchInterval: TIMING.DASHBOARD_REFRESH * 1000,
  });
}

/**
 * Fetch recent events for dashboard with optional category and event-type filtering.
 * @param limit - Maximum number of events to return (default: 10)
 * @param categories - Array of category strings to filter by (e.g., SERVER_ACTIVITY_CATEGORIES)
 * @param pageToken - Page token for server-side pagination
 * @param eventTypes - Array of event type strings to filter by (default: SERVER_ACTIVITY_EVENT_TYPES)
 */
export function useDashboardEvents(
  limit = 10,
  categories?: readonly string[],
  pageToken?: string,
  eventTypes: readonly string[] = SERVER_ACTIVITY_EVENT_TYPES,
) {
  return useQuery({
    queryKey: queryKeys.events.list(categories, pageToken, eventTypes),
    queryFn: () =>
      apiService.getEvents(limit, categories, pageToken, eventTypes),
    refetchInterval: TIMING.DASHBOARD_REFRESH * 1000,
  });
}

/**
 * Combined dashboard stats hook
 * Aggregates base station and endpoint counts
 */
export function useDashboardStats() {
  const baseStationsQuery = useQuery({
    queryKey: queryKeys.baseStations.list(),
    queryFn: () => apiService.getBaseStations(),
  });

  const endpointsQuery = useQuery({
    queryKey: queryKeys.endpoints.list(),
    queryFn: () => apiService.getEndpoints(),
  });

  return {
    baseStations: baseStationsQuery.data ?? [],
    endpoints: endpointsQuery.data ?? [],
    isLoading: baseStationsQuery.isLoading || endpointsQuery.isLoading,
    isError: baseStationsQuery.isError || endpointsQuery.isError,
    error: baseStationsQuery.error || endpointsQuery.error,
    refetch: () => {
      baseStationsQuery.refetch();
      endpointsQuery.refetch();
    },
  };
}
