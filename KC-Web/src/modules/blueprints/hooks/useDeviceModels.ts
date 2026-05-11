/**
 * Device Model Domain Hooks
 *
 * React Query hooks for device model CRUD operations.
 * Uses centralized query keys for cache management.
 */

import type {
  CreateDeviceModelRequest,
  UpdateDeviceModelRequest,
} from "@api-types/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@services/api";
import { queryKeys } from "@config/query-keys";

/**
 * Hook to fetch device models for a manufacturer
 */
export function useDeviceModels(manufacturerId: string | undefined) {
  return useQuery({
    queryKey: queryKeys.blueprints.deviceModels(manufacturerId ?? ""),
    queryFn: () => api.getDeviceModels(manufacturerId!),
    enabled: !!manufacturerId,
  });
}

/**
 * Hook to create a new device model
 */
export function useCreateDeviceModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      manufacturerId,
      data,
    }: {
      manufacturerId: string;
      data: CreateDeviceModelRequest;
    }) => api.createDeviceModel(manufacturerId, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.blueprints.deviceModels(variables.manufacturerId),
      });
    },
  });
}

/**
 * Hook to update a device model
 */
export function useUpdateDeviceModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      manufacturerId: _manufacturerId,
      data,
    }: {
      id: string;
      manufacturerId: string;
      data: UpdateDeviceModelRequest;
    }) => api.updateDeviceModel(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.blueprints.deviceModels(variables.manufacturerId),
      });
    },
  });
}

/**
 * Hook to delete a device model
 */
export function useDeleteDeviceModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      manufacturerId: _manufacturerId,
    }: {
      id: string;
      manufacturerId: string;
    }) => api.deleteDeviceModel(id),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.blueprints.deviceModels(variables.manufacturerId),
      });
    },
  });
}
