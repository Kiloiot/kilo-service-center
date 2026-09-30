/**
 * Alert hooks. Alerts are system events, so every realtime event refreshes
 * them (dashboard keys).
 */

import type { AlertFilter } from "@api-types/alerts";
import type { PageRequest } from "@api-types/pagination";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { alertsApi } from "@services/api";
import { queryKeys } from "@config/query-keys";

export function useAlertSummary() {
  return useQuery({
    queryKey: queryKeys.dashboard.alertSummary(),
    queryFn: () => alertsApi.getSummary(),
  });
}

export function useAlerts(page: PageRequest, filter: AlertFilter) {
  return useQuery({
    queryKey: queryKeys.dashboard.alerts(page, filter),
    queryFn: () => alertsApi.listAlerts(page, filter),
    placeholderData: keepPreviousData,
  });
}
