/**
 * Organizations and memberships in the shapes the UI consumes.
 */

import {
  mapOrganization,
  mapOrganizationList,
  mapOrgUser,
  mapOrgUserList,
} from "@mappers";

import * as organizationsRpc from "@services/grpc/rpc/organizations";
import { ORG_STATE, PAGINATION } from "@constants/app";

export const organizationsApi = {
  async getOrganizations(
    limit: number = PAGINATION.ADMIN_LIST_PAGE_SIZE,
    offset = 0,
  ): Promise<{
    organizations: import("@api-types/api").OrganizationUI[];
    total: number;
  }> {
    const pageSize = limit + offset;
    const response = await organizationsRpc.listOrganizations({ pageSize });
    const organizations = mapOrganizationList(
      response.organizations.map((o) => ({
        id: o.id,
        name: o.name,
        description: o.description,
        tenant_id: parseInt(o.tenantId, 10) || 0,
        state: o.state || ORG_STATE.ACTIVE,
        created_at: o.createdAt?.toISOString() || new Date().toISOString(),
        updated_at: o.updatedAt?.toISOString() || new Date().toISOString(),
      })),
    );
    return {
      organizations: organizations.slice(offset, offset + limit),
      total: response.totalCount,
    };
  },

  async getOrganization(
    id: string,
  ): Promise<import("@api-types/api").OrganizationUI | null> {
    const response = await organizationsRpc.getOrganization(id);
    if (!response) return null;

    return mapOrganization({
      id: response.id,
      name: response.name,
      description: response.description,
      tenant_id: parseInt(response.tenantId, 10) || 0,
      state: response.state || ORG_STATE.ACTIVE,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  },

  async createOrganization(
    data: import("@api-types/api").CreateOrganizationRequest,
  ): Promise<import("@api-types/api").OrganizationUI> {
    const response = await organizationsRpc.createOrganization({
      name: data.name,
      description: data.description,
    });

    return mapOrganization({
      id: response.id,
      name: response.name,
      description: response.description,
      tenant_id: parseInt(response.tenantId, 10) || 0,
      state: ORG_STATE.ACTIVE,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  },

  async updateOrganization(
    id: string,
    data: import("@api-types/api").UpdateOrganizationRequest,
  ): Promise<import("@api-types/api").OrganizationUI> {
    const response = await organizationsRpc.updateOrganization(id, {
      name: data.name,
      description: data.description,
    });

    return mapOrganization({
      id: response.id,
      name: response.name,
      description: response.description,
      tenant_id: parseInt(response.tenantId, 10) || 0,
      state: ORG_STATE.ACTIVE,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  },

  async deleteOrganization(id: string): Promise<void> {
    await organizationsRpc.deleteOrganization(id);
  },

  async getOrgUsers(
    orgId: string,
    status?: string,
  ): Promise<{
    users: import("@api-types/api").OrganizationUserUI[];
    total: number;
  }> {
    const response = await organizationsRpc.listOrganizationUsers(orgId, {
      status,
    });
    return {
      users: mapOrgUserList(
        response.users.map((u) => ({
          user_id: u.userId,
          org_id: u.orgId,
          role: u.role,
          status: u.status,
          email: u.email,
          is_org_admin: u.isOrgAdmin ?? false,
          is_base_station_admin: u.isBaseStationAdmin ?? false,
          is_endpoint_admin: u.isEndpointAdmin ?? false,
          created_at: u.createdAt?.toISOString() || new Date().toISOString(),
          updated_at: u.updatedAt?.toISOString() || new Date().toISOString(),
        })),
      ),
      total: response.totalCount,
    };
  },

  async getOrgUser(
    orgId: string,
    userId: string,
  ): Promise<import("@api-types/api").OrganizationUserUI | null> {
    const response = await organizationsRpc.getOrganizationUser(orgId, userId);
    if (!response) return null;

    return mapOrgUser({
      user_id: response.userId,
      org_id: response.orgId,
      role: response.role,
      status: response.status,
      email: response.email,
      is_org_admin: response.isOrgAdmin ?? false,
      is_base_station_admin: response.isBaseStationAdmin ?? false,
      is_endpoint_admin: response.isEndpointAdmin ?? false,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  },

  async addOrgUser(
    orgId: string,
    data: import("@api-types/api").AddOrgUserRequest,
  ): Promise<import("@api-types/api").OrganizationUserUI> {
    const response = await organizationsRpc.addOrganizationUser(orgId, {
      userId: data.user_id,
      email: data.email,
      role: data.role,
      isOrgAdmin: data.is_org_admin,
      isBaseStationAdmin: data.is_base_station_admin,
      isEndpointAdmin: data.is_endpoint_admin,
    });

    return mapOrgUser({
      user_id: response.userId,
      org_id: response.orgId,
      role: response.role,
      status: response.status,
      email: response.email || "",
      is_org_admin: response.isOrgAdmin ?? false,
      is_base_station_admin: response.isBaseStationAdmin ?? false,
      is_endpoint_admin: response.isEndpointAdmin ?? false,
      created_at: response.createdAt?.toISOString() || "",
      updated_at: response.updatedAt?.toISOString() || "",
    });
  },

  async updateOrgUser(
    orgId: string,
    userId: string,
    data: import("@api-types/api").UpdateOrgUserRequest,
  ): Promise<import("@api-types/api").OrganizationUserUI> {
    const response = await organizationsRpc.updateOrganizationUser(
      orgId,
      userId,
      {
        role: data.role,
        isOrgAdmin: data.is_org_admin,
        isBaseStationAdmin: data.is_base_station_admin,
        isEndpointAdmin: data.is_endpoint_admin,
      },
    );

    return mapOrgUser({
      user_id: response.userId,
      org_id: response.orgId,
      role: response.role,
      status: response.status,
      email: response.email || "",
      is_org_admin: response.isOrgAdmin ?? false,
      is_base_station_admin: response.isBaseStationAdmin ?? false,
      is_endpoint_admin: response.isEndpointAdmin ?? false,
      created_at: response.createdAt?.toISOString() || "",
      updated_at: response.updatedAt?.toISOString() || "",
    });
  },

  async removeOrgUser(orgId: string, userId: string): Promise<void> {
    await organizationsRpc.removeOrganizationUser(orgId, userId);
  },
};
