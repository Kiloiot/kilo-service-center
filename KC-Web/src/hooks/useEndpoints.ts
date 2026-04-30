/**
 * Endpoint Hooks
 *
 * React Query hooks for endpoint data fetching and mutations.
 */

import type {
  CreateEndpointRequest,
  SCACIDownlinkQueueDTO,
  SendDownlinkRequest,
  UpdateEndpointRequest,
} from '@api-types/api';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiService } from '@services/api';
import { PAGINATION, REVOCABLE_QUEUE_STATUSES } from '@constants/app';
import type { EndpointFilters } from '@config/query-keys';
import { queryKeys } from '@config/query-keys';

/**
 * Fetch all endpoints with optional filters
 * Filters are passed to API even if backend ignores them (TypeScript aligned)
 */
export function useEndpoints(filters?: EndpointFilters) {
  return useQuery({
    queryKey: queryKeys.endpoints.list(filters),
    queryFn: () => apiService.getEndpoints(filters),
  });
}

/**
 * Fetch a single endpoint by EUI
 */
export function useEndpoint(eui: string) {
  return useQuery({
    queryKey: queryKeys.endpoints.detail(eui),
    queryFn: () => apiService.getEndpointById(eui),
    enabled: !!eui,
  });
}

/**
 * Create a new endpoint
 */
export function useCreateEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateEndpointRequest) => apiService.createEndpoint(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

/**
 * Update an existing endpoint (PATCH semantics)
 * @param id - Numeric endpoint ID
 * @param data - Partial update fields (only provided fields are updated)
 */
export function useUpdateEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ epEui, data }: { epEui: string; data: UpdateEndpointRequest }) =>
      apiService.updateEndpoint(epEui, data),
    onSuccess: () => {
      // Invalidate both list and detail queries
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

/**
 * Delete an endpoint
 */
export function useDeleteEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (eui: string) => apiService.deleteEndpoint(eui),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

/**
 * Attach an endpoint to the network
 * @param epEui - Endpoint EUI
 */
export function useAttachEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiService.attachEndpoint(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

/**
 * Detach an endpoint from the network
 * @param epEui - Endpoint EUI
 */
export function useDetachEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiService.detachEndpoint(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

/**
 * Fetch unified activity feed (events + messages) for an endpoint
 */
export function useEndpointActivity(epEui: string, pageToken?: string, pageSize = 50) {
  return useQuery({
    queryKey: queryKeys.endpoints.activity(epEui, pageToken, pageSize),
    queryFn: () => apiService.getEndpointActivity(epEui, pageToken, pageSize),
    enabled: !!epEui,
  });
}

// ============================================================================
// Downlink Hooks
// ============================================================================

/**
 * Fetch downlink queue for an endpoint (cursor-based pagination)
 */
export function useDownlinkQueue(eui: string, pageSize: number) {
  return useInfiniteQuery({
    queryKey: queryKeys.endpoints.downlinkQueue(eui, pageSize),
    queryFn: ({ pageParam }) => apiService.listDownlinkQueue(eui, pageSize, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    enabled: !!eui,
  });
}

/**
 * Fetch downlink results for an endpoint (cursor-based pagination)
 */
export function useDownlinkResults(eui: string, pageSize: number, statusFilter?: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.endpoints.downlinkResults(eui, statusFilter, pageSize),
    queryFn: ({ pageParam }) =>
      apiService.getDownlinkResults(eui, pageSize, pageParam, statusFilter),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    enabled: !!eui,
  });
}

/**
 * Send a downlink message
 */
export function useSendDownlink() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: SendDownlinkRequest) => apiService.sendDownlink(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

/**
 * Revoke a queued downlink message
 */
export function useRevokeDownlink() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ epEui, queueId }: { epEui: string; queueId: string }) =>
      apiService.revokeDownlink(epEui, queueId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

/**
 * Flush (revoke all revocable) downlink queue entries.
 * Paginates through all queue pages to collect every revocable entry,
 * then batch-revokes via Promise.allSettled.
 */
export function useFlushDownlinkQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ epEui }: { epEui: string }) => {
      const allRevocable: SCACIDownlinkQueueDTO[] = [];
      let pageToken: string | undefined;

      do {
        const response = await apiService.listDownlinkQueue(
          epEui,
          PAGINATION.DOWNLINK_FLUSH_PAGE_SIZE,
          pageToken
        );
        const revocable = response.messages.filter(
          (m) => m.status && REVOCABLE_QUEUE_STATUSES.has(m.status)
        );
        allRevocable.push(...revocable);
        pageToken = response.nextPageToken || undefined;
      } while (pageToken);

      if (allRevocable.length === 0) return { revoked: 0, failed: 0 };

      const results = await Promise.allSettled(
        allRevocable.map((m) => apiService.revokeDownlink(epEui, String(m.queId)))
      );

      return {
        revoked: results.filter((r) => r.status === 'fulfilled').length,
        failed: results.filter((r) => r.status === 'rejected').length,
      };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}
