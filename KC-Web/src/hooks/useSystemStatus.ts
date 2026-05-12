/**
 * System Status Hook
 *
 * React Query hook for fetching system health status.
 * Auto-refreshes at TIMING_STATUS_REFRESH interval.
 */

import { useQuery } from "@tanstack/react-query";

import { apiService } from "@services/api";
import { TIMING_STATUS_REFRESH } from "@constants/app";
import { queryKeys } from "@config/query-keys";

export interface UseSystemStatusOptions {
  /** Override the default refresh interval (milliseconds) */
  refetchInterval?: number;
  /** Enable or disable automatic refetching */
  enabled?: boolean;
}

/**
 * Fetch system status (health of all monitored services)
 *
 * @param options - Configuration options
 * @returns React Query result with system status data
 */
export function useSystemStatus(options?: UseSystemStatusOptions) {
  const { refetchInterval = TIMING_STATUS_REFRESH, enabled = true } =
    options ?? {};

  return useQuery({
    queryKey: queryKeys.system.status(),
    queryFn: () => apiService.getSystemStatus(),
    refetchInterval,
    enabled,
  });
}
