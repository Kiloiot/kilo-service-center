/**
 * User Hooks
 *
 * React Query hooks for user admin data fetching and mutations.
 */

import type { CreateUserRequest, UpdateUserRequest } from "@api-types/api";
import {
  type QueryClient,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { usersApi } from "@services/api";
import {
  MS_PER_SECOND,
  PAGINATION,
  TIMING,
  USER_LOOKUP_LIMIT,
} from "@constants/app";
import { queryKeys } from "@config/query-keys";

/** What a user change makes stale: the users, and the member lists and memberships that name them. */
function invalidateUserViews(queryClient: QueryClient): void {
  [
    queryKeys.users.all,
    queryKeys.organizations.usersAll(),
    queryKeys.userOrganizations.all,
  ].forEach((queryKey) => queryClient.invalidateQueries({ queryKey }));
}

/**
 * Fetch all users with pagination
 */
export function useUsers(
  limit: number = PAGINATION.ADMIN_LIST_PAGE_SIZE,
  offset = 0,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: queryKeys.users.list({ limit, offset }),
    queryFn: () => usersApi.getUsers(limit),
    enabled: options?.enabled ?? true,
  });
}

/**
 * Fetch a single user by ID
 */
export function useUser(id: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.users.detail(id),
    queryFn: () => usersApi.getUser(id),
    enabled: Boolean(id) && (options?.enabled ?? true),
  });
}

/**
 * Create a new user
 */
export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserRequest) => usersApi.createUser(data),
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
      usersApi.updateUser(id, data),
    onSuccess: () => invalidateUserViews(queryClient),
  });
}

/**
 * Delete a user
 */
export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => usersApi.deleteUser(id),
    onSuccess: () => invalidateUserViews(queryClient),
  });
}

/**
 * Change a user's password
 */
export function useChangePassword() {
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      usersApi.changeUserPassword(id, password),
  });
}

/**
 * Fetch users for autocomplete lookup
 * Uses large limit for client-side filtering (no search API available)
 */
export function useUsersForLookup(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.users.list({ limit: USER_LOOKUP_LIMIT, offset: 0 }),
    queryFn: () => usersApi.getUsers(USER_LOOKUP_LIMIT),
    enabled: options?.enabled ?? true,
    staleTime: TIMING.LIST_REFRESH * MS_PER_SECOND,
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
    queryKey: queryKeys.userOrganizations.list(userId),
    queryFn: () => usersApi.listUserOrganizations(userId),
    enabled: options?.enabled !== false && Boolean(userId),
  });
}
