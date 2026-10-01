/**
 * API Key React Query Hooks
 *
 * Provides hooks for listing, creating, and deleting API keys
 * with automatic cache invalidation via React Query.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiKeysApi } from "@services/api";
import { queryKeys } from "@config/query-keys";

export function useApiKeys(params?: {
  pageSize?: number;
  pageToken?: string;
  userId?: string;
}) {
  return useQuery({
    queryKey: queryKeys.apiKeys.list(params),
    queryFn: () => apiKeysApi.list(params),
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: { name: string; keyType: string; expiresAt?: Date }) =>
      apiKeysApi.create(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.all });
    },
  });
}

export function useDeleteApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiKeysApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.all });
    },
  });
}
