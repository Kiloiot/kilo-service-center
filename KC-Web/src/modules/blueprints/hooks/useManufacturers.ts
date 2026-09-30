/**
 * Manufacturer Domain Hooks
 *
 * React Query hooks for manufacturer CRUD operations.
 * Uses centralized query keys for cache management.
 */

import { useMemo } from "react";

import type {
  BlueprintScope,
  CreateManufacturerRequest,
  UpdateManufacturerRequest,
} from "@api-types/api";
import { useMutation, useQuery } from "@tanstack/react-query";

import { catalogApi } from "@services/api";
import { BLUEPRINT_SCOPE } from "@constants/app";
import { queryKeys } from "@config/query-keys";

import { useCatalogInvalidation } from "./useCatalogInvalidation";

/**
 * Hook to fetch all manufacturers
 */
export function useManufacturers(scope?: BlueprintScope) {
  return useQuery({
    queryKey: queryKeys.blueprints.manufacturers(scope),
    queryFn: () => catalogApi.getManufacturers(scope),
  });
}

/**
 * Manufacturers of both catalogs, the tenant's Custom ones first, for pickers
 * that may bind an endpoint to either catalog.
 */
export function useCatalogManufacturers() {
  const custom = useManufacturers(BLUEPRINT_SCOPE.CUSTOM);
  const system = useManufacturers(BLUEPRINT_SCOPE.SYSTEM);
  const data = useMemo(
    () => [...(custom.data ?? []), ...(system.data ?? [])],
    [custom.data, system.data],
  );
  return { data, isLoading: custom.isLoading || system.isLoading };
}

export function useManufacturer(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.blueprints.manufacturerDetail(id ?? ""),
    queryFn: () => catalogApi.getManufacturer(id!),
    enabled: !!id,
  });
}

/**
 * Hook to create a new manufacturer
 */
export function useCreateManufacturer() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: (data: CreateManufacturerRequest) =>
      catalogApi.createManufacturer(data),
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook to update a manufacturer
 */
export function useUpdateManufacturer() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: UpdateManufacturerRequest;
    }) => catalogApi.updateManufacturer(id, data),
    onSuccess: () => invalidateCatalog(),
  });
}

/**
 * Hook to delete a manufacturer
 */
export function useDeleteManufacturer() {
  const invalidateCatalog = useCatalogInvalidation();
  return useMutation({
    mutationFn: (id: string) => catalogApi.deleteManufacturer(id),
    onSuccess: () => invalidateCatalog(),
  });
}
