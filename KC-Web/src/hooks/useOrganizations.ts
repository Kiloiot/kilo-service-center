/**
 * Organization Hooks
 *
 * React Query hooks for organization admin data fetching and mutations.
 */

import type {
  AddOrgUserRequest,
  CreateOrganizationRequest,
  UpdateOrganizationRequest,
  UpdateOrgUserRequest,
} from '@api-types/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiService } from '@services/api';
import { queryKeys } from '@config/query-keys';

// ============================================================================
// Organization Hooks
// ============================================================================

/**
 * Fetch all organizations with pagination
 */
export function useOrganizations(
  limit = 50,
  offset = 0,
  tenantId?: number,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: queryKeys.organizations.list({ limit, offset, tenantId }),
    queryFn: () => apiService.getOrganizations(limit, offset, tenantId),
    enabled: options?.enabled ?? true,
  });
}

/**
 * Fetch a single organization by ID
 */
export function useOrganization(id: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.organizations.detail(id),
    queryFn: () => apiService.getOrganization(id),
    enabled: (options?.enabled ?? true) && !!id,
  });
}

/**
 * Create a new organization
 */
export function useCreateOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateOrganizationRequest) => apiService.createOrganization(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.all });
    },
  });
}

/**
 * Update an existing organization
 */
export function useUpdateOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateOrganizationRequest }) =>
      apiService.updateOrganization(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.detail(id) });
    },
  });
}

/**
 * Delete an organization permanently
 */
export function useDeleteOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiService.deleteOrganization(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.all });
    },
  });
}

// ============================================================================
// Organization User Hooks
// ============================================================================

/**
 * Fetch organization members with optional status filter
 */
export function useOrgUsers(orgId: string, status?: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.organizations.users(orgId),
    queryFn: () => apiService.getOrgUsers(orgId, status),
    enabled: (options?.enabled ?? true) && !!orgId,
  });
}

/**
 * Fetch a single organization member
 */
export function useOrgUser(orgId: string, userId: string) {
  return useQuery({
    queryKey: queryKeys.organizations.userDetail(orgId, userId),
    queryFn: () => apiService.getOrgUser(orgId, userId),
    enabled: !!orgId && !!userId,
  });
}

/**
 * Add a user to an organization.
 * Invalidates both the org members list and the user's memberships list.
 */
export function useAddOrgUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ orgId, data }: { orgId: string; data: AddOrgUserRequest }) =>
      apiService.addOrgUser(orgId, data),
    onSuccess: (_, { orgId, data }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.users(orgId) });
      if (data.user_id) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.userOrganizations(data.user_id),
        });
      }
    },
  });
}

/**
 * Update an organization member's role/permissions
 */
export function useUpdateOrgUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      orgId,
      userId,
      data,
    }: {
      orgId: string;
      userId: string;
      data: UpdateOrgUserRequest;
    }) => apiService.updateOrgUser(orgId, userId, data),
    onSuccess: (_, { orgId, userId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.users(orgId) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.organizations.userDetail(orgId, userId),
      });
    },
  });
}

/**
 * Remove a user from an organization.
 * Invalidates both the org members list and the user's memberships list.
 */
export function useRemoveOrgUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ orgId, userId }: { orgId: string; userId: string }) =>
      apiService.removeOrgUser(orgId, userId),
    onSuccess: (_, { orgId, userId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.users(orgId) });
      queryClient.invalidateQueries({
        queryKey: queryKeys.userOrganizations(userId),
      });
    },
  });
}
