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
} from "@api-types/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { organizationsApi } from "@services/api";
import { PAGINATION, TIMING_FALLBACK_POLL_MS } from "@constants/app";
import { queryKeys } from "@config/query-keys";

import { useCapabilities } from "./useCapabilities";

// ============================================================================
// Organization Hooks
// ============================================================================

/**
 * Fetch all organizations with pagination
 */
export function useOrganizations(
  limit: number = PAGINATION.ADMIN_LIST_PAGE_SIZE,
  offset = 0,
  tenantId?: number,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: queryKeys.organizations.list({ limit, offset, tenantId }),
    queryFn: () => organizationsApi.getOrganizations(limit, offset),
    enabled: options?.enabled ?? true,
  });
}

/**
 * Fetch a single organization by ID
 */
export function useOrganization(id: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.organizations.detail(id),
    queryFn: () => organizationsApi.getOrganization(id),
    enabled: (options?.enabled ?? true) && !!id,
  });
}

/**
 * Create a new organization
 */
export function useCreateOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateOrganizationRequest) =>
      organizationsApi.createOrganization(data),
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
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: UpdateOrganizationRequest;
    }) => organizationsApi.updateOrganization(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.all });
      queryClient.invalidateQueries({
        queryKey: queryKeys.organizations.detail(id),
      });
    },
  });
}

/**
 * Delete an organization permanently
 */
export function useDeleteOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => organizationsApi.deleteOrganization(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.organizations.all });
      queryClient.invalidateQueries({
        queryKey: queryKeys.userOrganizations.all,
      });
    },
  });
}

// ============================================================================
// Organization User Hooks
// ============================================================================

/**
 * Membership changes are audit events, which only administrators receive, so
 * the member list polls for every other viewer.
 */
function useMembershipPollInterval(): number | false {
  const { isServerAdmin } = useCapabilities();
  return isServerAdmin ? false : TIMING_FALLBACK_POLL_MS;
}

/**
 * Fetch organization members with optional status filter
 */
export function useOrgUsers(
  orgId: string,
  status?: string,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: queryKeys.organizations.users(orgId),
    queryFn: () => organizationsApi.getOrgUsers(orgId, status),
    enabled: (options?.enabled ?? true) && !!orgId,
    refetchInterval: useMembershipPollInterval(),
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
      organizationsApi.addOrgUser(orgId, data),
    onSuccess: (_, { orgId, data }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.organizations.users(orgId),
      });
      if (data.user_id) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.userOrganizations.list(data.user_id),
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
    }) => organizationsApi.updateOrgUser(orgId, userId, data),
    onSuccess: (_, { orgId, userId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.organizations.users(orgId),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.userOrganizations.list(userId),
      });
      // The member may be the signed-in user, whose roles then change.
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.rolesAll() });
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
      organizationsApi.removeOrgUser(orgId, userId),
    onSuccess: (_, { orgId, userId }) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.organizations.users(orgId),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.userOrganizations.list(userId),
      });
    },
  });
}
