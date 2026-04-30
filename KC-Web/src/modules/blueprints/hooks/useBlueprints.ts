/**
 * Blueprint Domain Hooks
 *
 * React Query hooks for blueprint CRUD operations.
 * Uses centralized query keys for cache management.
 */

import type {
  CreateBlueprintRequest,
  DecodePreviewRequest,
  UpdateBlueprintRequest,
} from '@api-types/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { api } from '@services/api';
import { BLUEPRINT_LABELS } from '@constants/messages';
import { queryKeys } from '@config/query-keys';

/**
 * Hook to fetch blueprints for a device model
 */
export function useBlueprints(deviceModelId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.blueprints.list(deviceModelId ?? ''),
    queryFn: () => api.getBlueprints(deviceModelId!),
    enabled: !!deviceModelId,
  });
}

/**
 * Hook to fetch a single blueprint by ID
 */
export function useBlueprint(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.blueprints.detail(id ?? ''),
    queryFn: () => api.getBlueprint(id!),
    enabled: !!id,
  });
}

/**
 * Hook to create a new blueprint
 */
export function useCreateBlueprint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      deviceModelId,
      data,
    }: {
      deviceModelId: string;
      data: CreateBlueprintRequest;
    }) => api.createBlueprint(deviceModelId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.blueprints.list(variables.deviceModelId),
      });
    },
  });
}

/**
 * Hook to update a blueprint
 */
export function useUpdateBlueprint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      deviceModelId: _deviceModelId,
      data,
    }: {
      id: string;
      deviceModelId: string;
      data: UpdateBlueprintRequest;
    }) => api.updateBlueprint(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.detail(variables.id) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.blueprints.list(variables.deviceModelId),
      });
    },
  });
}

/**
 * Hook to delete a blueprint
 */
export function useDeleteBlueprint() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, deviceModelId: _deviceModelId }: { id: string; deviceModelId: string }) =>
      api.deleteBlueprint(id),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.blueprints.list(variables.deviceModelId),
      });
    },
  });
}

/**
 * Hook to set a blueprint as default
 */
export function useSetBlueprintDefault() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id }: { id: string }) => api.setBlueprintDefault(id),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.detail(variables.id) });
      // Invalidate all lists since default status affects display
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.all });
    },
  });
}

/**
 * Hook for decode preview
 */
export function useDecodePreview(blueprintId: string | undefined) {
  return useMutation({
    mutationFn: (data: DecodePreviewRequest) => {
      if (!blueprintId) throw new Error(BLUEPRINT_LABELS.ERR_BLUEPRINT_ID_REQUIRED);
      return api.decodePreview(blueprintId, data);
    },
  });
}
