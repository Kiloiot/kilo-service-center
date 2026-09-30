/**
 * Traffic Hooks
 *
 * React Query hooks for uplinks, the downlink queue and downlink results.
 * Listings are paged server-side; realtime events invalidate the
 * queryKeys.traffic prefixes, so new rows arrive without a reload.
 */

import type {
  DownlinkQueueFilter,
  DownlinkResultFilter,
  Page,
  SendDownlinkRequest,
  UpdatePendingDownlinkRequest,
  UplinkFilter,
  UplinkUI,
} from "@api-types/api";
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { downlinkApi, trafficApi } from "@services/api";
import { UPLINK_SOURCE, type UplinkSource } from "@constants/app";
import { queryKeys } from "@config/query-keys";

const UPLINK_LISTINGS: Record<
  UplinkSource,
  (
    filter: UplinkFilter,
    page: number,
    pageSize: number,
  ) => Promise<Page<UplinkUI>>
> = {
  [UPLINK_SOURCE.TENANT]: (filter, page, pageSize) =>
    trafficApi.listUplinks(filter, page, pageSize),
  [UPLINK_SOURCE.STATION]: (filter, page, pageSize) =>
    trafficApi.listStationUplinks(filter, page, pageSize),
};

/** Uplinks from the source the view reads; the station source lists filter.bsEui. */
export function useUplinks(
  source: UplinkSource,
  filter: UplinkFilter,
  page: number,
  pageSize: number,
) {
  return useQuery({
    queryKey: queryKeys.traffic.uplinkList(source, filter, page, pageSize),
    queryFn: () => UPLINK_LISTINGS[source](filter, page, pageSize),
    placeholderData: keepPreviousData,
  });
}

export function useBaseStationTrafficSummary(bsEui: string) {
  return useQuery({
    queryKey: queryKeys.traffic.uplinkSummary({ bsEui }),
    queryFn: () => trafficApi.getBaseStationSummary(bsEui),
    enabled: !!bsEui,
  });
}

export function useEndpointTrafficSummary(epEui: string) {
  return useQuery({
    queryKey: queryKeys.traffic.uplinkSummary({ epEui }),
    queryFn: () => trafficApi.getEndpointSummary(epEui),
    enabled: !!epEui,
  });
}

export function useDownlinkQueue(
  filter: DownlinkQueueFilter,
  page: number,
  pageSize: number,
) {
  return useQuery({
    queryKey: queryKeys.traffic.downlinkQueue(filter, page, pageSize),
    queryFn: () => downlinkApi.listQueue(filter, page, pageSize),
    placeholderData: keepPreviousData,
  });
}

export function useDownlinkResults(
  filter: DownlinkResultFilter,
  page: number,
  pageSize: number,
) {
  return useQuery({
    queryKey: queryKeys.traffic.downlinkResults(filter, page, pageSize),
    queryFn: () => downlinkApi.listResults(filter, page, pageSize),
    placeholderData: keepPreviousData,
  });
}

/** Refreshes the downlink listings and the endpoint details a downlink change touches. */
function useInvalidateDownlinks() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.traffic.downlinks });
    queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
  };
}

export function useSendDownlink() {
  const onSuccess = useInvalidateDownlinks();
  return useMutation({
    mutationFn: (data: SendDownlinkRequest) => downlinkApi.sendDownlink(data),
    onSuccess,
  });
}

export function useUpdatePendingDownlink() {
  const onSuccess = useInvalidateDownlinks();
  return useMutation({
    mutationFn: (data: UpdatePendingDownlinkRequest) =>
      downlinkApi.updatePendingDownlink(data),
    onSuccess,
  });
}

export function useRevokeDownlink() {
  const onSuccess = useInvalidateDownlinks();
  return useMutation({
    mutationFn: ({ epEui, queueId }: { epEui: string; queueId: string }) =>
      downlinkApi.revokeDownlink(epEui, queueId),
    onSuccess,
  });
}

/** Revokes every revocable downlink the queue filter matches. */
export function useFlushDownlinkQueue() {
  const onSuccess = useInvalidateDownlinks();
  return useMutation({
    mutationFn: (filter: DownlinkQueueFilter) => downlinkApi.flushQueue(filter),
    onSuccess,
  });
}
