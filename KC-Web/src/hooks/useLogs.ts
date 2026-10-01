/**
 * Log Hooks
 *
 * React Query hooks for the Events/Audit logs and the Errors Center. Their
 * keys sit under queryKeys.events.all, which every realtime event refreshes.
 */

import type { ErrorGroupFilter, EventLogFilter } from "@api-types/api";
import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { logsApi } from "@services/api";
import { queryKeys } from "@config/query-keys";

export function useEventLog(
  filter: EventLogFilter,
  page: number,
  pageSize: number,
) {
  return useQuery({
    queryKey: queryKeys.events.log(filter, page, pageSize),
    queryFn: () => logsApi.listEvents(filter, page, pageSize),
    placeholderData: keepPreviousData,
  });
}

export function useErrorGroups(
  filter: ErrorGroupFilter,
  page: number,
  pageSize: number,
) {
  return useQuery({
    queryKey: queryKeys.events.errorGroups(filter, page, pageSize),
    queryFn: () => logsApi.listErrorGroups(filter, page, pageSize),
    placeholderData: keepPreviousData,
  });
}
