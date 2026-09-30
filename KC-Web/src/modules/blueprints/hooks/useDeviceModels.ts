/**
 * Device Model Domain Hooks
 *
 * React Query hooks for device model CRUD operations.
 * Uses centralized query keys for cache management.
 */

import type { BlueprintScope, UpdateDeviceModelRequest } from "@api-types/api";
import { useMutation, useQuery } from "@tanstack/react-query";

import { catalogApi } from "@services/api";
import { queryKeys } from "@config/query-keys";

import { useCatalogInvalidation } from "./useCatalogInvalidation";

/**
 * Hook to fetch device models for a manufacturer
 */
export function useDeviceModels(
  manufacturerId: string | undefined,
  scope?: BlueprintScope,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: queryKeys.blueprints.deviceModels(manufacturerId ?? "", scope),
    queryFn: () => catalogApi.getDeviceModels(manufacturerId!, scope),
    enabled: !!manufacturerId && (options.enabled ?? true),
  });
}

export function useDeviceModel(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.blueprints.deviceModelDetail(id ?? ""),
    queryFn: () => catalogApi.getDeviceModel(id!),
    enabled: !!id,
  });
}

export function useCreateDeviceModelWithBlueprint() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: (
      request: Parameters<typeof catalogApi.createDeviceModelWithBlueprint>[0],
    ) => catalogApi.createDeviceModelWithBlueprint(request),
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook to update a device model
 */
export function useUpdateDeviceModel() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: UpdateDeviceModelRequest;
    }) => catalogApi.updateDeviceModel(id, data),
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook to delete a device model
 */
export function useDeleteDeviceModel() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: (id: string) => catalogApi.deleteDeviceModel(id),
    onSuccess: () => invalidateCatalog(),
  });
}
