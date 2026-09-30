/**
 * Blueprint Domain Hooks
 *
 * React Query hooks for blueprint CRUD operations.
 * Uses centralized query keys for cache management.
 */

import type {
  BlueprintScope,
  CreateBlueprintRequest,
  DecodePreviewRequest,
  RegistrySubmitRequest,
  UpdateBlueprintRequest,
} from "@api-types/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { catalogApi } from "@services/api";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { queryKeys } from "@config/query-keys";

import { useCatalogInvalidation } from "./useCatalogInvalidation";

/**
 * Hook to fetch blueprints for a device model
 */
export function useBlueprints(
  deviceModelId: string | undefined,
  scope?: BlueprintScope,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: queryKeys.blueprints.list(deviceModelId ?? "", scope),
    queryFn: () => catalogApi.getBlueprints(deviceModelId!, scope),
    enabled: !!deviceModelId && (options.enabled ?? true),
  });
}

export function useModelSnapshotCount(
  deviceModelId: string | undefined,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: queryKeys.blueprints.modelSnapshotCount(deviceModelId ?? ""),
    queryFn: () => catalogApi.countModelSnapshotEndpoints(deviceModelId!),
    enabled: !!deviceModelId && (options.enabled ?? true),
  });
}

export function useBulkAssignBlueprint() {
  const queryClient = useQueryClient();
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: (
      request: Parameters<typeof catalogApi.bulkAssignBlueprint>[0],
    ) => catalogApi.bulkAssignBlueprint(request),
    onSuccess: () => {
      invalidateCatalog();
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
    },
  });
}

export function useSubmitToRegistry(blueprintId: string | null | undefined) {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: (data: RegistrySubmitRequest) => {
      if (!blueprintId)
        throw new Error(BLUEPRINT_LABELS.ERR_BLUEPRINT_ID_REQUIRED);
      return catalogApi.submitToRegistry(blueprintId, data);
    },
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook to fetch a single blueprint by ID
 */
export function useBlueprint(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.blueprints.detail(id ?? ""),
    queryFn: () => catalogApi.getBlueprint(id!),
    enabled: !!id,
  });
}

/**
 * Hook to create a new blueprint
 */
export function useCreateBlueprint() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: ({
      deviceModelId,
      data,
    }: {
      deviceModelId: string;
      data: CreateBlueprintRequest;
    }) => catalogApi.createBlueprint(deviceModelId, data),
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook to update a blueprint
 */
export function useUpdateBlueprint() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateBlueprintRequest }) =>
      catalogApi.updateBlueprint(id, data),
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook to set a blueprint as default
 */
export function useSetBlueprintDefault() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: ({ id }: { id: string }) => catalogApi.setBlueprintDefault(id),
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook for decode preview
 */
export function useDecodePreview(blueprintId: string | undefined) {
  return useMutation({
    mutationFn: (data: DecodePreviewRequest) => {
      if (!blueprintId)
        throw new Error(BLUEPRINT_LABELS.ERR_BLUEPRINT_ID_REQUIRED);
      return catalogApi.decodePreview(blueprintId, data);
    },
  });
}
