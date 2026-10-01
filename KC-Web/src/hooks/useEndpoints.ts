/**
 * Endpoint Hooks
 *
 * React Query hooks for endpoint data fetching and mutations.
 */

import type {
  ActivityFilter,
  CreateEndpointRequest,
  UpdateEndpointRequest,
} from "@api-types/api";
import {
  type QueryClient,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { endpointsApi } from "@services/api";
import type { EndpointKeyName } from "@constants/app";
import { queryKeys } from "@config/query-keys";

/**
 * What an endpoint change makes stale: the endpoints, the model counts of
 * endpoints bound to a blueprint, and the downlink queue of a removed endpoint.
 */
function invalidateEndpointViews(queryClient: QueryClient): void {
  [
    queryKeys.endpoints.all,
    queryKeys.blueprints.modelSnapshotCounts(),
    queryKeys.traffic.downlinks,
  ].forEach((queryKey) => queryClient.invalidateQueries({ queryKey }));
}

/**
 * Fetch all endpoints. The optional `filters` arg is kept for callers
 * that already build a filters object — it currently scopes the React Query
 * cache key only; the gRPC list call is unfiltered server-side.
 */
export function useEndpoints() {
  return useQuery({
    queryKey: queryKeys.endpoints.list(),
    queryFn: () => endpointsApi.getEndpoints(),
  });
}

/**
 * Fetch a single endpoint by EUI
 */
export function useEndpoint(eui: string) {
  return useQuery({
    queryKey: queryKeys.endpoints.detail(eui),
    queryFn: () => endpointsApi.getEndpointById(eui),
    enabled: !!eui,
  });
}

/**
 * Create a new endpoint
 */
export function useCreateEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateEndpointRequest) =>
      endpointsApi.createEndpoint(data),
    onSuccess: () => invalidateEndpointViews(queryClient),
  });
}

/**
 * Reveal one stored key of an endpoint. A mutation, so the key is never kept
 * in the query cache; each call is recorded by the service.
 */
export function useRevealEndpointKey() {
  return useMutation({
    mutationFn: ({ epEui, key }: { epEui: string; key: EndpointKeyName }) =>
      endpointsApi.revealEndpointKey(epEui, key),
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
    mutationFn: ({
      epEui,
      data,
    }: {
      epEui: string;
      data: UpdateEndpointRequest;
    }) => endpointsApi.updateEndpoint(epEui, data),
    onSuccess: () => invalidateEndpointViews(queryClient),
  });
}

/**
 * Delete an endpoint
 */
export function useDeleteEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (eui: string) => endpointsApi.deleteEndpoint(eui),
    onSuccess: () => invalidateEndpointViews(queryClient),
  });
}

/**
 * Attach an endpoint to the network
 * @param epEui - Endpoint EUI
 */
export function useAttachEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => endpointsApi.attachEndpoint(id),
    onSuccess: () => invalidateEndpointViews(queryClient),
  });
}

/**
 * Detach an endpoint from the network
 * @param epEui - Endpoint EUI
 */
export function useDetachEndpoint() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => endpointsApi.detachEndpoint(id),
    onSuccess: () => invalidateEndpointViews(queryClient),
  });
}

/**
 * Fetch unified activity feed (events + messages) for an endpoint
 */
export function useEndpointActivity(
  epEui: string,
  filter: ActivityFilter,
  pageToken: string,
  pageSize: number,
) {
  return useQuery({
    queryKey: queryKeys.endpoints.activity(epEui, filter, pageToken, pageSize),
    queryFn: () =>
      endpointsApi.getEndpointActivity(epEui, filter, pageToken, pageSize),
    enabled: !!epEui,
  });
}
