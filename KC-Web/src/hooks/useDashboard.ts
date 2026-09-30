/**
 * Dashboard Hooks
 *
 * React Query hooks for dashboard data fetching.
 */

import { useQuery } from "@tanstack/react-query";

import { baseStationsApi, endpointsApi, systemApi } from "@services/api";
import { queryKeys } from "@config/query-keys";

/** Analytics overview for the dashboard; the dashboard realtime families refresh it. */
export function useDashboardAnalytics() {
  return useQuery({
    queryKey: queryKeys.dashboard.analytics(),
    queryFn: () => systemApi.getAnalyticsOverview(),
  });
}

/** Which lists the signed-in user may read for the dashboard counts. */
export interface DashboardStatsAccess {
  baseStations: boolean;
  endpoints: boolean;
}

/** The base stations and endpoints the user may read, for the dashboard counts. */
export function useDashboardStats(access: DashboardStatsAccess) {
  const baseStationsQuery = useQuery({
    queryKey: queryKeys.baseStations.list(),
    queryFn: () => baseStationsApi.getBaseStations(),
    enabled: access.baseStations,
  });

  const endpointsQuery = useQuery({
    queryKey: queryKeys.endpoints.list(),
    queryFn: () => endpointsApi.getEndpoints(),
    enabled: access.endpoints,
  });

  return {
    baseStations: baseStationsQuery.data ?? [],
    endpoints: endpointsQuery.data ?? [],
  };
}
