/**
 * Manufacturer Domain Hooks
 *
 * React Query hooks for manufacturer CRUD operations.
 * Uses centralized query keys for cache management.
 */

import type { CreateManufacturerRequest, UpdateManufacturerRequest } from '@api-types/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { api } from '@services/api';
import { queryKeys } from '@config/query-keys';

/**
 * Hook to fetch all manufacturers
 */
export function useManufacturers() {
  return useQuery({
    queryKey: queryKeys.blueprints.manufacturers(),
    queryFn: () => api.getManufacturers(),
  });
}

/**
 * Hook to create a new manufacturer
 */
export function useCreateManufacturer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateManufacturerRequest) => api.createManufacturer(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.manufacturers() });
    },
  });
}

/**
 * Hook to update a manufacturer
 */
export function useUpdateManufacturer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateManufacturerRequest }) =>
      api.updateManufacturer(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.manufacturers() });
    },
  });
}

/**
 * Hook to delete a manufacturer
 */
export function useDeleteManufacturer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteManufacturer(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.manufacturers() });
    },
  });
}
