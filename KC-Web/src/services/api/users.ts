/**
 * User administration in the shapes the UI consumes.
 */

import { mapUser, mapUserList } from "@mappers";

import * as usersRpc from "@services/grpc/rpc/users";
import { PAGINATION } from "@constants/app";

export const usersApi = {
  async getUsers(
    limit: number = PAGINATION.ADMIN_LIST_PAGE_SIZE,
    offset = 0,
  ): Promise<{
    users: import("@api-types/api").SystemUserUI[];
    total: number;
  }> {
    const response = await usersRpc.listUsers({ pageSize: limit + offset });
    const mapped = mapUserList(
      response.users.map((u) => ({
        id: u.id,
        email: u.email,
        is_admin: u.isAdmin,
        is_active: u.isActive,
        is_tenant_manager: u.isTenantManager,
        is_base_station_manager: u.isBaseStationManager,
        is_endpoint_manager: u.isEndpointManager,
        note: u.note,
        created_at: u.createdAt?.toISOString() || new Date().toISOString(),
        updated_at: u.updatedAt?.toISOString() || new Date().toISOString(),
      })),
    );
    return {
      users: mapped.slice(offset, offset + limit),
      total: response.totalCount,
    };
  },

  async getUser(
    id: string,
  ): Promise<import("@api-types/api").SystemUserUI | null> {
    const response = await usersRpc.getUser(id);
    if (!response) return null;

    return mapUser({
      id: response.id,
      email: response.email,
      is_admin: response.isAdmin,
      is_active: response.isActive,
      is_tenant_manager: response.isTenantManager,
      is_base_station_manager: response.isBaseStationManager,
      is_endpoint_manager: response.isEndpointManager,
      note: response.note,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  },

  async createUser(
    data: import("@api-types/api").CreateUserRequest,
  ): Promise<import("@api-types/api").SystemUserUI> {
    const response = await usersRpc.createUser({
      email: data.email,
      password: data.password,
      isAdmin: data.is_admin,
      isActive: data.is_active,
      isTenantManager: data.is_tenant_manager,
      isBaseStationManager: data.is_base_station_manager,
      isEndpointManager: data.is_endpoint_manager,
      note: data.note,
    });

    return mapUser({
      id: response.id,
      email: response.email,
      is_admin: response.isAdmin ?? false,
      is_active: response.isActive ?? true,
      is_tenant_manager: response.isTenantManager ?? false,
      is_base_station_manager: response.isBaseStationManager ?? false,
      is_endpoint_manager: response.isEndpointManager ?? false,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  },

  async updateUser(
    id: string,
    data: import("@api-types/api").UpdateUserRequest,
  ): Promise<import("@api-types/api").SystemUserUI> {
    const response = await usersRpc.updateUser(id, {
      email: data.email,
      note: data.note,
      isAdmin: data.is_admin,
      isActive: data.is_active,
      isTenantManager: data.is_tenant_manager,
      isBaseStationManager: data.is_base_station_manager,
      isEndpointManager: data.is_endpoint_manager,
    });

    return mapUser({
      id: response.id,
      email: response.email,
      note: response.note,
      is_admin: response.isAdmin ?? false,
      is_active: response.isActive ?? true,
      is_tenant_manager: response.isTenantManager ?? false,
      is_base_station_manager: response.isBaseStationManager ?? false,
      is_endpoint_manager: response.isEndpointManager ?? false,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  },

  async deleteUser(id: string): Promise<void> {
    await usersRpc.deleteUser(id);
  },

  async changeUserPassword(id: string, password: string): Promise<void> {
    await usersRpc.updateUserPassword(id, password);
  },

  async listUserOrganizations(userId: string): Promise<{
    memberships: Array<{
      orgId: string;
      orgName: string;
      role: string;
      status: string;
    }>;
  }> {
    return usersRpc.listUserOrganizations(userId);
  },
};
