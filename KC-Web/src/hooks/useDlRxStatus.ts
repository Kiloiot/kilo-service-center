/**
 * DL RX status hooks. The queries live under the end point's detail key, so
 * every realtime event that refreshes the end point refreshes them too.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { dlRxStatusApi } from "@services/api";
import { ENDPOINT_DETAIL_QUERY_SEGMENT, PAGINATION } from "@constants/app";
import { queryKeys } from "@config/query-keys";

const dlRxStatusKey = (eui: string) =>
  [
    ...queryKeys.endpoints.detail(eui),
    ENDPOINT_DETAIL_QUERY_SEGMENT.DL_RX_STATUS,
  ] as const;

const dlRxStatusQueriesKey = (eui: string) =>
  [
    ...queryKeys.endpoints.detail(eui),
    ENDPOINT_DETAIL_QUERY_SEGMENT.DL_RX_STATUS_QUERIES,
  ] as const;

/** The DL RX statuses an end point reported, newest first. */
export function useDlRxStatuses(eui: string) {
  return useQuery({
    queryKey: dlRxStatusKey(eui),
    queryFn: () =>
      dlRxStatusApi.getStatuses(eui, PAGINATION.DL_RX_STATUS_PAGE_SIZE),
    enabled: !!eui,
  });
}

/** The DL RX status queries sent for an end point, newest first. */
export function useDlRxStatusQueries(eui: string) {
  return useQuery({
    queryKey: dlRxStatusQueriesKey(eui),
    queryFn: () =>
      dlRxStatusApi.getQueries(eui, PAGINATION.DL_RX_STATUS_PAGE_SIZE),
    enabled: !!eui,
  });
}

/** Asks the end point's serving base station for its DL RX status. */
export function useQueryDlRxStatus(eui: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => dlRxStatusApi.query(eui),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: dlRxStatusQueriesKey(eui) });
    },
  });
}
