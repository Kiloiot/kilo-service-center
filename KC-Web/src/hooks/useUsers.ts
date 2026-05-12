/**
 * User Hooks
 *
 * React Query hooks for user admin data fetching and mutations.
 */

import type { CreateUserRequest, UpdateUserRequest } from "@api-types/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiService } from "@services/api";
import { TIMING, USER_LOOKUP_LIMIT } from "@constants/app";
import { queryKeys } from "@config/query-keys";

/**
 * Fetch all users with pagination
 */
export function useUsers(
  limit = 50,
  offset = 0,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: queryKeys.users.list({ limit, offset }),
    queryFn: () => apiService.getUsers(limit),
    enabled: options?.enabled ?? true,
  });
}

/**
 * Fetch a single user by ID
 */
export function useUser(id: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.users.detail(id),
    queryFn: () => apiService.getUser(id),
    enabled: Boolean(id) && (options?.enabled ?? true),
  });
}

/**
 * Create a new user
 */
export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserRequest) => apiService.createUser(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
    },
  });
}

/**
 * Update an existing user
 */
export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserRequest }) =>
      apiService.updateUser(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.users.detail(id) });
    },
  });
}

/**
 * Delete a user
 */
export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiService.deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
    },
  });
}

/**
 * Change a user's password
 */
export function useChangePassword() {
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      apiService.changeUserPassword(id, password),
  });
}

/**
 * Fetch users for autocomplete lookup
 * Uses large limit for client-side filtering (no search API available)
 */
export function useUsersForLookup(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.users.list({ limit: USER_LOOKUP_LIMIT, offset: 0 }),
    queryFn: () => apiService.getUsers(USER_LOOKUP_LIMIT),
    enabled: options?.enabled ?? true,
    staleTime: TIMING.LIST_REFRESH * 1000, // Use centralized timing constant (30s)
  });
}

/**
 * Fetch organizations a user belongs to
 */
export function useUserOrganizations(
  userId: string,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: queryKeys.userOrganizations(userId),
    queryFn: () => apiService.listUserOrganizations(userId),
    enabled: options?.enabled !== false && Boolean(userId),
  });
}
